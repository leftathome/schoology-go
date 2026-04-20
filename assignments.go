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

// Assignment represents a single upcoming or overdue assignment as
// rendered by /iapi/parent/overdue_submissions/{child_uid}.
type Assignment struct {
	// ID is Schoology's numeric assignment id, parsed from the
	// /assignment/{id} href.
	ID int64

	// Title is the assignment link text.
	Title string

	// CourseTitle is the course name (e.g. "Example Course A").
	CourseTitle string

	// DueAt is the due time, from the data-start unix seconds on the
	// event div. Stored in UTC.
	DueAt time.Time

	// URL is the path-only link to the assignment
	// (e.g. "/assignment/9100000001").
	URL string

	// Status indicates whether the assignment is Overdue or Upcoming.
	Status AssignmentStatus
}

// AssignmentStatus is whether an assignment is overdue or upcoming,
// derived from the section heading preceding the event in the HTML.
type AssignmentStatus string

const (
	AssignmentStatusOverdue  AssignmentStatus = "overdue"
	AssignmentStatusUpcoming AssignmentStatus = "upcoming"
)

// overdueSubmissionsEnvelope is the outer shape of the
// /iapi/parent/overdue_submissions/{child_uid} response:
// {"response_code":200,"body":{"html":"<escaped HTML>"}}.
type overdueSubmissionsEnvelope struct {
	ResponseCode int `json:"response_code"`
	Body         struct {
		HTML string `json:"html"`
	} `json:"body"`
}

// assignmentIDRe extracts the numeric id from a Schoology assignment URL
// of the form /assignment/{id}.
var assignmentIDRe = regexp.MustCompile(`^/assignment/(\d+)(?:[/?#]|$)`)

// GetOverdueSubmissions fetches and parses the overdue/upcoming
// submissions page for the given child UID.
//
// Returns (assignments, parseErrs, err):
//   - assignments: the successfully parsed items (may be partial)
//   - parseErrs: a nil-or-non-nil ParseErrors collecting per-item
//     failures; the operation still "succeeded" if this is non-nil
//   - err: non-nil only for hard failures (HTTP, JSON decode, auth)
func (c *Client) GetOverdueSubmissions(ctx context.Context, childUID int64) ([]*Assignment, ParseErrors, error) {
	const op = "GetOverdueSubmissions"

	path := fmt.Sprintf("/iapi/parent/overdue_submissions/%d", childUID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, nil, err
	}

	var env overdueSubmissionsEnvelope
	if err := decodeJSON(resp, &env); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, nil, err
	}

	assignments, perrs := parseOverdueSubmissions(env.Body.HTML)
	return assignments, perrs, nil
}

// parseOverdueSubmissions parses the HTML fragment from the
// overdue_submissions envelope into a slice of *Assignment. It returns
// the successfully parsed rows plus a ParseErrors collecting per-item
// failures; a single malformed row does not fail the whole page.
//
// Empty input returns (nil, nil).
func parseOverdueSubmissions(html string) ([]*Assignment, ParseErrors) {
	if strings.TrimSpace(html) == "" {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		var perrs ParseErrors
		perrs.Append(NewParseError("parseOverdueSubmissions", "failed to parse HTML: "+err.Error()))
		return nil, perrs
	}

	var (
		out   []*Assignment
		perrs ParseErrors
	)

	// Walk all nodes inside each upcoming-events container in document
	// order so we can track which h3.submissions-title section each
	// upcoming-event belongs to.
	doc.Find(".upcoming-events").Each(func(_ int, container *goquery.Selection) {
		status := AssignmentStatusUpcoming
		container.Find("h3.submissions-title, .upcoming-event").Each(func(_ int, sel *goquery.Selection) {
			if sel.Is("h3.submissions-title") {
				status = statusFromHeading(sel.Text())
				return
			}
			a, perr := parseUpcomingEvent(sel, status)
			if perr != nil {
				perrs.Append(perr)
				return
			}
			out = append(out, a)
		})
	})

	if len(perrs) == 0 {
		return out, nil
	}
	return out, perrs
}

// statusFromHeading maps an h3.submissions-title's text to an
// AssignmentStatus. "OVERDUE" (case-insensitive) maps to Overdue; any
// other text (typically "UPCOMING") maps to Upcoming.
func statusFromHeading(text string) AssignmentStatus {
	if strings.EqualFold(strings.TrimSpace(text), "OVERDUE") {
		return AssignmentStatusOverdue
	}
	return AssignmentStatusUpcoming
}

// parseUpcomingEvent extracts a single Assignment from a
// .upcoming-event element. Returns a *Error with ErrCodeParse for
// rows that are missing required fields (link, id, data-start).
func parseUpcomingEvent(event *goquery.Selection, status AssignmentStatus) (*Assignment, *Error) {
	const op = "parseOverdueSubmissions: .upcoming-event"

	link := event.Find(`.event-title a[href^="/assignment/"]`).First()
	if link.Length() == 0 {
		return nil, NewParseError(op, "missing assignment link")
	}
	href, _ := link.Attr("href")
	id := parseAssignmentID(href)
	if id == 0 {
		return nil, NewParseError(op, "could not parse assignment id from href: "+href)
	}

	title := strings.TrimSpace(link.Text())
	if title == "" {
		return nil, NewParseError(op, fmt.Sprintf("assignment %d has empty title", id))
	}

	dataStart, ok := event.Attr("data-start")
	if !ok || strings.TrimSpace(dataStart) == "" {
		return nil, NewParseError(op, fmt.Sprintf("assignment %d missing data-start attribute", id))
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(dataStart), 10, 64)
	if err != nil {
		return nil, NewParseError(op, fmt.Sprintf("assignment %d has invalid data-start %q: %v", id, dataStart, err))
	}

	// Course name: the last .readonly-title.event-subtitle inside
	// .event-title is the course name; the first is the "N days overdue"
	// label. Be defensive: if there's only one, use it.
	subtitles := event.Find(".event-title .readonly-title.event-subtitle")
	var course string
	if n := subtitles.Length(); n > 0 {
		course = strings.TrimSpace(subtitles.Eq(n - 1).Text())
	}

	return &Assignment{
		ID:          id,
		Title:       title,
		CourseTitle: course,
		DueAt:       time.Unix(secs, 0).UTC(),
		URL:         href,
		Status:      status,
	}, nil
}

// parseAssignmentID returns the numeric assignment id embedded in href,
// or 0 if href does not match /assignment/{id}.
func parseAssignmentID(href string) int64 {
	m := assignmentIDRe.FindStringSubmatch(href)
	if m == nil {
		return 0
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0
	}
	return id
}
