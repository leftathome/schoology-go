package schoology

import (
	"errors"
	"fmt"
)

// Sentinel errors for common cases
var (
	// ErrNotAuthenticated indicates the client is not authenticated
	ErrNotAuthenticated = &Error{
		Code:    ErrCodeAuth,
		Message: "not authenticated",
	}

	// ErrSessionExpired indicates the session has expired
	ErrSessionExpired = &Error{
		Code:    ErrCodeAuth,
		Message: "session expired",
	}

	// ErrInvalidSession indicates invalid session credentials were provided
	ErrInvalidSession = &Error{
		Code:    ErrCodeAuth,
		Message: "invalid session credentials",
	}

	// ErrNotFound indicates the requested resource was not found
	ErrNotFound = &Error{
		Code:    ErrCodeNotFound,
		Message: "resource not found",
	}

	// ErrRateLimited indicates the client has been rate limited
	ErrRateLimited = &Error{
		Code:    ErrCodeRateLimit,
		Message: "rate limited - too many requests",
	}

	// ErrInvalidResponse indicates an unexpected response from the server
	ErrInvalidResponse = &Error{
		Code:    ErrCodeServer,
		Message: "invalid response from server",
	}

	// ErrPermissionDenied indicates access to the resource is not permitted
	ErrPermissionDenied = &Error{
		Code:    ErrCodePermission,
		Message: "permission denied",
	}
)

// ErrorCode represents the category of error
type ErrorCode string

const (
	// ErrCodeAuth indicates an authentication or authorization error
	ErrCodeAuth ErrorCode = "auth"

	// ErrCodeNotFound indicates a resource was not found
	ErrCodeNotFound ErrorCode = "not_found"

	// ErrCodeRateLimit indicates rate limiting
	ErrCodeRateLimit ErrorCode = "rate_limit"

	// ErrCodeServer indicates a server or API error
	ErrCodeServer ErrorCode = "server"

	// ErrCodeClient indicates a client-side error
	ErrCodeClient ErrorCode = "client"

	// ErrCodePermission indicates a permission error
	ErrCodePermission ErrorCode = "permission"

	// ErrCodeNetwork indicates a network connectivity error
	ErrCodeNetwork ErrorCode = "network"

	// ErrCodeParse indicates an HTML or response parse failure
	ErrCodeParse ErrorCode = "parse"
)

// Error represents an error from the Schoology client
type Error struct {
	// Code categorizes the error
	Code ErrorCode

	// Message is a human-readable error message
	Message string

	// Op is the operation being performed when the error occurred
	Op string

	// Err is the underlying error, if any
	Err error

	// StatusCode is the HTTP status code, if applicable
	StatusCode int

	// RetryAfter indicates seconds to wait before retrying, if applicable
	RetryAfter int
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Op != "" {
		if e.Err != nil {
			return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
		}
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements error equality checking
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code && e.Message == t.Message
}

// IsAuthError returns true if the error is an authentication error
func IsAuthError(err error) bool {
	e, ok := err.(*Error)
	return ok && e.Code == ErrCodeAuth
}

// IsNotFoundError returns true if the error is a not found error
func IsNotFoundError(err error) bool {
	e, ok := err.(*Error)
	return ok && e.Code == ErrCodeNotFound
}

// IsRateLimitError returns true if the error is a rate limit error
func IsRateLimitError(err error) bool {
	e, ok := err.(*Error)
	return ok && e.Code == ErrCodeRateLimit
}

// IsParseError returns true if the error (or any wrapped error) is a parse error.
func IsParseError(err error) bool {
	if e, ok := err.(*Error); ok && e.Code == ErrCodeParse {
		return true
	}
	var p ParseErrors
	return errors.As(err, &p)
}

// NewParseError constructs an *Error with ErrCodeParse. Op should identify
// the resource and selector path (e.g. "parse grading_report: tr.grade-row").
func NewParseError(op, message string) *Error {
	return &Error{
		Code:    ErrCodeParse,
		Op:      op,
		Message: message,
	}
}

// ParseErrors is a collection of per-item parse failures from a single
// HTML page. Parsers return their best-effort result alongside a
// ParseErrors value so a single malformed row does not fail the whole
// page — callers see both the rows that parsed and the ones that did not.
type ParseErrors []*Error

// Error implements the error interface.
func (p ParseErrors) Error() string {
	switch len(p) {
	case 0:
		return "no parse errors"
	case 1:
		return p[0].Error()
	default:
		return fmt.Sprintf("%s (and %d more parse errors)", p[0].Error(), len(p)-1)
	}
}

// Unwrap exposes the individual errors so errors.Is / errors.As can
// traverse the collection.
func (p ParseErrors) Unwrap() []error {
	out := make([]error, len(p))
	for i, e := range p {
		out[i] = e
	}
	return out
}

// Append adds err to the collection. err must be non-nil.
func (p *ParseErrors) Append(err *Error) {
	*p = append(*p, err)
}

// AsError returns nil if p is empty, otherwise p itself. Convenience for
// parsers ending with `return result, errs.AsError()`.
func (p ParseErrors) AsError() error {
	if len(p) == 0 {
		return nil
	}
	return p
}

// IsRetryable returns true if the error is likely transient and the operation can be retried
func IsRetryable(err error) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	switch e.Code {
	case ErrCodeRateLimit, ErrCodeNetwork:
		return true
	case ErrCodeServer:
		// 5xx errors are generally retryable
		return e.StatusCode >= 500 && e.StatusCode < 600
	default:
		return false
	}
}
