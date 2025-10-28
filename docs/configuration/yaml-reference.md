# YAML Configuration Reference

Complete reference for all Navigator configuration options.

## Configuration Structure Overview

Navigator's YAML configuration is organized into logical sections:

```yaml
server:                    # HTTP server settings
  listen: 3000
  hostname: "localhost"
  root_path: "/prefix"
  static:                  # Static file configuration
    public_dir: "./public"
    allowed_extensions: [...]
    try_files: [...]
    cache_control: {...}
  idle:                    # Fly.io machine idle management
    action: suspend
    timeout: 20m

auth:                      # Authentication (top-level)
  enabled: true
  realm: "Restricted"
  htpasswd: "./htpasswd"
  public_paths: [...]

maintenance:               # Maintenance page configuration
  page: "/503.html"

applications:              # Multi-tenant app configuration
  pools: {...}
  framework: {...}
  env: {...}
  tenants: [...]

managed_processes:         # External process management
  - name: redis
    command: redis-server
    ...

routes:                    # URL routing and rewriting
  rewrites: [...]
  reverse_proxies: [...]
  fly:                     # Fly.io-specific routing
    replay: [...]

hooks:                     # Lifecycle hooks
  server: {...}
  tenant: {...}

logging:                   # Logging configuration
  format: json
  file: "..."
  vector: {...}
```

## server

HTTP server configuration.

```yaml
server:
  listen: 3000                    # Port to listen on (required)
  hostname: "localhost"           # Hostname for requests (optional)
  root_path: "/showcase"          # Root URL path prefix (optional)

  # Static file configuration
  static:
    public_dir: "./public"        # Directory for static files (optional)
    allowed_extensions:           # Allowed file extensions (optional)
      - html
      - css
      - js
      - png
      - jpg
    try_files:                    # Try files suffixes (optional)
      - index.html
      - .html
      - .htm
    cache_control:
      default: "1h"               # Default cache duration
      overrides:                  # Path-specific cache durations
        - path: "/assets/"
          max_age: "24h"

  # Machine idle configuration (Fly.io)
  idle:
    action: suspend               # "suspend" or "stop"
    timeout: 20m                  # Duration: "30s", "5m", "1h30m"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `listen` | integer/string | `3000` | Port to bind HTTP server |
| `hostname` | string | `""` | Hostname for Host header matching |
| `root_path` | string | `""` | Root URL path prefix (e.g., "/showcase") |

### server.static

Static file serving configuration.

```yaml
server:
  static:
    public_dir: "./public"
    allowed_extensions: [html, css, js, png, jpg]
    try_files: [index.html, .html, .htm]
    cache_control:
      default: "1h"
      overrides:
        - path: "/assets/"
          max_age: "24h"
        - path: "/images/"
          max_age: "12h"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `public_dir` | string | `"./public"` | Directory for static files |
| `allowed_extensions` | array | `[]` | File extensions allowed (empty = all allowed) |
| `try_files` | array | `[]` | Suffixes to try when resolving paths |
| `cache_control` | object | - | Cache header configuration |
| `cache_control.default` | string | `""` | Default cache duration (e.g., "1h") |
| `cache_control.overrides` | array | `[]` | Path-specific cache configurations |
| `cache_control.overrides[].path` | string | - | URL path prefix to match |
| `cache_control.overrides[].max_age` | string | - | Cache duration (e.g., "24h") |

**Allowed Extensions**: If omitted or empty, all files in `public_dir` can be served. If specified, only files with these extensions can be served.

**Try Files**: When present, Navigator attempts each suffix in order before falling back to the application.

Example: For request `/studios/boston` with `try_files: [index.html, .html, .htm]`:
1. Try `public/studios/boston` (exact match)
2. Try `public/studios/boston/index.html`
3. Try `public/studios/boston.html`
4. Try `public/studios/boston.htm`
5. Fall back to application

### server.idle

Machine auto-suspend/stop configuration (Fly.io only).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `action` | string | `""` | Action to take: "suspend" or "stop" |
| `timeout` | string | `""` | Idle duration before action (e.g., "20m", "1h") |

### server.cgi_scripts

CGI script configuration for executing standalone scripts directly.

```yaml
server:
  cgi_scripts:
    - path: /admin/sync           # URL path (exact match)
      script: /opt/scripts/sync.rb # Path to executable script
      method: POST                 # HTTP method (optional, empty = all methods)
      user: appuser               # Unix user (optional, requires root)
      group: appgroup             # Unix group (optional)
      allowed_users:              # Access control (optional)
        - admin
        - operator
      timeout: 5m                 # Execution timeout (optional)
      reload_config: config/navigator.yml  # Reload config after execution (optional)
      env:                        # Additional environment variables
        RAILS_DB_VOLUME: /mnt/db
        RAILS_ENV: production
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | URL path for exact matching |
| `script` | string | Yes | Absolute path to executable CGI script |
| `method` | string | No | HTTP method restriction (GET, POST, etc.). Empty = all methods |
| `user` | string | No | Unix user to run script as (requires Navigator running as root) |
| `group` | string | No | Unix group to run script as |
| `allowed_users` | array | No | Usernames allowed to access this script. Empty = all authenticated users |
| `env` | map | No | Additional environment variables |
| `reload_config` | string | No | Config file to reload after successful execution |
| `timeout` | string | No | Execution timeout (e.g., "30s", "5m"). Zero = no timeout |

**Access Control**: When `allowed_users` is specified, only those usernames can access the script (returns 403 Forbidden for other authenticated users). If `allowed_users` is empty or not specified, all authenticated users can access the script. Scripts on paths listed in `auth.public_paths` can be accessed without authentication.

**See Also**: [CGI Scripts Documentation](../features/cgi-scripts.md) for detailed usage examples.

## auth

Authentication configuration using htpasswd files.

```yaml
auth:
  enabled: true                   # Enable/disable authentication
  realm: "Restricted"             # Authentication realm name
  htpasswd: "./htpasswd"          # Path to htpasswd file
  public_paths:                   # Simple patterns that bypass authentication
    - "/assets/"
    - "/favicon.ico"
    - "*.css"
    - "*.js"
  auth_patterns:                  # Advanced regex patterns for auth control
    - pattern: "^/showcase/2025/(boston|seattle)/?$"
      action: "off"               # "off" = bypass auth, or realm name
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable authentication |
| `realm` | string | `"Restricted"` | Basic Auth realm displayed in browser |
| `htpasswd` | string | `""` | Path to htpasswd file |
| `public_paths` | array | `[]` | Glob/prefix patterns for paths that bypass auth |
| `auth_patterns` | array | `[]` | Regex patterns with actions for auth control |

**Auth patterns** support complex regex matching and are checked before `public_paths`. Each pattern has:
- `pattern`: Regular expression to match against the request path
- `action`: `"off"` to bypass auth, or a realm name to require auth with that realm

See [Authentication](authentication.md) for detailed examples and performance tips.

### Supported htpasswd Formats

- APR1 (Apache MD5)
- bcrypt
- SHA
- MD5-crypt
- Plain text (not recommended)

## maintenance

Maintenance mode configuration.

```yaml
maintenance:
  enabled: false                  # Enable maintenance mode (dynamic requests get maintenance page)
  page: "/503.html"               # Path to maintenance page (within public_dir)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable maintenance mode - serves maintenance page for dynamic requests (static files still served) |
| `page` | string | `"/503.html"` | Path to custom maintenance page |

Navigator serves the maintenance page in these scenarios:

**Explicit Maintenance Mode** (`enabled: true`):

What **continues working** during maintenance mode:
- ✅ Health checks (`/up`) - for load balancer monitoring
- ✅ Authentication - htpasswd rules still enforced
- ✅ Static files - all files matching `allowed_extensions`
- ✅ Try files - extensionless URL resolution (e.g., `/page` → `/page.html`)
- ✅ Redirects - configured URL redirects
- ✅ Rewrites - URL path modifications
- ✅ Fly-Replay routes - PDF/XLSX generation, cross-region routing
- ✅ CGI scripts - configuration update endpoints
- ✅ Reverse proxies - WebSocket connections, external service proxies

What **shows maintenance page**:
- ❌ Dynamic web application requests only

This design ensures optimal user experience during maintenance - infrastructure and static content remain accessible while protecting the application during updates.

**Automatic Error Handling** (`enabled: false`, default):
- An application is starting and exceeds the `startup_timeout`
- A Fly-Replay target is unavailable

The maintenance page is served from `{server.static.public_dir}/{maintenance.page}` (e.g., `public/503.html`). If this file doesn't exist, Navigator serves a default maintenance page.

**Recommended 503.html:**

```html
<!DOCTYPE html>
<html>
<head>
  <title>Service Starting (503)</title>
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <meta http-equiv="refresh" content="5">  <!-- Auto-refresh every 5 seconds -->
  <style>
    body {
      font-family: arial, sans-serif;
      text-align: center;
      padding: 50px;
    }
  </style>
</head>
<body>
  <h1>Service Starting...</h1>
  <p>The application is starting up. This page will refresh automatically.</p>
</body>
</html>
```

The `<meta http-equiv="refresh" content="5">` tag automatically reloads the page every 5 seconds, so users don't need to manually refresh while waiting for the application to become ready.

**Example: Startup Maintenance Mode**

Use maintenance mode during container startup to provide immediate user feedback while initialization completes:

```yaml
# config/navigator-maintenance.yml - Initial startup config
server:
  listen: 3000
  static:
    public_dir: public

maintenance:
  enabled: true  # Dynamic requests get 503 page with auto-refresh, static files served
  page: /503.html

hooks:
  server:
    ready:
      - command: /app/script/initialize.sh  # Sync S3, generate config, etc.
        timeout: 5m
        reload_config: config/navigator.yml  # Switch to normal operation
```

**Flow:**
1. Navigator starts with `navigator-maintenance.yml` (maintenance enabled)
2. Server listens immediately, serves 503.html to dynamic requests (static files still served)
3. Ready hook runs asynchronously (initialization tasks)
4. Hook completes and triggers reload to `navigator.yml` (maintenance disabled)
5. Normal operation begins

This provides fast cold starts (~1s to first response) while long-running initialization happens in the background.

## applications

Application configuration for multi-tenant deployments.

```yaml
applications:
  # Process pool configuration
  pools:
    max_size: 10                  # Maximum app processes
    timeout: 5m                   # Idle timeout (duration format)
    start_port: 4000              # Starting port for allocation
    default_memory_limit: "512M"  # Default memory limit per tenant (Linux only)
    user: "rails"                 # Default user to run tenants as (Unix only)
    group: "rails"                # Default group to run tenants as (Unix only)

  # Startup configuration
  health_check: "/up"             # Default health check endpoint
  startup_timeout: "5s"           # Timeout before showing maintenance page

  # WebSocket connection tracking (default: true)
  track_websockets: true          # Track WebSocket connections globally

  # Framework configuration (optional, can be per-tenant)
  framework:
    command: bin/rails
    args: ["server", "-p", "${port}"]
    app_directory: "/rails"
    port_env_var: PORT
    start_delay: 2s

  # Environment variable templates
  env:
    DATABASE_URL: "sqlite3://db/${database}.sqlite3"
    RAILS_ENV: production

  # Individual tenant applications
  tenants:
    - path: "/tenant1/"           # URL path prefix (required)
      var:                        # Template variables
        database: tenant1
      env:                        # Tenant-specific environment
        TENANT_NAME: "Tenant 1"

      # Optional tenant-specific overrides
      root: "/app"                # Application directory
      public_dir: "public"        # Public files directory
      framework: rails            # Framework type
      runtime: bundle             # Runtime command
      server: exec                # Server command
      args: ["puma", "-p", "${port}"]  # Server arguments
      health_check: "/health"     # Override: custom health check endpoint
      startup_timeout: "10s"      # Override: wait longer for this tenant
      track_websockets: false     # Override: disable WebSocket tracking
      memory_limit: "1G"          # Override: memory limit for this tenant (Linux only)
      user: "app"                 # Override: user for this tenant (Unix only)
      group: "app"                # Override: group for this tenant (Unix only)

      # Tenant-specific lifecycle hooks
      hooks:
        start:
          - command: /app/bin/tenant-init.sh
            timeout: 30s
        stop:
          - command: /app/bin/tenant-cleanup.sh
            timeout: 30s
```

### applications.pools

Process pool management for tenant applications.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_size` | integer | `10` | Maximum number of app processes |
| `timeout` | string | `"5m"` | Idle timeout before stopping processes (duration: "5m", "10m"). Also controls automatic cleanup of deleted tenants after config reload. |
| `start_port` | integer | `4000` | Starting port for dynamic allocation |
| `default_memory_limit` | string | `""` | Default memory limit (e.g., "512M", "1G") - Linux only, requires root |
| `user` | string | `""` | Default user to run tenant processes as - Unix only |
| `group` | string | `""` | Default group to run tenant processes as - Unix only |

> **Note**: The `timeout` setting controls both resource management (stopping idle processes) and configuration reload cleanup (automatically removing deleted tenants). See [Configuration Hot Reload - Tenant Lifecycle](../features/hot-reload.md#tenant-lifecycle-during-reload) for details on tenant behavior during config reload.

**Memory Limits (Linux only)**:
- Requires running Navigator as root on Linux with cgroups v2
- Uses Linux cgroups to enforce per-tenant memory limits
- When a tenant exceeds its limit, the kernel OOM kills only that tenant
- OOM-killed tenants are removed from the registry and restart on next request
- Cgroups persist during idle timeout, only cleaned up on Navigator shutdown
- Supported formats: "512M", "1G", "2048M", "1.5G"
- On non-Linux or non-root: Configuration is ignored (graceful degradation)

**User/Group Credentials (Unix only)**:
- Runs tenant processes as specified non-root user for security isolation
- Navigator must run as root to drop privileges to specified user/group
- Enhances security by limiting tenant process permissions
- On Windows or when not running as root: Configuration is ignored

### applications.health_check

Global default health check endpoint for application readiness detection. Can be overridden per-tenant.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `health_check` | string | `"/"` | HTTP endpoint for health checks (e.g., "/up", "/health") |

### applications.startup_timeout

Global timeout for waiting for applications to become ready before serving the maintenance page. Can be overridden per-tenant.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `startup_timeout` | string | `"5s"` | Duration to wait for app startup (e.g., "5s", "10s", "30s") |

When an application is starting, Navigator waits up to this duration for the health check to pass. If the timeout is reached, Navigator serves the configured maintenance page (typically `public/503.html`) instead of returning a 502 error. The maintenance page can include `<meta http-equiv="refresh" content="5">` to auto-refresh.

**Example:**

```yaml
applications:
  startup_timeout: "10s"    # Wait 10 seconds globally

  tenants:
    - name: slow-app
      path: /slow/
      startup_timeout: "30s"  # Override: wait 30 seconds for this tenant
```

### applications.track_websockets

Global setting for WebSocket connection tracking. When enabled, Navigator tracks active WebSocket connections to prevent apps from shutting down during idle timeouts. Can be overridden per-tenant.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `track_websockets` | boolean | `true` | Track WebSocket connections to prevent idle shutdown |

**When to disable**: Tenants that proxy WebSockets to standalone servers (e.g., separate Action Cable) or don't handle WebSockets directly.

### applications.framework

Default framework configuration (can be overridden per-tenant).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `command` | string | `""` | Framework command (e.g., "bin/rails") |
| `args` | array | `[]` | Command arguments with ${port} substitution |
| `app_directory` | string | `""` | Application root directory |
| `port_env_var` | string | `"PORT"` | Environment variable for port number |
| `start_delay` | string | `"0s"` | Delay before starting (duration format) |

### applications.env

Environment variable templates with `${variable}` substitution from tenant `var` values.

### applications.tenants

List of tenant applications.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | ✓ | URL path prefix (must start/end with /) |
| `var` | object | | Template variables for env substitution |
| `env` | object | | Tenant-specific environment variables |
| `root` | string | | Application root directory |
| `public_dir` | string | | Public files directory |
| `framework` | string | | Framework type override |
| `runtime` | string | | Runtime command override |
| `server` | string | | Server command override |
| `args` | array | | Server arguments override |
| `health_check` | string | | Health check endpoint override (e.g., "/up") |
| `track_websockets` | boolean | | Override WebSocket tracking (nil = use global) |
| `memory_limit` | string | | Memory limit override (e.g., "1G") - Linux only |
| `user` | string | | User override (runs as this user) - Unix only |
| `group` | string | | Group override (runs as this group) - Unix only |
| `hooks` | object | | Tenant-specific lifecycle hooks |

**Note**: The `name` field is automatically derived from the `path` (e.g., `/showcase/2025/boston/` → `2025/boston`).

**Per-Tenant Memory Limits**: Useful for tenants with different resource requirements. For example, a large event might use `memory_limit: "1G"` while smaller events use the pool default of `512M`.

## managed_processes

External processes managed by Navigator.

```yaml
managed_processes:
  - name: redis                   # Process identifier
    command: redis-server         # Command to execute
    args: ["--port", "6379"]     # Command arguments
    working_dir: "/var/lib/redis" # Working directory
    env:                         # Environment variables
      REDIS_PORT: "6379"
    auto_restart: true           # Restart on crash
    start_delay: 0               # Delay before starting (seconds)
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Unique process identifier |
| `command` | string | ✓ | Command to execute |
| `args` | array | | Command arguments |
| `working_dir` | string | | Working directory |
| `env` | object | | Environment variables |
| `auto_restart` | boolean | | Restart process on crash |
| `start_delay` | integer | | Delay before starting (seconds) |

## routes

URL routing and rewriting rules.

```yaml
routes:
  rewrites:                      # URL rewrite rules
    - pattern: "^/old/(.*)"      # Regex pattern
      replacement: "/new/$1"     # Replacement string
      redirect: true             # HTTP redirect vs rewrite
      status: 301               # HTTP status code

  reverse_proxies:               # Reverse proxy routes
    - name: "api-backend"        # Route name
      path: "^/api/"             # URL pattern (regex)
      target: "https://api.example.com"  # Target URL
      strip_path: true           # Remove prefix before proxying
      headers:                   # Custom headers
        X-Forwarded-Host: "$host"

  fly:                           # Fly.io-specific routing
    replay:                      # Fly-Replay routing
      - path: "^/api/"           # URL pattern
        region: "syd"            # Target region
        app: "my-app"            # Target app
        machine: "abc123"        # Target machine ID
        status: 307             # HTTP status
        methods: [GET, POST]    # HTTP methods
```

### rewrites

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | ✓ | Regular expression pattern |
| `replacement` | string | ✓ | Replacement string |
| `redirect` | boolean | | Send HTTP redirect vs internal rewrite |
| `status` | integer | | HTTP status code for redirects |

### reverse_proxies

Reverse proxy routes to external services.

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `name` | string | - | ✓ | Descriptive name for the route |
| `path` | string | - | | Regular expression pattern (alternative to prefix) |
| `prefix` | string | - | | Simple prefix match (alternative to path) |
| `target` | string | - | ✓ | Target URL (supports `$1`, `$2` for capture groups) |
| `strip_path` | boolean | `false` | | Remove matched prefix before proxying |
| `headers` | object | - | | Custom headers (supports `$host`, `$remote_addr`, `$scheme`) |
| `websocket` | boolean | `false` | | Enable WebSocket proxying |

**Note:** Either `path` (regex) or `prefix` (simple string) must be specified, but not both.

**Capture Group Substitution:**

Use regex capture groups in `path` and reference them in `target` with `$1`, `$2`, etc.

```yaml
routes:
  reverse_proxies:
    # Single capture group
    - name: user-api
      path: "^/users/([0-9]+)$"
      target: "https://api.example.com/v1/user/$1"

    # Multiple capture groups
    - name: archive-proxy
      path: "^/archive/([0-9]{4})/([0-9]{2})/(.*)$"
      target: "https://archive.example.com/$1-$2/$3"
```

Request `/users/123` → Proxies to `https://api.example.com/v1/user/123`

### routes.fly

Fly.io-specific routing configuration.

#### routes.fly.replay

Fly-Replay routing for region/machine targeting.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | ✓ | URL regex pattern |
| `region` | string | | Target Fly.io region |
| `app` | string | | Target Fly.io app name |
| `machine` | string | | Target machine ID (requires app) |
| `status` | integer | | HTTP status code |
| `methods` | array | | HTTP methods to match |

## hooks

Lifecycle hooks for server and tenant events.

```yaml
hooks:
  # Server lifecycle hooks
  server:
    start:                        # Execute when Navigator starts
      - command: /usr/local/bin/init.sh
        args: ["--setup"]
        timeout: 30s
    ready:                        # Execute asynchronously after Navigator starts listening (initial start + config reloads)
      - command: curl
        args: ["-X", "POST", "http://monitoring.example.com/ready"]
        timeout: 5s
    idle:                         # Execute before machine suspend/stop
      - command: /usr/local/bin/backup-to-s3.sh
        timeout: 5m
    resume:                       # Execute after machine resume
      - command: /usr/local/bin/restore-from-s3.sh
        timeout: 2m

  # Default tenant lifecycle hooks (executed for all tenants)
  tenant:
    start:                        # Execute after tenant app starts
      - command: /usr/local/bin/tenant-init.sh
        args: ["${database}"]
        timeout: 30s
    stop:                         # Execute before tenant app stops
      - command: /usr/local/bin/tenant-backup.sh
        args: ["${database}"]
        timeout: 2m
```

### hooks.server

Server-level lifecycle hooks execute at key Navigator events.

| Event | When Executed | Use Cases |
|-------|---------------|-----------|
| `start` | Before Navigator accepts requests | Initialize services, run migrations |
| `ready` | After Navigator starts listening | Notify monitoring, warm caches |
| `idle` | Before machine suspend/stop (Fly.io) | Upload data to S3, checkpoint state |
| `resume` | After machine resume (Fly.io) | Download data from S3, reconnect services |

### hooks.tenant

Default tenant hooks execute for all tenant applications. Can be overridden per-tenant under `applications.tenants[].hooks`.

| Event | When Executed | Use Cases |
|-------|---------------|-----------|
| `start` | After tenant app starts | Initialize tenant data, warm caches |
| `stop` | Before tenant app stops | Backup database, sync to storage |

### Hook Configuration

Each hook entry supports:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | ✓ | Command to execute |
| `args` | array | | Command arguments (supports ${var} substitution) |
| `timeout` | string | | Max execution time (duration: "30s", "5m") |
| `reload_config` | string | | Config file to reload after hook succeeds (server hooks only). Only reloads if path differs OR file modified during execution. |

**Reload Logic**:

The `reload_config` field triggers smart configuration reload:
- Reloads if config file path differs from current config
- Reloads if config file was modified during hook execution
- Skips reload if nothing changed (improves performance)

See [Lifecycle Hooks](../features/lifecycle-hooks.md#configuration-reload) for details.

**Environment**:
- Server hooks receive Navigator's environment
- Tenant hooks receive tenant's full environment (including `env` and `var` values)

**Execution Order**:
- Multiple hooks execute sequentially in order
- Failed hooks log errors but don't stop Navigator
- Tenant stop: default hooks → tenant-specific hooks

## logging

Logging configuration for Navigator and managed processes.

```yaml
logging:
  format: json                    # "text" or "json"
  file: "/var/log/navigator.log" # Optional file output

  # Vector integration (optional)
  vector:
    enabled: true
    socket: "/tmp/vector.sock"
    config: "/etc/vector/vector.toml"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `format` | string | `"text"` | Log format: "text" or "json" |
| `file` | string | `""` | Optional file path for log output (supports {{app}} template) |

### logging.vector

Professional log aggregation with automatic Vector process management.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable Vector integration |
| `socket` | string | `""` | Unix socket path for Vector communication |
| `config` | string | `""` | Path to vector.toml configuration file |

**Log Format**: All process output (Navigator, web apps, managed processes) includes:
- **Text mode**: `[source.stream]` prefix (e.g., `[2025/boston.stdout]`)
- **JSON mode**: Structured with `timestamp`, `source`, `stream`, `message`, `tenant` fields

## Environment Variable Substitution

Navigator supports environment variable substitution using `${VAR}` syntax:

```yaml
applications:
  global_env:
    # Simple substitution
    DATABASE_URL: "${DATABASE_URL}"
    
    # With default value
    REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"
    
    # Nested in strings
    LOG_LEVEL: "Rails ${RAILS_ENV} logging"
```

## Template Variables

Use template variables for multi-tenant configurations:

```yaml
applications:
  env:
    DATABASE_NAME: "${database}"
    TENANT_ID: "${tenant_id}"
    
  tenants:
    - name: customer1
      var:
        database: "app_customer1"
        tenant_id: "cust1"
```

## Validation Rules

Navigator validates configuration on startup:

1. **Required fields**: `server.listen`, `applications.tenants[].path`
2. **Port ranges**: Listen port must be 1-65535
3. **Path format**: Tenant paths must start and end with `/`
4. **File paths**: Must be accessible by Navigator process (htpasswd, config files, maintenance page)
5. **Regex patterns**: Must compile successfully (routes.rewrites, routes.fly.replay)
6. **Process names**: Must be unique within managed_processes
7. **Duration format**: Must be valid Go duration (e.g., "30s", "5m", "1h30m")
8. **Hook timeouts**: Should be reasonable (<10m for most operations)

## Examples

### Basic Single App

```yaml
server:
  listen: 3000

applications:
  tenants:
    - name: app
      path: /
      working_dir: /var/www/app
```

### Multi-Tenant with Auth

```yaml
server:
  listen: 3000
  static:
    public_dir: /var/www/public
    allowed_extensions: [html, css, js, png, jpg, gif]
    cache_control:
      default: "1h"
      overrides:
        - path: /assets/
          max_age: 24h

auth:
  enabled: true
  realm: "Restricted"
  htpasswd: /etc/navigator/htpasswd
  public_paths: ["/assets/", "*.css", "*.js"]

maintenance:
  page: "/503.html"

applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000

  env:
    DATABASE_URL: "sqlite3://db/${database}.sqlite3"

  tenants:
    - path: /tenant1/
      var:
        database: tenant1
    - path: /tenant2/
      var:
        database: tenant2
```

### With Background Jobs and Hooks

```yaml
server:
  listen: 3000
  idle:
    action: suspend
    timeout: 20m

hooks:
  server:
    idle:
      - command: /usr/local/bin/backup-all.sh
        timeout: 5m
  tenant:
    stop:
      - command: /usr/local/bin/backup-tenant.sh
        args: ["${database}"]
        timeout: 2m

managed_processes:
  - name: redis
    command: redis-server
    args: ["/etc/redis/redis.conf"]
    auto_restart: true

  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /app
    env:
      REDIS_URL: redis://localhost:6379
    auto_restart: true
    start_delay: 2s

applications:
  pools:
    max_size: 5
    timeout: 10m
    start_port: 4000

  env:
    REDIS_URL: redis://localhost:6379
    DATABASE_URL: "sqlite3://db/${database}.sqlite3"

  tenants:
    - path: /
      var:
        database: production
```

### Complete Fly.io Configuration

```yaml
server:
  listen: 3000
  static:
    public_dir: /var/www/public
    allowed_extensions: [html, css, js, png, jpg, svg, ico, woff2]
    try_files: [index.html, .html]
    cache_control:
      default: "1h"
      overrides:
        - path: /assets/
          max_age: 7d
  idle:
    action: suspend
    timeout: 20m

auth:
  enabled: true
  realm: "Protected Application"
  htpasswd: /etc/navigator/htpasswd
  public_paths:
    - "/assets/*"
    - "*.css"
    - "*.js"
    - "/favicon.ico"

maintenance:
  page: "/503.html"

routes:
  rewrites:
    - pattern: "^/old-path/(.*)"
      replacement: "/new-path/$1"
      redirect: true
      status: 301

  reverse_proxies:
    - name: api-backend
      path: "^/api/"
      target: "https://api.example.com"
      strip_path: true

  fly:
    replay:
      - path: "^/regions/syd/"
        region: syd
        status: 307

applications:
  pools:
    max_size: 10
    timeout: 10m
    start_port: 4000

  health_check: "/up"
  startup_timeout: "5s"

  env:
    DATABASE_URL: "sqlite3://db/${database}.sqlite3"
    RAILS_ENV: production

  tenants:
    - path: /tenant1/
      var:
        database: tenant1
    - path: /tenant2/
      var:
        database: tenant2

managed_processes:
  - name: redis
    command: redis-server
    args: ["/etc/redis/redis.conf"]
    auto_restart: true

hooks:
  server:
    idle:
      - command: /usr/local/bin/backup-to-s3.sh
        timeout: 5m
    resume:
      - command: /usr/local/bin/restore-from-s3.sh
        timeout: 2m

logging:
  format: json
  file: /var/log/navigator.log
```

## See Also

- [Configuration Overview](index.md)
- [Server Settings](server.md)
- [Applications](applications.md)
- [Authentication](authentication.md)
- [Lifecycle Hooks](../features/lifecycle-hooks.md)
- [Machine Suspend](../features/machine-suspend.md)
- [Logging](../features/logging.md)
- [Examples](../examples/index.md)