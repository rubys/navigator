# YAML Configuration Reference

Complete reference for all Navigator configuration options.

## server

HTTP server configuration.

```yaml
server:
  listen: 3000                    # Port to listen on (required)
  hostname: "localhost"           # Hostname for requests (optional)
  public_dir: "./public"          # Default public directory (optional)
  root_path: "/showcase"          # Root URL path prefix (optional)
  authentication: "./htpasswd"    # Path to htpasswd file (optional)
  auth_exclude:                   # Paths excluded from auth (optional)
    - "/assets/"
    - "*.css"
    - "*.js"

  # Machine idle configuration (Fly.io)
  idle:
    action: suspend               # "suspend" or "stop"
    timeout: 20m                  # Duration: "30s", "5m", "1h30m"

  # Sticky sessions configuration
  sticky_sessions:
    enabled: true
    cookie_name: "_navigator_machine"
    cookie_max_age: "2h"
    cookie_secure: true
    cookie_httponly: true
    cookie_samesite: "Lax"
    cookie_path: "/"
    paths:                        # Optional: specific paths for sticky sessions
      - "/app/*"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `listen` | integer/string | `3000` | Port to bind HTTP server |
| `hostname` | string | `""` | Hostname for Host header matching |
| `public_dir` | string | `"./public"` | Default directory for static files |
| `root_path` | string | `""` | Root URL path prefix (e.g., "/showcase") |
| `authentication` | string | `""` | Path to htpasswd file for authentication |
| `auth_exclude` | array | `[]` | Glob patterns for paths excluded from auth |

### server.idle

Machine auto-suspend/stop configuration (Fly.io only).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `action` | string | `""` | Action to take: "suspend" or "stop" |
| `timeout` | string | `""` | Idle duration before action (e.g., "20m", "1h") |

### server.sticky_sessions

Cookie-based session affinity for routing requests to the same machine.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable sticky sessions |
| `cookie_name` | string | `"_navigator"` | Name of the session cookie |
| `cookie_max_age` | string | `"24h"` | Cookie lifetime (duration format) |
| `cookie_secure` | boolean | `true` | Set Secure flag (HTTPS only) |
| `cookie_httponly` | boolean | `true` | Set HttpOnly flag |
| `cookie_samesite` | string | `"Lax"` | SameSite attribute: "Strict", "Lax", "None" |
| `cookie_path` | string | `"/"` | Cookie path |
| `paths` | array | `[]` | Specific URL patterns for sticky sessions |

## auth

Authentication configuration using htpasswd files.

```yaml
auth:
  enabled: true                   # Enable/disable authentication
  realm: "Protected Area"         # Authentication realm name
  htpasswd: "./htpasswd"          # Path to htpasswd file
  public_paths:                   # Paths that bypass authentication
    - "/assets/"
    - "/favicon.ico"
    - "*.css"
    - "*.js"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable authentication |
| `realm` | string | `"Protected Area"` | Basic Auth realm |
| `htpasswd` | string | `""` | Path to htpasswd file |
| `public_paths` | array | `[]` | Glob patterns for public paths |

### Supported htpasswd Formats

- APR1 (Apache MD5)
- bcrypt
- SHA
- MD5-crypt
- Plain text (not recommended)

## static

Static file serving configuration.

```yaml
static:
  directories:                    # Static directory mappings
    - path: "/assets/"            # URL path
      root: "public/assets/"      # Filesystem path
      cache: 86400                # Cache header in seconds
  extensions: [css, js, png, jpg] # File extensions to serve
  try_files:
    enabled: true                 # Enable try_files behavior
    suffixes: [".html", ".htm"]   # Suffixes to try
    fallback: rails               # Fallback to Rails if no file found
```

### directories

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | URL path prefix (must start/end with /) |
| `root` | string | Filesystem directory path |
| `cache` | integer | Cache-Control max-age in seconds |

### extensions

List of file extensions to serve directly from filesystem.

### try_files

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable try_files behavior |
| `suffixes` | array | `[]` | File suffixes to try |
| `fallback` | string | `"rails"` | Fallback when no file found |

## applications

Application configuration for multi-tenant deployments.

```yaml
applications:
  # Process pool configuration
  pools:
    max_size: 10                  # Maximum app processes
    timeout: 5m                   # Idle timeout (duration format)
    start_port: 4000              # Starting port for allocation

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
      track_websockets: false     # Override: disable WebSocket tracking

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
| `timeout` | string | `"5m"` | Idle timeout (duration: "5m", "10m") |
| `start_port` | integer | `4000` | Starting port for dynamic allocation |

### applications.health_check

Global default health check endpoint for application readiness detection. Can be overridden per-tenant.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `health_check` | string | `"/"` | HTTP endpoint for health checks (e.g., "/up", "/health") |

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
| `hooks` | object | | Tenant-specific lifecycle hooks |

**Note**: The `name` field is automatically derived from the `path` (e.g., `/showcase/2025/boston/` → `2025/boston`).

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
      
  fly_replay:                    # Fly.io replay routing
    - path: "^/api/"             # URL pattern
      region: "syd"              # Target region
      app: "my-app"              # Target app
      machine: "abc123"          # Target machine ID
      status: 307               # HTTP status
      methods: [GET, POST]      # HTTP methods
```

### rewrites

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | ✓ | Regular expression pattern |
| `replacement` | string | ✓ | Replacement string |
| `redirect` | boolean | | Send HTTP redirect vs internal rewrite |
| `status` | integer | | HTTP status code for redirects |

### fly_replay

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
    ready:                        # Execute when Navigator is ready
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
4. **File paths**: Must be accessible by Navigator process (htpasswd, config files)
5. **Regex patterns**: Must compile successfully (routes, fly_replay)
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
  authentication: /etc/navigator/htpasswd
  auth_exclude: ["/assets/", "*.css", "*.js"]

static:
  directories:
    - path: /assets/
      dir: assets/
      cache: 24h

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

## See Also

- [Configuration Overview](index.md)
- [Server Settings](server.md)
- [Applications](applications.md)
- [Authentication](authentication.md)
- [Lifecycle Hooks](../features/lifecycle-hooks.md)
- [Machine Suspend](../features/machine-suspend.md)
- [Sticky Sessions](../features/sticky-sessions.md)
- [Logging](../features/logging.md)
- [Examples](../examples/index.md)