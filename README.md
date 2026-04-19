# schoology-go

> A Go client library for accessing Schoology's e-learning platform

[![Go Reference](https://pkg.go.dev/badge/github.com/leftathome/schoology-go.svg)](https://pkg.go.dev/github.com/leftathome/schoology-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/leftathome/schoology-go)](https://goreportcard.com/report/github.com/leftathome/schoology-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Overview

`schoology-go` is a Go client library for accessing Schoology's Learning Management System. It provides a simple, idiomatic Go interface for retrieving courses, assignments, grades, and more.

This library is designed for parents and students who want to programmatically access their own educational data for personal use (dashboards, analytics, notifications, etc.).

## Status

> **Pre-v0.1.0.** The library talks to Schoology's internal `iapi/iapi2`
> endpoints via session cookies. The surface is currently limited to the
> two endpoints we've verified against live traffic (courses and
> parent/children). Assignments, grades, and messages require HTML
> parsing for parent accounts and are planned for v0.1.0 — see
> `IMPLEMENTATION_PLAN.md` and `bd list`.

## Features

- Session-based authentication (no OAuth approval needed)
- List the courses/sections on the current enrollment
- List the children on a parent account
- Context-aware API for timeouts and cancellation
- Typed errors with retryability helpers
- Zero external dependencies

## Installation

```bash
go get github.com/leftathome/schoology-go
```

## Quick Start

### 1. Extract Your Session Cookies

See [docs/SESSION_EXTRACTION.md](docs/SESSION_EXTRACTION.md) for detailed instructions.

In Chrome:
1. Log into Schoology
2. Open DevTools (F12) → Application → Cookies
3. Find your Schoology domain cookies
4. Copy: `SESS*`, `CSRF_TOKEN`, `CSRF_KEY`, and your user ID

### 2. Use the Library

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/leftathome/schoology-go"
)

func main() {
    ctx := context.Background()

    // Create client with session cookies
    client, err := schoology.NewClient(
        "yourschool.schoology.com",
        schoology.WithSession(
            "your-session-id",
            "your-csrf-token",
            "your-csrf-key",
            "your-user-id",
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    children, err := client.GetChildren(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, ch := range children {
        fmt.Printf("child %d: %s\n", ch.UID, ch.Username)
    }

    courses, err := client.GetCourses(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, c := range courses {
        fmt.Printf("%s / %s @ %s\n", c.CourseTitle, c.SectionTitle, c.BuildingTitle)
    }
}
```

## Documentation

- [Session Extraction Guide](docs/SESSION_EXTRACTION.md) - How to get your session cookies
- [API Reference](https://pkg.go.dev/github.com/leftathome/schoology-go) - Full API documentation
- [Examples](examples/) - Working code examples

## API Overview

### Client Creation

```go
client, err := schoology.NewClient(host, options...)
```

Options:
- `WithSession(sessID, csrfToken, csrfKey, uid)` - Set session authentication
- `WithHTTPClient(httpClient)` - Use custom HTTP client
- `WithTimeout(duration)` - Set request timeout

### Available Methods (currently verified against live Schoology)

**Courses:**
- `GetCourses(ctx) ([]*Course, error)` — list courses for the current enrollment. For a parent account this returns the courses for the currently selected child (`session.view_child` in `/iapi/parent/info`).

**Parent accounts:**
- `GetChildren(ctx) ([]*Child, error)` — list the parent's children.

**Session:**
- `ValidateSession(ctx) error` — confirm the session still works.
- `GetSessionInfo() *Session` — snapshot of the current session.
- `SessionTimeRemaining() time.Duration` — approximate time until expiry.

Assignments, grades, messages, attendance, and calendar events are planned for v0.1.0 via HTML parsing. See `IMPLEMENTATION_PLAN.md` and open `bd` issues.

## Examples

See the [examples/](examples/) directory:

- `examples/basic/` — validate a session, list children, list courses

## Security & Privacy

**IMPORTANT**: This library handles sensitive educational data protected by FERPA and other privacy laws.

- Only use with your own credentials or those you're authorized to access
- Never share session tokens or credentials
- All data stays on your infrastructure (library makes no external calls except to Schoology)
- Sessions expire - you'll need to refresh cookies periodically
- Use HTTPS for all connections

See [Security Considerations](#security-considerations) for more details.

## Authentication Methods

### Session Tokens (Current - v0.1.0)

This approach requires manually extracting session cookies from your browser. Sessions typically last 7-14 days.

**Advantages:**
- No password storage
- You control authentication
- Simple and secure

**Disadvantages:**
- Manual cookie refresh needed
- Requires browser DevTools knowledge

### Credential Automation (Coming in v0.2.0)

Future versions will support automated login with username/password.

## Limitations

- Session-based auth requires periodic manual cookie refresh
- Some Schoology features may not be available via internal APIs
- API endpoints are reverse-engineered and may change
- Rate limiting is client-side only (be conservative)

## Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires real session)
export SCHOOLOGY_HOST="yourschool.schoology.com"
export SCHOOLOGY_SESS_ID="your-session"
export SCHOOLOGY_CSRF_TOKEN="your-token"
export SCHOOLOGY_CSRF_KEY="your-key"
export SCHOOLOGY_UID="your-uid"
go test -tags=integration ./...
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Roadmap

### Pre-v0.1.0 (current)
- [x] Session-based authentication against verified endpoints
- [x] Courses list (live-verified against a real Schoology instance)
- [x] Parent children list (`/iapi/parent/info`)

### v0.1.0 (next)
- [ ] Assignments, grades, messages via HTML parsing
- [ ] Redacted HTML fixtures and per-page parsers
- [ ] Child-switch helper for parent accounts

### v0.2.0 (planned)
- [ ] Credential-based authentication (chromedp or Playwright)
- [ ] Automatic session refresh
- [ ] Calendar / events

### v0.3.0 (future)
- [ ] Rate limiting, response caching, retries, CLI tool

## FAQ

**Q: Do I need district API approval?**
A: No! This library uses session-based authentication, just like your browser.

**Q: Is this legal?**
A: Yes, for personal use with your own credentials. You have the right to access your educational records under FERPA.

**Q: Will sessions expire?**
A: Yes, typically after 7-14 days. You'll need to extract fresh cookies.

**Q: Can I use this for multiple users?**
A: This library is designed for single-user, personal use. Multi-user scenarios require additional security considerations.

**Q: What if Schoology changes their API?**
A: We'll update the library as needed. This is a community effort - contributions welcome!

## Related Projects

- [trunchbull](https://github.com/leftathome/trunchbull) - Academic dashboard using this library
- [powerschool-go](https://github.com/leftathome/powerschool-go) - Similar library for PowerSchool

## License

MIT License - see [LICENSE](LICENSE) for details.

## Disclaimer

This software is provided "as is" without warranty. The authors are not responsible for any misuse or violations of terms of service. Use responsibly and at your own risk.

This is an unofficial library and is not affiliated with or endorsed by Schoology/PowerSchool.

## Support

- [GitHub Issues](https://github.com/leftathome/schoology-go/issues) - Bug reports and feature requests
- [Discussions](https://github.com/leftathome/schoology-go/discussions) - Questions and community support

---

Made with care for parents and students who want to understand their academic data.
