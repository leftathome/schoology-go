package schoology

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParseOverdueSubmissions_WithAssignments(t *testing.T) {
	raw, err := os.ReadFile("internal/testdata/html/overdue_submissions_with_assignments.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// Decode the envelope so we're exercising the exact HTML the wire
	// produces (with the JSON un-escaping applied).
	env := decodeEnvelope(t, raw)

	assignments, perrs := parseOverdueSubmissions(env.Body.HTML)
	if perrs != nil {
		t.Fatalf("unexpected parse errors: %v", perrs)
	}
	if len(assignments) != 2 {
		t.Fatalf("got %d assignments, want 2", len(assignments))
	}

	tests := []struct {
		idx    int
		id     int64
		title  string
		course string
		url    string
		unix   int64
	}{
		{0, 9100000001, "Week 4 lab worksheet", "Example Course A", "/assignment/9100000001", 1769759940},
		{1, 9100000002, "Chapter 7 reading response", "Example Course B", "/assignment/9100000002", 1771919940},
	}
	for _, tt := range tests {
		got := assignments[tt.idx]
		if got.ID != tt.id {
			t.Errorf("assignments[%d].ID = %d, want %d", tt.idx, got.ID, tt.id)
		}
		if got.Title != tt.title {
			t.Errorf("assignments[%d].Title = %q, want %q", tt.idx, got.Title, tt.title)
		}
		if got.CourseTitle != tt.course {
			t.Errorf("assignments[%d].CourseTitle = %q, want %q", tt.idx, got.CourseTitle, tt.course)
		}
		if got.URL != tt.url {
			t.Errorf("assignments[%d].URL = %q, want %q", tt.idx, got.URL, tt.url)
		}
		if got.Status != AssignmentStatusOverdue {
			t.Errorf("assignments[%d].Status = %q, want %q", tt.idx, got.Status, AssignmentStatusOverdue)
		}
		if got.DueAt.IsZero() {
			t.Errorf("assignments[%d].DueAt is zero", tt.idx)
		}
		if got.DueAt.Unix() != tt.unix {
			t.Errorf("assignments[%d].DueAt.Unix() = %d, want %d", tt.idx, got.DueAt.Unix(), tt.unix)
		}
		if got.DueAt.Location().String() != "UTC" {
			t.Errorf("assignments[%d].DueAt.Location() = %q, want UTC", tt.idx, got.DueAt.Location())
		}
	}
}

func TestParseOverdueSubmissions_Empty(t *testing.T) {
	raw, err := os.ReadFile("internal/testdata/html/overdue_submissions_empty.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	env := decodeEnvelope(t, raw)

	assignments, perrs := parseOverdueSubmissions(env.Body.HTML)
	if perrs != nil {
		t.Errorf("unexpected parse errors: %v", perrs)
	}
	if len(assignments) != 0 {
		t.Errorf("got %d assignments, want 0", len(assignments))
	}
}

func TestParseOverdueSubmissions_PartialMalformed(t *testing.T) {
	// First event is well-formed; second is missing data-start. We expect
	// the good row returned and a single parse error for the bad one.
	const html = `<div class='upcoming-events'>
  <h3 class='submissions-title'>OVERDUE</h3>
  <div class='upcoming-list'>
    <div class="upcoming-event" data-start="1769759940">
      <div class="upcoming-item-content">
        <span class="event-title">
          <a href="/assignment/9100000001">Good assignment</a>
          <span><span class='readonly-title event-subtitle'>1 day overdue</span><span class="readonly-title event-subtitle">Course A</span></span>
        </span>
      </div>
    </div>
    <div class="upcoming-event">
      <div class="upcoming-item-content">
        <span class="event-title">
          <a href="/assignment/9100000002">Missing data-start</a>
          <span><span class='readonly-title event-subtitle'>2 days overdue</span><span class="readonly-title event-subtitle">Course B</span></span>
        </span>
      </div>
    </div>
  </div>
</div>`

	assignments, perrs := parseOverdueSubmissions(html)
	if len(assignments) != 1 {
		t.Fatalf("got %d assignments, want 1", len(assignments))
	}
	if assignments[0].ID != 9100000001 {
		t.Errorf("assignments[0].ID = %d, want 9100000001", assignments[0].ID)
	}
	if len(perrs) != 1 {
		t.Fatalf("got %d parse errors, want 1: %v", len(perrs), perrs)
	}
	if perrs[0].Code != ErrCodeParse {
		t.Errorf("parse error code = %q, want %q", perrs[0].Code, ErrCodeParse)
	}
	if !IsParseError(perrs) {
		t.Errorf("IsParseError(perrs) = false, want true")
	}
}

func TestParseOverdueSubmissions_UpcomingHeading(t *testing.T) {
	// Assignments under an UPCOMING heading should get AssignmentStatusUpcoming.
	const html = `<div class='upcoming-events'>
  <h3 class='submissions-title'>UPCOMING</h3>
  <div class='upcoming-list'>
    <div class="upcoming-event" data-start="1800000000">
      <div class="upcoming-item-content">
        <span class="event-title">
          <a href="/assignment/9100000099">Future thing</a>
          <span><span class="readonly-title event-subtitle">Course X</span></span>
        </span>
      </div>
    </div>
  </div>
</div>`

	assignments, perrs := parseOverdueSubmissions(html)
	if perrs != nil {
		t.Fatalf("unexpected parse errors: %v", perrs)
	}
	if len(assignments) != 1 {
		t.Fatalf("got %d assignments, want 1", len(assignments))
	}
	if assignments[0].Status != AssignmentStatusUpcoming {
		t.Errorf("status = %q, want %q", assignments[0].Status, AssignmentStatusUpcoming)
	}
	if assignments[0].CourseTitle != "Course X" {
		t.Errorf("course = %q, want Course X", assignments[0].CourseTitle)
	}
}

func TestParseAssignmentID(t *testing.T) {
	tests := []struct {
		name string
		href string
		want int64
	}{
		{name: "path only", href: "/assignment/9100000001", want: 9100000001},
		{name: "trailing slash", href: "/assignment/42/", want: 42},
		{name: "with query", href: "/assignment/42?tab=foo", want: 42},
		{name: "not an assignment", href: "/course/1/gradebook", want: 0},
		{name: "empty", href: "", want: 0},
		{name: "non-numeric", href: "/assignment/abc", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAssignmentID(tt.href); got != tt.want {
				t.Errorf("parseAssignmentID(%q) = %d, want %d", tt.href, got, tt.want)
			}
		})
	}
}

func TestGetOverdueSubmissions_Mock(t *testing.T) {
	mockData, err := os.ReadFile("internal/testdata/html/overdue_submissions_with_assignments.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	const childUID int64 = 130401977
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/iapi/parent/overdue_submissions/130401977" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("X-CSRF-Token") == "" {
			t.Error("missing X-CSRF-Token header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mockData)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	assignments, perrs, err := client.GetOverdueSubmissions(context.Background(), childUID)
	if err != nil {
		t.Fatalf("GetOverdueSubmissions: %v", err)
	}
	if perrs != nil {
		t.Fatalf("unexpected parse errors: %v", perrs)
	}
	if len(assignments) != 2 {
		t.Fatalf("got %d assignments, want 2", len(assignments))
	}
	if assignments[0].ID != 9100000001 {
		t.Errorf("assignments[0].ID = %d, want 9100000001", assignments[0].ID)
	}
	if assignments[1].ID != 9100000002 {
		t.Errorf("assignments[1].ID = %d, want 9100000002", assignments[1].ID)
	}
}

func TestGetOverdueSubmissions_Empty(t *testing.T) {
	mockData, err := os.ReadFile("internal/testdata/html/overdue_submissions_empty.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mockData)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	assignments, perrs, err := client.GetOverdueSubmissions(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetOverdueSubmissions: %v", err)
	}
	if perrs != nil {
		t.Errorf("parse errs = %v, want nil", perrs)
	}
	if len(assignments) != 0 {
		t.Errorf("assignments = %v, want empty", assignments)
	}
}

// decodeEnvelope un-escapes the overdue_submissions JSON envelope and
// returns the embedded HTML string for parser tests.
func decodeEnvelope(t *testing.T, raw []byte) overdueSubmissionsEnvelope {
	t.Helper()
	var env overdueSubmissionsEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env
}
