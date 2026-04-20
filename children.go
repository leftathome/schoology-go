package schoology

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// viewAs flips the server-side session.view_child field to childUID by
// hitting /parent/switch_child/{childUID}. Schoology responds with a
// 200 text/html render of /parent/home; we drain and discard that body.
//
// This method is not part of the public surface: callers should reach
// for one of the per-child wrappers (GetCoursesForChild, …) which
// additionally hold the viewChildMu mutex so the follow-up read sees
// the intended child.
func (c *Client) viewAs(ctx context.Context, childUID int64) error {
	const op = "viewAs"

	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/parent/switch_child/%d", childUID), nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return err
	}
	defer resp.Body.Close()

	// Drain so the connection can be reused. The body is the rendered
	// /parent/home page; we don't care about its content.
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// GetCoursesForChild returns the courses/sections enrolled by the
// specified child from the parent account's perspective. It works by
// switching the session's view_child to childUID and then calling the
// same /iapi2/site-navigation/courses endpoint that GetCourses uses.
//
// The switch + fetch is serialized by Client.viewChildMu so concurrent
// GetCoursesForChild calls for different children do not race on the
// shared server-side view state.
func (c *Client) GetCoursesForChild(ctx context.Context, childUID int64) ([]*Course, error) {
	c.viewChildMu.Lock()
	defer c.viewChildMu.Unlock()

	if err := c.viewAs(ctx, childUID); err != nil {
		return nil, err
	}
	return c.GetCourses(ctx)
}
