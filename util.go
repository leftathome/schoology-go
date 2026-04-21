package schoology

import (
	"crypto/md5"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
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

// parsePathID extracts the numeric id that follows the given
// URL-path prefix (e.g. "/user/", "/assignment/", "/attachment/").
// Handles absolute and path-only URLs and ignores any query or
// fragment. Returns 0 on no match.
func parsePathID(rawURL, prefix string) int64 {
	if u, err := url.Parse(rawURL); err == nil {
		rawURL = u.Path
	}
	i := strings.Index(rawURL, prefix)
	if i < 0 {
		return 0
	}
	s := rawURL[i+len(prefix):]
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	id, err := strconv.ParseInt(s[:end], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// parseUserID returns the numeric user id in a /user/{uid} URL, or 0
// if the URL does not match.
func parseUserID(href string) int64 { return parsePathID(href, "/user/") }
