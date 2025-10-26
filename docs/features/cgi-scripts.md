# CGI Scripts

Navigator supports running CGI (Common Gateway Interface) scripts directly, without needing to start a full web application. This is ideal for lightweight endpoints that run standalone scripts, such as database synchronization, configuration updates, or status checks.

## Overview

CGI scripts provide a way to execute standalone scripts in response to HTTP requests. Navigator's CGI support includes:

- **User switching**: Run scripts as specific Unix users (requires root)
- **Method filtering**: Restrict scripts to specific HTTP methods
- **Environment variables**: Pass custom environment to scripts
- **Automatic reload**: Reload configuration after script execution
- **Timeout control**: Set execution time limits

## Configuration

Add CGI scripts to the `server` section of your configuration:

```yaml
server:
  listen: 3000
  cgi_scripts:
    - path: /admin/sync
      script: /opt/app/script/sync_databases.rb
      method: POST
      user: appuser
      group: appgroup
      timeout: 5m
      reload_config: config/navigator.yml
      env:
        RAILS_DB_VOLUME: /mnt/db
        RAILS_ENV: production
```

### Configuration Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | URL path to match (exact match) |
| `script` | string | Yes | Absolute path to the executable CGI script |
| `method` | string | No | HTTP method restriction (GET, POST, etc.). Empty = all methods |
| `user` | string | No | Unix user to run script as (empty = current user) |
| `group` | string | No | Unix group to run script as (empty = user's primary group) |
| `allowed_users` | list | No | Usernames allowed to access this script. Empty = all authenticated users |
| `env` | map | No | Additional environment variables to set |
| `reload_config` | string | No | Config file to reload after successful execution |
| `timeout` | string | No | Execution timeout (e.g., "30s", "5m"). Zero = no timeout |

## Example: Showcase Database Sync

Here's how to configure Navigator to handle Showcase's database sync endpoints without starting a Rails tenant:

```yaml
server:
  listen: 3000
  hostname: smooth.fly.dev
  cgi_scripts:
    # Database sync endpoint - updates index database from S3
    - path: /showcase/index_update
      script: /rails/script/sync_databases_s3.rb
      method: POST
      user: rails
      group: rails
      timeout: 10m
      reload_config: config/navigator.yml
      env:
        RAILS_DB_VOLUME: /mnt/db
        RAILS_ENV: production
        AWS_REGION: us-east-1

    # Index modification time check
    - path: /showcase/index_date
      script: /rails/script/check_index_date.rb
      method: GET
      user: rails
      timeout: 5s
      env:
        RAILS_DB_VOLUME: /mnt/db
```

## CGI Environment Variables

Navigator automatically sets standard CGI environment variables (RFC 3875):

| Variable | Description |
|----------|-------------|
| `GATEWAY_INTERFACE` | Always "CGI/1.1" |
| `SERVER_PROTOCOL` | HTTP protocol version (e.g., "HTTP/1.1") |
| `SERVER_SOFTWARE` | Always "Navigator" |
| `REQUEST_METHOD` | HTTP method (GET, POST, etc.) |
| `QUERY_STRING` | URL query parameters |
| `SCRIPT_NAME` | Request path |
| `SERVER_NAME` | Server hostname |
| `REMOTE_ADDR` | Client IP address |
| `CONTENT_TYPE` | Request Content-Type header |
| `CONTENT_LENGTH` | Request Content-Length header |
| `HTTP_*` | All HTTP headers as HTTP_* variables |

Additionally, Navigator passes all custom environment variables defined in the `env` section.

## Script Requirements

CGI scripts must:

1. **Be executable**: Set execute permission (`chmod +x script.rb`)
2. **Have shebang**: First line should be `#!/usr/bin/env ruby` (or appropriate interpreter)
3. **Output CGI headers**: Print headers followed by blank line before body
4. **Exit with status**: Exit 0 for success, non-zero for error

### Example Script

```ruby
#!/usr/bin/env ruby

# Read environment
db_volume = ENV['RAILS_DB_VOLUME'] || 'db'

# Perform work
result = sync_databases(db_volume)

# Output CGI response
puts "Content-Type: text/plain"
puts "Status: 200 OK"
puts ""
puts "Sync completed: #{result}"

exit 0
```

### Status Codes

CGI scripts can set HTTP status codes via the `Status` header:

```ruby
puts "Status: 404 Not Found"
puts "Status: 500 Internal Server Error"
puts "Status: 201 Created"
```

## User Switching (Unix Only)

CGI scripts can run as different users for security isolation:

```yaml
cgi_scripts:
  - path: /admin/backup
    script: /opt/scripts/backup.sh
    user: backup    # Run as 'backup' user
    group: backup   # Run as 'backup' group
```

**Requirements:**
- Navigator must be running as root
- The specified user and group must exist on the system
- Works on Unix/Linux only (not Windows)

**Without user switching:**
- Scripts run as the same user as Navigator
- Simpler but less secure

## Configuration Reload

The `reload_config` feature automatically reloads Navigator's configuration after successful script execution:

```yaml
cgi_scripts:
  - path: /admin/update
    script: /opt/scripts/update_htpasswd.rb
    reload_config: config/navigator.yml
```

### How It Works

Configuration reload is triggered **only if**:
1. The `reload_config` field is specified, AND
2. Either:
   - The config file path is different from the currently loaded config, OR
   - The config file was modified during script execution

This avoids unnecessary reloads when nothing has changed.

**Note**: CGI scripts and [lifecycle hooks](./lifecycle-hooks.md#configuration-reload) share the same smart reload logic, ensuring consistent behavior across Navigator.

### Use Cases

- **Authentication updates**: Reload after updating htpasswd file
- **Tenant changes**: Reload after adding/removing tenants
- **Dynamic configuration**: Scripts that generate new configuration

### Example: Update Authentication

```yaml
cgi_scripts:
  - path: /admin/update_users
    script: /opt/scripts/update_users.rb
    method: POST
    user: admin
    reload_config: config/navigator.yml
    env:
      HTPASSWD_FILE: /etc/navigator/htpasswd
```

```ruby
#!/usr/bin/env ruby

# Update htpasswd file
htpasswd_file = ENV['HTPASSWD_FILE']
update_users(htpasswd_file)

# Output success
puts "Content-Type: text/plain"
puts ""
puts "Users updated. Configuration will be reloaded."

# Exit success - Navigator will reload config automatically
exit 0
```

## Request Routing

CGI scripts are evaluated in the request handling order:

1. Health check endpoint (`/up`)
2. Authentication check
3. Rewrites and redirects
4. **CGI scripts** ← You are here
5. Reverse proxies
6. Static files
7. Web application proxy

This means:
- CGI scripts are subject to authentication (unless path is in `public_paths`)
- CGI scripts run before reverse proxies
- CGI scripts take precedence over web applications

## Timeout Handling

Set timeouts to prevent long-running scripts from blocking:

```yaml
cgi_scripts:
  - path: /api/sync
    script: /opt/scripts/sync.rb
    timeout: 2m  # Kill after 2 minutes
```

If a script exceeds its timeout:
- The script process is terminated
- HTTP 500 error is returned to the client
- Error is logged

## Error Handling

When a CGI script fails:

1. **Non-zero exit**: HTTP 500 returned, stderr logged
2. **Timeout**: Process killed, HTTP 500 returned
3. **Not found**: Error logged at startup, requests return 404
4. **Permission denied**: Error logged at startup

Check Navigator logs for CGI execution details:

```
level=INFO msg="Executing CGI script" script=/opt/sync.rb method=POST path=/admin/sync user=appuser
level=INFO msg="CGI script completed" script=/opt/sync.rb duration=1.2s
level=ERROR msg="CGI script execution failed" script=/opt/sync.rb error="exit status 1"
```

## Security Considerations

### User Permissions

When running scripts as different users:

```yaml
cgi_scripts:
  - path: /admin/restricted
    script: /opt/scripts/admin_task.rb
    user: admin  # Runs with elevated privileges
```

**Best practices:**
- Run scripts with minimum required privileges
- Don't run scripts as root unless absolutely necessary
- Use Navigator's authentication to control access
- Validate all script inputs

### Authentication

CGI scripts respect Navigator's authentication and support fine-grained access control:

```yaml
auth:
  enabled: true
  htpasswd: /etc/navigator/htpasswd
  public_paths:
    - /public/*
    - /health

cgi_scripts:
  # Restricted to specific users
  - path: /admin/sync
    script: /opt/sync.rb
    allowed_users:
      - admin
      - operator

  # Available to all authenticated users
  - path: /admin/status
    script: /opt/status.rb

  # Public endpoint (no authentication required)
  - path: /public/health
    script: /opt/health.sh
```

**Access Control Behavior:**

- **With `allowed_users`**: Only specified usernames can access the script (returns 403 Forbidden for other authenticated users)
- **Without `allowed_users`**: All authenticated users can access the script
- **Public paths**: Scripts on paths listed in `public_paths` can be accessed without authentication

**Example: Multi-level access control**

```yaml
auth:
  enabled: true
  htpasswd: /etc/navigator/htpasswd

cgi_scripts:
  # Admin-only: Database operations
  - path: /admin/db_sync
    script: /opt/scripts/sync_db.rb
    allowed_users:
      - admin

  # Operators can restart services
  - path: /admin/restart
    script: /opt/scripts/restart_service.sh
    allowed_users:
      - admin
      - operator
      - oncall

  # All authenticated users can check status
  - path: /admin/status
    script: /opt/scripts/check_status.sh
    # No allowed_users = all authenticated users
```

## Performance

CGI scripts start a new process for each request:

- **Startup cost**: Fork + exec overhead (~5-50ms)
- **Memory**: Each execution is independent
- **Scalability**: Suitable for low-to-medium traffic

**When to use CGI:**
- Infrequent operations (database sync, admin tasks)
- Lightweight checks (status, health)
- One-off scripts that don't justify a full web app

**When not to use CGI:**
- High-frequency endpoints
- Real-time applications
- WebSocket connections
- Long-running connections

## Comparison with Web Applications

| Feature | CGI Scripts | Web Applications |
|---------|-------------|------------------|
| Startup | Per request | Once, then reused |
| Memory | Minimal | Higher (persistent process) |
| Performance | Good for infrequent | Good for frequent |
| State | Stateless | Can maintain state |
| Use case | Admin tasks, checks | Full applications |

## Examples

### Database Status Check

```yaml
cgi_scripts:
  - path: /health/db
    script: /opt/scripts/check_db.sh
    method: GET
    timeout: 5s
```

```bash
#!/bin/sh
if [ -f "$RAILS_DB_VOLUME/index.sqlite3" ]; then
  mtime=$(stat -f %Sm -t %Y-%m-%dT%H:%M:%SZ "$RAILS_DB_VOLUME/index.sqlite3")
  echo "Content-Type: text/plain"
  echo ""
  echo "OK: Last modified $mtime"
  exit 0
else
  echo "Status: 503 Service Unavailable"
  echo "Content-Type: text/plain"
  echo ""
  echo "ERROR: Database not found"
  exit 1
fi
```

### Webhook Handler

```yaml
cgi_scripts:
  - path: /webhooks/github
    script: /opt/scripts/github_webhook.rb
    method: POST
    user: webhook
    timeout: 30s
    env:
      GITHUB_SECRET: "${GITHUB_WEBHOOK_SECRET}"
```

### Configuration Generator

```yaml
cgi_scripts:
  - path: /admin/generate_config
    script: /opt/scripts/generate_config.rb
    method: POST
    user: admin
    reload_config: config/navigator.yml
```

## Troubleshooting

### Script Not Executing

1. Check execute permission: `ls -l /path/to/script.cgi`
2. Verify shebang is correct: `head -1 /path/to/script.cgi`
3. Test script manually: `sudo -u appuser /path/to/script.cgi`
4. Check Navigator logs for errors

### Permission Denied

- Navigator must run as root for user switching
- Script file must be readable by target user
- Script directory must be accessible

### Script Times Out

- Increase timeout: `timeout: 10m`
- Optimize script performance
- Consider moving to web application if consistently slow

### Config Not Reloading

- Verify `reload_config` path is correct
- Check file modification time after script runs
- Ensure script modifies config before exiting
- Look for reload messages in logs

## See Also

- [Lifecycle Hooks](./lifecycle-hooks.md) - Server and tenant lifecycle automation
- [Configuration Reference](../configuration/yaml-reference.md) - Full YAML configuration options
- [Authentication](../configuration/authentication.md) - Securing CGI endpoints
