# Navigator Request Flow

This document provides a detailed explanation of how Navigator processes incoming HTTP requests, from initial receipt through final response.

## Overview

Navigator's request handling follows a carefully orchestrated sequence of decision points, each determining whether to process the request immediately or pass it to the next handler in the chain. The flow is implemented primarily in `internal/server/handler.go`.

## Request Flow Diagram

```
┌─────────────────────────────────────────────────────┐
│ 1. Incoming HTTP Request                            │
│    - Generate Request ID                            │
│    - Create ResponseRecorder for tracking           │
│    - Start idle tracking                            │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 2. Health Check? (/up)                              │
│    → YES: Return 200 OK                             │
│    → NO: Continue                                   │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 3. Authentication Check ⚡ EARLY ENFORCEMENT         │
│    - Check if path is public (auth exclusions)      │
│    - Validate Basic Auth credentials                │
│    → FAILED: Return 401 Unauthorized                │
│    → PASSED: Continue                               │
│                                                      │
│    ⚠️  SECURITY: Authentication happens BEFORE all  │
│        routing decisions to prevent bypass holes    │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 4. Sticky Session Routing (Fly.io)                  │
│    - Check sticky_sessions.enabled                  │
│    - Match against configured paths                 │
│    → MATCHED: Route to specific machine             │
│    → NO MATCH: Continue                             │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 5. Rewrite Rules                                    │
│    - Check server.rewrite_rules                     │
│    - Match path patterns and methods                │
│    → REDIRECT: Return 302 with new location         │
│    → FLY-REPLAY: Route to region/app/machine        │
│    → LAST: Rewrite path internally and continue     │
│    → NO MATCH: Continue                             │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 6. Reverse Proxies (Standalone Services)            │
│    - Check routes.reverse_proxies                   │
│    - Match path/prefix patterns                     │
│    → WEBSOCKET: Establish WebSocket proxy           │
│    → HTTP: Proxy to external service                │
│    → NO MATCH: Continue                             │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 7. Static File Serving                              │
│    - Check for static file extensions               │
│    - Look in configured public_dir                  │
│    → FOUND: Serve file with cache headers           │
│    → NOT FOUND: Continue                            │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 8. Try Files (Public Paths Only)                    │
│    - Only for paths without extensions              │
│    - Try configured suffixes (.html, etc.)          │
│    → FOUND: Serve file                              │
│    → NOT FOUND: Continue                            │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 9. Web Application Proxy                            │
│    - Extract tenant from path                       │
│    - Start or get existing app process              │
│    - Wait for app readiness (with timeout)          │
│    - Proxy request to tenant application            │
│    → SUCCESS: Return proxied response               │
│    → TIMEOUT: Serve maintenance page                │
│    → ERROR: Return 500 Internal Server Error        │
└───────────────────┬─────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────┐
│ 10. Response Completion                             │
│     - Stop idle tracking                            │
│     - Log request details                           │
│     - Return to client                              │
└─────────────────────────────────────────────────────┘
```

## Detailed Handler Flow

### 1. Request Initialization

**File:** `internal/server/handler.go:54` (`ServeHTTP`)

When a request arrives, Navigator immediately:

1. **Generates a Request ID** - Creates unique identifier if not already set by upstream proxy
2. **Creates ResponseRecorder** - Wraps the response writer to capture status codes, sizes, and metadata
3. **Starts Idle Tracking** - Notifies idle manager that a request is active (prevents premature machine suspension)

```go
// Generate request ID if not present
requestID := r.Header.Get("X-Request-Id")
if requestID == "" {
    requestID = utils.GenerateRequestID()
    r.Header.Set("X-Request-Id", requestID)
}

// Create response recorder for logging and tracking
recorder := NewResponseRecorder(w, h.idleManager)
defer recorder.Finish(r)

// Start idle tracking
recorder.StartTracking()
```

### 2. Health Check Endpoint

**File:** `internal/server/handler.go:77` (`handleHealthCheck`)

The `/up` endpoint provides a simple health check for load balancers and monitoring systems. It always returns:
- Status: `200 OK`
- Body: `OK`
- No authentication required
- No logging overhead

This is the fastest exit path from the request handler.

### 3. Sticky Session Routing

**File:** `internal/server/handler.go:134` (`handleStickySession`)

For Fly.io deployments with sticky sessions enabled, Navigator checks for session cookies that bind users to specific machines.

**Configuration:**
```yaml
sticky_sessions:
  enabled: true
  cookie_name: "_navigator_machine"
  cookie_max_age: "2h"
  paths:
    - "/app/*"
    - "/dashboard/*"
```

**Behavior:**
- Checks if request path matches configured sticky session paths
- Looks for existing session cookie
- If cookie references unavailable machine, serves maintenance page
- If no cookie, allows request to proceed normally
- Sets cookie after successful routing

**Use Cases:**
- WebSocket connections requiring same backend
- Session state stored in memory
- Multi-region deployments with user affinity

### 4. Rewrite Rules

**File:** `internal/server/handler.go:159` (`handleRewrites`)

Rewrite rules modify request paths or redirect requests before further processing.

**Configuration:**
```yaml
server:
  rewrite_rules:
    - pattern: "^/old-path/(.*)$"
      replacement: "/new-path/$1"
      flag: redirect
    - pattern: "^/region/(.*)$"
      replacement: "/app/$1"
      flag: fly-replay:sjc
      methods: [GET, POST]
```

**Supported Flags:**

#### `redirect` - External Redirect
Returns HTTP 302 with new location:
```go
newPath := rule.Pattern.ReplaceAllString(r.URL.Path, rule.Replacement)
http.Redirect(w, r, newPath, http.StatusFound)
```

#### `fly-replay:TARGET:STATUS` - Fly.io Region Routing
Routes requests to specific Fly.io regions, apps, or machines:
- **Region:** `fly-replay:sjc` → San Jose datacenter
- **App:** `fly-replay:app=myapp` → Specific Fly app
- **Machine:** `fly-replay:machine=abc123:myapp` → Specific machine instance

**Smart Routing Logic:**

**File:** `internal/server/fly_replay.go:23` (`ShouldUseFlyReplay`)

Navigator automatically chooses between Fly-Replay and reverse proxy based on request size:

- **Requests < 1MB:** Uses Fly-Replay (lets Fly.io proxy handle it)
- **Requests ≥ 1MB:** Uses internal reverse proxy (avoids Fly-Replay limitations)

This ensures large uploads work reliably while keeping most requests fast.

**Automatic Fallback:**

When reverse proxy is needed, Navigator constructs internal Fly.io URLs:

```go
// Region-based: http://sjc.myapp.internal:3000/path
targetURL := fmt.Sprintf("http://%s.%s.internal:%d%s",
    region, flyAppName, listenPort, r.URL.Path)

// Machine-based: http://abc123.vm.myapp.internal:3000/path
targetURL := fmt.Sprintf("http://%s.vm.%s.internal:%d%s",
    machineID, appName, listenPort, r.URL.Path)
```

**Retry Detection:**

Navigator adds `X-Navigator-Retry: true` header when using Fly-Replay within the same app. If the request comes back (machine unavailable), it serves a maintenance page instead of infinite loops.

#### `last` - Internal Rewrite
Modifies path and continues processing:
```go
r.URL.Path = rule.Pattern.ReplaceAllString(r.URL.Path, rule.Replacement)
// Continue to next handler with modified path
```

### 5. Reverse Proxy Routes

**File:** `internal/server/proxy.go:37` (`handleReverseProxies`)

Reverse proxies route requests to standalone services like Redis, Action Cable servers, or other external backends.

**Configuration:**
```yaml
routes:
  reverse_proxies:
    - prefix: /cable
      target: http://localhost:28080
      websocket: true
      strip_path: true
      headers:
        X-Forwarded-For: $remote_addr
    - path: "^/api/v1/(.*)$"
      target: "http://api-service:8080/v1/$1"
      headers:
        X-Real-IP: $remote_addr
```

**Matching Logic:**
- **Prefix matching:** Simple string prefix (`/cable` matches `/cable/foo`)
- **Regex matching:** Full pattern match (`^/api/v1/(.*)$`)

**Path Handling:**

Navigator supports several path transformation strategies:

1. **Simple Proxy:** Forwards path as-is to target
2. **Strip Path:** Removes matched portion before forwarding
3. **Capture Groups:** Uses regex groups to rebuild path (`$1`, `$2`)

**WebSocket Support:**

**File:** `internal/server/proxy.go:180` (`handleWebSocketProxy`)

For WebSocket-enabled routes, Navigator:

1. Establishes connection to backend WebSocket server
2. Upgrades client connection
3. Proxies messages bidirectionally
4. Filters hop-by-hop headers (Connection, Upgrade)
5. Filters WebSocket handshake headers (connection-specific)
6. Forwards application headers (Sec-WebSocket-Protocol)

The proxy maintains two separate WebSocket connections:
- Client ↔ Navigator
- Navigator ↔ Backend

This allows Navigator to monitor, log, and handle errors for each connection independently.

### 3. Authentication Check ⚡ EARLY ENFORCEMENT

**File:** `internal/server/handler.go:82` (happens immediately after health check)
**Implementation:** `internal/auth/auth.go:46` (`CheckAuth`)

Navigator supports HTTP Basic Authentication using htpasswd files. **IMPORTANT:** Authentication is enforced **early** in the request flow, immediately after the health check and **before** all routing decisions. This prevents authentication bypass via reverse proxies, fly-replay, redirects, or WebSocket endpoints.

**Security Note:** Prior to October 2025, authentication happened later in the flow, which created bypass holes where reverse proxy routes (including Action Cable WebSocket) and fly-replay rewrites could be accessed without authentication. This has been fixed by moving auth enforcement to happen early.

**Configuration:**
```yaml
auth:
  enabled: true
  htpasswd: /path/to/.htpasswd
  realm: "Private Area"
  public_paths:
    - /assets/
    - /public/
    - "*.css"
    - "*.js"
  auth_patterns:
    - pattern: "^/showcase/2025/(raleigh|boston|seattle)/?$"
      action: "off"
    - pattern: "^/showcase/2025/(raleigh|boston)/public/"
      action: "off"
```

**Authentication Flow:**

1. **Check Auth Patterns:** Evaluate regex patterns first (most flexible)
   - Compiled regex patterns from `auth_patterns` configuration
   - Each pattern has an action: `"off"` (skip auth) or realm name (require auth with specific realm)
   - Supports grouped alternations for performance: `(token1|token2|token3)`
   - **Performance tip:** Use fewer patterns with alternations instead of many individual patterns

2. **Check Public Paths:** Skip auth for glob/prefix patterns
   - Prefix matches: `/assets/` matches `/assets/app.js`
   - Glob patterns: `*.css` matches `/styles/main.css`
   - Simpler than regex for common cases

3. **Extract Credentials:** Parse Basic Auth header
   ```go
   username, password, ok := r.BasicAuth()
   ```

4. **Validate Credentials:** Check against htpasswd file
   - Supports multiple hash formats: MD5, bcrypt, SHA1, crypt
   - Uses `github.com/tg123/go-htpasswd` library

5. **Send Challenge:** If validation fails
   ```
   HTTP/1.1 401 Unauthorized
   WWW-Authenticate: Basic realm="Private Area"
   ```

**Auth Pattern Matching Order:**

Navigator evaluates auth exclusions in this order:

1. **Auth Patterns (regex):** Most flexible, checked first
   - Allows complex matching with capture groups
   - Can override realm on per-pattern basis
   - Example: Match studio index pages but not tenant apps

2. **Public Paths (glob/prefix):** Simple patterns, checked second
   - Fast prefix matching (`/assets/`)
   - Glob pattern matching (`*.css`)
   - Easier to configure for common cases

**Auth Patterns vs Public Paths:**

Use **auth_patterns** when you need:
- Complex path matching (e.g., `/year/(studio1|studio2|studio3)/?$`)
- Exact path matching (e.g., index pages but not subdirectories)
- Per-pattern realm overrides
- Grouped alternations for performance (10 patterns vs 100+)

Use **public_paths** when you need:
- Simple prefix matching (e.g., `/assets/`)
- Glob patterns (e.g., `*.css`)
- Quick configuration without regex

**Performance Optimization:**

When configuring many auth exclusions, use grouped alternations:

```yaml
# GOOD: One pattern with alternation (fast)
auth_patterns:
  - pattern: "^/showcase/2025/(boston|seattle|raleigh|portland)/?$"
    action: "off"

# AVOID: Multiple individual patterns (slower)
auth_patterns:
  - pattern: "^/showcase/2025/boston/?$"
    action: "off"
  - pattern: "^/showcase/2025/seattle/?$"
    action: "off"
  - pattern: "^/showcase/2025/raleigh/?$"
    action: "off"
```

Grouped alternations reduce:
- Regex compilation overhead
- Number of patterns to check per request
- Memory footprint

### 7. Static File Serving

**File:** `internal/server/static.go:48` (`ServeStatic`)

Navigator serves static files directly from the filesystem, bypassing the web application for performance.

**Configuration:**
```yaml
server:
  static:
    public_dir: public
    allowed_extensions: [html, css, js, png, jpg, svg]
    cache_control:
      default: "1h"
      overrides:
        - path: /assets/
          max_age: "24h"
```

**Static File Flow:**

1. **Check Extension:** Verify file has allowed extension (if configured)
2. **Build File Path:** Join public_dir with request path
3. **Check Existence:** Use `os.Stat()` to verify file exists
4. **Set Headers:**
   - Content-Type: Automatic MIME type detection
   - Cache-Control: Based on path-specific configuration
5. **Serve File:** Use `http.ServeFile()` for efficient serving

**Cache Control:**

Navigator supports flexible cache header configuration:

```yaml
cache_control:
  default: "1h"           # All static files
  overrides:
    - path: /assets/      # Fingerprinted assets
      max_age: "24h"
    - path: /favicon.ico  # Specific files
      max_age: "7d"
```

Duration parsing supports:
- Seconds: `3600` or `3600s`
- Minutes: `60m`
- Hours: `24h`
- Days: `7d` (parsed as hours: `168h`)

**Root Path Stripping:**

For applications mounted under a prefix (e.g., `/showcase`), Navigator automatically strips the root path before looking for files:

```go
// Request: /showcase/assets/app.js
// Root path: /showcase
// File lookup: public/assets/app.js
```

### 8. Try Files

**File:** `internal/server/static.go:117` (`TryFiles`)

Try Files allows serving files with different extensions than the requested path, useful for static sites and SPAs.

**Configuration:**
```yaml
server:
  static:
    try_files: [.html, /index.html, .htm]
```

**Try Files Flow:**

1. **Check Path:** Only process paths without extensions
2. **Skip Tenant Paths:** Don't interfere with web app routing
3. **Try Extensions:** Attempt each suffix in order
   - `/studios/boston` → `/studios/boston.html`
   - `/docs/guide` → `/docs/guide/index.html`
4. **Serve First Match:** Return first existing file

**Use Cases:**
- Static site generators (Jekyll, Hugo)
- Single-page applications (React, Vue)
- Clean URLs without extensions

**Important:** Try Files only applies to public paths (paths that don't match tenant routes). This prevents conflicts with dynamic application routing.

### 9. Web Application Proxy

**File:** `internal/server/handler.go:262` (`handleWebAppProxy`)

This is the core of Navigator's multi-tenant functionality. It routes requests to the appropriate tenant application.

**Configuration:**
```yaml
applications:
  startup_timeout: "30s"  # Global default
  tenants:
    - name: "2025/boston"
      path: "/2025/boston"
      root: /var/www/boston
      startup_timeout: "45s"  # Tenant-specific override
```

**Tenant Matching:**

Navigator uses **longest prefix matching** to find the appropriate tenant:

```go
// Finds the tenant with the longest matching path prefix
// /2025/boston/classes → matches "2025/boston" (not just "2025")
tenantName, found := h.extractTenantFromPath(r.URL.Path)
```

**Application Lifecycle:**

**File:** `internal/process/app_manager.go`

1. **Get or Start App:**
   ```go
   app, err := h.appManager.GetOrStartApp(tenantName)
   ```
   - Checks if app is already running
   - If not, starts new app process:
     - Allocates dynamic port (4000-4099)
     - Sets up environment variables
     - Executes configured runtime/server
     - Creates PID file for process tracking

2. **Wait for Readiness:**
   ```go
   select {
   case <-app.ReadyChan():
       // App is ready, proceed with proxy
   case <-time.After(startupTimeout):
       // Timeout, serve maintenance page
   case <-r.Context().Done():
       // Client disconnected, return 499
   }
   ```

   **Startup Timeout Precedence:**
   1. Tenant-specific `startup_timeout` (highest priority)
   2. Global `applications.startup_timeout`
   3. Default: 30 seconds

   **Health Check:**
   Applications signal readiness through health checks. Navigator polls `http://localhost:PORT/up` until:
   - Returns HTTP 200
   - Or timeout expires

3. **Proxy Request:**

   **File:** `internal/proxy/proxy.go:274` (`ProxyWithWebSocketSupport`)

   Once app is ready, Navigator proxies the request:

   ```go
   targetURL := fmt.Sprintf("http://localhost:%d", app.Port)
   proxy.ProxyWithWebSocketSupport(w, r, targetURL, wsPtr)
   ```

   **Features:**
   - WebSocket detection and handling
   - WebSocket connection tracking (optional)
   - X-Forwarded-* headers preserved
   - Client disconnect detection (499 status)
   - Automatic retry with exponential backoff

**WebSocket Tracking:**

**Configuration:**
```yaml
applications:
  track_websockets: true  # Global setting
  tenants:
    - name: "app1"
      track_websockets: false  # Per-tenant override
```

When enabled, Navigator tracks active WebSocket connections:
- Increments counter on connection upgrade
- Decrements on connection close
- Prevents app shutdown while WebSockets active
- Useful for idle management and metrics

**Retry Logic:**

**File:** `internal/proxy/proxy.go:84` (`HandleProxyWithRetry`)

For regular HTTP requests (GET/HEAD), Navigator implements automatic retry:

1. **Connection Failures:** Retry with exponential backoff
   - Initial delay: 10ms
   - Max delay: 500ms
   - Max duration: 3 seconds

2. **Response Buffering:**
   - Buffers responses up to 64KB
   - Allows retry if connection fails mid-response
   - Large responses stream directly (no buffering)

3. **Safety:**
   - Only retries safe methods (GET, HEAD)
   - Non-idempotent methods (POST, PUT) fail immediately
   - Prevents duplicate operations

**Error Handling:**

Navigator distinguishes between different failure scenarios:

1. **Client Disconnect (499):**
   - Client closed connection while waiting
   - No error logged (expected behavior)
   - Similar to nginx behavior

2. **App Startup Timeout:**
   - Serves maintenance page
   - Status: 503 Service Unavailable
   - Configurable maintenance.html

3. **Proxy Failure (502):**
   - Backend connection failed
   - After all retries exhausted
   - Logged as error

### 10. Response Completion

**File:** `internal/server/handler.go:405` (`Finish`)

After the request is processed, Navigator finalizes:

1. **Stop Idle Tracking:**
   ```go
   if r.idleManager != nil && r.tracked {
       r.idleManager.RequestFinished()
   }
   ```
   Decrements active request counter, enabling idle detection.

2. **Log Request:**

   **File:** `internal/server/logging.go`

   Structured logging includes:
   - Method, path, status code
   - Response size and duration
   - Tenant name (if applicable)
   - Response type (static, proxy, fly-replay)
   - Backend information
   - Request ID for tracing

   **Example Log:**
   ```
   INFO Request completed method=GET path=/2025/boston/classes status=200
        size=4523 duration=145ms tenant=2025/boston response_type=proxy
        request_id=abc123
   ```

3. **Return Response:**
   All headers and body already written during processing.

## Special Cases and Edge Conditions

### Client Disconnects

Navigator detects client disconnects throughout the request lifecycle:

**During App Startup:**
```go
case <-r.Context().Done():
    w.WriteHeader(499)  // Client closed connection
```

**During Proxy:**
```go
if r.Context().Err() == context.Canceled {
    // Client disconnected, don't log as error
    w.WriteHeader(499)
}
```

Status code 499 follows nginx convention for client-closed requests.

### Maintenance Pages

Navigator serves maintenance pages in several scenarios:

1. **App Startup Timeout:** App not ready within configured timeout
2. **Fly-Replay Retry:** Target machine unavailable (retry detected)
3. **Sticky Session Failure:** Cookie references unavailable machine

**Maintenance Page Locations:**
1. `public/maintenance.html` (preferred)
2. Built-in generic message

### Large Request Handling

For requests ≥ 1MB (Fly-Replay limitation):

1. **Automatic Detection:**
   ```go
   if r.ContentLength >= MaxFlyReplaySize {
       // Use reverse proxy instead
   }
   ```

2. **Fallback Construction:**
   Navigator builds internal Fly.io URLs:
   ```
   http://region.app.internal:3000/path
   ```

3. **Transparent to Client:**
   No configuration needed, automatic behavior.

### Process Recovery

**File:** `internal/process/app_manager.go`

If an app process crashes:

1. **Detection:** Proxy connection refused
2. **Cleanup:** Remove stale PID file
3. **Restart:** Call `GetOrStartApp()` again
4. **Retry:** Original request retried automatically

### Graceful Shutdown

**File:** `cmd/navigator-refactored/main.go:321` (`handleShutdown`)

When Navigator receives SIGTERM/SIGINT:

1. **Stop Idle Manager:** Prevent new suspensions
2. **Shutdown HTTP Server:** Stop accepting new connections
3. **Stop Web Apps:** Terminate tenant applications
4. **Stop Managed Processes:** Terminate Redis, Sidekiq, etc.
5. **Timeout:** 30 seconds for complete shutdown

All managers receive context for coordinated shutdown.

## Configuration Reload

**File:** `cmd/navigator-refactored/main.go:264` (`handleReload`)

Navigator supports live configuration reload via SIGHUP:

```bash
kill -HUP $(cat /tmp/navigator.pid)
# or
./bin/navigator -s reload
```

**Reload Process:**

1. **Load New Config:** Parse YAML, validate settings
2. **Update Managers:**
   - App Manager: New tenant configuration
   - Process Manager: Managed processes diff
   - Idle Manager: New timeout/action
3. **Reload Auth:** Refresh htpasswd file
4. **Update Handler:** Recreate handler with new config
5. **Execute Start Hooks:** Run configured hooks

**What Gets Updated:**
- Tenant configurations
- Managed processes (starts new, stops removed)
- Authentication settings
- Static file configuration
- Rewrite rules
- Reverse proxy routes
- Idle timeouts

**What Doesn't Change:**
- Listen address/port (requires restart)
- Running tenant applications (not restarted)
- Active connections (continue uninterrupted)

## Performance Optimizations

### Early Exit Paths

Navigator prioritizes fast paths:

1. **Health checks:** Immediate 200 OK response
2. **Static files:** Direct filesystem serving
3. **Reverse proxies:** Skip tenant matching

### Dynamic Port Allocation

Instead of sequential port assignment, Navigator uses availability checking:

```go
// Finds first available port in range
port := findAvailablePort(4000, 4099)
```

Benefits:
- Prevents port conflicts
- Faster app startup
- No coordination needed

### Response Buffering

For retry capability, Navigator buffers up to 64KB:

```go
if w.body.Len() + len(b) > MaxRetryBufferSize {
    // Switch to streaming mode
    w.Commit()
    w.written = true
}
```

This balances:
- Retry capability for most responses
- Memory efficiency for large responses
- Streaming for downloads/uploads

### WebSocket Optimization

Navigator passes WebSocket connections through with minimal overhead:

1. Single upgrade per connection
2. Direct message forwarding (no buffering)
3. Goroutine-based bidirectional proxy
4. Optional tracking for idle management

## Debugging and Observability

### Log Levels

Set via `LOG_LEVEL` environment variable:

```bash
LOG_LEVEL=debug ./bin/navigator config.yml
```

**Debug Level:** Shows detailed request routing:
```
DEBUG Request received method=GET path=/2025/boston/classes
DEBUG Checking static file path=/2025/boston/classes.css
DEBUG Tenant extraction result tenantName=2025/boston found=true
DEBUG Proxying request target=http://localhost:4001
```

**Info Level:** Shows request completion and significant events:
```
INFO Request completed method=GET path=/classes status=200 duration=145ms
INFO App started tenant=2025/boston port=4001
```

### Request Metadata

Every request tracks metadata for logging:

```go
recorder.SetMetadata("response_type", "static")
recorder.SetMetadata("tenant", tenantName)
recorder.SetMetadata("file_path", fsPath)
```

Metadata appears in structured logs for analysis.

### Request Tracing

Request IDs flow through the entire system:

```go
requestID := r.Header.Get("X-Request-Id")
// Passed to backend applications via X-Request-Id header
```

Enables end-to-end tracing across Navigator and tenant apps.

## Summary

Navigator's request handling prioritizes:

1. **Performance:** Early exits for common cases
2. **Reliability:** Automatic retry and error recovery
3. **Flexibility:** Multiple routing strategies
4. **Observability:** Comprehensive logging and tracing
5. **Simplicity:** Clear flow with minimal configuration

The modular design allows each component to be tested independently while maintaining a cohesive request flow.

## References

### Key Source Files

- **Main Handler:** `internal/server/handler.go`
- **Static Files:** `internal/server/static.go`
- **Reverse Proxy:** `internal/server/proxy.go`
- **Fly-Replay:** `internal/server/fly_replay.go`
- **Authentication:** `internal/auth/auth.go`
- **Proxy Logic:** `internal/proxy/proxy.go`
- **Process Management:** `internal/process/app_manager.go`
- **Server Lifecycle:** `cmd/navigator-refactored/main.go`

### Related Documentation

- [Configuration Reference](../configuration/yaml-reference.md)
- [Deployment Guide](../deployment/production.md)
- [Features Overview](../features/overview.md)
