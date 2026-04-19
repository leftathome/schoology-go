package schoology

import (
	"context"
	"net/http"
	"sort"
)

// GetCourses returns the courses/sections the current session is enrolled in.
// For a parent session the result is scoped to the currently-selected child
// (session.view_child in /iapi/parent/info); see GetChildren.
func (c *Client) GetCourses(ctx context.Context) ([]*Course, error) {
	const op = "GetCourses"

	resp, err := c.do(ctx, http.MethodGet, "/iapi2/site-navigation/courses", nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	defer resp.Body.Close()

	var env coursesEnvelope
	if err := decodeJSON(resp, &env); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	return env.Data.Courses, nil
}

// GetChildren returns the children associated with a parent account, derived
// from /iapi/parent/info. Returns an empty slice for non-parent accounts.
// Order is by UID for stability.
func (c *Client) GetChildren(ctx context.Context) ([]*Child, error) {
	const op = "GetChildren"

	resp, err := c.do(ctx, http.MethodGet, "/iapi/parent/info", nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	defer resp.Body.Close()

	var env parentInfoEnvelope
	if err := decodeJSON(resp, &env); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	out := make([]*Child, 0, len(env.Body.Children))
	for _, ch := range env.Body.Children {
		out = append(out, ch)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UID < out[j].UID })
	return out, nil
}
