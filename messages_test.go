package schoology

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	inboxFixturePath       = "internal/testdata/html/messages_inbox_empty.html"
	inboxFilledFixturePath = "internal/testdata/html/messages_inbox_filled.html"
)

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

func TestParseInbox_FilledFixture(t *testing.T) {
	// Hand-crafted fixture models the structure we inferred from the
	// empty-state capture + Drupal 7 messages-table conventions. If
	// Schoology changes the DOM shape on the real site the
	// integration test will break first, but this test gives us an
	// offline regression catch for the parser logic itself.
	b, err := os.ReadFile(inboxFilledFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	threads, perrs := parseInbox(string(b))

	// Three well-formed rows (#8800000001 unread, #8800000002 read,
	// #8800000003 with empty subject). The fourth malformed row
	// (plain text only, no subject link and only 2 cells) parses
	// into a thread with "Malformed row with no link" as subject.
	// The empty-subject row (#8800000003) becomes a parseErr.
	if got := len(perrs); got != 1 {
		t.Errorf("parse errors = %d (%v), want 1", got, perrs)
	}
	if len(threads) < 2 {
		t.Fatalf("threads = %d, want >=2", len(threads))
	}

	// First thread (unread, with subject link).
	first := threads[0]
	if first.ID != 8800000001 {
		t.Errorf("first.ID = %d, want 8800000001", first.ID)
	}
	if first.Subject != "Field trip form reminder" {
		t.Errorf("first.Subject = %q", first.Subject)
	}
	if !first.Unread {
		t.Error("first.Unread = false, want true")
	}
	if first.SenderUID != 1000000010 {
		t.Errorf("first.SenderUID = %d, want 1000000010", first.SenderUID)
	}
	if first.LastActivity.IsZero() {
		t.Error("first.LastActivity not parsed")
	}

	// Second thread (read).
	second := threads[1]
	if second.ID != 8800000002 {
		t.Errorf("second.ID = %d, want 8800000002", second.ID)
	}
	if second.Unread {
		t.Error("second.Unread = true, want false")
	}
}

func TestParseInbox_MalformedOnly(t *testing.T) {
	// A table with rows that have no subject at all accumulates into
	// ParseErrors without crashing.
	html := `<table><tbody>
        <tr><td><a href="/messages/thread/1"></a></td><td>X</td><td>Tue Mar 31, 2026 at 9:41 am</td></tr>
    </tbody></table>`
	threads, perrs := parseInbox(html)
	if len(threads) != 0 {
		t.Errorf("threads = %v, want 0", threads)
	}
	if len(perrs) != 1 {
		t.Errorf("perrs = %v, want 1", perrs)
	}
}

func TestParseInboxRow_MessageCountBadge(t *testing.T) {
	html := `<table><tbody>
        <tr>
          <td><a href="/messages/thread/9">Sub</a> <span class="badge">7</span></td>
          <td><a href="/user/42">X</a></td>
          <td>Tue Mar 31, 2026 at 9:41 am</td>
        </tr>
    </tbody></table>`
	threads, perrs := parseInbox(html)
	if len(perrs) != 0 {
		t.Fatalf("perrs = %v", perrs)
	}
	if len(threads) != 1 {
		t.Fatalf("threads = %d, want 1", len(threads))
	}
	if threads[0].MessageCount != 7 {
		t.Errorf("MessageCount = %d, want 7", threads[0].MessageCount)
	}
}
