package schoology

// Types here describe the endpoints verified against live Schoology
// traffic. Speculative types for assignments, grades, messages,
// attendance, etc. used to live here; they were removed after the
// discovery pass found no clean JSON endpoints for those resources on
// a parent account. Re-add them as real types when the HTML-parsing
// work lands (see bd issues).

// Course represents a single course/section enrollment as returned by
// /iapi2/site-navigation/courses. The frontend calls the numeric id
// "nid" (the section node id) and "courseNid" (the parent course node
// id); these are the keys the API returns, not the v1 REST field names.
type Course struct {
	NID                   int64  `json:"nid"`                   // section id
	CourseNID             int64  `json:"courseNid"`             // parent course id
	CourseTitle           string `json:"courseTitle"`           // e.g. "Advisory"
	SectionTitle          string `json:"sectionTitle"`          // e.g. "S2 7(A) 1013"
	BuildingTitle         string `json:"buildingTitle"`         // school name
	LogoImgSrc            string `json:"logoImgSrc"`            // URL
	Weight                int    `json:"weight"`                // display order
	CourseLandingPageType string `json:"courseLandingPageType"` // e.g. "materials"
	IsCSL                 bool   `json:"isCsl"`
	AdminType             string `json:"adminType"` // viewer's role in the section
}

// coursesEnvelope is the outer shape of the /iapi2/site-navigation/courses
// response: {"data": {"courses": [...]}}.
type coursesEnvelope struct {
	Data struct {
		Courses []*Course `json:"courses"`
	} `json:"data"`
}

// Child is a single entry under /iapi/parent/info body.children.
// The response object is keyed by UID; we flatten it into a slice.
type Child struct {
	UID          int64  `json:"uid"`
	Username     string `json:"username"`
	ProfileTiny  string `json:"profile_tiny"`
	ProfileURL   string `json:"profile_url"`
	RecentCounts map[string]int `json:"recent_counts"`
}

// parentInfoEnvelope is the outer shape of the /iapi/parent/info response.
type parentInfoEnvelope struct {
	ResponseCode int `json:"response_code"`
	Body         struct {
		Session struct {
			ViewMode  int    `json:"view_mode"`
			ViewChild string `json:"view_child"`
		} `json:"session"`
		Children map[string]*Child `json:"children"`
	} `json:"body"`
}
