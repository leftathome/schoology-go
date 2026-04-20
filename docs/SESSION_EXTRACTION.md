# Establishing a Session

`schoology-go` makes authenticated requests using an already-established
Schoology session — four values (`SessID`, `CSRFToken`, `CSRFKey`,
`UID`) passed to `schoology.WithSession`. This doc covers the two ways
to get those values.

## Why not username/password?

Most Schoology tenants front authentication with district SSO
(PowerSchool SAML, Google Workspace, Okta, Clever, etc.). A scripted
username/password login works on a small minority of tenants and
breaks everywhere else. Session-cookie reuse works universally,
because it sidesteps authentication entirely: the user logs in once
through whatever mechanism their district uses, and the library
reuses that session.

## Option A (recommended): the `auth` subpackage

`github.com/leftathome/schoology-go/auth` drives a browser for you.

### Interactive login (SSO, MFA, anything)

Opens a visible Chromium window, lets the user finish whatever login
flow their district uses, then extracts the session and returns.

```go
import (
    "context"
    "github.com/leftathome/schoology-go"
    "github.com/leftathome/schoology-go/auth"
)

creds, err := auth.Login(context.Background(), "yourschool.schoology.com")
if err != nil { /* handle */ }

client, err := auth.NewClient(creds)
// or: schoology.NewClient("yourschool.schoology.com",
//        schoology.WithSession(creds.SessID, creds.CSRFToken, creds.CSRFKey, creds.UID))
```

### Scripted password login (tenants with native Schoology auth)

```go
creds, err := auth.LoginWithPassword(ctx, "yourschool.schoology.com",
    "parent@example.com", "correct horse battery staple")
```

Returns `ErrSSORequired` if the server redirects off-host (district
uses SSO — fall back to `auth.Login`), or `ErrBadCredentials` if the
`/login` form re-renders (wrong username or password).

### Reusing sessions across runs

```go
const path = ".schoology-session.json" // add to .gitignore

creds, err := auth.LoadCredentials(path)
if err != nil {
    creds, err = auth.Login(ctx, host)
    if err != nil { /* handle */ }
    _ = auth.SaveCredentials(path, creds)
}

client, err := auth.NewClient(creds)
```

The file is written with `0600` permissions (user-only read/write).

### First-run note

On first invocation, `rod` downloads a Chromium build (~140 MB) into
its cache directory. Subsequent runs reuse it. Override with
`auth.WithBrowserBinary("/usr/bin/google-chrome")` if you already have
a Chromium-compatible browser installed.

## Option B: manual extraction

If you're setting up credentials for an unattended job (e.g. CI), you
may prefer to extract values by hand and store them in a secret
manager. Here's what each value is and where to find it.

> **The old version of this doc said three of these were cookies.**
> **They are not.** Only `SessID` is a cookie. `CSRFToken`, `CSRFKey`,
> and `UID` live in the page's embedded `Drupal.settings` JSON.

### 1. `SessID` — the `SESS…` cookie (HttpOnly)

1. Log into `https://yourschool.schoology.com` in Chrome/Firefox.
2. DevTools → **Application** (Chrome) or **Storage** (Firefox) tab →
   **Cookies** → your Schoology domain.
3. Find the cookie whose name starts with `SESS` followed by 32 hex
   characters (e.g. `SESS0f3c278d8cbcca12eab60706abd3d4f3`). The name
   is `SESS` + md5(host), so it varies by tenant.
4. Copy the **Value**. That's `SCHOOLOGY_SESS_ID`.

The cookie is `HttpOnly`, so it won't appear in `document.cookie` —
you need the DevTools cookie inspector, not the Console.

### 2, 3, 4. `CSRFToken`, `CSRFKey`, `UID` — from `Drupal.settings`

1. Still on a logged-in Schoology page, open DevTools → **Console**.
2. Paste and run:

   ```js
   JSON.stringify({
     csrf_token: Drupal.settings.s_common.csrf_token,
     csrf_key:   Drupal.settings.s_common.csrf_key,
     uid:        String(Drupal.settings.s_common.user.uid),
   }, null, 2)
   ```

3. Copy each string into the matching env var:
   - `csrf_token` → `SCHOOLOGY_CSRF_TOKEN`
   - `csrf_key`   → `SCHOOLOGY_CSRF_KEY`
   - `uid`        → `SCHOOLOGY_UID`

### Storing the values

```bash
# .env.integration (add to .gitignore!)
SCHOOLOGY_HOST=yourschool.schoology.com
SCHOOLOGY_SESS_ID=...
SCHOOLOGY_CSRF_TOKEN=...
SCHOOLOGY_CSRF_KEY=...
SCHOOLOGY_UID=...
```

With 1Password CLI:

```bash
# store under op://Private/Schoology Session/{host,sess_id,...}
op run --env-file=.env.integration -- go test -tags=integration -v
```

## Session lifetime

Drupal session cookies on Schoology last ~14 days by default (the
library sets `ExpiresAt` to 14 days from session creation). When
`ValidateSession` starts returning auth errors, re-run `auth.Login`
(or re-extract manually) to get fresh values.

## Security

- Treat session cookies like passwords. A valid `SESS…` cookie grants
  full API access to your Schoology account until it expires.
- Do not commit `.env.integration` or `.schoology-session.json`.
- Prefer a secret manager (1Password CLI, `pass`, HashiCorp Vault,
  etc.) over plain files when automating.
- Log out of Schoology in your browser when you're done developing —
  it invalidates the session server-side.

## Troubleshooting

**"no SESS\* cookie on authenticated page"** — the browser didn't
actually complete login (redirect was interrupted, or you closed the
window early). Retry.

**`auth.Login` just hangs** — you probably haven't reached `/home`
or `/parent/home` yet. Make sure the browser window has completed
its SSO flow. If you're still stuck, widen the timeout with
`auth.WithTimeout(15 * time.Minute)`.

**`ErrSSORequired` from `LoginWithPassword`** — your tenant redirected
off-host for SSO. Use `auth.Login` instead.

**`ErrBadCredentials` when you're sure the password is right** —
check whether your account requires MFA. If it does, scripted password
login can't work; use `auth.Login` and complete the MFA step manually.
