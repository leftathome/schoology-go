package schoology

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// GetAssignments retrieves all assignments for a specific course/section
func (c *Client) GetAssignments(ctx context.Context, sectionID string) ([]*Assignment, error) {
	const op = "GetAssignments"

	if sectionID == "" {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "sectionID cannot be empty",
			Op:      op,
		}
	}

	path := fmt.Sprintf("/iapi2/sections/%s/assignments", sectionID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Assignments []*Assignment `json:"assignments"`
	}

	if err := decodeJSON(resp, &result); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	return result.Assignments, nil
}

// GetAssignment retrieves details for a specific assignment
func (c *Client) GetAssignment(ctx context.Context, assignmentID string) (*Assignment, error) {
	const op = "GetAssignment"

	if assignmentID == "" {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "assignmentID cannot be empty",
			Op:      op,
		}
	}

	path := fmt.Sprintf("/iapi2/assignments/%s", assignmentID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	defer resp.Body.Close()

	var assignment Assignment
	if err := decodeJSON(resp, &assignment); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	return &assignment, nil
}

// GetUpcomingAssignments retrieves all upcoming assignments across all courses
func (c *Client) GetUpcomingAssignments(ctx context.Context) ([]*Assignment, error) {
	const op = "GetUpcomingAssignments"

	// First get all courses
	courses, err := c.GetCourses(ctx)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "failed to retrieve courses",
			Op:      op,
			Err:     err,
		}
	}

	// Collect all assignments from all courses
	var allAssignments []*Assignment
	now := time.Now()

	for _, course := range courses {
		if !course.Active {
			continue
		}

		assignments, err := c.GetAssignments(ctx, course.SectionID)
		if err != nil {
			// Log error but continue with other courses
			continue
		}

		// Filter for upcoming assignments only
		for _, assignment := range assignments {
			if assignment.DueDate.After(now) && assignment.Status != StatusGraded {
				allAssignments = append(allAssignments, assignment)
			}
		}
	}

	return allAssignments, nil
}
