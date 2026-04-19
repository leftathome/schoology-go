package schoology

import (
	"fmt"
	"net/url"
)

// mustParseURL parses a URL and panics if it fails
// This is safe to use for hardcoded URLs that we know are valid
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}

// sessCookieName derives the session cookie name from the session ID.
// Real Schoology session cookies follow the Drupal SESS<md5(hostname)> pattern;
// see bd issue schoology-go-9xy for verifying and replacing this heuristic.
// Until then, fall back to the first 8 chars of the session ID (or the whole
// ID if shorter) so short test IDs don't panic.
func sessCookieName(sessID string) string {
	suffix := sessID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	return fmt.Sprintf("SESS%s", suffix)
}
