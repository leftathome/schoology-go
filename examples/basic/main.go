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
	host := os.Getenv("SCHOOLOGY_HOST")
	sessID := os.Getenv("SCHOOLOGY_SESS_ID")
	csrfToken := os.Getenv("SCHOOLOGY_CSRF_TOKEN")
	csrfKey := os.Getenv("SCHOOLOGY_CSRF_KEY")
	uid := os.Getenv("SCHOOLOGY_UID")

	if host == "" {
		log.Fatal("SCHOOLOGY_HOST is required")
	}
	if sessID == "" || csrfToken == "" || csrfKey == "" || uid == "" {
		log.Fatal("SCHOOLOGY_SESS_ID, SCHOOLOGY_CSRF_TOKEN, SCHOOLOGY_CSRF_KEY, SCHOOLOGY_UID are required")
	}

	client, err := schoology.NewClient(host, schoology.WithSession(sessID, csrfToken, csrfKey, uid))
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	if err := client.ValidateSession(ctx); err != nil {
		log.Fatalf("ValidateSession: %v", err)
	}
	fmt.Printf("Session valid (expires %s)\n\n", client.GetSessionInfo().ExpiresAt.Format(time.RFC3339))

	children, err := client.GetChildren(ctx)
	if err != nil {
		log.Fatalf("GetChildren: %v", err)
	}
	fmt.Printf("Children (%d):\n", len(children))
	for _, ch := range children {
		fmt.Printf("  uid=%d username=%q\n", ch.UID, ch.Username)
	}
	fmt.Println()

	courses, err := client.GetCourses(ctx)
	if err != nil {
		log.Fatalf("GetCourses: %v", err)
	}
	fmt.Printf("Courses for currently-viewed enrollment (%d):\n", len(courses))
	for _, c := range courses {
		fmt.Printf("  nid=%d %q / %q @ %s (%s)\n",
			c.NID, c.CourseTitle, c.SectionTitle, c.BuildingTitle, c.AdminType)
	}
}
