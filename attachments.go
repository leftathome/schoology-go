package schoology

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Attachment describes a downloadable file associated with a message
// thread, feed post, or other resource. Schoology's attachment URL
// has the shape /attachment/{id}/source/{hash}.{ext}?checkad=1;
// ID is parsed from that URL, while Filename and MimeType come from
// the parent resource's HTML (and may be zero-valued when the page
// doesn't surface them).
type Attachment struct {
	ID       int64  // from the /attachment/{id}/... URL
	Filename string // display name, not the hashed URL-path name
	URL      string // source URL, includes the ?checkad=1 tail
	MimeType string // best-effort; may be the Schoology label ("Adobe PDF") rather than an RFC media type
}

// parseAttachmentID returns the numeric attachment id embedded in rawURL,
// or 0 if the URL does not match /attachment/{id}/... .
func parseAttachmentID(rawURL string) int64 { return parsePathID(rawURL, "/attachment/") }

// DownloadAttachment does an authenticated GET for an attachment URL
// and returns the raw body. The caller MUST close the returned
// io.ReadCloser. Non-2xx responses return a *Error — use IsAuthError /
// IsNotFoundError / IsPermissionError (or errors.Is against the
// sentinels in errors.go) to branch on the cause.
//
// url may be absolute (starts with "https://") or path-only (starts
// with "/"). Absolute URLs pointing at the same host as the client
// are rewritten to path-only form so the client's session cookies
// attach correctly; absolute URLs pointing elsewhere are rejected
// with ErrCodeClient so session cookies cannot be misdirected.
func (c *Client) DownloadAttachment(ctx context.Context, url string) (io.ReadCloser, error) {
	const op = "DownloadAttachment"

	path, err := c.attachmentPath(url)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeClient,
			Op:      op,
			Message: err.Error(),
		}
	}

	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, withOp(op, err)
	}

	// Expired sessions get a 200 with the login page HTML instead of
	// the file bytes. Treat that as auth failure rather than handing
	// the login form to a caller that expected a PDF.
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(strings.ToLower(ct), "text/html") {
		resp.Body.Close()
		return nil, &Error{
			Code:    ErrCodeAuth,
			Op:      op,
			Message: "attachment returned text/html (session expired?)",
		}
	}

	return resp.Body, nil
}

// attachmentPath normalizes rawURL to a path-only form that can be
// passed to Client.do.
func (c *Client) attachmentPath(rawURL string) (string, error) {
	if rawURL == "" {
		return "", errors.New("empty URL")
	}
	if strings.HasPrefix(rawURL, "/") {
		return rawURL, nil
	}
	expectedPrefix := c.baseURL + "/"
	if !strings.HasPrefix(rawURL, expectedPrefix) {
		return "", fmt.Errorf("URL does not match client host: %s", rawURL)
	}
	return rawURL[len(c.baseURL):], nil
}
