package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func validCreds() *Credentials {
	return &Credentials{
		Host:      "example.schoology.com",
		SessID:    "sess-value",
		CSRFToken: "token",
		CSRFKey:   "key",
		UID:       "12345",
	}
}

func TestCredentials_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Credentials)
		wantErr string // substring match, "" = expect no error
	}{
		{name: "valid", mutate: func(c *Credentials) {}, wantErr: ""},
		{name: "missing host", mutate: func(c *Credentials) { c.Host = "" }, wantErr: "Host"},
		{name: "missing sess", mutate: func(c *Credentials) { c.SessID = "" }, wantErr: "SessID"},
		{name: "missing token", mutate: func(c *Credentials) { c.CSRFToken = "" }, wantErr: "CSRFToken"},
		{name: "missing key", mutate: func(c *Credentials) { c.CSRFKey = "" }, wantErr: "CSRFKey"},
		{name: "missing uid", mutate: func(c *Credentials) { c.UID = "" }, wantErr: "UID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validCreds()
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error mentioning %q, got nil", tt.wantErr)
			}
			if !containsAll(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCredentials_NilValidate(t *testing.T) {
	var c *Credentials
	if err := c.Validate(); err == nil {
		t.Fatal("nil.Validate() should error, got nil")
	}
}

func TestSaveLoadCredentials_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	want := validCreds()

	if err := SaveCredentials(path, want); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file perms = %v, want 0600", mode)
	}

	got, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if *got != *want {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestSaveCredentials_RejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	if err := SaveCredentials(path, &Credentials{Host: "h"}); err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestLoadCredentials_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCredentials(path); err == nil {
		t.Error("expected parse error, got nil")
	}
}

func TestLoadCredentials_MissingFile(t *testing.T) {
	_, err := LoadCredentials(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadCredentials_IncompleteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	if err := os.WriteFile(path, []byte(`{"host":"x","sess_id":"y"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadCredentials(path)
	if err == nil {
		t.Error("expected validation error on incomplete file")
	}
}

func TestNewClient_ValidatesBeforeBuilding(t *testing.T) {
	_, err := NewClient(&Credentials{Host: "x"})
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestNewClient_Valid(t *testing.T) {
	client, err := NewClient(validCreds())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if !client.IsAuthenticated() {
		t.Error("new client reports not authenticated")
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !stringContains(s, p) {
			return false
		}
	}
	return true
}

func stringContains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
