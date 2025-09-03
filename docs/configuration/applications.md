# Application Configuration

The `applications` section defines how Navigator manages web applications across different frameworks, including routing, environment variables, and tenant configurations.

## Basic Structure

```yaml
applications:
  # Framework-specific configuration
  framework:
    runtime_executable: ruby
    server_executable: bin/rails
    server_command: server
    server_args: ["-p", "${port}"]
    app_directory: /rails
    port_env_var: PORT
    startup_delay: 5
  
  # Global environment variables for all applications
  global_env:
    RAILS_ENV: production
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
  
  # Environment template with variable substitution
  env:
    RAILS_APP_DB: "${database}"
    RAILS_APP_OWNER: "${owner}"
    PIDFILE: "pids/${database}.pid"
  
  # Individual application tenants
  tenants:
    - name: myapp
      path: /
      working_dir: /var/www/myapp
      var:
        database: "myapp_production"
        owner: "My Company"
```

## Framework Configuration

Navigator supports multiple web frameworks through the `framework` section. This makes Navigator framework-independent while providing sensible Rails defaults:

```yaml
applications:
  framework:
    runtime_executable: ruby           # Runtime interpreter (ruby, node, python)
    server_executable: bin/rails       # Server command (bin/rails, server.js, manage.py)
    server_command: server             # Subcommand (server, runserver)
    server_args: ["-p", "${port}"]    # Arguments (port will be substituted)
    app_directory: /rails              # Working directory for apps
    port_env_var: PORT                 # Environment variable for port
    startup_delay: 5                   # Seconds to wait before marking ready
```

### Framework Examples

=== "Rails (Default)"

    ```yaml
    framework:
      runtime_executable: ruby
      server_executable: bin/rails
      server_command: server
      server_args: ["-p", "${port}"]
      app_directory: /rails
      port_env_var: PORT
      startup_delay: 5
    ```

=== "Django"

    ```yaml
    framework:
      runtime_executable: python
      server_executable: manage.py
      server_command: runserver
      server_args: ["0.0.0.0:${port}"]
      app_directory: /app
      port_env_var: PORT
      startup_delay: 3
    ```

=== "Node.js"

    ```yaml
    framework:
      runtime_executable: node
      server_executable: server.js
      server_command: ""              # No subcommand needed
      server_args: []                 # Port handled via environment
      app_directory: /app
      port_env_var: PORT
      startup_delay: 2
    ```

=== "Custom Framework"

    ```yaml
    framework:
      runtime_executable: /usr/local/bin/myruntime
      server_executable: bin/start-server
      server_command: production
      server_args: ["--port", "${port}", "--workers", "4"]
      app_directory: /myapp
      port_env_var: SERVER_PORT
      startup_delay: 10
    ```

### Configuration Fields

| Field | Required | Description | Example |
|-------|----------|-------------|---------|
| `runtime_executable` | Yes | Runtime interpreter command | `ruby`, `python`, `node` |
| `server_executable` | Yes | Server script/command | `bin/rails`, `manage.py`, `server.js` |
| `server_command` | No | Subcommand for server | `server`, `runserver` |
| `server_args` | No | Additional arguments | `["-p", "${port}"]` |
| `app_directory` | No | Working directory | `/rails`, `/app` (defaults to `/app`) |
| `port_env_var` | No | Port environment variable | `PORT` (default) |
| `startup_delay` | No | Startup delay in seconds | `5` (default) |

### Template Variables

- `${port}` - Dynamically allocated port number
- Can be used in `server_args` for port substitution

## Global Environment Variables

The `global_env` section sets environment variables for all web applications:

```yaml
applications:
  global_env:
    RAILS_ENV: production
    DATABASE_URL: "${DATABASE_URL}"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"
    RAILS_SERVE_STATIC_FILES: "false"  # Navigator handles static files
```

### Common Global Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `RAILS_ENV` | Rails environment | `production`, `staging`, `development` |
| `SECRET_KEY_BASE` | Rails secret key | From environment variable |
| `DATABASE_URL` | Database connection | PostgreSQL, MySQL, SQLite URLs |
| `REDIS_URL` | Redis connection | For caching and sessions |
| `RAILS_SERVE_STATIC_FILES` | Static file handling | `false` (let Navigator handle) |

## Environment Templates

Use the `env` section for variable substitution across tenants:

```yaml
applications:
  env:
    # Database configuration per tenant
    RAILS_APP_DB: "${database}"
    DATABASE_URL: "postgresql://user:pass@localhost/${database}"
    
    # Tenant-specific settings
    RAILS_APP_OWNER: "${owner}"
    RAILS_STORAGE: "${storage_path}"
    RAILS_APP_LOGO: "${logo_file}"
    
    # Process management
    PIDFILE: "pids/${database}.pid"
```

Each tenant can define `var` values that get substituted into the template.

## Tenant Configuration

### Basic Tenant

```yaml
tenants:
  - name: myapp           # Unique identifier
    path: /               # URL path prefix
    working_dir: /var/www/myapp  # Rails application directory
```

### Multi-Tenant Setup

```yaml
tenants:
  - name: client-a
    path: /clients/a/
    working_dir: /var/www/app
    var:
      database: "client_a_db"
      owner: "Client A Inc"
      storage_path: "/storage/client-a"
  
  - name: client-b
    path: /clients/b/
    working_dir: /var/www/app
    var:
      database: "client_b_db" 
      owner: "Client B Ltd"
      storage_path: "/storage/client-b"
```

### Tenant-Specific Environment

Override environment variables per tenant:

```yaml
tenants:
  - name: main-app
    path: /
    working_dir: /var/www/app
    env:
      RAILS_ENV: production
      SPECIAL_FEATURE: "enabled"
  
  - name: staging-app
    path: /staging/
    working_dir: /var/www/app
    env:
      RAILS_ENV: staging
      DEBUG_MODE: "true"
```

## Advanced Configuration

### Special Tenants

Mark tenants as special to skip variable substitution:

```yaml
tenants:
  - name: main-app
    path: /
    working_dir: /var/www/app
    # Uses env template with variable substitution
    var:
      database: "main_db"
  
  - name: cable
    path: /cable
    working_dir: /var/www/app
    special: true  # Skip variable substitution
    env:
      RAILS_ENV: production
      CABLE_ADAPTER: redis
```

### Pattern Matching

Use patterns for flexible routing:

```yaml
tenants:
  - name: api-v1
    path: /api/v1/
    working_dir: /var/www/api
  
  - name: websocket
    path: /ws/
    match_pattern: "*/ws/*"  # Matches any path with /ws/
    working_dir: /var/www/websocket
```

### Standalone Servers

Proxy to external services instead of Rails:

```yaml
tenants:
  - name: action-cable
    path: /cable/
    standalone_server: "localhost:28080"  # Proxy to external server
  
  - name: api-service
    path: /api/
    standalone_server: "api.internal.com:8080"
```

### Process Limits

Control concurrent requests per tenant:

```yaml
tenants:
  - name: high-traffic
    path: /
    working_dir: /var/www/app
    max_concurrent_requests: 10  # Limit concurrent requests
  
  - name: websocket
    path: /ws/
    working_dir: /var/www/app
    force_max_concurrent_requests: 0  # Unlimited (for WebSockets)
```

## Configuration Examples

### Single Application

```yaml
applications:
  global_env:
    RAILS_ENV: production
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    DATABASE_URL: "${DATABASE_URL}"
  
  tenants:
    - name: app
      path: /
      working_dir: /var/www/app
```

### Multi-Tenant SaaS

```yaml
applications:
  global_env:
    RAILS_ENV: production
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
  
  env:
    DATABASE_URL: "postgresql://user:pass@localhost/${tenant_db}"
    RAILS_APP_TENANT: "${tenant_name}"
    STORAGE_PATH: "/storage/${tenant_id}"
    REDIS_NAMESPACE: "${tenant_id}"
  
  tenants:
    - name: acme-corp
      path: /acme/
      working_dir: /var/www/saas
      var:
        tenant_db: "acme_production"
        tenant_name: "Acme Corporation"
        tenant_id: "acme"
    
    - name: widget-inc
      path: /widget/
      working_dir: /var/www/saas
      var:
        tenant_db: "widget_production" 
        tenant_name: "Widget Inc"
        tenant_id: "widget"
```

### API + Admin Split

```yaml
applications:
  global_env:
    RAILS_ENV: production
    DATABASE_URL: "${DATABASE_URL}"
  
  tenants:
    - name: api
      path: /api/
      working_dir: /var/www/api
      env:
        API_MODE: "true"
        CORS_ORIGINS: "*.example.com"
    
    - name: admin
      path: /admin/
      working_dir: /var/www/admin
      env:
        ADMIN_MODE: "true"
        SESSION_TIMEOUT: "3600"
    
    - name: main
      path: /
      working_dir: /var/www/frontend
```

### Development Setup

```yaml
applications:
  global_env:
    RAILS_ENV: development
    
  tenants:
    - name: dev
      path: /
      working_dir: .  # Current directory
      env:
        RAILS_LOG_LEVEL: debug
        EAGER_LOADING: "false"
```

## Variable Substitution

### Syntax

- `${variable}` - Replace with variable value
- `${variable:-default}` - Use default if variable not set
- Variables defined in tenant's `var` section

### Example

```yaml
env:
  DATABASE_URL: "postgresql://user:pass@${db_host:-localhost}/${database}"
  STORAGE_PATH: "/storage/${environment}/${tenant_id}"
  LOG_FILE: "/var/log/app-${tenant_id}.log"

tenants:
  - name: production-client
    var:
      database: "client_prod"
      db_host: "prod-db.example.com"
      environment: "production"
      tenant_id: "client1"
```

**Results in**:
```
DATABASE_URL=postgresql://user:pass@prod-db.example.com/client_prod
STORAGE_PATH=/storage/production/client1
LOG_FILE=/var/log/app-client1.log
```

## Best Practices

### 1. Use Global Environment for Common Settings

```yaml
global_env:
  RAILS_ENV: production
  SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
  RAILS_LOG_TO_STDOUT: "true"
  RAILS_SERVE_STATIC_FILES: "false"
```

### 2. Template Repeated Patterns

```yaml
# Instead of repeating in each tenant
env:
  DATABASE_URL: "postgresql://user:pass@localhost/${db_name}"
  REDIS_NAMESPACE: "${tenant_prefix}"
  STORAGE_PATH: "/storage/${tenant_prefix}"
```

### 3. Secure Sensitive Data

```yaml
# Good - use environment variables
global_env:
  SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
  DATABASE_PASSWORD: "${DB_PASSWORD}"

# Bad - hardcoded secrets
global_env:
  SECRET_KEY_BASE: "hardcoded-secret"  # Never do this!
```

### 4. Organize by Environment

```yaml
# Production config
global_env:
  RAILS_ENV: production
  RAILS_LOG_LEVEL: info

# Development config  
global_env:
  RAILS_ENV: development
  RAILS_LOG_LEVEL: debug
```

## Troubleshooting

### Rails App Won't Start

1. **Check working directory**:
   ```bash
   ls -la /var/www/app
   cat /var/www/app/Gemfile
   ```

2. **Verify environment variables**:
   ```bash
   cd /var/www/app
   RAILS_ENV=production bundle exec rails runner "puts Rails.env"
   ```

3. **Test Rails manually**:
   ```bash
   cd /var/www/app
   RAILS_ENV=production bundle exec rails server -p 4001
   ```

### Environment Variables Not Set

1. **Check substitution**:
   ```yaml
   # Ensure variables are defined in tenant
   tenants:
     - name: myapp
       var:
         database: "myapp_db"  # Required for ${database}
   ```

2. **Test manually**:
   ```bash
   echo "DATABASE_URL=${DATABASE_URL}"
   ```

### Path Routing Issues

1. **Check path conflicts**:
   ```yaml
   # More specific paths first
   tenants:
     - name: api
       path: /api/v1/     # Specific
     - name: main
       path: /            # Catch-all last
   ```

2. **Test routing**:
   ```bash
   curl -I http://localhost:3000/api/v1/users
   curl -I http://localhost:3000/
   ```

## See Also

- [YAML Reference](yaml-reference.md)
- [Environment Variables](../reference/environment.md)
- [Process Management](../features/process-management.md)
- [Examples](../examples/index.md)