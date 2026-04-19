package schoology

import (
	"crypto/md5"
	"encoding/hex"
	"net/url"
)

// mustParseURL parses a URL and panics if it fails.
// Safe for hardcoded URLs we know are valid.
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}

// sessCookieName returns the Schoology session cookie name for a host.
// Schoology is built on Drupal 7, which names the session cookie
// "SESS" + md5(cookie_domain). The md5 is computed over the bare
// hostname (no leading dot, no scheme, no trailing slash) — this was
// verified against a live Schoology session.
func sessCookieName(host string) string {
	sum := md5.Sum([]byte(host))
	return "SESS" + hex.EncodeToString(sum[:])
}
