# Single Rails Application

The simplest Navigator setup - serving one Rails application with optimized static file handling.

## Use Case

- Single Rails application
- Static assets served directly
- Optional authentication
- Development or production

## Basic Configuration

```yaml title="navigator.yml"
server:
  listen: 3000
  public_dir: ./public

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/rails/app
```

## Production Configuration

A complete production setup with all optimizations:

```yaml title="navigator.yml"
server:
  listen: 3000
  hostname: example.com
  public_dir: /var/www/myapp/public

# Serve static files directly
static:
  directories:
    - path: /assets/
      root: /var/www/myapp/public/assets/
      cache: 86400  # 24 hours
    - path: /packs/
      root: /var/www/myapp/public/packs/
      cache: 31536000  # 1 year (webpack assets)
  extensions: [html, css, js, png, jpg, gif, ico, svg, woff, woff2, ttf, eot]
  try_files:
    enabled: true
    suffixes: [".html", "index.html"]
    fallback: rails

# Optional authentication
auth:
  enabled: true
  realm: "MyApp"
  htpasswd: /etc/navigator/htpasswd
  public_paths:
    - /assets/
    - /packs/
    - /robots.txt
    - /favicon.ico

# Rails application
applications:
  global_env:
    RAILS_ENV: production
    RAILS_LOG_TO_STDOUT: "true"
    RAILS_SERVE_STATIC_FILES: "false"  # Navigator handles this
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    DATABASE_URL: "${DATABASE_URL}"
    
  tenants:
    - name: myapp
      path: /
      working_dir: /var/www/myapp
      
# Resource management
pools:
  max_size: 5
  idle_timeout: 600  # 10 minutes
  start_port: 4000
```

## Setup Steps

### 1. Prepare Your Rails Application

```bash
# Precompile assets for production
RAILS_ENV=production bundle exec rails assets:precompile

# Create database
RAILS_ENV=production bundle exec rails db:create db:migrate
```

### 2. Create Authentication File (Optional)

```bash
# Create htpasswd file
htpasswd -c /etc/navigator/htpasswd admin

# Add additional users
htpasswd /etc/navigator/htpasswd user2
```

### 3. Set Environment Variables

```bash
# Create .env file or export variables
export SECRET_KEY_BASE=$(bundle exec rails secret)
export DATABASE_URL="postgresql://user:pass@localhost/myapp"
export RAILS_ENV=production
```

### 4. Start Navigator

```bash
navigator /etc/navigator/navigator.yml
```

## Testing

### Verify Static Files

```bash
# Should serve directly (fast)
curl -I http://localhost:3000/assets/application.css
# Look for: X-Served-By: Navigator

# Should serve from public directory
curl -I http://localhost:3000/robots.txt
```

### Check Rails Application

```bash
# Should proxy to Rails
curl http://localhost:3000/

# Check authentication (if enabled)
curl -u admin:password http://localhost:3000/admin
```

### Monitor Processes

```bash
# See Navigator and Rails processes
ps aux | grep -E '(navigator|rails|ruby)'

# Check ports in use
netstat -tlnp | grep -E '(3000|400[0-9])'
```

## Development Configuration

Simplified setup for development:

```yaml title="navigator-dev.yml"
server:
  listen: 3000
  public_dir: ./public

applications:
  global_env:
    RAILS_ENV: development
    
  tenants:
    - name: dev
      path: /
      working_dir: .

# No authentication in development
auth:
  enabled: false

# Shorter idle timeout for development
pools:
  idle_timeout: 60  # 1 minute
```

## Performance Optimizations

### 1. Preload Application

Keep Rails always running:

```yaml
applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /var/www/myapp
      min_instances: 1  # Always keep 1 instance
```

### 2. Increase Pool Size

For high traffic:

```yaml
pools:
  max_size: 20  # More Rails processes
  idle_timeout: 1800  # 30 minutes
```

### 3. Cache Headers

Optimize browser caching:

```yaml
static:
  directories:
    - path: /assets/
      root: public/assets/
      cache: 31536000  # 1 year for fingerprinted assets
    - path: /images/
      root: public/images/
      cache: 3600  # 1 hour for regular images
```

## Common Issues

### Rails doesn't start

```bash
# Check logs
LOG_LEVEL=debug navigator navigator.yml

# Verify Rails can start standalone
cd /var/www/myapp
bundle exec rails server
```

### Assets not loading

```bash
# Verify assets are precompiled
ls -la public/assets/

# Check static configuration
curl -v http://localhost:3000/assets/application.css
```

### Authentication not working

```bash
# Test htpasswd file
htpasswd -v /etc/navigator/htpasswd admin

# Check public paths
# Ensure assets are in public_paths list
```

## Variations

### With SSL Termination (Behind Proxy)

```yaml
applications:
  global_env:
    RAILS_ENV: production
    FORCE_SSL: "false"  # Proxy handles SSL
    
  tenants:
    - name: myapp
      path: /
      headers:
        X-Forwarded-Proto: "${HTTP_X_FORWARDED_PROTO}"
        X-Forwarded-For: "${HTTP_X_FORWARDED_FOR}"
```

### With Custom Domain

```yaml
server:
  listen: 3000
  hostname: myapp.com  # Your domain

applications:
  global_env:
    RAILS_HOST: myapp.com
```

## Next Steps

- Add [Redis for caching](with-redis.md)
- Add [Sidekiq for background jobs](with-sidekiq.md)
- Set up [systemd service](systemd.md)
- Deploy to [production](../deployment/production.md)