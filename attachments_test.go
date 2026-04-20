package schoology

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseAttachmentID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want int64
	}{
		{name: "absolute", url: "https://example.schoology.com/attachment/3000001/source/abc.pdf?checkad=1", want: 3000001},
		{name: "path-only", url: "/attachment/3000001/source/abc.pdf?checkad=1", want: 3000001},
		{name: "trailing slash, no source", url: "/attachment/77/", want: 77},
		{name: "no path after id", url: "/attachment/42", want: 42},
		{name: "not an attachment url", url: "/course/1/gradebook", want: 0},
		{name: "empty", url: "", want: 0},
		{name: "non-numeric id", url: "/attachment/abc/source/x.pdf", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAttachmentID(tt.url); got != tt.want {
				t.Errorf("parseAttachmentID(%q) = %d, want %d", tt.url, got, tt.want)
			}
		})
	}
}

func TestDownloadAttachment_ReturnsBody(t *testing.T) {
	const payload = "PDF bytes here (not really a pdf)"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/attachment/3000001/source/abc.pdf" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.RawQuery != "checkad=1" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		if r.Header.Get("X-CSRF-Token") == "" {
			t.Error("missing X-CSRF-Token header")
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	rc, err := client.DownloadAttachment(context.Background(), "/attachment/3000001/source/abc.pdf?checkad=1")
	if err != nil {
		t.Fatalf("DownloadAttachment: %v", err)
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(b) != payload {
		t.Errorf("body = %q, want %q", b, payload)
	}
}

func TestDownloadAttachment_AbsoluteSameHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	url := client.baseURL + "/attachment/1/source/x.pdf?checkad=1"
	rc, err := client.DownloadAttachment(context.Background(), url)
	if err != nil {
		t.Fatalf("DownloadAttachment absolute: %v", err)
	}
	defer rc.Close()
}

func TestDownloadAttachment_AbsoluteWrongHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request should not reach server")
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	_, err := client.DownloadAttachment(context.Background(), "https://other.schoology.com/attachment/1/source/x.pdf")
	if err == nil {
		t.Fatal("expected error for wrong-host URL, got nil")
	}
	e, ok := err.(*Error)
	if !ok {
		t.Fatalf("err = %T %v, want *Error", err, err)
	}
	if e.Code != ErrCodeClient {
		t.Errorf("code = %s, want %s", e.Code, ErrCodeClient)
	}
}

func TestDownloadAttachment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	_, err := client.DownloadAttachment(context.Background(), "/attachment/404/source/x.pdf")
	if !IsNotFoundError(err) {
		t.Errorf("err = %v, want a not-found error", err)
	}
}

func TestDownloadAttachment_EmptyURL(t *testing.T) {
	client, _ := NewClient("school.schoology.com", WithSession("s", "t", "k", "u"))
	_, err := client.DownloadAttachment(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	e, ok := err.(*Error)
	if !ok || e.Code != ErrCodeClient {
		t.Errorf("err = %v, want client-code error", err)
	}
}

func TestAttachmentPath(t *testing.T) {
	c, _ := NewClient("school.schoology.com", WithSession("s", "t", "k", "u"))
	tests := []struct {
		name, in, want string
		wantErr        bool
	}{
		{name: "path only", in: "/attachment/1/x.pdf", want: "/attachment/1/x.pdf"},
		{name: "absolute same host", in: "https://school.schoology.com/attachment/1", want: "/attachment/1"},
		{name: "absolute other host", in: "https://evil.example/attachment/1", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.attachmentPath(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// sanity: the regex does not match a non-attachment url that happens to
// contain a digit sequence.
func TestParseAttachmentID_DoesNotFalsePositive(t *testing.T) {
	cases := []string{
		"/user/3000001/grades",
		"/course/3000001/materials",
		"/some-path-with-3000001-in-it",
		strings.Repeat("x", 100),
	}
	for _, c := range cases {
		if got := parseAttachmentID(c); got != 0 {
			t.Errorf("parseAttachmentID(%q) = %d, want 0", c, got)
		}
	}
}
