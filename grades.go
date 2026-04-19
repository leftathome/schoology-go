package schoology

import (
	"context"
	"fmt"
	"net/http"
)

// GetGrades retrieves all grades for the authenticated user
func (c *Client) GetGrades(ctx context.Context) ([]*Grade, error) {
	const op = "GetGrades"

	// Get all courses first
	courses, err := c.GetCourses(ctx)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "failed to retrieve courses",
			Op:      op,
			Err:     err,
		}
	}

	var allGrades []*Grade
	for _, course := range courses {
		if !course.Active {
			continue
		}

		grades, err := c.GetCourseGrades(ctx, course.SectionID)
		if err != nil {
			// Continue with other courses even if one fails
			continue
		}

		allGrades = append(allGrades, grades...)
	}

	return allGrades, nil
}

// GetCourseGrades retrieves all grades for a specific course/section
func (c *Client) GetCourseGrades(ctx context.Context, sectionID string) ([]*Grade, error) {
	const op = "GetCourseGrades"

	if sectionID == "" {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "sectionID cannot be empty",
			Op:      op,
		}
	}

	path := fmt.Sprintf("/iapi2/sections/%s/grades", sectionID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Grades []*Grade `json:"grades"`
	}

	if err := decodeJSON(resp, &result); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	return result.Grades, nil
}

// GetAssignmentGrade retrieves the grade for a specific assignment
func (c *Client) GetAssignmentGrade(ctx context.Context, sectionID, assignmentID string) (*Grade, error) {
	const op = "GetAssignmentGrade"

	if sectionID == "" || assignmentID == "" {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "sectionID and assignmentID cannot be empty",
			Op:      op,
		}
	}

	path := fmt.Sprintf("/iapi2/sections/%s/grades/%s", sectionID, assignmentID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	defer resp.Body.Close()

	var grade Grade
	if err := decodeJSON(resp, &grade); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	return &grade, nil
}

// GetGradingScale retrieves the grading scale for a section
func (c *Client) GetGradingScale(ctx context.Context, sectionID string) (*GradingScale, error) {
	const op = "GetGradingScale"

	if sectionID == "" {
		return nil, &Error{
			Code:    ErrCodeClient,
			Message: "sectionID cannot be empty",
			Op:      op,
		}
	}

	path := fmt.Sprintf("/iapi2/sections/%s/grading_scales", sectionID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}
	defer resp.Body.Close()

	var scale GradingScale
	if err := decodeJSON(resp, &scale); err != nil {
		if e, ok := err.(*Error); ok {
			e.Op = op
		}
		return nil, err
	}

	return &scale, nil
}
