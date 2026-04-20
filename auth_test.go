package schoology

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateSession_Valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"courses":[]}}`))
	}))
	defer server.Close()
	client := withMockedBase(t, server)
	if err := client.ValidateSession(context.Background()); err != nil {
		t.Errorf("ValidateSession: %v", err)
	}
}

func TestValidateSession_Expired(t *testing.T) {
	c, err := NewClient("school.schoology.com")
	if err != nil {
		t.Fatal(err)
	}
	// No session attached → IsAuthenticated() false.
	err = c.ValidateSession(context.Background())
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("err = %v, want ErrSessionExpired", err)
	}
}

func TestValidateSession_ServerReturns401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	client := withMockedBase(t, server)
	err := client.ValidateSession(context.Background())
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("err = %v, want ErrSessionExpired", err)
	}
}

func TestValidateSession_ServerReturns500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := withMockedBase(t, server)
	err := client.ValidateSession(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Not an auth error — should pass through unchanged.
	if errors.Is(err, ErrSessionExpired) {
		t.Errorf("err = %v, should not collapse 5xx into ErrSessionExpired", err)
	}
}

func TestGetSessionInfo(t *testing.T) {
	// Unauthenticated client: nil
	c, _ := NewClient("school.schoology.com")
	if got := c.GetSessionInfo(); got != nil {
		t.Errorf("unauth.GetSessionInfo() = %v, want nil", got)
	}

	// Authenticated client: copy, not alias.
	c2, _ := NewClient("school.schoology.com", WithSession("s", "t", "k", "u"))
	info := c2.GetSessionInfo()
	if info == nil {
		t.Fatal("GetSessionInfo() = nil on authed client")
	}
	if info.SessID != "s" {
		t.Errorf("SessID = %q, want 's'", info.SessID)
	}
	info.SessID = "mutated"
	if c2.session.SessID == "mutated" {
		t.Error("GetSessionInfo returned a reference — mutation leaked")
	}
}

func TestSessionTimeRemaining(t *testing.T) {
	c, _ := NewClient("school.schoology.com")
	if got := c.SessionTimeRemaining(); got != 0 {
		t.Errorf("unauth = %v, want 0", got)
	}

	c2, _ := NewClient("school.schoology.com", WithSession("s", "t", "k", "u"))
	if got := c2.SessionTimeRemaining(); got <= 0 {
		t.Errorf("authed = %v, want positive", got)
	}
	// Should be ~14 days.
	c2.session.ExpiresAt = time.Now().Add(1 * time.Hour)
	got := c2.SessionTimeRemaining()
	if got < 59*time.Minute || got > 61*time.Minute {
		t.Errorf("= %v, want ~1h", got)
	}
}

func TestUpdateSession_RejectsEmpty(t *testing.T) {
	c, _ := NewClient("school.schoology.com", WithSession("s", "t", "k", "u"))
	if err := c.UpdateSession("", "t", "k", "u"); err == nil {
		t.Error("UpdateSession with empty sessID should error")
	}
}
