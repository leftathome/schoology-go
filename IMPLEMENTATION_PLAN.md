# schoology-go Implementation Plan

## Project Overview

**Repository**: github.com/leftathome/schoology-go
**Purpose**: Modular, accessible, easy-to-test Go library for accessing the Schoology e-learning platform
**License**: MIT
**Target Version**: v0.1.0 (MVP)

This library is being built to support the Trunchbull academic dashboard project, but is designed as a standalone, reusable package for the Go community.

---

## Design Philosophy

Based on [LIBRARY_ANALYSIS.md](../trunchbull/docs/LIBRARY_ANALYSIS.md):

1. **Standalone, reusable library** - Not monolithic, easily imported by other projects
2. **Go-native** - Leverage Go's strengths (concurrency, standard library)
3. **Well-tested** - Comprehensive test coverage with mocked responses
4. **Well-documented** - Clear API docs, examples, and usage guides
5. **MIT licensed** - Maximum reusability for the community
6. **Session-based authentication** - Start with session tokens (MVP), add credential automation later

---

## Architecture Overview

### Authentication Strategy

**Phase 1 (v0.1.0)**: Session Token Approach
- User manually extracts session cookies from browser
- Library uses these cookies to make authenticated requests
- Simple, secure, user controls authentication
- No password storage required

**Phase 2 (v0.2.0)**: Credential Automation
- Add chromedp-based automated login
- Automatic session refresh
- Better UX while maintaining security

### API Access Strategy

Schoology's web interface uses internal API endpoints (`iapi2`) that don't require OAuth:
- `GET https://{school}.schoology.com/iapi2/site-navigation/courses`
- `GET https://{school}.schoology.com/iapi2/gradebook/{section_id}`
- `GET https://{school}.schoology.com/iapi2/assignment/{assignment_id}`

These endpoints work with session cookies + CSRF tokens.

---

## Project Structure

```
schoology-go/
├── README.md                 # Project overview, quick start
├── LICENSE                   # MIT license
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
├── IMPLEMENTATION_PLAN.md    # This file
├── client.go                 # Main client struct and constructor
├── auth.go                   # Authentication and session management
├── courses.go                # Course listing and details
├── assignments.go            # Assignment retrieval
├── grades.go                 # Grade retrieval
├── messages.go               # Message retrieval (Phase 2)
├── calendar.go               # Calendar/events (Phase 2)
├── types.go                  # Data models and structs
├── errors.go                 # Error types and handling
├── client_test.go            # Client tests
├── auth_test.go              # Auth tests
├── courses_test.go           # Course endpoint tests
├── assignments_test.go       # Assignment endpoint tests
├── grades_test.go            # Grade endpoint tests
├── examples/                 # Example usage
│   ├── basic/
│   │   └── main.go          # Basic usage example
│   └── session/
│       └── main.go          # Session management example
├── internal/                 # Internal packages (not exported)
│   ├── browser/             # Chromedp automation (Phase 2)
│   │   └── login.go
│   └── testdata/            # Test fixtures and mocked responses
│       ├── courses.json
│       ├── assignments.json
│       └── grades.json
└── docs/                    # Additional documentation
    ├── SESSION_EXTRACTION.md  # How to extract session cookies
    ├── API_REFERENCE.md       # Detailed API documentation
    └── CONTRIBUTING.md        # Contribution guidelines
```

---

## API Design

### Core Types

```go
package schoology

import (
    "context"
    "net/http"
    "time"
)

// Client is the main Schoology client
type Client struct {
    host       string
    session    *Session
    httpClient *http.Client
}

// Session holds authentication state
type Session struct {
    SessID     string
    CSRFToken  string
    CSRFKey    string
    UID        string
    ExpiresAt  time.Time
}

// Option configures the client
type Option func(*Client) error

// NewClient creates a new Schoology client
func NewClient(host string, opts ...Option) (*Client, error)

// WithSession sets session authentication
func WithSession(sessID, csrfToken, csrfKey, uid string) Option

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option

// WithTimeout sets request timeout
func WithTimeout(timeout time.Duration) Option
```

### Data Models

```go
// Course represents a Schoology course
type Course struct {
    ID          string
    Title       string
    CourseCode  string
    SectionID   string
    Teacher     string
    Period      string
    Room        string
    Active      bool
}

// Assignment represents a Schoology assignment
type Assignment struct {
    ID          string
    CourseID    string
    SectionID   string
    Title       string
    Description string
    DueDate     time.Time
    MaxPoints   float64
    Category    string
    Status      AssignmentStatus
    SubmittedAt *time.Time
}

type AssignmentStatus string

const (
    StatusPending   AssignmentStatus = "pending"
    StatusSubmitted AssignmentStatus = "submitted"
    StatusGraded    AssignmentStatus = "graded"
    StatusLate      AssignmentStatus = "late"
    StatusMissing   AssignmentStatus = "missing"
)

// Grade represents a grade for an assignment or course
type Grade struct {
    CourseID      string
    CourseName    string
    AssignmentID  *string  // nil for overall course grade
    Score         *float64
    MaxScore      float64
    Percentage    *float64
    LetterGrade   string
    GradingPeriod string
    LastUpdated   time.Time
    Comment       string
}

// Message represents a Schoology message (Phase 2)
type Message struct {
    ID          string
    FromName    string
    FromUserID  string
    Subject     string
    Body        string
    ReceivedAt  time.Time
    Read        bool
}

// Event represents a calendar event (Phase 2)
type Event struct {
    ID          string
    Title       string
    Description string
    StartDate   time.Time
    EndDate     time.Time
    EventType   EventType
}

type EventType string

const (
    EventAssignment   EventType = "assignment"
    EventTest         EventType = "test"
    EventEvent        EventType = "event"
    EventHoliday      EventType = "holiday"
)
```

### Core Methods (v0.1.0)

```go
// Authentication
func (c *Client) IsAuthenticated(ctx context.Context) bool
func (c *Client) RefreshSession(ctx context.Context) error

// Courses
func (c *Client) GetCourses(ctx context.Context) ([]*Course, error)
func (c *Client) GetCourse(ctx context.Context, courseID string) (*Course, error)

// Assignments
func (c *Client) GetAssignments(ctx context.Context, courseID string) ([]*Assignment, error)
func (c *Client) GetAssignment(ctx context.Context, assignmentID string) (*Assignment, error)
func (c *Client) GetUpcomingAssignments(ctx context.Context) ([]*Assignment, error)

// Grades
func (c *Client) GetGrades(ctx context.Context) ([]*Grade, error)
func (c *Client) GetCourseGrades(ctx context.Context, courseID string) ([]*Grade, error)
func (c *Client) GetAssignmentGrade(ctx context.Context, assignmentID string) (*Grade, error)
```

### Error Handling

```go
// Errors
var (
    ErrNotAuthenticated   = errors.New("not authenticated")
    ErrSessionExpired     = errors.New("session expired")
    ErrInvalidSession     = errors.New("invalid session credentials")
    ErrNotFound           = errors.New("resource not found")
    ErrRateLimited        = errors.New("rate limited")
    ErrInvalidResponse    = errors.New("invalid response from server")
)

// Error wraps errors with additional context
type Error struct {
    Op      string // Operation being performed
    Err     error  // Underlying error
    Message string // Human-readable message
}

func (e *Error) Error() string
func (e *Error) Unwrap() error
```

---

## Implementation Phases

### Phase 1: MVP (v0.1.0) - Week 1

**Goal**: Core functionality with session-based authentication

#### Task 1.1: Project Setup
- [x] Create repository structure
- [ ] Initialize Go module
- [ ] Add MIT license
- [ ] Create basic README
- [ ] Set up .gitignore

#### Task 1.2: Core Client Implementation
- [ ] Implement `Client` struct
- [ ] Implement `Session` struct
- [ ] Create `NewClient()` constructor
- [ ] Add option pattern for configuration
- [ ] Implement HTTP client with cookie jar
- [ ] Add request/response logging

#### Task 1.3: Authentication
- [ ] Implement session validation
- [ ] Add CSRF token handling
- [ ] Build cookie management
- [ ] Create session expiration detection
- [ ] Add error handling for auth failures

#### Task 1.4: Courses Endpoint
- [ ] Implement `GetCourses()`
- [ ] Implement `GetCourse()`
- [ ] Parse course JSON responses
- [ ] Map to `Course` struct
- [ ] Handle pagination if needed
- [ ] Add error handling

#### Task 1.5: Assignments Endpoint
- [ ] Implement `GetAssignments()`
- [ ] Implement `GetAssignment()`
- [ ] Implement `GetUpcomingAssignments()`
- [ ] Parse assignment JSON responses
- [ ] Map to `Assignment` struct
- [ ] Handle date/time parsing
- [ ] Add error handling

#### Task 1.6: Grades Endpoint
- [ ] Implement `GetGrades()`
- [ ] Implement `GetCourseGrades()`
- [ ] Implement `GetAssignmentGrade()`
- [ ] Parse gradebook responses
- [ ] Map to `Grade` struct
- [ ] Handle missing/null grades
- [ ] Add error handling

#### Task 1.7: Testing
- [ ] Create mock HTTP responses (testdata)
- [ ] Write unit tests for client
- [ ] Write unit tests for auth
- [ ] Write unit tests for courses
- [ ] Write unit tests for assignments
- [ ] Write unit tests for grades
- [ ] Aim for >80% coverage

#### Task 1.8: Documentation
- [ ] Write comprehensive README
- [ ] Create SESSION_EXTRACTION.md guide
- [ ] Add GoDoc comments to all exported types/functions
- [ ] Create basic usage example
- [ ] Create session management example
- [ ] Add troubleshooting section

#### Task 1.9: Release Preparation
- [ ] Run `go mod tidy`
- [ ] Run `go fmt ./...`
- [ ] Run `go vet ./...`
- [ ] Run all tests
- [ ] Tag v0.1.0 release
- [ ] Publish to GitHub

**Deliverables**:
- Working Go library with session-based auth
- Course, assignment, and grade retrieval
- Comprehensive tests and documentation
- Published v0.1.0 on GitHub

---

### Phase 2: Enhanced Features (v0.2.0) - Week 2

**Goal**: Add messages, calendar, and credential automation

#### Task 2.1: Messages Support
- [ ] Implement `GetMessages()`
- [ ] Implement `GetMessage()`
- [ ] Add read/unread filtering
- [ ] Parse message responses
- [ ] Add tests

#### Task 2.2: Calendar Support
- [ ] Implement `GetCalendar()`
- [ ] Implement `GetEvents()`
- [ ] Parse calendar responses
- [ ] Handle recurring events
- [ ] Add tests

#### Task 2.3: Credential Automation
- [ ] Add chromedp dependency
- [ ] Implement automated login
- [ ] Extract session cookies programmatically
- [ ] Add credential encryption
- [ ] Create `WithCredentials()` option
- [ ] Add tests

#### Task 2.4: Session Management Improvements
- [ ] Automatic session refresh
- [ ] Session persistence to disk
- [ ] Session expiration warnings
- [ ] Retry logic with backoff
- [ ] Circuit breaker pattern

#### Task 2.5: Documentation Updates
- [ ] Document new features
- [ ] Add credential automation guide
- [ ] Update examples
- [ ] Add security considerations

#### Task 2.6: Release
- [ ] Tag v0.2.0
- [ ] Update CHANGELOG
- [ ] Publish release

---

### Phase 3: Polish & Advanced Features (v0.3.0) - Week 3

**Goal**: Production-ready with advanced capabilities

#### Task 3.1: Advanced Features
- [ ] Rate limiting client-side
- [ ] Request caching
- [ ] Batch requests
- [ ] Webhook support (if available)
- [ ] File download support

#### Task 3.2: Error Handling Improvements
- [ ] Structured error types
- [ ] Retry policies
- [ ] Error categorization
- [ ] Better error messages

#### Task 3.3: Performance Optimizations
- [ ] Connection pooling
- [ ] Request concurrency
- [ ] Response streaming
- [ ] Memory optimizations

#### Task 3.4: Developer Experience
- [ ] CLI tool for testing
- [ ] Debug logging mode
- [ ] Request/response inspection
- [ ] Mock server for testing

---

## Testing Strategy

### Unit Tests
- Mock HTTP responses using `httptest`
- Test all public methods
- Test error conditions
- Test edge cases (nil values, empty responses)
- Use table-driven tests where appropriate

### Integration Tests
- Use real session tokens (from environment variables)
- Test against actual Schoology instance
- Run selectively (not in CI by default)
- Document how to run integration tests

### Example Test Structure

```go
func TestClient_GetCourses(t *testing.T) {
    tests := []struct {
        name       string
        response   string
        statusCode int
        want       []*Course
        wantErr    bool
    }{
        {
            name:       "successful response",
            response:   testdata.CoursesJSON,
            statusCode: http.StatusOK,
            want:       []*Course{...},
            wantErr:    false,
        },
        {
            name:       "session expired",
            response:   "",
            statusCode: http.StatusUnauthorized,
            want:       nil,
            wantErr:    true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

---

## Dependencies

### Core Dependencies (v0.1.0)
```go
require (
    // No external dependencies for basic HTTP client
    // Uses only Go standard library
)
```

### Optional Dependencies (v0.2.0+)
```go
require (
    github.com/chromedp/chromedp v0.9.5  // For credential automation
)
```

### Development Dependencies
```go
require (
    github.com/stretchr/testify v1.9.0  // For assertions in tests
)
```

---

## Security Considerations

### Session Token Handling
- Never log session tokens
- Store in memory only (don't persist to disk in v0.1.0)
- Clear on client shutdown
- Validate expiration before use

### HTTPS Enforcement
- All requests must use HTTPS
- Validate TLS certificates
- No insecure connections allowed

### Error Messages
- Don't expose tokens in error messages
- Sanitize logs
- Provide safe debugging information

### Rate Limiting
- Implement client-side rate limiting
- Default: 60 requests/minute
- Respect server rate limit headers
- Add backoff on rate limit errors

---

## Documentation Plan

### README.md
- Overview and features
- Quick start guide
- Installation instructions
- Basic usage example
- Link to full documentation
- License and contributing info

### SESSION_EXTRACTION.md
- Step-by-step guide with screenshots
- Chrome DevTools instructions
- Firefox Developer Tools instructions
- Safari Web Inspector instructions
- Troubleshooting common issues
- Security warnings

### API_REFERENCE.md
- Complete API documentation
- All types and methods
- Request/response examples
- Error handling guide
- Best practices

### Examples
- `examples/basic/` - Simple course listing
- `examples/session/` - Session management
- `examples/assignments/` - Upcoming assignments
- `examples/grades/` - Grade retrieval

---

## Success Criteria

### v0.1.0 Release Criteria
- [ ] All core endpoints implemented (courses, assignments, grades)
- [ ] Session authentication working
- [ ] Test coverage >80%
- [ ] All tests passing
- [ ] Documentation complete
- [ ] Examples working
- [ ] No critical security issues
- [ ] Successfully integrates with Trunchbull

### Quality Metrics
- Code coverage: >80%
- Go vet: 0 issues
- golangci-lint: 0 issues
- Documentation coverage: 100% of exported symbols
- Example success rate: 100%

---

## Integration with Trunchbull

Once v0.1.0 is released, Trunchbull will import it as:

```go
import "github.com/leftathome/schoology-go"

func main() {
    client, err := schoology.NewClient(
        "meanyms.schoology.com",
        schoology.WithSession(
            os.Getenv("SCHOOLOGY_SESS_ID"),
            os.Getenv("SCHOOLOGY_CSRF_TOKEN"),
            os.Getenv("SCHOOLOGY_CSRF_KEY"),
            os.Getenv("SCHOOLOGY_UID"),
        ),
    )

    ctx := context.Background()
    courses, err := client.GetCourses(ctx)
    // Use courses in dashboard
}
```

---

## Timeline

| Week | Focus | Deliverables |
|------|-------|--------------|
| 1 | Core implementation | v0.1.0 with session auth + basic endpoints |
| 2 | Enhanced features | v0.2.0 with messages, calendar, credentials |
| 3 | Polish | v0.3.0 with advanced features |
| 4 | Integration | Use in Trunchbull |

---

## Open Questions

1. **API Endpoint Discovery**
   - Need to inspect Schoology web app to find exact endpoints
   - Document API structure as we discover it
   - May need to adjust based on district's Schoology version

2. **Session Expiration Handling**
   - How long do sessions last? (Testing needed)
   - Should we auto-refresh or require manual re-auth?
   - Decision: Manual for v0.1.0, auto for v0.2.0

3. **Error Handling Strategy**
   - Return structured errors or sentinel errors?
   - Decision: Both - sentinel for common cases, structured for details

4. **Pagination**
   - Do Schoology endpoints paginate results?
   - How to handle large course/assignment lists?
   - Decision: Implement if needed, document if not

---

## Next Steps

1. **Immediate** (Today):
   - Initialize Go module
   - Create project structure
   - Set up basic files (LICENSE, README)

2. **This Week**:
   - Implement core client
   - Implement authentication
   - Start endpoint implementation
   - Begin testing

3. **Validation**:
   - Test with real Schoology account
   - Verify API endpoints
   - Confirm data structures match responses

---

## Notes

- This library fills a gap - no Go libraries exist for Schoology credential-based access
- Keep it simple and focused for v0.1.0
- Prioritize working code over perfect code
- Document everything as we learn
- Be prepared to adjust based on actual API behavior

---

**Plan Version**: 1.0
**Created**: 2025-10-24
**Status**: Ready to implement
**Next Review**: After v0.1.0 completion
