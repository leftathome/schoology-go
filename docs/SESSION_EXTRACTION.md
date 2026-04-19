# Session Cookie Extraction Guide

This guide explains how to extract session cookies from Schoology to use with the `schoology-go` library.

## Why Session Cookies?

Instead of requiring district-level API approval (which can take weeks or be denied), this library uses your existing parent/student login session. This is the same approach used by browser extensions and mobile apps.

**Important**: Session cookies are just as sensitive as your password. Keep them secure and never share them.

## Prerequisites

- An active Schoology parent or student account
- A web browser (Chrome, Firefox, or Safari)
- Access to browser Developer Tools

## Method 1: Chrome DevTools (Recommended)

### Step 1: Log Into Schoology

1. Open Chrome and navigate to your school's Schoology site
2. Log in with your username and password
3. Verify you can see your dashboard

### Step 2: Open Developer Tools

1. Press `F12` (or `Ctrl+Shift+I` on Windows/Linux, `Cmd+Option+I` on Mac)
2. Click on the **Application** tab
   - If you don't see it, click the `>>` button to find it

### Step 3: Navigate to Cookies

1. In the left sidebar, expand **Cookies**
2. Click on your Schoology domain (e.g., `https://yourschool.schoology.com`)

### Step 4: Extract Required Values

You need to find and copy these four values:

#### 1. Session ID (SESS*)
- Look for a cookie starting with `SESS` followed by a long string
- Example: `SESSabcd1234...`
- Copy the **Value** (the entire cookie value)
- This is your `SCHOOLOGY_SESS_ID`

#### 2. CSRF Token
- Find the cookie named `CSRF_TOKEN`
- Copy the **Value**
- This is your `SCHOOLOGY_CSRF_TOKEN`

#### 3. CSRF Key
- Find the cookie named `CSRF_KEY`
- Copy the **Value**
- This is your `SCHOOLOGY_CSRF_KEY`

#### 4. User ID
- Find the cookie named `UID` or look at the session data
- Copy the **Value**
- This is your `SCHOOLOGY_UID`

### Step 5: Store Securely

Create a file `.env.integration` (do NOT commit this to git):

```bash
SCHOOLOGY_HOST=yourschool.schoology.com
SCHOOLOGY_SESS_ID=your-session-id-value
SCHOOLOGY_CSRF_TOKEN=your-csrf-token-value
SCHOOLOGY_CSRF_KEY=your-csrf-key-value
SCHOOLOGY_UID=your-uid-value
```

## Method 2: Firefox Developer Tools

### Steps

1. Open Firefox and log into Schoology
2. Press `F12` to open Developer Tools
3. Click the **Storage** tab
4. Expand **Cookies** in the left sidebar
5. Click on your Schoology domain
6. Find and copy the same four values as described above

## Method 3: Safari Web Inspector

### Enable Developer Tools (First Time Only)

1. Open Safari Preferences
2. Go to **Advanced** tab
3. Check "Show Develop menu in menu bar"

### Extract Cookies

1. Log into Schoology in Safari
2. From the menu bar, choose **Develop** → **Show Web Inspector**
3. Click the **Storage** tab
4. Click **Cookies** → your Schoology domain
5. Find and copy the same four values

## Using 1Password (Recommended for Security)

Instead of storing credentials in plain text, you can store them in 1Password:

### Step 1: Create 1Password Item

1. Open 1Password
2. Create a new **Login** item
3. Title it "Schoology Session Credentials"
4. Add custom fields for:
   - `host` (text)
   - `sess_id` (password)
   - `csrf_token` (password)
   - `csrf_key` (password)
   - `uid` (text)

### Step 2: Create .env.integration with 1Password References

```bash
# .env.integration
SCHOOLOGY_HOST=op://Private/Schoology Session Credentials/host
SCHOOLOGY_SESS_ID=op://Private/Schoology Session Credentials/sess_id
SCHOOLOGY_CSRF_TOKEN=op://Private/Schoology Session Credentials/csrf_token
SCHOOLOGY_CSRF_KEY=op://Private/Schoology Session Credentials/csrf_key
SCHOOLOGY_UID=op://Private/Schoology Session Credentials/uid
```

### Step 3: Run with 1Password CLI

```bash
# Load values from 1Password and run your application
op run --env-file=.env.integration -- go run examples/basic/main.go

# Or for tests
op run --env-file=.env.integration -- go test -tags=integration -v
```

## Session Expiration

Sessions typically last **7-14 days** depending on school configuration.

### Signs Your Session Has Expired

- Getting "unauthorized" or "session expired" errors
- Can't fetch data even though credentials seem correct
- Response from Schoology is a login page redirect

### What to Do When Session Expires

1. Log into Schoology in your browser again
2. Extract fresh cookies following the steps above
3. Update your `.env.integration` file or 1Password item
4. Try again

## Security Best Practices

### DO:
- ✅ Store session cookies securely (1Password recommended)
- ✅ Add `.env.integration` to `.gitignore`
- ✅ Delete old sessions when no longer needed
- ✅ Use separate cookies for testing vs production
- ✅ Log out of Schoology when done to invalidate the session

### DON'T:
- ❌ Commit session cookies to version control
- ❌ Share cookies with others
- ❌ Store cookies in plain text files
- ❌ Reuse expired cookies
- ❌ Use cookies from public/shared computers

## Troubleshooting

### Problem: Can't find SESS* cookie

**Solution**: Make sure you're fully logged in. Try refreshing the page and checking again.

### Problem: Cookie values seem incomplete

**Solution**: Some browsers truncate long values. Double-click the value field to see the full value, or right-click → Copy.

### Problem: Getting 401 Unauthorized errors

**Solution**: Your session has likely expired. Extract fresh cookies.

### Problem: No UID cookie found

**Solution**: The UID might be embedded in the URL or in other page data. Check the browser's Network tab for API requests and look for a `uid` parameter.

### Problem: CSRF tokens missing

**Solution**: Some schools may not use CSRF protection. Try setting these to empty strings and see if it works. (This is less secure but may be the only option.)

## FAQ

**Q: How often do I need to refresh cookies?**
A: Typically every 7-14 days. The library will tell you when the session has expired.

**Q: Can I automate this?**
A: Yes! Version 0.2.0 will include automated login with username/password using headless browser automation.

**Q: Is this legal?**
A: Yes, you're using your own credentials to access data you already have permission to access. This is no different than using a browser.

**Q: Will this work for parent accounts?**
A: Yes! Parent accounts work the same way as student accounts.

**Q: What if my school uses single sign-on (SSO)?**
A: This should still work. Log in through your SSO provider, then extract cookies from the Schoology page once you're logged in.

## Next Steps

Once you have your session cookies:

1. Test them with the basic example:
   ```bash
   export SCHOOLOGY_HOST=yourschool.schoology.com
   export SCHOOLOGY_SESS_ID=your-session-id
   export SCHOOLOGY_CSRF_TOKEN=your-token
   export SCHOOLOGY_CSRF_KEY=your-key
   export SCHOOLOGY_UID=your-uid

   go run examples/basic/main.go
   ```

2. Or run integration tests:
   ```bash
   go test -tags=integration -v
   ```

3. Start building with the library!

## Need Help?

- Check the [main README](../README.md) for usage examples
- Open an issue on GitHub
- See [CONTRIBUTING.md](../CONTRIBUTING.md) for how to get help

---

**Remember**: Treat session cookies like passwords. Keep them secure!
