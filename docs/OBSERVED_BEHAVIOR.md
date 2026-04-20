# Observed Schoology Behavior

Empirical reference for `schoology-go`. Everything below comes from
live Playwright captures against a single real parent account; cite
this when asking "where did that assumption come from?". Anything not
listed here is speculative.

When Schoology changes, this file will drift. Keep it up to date:
each observation stanza has an **Observed:** line with the date and
version markers that were true at capture time.

## Version markers visible at capture time

None of these are a real Schoology platform version (Schoology does
not expose one). They are the proxies we have.

- **Footer copyright** (on every page): `PowerSchool © 2026` (Schoology
  was acquired by PowerSchool in 2019).
- **Bundle hash** (observed on `/parent/home`):
  `common-8ccf52ba0124a9c52142.js?69dea5c61dfc1b42`.
- **Drupal version**: Drupal 7 (inferred — all Drupal.settings paths,
  session cookie naming, form IDs match D7 conventions, not D8/D9/D10).
- **Browser driving the captures**: Chromium 147.
- **Aptrinsic (Gainsight) SDK on the page**: `sv=0.58.7`.

**Observed:** 2026-04-19 and 2026-04-20.

## Session + authentication

### Session cookie

- Name: `SESS` + md5(bare host). For `example.schoology.com` the cookie
  is `SESS0f3c278d8cbcca12eab60706abd3d4f3`. Formula verified against
  live traffic.
- Attributes: `HttpOnly`, `Secure`, `Path=/`. NOT visible via
  `document.cookie`. Visible in DevTools → Application → Cookies.
- Lifetime: ~14 days on our tenant. (`schoology-go` defaults
  `Session.ExpiresAt` to +14d on `WithSession`.)

### CSRF token / key

- Live in `Drupal.settings.s_common.csrf_token` and
  `Drupal.settings.s_common.csrf_key`, embedded in the page's
  `<script>` tag on any authenticated render.
- Sent as request headers `X-CSRF-Token` and `X-CSRF-Key` on XHRs.
- Rotate: we observed token/key change after a `/parent/switch_child/*`
  navigation. The library's `WithSession` values remain valid for
  CSRF-checked endpoints across such rotations in our captures, but
  if that breaks long-term the fix is re-extracting after a known
  rotation event.
- NOT cookies. (The old SESSION_EXTRACTION.md said they were; that
  was wrong.)

### User identity

- `Drupal.settings.s_common.user.uid` on any authenticated page
  holds the viewer's numeric UID as an integer.
- This is the correct source for the `uid` argument to
  `schoology.WithSession` — not a cookie.

### /login form (native Schoology password auth)

- URL: `POST /login`, form id `s-user-login-form`.
- Fields:
  - `mail` (text) — email or username
  - `pass` (password)
  - `school` (text) + `school_nid` (hidden) — school picker; on a
    tenant subdomain the school is implied and `school_nid` can be
    empty
  - `remember-school` (checkbox)
  - `form_build_id` (hidden, per-request Drupal CSRF)
  - `form_id` (hidden, always `s_user_login_form`)
  - `op` (submit, value `Log in`)
- Success: redirects to `/home` or `/parent/home` (200, text/html).
- Failure: re-renders `/login` with an error message.
- Off-host redirect on submit == tenant is SSO-backed; scripted
  login cannot complete.

**Observed:** 2026-04-20. The `auth` subpackage drives this form.

## Parent-facing endpoints

### Verified WORKING

| Path | Method | Shape | Notes |
|---|---|---|---|
| `/iapi/parent/info` | GET | `{response_code, body:{session:{view_mode, view_child}, children:{...}}}` | Source of truth for child UIDs and the current view-child. |
| `/iapi2/site-navigation/courses` | GET | `{data:{courses:[{nid, courseNid, courseTitle, sectionTitle, ...}]}}` | Scoped to `session.view_child`. Switch view first via `/parent/switch_child/{uid}` to get a different child's courses. |
| `/iapi/parent/overdue_submissions/{childUID}` | GET | `{response_code, body:{html: "..."}}` | `body.html` is an escaped HTML blob with `<div class='upcoming-events'>` containing `<h3 class='submissions-title'>OVERDUE</h3>` and/or `UPCOMING` sections. Blob is `""` when there's nothing to show. |
| `/home/feed?children={childUID}` | GET | `{css:{}, js:{}, output: "..."}` | `output` is feed HTML; see "HTML structures" below. Historical plan docs called the shape unknown; as of the 2026-04-19 capture it is JSON-wrapped HTML. |
| `/messages/inbox` | GET | text/html | Server-rendered. Empty state contains a `<tr class="empty-state">` row with "There are no available inbox messages." Filled-row structure is inferred, not verified (our capture account's inbox was empty). |
| `/messages/sent` | GET | text/html | Same structure as `/inbox`. |
| `/parent/switch_child/{childUID}` | GET | 302→200 (text/html) | A plain link in the top-right menu, NOT a POST. Side effect: flips `session.view_child` server-side. Redirects to `/parent/home`. Body can be discarded. Concurrent callers need to serialize (`Client.viewChildMu`). |
| `/attachment/{id}/source/{hash}.{ext}?checkad=1` | GET | the file | `{id}` is numeric, `{hash}` is 32-char hex (probably md5 of content), `{ext}` is the file extension. `?checkad=1` is required (omit it and Schoology returns 403). Auth via the session cookie. |
| `/login` | POST | 200/302 | Drupal 7 s_user_login_form. See "Authentication" above. |
| `/user/logout` | GET | 302→login | Terminates the server-side session. |

**Observed:** 2026-04-19 (2026-04-20 for `/parent/switch_child`).

### Verified NOT working (on the district we tested against)

These URLs exist on the server and return 200 but render a shell with
no data for parent accounts. Our best guess is that the district
(a large US K-12 system on a shared tenant) routes grade and calendar
data to a separate SIS (PowerSchool) and only uses Schoology for
course materials / messages / feed. Other districts likely differ.

| Path | What it actually renders |
|---|---|
| `/parent/grading_report/{uid}` | 302 → `/parent/home`. Endpoint absent. |
| `/user/{uid}/grades` | The child's profile page (Info / Portfolios tabs). No grade content. |
| `/course/{courseNid}/gradebook` | "Course Profile" page shell (materials sidebar only). No grade content. |
| `/course/{sectionNid}/grades_summary` | Same "Course Profile" shell. |
| `/parent/calendar` | 200 text/html shell. Event grid is JS-rendered from an XHR we did NOT identify; static HTML contains zero event data. |
| `/grades`, `/course/{nid}` (no sub-path) | 403. |

**Observed:** 2026-04-19.

**Implication for library scope:** grades and calendar parsers are
not shippable from the tenants we've touched. The parser issues stay
open in `bd` for a volunteer on a district that exposes this data.

### District identifier (aptrinsic telemetry)

The aptrinsic (Gainsight) beacon tags the user with
`accountId:"SGY-<districtID>-prod"` and name
`"<District Name>"` in the telemetry payload. This is a useful
fingerprint for "which district is this account on?" but is not a
public API — it's an analytics side channel and should not be
relied on for production behavior.

**Observed:** 2026-04-19.

## HTML structures we parse

All fixture files live under `internal/testdata/html/`. Selectors
here are what the parsers actually rely on; any change to them is
what'll break first when Schoology ships new markup.

### Overdue / upcoming assignments
(`/iapi/parent/overdue_submissions/{uid}` body.html)

```
div.upcoming-events
├── h3.submissions-title                 # "OVERDUE" | "UPCOMING"
└── div.upcoming-list
    ├── div.date-header                  # per-date grouper
    │   └── div.upcoming-date-title      # human-readable date
    └── div.upcoming-event[data-start]   # one per item
        └── div.upcoming-item-content
            ├── span.infotip[aria-label]
            │   └── span.event-title
            │       ├── a[href="/assignment/{id}"]    # title + URL
            │       └── span.readonly-title.event-subtitle  # x2; last is course name
            └── span.infotip.submission-infotip
                └── span.infotip-content              # "This was due on ..."
```

- `data-start` is unix seconds.
- The first `.readonly-title.event-subtitle` is a status label
  ("80 days overdue"); the second (or last) is the course title.
  Be defensive — if there's only one, use it.

### Home feed
(`/home/feed?children={uid}` output)

```
div.item-list
└── ul.s-edge-feed
    └── li[id="edge-assoc-<id>"][timestamp="<unix>"]
        └── div[class*="s-edge-type-..."]           # variant marker
            └── div.edge-item
                ├── div.edge-left
                │   └── div.profile-picture
                │       └── a[href="/user/{uid}"][title="Name"]
                │           └── img[alt="Name"]
                └── div.edge-main-wrapper
                    ├── span.edge-sentence
                    │   ├── div.update-sentence-inner
                    │   │   └── a[href="/course/{nid}"]  # first = postedTo
                    │   └── span.update-body             # post body
                    └── span.edge-main
                        └── div.attachments
                            └── div.attachments-files
                                └── div.attachments-file
                                    ├── span.attachments-file-icon
                                    │   └── span.visually-hidden   # MIME label
                                    └── span.attachments-file-name
                                        └── a[href="/attachment/{id}/source/..."]
                                            [aria-label="filename.ext"]
```

- `timestamp` attr is unix seconds.
- EdgeID in our types is the stripped-`edge-assoc-` part of `li[id]`.
- `.visually-hidden` inside `.attachments-file-icon` carries a
  human-readable MIME label ("Adobe PDF", "Microsoft Word", etc.)
  — NOT an RFC MIME type.

### Messages inbox (EMPTY state only verified)

```
table.messages-table
├── thead > tr (skipped)
└── tbody
    ├── tr.empty-state     # when inbox is empty
    └── tr[.unread]        # when a row is present
        ├── td              # subject
        │   └── a[href="/messages/thread/{id}"]  # subject + URL
        ├── td              # sender
        │   └── a[href="/user/{uid}"]            # name + UID
        └── td              # date ("Tue Mar 31, 2026 at 9:41 am")
```

- Filled-row selectors are **inferred from Drupal 7 conventions**,
  not verified against real data. First contact with a real filled
  inbox will either confirm or force a revision. See
  `internal/testdata/html/messages_inbox_filled.html` for our
  current hand-crafted fixture.

**Observed (empty only):** 2026-04-19.

## Behaviors we explicitly do NOT rely on

These are things that showed up but which we can't depend on without
more evidence across districts / over time:

- **Grade rendering on any URL.** The only districts we've probed
  show no parent-viewable grades. If a user reports that grades
  render for them, the URL they see is the one to scrape — don't
  guess.
- **Calendar event payload shape.** The page is a JS shell; the
  real endpoint is opaque to us as of 2026-04-19.
- **Attendance data.** Never probed.
- **Message thread page structure.** Never probed (inbox was empty).
- **The exact CSRF rotation rules.** We observed a rotation on
  child-switch but did not characterize what other events cause one.

## Refreshing this document

When Schoology ships a visible change:

1. Log in as a parent in a Chromium DevTools session.
2. Re-probe the endpoints listed in "Verified WORKING" with
   `await fetch('/path', {credentials:'include'})` from the
   Console.
3. Compare shape. If any endpoint's JSON keys or HTML selectors
   have shifted, update the relevant parser + fixture + this doc,
   and bump the **Observed:** lines.
4. The `hack/redact` tool is still around for cases where you
   want to commit a lightly-scrubbed real capture — just be careful;
   real Schoology HTML tends to contain more free-form teacher text
   than a string-replace redactor can safely scrub.
