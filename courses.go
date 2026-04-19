package schoology

import (
	"context"
	"fmt"
	"net/http"
)

// GetCourses retrieves all courses for the authenticated user
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

	var result struct {
		Courses []*Course `json:"courses"`
	}

	if err := decodeJSON(resp, &result); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	return result.Courses, nil
}

// GetCourse retrieves details for a specific course
func (c *Client) GetCourse(ctx context.Context, courseID string) (*Course, error) {
	const op = "GetCourse"

	if courseID == "" {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "courseID cannot be empty",
			Op:      op,
		}
	}

	path := fmt.Sprintf("/iapi2/courses/%s", courseID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	defer resp.Body.Close()

	var course Course
	if err := decodeJSON(resp, &course); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	return &course, nil
}
