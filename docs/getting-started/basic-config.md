# Basic Configuration

Learn the essential Navigator configuration options to get your Rails application running efficiently.

## Configuration File Structure

Navigator uses YAML configuration files with a clear, hierarchical structure:

```yaml title="navigator.yml"
# HTTP server settings
server:
  listen: 3000
  hostname: localhost
  public_dir: ./public

# Process pool management  
pools:
  max_size: 10
  idle_timeout: 300
  start_port: 4000

# Authentication (optional)
auth:
  enabled: false

# Static file serving (optional)
static:
  directories: []
  extensions: []

# Rails applications (required)
applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/app
```

## Server Configuration

The `server` section defines how Navigator listens for HTTP requests:

```yaml
server:
  listen: 3000              # Port to bind (required)
  hostname: "localhost"     # Hostname for routing (optional)
  public_dir: "./public"    # Default public directory (optional)
```

### Common Server Settings

| Environment | Port | Hostname | Public Dir |
|-------------|------|----------|------------|
| **Development** | 3000 | localhost | ./public |
| **Production** | 3000 or 80 | your-domain.com | /var/www/app/public |
| **Docker** | 3000 | 0.0.0.0 | ./public |

## Process Pool Configuration

The `pools` section controls how Navigator manages Rails processes:

```yaml
pools:
  max_size: 10          # Maximum Rails processes
  idle_timeout: 300     # Stop after 5 minutes of inactivity
  start_port: 4000      # Starting port for Rails processes
```

### Sizing Guidelines

| Server Size | max_size | idle_timeout | Use Case |
|-------------|----------|--------------|----------|
| **Small VPS** | 3-5 | 120s | Development, staging |
| **Medium Server** | 8-15 | 300s | Production, moderate traffic |
| **Large Server** | 20+ | 600s | High traffic, many tenants |

## Application Configuration

The `applications` section is where you define your Rails applications:

```yaml
applications:
  # Global environment variables for all apps
  global_env:
    RAILS_ENV: production
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    
  # Individual applications
  tenants:
    - name: myapp              # Unique identifier
      path: /                  # URL path prefix
      working_dir: /var/www/app # Rails application directory
```

### Single Application

Most common setup - one Rails app handling all requests:

```yaml
applications:
  global_env:
    RAILS_ENV: production
    DATABASE_URL: "${DATABASE_URL}"
    
  tenants:
    - name: main
      path: /
      working_dir: /var/www/app
```

### Multiple Applications

Route different paths to different Rails applications:

```yaml
applications:
  global_env:
    RAILS_ENV: production
    
  tenants:
    - name: api
      path: /api/
      working_dir: /var/www/api-app
      
    - name: admin  
      path: /admin/
      working_dir: /var/www/admin-app
      
    - name: main
      path: /
      working_dir: /var/www/main-app
```

## Environment Variables

Navigator supports environment variable substitution using `${VAR}` syntax:

```yaml
applications:
  global_env:
    # Simple substitution
    DATABASE_URL: "${DATABASE_URL}"
    
    # With default values
    REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"
    
    # Environment-specific
    RAILS_ENV: "${RAILS_ENV:-development}"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
```

### Setting Environment Variables

=== "Development"

    ```bash
    # Set in shell
    export RAILS_ENV=development
    export DATABASE_URL=postgres://localhost/myapp_dev
    
    # Or use .env file (not committed)
    echo "RAILS_ENV=development" >> .env
    echo "DATABASE_URL=postgres://localhost/myapp_dev" >> .env
    
    # Start Navigator
    ./navigator navigator.yml
    ```

=== "Production"

    ```bash
    # Set system-wide
    echo 'RAILS_ENV=production' >> /etc/environment
    echo 'SECRET_KEY_BASE=your-secret-key' >> /etc/environment
    
    # Or in systemd service
    [Service]
    Environment=RAILS_ENV=production
    Environment=SECRET_KEY_BASE=your-secret-key
    ExecStart=/usr/local/bin/navigator /etc/navigator/navigator.yml
    ```

## Static File Configuration

Serve static files directly for better performance:

```yaml
static:
  directories:
    - path: /assets/          # URL path
      root: public/assets/    # Filesystem path
      cache: 86400           # Cache for 24 hours
      
  extensions: [css, js, png, jpg, gif, ico]
```

### Performance Benefits

| File Type | Without Navigator | With Navigator | Improvement |
|-----------|------------------|----------------|-------------|
| **CSS/JS** | ~50ms (Rails) | ~2ms (direct) | 25x faster |
| **Images** | ~30ms (Rails) | ~1ms (direct) | 30x faster |
| **Fonts** | ~40ms (Rails) | ~1ms (direct) | 40x faster |

## Authentication Setup

Protect your application with HTTP Basic Authentication:

```yaml
auth:
  enabled: true
  realm: "My Application"
  htpasswd: /path/to/htpasswd
  public_paths:
    - /assets/
    - /favicon.ico
    - "*.css"
    - "*.js"
```

### Create htpasswd File

```bash
# Create htpasswd file
htpasswd -c /etc/navigator/htpasswd admin

# Add more users
htpasswd /etc/navigator/htpasswd user2
```

## Configuration Examples

### Development Configuration

```yaml title="config/navigator-dev.yml"
server:
  listen: 3000
  public_dir: ./public

pools:
  max_size: 3
  idle_timeout: 60        # Shorter timeout for development
  
applications:
  global_env:
    RAILS_ENV: development
    
  tenants:
    - name: dev
      path: /
      working_dir: .

# No auth in development
auth:
  enabled: false
```

### Staging Configuration

```yaml title="config/navigator-staging.yml"
server:
  listen: 3000
  public_dir: /var/www/app/public

pools:
  max_size: 5
  idle_timeout: 180

static:
  directories:
    - path: /assets/
      root: /var/www/app/public/assets/
      cache: 3600          # 1 hour cache for staging
  extensions: [css, js, png, jpg, gif]

auth:
  enabled: true
  realm: "Staging Environment"
  htpasswd: /etc/navigator/staging-htpasswd
  public_paths: ["/assets/", "*.css", "*.js"]

applications:
  global_env:
    RAILS_ENV: staging
    DATABASE_URL: "${STAGING_DATABASE_URL}"
    SECRET_KEY_BASE: "${STAGING_SECRET_KEY}"
    
  tenants:
    - name: staging
      path: /
      working_dir: /var/www/app
```

### Production Configuration

```yaml title="config/navigator-prod.yml"
server:
  listen: 3000
  hostname: myapp.com
  public_dir: /var/www/app/public

pools:
  max_size: 20
  idle_timeout: 600        # 10 minutes for production

static:
  directories:
    - path: /assets/
      root: /var/www/app/public/assets/
      cache: 31536000      # 1 year for fingerprinted assets
    - path: /images/
      root: /var/www/app/public/images/
      cache: 86400         # 1 day for images
  extensions: [css, js, map, png, jpg, gif, ico, svg, woff, woff2]

auth:
  enabled: true
  realm: "Production Application"
  htpasswd: /etc/navigator/htpasswd
  public_paths: 
    - /assets/
    - /images/ 
    - /robots.txt
    - /favicon.ico
    - "*.css"
    - "*.js"
    - "*.woff*"

applications:
  global_env:
    RAILS_ENV: production
    RAILS_SERVE_STATIC_FILES: "false"  # Navigator handles static files
    DATABASE_URL: "${DATABASE_URL}"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    REDIS_URL: "${REDIS_URL}"
    
  tenants:
    - name: production
      path: /
      working_dir: /var/www/app
```

## Testing Your Configuration

### Validate Configuration

```bash
# Test configuration syntax
./navigator --validate navigator.yml

# Or start in foreground to see any errors
./navigator navigator.yml
```

### Test Static Files

```bash
# Should be served directly by Navigator (fast)
curl -I http://localhost:3000/assets/application.css
# Look for "X-Served-By: Navigator" header

# Should be served by Rails (slower)
curl -I http://localhost:3000/some-rails-path
```

### Test Rails Connection

```bash
# Check Rails app starts
curl http://localhost:3000/

# Monitor process creation
ps aux | grep -E '(navigator|rails|ruby)'
```

## Configuration Best Practices

### 1. Use Environment-Specific Files

```bash
# Development
./navigator config/navigator-dev.yml

# Staging  
./navigator config/navigator-staging.yml

# Production
./navigator config/navigator-prod.yml
```

### 2. Secure Sensitive Data

```yaml
# Good - use environment variables
applications:
  global_env:
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    DATABASE_PASSWORD: "${DB_PASSWORD}"

# Bad - hardcoded secrets
applications:
  global_env:
    SECRET_KEY_BASE: "hardcoded-secret-key"  # Never do this!
```

### 3. Optimize for Your Use Case

```yaml
# High traffic site
pools:
  max_size: 30
  idle_timeout: 1800  # 30 minutes

# Low traffic site  
pools:
  max_size: 5
  idle_timeout: 120   # 2 minutes
```

### 4. Use Absolute Paths in Production

```yaml
# Good
server:
  public_dir: /var/www/app/public
applications:
  tenants:
    - working_dir: /var/www/app

# Avoid in production
server:
  public_dir: ./public     # Relative paths can break
```

## Common Configuration Issues

### Rails App Won't Start

1. **Check working directory**:
   ```yaml
   applications:
     tenants:
       - working_dir: /correct/path/to/app  # Must exist
   ```

2. **Verify environment variables**:
   ```bash
   # Test Rails can start
   cd /var/www/app
   RAILS_ENV=production bundle exec rails server
   ```

### Static Files Not Served

1. **Check path mapping**:
   ```yaml
   static:
     directories:
       - path: /assets/              # URL path
         root: public/assets/        # Must match filesystem
   ```

2. **Verify files exist**:
   ```bash
   ls -la /var/www/app/public/assets/
   ```

### Port Conflicts

1. **Change listen port**:
   ```yaml
   server:
     listen: 3001  # Use different port
   ```

2. **Check port usage**:
   ```bash
   netstat -tlnp | grep 3000
   ```

## Next Steps

Now that you understand basic configuration:

- [Add static file serving](../configuration/static-files.md)
- [Set up authentication](../configuration/authentication.md)
- [Configure multiple applications](../examples/multi-tenant.md)
- [Add background processes](../examples/with-sidekiq.md)

## See Also

- [Configuration Reference](../configuration/yaml-reference.md)
- [Server Settings](../configuration/server.md)
- [Application Configuration](../configuration/applications.md)
- [Examples](../examples/index.md)