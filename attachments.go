package schoology

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Attachment describes a downloadable file associated with a message
// thread, feed post, or other resource. Schoology's attachment URL
// has the shape /attachment/{id}/source/{hash}.{ext}?checkad=1;
// ID is parsed from that URL, while Filename, MimeType, and Size
// come from the parent resource's HTML (and may be zero-valued when
// the page doesn't surface them).
type Attachment struct {
	// ID is Schoology's numeric attachment id, parsed from the
	// /attachment/{id}/... URL.
	ID int64

	// Filename is the display name (e.g. "flyer.pdf"), not the
	// hashed name in the URL path.
	Filename string

	// URL is the fully-qualified or path-relative source URL,
	// including the ?checkad=1 tail. Callers pass this to
	// DownloadAttachment as-is.
	URL string

	// MimeType is a best-effort content type — "application/pdf",
	// "image/jpeg", etc. Empty when the HTML didn't label it.
	MimeType string

	// Size is the file size in bytes, or 0 if unknown. Schoology's
	// HTML pages rarely surface this.
	Size int64
}

// attachmentIDRe extracts the numeric id from a Schoology attachment
// URL. Works for absolute ("https://x.schoology.com/attachment/123/...")
// and path-only ("/attachment/123/...") forms.
var attachmentIDRe = regexp.MustCompile(`/attachment/(\d+)(?:/|$)`)

// parseAttachmentID returns the numeric attachment id embedded in rawURL,
// or 0 if the URL does not match /attachment/{id}/... .
func parseAttachmentID(rawURL string) int64 {
	m := attachmentIDRe.FindStringSubmatch(rawURL)
	if m == nil {
		return 0
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

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
// with an auth-code error.
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
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	return resp.Body, nil
}

// attachmentPath normalizes rawURL to a path-only form that can be
// passed to Client.do.
func (c *Client) attachmentPath(rawURL string) (string, error) {
	if rawURL == "" {
		return "", errAttachmentURL("empty URL")
	}
	if strings.HasPrefix(rawURL, "/") {
		return rawURL, nil
	}
	expectedPrefix := c.baseURL + "/"
	if !strings.HasPrefix(rawURL, expectedPrefix) {
		return "", errAttachmentURL("URL does not match client host: " + rawURL)
	}
	return rawURL[len(c.baseURL):], nil
}

type attachmentURLErr string

func (e attachmentURLErr) Error() string { return string(e) }

func errAttachmentURL(msg string) error { return attachmentURLErr(msg) }
