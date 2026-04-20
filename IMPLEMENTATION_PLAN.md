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

**Pages that DO render useful HTML for a parent:**

- `/messages/inbox` — HTML (we've only seen the empty-state so far;
  filled inbox structure still TBD)
- `/iapi/parent/overdue_submissions/{child_uid}` — JSON envelope with
  a pre-rendered HTML blob inside `body.html` (OVERDUE + UPCOMING
  sections; empty body when no submissions)
- `/home/feed?children={child_uid}` — JSON envelope `{css, js, output}`
  where `output` is the feed HTML (faculty posts, attachments,
  announcements). Earlier status said "shape not yet probed"; as of
  capture day 2026-04-19 the shape is **JSON-wrapped HTML**, not
  stand-alone HTML.

**Pages that EXIST on the server but render NO data for a parent
account on the district we captured against** (a large US K-12
district on a shared `*.schoology.com` tenant):

- `/parent/grading_report/{child_uid}` — redirects to `/parent/home`
- `/user/{child_uid}/grades` — 200 but renders the child's profile
  page (Info / Portfolios tabs only), no grade content
- `/course/{courseNid}/gradebook` — 200 but renders "Course Profile"
  shell (materials sidebar only)
- `/course/{sectionNid}/grades_summary` — 200 but also just the
  course profile shell
- `/parent/calendar` — 200 but the event grid is JS-rendered and
  contains zero static event data
- `/grades`, `/course/{nid}` (without a sub-path) — 403

For this deployment Schoology surfaces only the feed,
overdue/upcoming submissions, messages, and course materials —
grades and gradebooks appear to live in a separate district SIS.
v0.1.0 scope is narrowed to match (see below).

`Drupal.settings` on these pages does **not** carry structured data
for the resources — only config stubs
(e.g. `s_messaging.min_query_length`). The data is rendered directly
into the page DOM by the Drupal 7 backend.

## v0.1.0 goal

> Make `schoology-go` useful enough for a parent dashboard (trunchbull).
> That means assignments (overdue + upcoming), messages, and faculty
> feed posts with attachments retrievable as typed values — across
> every accessible child — via HTML scraping of the parent-facing
> pages that *do* expose data on the district we tested against.
> Grades are out for v0.1.0 (that district doesn't surface them in
> Schoology); re-evaluate on other districts once we have a
> volunteer tester.

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
  by a fixture under `internal/testdata/html/`.
  - `hack/redact.go` (committed) — the redactor itself. Still useful
    for point substitution (names, UIDs).
  - `hack/redact.config.json` (**gitignored**) — maps real names,
    UIDs, emails to placeholders. Stays local.
  - `internal/testdata/html/*` (committed) — the live captures
    contain too much free-form teacher communication for a pure
    string-replace redactor to scrub safely, so the committed
    fixtures are **synthesized** from observed structure: same
    selectors, same attribute shapes, placeholder content. Raw
    captures stay under `.playwright-captures/` (gitignored) so the
    unredacted traffic never leaks, and so parsers can be
    spot-checked against real data locally when a regression
    is suspected.
- **Error model**: on selector miss we return the existing `*Error`
  type with `Code: ErrCodeParse` (a new ErrorCode we'll add), plus
  `Op` set to the resource/selector path. Consistency with the rest
  of the library.
- **Graceful degradation**: parse-as-much-as-possible. A single
  malformed row shouldn't fail the whole page — emit the partial
  result + collect a slice of per-item parse errors on the return
  value so callers can see what we skipped.

## Scope (ordered by value for trunchbull)

1. **Assignments** — upcoming + overdue, per-child. Source:
   `/iapi/parent/overdue_submissions/{child_uid}` body.html
   (OVERDUE + UPCOMING sections). This is the highest-value
   endpoint given what's actually exposed to parents on the
   district we tested against.
2. **Course feed / announcements** — faculty posts in the home feed.
   `/home/feed?children={child_uid}` → JSON envelope `{css, js,
   output}` where `output` is the feed HTML. Attachment-bearing
   flyers and course updates surface here.
3. **Attachments** — first-class. Library exposes an
   `Attachment` type with `{ID, Filename, URL, MimeType, Size}` and a
   `DownloadAttachment(ctx, url) (io.ReadCloser, error)` that reuses
   the authenticated HTTP client. Attachments surface in message
   threads and feed posts (and presumably in materials/assignment
   descriptions once we probe those pages). Schoology attachment
   URL pattern: `/attachment/{id}/source/{hash}.ext?checkad=1`.
4. **Messages** (read-only) — inbox list (subject, sender, timestamp,
   read flag), thread read, **with attachment metadata**.
   `/messages/inbox` HTML. Currently only empty-inbox structure is
   captured; filled-inbox selectors need a second capture pass when
   there's mail to look at. No send for v0.1.0.

### Out for v0.1.0 on the tested district (see "Endpoints..." above)

- **Grades** (both cross-section summary and per-assignment detail):
  the district doesn't expose grade data to Schoology parent
  accounts. Keep the parser issues open so a volunteer on a
  different district can rehydrate them, but do not block v0.1.0
  ship on them.
- **Calendar / events**: the `/parent/calendar` page is a
  JS-rendered shell; the event grid is populated client-side from
  an XHR we haven't identified yet. Defer.
- **Attendance**: never probed; defer.

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
