package schoology

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const inboxFixturePath = "internal/testdata/html/messages_inbox_empty.html"

func TestParseInbox_EmptyFixture(t *testing.T) {
	b, err := os.ReadFile(inboxFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	threads, perrs := parseInbox(string(b))
	if perrs != nil {
		t.Errorf("parseInbox returned ParseErrors = %v, want nil", perrs)
	}
	if len(threads) != 0 {
		t.Errorf("parseInbox returned %d threads, want 0", len(threads))
	}
}

func TestGetInbox_EmptyFixtureServer(t *testing.T) {
	b, err := os.ReadFile(inboxFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages/inbox" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("X-CSRF-Token") == "" {
			t.Error("missing X-CSRF-Token header")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	threads, perrs, err := client.GetInbox(context.Background())
	if err != nil {
		t.Fatalf("GetInbox: %v", err)
	}
	if perrs != nil {
		t.Errorf("GetInbox returned ParseErrors = %v, want nil", perrs)
	}
	if len(threads) != 0 {
		t.Errorf("GetInbox returned %d threads, want 0", len(threads))
	}
}

func TestGetInbox_LoginPageReturnsAuthError(t *testing.T) {
	const loginHTML = `<!DOCTYPE html><html><body><form action="/login"><input name="pass"/></form></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(loginHTML))
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	threads, perrs, err := client.GetInbox(context.Background())
	if err == nil {
		t.Fatal("expected error for login-page response, got nil")
	}
	if threads != nil {
		t.Errorf("threads = %v, want nil", threads)
	}
	if perrs != nil {
		t.Errorf("perrs = %v, want nil", perrs)
	}

	e, ok := err.(*Error)
	if !ok {
		t.Fatalf("err = %T (%v), want *Error", err, err)
	}
	if e.Code != ErrCodeAuth {
		t.Errorf("err code = %s, want %s", e.Code, ErrCodeAuth)
	}
	if e.Op != "GetInbox" {
		t.Errorf("err op = %q, want %q", e.Op, "GetInbox")
	}
}
