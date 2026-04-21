# Contributing to schoology-go

This doc walks a new contributor from cloning the repo to shipping a
change, with special attention to how to iterate on the library using
your own Schoology account (because Schoology has no public sandbox).

If you haven't already, skim [AGENTS.md](AGENTS.md) — it has the
project's testing, PII, and issue-tracking policies; they apply to
humans too.

## Prerequisites

- Go 1.23+
- Git
- A Schoology account (parent or student) you're authorized to use
- A Chromium-compatible browser on your machine, if you want to iterate on
  the `auth` subpackage (`go-rod` will download one automatically on
  first use if you don't already have one)
- Optional: [1Password CLI](https://developer.1password.com/docs/cli/)
  for keeping session credentials out of plain-text files

## Setup

```bash
git clone https://github.com/YOUR_USERNAME/schoology-go.git
cd schoology-go
go mod download
go test ./...        # should be green with no network
```

The offline suite must pass on a freshly-cloned repo without any
credentials.

## Running tests

There are two suites:

### Offline suite (default)

```bash
go test -race -cover ./...
```

Runs without a network or credentials. Every parser has a
table-driven test backed by a synthetic fixture under
`internal/testdata/html/`. If you break a parser, a fixture-based
test will break offline — this is by design (see
[AGENTS.md](AGENTS.md) "Testing policy").

Target coverage for the root package and `internal/htmlfetch` is
>80%. The `auth` subpackage sits around 32% because the browser-
driven paths can't be unit-tested without Chromium; we test the
non-browser surface (credentials round-trip, URL matching, option
parsing) and the browser paths get exercised by hand.

### Integration suite (requires your session)

```bash
# Populate .env.integration (see "Getting your own session" below).
go test -tags=integration -v
```

These are **smoke checks**, not regression coverage. They prove "the
library talks to real Schoology today"; they'll start failing on
their own as your session cookie expires (~14 days) or as school-year
transitions change what data exists on your account. Do not treat a
passing integration run as evidence that nothing regressed — rely on
the offline suite for that.

## Getting your own session for local development

You need the session to run the integration suite and to sanity-check
new parsers against real HTML. Two paths — both write to an
`.env.integration` file (gitignored) that the integration test reads.

### Easy path: use the library's own `auth.Login`

```bash
cat > extract-session.go <<'EOF'
//go:build ignore
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/leftathome/schoology-go/auth"
)

func main() {
    if len(os.Args) != 2 {
        log.Fatal("usage: go run extract-session.go <host>")
    }
    creds, err := auth.Login(context.Background(), os.Args[1])
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("SCHOOLOGY_HOST=%s\n", creds.Host)
    fmt.Printf("SCHOOLOGY_SESS_ID=%s\n", creds.SessID)
    fmt.Printf("SCHOOLOGY_CSRF_TOKEN=%s\n", creds.CSRFToken)
    fmt.Printf("SCHOOLOGY_CSRF_KEY=%s\n", creds.CSRFKey)
    fmt.Printf("SCHOOLOGY_UID=%s\n", creds.UID)
}
EOF
go run extract-session.go yourschool.schoology.com > .env.integration
rm extract-session.go
```

A Chromium window opens, you log in however your district requires,
and the four env vars land in `.env.integration`. Delete the file
when you're done.

### Manual path

Follow the "Option B" steps in
[docs/SESSION_EXTRACTION.md](docs/SESSION_EXTRACTION.md) and put the
values in `.env.integration` yourself. Use this when you can't run a
headless browser (e.g. CI-ish environments, air-gapped boxes).

### Running the integration tests

```bash
# Bash / zsh
set -a; . ./.env.integration; set +a
go test -tags=integration -v -run TestIntegration

# With 1Password CLI (values stored as references in .env.integration)
op run --env-file=.env.integration -- go test -tags=integration -v
```

Delete `.env.integration` when you're done. The session is valid for
~14 days, but leaving the file around is a footgun.

## The hack cycle for a new parser

New parsers show up under `bd` — check `bd ready` for open parser
issues. The pattern we use (and that you should follow):

1. **Capture live HTML** from your own account via a browser with
   DevTools open. Save the raw response under `.playwright-captures/`
   (gitignored).
2. **Synthesize a fixture** under `internal/testdata/html/` that
   mirrors the structure of the real capture but has no PII — no
   real names, UIDs, district-specific section codes, emails, etc.
   `hack/redact` is available for pure string substitution but
   isn't sufficient for free-form text like teacher announcements;
   in practice, hand-crafted fixtures are cleaner.
3. **Write a table-driven parser test** against the fixture in
   `<resource>_test.go`.
4. **Write the parser** in `<resource>.go`. Follow the existing
   `(items, ParseErrors, error)` return pattern for HTML-based
   resources, and `(items, error)` for clean JSON.
5. **Run the offline suite**: `go test -race ./...`.
6. **Smoke-check against live data**: add an `integration_test.go`
   subtest, populate `.env.integration`, run the integration suite,
   eyeball the logged output. File follow-up `bd` issues for any
   discrepancies — don't fix them in the same commit.
7. **Update `docs/OBSERVED_BEHAVIOR.md`** with a new `Observed:`
   stanza if you probed a new endpoint or saw new HTML structure.
8. **PR** with a description that explains WHY, not just WHAT.

## PII discipline

The repo has had PII scrubs before. Fresh rules (per
[AGENTS.md](AGENTS.md)):

- No real child/teacher/parent names in any committed file.
- No real UIDs (use `1xxxxxxx`-style placeholders).
- No district-specific formatting (no `S2 7(A) 1013`-style section
  codes — use `Section A` / `Section 01`).
- No session-cookie values, CSRF tokens, or UIDs.
- `.playwright-captures/`, `.claude/`, `.env.integration`, and
  `.schoology-session.json` are all gitignored. Don't add exceptions.

Before each commit that touches tests or fixtures, run a quick
rescan:

```bash
grep -rnIE 'yourhost|realname|realschool|realdistrict' \
    --exclude-dir=.git --exclude-dir=.claude \
    --exclude-dir=.playwright-captures --exclude-dir=.playwright-mcp \
    --exclude-dir=.beads --exclude-dir=.dolt .
```

## Issue tracking

All task tracking goes through `bd` (beads). See
[AGENTS.md](AGENTS.md) for the full workflow; the short version:

```bash
bd ready              # find ready-to-work issues
bd show <id>          # details
bd update <id> --claim
# ... do the work ...
bd close <id>
```

Do **not** create Markdown TODO lists or track tasks in commit
messages — `bd` is the system of record.

## Adding an integration-test endpoint

If you add a new `Get*` method, also add a subtest to
`integration_test.go`:

- Guard it with `if ch.UID == 0 { t.Skip("no children") }` or
  similar when it's per-child.
- Use `t.Parallel()` unless the method takes the viewChild mutex
  (i.e. it's a `ForChild` variant that calls `viewAs`).
- Log counts and shapes, not PII. `t.Logf("child %d: %d items",
  ch.UID, len(items))` is fine; logging the full item bodies is
  not.
- Skip cleanly when env vars are missing.

## Commit + PR process

- Format: `go fmt ./...`
- Vet: `go vet ./...`
- Tests: `go test -race ./...`
- Commit message: one-line summary, blank line, paragraph describing
  the change. Reference the `bd` issue id in the commit body, not
  the title. We do not squash, so each commit should pass CI on its
  own.
- For changes to the public API surface, also update `README.md`
  (Getting Started) and the relevant `docs/*.md`.

## Security

- NEVER log or print session credentials (even in debug output).
- NEVER commit credentials to the repository.
- Error messages in library code must not echo CSRF tokens, session
  IDs, or raw response bodies.

If you find a security issue, please email the maintainer rather
than opening a public issue.

## License

By contributing, you agree your contributions will be MIT-licensed.
