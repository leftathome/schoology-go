//go:build integration
// +build integration

package schoology

import (
	"context"
	"os"
	"testing"
	"time"
)

// NOTE: These tests use real Schoology data and should ONLY be run locally
// with proper 1Password authentication.
//
// To run these tests:
// 1. Copy .env.example to .env.integration
// 2. Fill in SCHOOLOGY_HOST and 1Password secret references
// 3. Run: op run --env-file=.env.integration -- go test -tags=integration -v
//
// IMPORTANT: These tests access real student data. Handle with care.
// Do NOT commit .env.integration to version control.

func getTestClient(t *testing.T) *Client {
	t.Helper()

	host := os.Getenv("SCHOOLOGY_HOST")
	if host == "" {
		t.Skip("SCHOOLOGY_HOST not set - skipping integration tests")
	}

	// Check if we have session credentials
	sessID := os.Getenv("SCHOOLOGY_SESS_ID")
	csrfToken := os.Getenv("SCHOOLOGY_CSRF_TOKEN")
	csrfKey := os.Getenv("SCHOOLOGY_CSRF_KEY")
	uid := os.Getenv("SCHOOLOGY_UID")

	if sessID == "" || csrfToken == "" || csrfKey == "" || uid == "" {
		t.Skip("Session credentials not set - you need to extract session cookies first")
	}

	client, err := NewClient(
		host,
		WithSession(sessID, csrfToken, csrfKey, uid),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return client
}

func TestIntegration_ValidateSession(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	err := client.ValidateSession(ctx)
	if err != nil {
		t.Fatalf("Session validation failed: %v", err)
	}

	t.Logf("Session is valid")
	t.Logf("Session expires: %s", client.session.ExpiresAt.Format(time.RFC3339))
	t.Logf("Time remaining: %s", client.SessionTimeRemaining())
}

func TestIntegration_GetCourses(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	courses, err := client.GetCourses(ctx)
	if err != nil {
		t.Fatalf("Failed to get courses: %v", err)
	}

	if len(courses) == 0 {
		t.Log("No courses found - this might be expected for a parent account")
		return
	}

	t.Logf("Found %d courses", len(courses))
	for i, course := range courses {
		t.Logf("Course %d:", i+1)
		t.Logf("  ID: %s", course.ID)
		t.Logf("  Title: %s", course.Title)
		t.Logf("  Code: %s", course.CourseCode)
		t.Logf("  Section: %s", course.SectionID)
		t.Logf("  Teacher: %s", course.Teacher)
		t.Logf("  Active: %v", course.Active)
	}
}

func TestIntegration_GetAssignments(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	// First get courses
	courses, err := client.GetCourses(ctx)
	if err != nil {
		t.Fatalf("Failed to get courses: %v", err)
	}

	if len(courses) == 0 {
		t.Skip("No courses found - cannot test assignments")
	}

	// Get assignments for the first active course
	var testCourse *Course
	for _, course := range courses {
		if course.Active {
			testCourse = course
			break
		}
	}

	if testCourse == nil {
		t.Skip("No active courses found")
	}

	t.Logf("Testing assignments for course: %s", testCourse.Title)

	assignments, err := client.GetAssignments(ctx, testCourse.SectionID)
	if err != nil {
		t.Fatalf("Failed to get assignments: %v", err)
	}

	t.Logf("Found %d assignments", len(assignments))
	for i, assignment := range assignments {
		if i >= 5 {
			t.Logf("... and %d more assignments", len(assignments)-5)
			break
		}
		t.Logf("Assignment %d:", i+1)
		t.Logf("  ID: %s", assignment.ID)
		t.Logf("  Title: %s", assignment.Title)
		t.Logf("  Due: %s", assignment.DueDate.Format("2006-01-02"))
		t.Logf("  Status: %s", assignment.Status)
		t.Logf("  Max Points: %.1f", assignment.MaxPoints)
	}
}

func TestIntegration_GetUpcomingAssignments(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	assignments, err := client.GetUpcomingAssignments(ctx)
	if err != nil {
		t.Fatalf("Failed to get upcoming assignments: %v", err)
	}

	t.Logf("Found %d upcoming assignments", len(assignments))
	for i, assignment := range assignments {
		if i >= 10 {
			t.Logf("... and %d more upcoming assignments", len(assignments)-10)
			break
		}
		daysUntilDue := time.Until(assignment.DueDate).Hours() / 24
		t.Logf("Assignment %d:", i+1)
		t.Logf("  Title: %s", assignment.Title)
		t.Logf("  Course: %s", assignment.CourseID)
		t.Logf("  Due: %s (%.0f days)", assignment.DueDate.Format("2006-01-02"), daysUntilDue)
		t.Logf("  Status: %s", assignment.Status)
	}
}

func TestIntegration_GetGrades(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	grades, err := client.GetGrades(ctx)
	if err != nil {
		t.Fatalf("Failed to get grades: %v", err)
	}

	t.Logf("Found %d grades", len(grades))
	for i, grade := range grades {
		if i >= 10 {
			t.Logf("... and %d more grades", len(grades)-10)
			break
		}
		t.Logf("Grade %d:", i+1)
		t.Logf("  Course: %s", grade.CourseName)
		if grade.AssignmentName != "" {
			t.Logf("  Assignment: %s", grade.AssignmentName)
		}
		if grade.Score != nil {
			t.Logf("  Score: %.1f/%.1f", *grade.Score, grade.MaxScore)
		}
		if grade.Percentage != nil {
			t.Logf("  Percentage: %.1f%%", *grade.Percentage)
		}
		if grade.LetterGrade != "" {
			t.Logf("  Letter Grade: %s", grade.LetterGrade)
		}
	}
}

func TestIntegration_SessionExpiration(t *testing.T) {
	client := getTestClient(t)

	if !client.IsAuthenticated() {
		t.Fatal("Client should be authenticated")
	}

	sessionInfo := client.GetSessionInfo()
	if sessionInfo == nil {
		t.Fatal("Session info should not be nil")
	}

	t.Logf("Session Info:")
	t.Logf("  UID: %s", sessionInfo.UID)
	t.Logf("  Expires: %s", sessionInfo.ExpiresAt.Format(time.RFC3339))
	t.Logf("  Time Remaining: %s", client.SessionTimeRemaining())
}
