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
