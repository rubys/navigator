# Authentication Failure Investigation

**Date Started**: 2025-10-15
**Issue**: Intermittent 401 authentication failures for valid credentials, succeeding on retry

## Problem Summary

Users with valid HTTP Basic Auth credentials are experiencing intermittent 401 Unauthorized responses, which succeed when the browser retries the same request with the same credentials. The pattern shows:
- First request: 401 auth-failure
- Retry (same credentials): 200 success

## Affected Users

Primary affected users identified from logs:
- **cranford**: 381 failures out of 1,479 requests (26% failure rate)
- **roanoke**: 5 failures out of 73 requests (7% failure rate)

Unaffected users (for comparison):
- **milano**: 913 requests, 0 failures
- **Boston**: 429 requests, 0 failures
- **adelaide**: 614 requests, 0 failures
- **test**: Created during investigation, worked correctly

## Timeline Deployed 2025-10-15

**Deployment time noted**: 2025-10-15 at 22:06 UTC (when mutex fix was deployed)

### Initial Hypothesis: Concurrency Bug in go-htpasswd (DISPROVEN)

**Theory**: The `go-htpasswd` library's `Match()` function has concurrency issues when called from multiple goroutines simultaneously.

**Fix Attempted**: Added `sync.RWMutex` to protect concurrent access to `htpasswd.File.Match()` in `internal/auth/auth.go`:
```go
type BasicAuth struct {
    File    *htpasswd.File
    Realm   string
    Exclude []string
    mu      sync.RWMutex // Protects concurrent access to File
}

func (a *BasicAuth) CheckAuth(r *http.Request) bool {
    if a != nil {
        a.mu.RLock()
        defer a.mu.RUnlock()
    }
    // ... rest of auth check
}
```

**Result**: Fix deployed but problem persisted. This ruled out simple concurrency issues.

### Second Hypothesis: Fly-Replay Authorization Header Loss (CURRENT THEORY)

**Discovery**: Analysis of logs revealed the 401 failures occur specifically during cross-region fly-replay scenarios:

**Pattern from logs** (cranford user, region: ewr):
```
21:34:00.581 → 307 fly-replay (from source region)
21:34:00.618 → 401 auth-failure (at target region ewr)
21:34:00.712 → 307 fly-replay (browser retry)
21:34:00.825 → 200 proxy (success)
```

**Theory**: When Navigator sends a fly-replay JSON response to Fly.io's proxy, the `transform.set_headers` field only includes `X-Navigator-Retry: true` but doesn't explicitly preserve the `Authorization` header. Fly.io's proxy may not forward the Authorization header to the target region on the first attempt.

**Fix Implemented**: Modified `internal/server/fly_replay.go` to explicitly include the Authorization header in the transform:

```go
buildTransformHeaders := func() []map[string]string {
    headers := []map[string]string{
        {"name": "X-Navigator-Retry", "value": "true"},
    }

    // Explicitly preserve Authorization header if present
    if authHeader := r.Header.Get("Authorization"); authHeader != "" {
        headers = append(headers, map[string]string{
            "name":  "Authorization",
            "value": authHeader,
        })
    }

    return headers
}
```

Applied to all three fly-replay types:
1. Region-based: `region: "ewr,any"`
2. App-based: `app: "app-name"`
3. Machine-based: `app: "app-name", prefer_instance: "machine-id"`

**Status**: Fix deployed 2025-10-15, awaiting verification with affected users.

### User Observation: Authentication Enforcement Timing

**Important Context** (from user):
> "replay requests that were reverse proxied to a tenant worked with authentication, it is when I asked for navigator to enforce authentication that this started showing up"

This suggests the issue may not be with fly-replay header forwarding, but rather with **when** Navigator enforces authentication vs when the Rails tenant enforces it.

**Implication**: The problem might be related to:
1. Navigator's authentication check happening before fly-replay
2. Different auth behavior between direct requests and fly-replayed requests
3. Timing of when auth is loaded/checked during the request lifecycle

## Technical Environment

### Htpasswd File Location
- **Configured path**: `/data/db/htpasswd`
- **Last modified**: Oct 14, 2025 21:45 UTC (confirmed up-to-date)
- **Update mechanism**: `script/update_htpasswd.rb` via server hooks (start/resume)

### Go-htpasswd Library
- **Version**: v1.2.4 (latest as of investigation)
- **Behavior**: Caches file contents in memory on load (does NOT read on every request)
- **Thread safety**: Not explicitly documented; assumed unsafe without external locking

### Affected Regions
- **cranford**: Region `ewr` (Newark)
- **roanoke**: Region unknown (likely also cross-region)
- Requests from other regions (e.g., `mia`, `iad`) get fly-replayed to target regions

### Password Hash Analysis
All password hashes examined use APR1 format with identical structure:
- **cranford**: `$apr1$fQldg.On$c56VYMGXdaM2KWCFu20Th.` (46 chars total)
- **roanoke**: `$apr1$vIJe.JkC$D9WbYjSEfg9wqbgbzGyy6/` (45 chars total)
- **Boston**: `$apr1$DPeEcJmB$koL1vqV5i1MXZ82Q1ZWnl.` (44 chars total)
- **test**: `$apr1$u.icEyyi$vFwcCKw46YVcZ0RStvNdV1` (42 chars total)

No corruption or encoding issues detected.

## Diagnostic Logging Implemented

**Current logging** (as of this document):

Location: `internal/auth/auth.go`, `CheckAuth()` function

Logs every authentication attempt with:
- `username`: Username from HTTP Basic Auth
- `matched`: Result from `htpasswd.File.Match()` (true/false)
- `password_hash`: MD5 hash of password (non-reversible, for correlation)
- `request_id`: X-Request-Id or X-Fly-Request-Id header
- `uri`: Request URI path
- `method`: HTTP method

**Critical modification**: Function **always returns true** to allow all requests through, enabling correlation analysis between `matched` result and eventual HTTP status code.

### Analysis Plan

1. **Collect logs** with all auth checks logged
2. **Correlate** `matched: false` with HTTP status codes:
   - If `matched: false` + `status: 200` → Rails tenant accepted the request (Navigator wrong)
   - If `matched: false` + `status: 401` → Rails tenant also rejected (both agree)
   - If `matched: true` + `status: 401` → Navigator passed but Rails rejected (mismatch)
3. **Compare password_hash** across requests:
   - Same hash = same password typed
   - Different hash = user typed different passwords
4. **Track request_id** to correlate auth check with final HTTP response

## Files Modified

### Navigator Changes

1. **internal/auth/auth.go**
   - Added `sync.RWMutex` for concurrency protection
   - Added comprehensive logging of all auth checks
   - Modified to always return `true` for correlation analysis
   - Commit: (mutex fix) and (logging implementation)

2. **internal/server/fly_replay.go**
   - Added Authorization header preservation in fly-replay transform
   - Applies to region, app, and machine-based fly-replay
   - Commit: d4075e22c571d5fa4b6dfad36b87321cd775c49a

3. **lib/htpasswd_updater.rb** (Showcase)
   - Added atomic file writes (temp file + rename)
   - Defense-in-depth to prevent partial reads
   - Commit: 6918fa87

## Configuration Details

### Navigator Configuration
```yaml
auth:
  enabled: true
  realm: "Showcase"
  htpasswd: "/data/db/htpasswd"
  public_paths:
    - "/showcase/assets/"
    - "/showcase/cable"
    # ... additional public paths
  auth_patterns:
    - pattern: "^/showcase/2025/(?:cranford|roanoke|...)/public/"
      action: "off"
```

### Hooks Configuration
```yaml
hooks:
  server:
    start:
      - command: "/rails/script/update_htpasswd.rb"
        timeout: "30s"
    resume:
      - command: "/rails/script/update_htpasswd.rb"
        timeout: "30s"
```

## Next Steps

### Immediate Actions (Logging Analysis)

1. **Deploy logging version** to production
2. **Wait for affected users** (cranford, roanoke) to become active
3. **Collect logs** with auth check entries
4. **Analyze correlation**:
   ```bash
   # Extract auth checks and match with status codes
   grep '"auth check"' logs.json | jq -r '[.request_id, .username, .matched, .password_hash] | @tsv' > auth_checks.tsv
   grep -E '"status":(200|401)' logs.json | jq -r '[.request_id, .status, .remote_user] | @tsv' > responses.tsv
   # Join on request_id to correlate
   ```

### Hypotheses to Test

1. **Fly-Replay Header Loss**: Does preserving Authorization header in fly-replay fix the issue?
   - **Test**: Compare failure rates before/after fly-replay fix deployment
   - **Expected**: If this is the cause, failures should drop to 0%

2. **Navigator vs Rails Auth Mismatch**: Does Navigator's auth check disagree with Rails?
   - **Test**: Compare `matched` result with HTTP status code
   - **Expected**: If Navigator is wrong, we'll see `matched: false` + `status: 200`

3. **Password Variation**: Are users typing different passwords on retry?
   - **Test**: Compare `password_hash` for same user across requests
   - **Expected**: Same hash = same password, different hash = user error

4. **Go-htpasswd Bug**: Does `Match()` return incorrect results non-deterministically?
   - **Test**: Same password_hash should always produce same `matched` result
   - **Expected**: If library is buggy, same hash will show both true and false

### Alternative Theories to Explore

1. **Request Lifecycle Timing**:
   - Auth check happens before fly-replay decision
   - Different auth behavior for direct vs fly-replayed requests
   - Race condition between auth loading and request handling

2. **Htpasswd File Reload**:
   - File reloaded mid-request after SIGHUP
   - Old vs new password hash mismatch
   - Check: correlate 401s with config reload timestamps

3. **Client-Side Issues**:
   - Browser sending credentials incorrectly
   - Base64 encoding issues
   - Check: same password_hash across all requests from same user

## Known Working Configurations

- **Direct requests** (no fly-replay): Authentication works correctly
- **Reverse proxy fallback** (>1MB requests): Authentication works correctly
- **Users in same region**: No authentication failures observed

## References

- **Go-htpasswd**: https://github.com/tg123/go-htpasswd (v1.2.4)
- **Fly-Replay API**: https://fly.io/docs/reference/fly-replay/
- **Navigator Documentation**: https://rubys.github.io/navigator/

## Investigation Notes

### Browser User Agents
- **cranford**: Chrome 141 on macOS (1443 req) + Chrome 110 on Windows (36 req)
- **roanoke**: Chrome 140 on Windows (71 req) + Chrome 110 on Windows (2 req)
- **Boston**: Chrome 141 on Windows (406 req) + Chrome 110 on Windows (23 req)
- **milano**: Edge 141 on Windows (903 req)

No clear browser correlation identified.

### Redirect Loop Observed
During investigation, test user experienced infinite redirect loop:
```
307 fly-replay → 401 auth-failure → 307 fly-replay → 401 auth-failure → ...
```

This pattern suggests authentication is failing consistently after fly-replay, supporting the header loss hypothesis. However, user notes that "replay requests that were reverse proxied to a tenant worked with authentication," which contradicts this theory.

### Key Insight: Authentication Enforcement Point

The user's observation that the issue appeared "when I asked for navigator to enforce authentication" suggests:
- Previously: Rails tenant handled all authentication
- Now: Navigator enforces authentication BEFORE proxying to tenant
- Issue: Navigator's auth check may interfere with fly-replay mechanism

**Question to investigate**: Does Navigator check auth before or after deciding to fly-replay?

Current code flow (from `internal/server/handler.go`):
1. Check if path should be excluded from auth
2. If auth required → check credentials
3. If auth fails → return 401
4. If auth passes → continue to routing logic (including fly-replay)

This means authentication is checked BEFORE fly-replay decision, so the Authorization header should already be validated by the time fly-replay happens. This suggests the fly-replay header loss theory may be incorrect.

## Summary

The investigation has identified two potential root causes:

1. **Fly-Replay Header Loss** (FIX DEPLOYED): Authorization header not preserved during fly-replay
2. **Authentication Timing** (NEEDS INVESTIGATION): Navigator enforcing auth before fly-replay may interact poorly with cross-region requests

The comprehensive logging now deployed will allow correlation of authentication results with HTTP status codes to determine which hypothesis is correct.

**Status**: Awaiting affected user activity to collect diagnostic data.
