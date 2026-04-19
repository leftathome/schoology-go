package htmlfetch

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// newResp builds an *http.Response from a content-type and body string.
// Caller does not need to close body; Parse closes it.
func newResp(ct, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{ct}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestParse_HTMLOK(t *testing.T) {
	resp := newResp("text/html; charset=utf-8", `<html><body><div class="x">hi</div></body></html>`)
	doc, err := Parse(resp)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got := doc.Find("div.x").Text(); got != "hi" {
		t.Errorf("parsed text = %q, want %q", got, "hi")
	}
}

func TestParse_PlainHTML(t *testing.T) {
	resp := newResp("text/html", `<html><body><p>ok</p></body></html>`)
	if _, err := Parse(resp); err != nil {
		t.Fatalf("text/html (no charset) should parse: %v", err)
	}
}

func TestParse_NotHTML(t *testing.T) {
	tests := []struct {
		name, ct string
	}{
		{name: "json", ct: "application/json"},
		{name: "empty", ct: ""},
		{name: "plain text", ct: "text/plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := newResp(tt.ct, `{"a":1}`)
			_, err := Parse(resp)
			if !errors.Is(err, ErrNotHTML) {
				t.Errorf("err = %v, want ErrNotHTML", err)
			}
		})
	}
}

func TestParse_LoginRedirectByPassInput(t *testing.T) {
	body := `<html><body>
	  <form><input type="password" name="pass" /></form>
	</body></html>`
	resp := newResp("text/html", body)
	_, err := Parse(resp)
	if !errors.Is(err, ErrLoginRedirect) {
		t.Errorf("err = %v, want ErrLoginRedirect", err)
	}
}

func TestParse_LoginRedirectByFormAction(t *testing.T) {
	body := `<html><body>
	  <form action="/login" method="post">
	    <input type="text" name="mail" />
	  </form>
	</body></html>`
	resp := newResp("text/html", body)
	_, err := Parse(resp)
	if !errors.Is(err, ErrLoginRedirect) {
		t.Errorf("err = %v, want ErrLoginRedirect", err)
	}
}

func TestParse_NonLoginHTMLDoesNotFalsePositive(t *testing.T) {
	// A normal Schoology HTML page with no password field / login form
	// action should not trigger the login-redirect check.
	body := `<html><body>
	  <form action="/node/123/edit"><input type="text" name="title" /></form>
	</body></html>`
	resp := newResp("text/html", body)
	if _, err := Parse(resp); err != nil {
		t.Errorf("unexpected err on normal page: %v", err)
	}
}

func TestParse_ClosesBody(t *testing.T) {
	// Parse must close resp.Body on every return path, or callers leak
	// connections. We wrap a reader to observe closure.
	for _, tc := range []struct {
		name string
		resp *http.Response
	}{
		{
			name: "ok path",
			resp: &http.Response{
				Header: http.Header{"Content-Type": []string{"text/html"}},
				Body:   &trackingCloser{r: strings.NewReader(`<html></html>`)},
			},
		},
		{
			name: "not html",
			resp: &http.Response{
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   &trackingCloser{r: strings.NewReader(`{}`)},
			},
		},
		{
			name: "login redirect",
			resp: &http.Response{
				Header: http.Header{"Content-Type": []string{"text/html"}},
				Body:   &trackingCloser{r: strings.NewReader(`<form><input name="pass"/></form>`)},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.resp.StatusCode = 200
			_, _ = Parse(tc.resp)
			body := tc.resp.Body.(*trackingCloser)
			if !body.closed {
				t.Error("Parse did not close resp.Body")
			}
		})
	}
}

type trackingCloser struct {
	r      io.Reader
	closed bool
}

func (t *trackingCloser) Read(p []byte) (int, error) { return t.r.Read(p) }
func (t *trackingCloser) Close() error               { t.closed = true; return nil }
