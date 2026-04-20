package schoology

import (
	"crypto/md5"
	"encoding/hex"
	"net/url"
	"regexp"
	"strconv"
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

// userIDRe extracts the numeric uid from a /user/{uid} href.
var userIDRe = regexp.MustCompile(`/user/(\d+)`)

// parseUserID returns the numeric user id embedded in href, or 0 if
// the href does not match /user/{uid}. Handles absolute and path-only
// forms and strips any query/fragment before matching.
func parseUserID(href string) int64 {
	if u, err := url.Parse(href); err == nil {
		href = u.Path
	}
	m := userIDRe.FindStringSubmatch(href)
	if m == nil {
		return 0
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0
	}
	return id
}
