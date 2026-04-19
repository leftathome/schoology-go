package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/leftathome/schoology-go"
)

func main() {
	// Get configuration from environment variables
	host := os.Getenv("SCHOOLOGY_HOST")
	sessID := os.Getenv("SCHOOLOGY_SESS_ID")
	csrfToken := os.Getenv("SCHOOLOGY_CSRF_TOKEN")
	csrfKey := os.Getenv("SCHOOLOGY_CSRF_KEY")
	uid := os.Getenv("SCHOOLOGY_UID")

	if host == "" {
		log.Fatal("SCHOOLOGY_HOST environment variable is required")
	}
	if sessID == "" || csrfToken == "" || csrfKey == "" || uid == "" {
		log.Fatal("Session credentials required: SCHOOLOGY_SESS_ID, SCHOOLOGY_CSRF_TOKEN, SCHOOLOGY_CSRF_KEY, SCHOOLOGY_UID")
	}

	// Create client
	client, err := schoology.NewClient(
		host,
		schoology.WithSession(sessID, csrfToken, csrfKey, uid),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Validate session
	fmt.Println("Validating session...")
	if err := client.ValidateSession(ctx); err != nil {
		log.Fatalf("Session validation failed: %v", err)
	}
	fmt.Printf("✓ Session is valid (expires: %s)\n\n", client.GetSessionInfo().ExpiresAt.Format(time.RFC3339))

	// Get courses
	fmt.Println("Fetching courses...")
	courses, err := client.GetCourses(ctx)
	if err != nil {
		log.Fatalf("Failed to get courses: %v", err)
	}

	fmt.Printf("Found %d courses:\n\n", len(courses))
	for i, course := range courses {
		fmt.Printf("%d. %s (%s)\n", i+1, course.Title, course.CourseCode)
		fmt.Printf("   Teacher: %s\n", course.Teacher)
		fmt.Printf("   Period: %s | Room: %s\n", course.Period, course.Room)
		fmt.Printf("   Active: %v\n", course.Active)

		if !course.Active {
			fmt.Println()
			continue
		}

		// Get assignments for this course
		assignments, err := client.GetAssignments(ctx, course.SectionID)
		if err != nil {
			fmt.Printf("   ⚠ Failed to get assignments: %v\n\n", err)
			continue
		}

		// Count upcoming assignments
		now := time.Now()
		upcoming := 0
		for _, a := range assignments {
			if a.DueDate.After(now) && a.Status != schoology.StatusGraded {
				upcoming++
			}
		}

		fmt.Printf("   Assignments: %d total, %d upcoming\n", len(assignments), upcoming)

		// Get grades
		grades, err := client.GetCourseGrades(ctx, course.SectionID)
		if err != nil {
			fmt.Printf("   ⚠ Failed to get grades: %v\n\n", err)
			continue
		}

		if len(grades) > 0 {
			for _, grade := range grades {
				if grade.LetterGrade != "" {
					fmt.Printf("   Current Grade: %s", grade.LetterGrade)
					if grade.Percentage != nil {
						fmt.Printf(" (%.1f%%)", *grade.Percentage)
					}
					fmt.Println()
					break
				}
			}
		}

		fmt.Println()
	}

	// Get all upcoming assignments
	fmt.Println("\nFetching all upcoming assignments...")
	upcoming, err := client.GetUpcomingAssignments(ctx)
	if err != nil {
		log.Fatalf("Failed to get upcoming assignments: %v", err)
	}

	if len(upcoming) == 0 {
		fmt.Println("No upcoming assignments!")
	} else {
		fmt.Printf("\n%d Upcoming Assignments:\n\n", len(upcoming))
		for i, assignment := range upcoming {
			if i >= 10 {
				fmt.Printf("... and %d more\n", len(upcoming)-10)
				break
			}
			daysUntil := time.Until(assignment.DueDate).Hours() / 24
			fmt.Printf("• %s\n", assignment.Title)
			fmt.Printf("  Course: %s\n", assignment.CourseID)
			fmt.Printf("  Due: %s (in %.0f days)\n", assignment.DueDate.Format("Mon Jan 2"), daysUntil)
			if assignment.MaxPoints > 0 {
				fmt.Printf("  Points: %.0f\n", assignment.MaxPoints)
			}
			fmt.Println()
		}
	}
}
