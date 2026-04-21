# schoology-go — Current Status

## Summary

`schoology-go` is a Go library for reading a Schoology parent/student
account via its existing browser session. The v0.1.0 surface is
complete and verified end-to-end against a live parent account; see
[docs/OBSERVED_BEHAVIOR.md](docs/OBSERVED_BEHAVIOR.md) for the
empirical reference.

**Status:** v0.1.0 feature-complete. Awaiting tag.

## What's shipped

| Surface | Source endpoint | Notes |
|---|---|---|
| `auth.Login` / `auth.LoginWithPassword` | rod-driven browser against `/login` | Visible Chromium for SSO / MFA; headless scripted for native-password tenants. |
| `auth.Credentials` + `Save/LoadCredentials` | — | Persist the four session values across runs (0600 JSON). |
| `GetChildren` | `/iapi/parent/info` | Parent-account child list. |
| `GetCourses` | `/iapi2/site-navigation/courses` | Scoped to the session's current `view_child`. |
| `GetCoursesForChild` | same, after `/parent/switch_child/{uid}` | Mutex-serialized switch+read pair. |
| `GetOverdueSubmissions` | `/iapi/parent/overdue_submissions/{uid}` | JSON envelope, HTML in `body.html`. |
| `GetFeed` | `/home/feed?children={uid}` | JSON envelope, HTML in `output`. |
| `GetInbox` | `/messages/inbox` | Empty-state verified live; filled-row selectors inferred + fixture-tested. |
| `DownloadAttachment` | `/attachment/{id}/source/...?checkad=1` | Streamed; rejects text/html as auth-expired. |

## What's explicitly deferred (and why)

The districts we captured against route grade and calendar data to a
separate SIS, so Schoology itself serves parents an empty shell for
those pages. These issues stay open in `bd` awaiting a volunteer on
a district that does expose the data:

- `schoology-go-84g` — `/parent/grading_report/{uid}` cross-section summary
- `schoology-go-gor` — `/course/{courseNid}/gradebook` per-assignment detail
- `schoology-go-set` — `/parent/calendar` event grid

## Testing

```bash
# Offline — must pass on a bare clone, no credentials.
go test -race -cover ./...

# Integration — your own session required; see CONTRIBUTING.md.
go test -tags=integration -v
```

Coverage:

| Package | Coverage |
|---|---|
| `github.com/leftathome/schoology-go` | ~89% |
| `.../internal/htmlfetch` | ~95% |
| `.../hack/redact` | ~78% (main() CLI plumbing uncovered) |
| `.../auth` | ~32% (browser-driven paths require real Chromium) |

See [AGENTS.md](AGENTS.md) "Testing policy" for the rationale behind
these targets.

## Pointers

- [README.md](README.md) — getting-started guide
- [CONTRIBUTING.md](CONTRIBUTING.md) — local dev + how to test with your own session
- [docs/OBSERVED_BEHAVIOR.md](docs/OBSERVED_BEHAVIOR.md) — empirical endpoint reference with dates
- [docs/SESSION_EXTRACTION.md](docs/SESSION_EXTRACTION.md) — how to establish a session (library-driven or manual)
- [AGENTS.md](AGENTS.md) — testing / PII / bd-workflow policy
- [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) — how the v0.1.0 scope was reached
