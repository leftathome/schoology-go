package schoology

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Post is a single entry in the parent home feed — typically a
// faculty update posted to a course or group, with optional
// attachments.
type Post struct {
	EdgeID      string        // stable id from li[id="edge-assoc-<id>"]
	Timestamp   time.Time     // from li[timestamp] unix seconds
	AuthorName  string        // from the link text / alt attribute
	AuthorUID   int64         // parsed from /user/{uid} href
	Body        string        // text of .update-body (whitespace-collapsed)
	PostedTo    string        // link text of .update-sentence-inner a[href^="/course/"]
	PostedToURL string        // the href itself
	Attachments []*Attachment // from .attachments-file
}

// feedEnvelope is the JSON wrapper /home/feed returns; the HTML we
// actually care about is escaped into the "output" field.
type feedEnvelope struct {
	Output string `json:"output"`
}

// whitespaceRe collapses runs of whitespace to a single space.
var whitespaceRe = regexp.MustCompile(`\s+`)

// GetFeed fetches and parses the home-feed page scoped to childUID.
//
// Returns (posts, parseErrs, err):
//   - posts: successfully parsed posts (may be partial)
//   - parseErrs: nil-or-non-nil ParseErrors for per-item failures
//   - err: non-nil only for hard failures (HTTP, JSON decode, auth)
func (c *Client) GetFeed(ctx context.Context, childUID int64) ([]*Post, ParseErrors, error) {
	const op = "GetFeed"

	path := fmt.Sprintf("/home/feed?children=%d", childUID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, withOp(op, err)
	}

	var env feedEnvelope
	if err := decodeJSON(resp, &env); err != nil {
		return nil, nil, withOp(op, err)
	}

	posts, perrs := parseFeed(env.Output)
	return posts, perrs, nil
}

// parseFeed reads a home-feed HTML fragment and returns the parsed
// posts alongside any per-item parse errors.
func parseFeed(html string) ([]*Post, ParseErrors) {
	const op = "parseFeed"

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		var perrs ParseErrors
		perrs.Append(NewParseError(op, "goquery: "+err.Error()))
		return nil, perrs
	}

	var (
		posts []*Post
		perrs ParseErrors
	)

	doc.Find("ul.s-edge-feed > li").Each(func(i int, li *goquery.Selection) {
		post, err := parseFeedItem(li)
		if err != nil {
			perrs.Append(err)
			return
		}
		posts = append(posts, post)
	})

	return posts, perrs
}

// parseFeedItem turns a single <li> into a *Post, or returns a
// *Error describing why the item could not be parsed.
func parseFeedItem(li *goquery.Selection) (*Post, *Error) {
	const op = "parseFeed: li"

	rawID, ok := li.Attr("id")
	if !ok || !strings.HasPrefix(rawID, "edge-assoc-") {
		return nil, NewParseError(op, "missing or malformed edge-assoc id")
	}
	edgeID := strings.TrimPrefix(rawID, "edge-assoc-")
	if edgeID == "" {
		return nil, NewParseError(op, "empty edge-assoc id")
	}

	tsAttr, ok := li.Attr("timestamp")
	if !ok {
		return nil, NewParseError(op, "missing timestamp attr on li#"+rawID)
	}
	tsInt, err := strconv.ParseInt(tsAttr, 10, 64)
	if err != nil {
		return nil, NewParseError(op, "invalid timestamp "+strconv.Quote(tsAttr)+" on li#"+rawID)
	}
	ts := time.Unix(tsInt, 0).UTC()

	post := &Post{
		EdgeID:    edgeID,
		Timestamp: ts,
	}

	// Author from the profile-picture link.
	authorLink := li.Find(".edge-left .profile-picture a").First()
	if authorLink.Length() > 0 {
		if href, ok := authorLink.Attr("href"); ok {
			post.AuthorUID = parseUserID(href)
		}
		if title, ok := authorLink.Attr("title"); ok && title != "" {
			post.AuthorName = title
		} else if alt, ok := authorLink.Find("img").First().Attr("alt"); ok {
			post.AuthorName = alt
		}
	}

	// Body text, whitespace-collapsed.
	if body := li.Find(".update-body").First(); body.Length() > 0 {
		post.Body = collapseWhitespace(body.Text())
	}

	// PostedTo course link (first /course/ anchor inside update-sentence-inner).
	courseLink := li.Find(`.update-sentence-inner a[href^="/course/"]`).First()
	if courseLink.Length() > 0 {
		post.PostedTo = collapseWhitespace(courseLink.Text())
		if href, ok := courseLink.Attr("href"); ok {
			post.PostedToURL = href
		}
	}

	// Attachments.
	li.Find(".attachments .attachments-file").Each(func(_ int, af *goquery.Selection) {
		att := parseFeedAttachment(af)
		if att != nil {
			post.Attachments = append(post.Attachments, att)
		}
	})

	return post, nil
}

// parseFeedAttachment turns a .attachments-file node into an
// *Attachment. Returns nil when the node has no usable URL.
func parseFeedAttachment(af *goquery.Selection) *Attachment {
	nameLink := af.Find(".attachments-file-name a").First()
	if nameLink.Length() == 0 {
		return nil
	}
	href, ok := nameLink.Attr("href")
	if !ok || href == "" {
		return nil
	}

	att := &Attachment{
		URL: href,
		ID:  parseAttachmentID(href),
	}

	if label, ok := nameLink.Attr("aria-label"); ok && label != "" {
		att.Filename = label
	} else {
		att.Filename = collapseWhitespace(nameLink.Text())
	}

	if mime := strings.TrimSpace(af.Find(".attachments-file-icon .visually-hidden").First().Text()); mime != "" {
		att.MimeType = mime
	}

	return att
}

// collapseWhitespace trims and folds runs of whitespace to a single
// space. Used for body text and link labels pulled from HTML.
func collapseWhitespace(s string) string {
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(s, " "))
}
