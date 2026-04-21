package schoology

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/leftathome/schoology-go/internal/htmlfetch"
)

// MessageThread is a conversation in the inbox — subject, counterparty,
// and last-activity metadata. The message bodies themselves are not
// available from the inbox list; callers fetch an individual thread
// separately (out of scope for v0.1.0).
type MessageThread struct {
	// ID is the Schoology thread id, or 0 when the inbox row did not
	// surface one.
	ID int64

	// Subject is the plain-text thread subject.
	Subject string

	// SenderName is the display name of the counterparty on the thread.
	SenderName string

	// SenderUID is the Schoology user id of the counterparty, or 0 if
	// the inbox row did not link to a user profile.
	SenderUID int64

	// LastActivity is the parsed last-activity timestamp. Zero value
	// when the date cell could not be parsed.
	LastActivity time.Time

	// Unread is true when the row is styled as unread (class "unread").
	Unread bool

	// MessageCount is the number of messages in the thread when a count
	// badge is present; 0 otherwise.
	MessageCount int

	// URL is the path to the thread page (e.g. "/messages/thread/123"),
	// empty when unknown.
	URL string
}

// inboxDateLayouts are the layouts parseInbox tries against the date
// cell. They're best-effort — on miss we leave LastActivity zero.
var inboxDateLayouts = []string{
	"Mon Jan 2, 2006 at 3:04 pm",
	"Mon Jan 2, 2006 at 3:04pm",
	"Jan 2, 2006 at 3:04 pm",
	"Jan 2, 2006",
}

// GetInbox fetches and parses /messages/inbox.
//
// Returns (threads, parseErrs, err):
//   - threads: the successfully parsed threads (may be partial / empty)
//   - parseErrs: nil-or-non-nil ParseErrors for per-row failures
//   - err: non-nil only for hard failures (HTTP, auth, non-HTML response)
func (c *Client) GetInbox(ctx context.Context) ([]*MessageThread, ParseErrors, error) {
	const op = "GetInbox"

	resp, err := c.do(ctx, http.MethodGet, "/messages/inbox", nil)
	if err != nil {
		return nil, nil, withOp(op, err)
	}

	doc, err := htmlfetch.Parse(resp)
	if err != nil {
		switch {
		case errors.Is(err, htmlfetch.ErrLoginRedirect):
			return nil, nil, &Error{
				Code:    ErrCodeAuth,
				Op:      op,
				Message: "session expired (login page returned)",
				Err:     err,
			}
		case errors.Is(err, htmlfetch.ErrNotHTML):
			return nil, nil, &Error{
				Code:    ErrCodeServer,
				Op:      op,
				Message: "response was not text/html",
				Err:     err,
			}
		default:
			return nil, nil, &Error{
				Code:    ErrCodeParse,
				Op:      op,
				Message: "failed to parse HTML",
				Err:     err,
			}
		}
	}

	threads, perrs := parseInboxDoc(doc)
	return threads, perrs, nil
}

// parseInbox is a thin wrapper around parseInboxDoc that accepts a
// raw HTML string; tests use it with the committed fixtures.
//
// TODO(schoology-go-m7i-followup): the filled-row selectors in
// parseInboxRow are a best-effort guess based on Schoology's
// Drupal-7 patterns and the empty-state fixture's table header.
// They are NOT verified against real filled-inbox HTML because the
// capture account's inbox was empty. When a filled fixture is
// available, confirm / update the selectors and add a dedicated test.
func parseInbox(html string) ([]*MessageThread, ParseErrors) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		var perrs ParseErrors
		perrs.Append(NewParseError("parseInbox", "failed to build document: "+err.Error()))
		return nil, perrs
	}
	return parseInboxDoc(doc)
}

// parseInboxDoc walks the messages-inbox table in doc and returns
// the threads it could parse plus a ParseErrors for per-row
// failures. An empty inbox returns (nil, nil).
func parseInboxDoc(doc *goquery.Document) ([]*MessageThread, ParseErrors) {
	var (
		threads []*MessageThread
		perrs   ParseErrors
	)

	rows := doc.Find("table.messages-table tbody tr")

	rows.Each(func(_ int, s *goquery.Selection) {
		// Empty-state row: a single td with a colspan. Return early —
		// this is a success signal, not an error.
		if s.HasClass("empty-state") {
			return
		}

		// Best-effort filled-row parse. Any failure turns into a
		// per-row ParseError; we keep going rather than short-circuit.
		t, perr := parseInboxRow(s)
		if perr != nil {
			perrs.Append(perr)
			return
		}
		if t != nil {
			threads = append(threads, t)
		}
	})

	if len(perrs) == 0 {
		return threads, nil
	}
	return threads, perrs
}

// parseInboxRow turns a single <tr> into a MessageThread using the
// best-effort selectors documented above. Returns (nil, nil) for rows
// that don't look like thread rows (e.g. header rows slipping in).
func parseInboxRow(s *goquery.Selection) (*MessageThread, *Error) {
	tds := s.Find("td")
	if tds.Length() == 0 {
		// Skip header / spacer rows silently.
		return nil, nil
	}

	t := &MessageThread{
		Unread: s.HasClass("unread"),
	}

	// Subject cell: first td, usually contains <a href="/messages/thread/{id}">Subject</a>.
	subjectCell := tds.Eq(0)
	subjectLink := subjectCell.Find("a").First()
	if subjectLink.Length() > 0 {
		t.Subject = strings.TrimSpace(subjectLink.Text())
		if href, ok := subjectLink.Attr("href"); ok {
			t.URL = href
			t.ID = parsePathID(href, "/messages/thread/")
		}
	} else {
		t.Subject = strings.TrimSpace(subjectCell.Text())
	}

	if t.Subject == "" {
		return nil, NewParseError("parseInbox: tr", "missing subject")
	}

	// Sender cell: second td, usually <a href="/user/{uid}">Name</a>.
	if tds.Length() >= 2 {
		senderCell := tds.Eq(1)
		senderLink := senderCell.Find("a").First()
		if senderLink.Length() > 0 {
			t.SenderName = strings.TrimSpace(senderLink.Text())
			if href, ok := senderLink.Attr("href"); ok {
				t.SenderUID = parseUserID(href)
			}
		} else {
			t.SenderName = strings.TrimSpace(senderCell.Text())
		}
	}

	// Date cell: third td, free text.
	if tds.Length() >= 3 {
		dateText := strings.TrimSpace(tds.Eq(2).Text())
		for _, layout := range inboxDateLayouts {
			if ts, err := time.Parse(layout, dateText); err == nil {
				t.LastActivity = ts
				break
			}
		}
	}

	// Message-count badge: look for a count-style element inside the row.
	if badge := s.Find(".message-count, .count, .badge").First(); badge.Length() > 0 {
		if n, err := strconv.Atoi(strings.TrimSpace(badge.Text())); err == nil {
			t.MessageCount = n
		}
	}

	return t, nil
}
