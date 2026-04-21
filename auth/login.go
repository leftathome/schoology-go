package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Sentinel errors callers can branch on.
var (
	// ErrSSORequired indicates that a scripted password login landed
	// off the tenant host (likely redirected to a district SSO
	// provider). Caller should fall back to Login().
	ErrSSORequired = errors.New("auth: login redirected off-host (SSO required)")

	// ErrBadCredentials indicates the form resubmitted /login — the
	// server rejected the credentials.
	ErrBadCredentials = errors.New("auth: invalid username or password")

	// ErrLoginTimeout indicates that the caller's context expired
	// before the browser reached a logged-in page.
	ErrLoginTimeout = errors.New("auth: login timed out")
)

// homePathRe matches Schoology's post-login landing paths.
var homePathRe = regexp.MustCompile(`^/(home|parent/home)(/.*|\?.*|$)`)

// options configures a Login / LoginWithPassword call.
type options struct {
	headless bool
	bin      string
	timeout  time.Duration
	pollInt  time.Duration
}

func defaults() *options {
	return &options{
		headless: false, // visible by default for the interactive path
		timeout:  5 * time.Minute,
		pollInt:  500 * time.Millisecond,
	}
}

// Option configures a login call.
type Option func(*options)

// WithHeadless forces the browser to run headless. Defaults to
// visible for Login (so the user can SSO), headless for
// LoginWithPassword.
func WithHeadless(b bool) Option {
	return func(o *options) { o.headless = b }
}

// WithBrowserBinary overrides the Chromium executable rod uses. When
// unset, rod downloads a Chromium build into its cache dir on first
// use.
func WithBrowserBinary(path string) Option {
	return func(o *options) { o.bin = path }
}

// WithTimeout caps the total wall-clock time of the login flow. The
// default is 5 minutes; set shorter for automated runs or longer for
// slow SSO flows.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// newBrowser applies o and returns a connected rod browser. The caller
// must Close it.
func newBrowser(ctx context.Context, o *options) (*rod.Browser, error) {
	l := launcher.New()
	if o.bin != "" {
		l = l.Bin(o.bin)
	}
	l = l.Headless(o.headless)

	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("auth: launch browser: %w", err)
	}
	br := rod.New().ControlURL(url).Context(ctx)
	if err := br.Connect(); err != nil {
		return nil, fmt.Errorf("auth: connect browser: %w", err)
	}
	return br, nil
}

// Login opens a Chromium window pointed at https://{host}/ and blocks
// until the page URL matches Schoology's post-login landing (/home or
// /parent/home), then extracts the session credentials. The browser
// is visible by default so the user can complete whatever login
// flow their district uses (SSO, MFA, the native password form, …).
//
// Cancel ctx (or rely on WithTimeout) to abort a stuck login.
func Login(ctx context.Context, host string, opts ...Option) (*Credentials, error) {
	o := defaults()
	for _, opt := range opts {
		opt(o)
	}

	loginCtx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	browser, err := newBrowser(loginCtx, o)
	if err != nil {
		return nil, err
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{URL: rodURL(host)})
	if err != nil {
		return nil, fmt.Errorf("auth: open page: %w", err)
	}

	if err := waitForLogin(loginCtx, page, host, o.pollInt); err != nil {
		return nil, err
	}
	return extractCredentials(page, host)
}

// LoginWithPassword scripts Schoology's native Drupal 7 /login form
// (s_user_login_form). The form takes `mail`, `pass`, and an optional
// `school_nid`; on a tenant subdomain like "school.schoology.com" the
// school is usually pre-identified by host and school_nid can be left
// empty.
//
// Returns ErrSSORequired if the login flow redirects off the tenant
// host (common on SSO-backed districts; the caller should fall back to
// Login). Returns ErrBadCredentials if the server re-renders /login
// after submission (usually wrong username or password).
//
// Defaults to headless.
func LoginWithPassword(ctx context.Context, host, username, password string, opts ...Option) (*Credentials, error) {
	if username == "" || password == "" {
		return nil, errors.New("auth: username and password are required")
	}

	o := defaults()
	o.headless = true
	for _, opt := range opts {
		opt(o)
	}

	loginCtx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	browser, err := newBrowser(loginCtx, o)
	if err != nil {
		return nil, err
	}
	defer browser.Close()

	loginURL := rodURL(host) + "login"
	page, err := browser.Page(proto.TargetCreateTarget{URL: loginURL})
	if err != nil {
		return nil, fmt.Errorf("auth: open login page: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("auth: wait load: %w", err)
	}

	mail, err := page.Element("#edit-mail")
	if err != nil {
		return nil, fmt.Errorf("auth: find mail field: %w", err)
	}
	if err := mail.Input(username); err != nil {
		return nil, fmt.Errorf("auth: fill mail: %w", err)
	}

	pass, err := page.Element("#edit-pass")
	if err != nil {
		return nil, fmt.Errorf("auth: find pass field: %w", err)
	}
	if err := pass.Input(password); err != nil {
		return nil, fmt.Errorf("auth: fill pass: %w", err)
	}

	submit, err := page.Element("#edit-submit")
	if err != nil {
		return nil, fmt.Errorf("auth: find submit: %w", err)
	}
	if err := submit.Click("left", 1); err != nil {
		return nil, fmt.Errorf("auth: click submit: %w", err)
	}

	// Wait for navigation off /login or for an external redirect.
	if err := waitForPostSubmit(loginCtx, page, host, o.pollInt); err != nil {
		return nil, err
	}
	return extractCredentials(page, host)
}

// rodURL returns the tenant's home URL with a trailing slash.
func rodURL(host string) string { return "https://" + host + "/" }

// waitForLogin polls page.Info().URL until it looks like a logged-in
// Schoology page.
func waitForLogin(ctx context.Context, page *rod.Page, host string, pollInt time.Duration) error {
	return pollUntil(ctx, pollInt, func() (bool, error) {
		info, err := page.Info()
		if err != nil {
			return false, nil
		}
		return isHomeURL(info.URL, host), nil
	})
}

// badCredsSettleCount is the number of consecutive /login observations
// waitForPostSubmit tolerates before declaring ErrBadCredentials. Three
// polls at the default 500ms interval means ~1.5s of grace for a slow
// server before the caller gets a fast failure instead of the full
// timeout.
const badCredsSettleCount = 3

// waitForPostSubmit polls for one of three outcomes: landed on home
// (success), navigated off-host (SSO), or still on /login (bad creds).
// Fast-fails with ErrBadCredentials after the URL stays on /login for
// badCredsSettleCount consecutive polls rather than waiting the full
// context timeout.
func waitForPostSubmit(ctx context.Context, page *rod.Page, host string, pollInt time.Duration) error {
	var (
		offHost     bool
		loginStreak int
	)
	err := pollUntil(ctx, pollInt, func() (bool, error) {
		info, err := page.Info()
		if err != nil {
			return false, nil
		}
		if isHomeURL(info.URL, host) {
			return true, nil
		}
		if !strings.HasPrefix(info.URL, "https://"+host+"/") && !strings.HasPrefix(info.URL, "about:") {
			offHost = true
			return true, nil
		}
		if strings.Contains(info.URL, "/login") {
			loginStreak++
			if loginStreak >= badCredsSettleCount {
				return true, ErrBadCredentials
			}
		} else {
			loginStreak = 0
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	if offHost {
		return ErrSSORequired
	}
	return nil
}

// pollUntil calls check every pollInt until it returns true or ctx
// expires. Context expiry returns ErrLoginTimeout.
func pollUntil(ctx context.Context, pollInt time.Duration, check func() (bool, error)) error {
	t := time.NewTicker(pollInt)
	defer t.Stop()
	for {
		ok, err := check()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ErrLoginTimeout
		case <-t.C:
		}
	}
}

// isHomeURL reports whether u points at /home or /parent/home on host.
func isHomeURL(u, host string) bool {
	prefix := "https://" + host
	if !strings.HasPrefix(u, prefix) {
		return false
	}
	rest := strings.TrimPrefix(u, prefix)
	return homePathRe.MatchString(rest)
}
