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
  public_dir: ./public

# Process pool management
pools:
  max_size: 10
  idle_timeout: 300
  start_port: 4000

# Authentication
auth:
  enabled: true
  htpasswd: ./htpasswd

# Static file serving
static:
  directories: []
  extensions: []

# Applications
applications:
  global_env: {}
  env: {}
  tenants: []

# Managed processes
managed_processes: []

# Routing rules
routes:
  rewrites: []
  fly_replay: []

# Machine suspension (Fly.io)
suspend:
  enabled: false
  idle_timeout: 600
```

## Quick Examples

### Minimal Configuration

```yaml
server:
  listen: 3000

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
  public_dir: /var/www/app/public

auth:
  enabled: true
  htpasswd: /etc/navigator/htpasswd

static:
  directories:
    - path: /assets/
      root: /var/www/app/public/assets/
      cache: 86400

applications:
  global_env:
    RAILS_ENV: production
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    
  tenants:
    - name: production
      path: /
      working_dir: /var/www/app

pools:
  max_size: 20
  idle_timeout: 600
```

## Configuration Sections

| Section | Purpose | Required | Details |
|---------|---------|----------|---------|
| [server](server.md) | HTTP server settings | ✓ | Port, hostname, paths |
| [pools](pools.md) | Process pool management | | Resource limits |
| [auth](authentication.md) | Authentication setup | | htpasswd support |
| [static](static-files.md) | Static file serving | | Direct filesystem serving |
| [applications](applications.md) | Rails applications | ✓ | Tenants and environment |
| [managed_processes](processes.md) | External processes | | Redis, Sidekiq, etc. |
| [routes](routing.md) | URL routing rules | | Rewrites, redirects |
| [suspend](suspend.md) | Machine suspension | | Fly.io auto-suspend |

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
# Good
working_dir: /var/www/app
htpasswd: /etc/navigator/htpasswd

# Avoid
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

If migrating from nginx + Passenger, see [Migration Guide](../migration/from-nginx.md) for configuration mapping.

## Complete Example

See [examples](../examples/index.md) for complete, working configurations for various scenarios.

## Next Steps

- [Server Settings](server.md) - Configure ports and hostnames
- [Applications](applications.md) - Set up Rails applications
- [Authentication](authentication.md) - Protect your applications
- [Static Files](static-files.md) - Optimize asset serving
- [YAML Reference](yaml-reference.md) - Complete configuration reference