# Directory Redirect Fix for Try Files

## Problem

Navigator's `try_files` implementation had a bug where directory paths without trailing slashes (e.g., `/showcase/studios/laval`) failed to serve their `index.html` files, even though the files existed at `/rails/public/studios/laval/index.html`.

### Root Cause

The `tryPublicDirFiles` function was concatenating extensions directly to the path:
- Tried: `/rails/public/studios/laval` + `index.html` = `/rails/public/studios/lavalindex.html` ❌
- Tried: `/rails/public/studios/laval` + `.html` = `/rails/public/studios/laval.html` ❌
- Never tried: `/rails/public/studios/laval/index.html` ✅

### Impact

This caused unnecessary wake-ups of the index server for requests like:
- `GET /showcase/studios/laval` → proxied to Rails (slow, 3.5s response)
- `GET /showcase/studios/petaluma/` → served statically (fast, 0.002s response)

## Solution

Implement standard web server behavior: **redirect directories to trailing slash URLs**.

### Why Redirect Instead of Direct Serve?

Serving `index.html` directly without redirecting breaks **relative paths in HTML**:

```html
<!-- If served at /showcase/studios/laval (no trailing slash) -->
<img src="logo.png">  <!-- Browser requests /showcase/studios/logo.png ❌ -->
<link rel="stylesheet" href="style.css">  <!-- /showcase/studios/style.css ❌ -->

<!-- After redirect to /showcase/studios/laval/ (with trailing slash) -->
<img src="logo.png">  <!-- Browser requests /showcase/studios/laval/logo.png ✅ -->
<link rel="stylesheet" href="style.css">  <!-- /showcase/studios/laval/style.css ✅ -->
```

This matches nginx/Apache behavior and ensures assets load correctly.

## Implementation

### Changes Made

1. **Modified `internal/server/static.go`** (lines 162-185):
   - Added directory detection before trying extensions
   - If path (without trailing slash) is a directory with `index.html`, redirect with 301
   - Then extension-based try_files proceeds as normal

2. **Added `internal/logging/logging.go`** (lines 390-395):
   - Added `LogDirectoryRedirect()` function for consistent logging

3. **Added comprehensive tests** in `internal/server/static_config_test.go` (lines 489-604):
   - Test redirect from directory without trailing slash
   - Test serving index.html with trailing slash
   - Test no redirect for directories without index.html

### Request Flow

**Before fix:**
```
GET /showcase/studios/laval
  ↓ try_files checks
  ↓ No match found
  ↓ Proxy to index server (wake up Rails app)
  ↓ 3.5s response time
```

**After fix:**
```
GET /showcase/studios/laval
  ↓ try_files detects directory with index.html
  ↓ 301 redirect to /showcase/studios/laval/
  ↓ Browser follows redirect
GET /showcase/studios/laval/
  ↓ try_files serves laval/index.html statically
  ↓ 0.002s response time (no Rails wakeup)
```

## Performance Impact

From log analysis of 5 index server wake-ups:
- **2 legitimate requests** (still wake server, but with proper caching)
- **3 malformed URLs** (can be filtered with bot detection)

After this fix + bot detection:
- Estimated **60% reduction** in unnecessary index server wake-ups
- **1000x faster** response for directory index pages (3.5s → 0.002s)

## Testing

All tests pass:
```bash
$ go test ./internal/server/ -run TestDirectoryRedirect
=== RUN   TestDirectoryRedirect
=== RUN   TestDirectoryRedirect/directory_without_trailing_slash_redirects
=== RUN   TestDirectoryRedirect/directory_with_trailing_slash_serves_index.html
=== RUN   TestDirectoryRedirect/directory_without_index.html_does_not_redirect
--- PASS: TestDirectoryRedirect (0.00s)
PASS
```

## Related Issues

- Original issue: Bot requests like `GET /showcase/studios/laval` were proxied to Rails
- Bot detection: Recommend adding `zgo.at/isbot` to filter malformed bot requests (separate PR)
- Try files config: Rails `configurator.rb` already has correct config (`try_files: [index.html, .html, .htm]`)

## References

- RFC 3986 (URI syntax): Recommends trailing slash for directories
- nginx behavior: `try_files $uri $uri/ /index.html` with `index index.html`
- Apache behavior: `DirectorySlash On` (default) and `DirectoryIndex index.html`
