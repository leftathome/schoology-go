package schoology

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func loadFeedFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("internal/testdata/html/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var env feedEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", name, err)
	}
	return env.Output
}

func TestParseFeed_Fixture(t *testing.T) {
	output := loadFeedFixture(t, "home_feed.json")

	posts, perrs := parseFeed(output)
	if perrs != nil {
		t.Fatalf("unexpected parse errors: %v", perrs)
	}
	if len(posts) != 2 {
		t.Fatalf("len(posts) = %d, want 2", len(posts))
	}

	// First post.
	p0 := posts[0]
	if p0.EdgeID != "44000000001" {
		t.Errorf("p0.EdgeID = %q, want 44000000001", p0.EdgeID)
	}
	if p0.Timestamp.Unix() != 1774975277 {
		t.Errorf("p0.Timestamp.Unix() = %d, want 1774975277", p0.Timestamp.Unix())
	}
	if p0.Timestamp.Location().String() != "UTC" {
		t.Errorf("p0.Timestamp location = %q, want UTC", p0.Timestamp.Location())
	}
	if p0.AuthorName != "Example Teacher" {
		t.Errorf("p0.AuthorName = %q, want Example Teacher", p0.AuthorName)
	}
	if p0.AuthorUID != 1000000001 {
		t.Errorf("p0.AuthorUID = %d, want 1000000001", p0.AuthorUID)
	}
	if p0.PostedTo != "Example Course A" {
		t.Errorf("p0.PostedTo = %q, want Example Course A", p0.PostedTo)
	}
	if p0.PostedToURL != "/course/2000000001" {
		t.Errorf("p0.PostedToURL = %q, want /course/2000000001", p0.PostedToURL)
	}
	wantBody := "Please see the attached flyer for this week's activity bus schedule."
	if p0.Body != wantBody {
		t.Errorf("p0.Body = %q, want %q", p0.Body, wantBody)
	}
	if len(p0.Attachments) != 1 {
		t.Fatalf("p0 attachments = %d, want 1", len(p0.Attachments))
	}
	a0 := p0.Attachments[0]
	if a0.ID != 3000000001 {
		t.Errorf("a0.ID = %d, want 3000000001", a0.ID)
	}
	if a0.Filename != "example_flyer.pdf" {
		t.Errorf("a0.Filename = %q, want example_flyer.pdf", a0.Filename)
	}
	if a0.MimeType != "Adobe PDF" {
		t.Errorf("a0.MimeType = %q, want Adobe PDF", a0.MimeType)
	}
	if !strings.HasPrefix(a0.URL, "/attachment/3000000001/source/") {
		t.Errorf("a0.URL = %q, want /attachment/3000000001/source/...", a0.URL)
	}

	// Second post.
	p1 := posts[1]
	if p1.EdgeID != "44000000002" {
		t.Errorf("p1.EdgeID = %q, want 44000000002", p1.EdgeID)
	}
	if p1.Timestamp.Unix() != 1774800000 {
		t.Errorf("p1.Timestamp.Unix() = %d, want 1774800000", p1.Timestamp.Unix())
	}
	if p1.AuthorName != "Another Teacher" {
		t.Errorf("p1.AuthorName = %q, want Another Teacher", p1.AuthorName)
	}
	if p1.AuthorUID != 1000000002 {
		t.Errorf("p1.AuthorUID = %d, want 1000000002", p1.AuthorUID)
	}
	if p1.PostedTo != "Example Course B" {
		t.Errorf("p1.PostedTo = %q, want Example Course B", p1.PostedTo)
	}
	if p1.PostedToURL != "/course/2000000002" {
		t.Errorf("p1.PostedToURL = %q, want /course/2000000002", p1.PostedToURL)
	}
	if p1.Body != "Quick reminder: quiz on Friday." {
		t.Errorf("p1.Body = %q, want Quick reminder: quiz on Friday.", p1.Body)
	}
	if len(p1.Attachments) != 0 {
		t.Errorf("p1 attachments = %d, want 0", len(p1.Attachments))
	}
}

func TestParseFeed_Empty(t *testing.T) {
	output := loadFeedFixture(t, "home_feed_empty.json")
	posts, perrs := parseFeed(output)
	if len(posts) != 0 {
		t.Errorf("len(posts) = %d, want 0", len(posts))
	}
	if perrs != nil {
		t.Errorf("perrs = %v, want nil", perrs)
	}
}

func TestParseFeed_MalformedPartialSuccess(t *testing.T) {
	// One good li, one missing timestamp, one missing edge-assoc id.
	html := `<div class="item-list"><ul class="s-edge-feed">
<li id="edge-assoc-77" timestamp="1700000000">
  <div class="edge-left"><div class="profile-picture">
    <a href="/user/42" title="Good Author"><img alt="Good Author" /></a>
  </div></div>
  <div class="update-sentence-inner"><a href="/course/9">Some Course</a></div>
  <span class="update-body"><p>hello</p></span>
</li>
<li id="edge-assoc-88">
  <div class="edge-left"><div class="profile-picture">
    <a href="/user/1" title="No Timestamp"><img alt="x"/></a>
  </div></div>
  <span class="update-body"><p>no ts</p></span>
</li>
<li id="not-an-edge-id" timestamp="1700000000">
  <span class="update-body"><p>bad id</p></span>
</li>
</ul></div>`

	posts, perrs := parseFeed(html)
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1", len(posts))
	}
	if posts[0].EdgeID != "77" {
		t.Errorf("posts[0].EdgeID = %q, want 77", posts[0].EdgeID)
	}
	if posts[0].Body != "hello" {
		t.Errorf("posts[0].Body = %q, want hello", posts[0].Body)
	}
	if len(perrs) != 2 {
		t.Fatalf("len(perrs) = %d, want 2, errs=%v", len(perrs), perrs)
	}
	if !IsParseError(perrs.AsError()) {
		t.Error("parse errors should be identifiable via IsParseError")
	}
}

func TestGetFeed_Mock(t *testing.T) {
	mockData, err := os.ReadFile("internal/testdata/html/home_feed.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/home/feed" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.URL.Query().Get("children"); got != "555" {
			t.Errorf("children query = %q, want 555", got)
		}
		if r.Header.Get("X-CSRF-Token") == "" {
			t.Error("missing X-CSRF-Token header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mockData)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	posts, perrs, err := client.GetFeed(context.Background(), 555)
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if perrs != nil {
		t.Errorf("perrs = %v, want nil", perrs)
	}
	if len(posts) != 2 {
		t.Fatalf("len(posts) = %d, want 2", len(posts))
	}
	if posts[0].AuthorName != "Example Teacher" {
		t.Errorf("posts[0].AuthorName = %q, want Example Teacher", posts[0].AuthorName)
	}
	if len(posts[0].Attachments) != 1 {
		t.Errorf("posts[0].Attachments = %d, want 1", len(posts[0].Attachments))
	}
}

func TestGetFeed_Empty(t *testing.T) {
	mockData, err := os.ReadFile("internal/testdata/html/home_feed_empty.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mockData)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	posts, perrs, err := client.GetFeed(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if perrs != nil {
		t.Errorf("perrs = %v, want nil", perrs)
	}
	if len(posts) != 0 {
		t.Errorf("len(posts) = %d, want 0", len(posts))
	}
}

func TestParseUserID(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"/user/123", 123},
		{"/user/456/something", 456},
		{"https://x.schoology.com/user/789", 789},
		{"/course/1", 0},
		{"", 0},
		{"/user/abc", 0},
	}
	for _, tt := range tests {
		if got := parseUserID(tt.in); got != tt.want {
			t.Errorf("parseUserID(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestCollapseWhitespace(t *testing.T) {
	tests := []struct{ in, want string }{
		{"  hello   world  ", "hello world"},
		{"a\nb\tc", "a b c"},
		{"", ""},
		{"single", "single"},
	}
	for _, tt := range tests {
		if got := collapseWhitespace(tt.in); got != tt.want {
			t.Errorf("collapseWhitespace(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
