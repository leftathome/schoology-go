# schoology-go

> A Go client library for reading your own Schoology data

[![Go Reference](https://pkg.go.dev/badge/github.com/leftathome/schoology-go.svg)](https://pkg.go.dev/github.com/leftathome/schoology-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/leftathome/schoology-go)](https://goreportcard.com/report/github.com/leftathome/schoology-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`schoology-go` is a Go client library that reads data from Schoology
(a K-12 LMS) using your existing browser session. It's aimed at
parents and students who want programmatic access to their own
academic data — dashboards, automations, notifications — without
going through Schoology's district-approval API process.

## Status

**v0.1.0**, feature-complete against the endpoints a parent account
actually exposes on the districts we've tested. See
[docs/OBSERVED_BEHAVIOR.md](docs/OBSERVED_BEHAVIOR.md) for the
endpoint-by-endpoint reality check.

What works:

- List the courses / sections of every child on the account
- Fetch overdue + upcoming assignments per child
- Fetch faculty feed posts (with attachments) per child
- Fetch messages inbox + download attachments

What's explicitly **out of scope for v0.1.0** on the districts we
tested (they route the data through a separate SIS):

- Per-assignment grade detail
- Cross-section grade reports
- Calendar / events
- Attendance

The grade and calendar parser stubs are kept open as bead issues so a
volunteer on a district that does expose this data via Schoology can
contribute a capture + parser.

## Install

```bash
go get github.com/leftathome/schoology-go
```

Requires Go 1.23+. If you plan to use `auth.Login` (browser-driven
session establishment) you'll also need a Chromium-compatible browser
available — `go-rod` auto-downloads one on first use (~140 MB into
its cache dir) unless you point it at an existing binary.

## Getting started

There are two paths to a logged-in client. Pick based on how you plan
to run the caller.

### Path A: `auth.Login` (interactive, one-shot)

Best for first-time setup, CLI tools, and schools with SSO. Opens a
visible Chromium window, you complete whatever login flow your
district uses (SSO, MFA, the native password form, …), and the
library captures the session when the browser lands on the home page.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/leftathome/schoology-go"
    "github.com/leftathome/schoology-go/auth"
)

func main() {
    ctx := context.Background()

    // Opens a browser, waits for you to log in.
    creds, err := auth.Login(ctx, "yourschool.schoology.com")
    if err != nil {
        log.Fatal(err)
    }
    // Optional: reuse the session on the next run.
    _ = auth.SaveCredentials(".schoology-session.json", creds)

    client, err := auth.NewClient(creds)
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
}
```

Between runs, `auth.LoadCredentials(".schoology-session.json")`
returns the saved value; the session lasts ~14 days, after which
you'll get an auth error and need to `auth.Login` again.

For districts that use Schoology's native password auth (no SSO
redirect), `auth.LoginWithPassword(ctx, host, user, pass)` skips the
browser window — headless by default. It returns `ErrSSORequired` if
the tenant redirects off-host mid-login, so you can fall back to
`auth.Login`.

### Path B: pre-extracted credentials (scripted / unattended)

Suitable when you've already extracted the four session values from
a browser (see [docs/SESSION_EXTRACTION.md](docs/SESSION_EXTRACTION.md))
and want to run without spawning a Chromium process.

```go
client, err := schoology.NewClient(
    "yourschool.schoology.com",
    schoology.WithSession(sessID, csrfToken, csrfKey, uid),
)
```

This is what the integration tests use.

## Usage

```go
ctx := context.Background()

// Children + their courses.
children, err := client.GetChildren(ctx)
for _, ch := range children {
    courses, err := client.GetCoursesForChild(ctx, ch.UID)
    for _, c := range courses {
        fmt.Printf("  %s / %s\n", c.CourseTitle, c.SectionTitle)
    }
}

// Overdue + upcoming assignments for a child.
assignments, parseErrs, err := client.GetOverdueSubmissions(ctx, childUID)
for _, a := range assignments {
    fmt.Printf("%s %q (%s) due %s\n",
        a.Status, a.Title, a.CourseTitle, a.DueAt.Format(time.RFC3339))
}
// parseErrs is a nil-or-non-nil ParseErrors — individual rows that
// failed to parse. The operation still "succeeded" even when it's
// non-nil; non-nil `err` only means a hard HTTP / auth failure.

// Faculty feed + attachments.
posts, parseErrs, err := client.GetFeed(ctx, childUID)
for _, p := range posts {
    fmt.Printf("%s posted %q\n", p.AuthorName, p.Body)
    for _, att := range p.Attachments {
        rc, err := client.DownloadAttachment(ctx, att.URL)
        // ... copy rc to disk, etc. ...
        rc.Close()
    }
}

// Messages inbox (read-only).
threads, _, err := client.GetInbox(ctx)
for _, t := range threads {
    fmt.Printf("[%s] %s from %s\n",
        t.LastActivity.Format(time.RFC3339), t.Subject, t.SenderName)
}
```

All methods take `context.Context` and respect cancellation /
deadlines.

## Error handling

The library returns `*schoology.Error` from every transport-layer
failure. The `Code` field categorizes the error; use `errors.Is` or
the typed helpers to branch on the cause:

```go
_, err := client.GetCourses(ctx)
switch {
case errors.Is(err, schoology.ErrSessionExpired) || schoology.IsAuthError(err):
    // Session is bad. Re-run auth.Login to refresh.
case schoology.IsRateLimitError(err):
    // Back off and retry.
case schoology.IsRetryable(err):
    // Transient; retry with exponential backoff.
}
```

HTML parsers return a third value, `ParseErrors`, collecting per-item
failures (a single malformed row doesn't fail the whole page). The
`err` value is only non-nil for hard failures.

## Security & privacy

- Session cookies are as sensitive as your password. Never share
  them, never commit them, keep them in a secret manager.
- `.schoology-session.json` is written 0600 by `SaveCredentials` —
  add the path to your `.gitignore`.
- The library talks only to your tenant's Schoology host; no
  analytics or phone-home.
- FERPA applies to the data you fetch. Use only with credentials
  you're authorized to use.

## Documentation

- [docs/SESSION_EXTRACTION.md](docs/SESSION_EXTRACTION.md) — how to
  establish a session, via `auth.Login` or manual cookie extraction
- [docs/OBSERVED_BEHAVIOR.md](docs/OBSERVED_BEHAVIOR.md) — empirical
  endpoint + HTML-shape reference with observation dates
- [CONTRIBUTING.md](CONTRIBUTING.md) — how to hack on the library
  locally, including how to use your own session for development
- [AGENTS.md](AGENTS.md) — testing / PII / issue-tracking policy for
  contributors (human or agent)
- [Go API reference](https://pkg.go.dev/github.com/leftathome/schoology-go)

## Related

- [trunchbull](https://github.com/leftathome/trunchbull) — parent
  dashboard built on this library
- [powerschool-go](https://github.com/leftathome/powerschool-go) —
  sibling library for PowerSchool (often where grades live)

## License

MIT — see [LICENSE](LICENSE).

## Disclaimer

Unofficial; not affiliated with or endorsed by Schoology or
PowerSchool. The Schoology API is reverse-engineered from the
browser UI and may change — see
[docs/OBSERVED_BEHAVIOR.md](docs/OBSERVED_BEHAVIOR.md) for our
last-verified dates.
