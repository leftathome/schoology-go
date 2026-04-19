package schoology

import "net/url"

// mustParseURL parses a URL and panics if it fails
// This is safe to use for hardcoded URLs that we know are valid
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
