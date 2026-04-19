# schoology-go — v0.1.0 Implementation Plan

> This file supersedes the pre-discovery plan (the previous version was
> written before we ran live traffic against Schoology; most of its
> assumptions turned out to be wrong). Keep this file in sync with the
> open `bd` issues as work progresses.

## Where we are

**Pre-v0.1.0 cleanup landed in commit `3269496`:**

- Session cookie name formula fixed (`SESS` + md5(bare host), verified
  live).
- `Course` type + `GetCourses()` match the real
  `/iapi2/site-navigation/courses` envelope.
- `Child` type + `GetChildren()` added against `/iapi/parent/info`.
- All speculative types and `assignments.go` / `grades.go` removed —
  they referenced paths that return 404 on real Schoology.

**Endpoints that do NOT exist (as probed on a parent account):**

- `/iapi*/sections/{id}/assignments`, `/iapi*/sections/{id}/grades`,
  `/iapi*/gradebook/{id}`, `/iapi*/courses/{id}` — all 404 or 403.
- `/iapi*/messages*`, `/iapi*/notifications`, `/iapi*/events/*` — all
  404 or 403.

**Pages that DO exist but are HTML-only:**

- `/parent/grading_report/{child_uid}` — HTML
- `/course/{courseNid}/gradebook` — HTML ("Course Profile")
- `/messages/inbox` — HTML
- `/parent/calendar` — HTML
- `/iapi/parent/overdue_submissions/{child_uid}` — JSON envelope with
  a pre-rendered HTML blob inside `body.html`

`Drupal.settings` on these pages does **not** carry structured data
for the resources — only config stubs
(e.g. `s_messaging.min_query_length`). The data is rendered directly
into the page DOM by the Drupal 7 backend.

## v0.1.0 goal

> Make `schoology-go` useful enough for a parent dashboard (trunchbull).
> That means grades, assignments, and messages retrievable as typed
> values — across every accessible child — even if the underlying
> transport is HTML scraping.

## Design principles

- **Multi-child first.** Parent accounts routinely have more than one
  child (our discovery pass saw two). Never assume a UID. Every
  per-child method takes the UID explicitly as a parameter.
- **Don't lean on Drupal session state.** Schoology's web UI is modal:
  `session.view_child` dictates what the courses endpoint returns. The
  library MUST NOT make callers reason about that. We hide the
  view-switch POST inside per-child methods, or prefer URL-parameterized
  endpoints that already encode the child UID
  (`/parent/grading_report/{child_uid}`,
  `/iapi/parent/overdue_submissions/{child_uid}`). Many of the parent
  endpoints we found already do this — good.
- **Parallel-safe.** If a caller runs `GetGradesForChild` for two
  different children concurrently, the library must not race on a
  shared `view_child` state. Likely implication: either serialize
  view-switching operations, or keep each logical child "session" on
  its own HTTP client with its own cookie jar.
- **`GetCourses(ctx)` as it stands is ambiguous for a parent.** It
  returns whatever the server's current `view_child` says. We keep
  it for now (low-level, maps cleanly to the endpoint) but the
  intended callable from user code is
  `GetCoursesForChild(ctx, childUID) ([]*Course, error)`.

## Approach: HTML parsing via `goquery`

- **Library**: `github.com/PuerkitoBio/goquery`. Confirmed.
- **Architecture**: one file per resource (`grades.go`, `assignments.go`,
  `messages.go`, `attachments.go`, …). Each file exposes typed getter
  methods plus internal parse helpers. Shared HTML-loading lives in a
  new `internal/htmlfetch/` package to keep `goquery` usage out of the
  resource files where possible.
- **Fixture strategy**: each parser ships a table-driven test backed
  by a redacted HTML fixture under `internal/testdata/html/*.html`.
  Split:
  - `hack/redact.go` (committed) — the redactor itself. Pure logic,
    no real-name data.
  - `hack/redact.config.json` (**gitignored**) — maps real names,
    UIDs, emails to stable placeholders (`Student Alpha`, `UID-1001`,
    `example.com`). Stays local. Gets regenerated each time someone
    runs the capture flow on their own account.
  - `internal/testdata/html/*.html` (committed) — only the output of
    the redactor, not the raw captures. `.playwright-captures/` stays
    gitignored so the unredacted traffic never leaks.
- **Error model**: on selector miss we return the existing `*Error`
  type with `Code: ErrCodeParse` (a new ErrorCode we'll add), plus
  `Op` set to the resource/selector path. Consistency with the rest
  of the library.
- **Graceful degradation**: parse-as-much-as-possible. A single
  malformed row shouldn't fail the whole page — emit the partial
  result + collect a slice of per-item parse errors on the return
  value so callers can see what we skipped.

## Scope (ordered by value for trunchbull)

1. **Grades** — per-child summary AND per-section per-assignment
   detail. Two parsers:
   - `/parent/grading_report/{child_uid}` → cross-section summary
   - `/course/{courseNid}/gradebook` → per-assignment detail
2. **Assignments** — upcoming + overdue, per-child. Source:
   `/iapi/parent/overdue_submissions/{child_uid}` body.html, plus
   calendar HTML for upcoming.
3. **Messages** (read-only) — inbox list (subject, sender, timestamp,
   read flag), thread read, **with attachment metadata**.
   `/messages/inbox` HTML. No send for v0.1.0.
4. **Attachments** — first-class. Library exposes an
   `Attachment` type with `{ID, Filename, URL, MimeType, Size}` and a
   `DownloadAttachment(ctx, url) (io.ReadCloser, error)` that reuses
   the authenticated HTTP client. Attachments surface everywhere they
   exist: message threads, course post feeds, assignment descriptions,
   materials. Schoology attachment URL pattern:
   `/attachment/{id}/source/{hash}.ext?checkad=1`.
5. **Course feed / announcements** — faculty posts in the home feed.
   `/home/feed?children={child_uid}` returned 200 but shape is not
   yet probed; a 10-minute probe session before this item starts to
   confirm JSON vs HTML. This is where attachment-bearing flyers and
   course updates show up.
6. **Calendar / events** — combined feed. *Stretch.*
7. **Attendance** — if accessible to a parent account; probe first.
   *Stretch.*

Items 1–5 are the v0.1.0 bar. 6–7 are stretch; if they drop they
become v0.1.1.

## Cross-cutting work

- **Child-switch discovery is a hard prerequisite** for anything that
  needs `view_child` scoping (courses; possibly more we haven't mapped
  yet). Small Playwright session to capture the POST that the child
  menu fires, then implement `viewAs(ctx, childUID) error` as an
  internal helper (not part of the public surface). Public per-child
  methods (`GetCoursesForChild`, etc.) call it transparently.
- **Concurrency model for per-child calls**: easiest correct answer
  is a mutex around viewAs→operation sequences. Faster answer is one
  cookie-jar per child. Decide when we see the POST shape — if it's
  stateless (a query param), this problem evaporates.
- **Parser versioning**. Each parser records the fixture/selector
  version it validated against. Surface it on the returned values
  (e.g. a `Meta` field, or on a wrapping result type) so stale
  fixtures show up loudly in integration output.
- **Context + cancellation** already propagates through `do()`; keep
  that through the new parse pipeline too.

## Work breakdown (→ bd issues)

Each item below lives as a `bd` issue under epic `schoology-go-v01`.
The epic and issues will be created when this plan is accepted.

| # | bd | Issue | Blocked by |
|---|----|---|---|
| 1 | — | `hack/redact.go` + schema for `hack/redact.config.json` (gitignored config). CLI: `go run ./hack/redact -in <capture> -out <fixture>`. | — |
| 2 | — | Capture representative HTML fixtures for grades / assignments / messages / feed from a live session. Run through the redactor. Commit only redacted output. | 1 |
| 3 | — | Add `goquery` dependency + `internal/htmlfetch` helper that returns a `*goquery.Document` for a path, with content-type validation and redirect-to-login detection. | — |
| 4 | — | Add `ErrCodeParse` + per-item partial-error slice. | — |
| 5 | — | Parse `/parent/grading_report/{child_uid}` → `[]*Grade` (cross-section summary per child). | 2, 3, 4 |
| 6 | — | Parse `/course/{courseNid}/gradebook` → `[]*Grade` (per-assignment detail). | 2, 3, 4 |
| 7 | — | Parse `/iapi/parent/overdue_submissions/{child_uid}` body.html + upcoming source → `[]*Assignment`. | 2, 3, 4 |
| 8 | — | Attachment type + `DownloadAttachment(ctx, url) (io.ReadCloser, error)`. | 3 |
| 9 | — | Parse `/messages/inbox` → `[]*MessageThread` + thread read, with attachment extraction. | 2, 3, 4, 8 |
| 10 | — | Probe `/home/feed?children={child_uid}` shape (JSON vs HTML); implement `GetFeed(ctx, childUID)` and attach-bearing post parsing. | 2, 3, 4, 8 |
| 11 | — | Live endpoint-capture session for the child-switch POST; implement internal `viewAs(ctx, childUID)` + `GetCoursesForChild(ctx, childUID)`. Load-bearing prerequisite for anything that relies on view_child scoping. | — |
| 12 | — | Parse `/parent/calendar` → `[]*Event`. *Stretch.* | 2, 3, 4 |
| 13 | — | Integration-test pass against a real session — closes `schoology-go-n1g`. | 5, 6, 7, 9, 10 |
| 14 | — | Raise coverage to >80% on stable surfaces — closes `schoology-go-3b7`. | — |

## Decisions (locked)

- `goquery` for parsing.
- Redactor script in VCS; its config (real-name → placeholder map)
  stays local and gitignored.
- Per-assignment grade detail is in scope, not stretch.
- Messages: read + attachment metadata + download. No send.
- Parse errors use existing `*Error` with a new `ErrCodeParse`.

## Known unknowns (non-blocking)

- `/home/feed?children={child_uid}` shape. Needs a 10-minute probe
  before item 10 starts.
- Child-switch POST details (item 11) — single capture session against
  a live account.
- Whether attendance data is reachable at all for a parent account.
