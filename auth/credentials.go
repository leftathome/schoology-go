// Package auth drives a browser to produce the session values that
// schoology.NewClient + schoology.WithSession need.
//
// Two entry points:
//
//   - Login(ctx, host) opens a visible Chromium window, lets the user
//     complete whatever login flow their district uses (SSO, MFA, etc.),
//     then extracts the session once the browser lands on a logged-in
//     page. Use this when you don't know (or don't want to care) what
//     the school's auth is.
//
//   - LoginWithPassword(ctx, host, user, pass) scripts Schoology's
//     native Drupal 7 /login form. Works for tenants with password
//     auth; for SSO-backed tenants it will either redirect off-host
//     (returning ErrSSORequired) or simply fail, and the caller should
//     fall back to Login.
//
// Both flows return a *Credentials that the caller plugs into
// schoology.WithSession — or into this package's NewClient helper.
//
// Session reuse: SaveCredentials + LoadCredentials persist the four
// values to a JSON file (0600) so the browser only has to run once
// per ~14-day cookie lifetime.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	schoology "github.com/leftathome/schoology-go"
)

// Credentials is the minimum set of values that schoology.NewClient
// needs to make authenticated requests.
type Credentials struct {
	// Host is the tenant subdomain, e.g. "meanyms.schoology.com".
	Host string `json:"host"`

	// SessID is the value of the Drupal session cookie
	// (SESS + md5(host)). HttpOnly in the browser; the auth package
	// extracts it from the browser's cookie store.
	SessID string `json:"sess_id"`

	// CSRFToken is Drupal.settings.s_common.csrf_token on any
	// authenticated Schoology page.
	CSRFToken string `json:"csrf_token"`

	// CSRFKey is Drupal.settings.s_common.csrf_key on any
	// authenticated page.
	CSRFKey string `json:"csrf_key"`

	// UID is the viewer's numeric user id, as a string. Found in
	// Drupal.settings.s_common.user.uid on any authenticated page.
	UID string `json:"uid"`
}

// Validate returns an error if any required field is empty.
func (c *Credentials) Validate() error {
	switch {
	case c == nil:
		return errors.New("auth: nil credentials")
	case c.Host == "":
		return errors.New("auth: credentials missing Host")
	case c.SessID == "":
		return errors.New("auth: credentials missing SessID")
	case c.CSRFToken == "":
		return errors.New("auth: credentials missing CSRFToken")
	case c.CSRFKey == "":
		return errors.New("auth: credentials missing CSRFKey")
	case c.UID == "":
		return errors.New("auth: credentials missing UID")
	}
	return nil
}

// SaveCredentials writes c to path as JSON with 0600 permissions so
// the file is only readable by the current user.
func SaveCredentials(path string, c *Credentials) error {
	if err := c.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("auth: marshal credentials: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("auth: write credentials: %w", err)
	}
	return nil
}

// LoadCredentials reads and validates credentials from path.
func LoadCredentials(path string) (*Credentials, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("auth: read credentials: %w", err)
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("auth: parse credentials: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// NewClient is a convenience that builds a fully-authenticated
// *schoology.Client from c and any additional schoology options.
// Equivalent to schoology.NewClient(c.Host, schoology.WithSession(...)).
func NewClient(c *Credentials, opts ...schoology.Option) (*schoology.Client, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	all := append([]schoology.Option{
		schoology.WithSession(c.SessID, c.CSRFToken, c.CSRFKey, c.UID),
	}, opts...)
	return schoology.NewClient(c.Host, all...)
}
