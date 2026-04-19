# schoology-go — Current Status

## Summary

`schoology-go` is a Go client for Schoology's internal `iapi/iapi2`
endpoints via session cookies. Live endpoint discovery against a real
Schoology instance on 2026-04-19 found that most resource types
(assignments, grades, messages) have **no clean JSON endpoint** for a
parent account — those pages are server-rendered HTML with no
supplementary XHR. The library's scope has been narrowed accordingly.

**Status**: Pre-v0.1.0 — code-complete for the endpoints we verified
to return structured JSON. Shipping v0.1.0 requires HTML parsing for
the remaining resources, tracked as a follow-up.

## Live-verified endpoints

| Endpoint | Exposed as | Notes |
|---|---|---|
| `GET /iapi2/site-navigation/courses` | `GetCourses()` | Envelope: `{data: {courses: [...]}}`. Scoped to the session's currently-viewed enrollment (for parents, `session.view_child`). |
| `GET /iapi/parent/info` | `GetChildren()` | Envelope: `{response_code, body: {session, children}}`. We flatten the children map into a sorted slice. |

Cookie: `SESS` + md5(bare-hostname). Formula verified against a live
session (real cookie name matched the md5 of the bare hostname, with
no leading dot or URL scheme). CSRF headers confirmed as `X-CSRF-Token`
and `X-CSRF-Key`.

## Known gaps (deferred to v0.1.0 via HTML parsing)

- Assignments list / detail
- Grades (per-section, per-child, per-assignment)
- Messages (inbox, thread read)
- Calendar / events
- Attendance

Probing found that the parent-facing pages (`/parent/grading_report`,
`/messages/inbox`, `/course/{nid}/gradebook`) return HTML with no
supplementary JSON endpoints. The Drupal `Drupal.settings` blob on
those pages also does not contain structured data for these resources
— only config stubs (e.g. `s_messaging.min_query_length`). The v0.1.0
plan is to add `goquery`-based parsers per resource, backed by
redacted HTML fixtures.

## Outstanding beads (`bd list`)

- `schoology-go-9xy` — Cookie-name verification. **Closed**: formula
  fixed and tested against real cookie name.
- `schoology-go-401` — iapi endpoint discovery. **Mostly resolved**
  via live capture. Remaining: find parent-scoped JSON for anything
  beyond courses/children (none found in this pass).
- `schoology-go-n1g` — Run integration tests end-to-end.
- `schoology-go-3b7` — Raise unit-test coverage above 80%.
- `schoology-go-oou` — chromedp vs Playwright MCP decision (likely
  reframed now that Playwright MCP has proven itself for endpoint
  discovery; runtime automation may still prefer chromedp).

## Testing

```bash
# Unit tests (mocked)
go test ./...

# Integration tests (requires real session cookies)
op run --env-file=.env.integration -- go test -tags=integration -v
```
