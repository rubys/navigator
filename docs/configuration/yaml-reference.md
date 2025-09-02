# YAML Configuration Reference

Complete reference for all Navigator configuration options.

## server

HTTP server configuration.

```yaml
server:
  listen: 3000                    # Port to listen on (required)
  hostname: "localhost"           # Hostname for requests (optional)
  public_dir: "./public"          # Default public directory (optional)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `listen` | integer | `3000` | Port to bind HTTP server |
| `hostname` | string | `""` | Hostname for Host header matching |
| `public_dir` | string | `"./public"` | Default directory for static files |

## pools

Process pool management settings.

```yaml
pools:
  max_size: 10                    # Maximum Rails processes
  idle_timeout: 300               # Seconds before stopping idle process
  start_port: 4000                # Starting port for Rails processes
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_size` | integer | `10` | Maximum number of Rails processes |
| `idle_timeout` | integer | `300` | Idle timeout in seconds |
| `start_port` | integer | `4000` | Starting port for Rails process allocation |

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

Rails application configuration.

```yaml
applications:
  global_env:                     # Environment variables for all apps
    RAILS_ENV: production
    
  env:                           # Template environment variables
    DATABASE_URL: "postgres://localhost/${database}"
    
  tenants:                       # Individual applications
    - name: myapp                # Unique identifier
      path: "/"                  # URL path prefix
      working_dir: "/var/www/app" # Rails application directory
      env:                       # App-specific environment
        DATABASE_NAME: myapp
      var:                       # Template variables
        database: myapp_db
      min_instances: 0           # Minimum running processes
      max_processes: 5           # Maximum processes for this app
      idle_timeout: 300          # Custom idle timeout
      standalone_server: "localhost:8080"  # Proxy to external server
      match_pattern: "*/api"     # Pattern matching for requests
      force_max_concurrent_requests: 1     # Limit concurrent requests
      auth_realm: "Admin"        # Custom auth realm
      exclude_methods: [DELETE]  # HTTP methods to exclude from proxy
```

### tenants

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Unique identifier for the application |
| `path` | string | ✓ | URL path prefix (must start/end with /) |
| `working_dir` | string | | Rails application directory |
| `env` | object | | Application-specific environment variables |
| `var` | object | | Template variables for substitution |
| `min_instances` | integer | | Minimum running processes |
| `max_processes` | integer | | Maximum processes for this app |
| `idle_timeout` | integer | | Custom idle timeout in seconds |
| `standalone_server` | string | | Proxy to external server (host:port) |
| `match_pattern` | string | | Glob pattern for URL matching |
| `force_max_concurrent_requests` | integer | | Limit concurrent requests (0=unlimited) |
| `auth_realm` | string | | Custom authentication realm |
| `exclude_methods` | array | | HTTP methods to exclude from proxy |

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

## suspend

Machine suspension configuration (Fly.io only).

```yaml
suspend:
  enabled: false                 # Enable auto-suspend
  idle_timeout: 600             # Idle time before suspend (seconds)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable auto-suspend feature |
| `idle_timeout` | integer | `600` | Idle timeout in seconds |

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

1. **Required fields**: `server.listen`, `applications.tenants[].name`, `applications.tenants[].path`
2. **Port ranges**: Listen port must be 1-65535
3. **Path format**: Paths must start and end with `/`
4. **File paths**: Must be accessible by Navigator process
5. **Regex patterns**: Must compile successfully
6. **Process names**: Must be unique within managed_processes

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

auth:
  enabled: true
  htpasswd: /etc/navigator/htpasswd
  public_paths: ["/assets/", "*.css", "*.js"]

static:
  directories:
    - path: /assets/
      root: public/assets/
      cache: 86400

applications:
  env:
    DATABASE_URL: "postgres://localhost/${database}"
    
  tenants:
    - name: tenant1
      path: /tenant1/
      var:
        database: app_tenant1
    - name: tenant2
      path: /tenant2/
      var:
        database: app_tenant2
```

### With Background Jobs

```yaml
managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /var/www/app
    env:
      REDIS_URL: redis://localhost:6379
    auto_restart: true
    start_delay: 2

applications:
  global_env:
    REDIS_URL: redis://localhost:6379
    
  tenants:
    - name: app
      path: /
      working_dir: /var/www/app
```

## See Also

- [Configuration Overview](index.md)
- [Server Settings](server.md)
- [Applications](applications.md)
- [Authentication](authentication.md)
- [Examples](../examples/index.md)