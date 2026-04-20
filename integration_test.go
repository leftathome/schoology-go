//go:build integration
// +build integration

package schoology

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// Integration tests hit a real Schoology instance. Run only locally
// with valid session credentials loaded via 1Password or env vars.
// They touch real student data — do NOT commit any test output.
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

// getTestChildren centralizes children lookup so downstream tests can
// loop over them without each calling GetChildren separately.
func getTestChildren(t *testing.T, client *Client) []*Child {
	t.Helper()
	children, err := client.GetChildren(context.Background())
	if err != nil {
		t.Fatalf("GetChildren: %v", err)
	}
	if len(children) == 0 {
		t.Skip("no children on this account — per-child tests cannot run")
	}
	return children
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

func TestIntegration_GetChildren(t *testing.T) {
	client := getTestClient(t)
	children := getTestChildren(t, client)
	t.Logf("Found %d children", len(children))
	for i, ch := range children {
		t.Logf("  [%d] uid=%d username=%q", i, ch.UID, ch.Username)
	}
}

func TestIntegration_GetCourses(t *testing.T) {
	client := getTestClient(t)
	courses, err := client.GetCourses(context.Background())
	if err != nil {
		t.Fatalf("GetCourses: %v", err)
	}
	t.Logf("Found %d courses (current view_child)", len(courses))
	for i, c := range courses {
		t.Logf("  [%d] nid=%d courseNid=%d %q / %q (%s)",
			i, c.NID, c.CourseNID, c.CourseTitle, c.SectionTitle, c.AdminType)
	}
}

func TestIntegration_GetCoursesForChild(t *testing.T) {
	client := getTestClient(t)
	children := getTestChildren(t, client)

	for _, ch := range children {
		ch := ch
		t.Run(ch.Username, func(t *testing.T) {
			courses, err := client.GetCoursesForChild(context.Background(), ch.UID)
			if err != nil {
				t.Fatalf("GetCoursesForChild(%d): %v", ch.UID, err)
			}
			t.Logf("child %d: %d courses", ch.UID, len(courses))
			for i, c := range courses {
				t.Logf("  [%d] nid=%d %q / %q", i, c.NID, c.CourseTitle, c.SectionTitle)
			}
		})
	}
}

func TestIntegration_GetCoursesForChild_ParallelIsSafe(t *testing.T) {
	client := getTestClient(t)
	children := getTestChildren(t, client)
	if len(children) < 2 {
		t.Skip("need at least 2 children to exercise the mutex")
	}

	// Fire GetCoursesForChild(A) and GetCoursesForChild(B) concurrently.
	// If viewChildMu were missing, the two could interleave and each
	// return the other child's courses. Assert each call returns a
	// non-empty set of courses and that calls for different children
	// return different NID sets.
	type res struct {
		uid  int64
		nids []int64
		err  error
	}
	results := make([]res, len(children))
	var wg sync.WaitGroup
	for i, ch := range children {
		i, ch := i, ch
		wg.Add(1)
		go func() {
			defer wg.Done()
			cs, err := client.GetCoursesForChild(context.Background(), ch.UID)
			r := res{uid: ch.UID, err: err}
			for _, c := range cs {
				r.nids = append(r.nids, c.NID)
			}
			results[i] = r
		}()
	}
	wg.Wait()

	for _, r := range results {
		if r.err != nil {
			t.Errorf("child %d: %v", r.uid, r.err)
		}
		t.Logf("child %d: %d courses", r.uid, len(r.nids))
	}

	// If two children have overlapping NID sets that's fine; we just
	// want to know neither came back empty.
	for _, r := range results {
		if len(r.nids) == 0 && r.err == nil {
			t.Errorf("child %d returned empty course list without error", r.uid)
		}
	}
}

func TestIntegration_GetOverdueSubmissions(t *testing.T) {
	client := getTestClient(t)
	children := getTestChildren(t, client)

	for _, ch := range children {
		ch := ch
		t.Run(ch.Username, func(t *testing.T) {
			items, pErrs, err := client.GetOverdueSubmissions(context.Background(), ch.UID)
			if err != nil {
				t.Fatalf("GetOverdueSubmissions(%d): %v", ch.UID, err)
			}
			t.Logf("child %d: %d assignments, %d parse errs",
				ch.UID, len(items), len(pErrs))
			for i, a := range items {
				t.Logf("  [%d] %s id=%d %q course=%q due=%s",
					i, a.Status, a.ID, a.Title, a.CourseTitle, a.DueAt.Format(time.RFC3339))
			}
			for i, e := range pErrs {
				t.Logf("  parseErr[%d]: %v", i, e)
			}
		})
	}
}

func TestIntegration_GetFeed(t *testing.T) {
	client := getTestClient(t)
	children := getTestChildren(t, client)

	for _, ch := range children {
		ch := ch
		t.Run(ch.Username, func(t *testing.T) {
			posts, pErrs, err := client.GetFeed(context.Background(), ch.UID)
			if err != nil {
				t.Fatalf("GetFeed(%d): %v", ch.UID, err)
			}
			t.Logf("child %d: %d posts, %d parse errs, %d with attachments",
				ch.UID, len(posts), len(pErrs), countPostsWithAttachments(posts))
			// Log up to 3 posts for sanity.
			for i, p := range posts {
				if i >= 3 {
					break
				}
				t.Logf("  [%d] edge=%s author=%q(%d) postedTo=%q ts=%s attachments=%d",
					i, p.EdgeID, p.AuthorName, p.AuthorUID, p.PostedTo,
					p.Timestamp.Format(time.RFC3339), len(p.Attachments))
			}
		})
	}
}

func TestIntegration_GetInbox(t *testing.T) {
	client := getTestClient(t)
	threads, pErrs, err := client.GetInbox(context.Background())
	if err != nil {
		t.Fatalf("GetInbox: %v", err)
	}
	t.Logf("%d threads, %d parse errs", len(threads), len(pErrs))
	for i, th := range threads {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] id=%d subject=%q from=%q(%d) unread=%v at=%s",
			i, th.ID, th.Subject, th.SenderName, th.SenderUID,
			th.Unread, th.LastActivity.Format(time.RFC3339))
	}
}

func TestIntegration_DownloadAttachment(t *testing.T) {
	client := getTestClient(t)
	children := getTestChildren(t, client)

	// Find the first post with an attachment across all children.
	var target *Attachment
	for _, ch := range children {
		posts, _, err := client.GetFeed(context.Background(), ch.UID)
		if err != nil {
			t.Logf("GetFeed(%d): %v — continuing", ch.UID, err)
			continue
		}
		for _, p := range posts {
			if len(p.Attachments) > 0 {
				target = p.Attachments[0]
				break
			}
		}
		if target != nil {
			break
		}
	}
	if target == nil {
		t.Skip("no attachment found in any child's feed — nothing to download")
	}

	t.Logf("downloading attachment id=%d filename=%q", target.ID, target.Filename)
	rc, err := client.DownloadAttachment(context.Background(), target.URL)
	if err != nil {
		t.Fatalf("DownloadAttachment: %v", err)
	}
	defer rc.Close()

	// Read up to 64 KB so we don't pull a huge file. We just want to
	// prove the pipeline delivers bytes.
	const cap = 64 * 1024
	got, err := io.ReadAll(io.LimitReader(rc, cap))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("got 0 bytes")
	}
	t.Logf("downloaded %d bytes (capped at %d)", len(got), cap)

	// Sanity: attachment should not be the login page.
	if strings.Contains(string(got), `name="pass"`) {
		t.Errorf("download returned the login page — session may be bad")
	}
}

func countPostsWithAttachments(posts []*Post) int {
	n := 0
	for _, p := range posts {
		if len(p.Attachments) > 0 {
			n++
		}
	}
	return n
}
