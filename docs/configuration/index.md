# Configuration

Navigator uses YAML configuration files to define server settings, applications, routing rules, and managed processes.

## Configuration File

Navigator looks for configuration in the following order:
1. File specified on command line: `navigator /path/to/config.yml`
2. Default location: `config/navigator.yml`
3. Current directory: `navigator.yml`

## Configuration Structure

A complete Navigator configuration has these main sections:

```yaml
# Server settings
server:
  listen: 3000
  hostname: localhost

  # Static file serving
  static:
    public_dir: ./public
    directories: []
    extensions: []

  # Machine idle management (Fly.io)
  idle:
    action: suspend  # or "stop"
    timeout: 10m

# Authentication
auth:
  enabled: true
  htpasswd: ./htpasswd

# Applications
applications:
  # Process pool management
  pools:
    max_size: 10
    idle_timeout: 5m
    start_port: 4000

  global_env: {}
  env: {}
  tenants: []

# Managed processes
managed_processes: []

# Routing rules
routes:
  rewrites: []
  fly:
    replay: []

# Maintenance page
maintenance:
  enabled: false
  page: public/503.html
```

## Quick Examples

### Minimal Configuration

```yaml
server:
  listen: 3000
  static:
    public_dir: public

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/app
```

### Production Configuration

```yaml
server:
  listen: 3000
  hostname: example.com

  static:
    public_dir: /var/www/app/public
    directories:
      - path: /assets/
        root: /var/www/app/public/assets/
        cache: 86400

auth:
  enabled: true
  htpasswd: /etc/navigator/htpasswd

applications:
  pools:
    max_size: 20
    idle_timeout: 10m

  global_env:
    RAILS_ENV: production
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"

  tenants:
    - name: production
      path: /
      working_dir: /var/www/app
```

## Configuration Sections

| Section | Purpose | Required | Details |
|---------|---------|----------|---------|
| [server](server.md) | HTTP server settings | ✓ | Port, hostname, static files, idle |
| [auth](authentication.md) | Authentication setup | | htpasswd support |
| [applications](applications.md) | Rails applications | ✓ | Tenants, pools, environment |
| [managed_processes](processes.md) | External processes | | Redis, Sidekiq, etc. |
| [routes](routing.md) | URL routing rules | | Rewrites, Fly-Replay |
| [maintenance](maintenance.md) | Maintenance pages | | Custom 503 pages |

## Environment Variables

Navigator supports environment variable substitution in configuration:

```yaml
applications:
  global_env:
    DATABASE_URL: "${DATABASE_URL}"
    REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"
```

Variables can have default values using `${VAR:-default}` syntax.

## Template Variables

For multi-tenant applications, use template variables:

```yaml
applications:
  env:
    DATABASE_NAME: "${database}"
    TENANT_NAME: "${name}"
    
  tenants:
    - name: customer1
      path: /customer1/
      var:
        database: app_customer1
        name: "Customer One"
```

## Configuration Reload

Navigator supports live configuration reload without restart:

```bash
# Send reload signal
navigator -s reload

# Or use kill directly
kill -HUP $(cat /tmp/navigator.pid)
```

!!! warning "Reload Limitations"
    - Port changes require restart
    - New managed processes start on reload
    - Existing Rails processes continue until idle

## Validation

Navigator validates configuration on startup:
- Required fields must be present
- Paths must be valid
- Ports must be available
- Regex patterns must compile

## Best Practices

### 1. Use Absolute Paths

```yaml
# Good - Absolute paths
working_dir: /var/www/app
htpasswd: /etc/navigator/htpasswd

# Avoid - Relative paths
working_dir: ./app
htpasswd: ../htpasswd
```

### 2. Separate Environments

```bash
# Development
navigator config/navigator.dev.yml

# Production
navigator config/navigator.prod.yml
```

### 3. Secure Sensitive Data

```yaml
# Use environment variables
applications:
  global_env:
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    DATABASE_PASSWORD: "${DB_PASSWORD}"
    
# Never commit secrets
# .gitignore
config/navigator.yml
config/*.yml
!config/*.example.yml
```

### 4. Optimize Static Serving

```yaml
server:
  static:
    # Fingerprinted assets (far-future cache)
    directories:
      - path: /assets/
        root: public/assets/
        cache: 31536000  # 1 year

      # Regular images (shorter cache)
      - path: /images/
        root: public/images/
        cache: 3600  # 1 hour
```

## Migration from nginx

Migration from nginx + Passenger is supported with configuration mapping.

## Complete Example

See [examples](../examples/index.md) for complete, working configurations for various scenarios.

## Next Steps

- [Server Settings](server.md) - Configure ports and hostnames
- [Applications](applications.md) - Set up Rails applications
- [Authentication](authentication.md) - Protect your applications
- [Static Files](static-files.md) - Optimize asset serving
- [YAML Reference](yaml-reference.md) - Complete configuration reference