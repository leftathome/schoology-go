package schoology

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ValidateSession checks if the current session is valid by making a test request
func (c *Client) ValidateSession(ctx context.Context) error {
	if !c.IsAuthenticated() {
		return ErrSessionExpired
	}

	// Make a simple request to verify the session works
	resp, err := c.do(ctx, http.MethodGet, "/iapi2/site-navigation/courses", nil)
	if err != nil {
		if IsAuthError(err) {
			return ErrSessionExpired
		}
		return err
	}
	defer resp.Body.Close()

	return nil
}

// UpdateSession updates the session credentials
func (c *Client) UpdateSession(sessID, csrfToken, csrfKey, uid string) error {
	if sessID == "" || csrfToken == "" || csrfKey == "" || uid == "" {
		return &Error{
			Code:    ErrCodeClient,
			Message: "all session parameters are required",
		}
	}

	c.session = &Session{
		SessID:    sessID,
		CSRFToken: csrfToken,
		CSRFKey:   csrfKey,
		UID:       uid,
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
	}

	// Update cookies
	baseURL := fmt.Sprintf("https://%s", c.host)
	cookies := []*http.Cookie{
		{
			Name:     fmt.Sprintf("SESS%s", sessID[:8]),
			Value:    sessID,
			Path:     "/",
			Expires:  c.session.ExpiresAt,
			Secure:   true,
			HttpOnly: true,
		},
	}

	c.httpClient.Jar.SetCookies(mustParseURL(baseURL), cookies)

	return nil
}

// GetSessionInfo returns information about the current session
func (c *Client) GetSessionInfo() *Session {
	if c.session == nil {
		return nil
	}

	// Return a copy to prevent external modification
	return &Session{
		SessID:    c.session.SessID,
		CSRFToken: c.session.CSRFToken,
		CSRFKey:   c.session.CSRFKey,
		UID:       c.session.UID,
		ExpiresAt: c.session.ExpiresAt,
	}
}

// SessionTimeRemaining returns the time remaining before session expiration
func (c *Client) SessionTimeRemaining() time.Duration {
	if c.session == nil {
		return 0
	}
	return time.Until(c.session.ExpiresAt)
}
