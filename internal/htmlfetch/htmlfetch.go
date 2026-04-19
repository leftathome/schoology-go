// Package htmlfetch turns authenticated Schoology HTML responses into
// goquery documents, validating content-type and detecting the login
// page that Schoology serves when the session is bad.
//
// This package is intentionally transport-agnostic: callers in the
// root schoology package own HTTP dispatch (so CSRF headers, session
// cookies, and *schoology.Error wrapping stay in one place) and pass
// the *http.Response in here for parsing.
package htmlfetch

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Sentinel errors returned by Parse. Callers typically wrap these in
// their own error type with an operation label.
var (
	// ErrNotHTML indicates the Content-Type was not text/html.
	ErrNotHTML = errors.New("htmlfetch: response is not text/html")

	// ErrLoginRedirect indicates the response looks like the Schoology
	// login page, which Schoology serves with status 200 when the
	// session cookie is missing or expired.
	ErrLoginRedirect = errors.New("htmlfetch: response is the login page (session expired?)")
)

// Parse reads resp.Body, validates the response is HTML, detects a
// login redirect, and returns a parsed goquery.Document. Parse always
// closes resp.Body, whether it returns successfully or not.
func Parse(resp *http.Response) (*goquery.Document, error) {
	defer resp.Body.Close()

	if !isHTMLContentType(resp.Header.Get("Content-Type")) {
		// Drain remaining body so the underlying connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, ErrNotHTML
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	if isLoginPage(doc) {
		return nil, ErrLoginRedirect
	}

	return doc, nil
}

// isHTMLContentType returns true if ct's media type is text/html.
// Schoology sometimes returns parameters like "text/html; charset=utf-8"
// so we match on the prefix and ignore anything after a semicolon.
func isHTMLContentType(ct string) bool {
	ct = strings.TrimSpace(strings.ToLower(ct))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct == "text/html"
}

// isLoginPage uses conservative selectors to recognize Schoology's
// login page. Schoology is Drupal 7; the login form is a standard
// user-login form whose password input is named "pass" and whose
// form action contains "login". Either signal is enough — being
// generous with detection avoids returning a parse-misery to callers
// when the real problem is an expired session.
func isLoginPage(doc *goquery.Document) bool {
	if doc.Find(`input[name="pass"]`).Length() > 0 {
		return true
	}
	if doc.Find(`form[action*="login"]`).Length() > 0 {
		return true
	}
	return false
}
