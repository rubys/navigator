# Lifecycle Hooks

Navigator provides powerful lifecycle hooks that let you execute custom commands at key points in the application lifecycle. This enables integration with external services, data synchronization, monitoring, and custom automation workflows.

## Overview

Lifecycle hooks are shell commands that execute automatically when specific events occur. Navigator supports two types of hooks:

1. **Server Hooks** - Execute at Navigator lifecycle events (start, ready, idle, resume)
2. **Tenant Hooks** - Execute at tenant application lifecycle events (start, stop)

## Server Hooks

Server hooks execute at key Navigator lifecycle events.

### Hook Types

| Hook | When Executed | Common Use Cases |
|------|---------------|------------------|
| `start` | Before Navigator accepts requests | Database migrations, service initialization |
| `ready` | Asynchronously after Navigator starts listening **or after configuration reload** | Notify monitoring systems, warm caches, prerender content |
| `idle` | Before machine suspend/stop (Fly.io) | Upload databases to S3, checkpoint state |
| `resume` | After machine resume (Fly.io) | Download databases from S3, reconnect services |

### Configuration

```yaml
hooks:
  server:
    start:
      - command: /usr/local/bin/init-services.sh
        args: ["--migrate"]
        timeout: 2m

    ready:
      - command: curl
        args: ["-X", "POST", "https://monitoring.example.com/navigator/ready"]
        timeout: 5s

    idle:
      - command: /rails/script/backup-to-s3.sh
        args: ["--all"]
        timeout: 5m

    resume:
      - command: /rails/script/restore-from-s3.sh
        args: ["--all"]
        timeout: 3m
```

### Example: S3 Database Sync

This example shows automatic database synchronization with S3 storage before machine suspension:

```yaml
server:
  idle:
    action: suspend
    timeout: 20m

hooks:
  server:
    idle:
      - command: /usr/local/bin/sync-to-s3.sh
        args: ["upload", "/data/db"]
        timeout: 5m
    resume:
      - command: /usr/local/bin/sync-from-s3.sh
        args: ["download", "/data/db"]
        timeout: 3m
```

## Tenant Hooks

Tenant hooks execute when tenant applications start or stop. These can be defined as defaults (applying to all tenants) or per-tenant overrides.

### Hook Types

| Hook | When Executed | Common Use Cases |
|------|---------------|------------------|
| `start` | After tenant app process starts | Initialize tenant data, warm caches |
| `stop` | Before tenant app process stops | Backup tenant database, sync to storage |

### Default Tenant Hooks

Default hooks execute for all tenant applications:

```yaml
hooks:
  tenant:
    start:
      - command: /usr/local/bin/tenant-init.sh
        args: ["${database}"]
        timeout: 30s

    stop:
      - command: /usr/local/bin/tenant-backup.sh
        args: ["${database}"]
        timeout: 2m
```

### Per-Tenant Hooks

Individual tenants can override or supplement default hooks:

```yaml
applications:
  tenants:
    - path: /showcase/2025/boston/
      var:
        database: 2025-boston
      hooks:
        start:
          - command: /rails/bin/tenant-migrate.sh
            args: ["${database}"]
            timeout: 1m
        stop:
          - command: /rails/bin/tenant-checkpoint.sh
            args: ["${database}"]
            timeout: 30s
```

### Execution Order

When a tenant stops, hooks execute in this order:

1. **Default tenant stop hooks** (from `hooks.tenant.stop`)
2. **Tenant-specific stop hooks** (from `applications.tenants[].hooks.stop`)

Both sets of hooks execute sequentially. If one fails, the next still runs.

## Hook Configuration

Each hook entry supports these fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | ✓ | Command to execute (full path or PATH executable) |
| `args` | array | | Command arguments |
| `timeout` | string | | Maximum execution time (duration format: "30s", "5m") |
| `reload_config` | string | | Config file to reload after successful hook execution (server hooks only). Only reloads if file path differs OR file was modified during hook execution. |

### Configuration Reload

The `reload_config` field allows hooks to trigger Navigator configuration reload after successful execution. This is useful when hooks modify configuration files or related resources (like htpasswd files).

**Smart Reload Logic:**

Navigator only reloads configuration when necessary:

1. **Different config file**: If `reload_config` specifies a different path than currently loaded
2. **File modified during execution**: If the config file's modification time changed during hook execution

This avoids unnecessary reloads when nothing has changed, improving performance and reducing overhead.

**Example - Update htpasswd and reload:**

```yaml
hooks:
  server:
    ready:
      - command: /usr/local/bin/update-users.sh
        timeout: 30s
        reload_config: config/navigator.yml  # Reload to pick up htpasswd changes
```

**How it works:**

1. Navigator records start time before executing hook
2. Hook runs (e.g., updates `/etc/navigator/htpasswd`)
3. After successful completion, Navigator checks:
   - Is `reload_config` path different from current config? → Reload
   - Was config file modified after start time? → Reload
   - Otherwise → Skip reload (no changes detected)
4. If reload triggered, Navigator loads new config and updates all managers

**Common use cases:**

- Update htpasswd file and reload auth configuration
- Modify tenant list and reload tenant routing
- Update managed processes and restart/stop them
- Change static file configuration

**Note**: Only applies to **server hooks** (`start`, `ready`, `idle`, `resume`). Tenant hooks do not support `reload_config`.

### Variable Substitution

Hooks support variable substitution from tenant configuration:

```yaml
applications:
  env:
    DATABASE_URL: "sqlite3:///data/db/${database}.sqlite3"
    STORAGE_PATH: "/data/storage/${database}"

  tenants:
    - path: /showcase/2025/boston/
      var:
        database: "2025-boston"
        region: "us-east"

      hooks:
        stop:
          - command: /usr/local/bin/backup.sh
            args: ["${database}", "${region}", "${STORAGE_PATH}"]
            timeout: 2m
```

Variables are substituted from:
- Tenant `var` values
- Tenant `env` values
- Parent process environment

## Environment Variables

### Server Hooks

Server hooks receive Navigator's environment, including:
- All environment variables Navigator was started with
- `FLY_REGION`, `FLY_APP_NAME`, etc. (on Fly.io)

### Tenant Hooks

Tenant hooks receive the full tenant environment, including:
- All tenant `env` values
- Expanded template variables from `var`
- Navigator's parent environment

Example tenant environment:

```yaml
applications:
  env:
    RAILS_ENV: production
    DATABASE_URL: "sqlite3:///data/${database}.sqlite3"

  tenants:
    - path: /app/
      var:
        database: myapp
      env:
        TENANT_NAME: "My Application"

      hooks:
        stop:
          - command: /usr/local/bin/backup.sh
            # Receives:
            # RAILS_ENV=production
            # DATABASE_URL=sqlite3:///data/myapp.sqlite3
            # TENANT_NAME=My Application
            # database=myapp
```

## Error Handling

### Behavior on Failure

- **Logging**: Failed hooks log detailed error messages
- **Non-blocking**: Hook failures don't stop Navigator
- **Timeout**: Hooks exceeding timeout are terminated
- **Sequential**: Multiple hooks run in order; failures don't skip subsequent hooks
- **Reload on success only**: `reload_config` only triggers if hook exits with status 0

### Exit Codes

Hooks should return standard exit codes:
- `0` - Success
- `Non-zero` - Failure (logged but Navigator continues)

### Best Practices

1. **Set appropriate timeouts**: Prevent hooks from hanging indefinitely
2. **Handle errors gracefully**: Scripts should be idempotent
3. **Log verbosely**: Hook output captured in Navigator logs
4. **Test hooks independently**: Verify scripts work standalone
5. **Use absolute paths**: Don't rely on PATH for critical commands

## Use Cases

### 1. Database Synchronization (Fly.io)

Sync SQLite databases to S3 before machine suspension:

```yaml
server:
  idle:
    action: suspend
    timeout: 15m

hooks:
  server:
    idle:
      - command: /rails/script/sync-db-to-s3.sh
        args: ["upload", "--all"]
        timeout: 5m
    resume:
      - command: /rails/script/sync-db-from-s3.sh
        args: ["download", "--only-missing"]
        timeout: 2m

  tenant:
    stop:
      - command: /rails/script/sync-db-to-s3.sh
        args: ["upload", "--database", "${database}"]
        timeout: 2m
```

### 2. Monitoring Integration

Notify external monitoring when Navigator starts and machines change state:

```yaml
hooks:
  server:
    start:
      - command: curl
        args:
          - "-X"
          - "POST"
          - "https://monitoring.example.com/events"
          - "-d"
          - '{"event":"navigator_start","region":"${FLY_REGION}"}'
        timeout: 5s

    ready:
      - command: /usr/local/bin/notify-datadog.sh
        args: ["navigator.ready", "1", "${FLY_REGION}"]
        timeout: 5s

    idle:
      - command: /usr/local/bin/notify-datadog.sh
        args: ["navigator.suspend", "1", "${FLY_REGION}"]
        timeout: 5s
```

### 3. Database Migrations

Run database migrations when tenants start:

```yaml
hooks:
  tenant:
    start:
      - command: bundle
        args: ["exec", "rails", "db:migrate"]
        timeout: 2m
```

### 4. Cache Warming

Warm application caches after startup:

```yaml
hooks:
  server:
    ready:
      - command: /usr/local/bin/warm-cache.sh
        args: ["--preload-assets"]
        timeout: 30s

  tenant:
    start:
      - command: curl
        args: ["http://localhost:${port}/admin/cache/warm"]
        timeout: 10s
```

### 5. Content Prerendering After Config Updates

Use ready hooks to regenerate static content after configuration reloads:

```yaml
hooks:
  server:
    ready:
      - command: /rails/script/prerender.sh
        timeout: 10m
        # Runs after initial start AND after config reloads
        # (CGI reload, SIGHUP reload, or resume reload)

server:
  cgi_scripts:
    - path: /admin/update_config
      script: /rails/script/update_configuration.rb
      reload_config: config/navigator.yml  # Triggers ready hook after reload
```

**How it works:**

1. CGI script updates configuration files (htpasswd, maps, navigator.yml)
2. CGI script returns success (fast, <5 seconds)
3. Navigator detects navigator.yml changed → reloads configuration
4. **Ready hook executes asynchronously** (prerender runs in background)
5. Navigator continues serving requests with new config while prerender completes

**Benefits:**
- Fast cold starts (server listens immediately, initialization runs in background)
- Zero downtime (Navigator serves requests while hooks execute)
- Consistent behavior (same hook runs on initial start and reloads)
- Maintenance page served during initialization

### 6. State Checkpoint

Checkpoint application state before shutdown:

```yaml
hooks:
  tenant:
    stop:
      - command: bundle
        args: ["exec", "rails", "runner", "AppState.checkpoint!"]
        timeout: 30s
```

### 7. Dynamic User Management with Reload

Update htpasswd file and automatically reload authentication:

```yaml
# Script: /usr/local/bin/add-user.sh
#!/bin/bash
# Add user to htpasswd file
htpasswd -b /etc/navigator/htpasswd "$1" "$2"

# Touch config file to trigger reload
touch config/navigator.yml

hooks:
  server:
    ready:
      - command: /usr/local/bin/add-user.sh
        args: ["newuser", "password123"]
        timeout: 5s
        reload_config: config/navigator.yml  # Reload to pick up new user
```

**How it works:**

1. Hook executes script that updates htpasswd file
2. Script touches config file to update modification time
3. After successful execution, Navigator detects config file was modified
4. Navigator reloads configuration (including htpasswd file)
5. New user can authenticate immediately

**Smart reload behavior:**

- If config file not modified: No reload (efficient)
- If config file modified during hook: Reload triggered
- If reload_config path differs: Always reload

### 8. Maintenance Mode with Config Switching

Start Navigator with a minimal maintenance configuration, run initialization tasks, then switch to full configuration:

```yaml
# config/navigator-maintenance.yml
server:
  listen: 3000
  static:
    public_dir: public
    allowed_extensions: [html, css]

# Maintenance page configuration
maintenance:
  page: /503.html

hooks:
  server:
    ready:
      - command: ruby
        args: ["script/initialization.rb"]
        timeout: 5m
        reload_config: config/navigator.yml  # Switch to full config after init
```

**How it works:**
1. Navigator starts with `config/navigator-maintenance.yml`
2. Shows maintenance page (503.html) to all requests
3. Ready hook executes `script/initialization.rb` (database sync, cache warm, etc.)
4. After successful hook execution, Navigator checks if reload needed:
   - Config path `config/navigator.yml` differs from current → Reload triggered
   - Navigator loads full configuration and starts all tenants
5. Full application becomes available with all tenants and features

**Benefits:**
- Eliminates need for wrapper scripts
- Reduces memory footprint (no persistent Ruby process)
- Provides user-friendly maintenance page during initialization
- Automatic transition to full configuration
- Smart reload only when config path differs or file changes

## Debugging Hooks

### View Hook Execution

Hooks execution appears in Navigator logs:

```
level=INFO msg="Executing hook" type=server.idle command=/usr/local/bin/backup.sh
level=INFO msg="Hook completed" type=server.idle duration=2.3s
```

```
level=INFO msg="Executing hook" type=tenant.stop.default command=/rails/script/sync.sh
level=INFO msg="Hook output" type=tenant.stop.default output="Synced 3 databases"
```

### Common Issues

**Hook not executing:**
- Check command path is correct (use absolute paths)
- Verify file has execute permissions (`chmod +x`)
- Ensure timeout is sufficient

**Hook timing out:**
- Increase timeout value
- Check for hanging processes or network delays
- Add logging to script to identify bottlenecks

**Wrong environment variables:**
- Server hooks: Use Navigator's environment
- Tenant hooks: Use tenant's full environment
- Check variable substitution syntax: `${var}` not `$var`

## Performance Considerations

### Hook Timing

- **start**: Blocks Navigator startup (keep fast)
- **ready**: Executes asynchronously after server starts listening (doesn't block requests)
- **idle**: Delays suspension (balance speed vs completeness)
- **resume**: Blocks request handling (keep fast)
- **tenant.start**: Delays tenant availability
- **tenant.stop**: Delays tenant shutdown (usually acceptable)

### Recommendations

1. **Keep start/resume hooks fast** (<5s) - these block critical startup
2. **Use ready for slow initialization** - server serves maintenance page while hooks run
3. **Balance idle hook completeness vs timeout** (users waiting)
4. **Run long operations in ready hooks** - they execute asynchronously
5. **Use conditional logic** (skip if already synced)

## See Also

- [YAML Configuration Reference](../configuration/yaml-reference.md#hooks) - Complete hooks syntax
- [CGI Scripts](cgi-scripts.md) - Similar reload_config behavior for CGI scripts
- [Machine Suspend](machine-suspend.md) - Auto-suspend configuration
- [Use Cases: Machine Auto-Suspend](../use-cases.md#use-case-2-machine-auto-suspend-flyio) - Real-world examples
- [Examples: Suspend/Stop](../../examples/suspend-stop-example.yml) - Working configuration
