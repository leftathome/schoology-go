package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIsHomeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		host string
		want bool
	}{
		{name: "parent home exact", url: "https://x.schoology.com/parent/home", host: "x.schoology.com", want: true},
		{name: "parent home trailing slash", url: "https://x.schoology.com/parent/home/", host: "x.schoology.com", want: true},
		{name: "student home", url: "https://x.schoology.com/home", host: "x.schoology.com", want: true},
		{name: "home with query", url: "https://x.schoology.com/home?x=1", host: "x.schoology.com", want: true},
		{name: "login page", url: "https://x.schoology.com/login", host: "x.schoology.com", want: false},
		{name: "wrong host", url: "https://y.schoology.com/home", host: "x.schoology.com", want: false},
		{name: "off host", url: "https://sso.example.com/auth", host: "x.schoology.com", want: false},
		{name: "home-ish but not", url: "https://x.schoology.com/homesick", host: "x.schoology.com", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHomeURL(tt.url, tt.host); got != tt.want {
				t.Errorf("isHomeURL(%q, %q) = %v, want %v", tt.url, tt.host, got, tt.want)
			}
		})
	}
}

func TestSessCookieRe(t *testing.T) {
	tests := []struct {
		name   string
		cookie string
		want   bool
	}{
		{name: "valid", cookie: "SESS0f255c5e53fbffbe66e5cfcd82ab81b4", want: true},
		{name: "too short", cookie: "SESS0f255c5e53fbffbe66e5cfcd82ab81b", want: false},
		{name: "too long", cookie: "SESS0f255c5e53fbffbe66e5cfcd82ab81b44", want: false},
		{name: "uppercase hex", cookie: "SESS0F255C5E53FBFFBE66E5CFCD82AB81B4", want: false},
		{name: "wrong prefix", cookie: "XSESS0f255c5e53fbffbe66e5cfcd82ab81b4", want: false},
		{name: "different cookie", cookie: "has_js", want: false},
		{name: "empty", cookie: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessCookieRe.MatchString(tt.cookie); got != tt.want {
				t.Errorf("match(%q) = %v, want %v", tt.cookie, got, tt.want)
			}
		})
	}
}

func TestPollUntil_ReturnsWhenCheckTrue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	calls := 0
	err := pollUntil(ctx, 5*time.Millisecond, func() (bool, error) {
		calls++
		return calls >= 3, nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestPollUntil_TimesOut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := pollUntil(ctx, 5*time.Millisecond, func() (bool, error) {
		return false, nil
	})
	if !errors.Is(err, ErrLoginTimeout) {
		t.Errorf("err = %v, want ErrLoginTimeout", err)
	}
}

func TestPollUntil_CheckError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	sentinel := errors.New("boom")
	err := pollUntil(ctx, 5*time.Millisecond, func() (bool, error) {
		return false, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want wrapped sentinel", err)
	}
}

func TestOptions_Defaults(t *testing.T) {
	o := defaults()
	if o.headless {
		t.Error("default headless = true, want false (visible so user can SSO)")
	}
	if o.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want 5m", o.timeout)
	}
	if o.pollInt == 0 {
		t.Error("pollInt = 0")
	}
}

func TestOptions_Overrides(t *testing.T) {
	o := defaults()
	WithHeadless(true)(o)
	WithTimeout(3 * time.Second)(o)
	WithBrowserBinary("/usr/bin/brave")(o)

	if !o.headless {
		t.Error("WithHeadless(true) had no effect")
	}
	if o.timeout != 3*time.Second {
		t.Errorf("timeout = %v, want 3s", o.timeout)
	}
	if o.bin != "/usr/bin/brave" {
		t.Errorf("bin = %q, want /usr/bin/brave", o.bin)
	}
}

func TestLoginWithPassword_EmptyCreds(t *testing.T) {
	_, err := LoginWithPassword(context.Background(), "x.schoology.com", "", "")
	if err == nil {
		t.Error("expected error for empty username/password")
	}
}
