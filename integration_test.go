//go:build integration
// +build integration

package schoology

import (
	"context"
	"os"
	"testing"
	"time"
)

// NOTE: These tests hit a real Schoology instance. Run only locally with
// valid session cookies loaded via 1Password or env vars. They touch
// real student data, so do not commit outputs.
//
//   op run --env-file=.env.integration -- go test -tags=integration -v

func getTestClient(t *testing.T) *Client {
	t.Helper()
	host := os.Getenv("SCHOOLOGY_HOST")
	if host == "" {
		t.Skip("SCHOOLOGY_HOST not set - skipping integration tests")
	}
	sessID := os.Getenv("SCHOOLOGY_SESS_ID")
	csrfToken := os.Getenv("SCHOOLOGY_CSRF_TOKEN")
	csrfKey := os.Getenv("SCHOOLOGY_CSRF_KEY")
	uid := os.Getenv("SCHOOLOGY_UID")
	if sessID == "" || csrfToken == "" || csrfKey == "" || uid == "" {
		t.Skip("Session credentials not set - extract session cookies first")
	}
	client, err := NewClient(host, WithSession(sessID, csrfToken, csrfKey, uid))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}

func TestIntegration_ValidateSession(t *testing.T) {
	client := getTestClient(t)
	if err := client.ValidateSession(context.Background()); err != nil {
		t.Fatalf("Session validation failed: %v", err)
	}
	t.Logf("Session valid, expires: %s (%s remaining)",
		client.session.ExpiresAt.Format(time.RFC3339),
		client.SessionTimeRemaining())
}

func TestIntegration_GetCourses(t *testing.T) {
	client := getTestClient(t)
	courses, err := client.GetCourses(context.Background())
	if err != nil {
		t.Fatalf("GetCourses: %v", err)
	}
	t.Logf("Found %d courses", len(courses))
	for i, c := range courses {
		t.Logf("  [%d] nid=%d courseNid=%d %q / %q (%s)",
			i, c.NID, c.CourseNID, c.CourseTitle, c.SectionTitle, c.AdminType)
	}
}

func TestIntegration_GetChildren(t *testing.T) {
	client := getTestClient(t)
	children, err := client.GetChildren(context.Background())
	if err != nil {
		t.Fatalf("GetChildren: %v", err)
	}
	t.Logf("Found %d children", len(children))
	for i, ch := range children {
		t.Logf("  [%d] uid=%d username=%q", i, ch.UID, ch.Username)
	}
}
