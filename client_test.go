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
		{
			name:    "valid host",
			host:    "school.schoology.com",
			wantErr: false,
		},
		{
			name:    "host with https prefix",
			host:    "https://school.schoology.com",
			wantErr: false,
		},
		{
			name:    "empty host",
			host:    "",
			wantErr: true,
		},
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
		{
			name:      "valid session",
			sessID:    "test-session-id",
			csrfToken: "test-csrf-token",
			csrfKey:   "test-csrf-key",
			uid:       "12345",
			wantErr:   false,
		},
		{
			name:      "missing sessID",
			sessID:    "",
			csrfToken: "test-csrf-token",
			csrfKey:   "test-csrf-key",
			uid:       "12345",
			wantErr:   true,
		},
		{
			name:      "missing csrfToken",
			sessID:    "test-session-id",
			csrfToken: "",
			csrfKey:   "test-csrf-key",
			uid:       "12345",
			wantErr:   true,
		},
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

func TestGetCourses_Mock(t *testing.T) {
	// Read mock data
	mockData, err := os.ReadFile("internal/testdata/courses.json")
	if err != nil {
		t.Fatalf("failed to read mock data: %v", err)
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/iapi2/site-navigation/courses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Check for CSRF headers
		if r.Header.Get("X-CSRF-Token") == "" {
			t.Error("missing X-CSRF-Token header")
		}
		if r.Header.Get("X-CSRF-Key") == "" {
			t.Error("missing X-CSRF-Key header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(mockData)
	}))
	defer server.Close()

	// Create client pointing to mock server
	client, err := NewClient(
		server.URL,
		WithSession("test-sess", "test-token", "test-key", "12345"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Override baseURL to point to mock server
	client.baseURL = server.URL

	ctx := context.Background()
	courses, err := client.GetCourses(ctx)
	if err != nil {
		t.Fatalf("GetCourses() error = %v", err)
	}

	if len(courses) != 2 {
		t.Errorf("expected 2 courses, got %d", len(courses))
	}

	// Verify first course
	if courses[0].Title != "Mathematics 7" {
		t.Errorf("expected course title 'Mathematics 7', got '%s'", courses[0].Title)
	}
	if courses[0].CourseCode != "MATH7" {
		t.Errorf("expected course code 'MATH7', got '%s'", courses[0].CourseCode)
	}
}

func TestSessCookieName(t *testing.T) {
	tests := []struct {
		name   string
		sessID string
		want   string
	}{
		{name: "long id truncated", sessID: "abcdefghijkl", want: "SESSabcdefgh"},
		{name: "exactly 8 chars", sessID: "abcdefgh", want: "SESSabcdefgh"},
		{name: "short id used whole", sessID: "abc", want: "SESSabc"},
		{name: "empty id", sessID: "", want: "SESS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessCookieName(tt.sessID); got != tt.want {
				t.Errorf("sessCookieName(%q) = %q, want %q", tt.sessID, got, tt.want)
			}
		})
	}
}

func TestUpdateSession_ShortID(t *testing.T) {
	c, err := NewClient(
		"school.schoology.com",
		WithSession("longsess", "token", "key", "uid"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// A session ID shorter than 8 chars used to panic on sessID[:8].
	if err := c.UpdateSession("abc", "token2", "key2", "uid2"); err != nil {
		t.Fatalf("UpdateSession with short ID: %v", err)
	}
	if c.session.SessID != "abc" {
		t.Errorf("expected session ID 'abc', got %q", c.session.SessID)
	}
}

func TestCheckResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		wantCode   ErrorCode
	}{
		{
			name:       "success 200",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
			wantCode:   ErrCodeAuth,
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			wantErr:    true,
			wantCode:   ErrCodePermission,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			wantCode:   ErrCodeNotFound,
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			wantErr:    true,
			wantCode:   ErrCodeRateLimit,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
			wantCode:   ErrCodeServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     http.Header{},
			}

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
