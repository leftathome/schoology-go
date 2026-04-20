package schoology

import (
	"errors"
	"testing"
)

func TestNewParseError(t *testing.T) {
	err := NewParseError("parse grading_report: tr.grade-row", "missing score cell")
	if err.Code != ErrCodeParse {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeParse)
	}
	if err.Op != "parse grading_report: tr.grade-row" {
		t.Errorf("Op = %q, want the operation label", err.Op)
	}
	if err.Message != "missing score cell" {
		t.Errorf("Message = %q, want the provided message", err.Message)
	}
}

func TestIsParseError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "parse error", err: NewParseError("op", "msg"), want: true},
		{name: "auth error", err: ErrNotAuthenticated, want: false},
		{
			name: "parse errors slice, non-empty",
			err:  ParseErrors{NewParseError("op", "msg")},
			want: true,
		},
		{
			name: "plain go error",
			err:  errors.New("boom"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsParseError(tt.err); got != tt.want {
				t.Errorf("IsParseError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestParseErrors_Error(t *testing.T) {
	var empty ParseErrors
	if got := empty.Error(); got != "no parse errors" {
		t.Errorf("empty.Error() = %q", got)
	}

	one := ParseErrors{NewParseError("op1", "first")}
	if got := one.Error(); got != "op1: first" {
		t.Errorf("one.Error() = %q", got)
	}

	many := ParseErrors{
		NewParseError("op1", "first"),
		NewParseError("op2", "second"),
		NewParseError("op3", "third"),
	}
	want := "op1: first (and 2 more parse errors)"
	if got := many.Error(); got != want {
		t.Errorf("many.Error() = %q, want %q", got, want)
	}
}

func TestParseErrors_AsError(t *testing.T) {
	var empty ParseErrors
	if got := empty.AsError(); got != nil {
		t.Errorf("empty.AsError() = %v, want nil", got)
	}

	nonEmpty := ParseErrors{NewParseError("op", "msg")}
	if got := nonEmpty.AsError(); got == nil {
		t.Error("nonEmpty.AsError() = nil, want non-nil")
	}
}

func TestParseErrors_Append(t *testing.T) {
	var errs ParseErrors
	errs.Append(NewParseError("op1", "first"))
	errs.Append(NewParseError("op2", "second"))
	if len(errs) != 2 {
		t.Fatalf("len = %d, want 2", len(errs))
	}
	if errs[0].Op != "op1" || errs[1].Op != "op2" {
		t.Errorf("append order wrong: %+v", errs)
	}
}

func TestParseErrors_Unwrap(t *testing.T) {
	errs := ParseErrors{
		NewParseError("op1", "first"),
		NewParseError("op2", "second"),
	}
	got := errs.Unwrap()
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != errs[0] || got[1] != errs[1] {
		t.Error("Unwrap() did not preserve order/identity")
	}

	// Empty slice → empty unwrap result.
	var empty ParseErrors
	if len(empty.Unwrap()) != 0 {
		t.Error("empty.Unwrap() not empty")
	}
}

func TestParseErrors_ErrorsAs(t *testing.T) {
	// A ParseErrors returned as a plain `error` should still be
	// discoverable via errors.As.
	errs := ParseErrors{NewParseError("op", "msg")}
	var err error = errs.AsError()
	var got ParseErrors
	if !errors.As(err, &got) {
		t.Fatal("errors.As failed to unwrap ParseErrors")
	}
	if len(got) != 1 {
		t.Errorf("unwrapped len = %d, want 1", len(got))
	}
}

func TestError_Error(t *testing.T) {
	inner := errors.New("inner boom")
	tests := []struct {
		name string
		e    *Error
		want string
	}{
		{
			name: "op only",
			e:    &Error{Op: "GetX", Message: "boom"},
			want: "GetX: boom",
		},
		{
			name: "op + inner",
			e:    &Error{Op: "GetX", Message: "boom", Err: inner},
			want: "GetX: boom: inner boom",
		},
		{
			name: "no op, with inner",
			e:    &Error{Message: "boom", Err: inner},
			want: "boom: inner boom",
		},
		{
			name: "message only",
			e:    &Error{Message: "just msg"},
			want: "just msg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	e := &Error{Message: "boom", Err: inner}
	if got := e.Unwrap(); got != inner {
		t.Errorf("Unwrap() = %v, want %v", got, inner)
	}
	if !errors.Is(e, inner) {
		t.Error("errors.Is should traverse Error.Err")
	}

	bare := &Error{Message: "boom"}
	if got := bare.Unwrap(); got != nil {
		t.Errorf("Unwrap on bare = %v, want nil", got)
	}
}

func TestError_Is(t *testing.T) {
	a := &Error{Code: ErrCodeAuth, Message: "not authenticated"}
	b := &Error{Code: ErrCodeAuth, Message: "not authenticated"}
	c := &Error{Code: ErrCodeAuth, Message: "different"}
	d := &Error{Code: ErrCodeNotFound, Message: "not authenticated"}

	if !a.Is(b) {
		t.Error("a.Is(b) = false, want true (same code + message)")
	}
	if a.Is(c) {
		t.Error("a.Is(c) = true, want false (different message)")
	}
	if a.Is(d) {
		t.Error("a.Is(d) = true, want false (different code)")
	}
	// non-*Error target
	if a.Is(errors.New("plain")) {
		t.Error("a.Is(plain error) = true, want false")
	}

	// sentinel via errors.Is
	if !errors.Is(ErrNotAuthenticated, ErrNotAuthenticated) {
		t.Error("errors.Is sentinel self-check failed")
	}
}

func TestIsAuthError(t *testing.T) {
	if !IsAuthError(ErrNotAuthenticated) {
		t.Error("IsAuthError(ErrNotAuthenticated) = false")
	}
	if IsAuthError(ErrNotFound) {
		t.Error("IsAuthError(ErrNotFound) = true")
	}
	if IsAuthError(errors.New("plain")) {
		t.Error("IsAuthError(plain) = true")
	}
	if IsAuthError(nil) {
		t.Error("IsAuthError(nil) = true")
	}
}

func TestIsRateLimitError(t *testing.T) {
	if !IsRateLimitError(ErrRateLimited) {
		t.Error("IsRateLimitError(ErrRateLimited) = false")
	}
	if IsRateLimitError(ErrNotAuthenticated) {
		t.Error("IsRateLimitError(auth) = true")
	}
	if IsRateLimitError(errors.New("plain")) {
		t.Error("IsRateLimitError(plain) = true")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "rate limit", err: ErrRateLimited, want: true},
		{name: "network", err: &Error{Code: ErrCodeNetwork, Message: "x"}, want: true},
		{name: "5xx", err: &Error{Code: ErrCodeServer, StatusCode: 503}, want: true},
		{name: "4xx server err", err: &Error{Code: ErrCodeServer, StatusCode: 400}, want: false},
		{name: "auth", err: ErrNotAuthenticated, want: false},
		{name: "plain", err: errors.New("boom"), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
