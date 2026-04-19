# Contributing to schoology-go

Thank you for your interest in contributing to `schoology-go`! This document provides guidelines and instructions for contributing.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Remember this handles sensitive student data - security and privacy are paramount

## Getting Started

### Prerequisites

- Go 1.23 or later
- Git
- A Schoology account for testing (parent or student)
- (Optional) 1Password CLI for secure credential management

### Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/schoology-go.git
   cd schoology-go
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Run tests:
   ```bash
   go test -v
   ```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

Use prefixes:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation only
- `test/` - Test improvements
- `refactor/` - Code refactoring

### 2. Make Changes

- Write clean, idiomatic Go code
- Follow existing code style
- Add tests for new functionality
- Update documentation as needed

### 3. Run Tests

```bash
# Unit tests
go test -v

# With coverage
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# Integration tests (requires session credentials)
# See docs/SESSION_EXTRACTION.md for setup
go test -tags=integration -v
```

### 4. Format and Lint

```bash
# Format code
go fmt ./...

# Run linter
go vet ./...

# (Optional) Use golangci-lint for comprehensive linting
golangci-lint run
```

### 5. Commit Changes

Use clear, descriptive commit messages:

```
Add GetMessages endpoint for retrieving Schoology messages

- Implements GET /iapi2/messages
- Adds Message type to types.go
- Includes unit tests with mocked responses
- Updates README with example usage
```

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a PR on GitHub with:
- Clear title and description
- Reference any related issues
- List what was changed and why
- Include test results

## Testing Guidelines

### Unit Tests

- Mock external API calls
- Test happy paths and error cases
- Use table-driven tests when appropriate
- Aim for >80% coverage

Example:
```go
func TestGetCourses(t *testing.T) {
	tests := []struct {
		name       string
		mockData   string
		wantErr    bool
		wantCount  int
	}{
		{
			name:      "success",
			mockData:  testdata.CoursesJSON,
			wantErr:   false,
			wantCount: 2,
		},
		// ... more test cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test implementation
		})
	}
}
```

### Integration Tests

**IMPORTANT**: Integration tests use real student data.

- Only run locally, never in CI
- Use `//go:build integration` tag
- Store credentials securely (1Password recommended)
- Be respectful of API rate limits
- Clean up test data if applicable

### Test Data

- Store mock JSON responses in `internal/testdata/`
- Anonymize any real data used in examples
- Don't commit real credentials or session tokens

## What to Contribute

### High Priority

- Additional endpoints (messages, events, materials, etc.)
- Error handling improvements
- Documentation improvements
- Test coverage increases
- Bug fixes

### Medium Priority

- Performance optimizations
- Better examples
- CLI tools
- Rate limiting improvements

### Future Features

- Credential-based authentication (chromedp)
- Automatic session refresh
- Response caching
- WebSocket support for real-time updates

## API Endpoint Implementation

When adding a new endpoint:

1. **Add types** to `types.go`:
   ```go
   type Message struct {
       ID      string    `json:"id"`
       Subject string    `json:"subject"`
       // ...
   }
   ```

2. **Create endpoint file** (e.g., `messages.go`):
   ```go
   func (c *Client) GetMessages(ctx context.Context) ([]*Message, error) {
       // Implementation
   }
   ```

3. **Add tests**:
   - Unit tests in `messages_test.go`
   - Mock data in `internal/testdata/messages.json`
   - Integration test in `integration_test.go`

4. **Update documentation**:
   - Add to README.md API overview
   - Create example in `examples/`
   - Update IMPLEMENTATION_PLAN.md

## Documentation

### Code Documentation

- All exported types and functions MUST have GoDoc comments
- Start comments with the name of the thing being documented
- Provide usage examples for complex functions

Example:
```go
// GetCourses retrieves all courses for the authenticated user.
// It returns an empty slice if no courses are found.
// The courses are returned in the order provided by Schoology.
//
// Example:
//
//	courses, err := client.GetCourses(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, course := range courses {
//	    fmt.Println(course.Title)
//	}
func (c *Client) GetCourses(ctx context.Context) ([]*Course, error) {
	// Implementation
}
```

### User Documentation

- Update README.md for new features
- Add examples for new functionality
- Keep SESSION_EXTRACTION.md current
- Update IMPLEMENTATION_PLAN.md progress

## Security Considerations

### Sensitive Data

- NEVER log or print session credentials
- NEVER commit credentials to the repository
- Be careful with error messages (don't expose tokens)
- Sanitize any debug output

### Code Review Checklist

Before submitting:
- [ ] No hardcoded credentials
- [ ] No credentials in test files
- [ ] Error messages don't leak sensitive data
- [ ] Proper input validation
- [ ] HTTPS enforced where applicable

## Pull Request Process

1. **Before Submitting**:
   - All tests pass
   - Code is formatted (`go fmt`)
   - No linter warnings (`go vet`)
   - Documentation updated
   - CHANGELOG.md updated (if applicable)

2. **PR Description Should Include**:
   - What changed and why
   - Any breaking changes
   - Testing performed
   - Screenshots/examples (if applicable)

3. **Review Process**:
   - Maintainers will review within a few days
   - Address feedback and update PR
   - Once approved, maintainer will merge

4. **After Merge**:
   - Delete your branch
   - Pull latest changes to your fork

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: New functionality (backward compatible)
- **PATCH** version: Bug fixes (backward compatible)

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for questions
- Email maintainers for security concerns

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to schoology-go!
