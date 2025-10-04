# Sticky Sessions

Sticky sessions (also called session affinity) ensure that requests from the same client are consistently routed to the same machine. This is crucial for maintaining stateful connections, accessing machine-local data, or preserving WebSocket connections.

## Overview

Navigator provides built-in sticky session support using HTTP cookies. When enabled, Navigator:

1. Generates a unique machine identifier for each instance
2. Stores it in an HTTP cookie on the client's first request
3. Routes subsequent requests with that cookie to the same machine
4. Handles failover gracefully when machines are unavailable

## Why Sticky Sessions?

### Use Cases

**WebSocket Connections**
- Maintains long-lived WebSocket connections to the same machine
- Ensures Action Cable subscriptions stay connected
- Prevents reconnection overhead

**Machine-Local Data**
- Access SQLite databases stored on specific machines
- Utilize machine-specific caches
- Read temporary files or session data

**Connection Pooling**
- Reuse database connections on the same machine
- Reduce connection establishment overhead
- Optimize resource usage

**Stateful Applications**
- Maintain in-memory state across requests
- Support applications that cache data locally
- Enable machine-specific optimizations

## Configuration

### Basic Configuration

```yaml
routes:
  fly:
    sticky_sessions:
      enabled: true
      cookie_name: "_navigator_machine"
      cookie_max_age: "2h"
      cookie_secure: true
      cookie_httponly: true
```

### Path-Specific Sessions

Limit sticky sessions to specific URL paths:

```yaml
routes:
  fly:
    sticky_sessions:
      enabled: true
      cookie_name: "_navigator_machine"
      cookie_max_age: "2h"
      paths:
        - "/app/*"
        - "/dashboard/*"
        - "/cable"  # Action Cable endpoint
```

### Complete Example

```yaml
server:
  listen: 3000
  static:
    public_dir: public

routes:
  fly:
    sticky_sessions:
      enabled: true
      cookie_name: "_navigator_session"
      cookie_max_age: "4h"
      cookie_secure: true
      cookie_httponly: true
      cookie_samesite: "Lax"
      paths:
        - "/app/*"
        - "/admin/*"
        - "/cable"

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /app
```

## Configuration Options

### `enabled`
**Type**: Boolean
**Default**: `false`
**Description**: Enable or disable sticky session support.

```yaml
routes:
  fly:
    sticky_sessions:
      enabled: true
```

### `cookie_name`
**Type**: String
**Default**: `"_navigator_machine"`
**Description**: Name of the HTTP cookie used to store the machine ID.

```yaml
routes:
  fly:
    sticky_sessions:
      cookie_name: "_my_app_session"
```

### `cookie_max_age`
**Type**: Duration String
**Default**: `"24h"`
**Description**: How long the cookie should persist. Uses Go duration format.

**Valid formats**:
- `"30m"` - 30 minutes
- `"2h"` - 2 hours
- `"24h"` - 24 hours
- `"72h"` - 3 days

```yaml
routes:
  fly:
    sticky_sessions:
      cookie_max_age: "2h"
```

### `cookie_secure`
**Type**: Boolean
**Default**: `false`
**Description**: Set the `Secure` flag, requiring HTTPS. Enable for production.

```yaml
routes:
  fly:
    sticky_sessions:
      cookie_secure: true
```

### `cookie_httponly`
**Type**: Boolean
**Default**: `true`
**Description**: Set the `HttpOnly` flag, preventing JavaScript access. Recommended for security.

```yaml
routes:
  fly:
    sticky_sessions:
      cookie_httponly: true
```

### `cookie_samesite`
**Type**: String (`"Strict"`, `"Lax"`, `"None"`)
**Default**: `"Lax"`
**Description**: Set the `SameSite` attribute for CSRF protection.

**Options**:
- `"Strict"` - Cookie only sent for same-site requests
- `"Lax"` - Cookie sent for same-site and top-level navigation
- `"None"` - Cookie sent with all requests (requires `cookie_secure: true`)

```yaml
routes:
  fly:
    sticky_sessions:
      cookie_samesite: "Strict"
```

### `paths`
**Type**: Array of Strings
**Default**: `[]` (all paths)
**Description**: URL patterns that should use sticky sessions. If empty, applies to all paths.

**Pattern matching**:
- Exact match: `"/cable"`
- Wildcard suffix: `"/app/*"`
- Wildcard prefix: `"*/admin"`

```yaml
routes:
  fly:
    sticky_sessions:
      paths:
        - "/app/*"
        - "/admin/*"
        - "/cable"
```

## How It Works

### First Request Flow

```
1. Client sends request → Navigator
2. Navigator checks for sticky session cookie
3. Cookie not found → Generate machine ID
4. Set cookie in response: _navigator_machine=fly-machine-abc123
5. Route request to available instance
6. Return response with cookie
```

### Subsequent Requests Flow

```
1. Client sends request with cookie → Navigator
2. Navigator reads machine ID from cookie: fly-machine-abc123
3. Check if target machine is available
4. If available → Route to that machine
5. If unavailable → Serve maintenance page or fallback
6. Return response
```

### Failover Behavior

When the target machine is unavailable:

**Option 1: Maintenance Page (Default)**
```
- Show user-friendly "Under Maintenance" page
- Log the unavailability for monitoring
- Return HTTP 503 status
```

**Option 2: Fallback Routing (Future)**
```
- Clear the sticky session cookie
- Route to an available machine
- Set new cookie with new machine ID
```

## Integration with Fly.io

Sticky sessions work seamlessly with Fly.io's distributed infrastructure:

### Multi-Region Support

```yaml
# Works across all Fly.io regions
routes:
  fly:
    sticky_sessions:
      enabled: true
      cookie_name: "_fly_machine"
      cookie_max_age: "2h"
```

**Benefits**:
- Requests consistently route to the same region
- Reduces latency by keeping users on nearby machines
- Maintains local state across requests

### Fly-Replay Compatibility

Sticky sessions integrate with Fly-Replay headers:

```yaml
routes:
  rewrites:
    - pattern: "^/api/(.*)"
      rewrite: "/$1"

  fly:
    replay:
      - path: "^/api/"
        region: ord  # Prefer Chicago
        status: 307

    sticky_sessions:
      enabled: true
      paths:
        - "/app/*"  # Sticky for app, not API
```

**Behavior**:
- API requests use Fly-Replay routing
- App requests use sticky sessions
- Can mix routing strategies per path

### Machine Suspension

Sticky sessions respect machine suspension:

```yaml
server:
  idle:
    action: suspend
    timeout: 20m

routes:
  fly:
    sticky_sessions:
      enabled: true
```

**Behavior**:
- User's first request wakes suspended machine
- Cookie routes subsequent requests to same machine
- Machine suspends after idle timeout
- Next request wakes it again

## Large Request Handling

Navigator automatically falls back to reverse proxy for requests larger than 1MB, even with sticky sessions enabled.

### Why?

Fly-Replay has a 1MB limit. Larger requests must be proxied directly.

### Behavior

```
Request ≤ 1MB → Use sticky session + Fly-Replay
Request > 1MB → Direct reverse proxy (bypasses Fly-Replay)
```

This happens automatically—no configuration needed.

## WebSocket Example

Perfect for Rails Action Cable with Solid Cable:

```yaml
server:
  listen: 3000
  static:
    public_dir: public

routes:
  fly:
    sticky_sessions:
      enabled: true
      cookie_name: "_cable_machine"
      cookie_max_age: "24h"
      paths:
        - "/cable"  # Only for WebSocket endpoint

applications:
  tenants:
    - name: web-app
      path: /
      working_dir: /app

  standalone_servers:
    - name: action-cable
      match_path: /cable
      command: bundle
      args: [exec, puma, -p, "4001", cable/config.ru]
      working_dir: /app
      port: 4001
```

**Benefits**:
- WebSocket connections stay on same machine
- Solid Cable SQLite database accessible locally
- No Redis required for cable state

## Multi-Tenant Example

Use sticky sessions with multi-tenant apps:

```yaml
routes:
  fly:
    sticky_sessions:
      enabled: true
      cookie_name: "_tenant_machine"
      cookie_max_age: "4h"

applications:
  tenants:
    - name: tenant-a
      path: /tenant-a
      working_dir: /app
      var:
        tenant_id: tenant_a

    - name: tenant-b
      path: /tenant-b
      working_dir: /app
      var:
        tenant_id: tenant_b
```

**Benefits**:
- Each tenant's requests route to consistent machine
- Tenant-specific SQLite databases stay local
- Reduces cross-machine data access

## Monitoring and Debugging

### Check Cookie in Browser

**Chrome DevTools**:
1. Open Developer Tools (F12)
2. Application tab → Cookies
3. Look for `_navigator_machine` cookie
4. Value shows machine ID (e.g., `fly-machine-abc123`)

### Check Cookie in Logs

```bash
# Enable debug logging
LOG_LEVEL=debug navigator config.yml

# Look for sticky session logs
tail -f /var/log/navigator.log | grep "sticky"
```

**Example log output**:
```
2024-09-30T12:00:00Z DEBUG Sticky session cookie found machine=fly-machine-abc123
2024-09-30T12:00:00Z INFO Routing to machine via Fly-Replay machine=fly-machine-abc123
```

### Test Sticky Routing

```bash
# Make request and capture cookie
curl -c cookies.txt -b cookies.txt http://localhost:3000/app

# Subsequent requests use same machine
curl -b cookies.txt http://localhost:3000/app
curl -b cookies.txt http://localhost:3000/app

# Check cookie value
cat cookies.txt | grep navigator_machine
```

## Security Considerations

### Cookie Security Flags

**Always enable in production**:
```yaml
routes:
  fly:
    sticky_sessions:
      cookie_secure: true      # Requires HTTPS
      cookie_httponly: true    # Prevents XSS
      cookie_samesite: "Lax"   # Prevents CSRF
```

### Cookie Tampering

Navigator validates machine IDs:
- Invalid machine IDs are ignored
- Tampered cookies result in new machine assignment
- Logs suspicious activity for monitoring

### Session Hijacking

Mitigate with:
- `HttpOnly` flag (prevents JavaScript access)
- `Secure` flag (HTTPS only)
- `SameSite` attribute (CSRF protection)
- Short cookie lifetimes (2-4 hours recommended)

## Performance Impact

### Overhead

**Minimal overhead**:
- Cookie read: ~0.1ms
- Machine ID lookup: ~0.1ms
- Total: ~0.2ms per request

### Benefits

**Performance gains**:
- Reuse database connections: 10-50ms saved
- Hit machine-local caches: 50-200ms saved
- Maintain WebSocket connections: Eliminates reconnect overhead

**Net result**: Sticky sessions improve performance in most scenarios.

## Troubleshooting

### Sessions Not Sticking

**Check cookie settings**:
```bash
# Verify cookie is set
curl -v http://localhost:3000/ 2>&1 | grep Set-Cookie
```

**Common issues**:
- `cookie_secure: true` without HTTPS → Cookie not set
- Browser blocking third-party cookies
- Path mismatch (cookie path doesn't match request path)

### Machine Unavailable Errors

**Check machine status**:
```bash
# Fly.io machines
fly machines list

# Check Navigator logs
journalctl -u navigator | grep "machine unavailable"
```

**Solutions**:
- Increase `cookie_max_age` to reduce expired sessions
- Implement graceful failover (fallback routing)
- Monitor machine health proactively

### Cookie Not Persisting

**Check cookie expiration**:
```yaml
sticky_sessions:
  cookie_max_age: "2h"  # Increase if sessions expire too quickly
```

**Browser behavior**:
- Private/incognito mode may block cookies
- Browser settings may clear cookies on exit
- Ad blockers may interfere with cookies

## Best Practices

### 1. Set Appropriate Timeout

```yaml
routes:
  fly:
    sticky_sessions:
      cookie_max_age: "2h"  # Balance between convenience and stale routes
```

**Recommendations**:
- Short-lived apps: 30m-1h
- Standard apps: 2-4h
- Long-lived apps: 24h (with health checks)

### 2. Use Path Restrictions

```yaml
routes:
  fly:
    sticky_sessions:
      paths:
        - "/app/*"   # Only app needs sticky sessions
        - "/cable"   # WebSocket endpoint
```

**Benefits**:
- Reduces unnecessary sticky routing
- Improves load distribution
- Clearer intent in configuration

### 3. Enable Security Flags

```yaml
routes:
  fly:
    sticky_sessions:
      cookie_secure: true      # Production only
      cookie_httponly: true    # Always enabled
      cookie_samesite: "Lax"   # CSRF protection
```

### 4. Monitor Cookie Usage

```bash
# Track sticky session metrics
journalctl -u navigator | grep "sticky" | wc -l

# Monitor machine unavailable errors
journalctl -u navigator | grep "machine unavailable" | wc -l
```

### 5. Plan for Failover

```yaml
# Coming soon: Automatic fallback routing
routes:
  fly:
    sticky_sessions:
      enabled: true
      fallback_mode: "route_to_available"  # Future feature
```

## See Also

- [WebSocket Support](websocket-support.md) - Using sticky sessions with Action Cable
- [Fly-Replay Routing](fly-replay.md) - Combining with regional routing
- [Use Cases: Sticky Sessions](../use-cases.md#use-case-4-sticky-sessions) - Real-world examples
- [Configuration Reference](../configuration/yaml-reference.md) - Complete YAML reference