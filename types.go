package schoology

import "time"

// Course represents a Schoology course/section
type Course struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	CourseCode  string    `json:"course_code"`
	SectionID   string    `json:"section_id"`
	SectionCode string    `json:"section_code"`
	Teacher     string    `json:"teacher"`
	TeacherID   string    `json:"teacher_id"`
	Period      string    `json:"period"`
	Room        string    `json:"room"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	GradeLevel  []int     `json:"grade_level"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
}

// Assignment represents a Schoology assignment
type Assignment struct {
	ID           string           `json:"id"`
	CourseID     string           `json:"course_id"`
	SectionID    string           `json:"section_id"`
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	DueDate      time.Time        `json:"due_date"`
	MaxPoints    float64          `json:"max_points"`
	Category     string           `json:"category"`
	CategoryID   string           `json:"category_id"`
	Status       AssignmentStatus `json:"status"`
	SubmittedAt  *time.Time       `json:"submitted_at,omitempty"`
	GradedAt     *time.Time       `json:"graded_at,omitempty"`
	Type         AssignmentType   `json:"type"`
	AllowLate    bool             `json:"allow_late"`
	ShowComments bool             `json:"show_comments"`
	Published    bool             `json:"published"`
}

// AssignmentStatus represents the status of an assignment
type AssignmentStatus string

const (
	StatusPending   AssignmentStatus = "pending"
	StatusSubmitted AssignmentStatus = "submitted"
	StatusGraded    AssignmentStatus = "graded"
	StatusLate      AssignmentStatus = "late"
	StatusMissing   AssignmentStatus = "missing"
	StatusExcused   AssignmentStatus = "excused"
)

// AssignmentType represents the type of assignment
type AssignmentType string

const (
	TypeAssignment AssignmentType = "assignment"
	TypeQuiz       AssignmentType = "quiz"
	TypeTest       AssignmentType = "test"
	TypeProject    AssignmentType = "project"
	TypeHomework   AssignmentType = "homework"
	TypeOther      AssignmentType = "other"
)

// Grade represents a grade for an assignment or course
type Grade struct {
	CourseID       string    `json:"course_id"`
	CourseName     string    `json:"course_name"`
	SectionID      string    `json:"section_id"`
	AssignmentID   *string   `json:"assignment_id,omitempty"` // nil for overall course grade
	AssignmentName string    `json:"assignment_name,omitempty"`
	Score          *float64  `json:"score,omitempty"`
	MaxScore       float64   `json:"max_score"`
	Percentage     *float64  `json:"percentage,omitempty"`
	LetterGrade    string    `json:"letter_grade"`
	GradingPeriod  string    `json:"grading_period"`
	LastUpdated    time.Time `json:"last_updated"`
	Comment        string    `json:"comment,omitempty"`
	ExceptionType  string    `json:"exception_type,omitempty"` // "late", "missing", "excused", "incomplete"
	OverrideGrade  *float64  `json:"override_grade,omitempty"`
}

// GradingScale represents grading scale information
type GradingScale struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Levels      []GradingScaleLevel `json:"levels"`
}

// GradingScaleLevel represents a level in a grading scale (e.g., A, B, C)
type GradingScaleLevel struct {
	Grade      string  `json:"grade"`
	MinPercent float64 `json:"min_percent"`
	MaxPercent float64 `json:"max_percent"`
	GPAValue   float64 `json:"gpa_value"`
}

// Message represents a Schoology private message
type Message struct {
	ID         string    `json:"id"`
	FromName   string    `json:"from_name"`
	FromUserID string    `json:"from_user_id"`
	ToUserIDs  []string  `json:"to_user_ids"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	ReceivedAt time.Time `json:"received_at"`
	Read       bool      `json:"read"`
	HasReply   bool      `json:"has_reply"`
}

// Event represents a calendar event in Schoology
type Event struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	AllDay      bool      `json:"all_day"`
	EventType   EventType `json:"event_type"`
	CourseID    string    `json:"course_id,omitempty"`
	CourseName  string    `json:"course_name,omitempty"`
}

// EventType represents the type of calendar event
type EventType string

const (
	EventAssignment   EventType = "assignment"
	EventTest         EventType = "test"
	EventEvent        EventType = "event"
	EventHoliday      EventType = "holiday"
	EventNoSchool     EventType = "no_school"
	EventEarlyRelease EventType = "early_release"
	EventOther        EventType = "other"
)

// Update represents a course update/announcement
type Update struct {
	ID          string       `json:"id"`
	CourseID    string       `json:"course_id"`
	CourseName  string       `json:"course_name"`
	Title       string       `json:"title"`
	Body        string       `json:"body"`
	AuthorName  string       `json:"author_name"`
	AuthorID    string       `json:"author_id"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	NumComments int          `json:"num_comments"`
	Likes       int          `json:"likes"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Announcement represents a school-wide announcement
type Announcement struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Body        string       `json:"body"`
	AuthorName  string       `json:"author_name"`
	AuthorID    string       `json:"author_id"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	NumComments int          `json:"num_comments"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Pinned      bool         `json:"pinned"`
}

// Discussion represents a course discussion thread
type Discussion struct {
	ID          string       `json:"id"`
	CourseID    string       `json:"course_id"`
	CourseName  string       `json:"course_name"`
	Title       string       `json:"title"`
	Body        string       `json:"body"`
	AuthorName  string       `json:"author_name"`
	AuthorID    string       `json:"author_id"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	NumReplies  int          `json:"num_replies"`
	Likes       int          `json:"likes"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Locked      bool         `json:"locked"`
	Published   bool         `json:"published"`
}

// Attachment represents a file attachment
type Attachment struct {
	ID         string    `json:"id"`
	FileName   string    `json:"file_name"`
	FileType   string    `json:"file_type"`
	FileSize   int64     `json:"file_size"`
	URL        string    `json:"url"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// User represents a Schoology user (student, parent, teacher)
type User struct {
	ID                string   `json:"id"`
	Username          string   `json:"username"`
	Email             string   `json:"email"`
	FirstName         string   `json:"first_name"`
	LastName          string   `json:"last_name"`
	PrimaryEmail      string   `json:"primary_email"`
	Position          string   `json:"position"`
	Gender            string   `json:"gender"`
	GradeLevel        string   `json:"grade_level"`
	GraduationYear    int      `json:"graduation_year"`
	ParentIDs         []string `json:"parent_ids,omitempty"`
	ChildIDs          []string `json:"child_ids,omitempty"`
	ProfilePictureURL string   `json:"profile_picture_url,omitempty"`
}

// Attendance represents attendance information
type Attendance struct {
	ID         string           `json:"id"`
	StudentID  string           `json:"student_id"`
	CourseID   string           `json:"course_id,omitempty"`
	CourseName string           `json:"course_name,omitempty"`
	Date       time.Time        `json:"date"`
	Period     string           `json:"period,omitempty"`
	Status     AttendanceStatus `json:"status"`
	Comment    string           `json:"comment,omitempty"`
	Excused    bool             `json:"excused"`
}

// AttendanceStatus represents attendance status
type AttendanceStatus string

const (
	AttendancePresent AttendanceStatus = "present"
	AttendanceAbsent  AttendanceStatus = "absent"
	AttendanceTardy   AttendanceStatus = "tardy"
	AttendanceExcused AttendanceStatus = "excused"
)

// GradingPeriod represents a grading period/term
type GradingPeriod struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Active    bool      `json:"active"`
}

// Material represents course materials (documents, links, etc.)
type Material struct {
	ID          string       `json:"id"`
	CourseID    string       `json:"course_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Type        MaterialType `json:"type"`
	URL         string       `json:"url,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	FolderID    string       `json:"folder_id,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// MaterialType represents the type of course material
type MaterialType string

const (
	MaterialDocument MaterialType = "document"
	MaterialLink     MaterialType = "link"
	MaterialVideo    MaterialType = "video"
	MaterialImage    MaterialType = "image"
	MaterialOther    MaterialType = "other"
)

// Album represents a photo/media album
type Album struct {
	ID          string    `json:"id"`
	CourseID    string    `json:"course_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	NumPhotos   int       `json:"num_photos"`
	CoverPhoto  string    `json:"cover_photo_url,omitempty"`
}

// Page represents a course page
type Page struct {
	ID        string    `json:"id"`
	CourseID  string    `json:"course_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
