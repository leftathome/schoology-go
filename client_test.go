package schoology

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{name: "valid host", host: "school.schoology.com", wantErr: false},
		{name: "host with https prefix", host: "https://school.schoology.com", wantErr: false},
		{name: "empty host", host: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.host)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if client == nil {
				t.Error("expected client, got nil")
			}
		})
	}
}

func TestWithSession(t *testing.T) {
	tests := []struct {
		name      string
		sessID    string
		csrfToken string
		csrfKey   string
		uid       string
		wantErr   bool
	}{
		{name: "valid session", sessID: "test-session-id", csrfToken: "test-csrf-token", csrfKey: "test-csrf-key", uid: "12345", wantErr: false},
		{name: "missing sessID", sessID: "", csrfToken: "test-csrf-token", csrfKey: "test-csrf-key", uid: "12345", wantErr: true},
		{name: "missing csrfToken", sessID: "test-session-id", csrfToken: "", csrfKey: "test-csrf-key", uid: "12345", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(
				"school.schoology.com",
				WithSession(tt.sessID, tt.csrfToken, tt.csrfKey, tt.uid),
			)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *Client
		want        bool
	}{
		{
			name: "no session",
			setupClient: func() *Client {
				c, _ := NewClient("school.schoology.com")
				return c
			},
			want: false,
		},
		{
			name: "valid session",
			setupClient: func() *Client {
				c, _ := NewClient(
					"school.schoology.com",
					WithSession("sess", "token", "key", "uid"),
				)
				return c
			},
			want: true,
		},
		{
			name: "expired session",
			setupClient: func() *Client {
				c, _ := NewClient(
					"school.schoology.com",
					WithSession("sess", "token", "key", "uid"),
				)
				c.session.ExpiresAt = time.Now().Add(-1 * time.Hour)
				return c
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			if got := client.IsAuthenticated(); got != tt.want {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSessCookieName(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		// "SESS" + md5(host) — formula was validated against a live
		// Schoology session; these expected values come straight from md5.
		{name: "schoology tld", host: "schoology.com", want: "SESS0f255c5e53fbffbe66e5cfcd82ab81b4"},
		{name: "example subdomain", host: "example.schoology.com", want: "SESS0f3c278d8cbcca12eab60706abd3d4f3"},
		{name: "short host", host: "x", want: "SESS9dd4e461268c8034f5c8564e155c67a6"},
		{name: "empty host", host: "", want: "SESSd41d8cd98f00b204e9800998ecf8427e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessCookieName(tt.host); got != tt.want {
				t.Errorf("sessCookieName(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestUpdateSession_ShortSessID(t *testing.T) {
	// Regression: the old sessCookieName sliced sessID[:8] and would panic
	// on short IDs. It now derives the cookie name from the host, so any
	// sessID length must work.
	c, err := NewClient(
		"school.schoology.com",
		WithSession("longsess", "token", "key", "uid"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := c.UpdateSession("abc", "token2", "key2", "uid2"); err != nil {
		t.Fatalf("UpdateSession with short ID: %v", err)
	}
	if c.session.SessID != "abc" {
		t.Errorf("expected session ID 'abc', got %q", c.session.SessID)
	}
}

// withMockedBase points the client at a test server by rewriting baseURL.
// It does not re-run WithSession, so cookies previously jarred against
// the prod host won't automatically attach — tests below rely on the
// CSRF headers which the server inspects instead.
func withMockedBase(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := NewClient(
		"school.schoology.com",
		WithSession("test-sess", "test-token", "test-key", "12345"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.baseURL = server.URL
	return client
}

func TestGetCourses_Mock(t *testing.T) {
	mockData, err := os.ReadFile("internal/testdata/courses.json")
	if err != nil {
		t.Fatalf("failed to read mock data: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/iapi2/site-navigation/courses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("X-CSRF-Token") == "" {
			t.Error("missing X-CSRF-Token header")
		}
		if r.Header.Get("X-CSRF-Key") == "" {
			t.Error("missing X-CSRF-Key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mockData)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	ctx := context.Background()
	courses, err := client.GetCourses(ctx)
	if err != nil {
		t.Fatalf("GetCourses() error = %v", err)
	}
	if len(courses) != 2 {
		t.Fatalf("expected 2 courses, got %d", len(courses))
	}
	if courses[0].CourseTitle != "Mathematics 7" {
		t.Errorf("expected course[0].CourseTitle = 'Mathematics 7', got %q", courses[0].CourseTitle)
	}
	if courses[0].NID != 7978424918 {
		t.Errorf("expected course[0].NID = 7978424918, got %d", courses[0].NID)
	}
	if courses[0].CourseNID != 1252139350 {
		t.Errorf("expected course[0].CourseNID = 1252139350, got %d", courses[0].CourseNID)
	}
	if courses[1].SectionTitle != "S2 7(B) 2201" {
		t.Errorf("expected course[1].SectionTitle = 'S2 7(B) 2201', got %q", courses[1].SectionTitle)
	}
}

func TestGetChildren_Mock(t *testing.T) {
	mockData, err := os.ReadFile("internal/testdata/parent_info.json")
	if err != nil {
		t.Fatalf("failed to read mock data: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/iapi/parent/info" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mockData)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	children, err := client.GetChildren(context.Background())
	if err != nil {
		t.Fatalf("GetChildren() error = %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	// Sorted by UID, so lower UID first.
	if children[0].UID != 130401977 {
		t.Errorf("expected children[0].UID = 130401977, got %d", children[0].UID)
	}
	if children[1].UID != 130401987 {
		t.Errorf("expected children[1].UID = 130401987, got %d", children[1].UID)
	}
}

func TestCheckResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		wantCode   ErrorCode
	}{
		{name: "success 200", statusCode: http.StatusOK, wantErr: false},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, wantErr: true, wantCode: ErrCodeAuth},
		{name: "forbidden", statusCode: http.StatusForbidden, wantErr: true, wantCode: ErrCodePermission},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true, wantCode: ErrCodeNotFound},
		{name: "rate limited", statusCode: http.StatusTooManyRequests, wantErr: true, wantCode: ErrCodeRateLimit},
		{name: "server error", statusCode: http.StatusInternalServerError, wantErr: true, wantCode: ErrCodeServer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode, Header: http.Header{}}
			err := checkResponse(resp)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				e, ok := err.(*Error)
				if !ok {
					t.Errorf("expected *Error, got %T", err)
					return
				}
				if e.Code != tt.wantCode {
					t.Errorf("expected error code %s, got %s", tt.wantCode, e.Code)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
