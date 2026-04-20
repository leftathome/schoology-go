package auth

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-rod/rod"
)

// sessCookieRe matches a Drupal SESS cookie name. Drupal 7 names the
// session cookie SESS + md5(host) — 32 lowercase hex chars. Matching
// this pattern avoids recomputing the hash from a host string that
// could be slightly off (leading dot, port, etc.).
var sessCookieRe = regexp.MustCompile(`^SESS[a-f0-9]{32}$`)

// drupalSettingsJS reads Drupal.settings.s_common into a small struct.
// Returned fields map one-to-one onto Credentials{CSRFToken, CSRFKey, UID}.
const drupalSettingsJS = `() => {
  const s = (window.Drupal && window.Drupal.settings) || {};
  const c = s.s_common || {};
  const user = c.user || {};
  return {
    csrfToken: c.csrf_token || '',
    csrfKey: c.csrf_key || '',
    uid: (user.uid !== undefined) ? String(user.uid) : '',
  };
}`

// extractCredentials reads the four session values from an
// already-authenticated rod Page. The page must be on a Schoology
// page that has Drupal.settings populated (any logged-in page does).
func extractCredentials(page *rod.Page, host string) (*Credentials, error) {
	// 1. Pull the SESS cookie from the browser's cookie store.
	cookies, err := page.Cookies(nil)
	if err != nil {
		return nil, fmt.Errorf("auth: read cookies: %w", err)
	}
	var sessID string
	for _, c := range cookies {
		if sessCookieRe.MatchString(c.Name) {
			sessID = c.Value
			break
		}
	}
	if sessID == "" {
		return nil, errors.New("auth: no SESS* cookie on authenticated page — login did not complete?")
	}

	// 2. Read Drupal.settings.s_common via page eval.
	result, err := page.Eval(drupalSettingsJS)
	if err != nil {
		return nil, fmt.Errorf("auth: eval Drupal.settings: %w", err)
	}

	csrfToken := result.Value.Get("csrfToken").String()
	csrfKey := result.Value.Get("csrfKey").String()
	uid := result.Value.Get("uid").String()

	creds := &Credentials{
		Host:      host,
		SessID:    sessID,
		CSRFToken: csrfToken,
		CSRFKey:   csrfKey,
		UID:       uid,
	}
	if err := creds.Validate(); err != nil {
		return nil, fmt.Errorf("auth: extracted credentials are incomplete — login may not have finished: %w", err)
	}
	return creds, nil
}
