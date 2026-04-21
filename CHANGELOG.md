# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-04-21

First public release. The library reads a parent/student Schoology
account via its existing browser session and exposes the endpoints
that a parent account actually surfaces on the districts we've
tested against. See [docs/OBSERVED_BEHAVIOR.md](docs/OBSERVED_BEHAVIOR.md)
for the empirical endpoint reference with capture dates.

### Added

#### Authentication

- `Client` with `NewClient(host, WithSession(...))` for scripted use
  with pre-extracted credentials.
- `schoology.WithHTTPClient` / `schoology.WithTimeout` client options.
- `auth` subpackage (`github.com/leftathome/schoology-go/auth`):
  - `auth.Login(ctx, host, opts...)` — opens a visible Chromium
    window via `go-rod`, waits for the user to complete whatever
    login flow their district uses (SSO, MFA, native password),
    then extracts the session and returns `*Credentials`.
  - `auth.LoginWithPassword(ctx, host, user, pass, opts...)` —
    headless scripted login against Schoology's native Drupal 7
    `s_user_login_form`. Returns `ErrSSORequired` when the tenant
    redirects off-host (caller should fall back to `Login`) or
    `ErrBadCredentials` when the server re-renders the login form
    (fast-fails within ~1.5s rather than waiting the full timeout).
  - `auth.SaveCredentials` / `auth.LoadCredentials` for JSON
    round-trip (0600 permissions) so the browser only has to run
    once per ~14-day cookie lifetime.
  - `auth.NewClient(creds, opts...)` convenience wrapper over
    `schoology.NewClient` + `WithSession`.
  - `auth.WithHeadless` / `auth.WithBrowserBinary` / `auth.WithTimeout`
    options.
- `ValidateSession` / `UpdateSession` / `GetSessionInfo` /
  `SessionTimeRemaining` on `*Client`.

#### Resources

- `Course` / `Child` types.
- `Client.GetCourses` — `/iapi2/site-navigation/courses`.
- `Client.GetChildren` — `/iapi/parent/info`.
- `Client.GetCoursesForChild(ctx, childUID)` — serializes a
  `viewAs(childUID)` + `GetCourses` pair under an internal mutex
  so concurrent calls for different children never interleave.
- `Assignment` + `AssignmentStatus` types.
- `Client.GetOverdueSubmissions(ctx, childUID)` — parses the
  overdue/upcoming HTML blob embedded in
  `/iapi/parent/overdue_submissions/{uid}`.
- `Post` type.
- `Client.GetFeed(ctx, childUID)` — parses the HTML-in-JSON
  envelope returned by `/home/feed?children={uid}`, including
  attachments.
- `MessageThread` type.
- `Client.GetInbox(ctx)` — parses `/messages/inbox` (empty-state
  verified live; filled-row selectors inferred from Drupal 7
  conventions and fixture-tested offline).
- `Attachment` type with `ID` / `Filename` / `URL` / `MimeType`.
- `Client.DownloadAttachment(ctx, url)` — authenticated streaming
  download. Rejects `text/html` responses as auth-expired so
  callers don't receive the login page as "the file".

#### Errors

- `*Error` with `Code` / `Op` / `Message` / `Err` / `StatusCode` /
  `RetryAfter` fields. `Error.Is` compares on `Code` only, matching
  Go's sentinel convention (e.g. `fs.ErrNotExist`).
- Error-code taxonomy: `ErrCodeAuth`, `ErrCodeNotFound`,
  `ErrCodeRateLimit`, `ErrCodeServer`, `ErrCodeClient`,
  `ErrCodePermission`, `ErrCodeNetwork`, `ErrCodeParse`.
- Sentinel errors: `ErrNotAuthenticated`, `ErrSessionExpired`,
  `ErrInvalidSession`, `ErrNotFound`, `ErrRateLimited`,
  `ErrInvalidResponse`, `ErrPermissionDenied`.
- Typed helpers: `IsAuthError`, `IsNotFoundError`,
  `IsRateLimitError`, `IsPermissionError`, `IsParseError`,
  `IsRetryable`.
- `ParseErrors` slice type — HTML parsers return `(items,
  ParseErrors, error)`; the per-item slice lets callers see which
  rows failed without losing the ones that parsed.

#### Tooling

- `internal/htmlfetch` package for Schoology-aware HTML parsing:
  validates `Content-Type`, detects the login-redirect page, and
  returns a `*goquery.Document`.
- `hack/redact` — CLI for scrubbing real-world HTML captures into
  committable fixtures via a gitignored `hack/redact.config.json`
  substitution map.

#### Tests + docs

- Offline fixture-backed test suite (89%+ coverage on the root
  package and 94%+ on `internal/htmlfetch`).
- Integration test suite under `-tags=integration` — smoke check
  against a live session, not regression coverage (see
  [AGENTS.md](AGENTS.md) "Testing policy").
- [README.md](README.md) — Getting Started for library consumers
  (both `auth.Login` and pre-extracted-credential paths).
- [CONTRIBUTING.md](CONTRIBUTING.md) — local-dev workflow for
  future maintainers, including how to get your own session into
  `.env.integration` for testing.
- [docs/OBSERVED_BEHAVIOR.md](docs/OBSERVED_BEHAVIOR.md) — dated
  empirical reference: endpoints that work, endpoints that don't,
  HTML structures the parsers depend on.
- [docs/SESSION_EXTRACTION.md](docs/SESSION_EXTRACTION.md) —
  re-written to reflect reality (CSRF token / key / UID live in
  `Drupal.settings.s_common`, not in cookies).

### Known limitations

- **Grades**: `/parent/grading_report/{uid}`, `/user/{uid}/grades`,
  `/course/{courseNid}/gradebook`, and `/course/{sectionNid}/grades_summary`
  do not render grade content for parent accounts on the districts
  we tested — the data appears to route through a separate SIS.
  Parser issues remain open (`schoology-go-84g`, `-gor`) for a
  contributor on a district that does expose grades.
- **Calendar**: `/parent/calendar` is a JS-rendered shell; the event
  grid is populated by an XHR we haven't identified. Parser issue
  `schoology-go-set` remains open.
- **Attendance**: never probed.
- **Message thread read** (individual thread HTML): capture account's
  inbox was empty at v0.1.0 time, so we have no real thread fixture.
  `GetInbox` returns thread list only; thread-page parsing is
  deferred.

### Known follow-ups (tracked in `bd`)

- `schoology-go-8ei` (P2): feed attachment filenames can include a
  duplicate suffix when the HTML carries both the display name and
  the file-system-safe variant as sibling text.
- `schoology-go-u5r` (P3): the feed parser emits a `ParseErrors`
  entry for system-generated notification posts (no author / course
  link). Should either be branched on variant type or downgraded
  to a warning.

[0.1.0]: https://github.com/leftathome/schoology-go/releases/tag/v0.1.0
