package schoology

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client is the main Schoology client
type Client struct {
	host       string
	session    *Session
	httpClient *http.Client
	baseURL    string
	timeout    time.Duration
}

// Session holds authentication state
type Session struct {
	SessID    string
	CSRFToken string
	CSRFKey   string
	UID       string
	ExpiresAt time.Time
}

// Option configures the client
type Option func(*Client) error

// NewClient creates a new Schoology client
func NewClient(host string, opts ...Option) (*Client, error) {
	if host == "" {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "host cannot be empty",
		}
	}

	// Remove protocol if provided
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")

	// Create cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "failed to create cookie jar",
			Err:     err,
		}
	}

	client := &Client{
		host:    host,
		baseURL: fmt.Sprintf("https://%s", host),
		timeout: 30 * time.Second,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// WithSession sets session authentication credentials
func WithSession(sessID, csrfToken, csrfKey, uid string) Option {
	return func(c *Client) error {
		if sessID == "" || csrfToken == "" || csrfKey == "" || uid == "" {
			return &Error{
				Code:    ErrCodeClient,
				Message: "all session parameters (sessID, csrfToken, csrfKey, uid) are required",
			}
		}

		c.session = &Session{
			SessID:    sessID,
			CSRFToken: csrfToken,
			CSRFKey:   csrfKey,
			UID:       uid,
			ExpiresAt: time.Now().Add(14 * 24 * time.Hour), // Default 14 day expiration
		}

		// Set cookies in the jar
		baseURL, _ := url.Parse(c.baseURL)
		cookies := []*http.Cookie{
			{
				Name:     sessCookieName(sessID),
				Value:    sessID,
				Domain:   baseURL.Host,
				Path:     "/",
				Expires:  c.session.ExpiresAt,
				Secure:   true,
				HttpOnly: true,
			},
		}
		c.httpClient.Jar.SetCookies(baseURL, cookies)

		return nil
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return &Error{
				Code:    ErrCodeClient,
				Message: "httpClient cannot be nil",
			}
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		if timeout <= 0 {
			return &Error{
				Code:    ErrCodeClient,
				Message: "timeout must be positive",
			}
		}
		c.timeout = timeout
		c.httpClient.Timeout = timeout
		return nil
	}
}

// IsAuthenticated checks if the client has valid authentication
func (c *Client) IsAuthenticated() bool {
	if c.session == nil {
		return false
	}
	return time.Now().Before(c.session.ExpiresAt)
}

// do executes an HTTP request with proper authentication headers
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if !c.IsAuthenticated() {
		return nil, &Error{
			Code:    ErrCodeAuth,
			Message: "not authenticated or session expired",
			Op:      method + " " + path,
		}
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "failed to create request",
			Op:      method + " " + path,
			Err:     err,
		}
	}

	// Set headers
	req.Header.Set("User-Agent", "schoology-go/0.1.0")
	req.Header.Set("Accept", "application/json")
	if c.session != nil {
		req.Header.Set("X-CSRF-Token", c.session.CSRFToken)
		req.Header.Set("X-CSRF-Key", c.session.CSRFKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeNetwork,
			Message: "request failed",
			Op:      method + " " + path,
			Err:     err,
		}
	}

	// Handle common HTTP error status codes
	if err := checkResponse(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}

// checkResponse validates the HTTP response and returns appropriate errors
func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &Error{
			Code:       ErrCodeAuth,
			Message:    "unauthorized - session may have expired",
			StatusCode: resp.StatusCode,
		}
	case http.StatusForbidden:
		return &Error{
			Code:       ErrCodePermission,
			Message:    "forbidden - access denied",
			StatusCode: resp.StatusCode,
		}
	case http.StatusNotFound:
		return &Error{
			Code:       ErrCodeNotFound,
			Message:    "resource not found",
			StatusCode: resp.StatusCode,
		}
	case http.StatusTooManyRequests:
		retryAfter := 60 // default 60 seconds
		if after := resp.Header.Get("Retry-After"); after != "" {
			fmt.Sscanf(after, "%d", &retryAfter)
		}
		return &Error{
			Code:       ErrCodeRateLimit,
			Message:    "rate limited - too many requests",
			StatusCode: resp.StatusCode,
			RetryAfter: retryAfter,
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return &Error{
			Code:       ErrCodeServer,
			Message:    "server error",
			StatusCode: resp.StatusCode,
		}
	default:
		return &Error{
			Code:       ErrCodeServer,
			Message:    fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
			StatusCode: resp.StatusCode,
		}
	}
}

// decodeJSON reads and decodes JSON from the response body
func decodeJSON(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return &Error{
			Code:    ErrCodeServer,
			Message: "failed to decode response",
			Err:     err,
		}
	}

	return nil
}
