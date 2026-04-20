package schoology

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func TestGetCoursesForChild_SwitchesThenLists(t *testing.T) {
	// The test server tracks the most-recent switch and returns a
	// different course list depending on which child is in view.
	const (
		childA int64 = 1000000001
		childB int64 = 1000000002
	)
	var current atomic.Int64
	current.Store(childB) // start on B

	coursesFor := func(uid int64) []byte {
		env := map[string]any{
			"data": map[string]any{
				"courses": []map[string]any{
					{"nid": uid*10 + 1, "courseTitle": "Course for child"},
				},
			},
		}
		b, _ := json.Marshal(env)
		return b
	}

	var switchCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/parent/switch_child/1000000001":
			switchCalls.Add(1)
			current.Store(childA)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<html><body>ok</body></html>"))
		case r.URL.Path == "/parent/switch_child/1000000002":
			switchCalls.Add(1)
			current.Store(childB)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<html><body>ok</body></html>"))
		case r.URL.Path == "/iapi2/site-navigation/courses":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(coursesFor(current.Load()))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	ctx := context.Background()

	got, err := client.GetCoursesForChild(ctx, childA)
	if err != nil {
		t.Fatalf("GetCoursesForChild(A): %v", err)
	}
	if len(got) != 1 || got[0].NID != childA*10+1 {
		t.Errorf("A got %+v, want nid=%d", got, childA*10+1)
	}

	got, err = client.GetCoursesForChild(ctx, childB)
	if err != nil {
		t.Fatalf("GetCoursesForChild(B): %v", err)
	}
	if len(got) != 1 || got[0].NID != childB*10+1 {
		t.Errorf("B got %+v, want nid=%d", got, childB*10+1)
	}

	if got := switchCalls.Load(); got != 2 {
		t.Errorf("switch_child calls = %d, want 2", got)
	}
}

func TestGetCoursesForChild_SerializesConcurrentCalls(t *testing.T) {
	// Two concurrent GetCoursesForChild calls for different children
	// must not interleave viewAs/GetCourses pairs. We enforce that by
	// recording request order and asserting that every switch is
	// immediately followed by a courses fetch for the *same* child.
	const (
		childA int64 = 1000000001
		childB int64 = 1000000002
	)

	var (
		mu      sync.Mutex
		current int64 = childA
		events  []string // sequence of "switch:A", "courses:A" etc.
	)
	tag := func(uid int64) string {
		if uid == childA {
			return "A"
		}
		return "B"
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/parent/switch_child/1000000001":
			mu.Lock()
			current = childA
			events = append(events, "switch:A")
			mu.Unlock()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<html></html>"))
		case "/parent/switch_child/1000000002":
			mu.Lock()
			current = childB
			events = append(events, "switch:B")
			mu.Unlock()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<html></html>"))
		case "/iapi2/site-navigation/courses":
			mu.Lock()
			events = append(events, "courses:"+tag(current))
			uid := current
			mu.Unlock()
			env := map[string]any{"data": map[string]any{"courses": []map[string]any{
				{"nid": uid*10 + 1, "courseTitle": "c"},
			}}}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(env)
		default:
			t.Errorf("unexpected: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := withMockedBase(t, server)

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make([]error, 2)
	go func() {
		defer wg.Done()
		_, errs[0] = client.GetCoursesForChild(context.Background(), childA)
	}()
	go func() {
		defer wg.Done()
		_, errs[1] = client.GetCoursesForChild(context.Background(), childB)
	}()
	wg.Wait()

	for i, e := range errs {
		if e != nil {
			t.Fatalf("goroutine %d: %v", i, e)
		}
	}

	// events should be 4 entries alternating switch/courses, with
	// every courses matching the preceding switch's tag.
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 4 {
		t.Fatalf("events = %v, want 4 entries", events)
	}
	for i := 0; i < len(events); i += 2 {
		sw := events[i]
		co := events[i+1]
		if sw[:7] != "switch:" || co[:8] != "courses:" {
			t.Fatalf("events not in switch/courses order: %v", events)
		}
		if sw[7:] != co[8:] {
			t.Errorf("switch/courses pair mismatch at %d: %s then %s (events=%v)", i, sw, co, events)
		}
	}
}

func TestGetCoursesForChild_SwitchHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/parent/switch_child/42" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		t.Errorf("courses should not be reached: %s", r.URL.Path)
	}))
	defer server.Close()

	client := withMockedBase(t, server)
	_, err := client.GetCoursesForChild(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error from forbidden switch, got nil")
	}
	e, ok := err.(*Error)
	if !ok {
		t.Fatalf("err = %T %v, want *Error", err, err)
	}
	if e.Op != "viewAs" {
		t.Errorf("Op = %q, want %q", e.Op, "viewAs")
	}
}
