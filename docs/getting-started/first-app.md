# Your First Rails Application

This guide walks you through serving your first Rails application with Navigator.

## Prerequisites

- Navigator installed ([Installation Guide](installation.md))
- A Rails application
- Basic understanding of YAML

## Step 1: Create Basic Configuration

Create a file named `navigator.yml` in your Rails application root:

```yaml title="navigator.yml"
server:
  listen: 3000
  public_dir: ./public

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: .
```

This minimal configuration:
- Listens on port 3000
- Serves static files from `./public`
- Routes all requests to your Rails app

## Step 2: Start Navigator

From your Rails application directory:

```bash
# If navigator is in your PATH
navigator navigator.yml

# Or specify the full path
/path/to/navigator navigator.yml
```

You should see output like:

```
INFO Starting Navigator listen=3000
INFO Configuration loaded tenants=1
INFO Server ready url=http://localhost:3000
```

## Step 3: Test Your Application

Open http://localhost:3000 in your browser. Navigator will:

1. Receive the request
2. Start your Rails application (first request only)
3. Proxy the request to Rails
4. Return the response

The first request may take a few seconds as Rails starts up.

## Step 4: Add Static File Serving

Improve performance by serving assets directly:

```yaml title="navigator.yml" hl_lines="5-10"
server:
  listen: 3000
  public_dir: ./public

  # Cache static assets
  cache_control:
    overrides:
      - path: /assets/
        max_age: 24h
  allowed_extensions: [css, js, png, jpg, gif, ico]

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: .
```

Now Navigator serves static files directly without starting Rails.

## Step 5: Add Authentication (Optional)

Protect your application with basic authentication:

```yaml title="navigator.yml" hl_lines="5-14"
server:
  listen: 3000
  public_dir: ./public

  # Cache static assets
  cache_control:
    overrides:
      - path: /assets/
        max_age: 24h
  allowed_extensions: [css, js, png, jpg, gif, ico]

auth:
  enabled: true
  htpasswd: ./htpasswd
  public_paths:
    - /assets/
    - /favicon.ico

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: .
```

Create an htpasswd file:

```bash
# Using htpasswd command (if available)
htpasswd -c htpasswd admin

# Or using online generator
echo "admin:$apr1$8QzHdF3N$Ht4rg7RHVV0000000000/" > htpasswd
```

## Step 6: Environment Variables

Pass environment variables to your Rails app:

```yaml title="navigator.yml" hl_lines="22-25"
server:
  listen: 3000
  public_dir: ./public

  # Cache static assets
  cache_control:
    overrides:
      - path: /assets/
        max_age: 24h
  allowed_extensions: [css, js, png, jpg, gif, ico]

auth:
  enabled: true
  htpasswd: ./htpasswd
  public_paths:
    - /assets/
    - /favicon.ico

applications:
  global_env:
    RAILS_ENV: production
    SECRET_KEY_BASE: your-secret-key-here

  tenants:
    - name: myapp
      path: /
      working_dir: .
```

## Common Issues and Solutions

### Rails doesn't start

Check the Rails application logs:

```bash
# Navigator shows Rails startup errors in its output
# Also check Rails logs
tail -f log/production.log
```

### Port already in use

Change the port in configuration:

```yaml
server:
  listen: 3001  # Different port
```

### Permission denied

Ensure Navigator can access your Rails directory:

```bash
# Check permissions
ls -la /path/to/rails/app

# Fix if needed
chmod -R 755 /path/to/rails/app
```

### Slow first request

This is normal - Rails takes time to start. Navigator will keep it running for subsequent requests. To prestart Rails:

```yaml
applications:
  tenants:
    - name: myapp
      path: /
      working_dir: .
      min_instances: 1  # Keep 1 instance always running
```

## What You've Learned

✅ Created a basic Navigator configuration
✅ Started Navigator with your Rails app
✅ Added static file serving for better performance
✅ Protected your app with authentication
✅ Configured environment variables

## Next Steps

- [Explore configuration options](basic-config.md)
- [Add Redis and background jobs](../examples/with-redis.md)
- [Set up multiple applications](../examples/multi-tenant.md)
- [Deploy to production](../deployment/production.md)