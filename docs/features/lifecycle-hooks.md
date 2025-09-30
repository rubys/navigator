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
| `ready` | After Navigator starts listening | Notify monitoring systems, warm caches |
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

### 5. State Checkpoint

Checkpoint application state before shutdown:

```yaml
hooks:
  tenant:
    stop:
      - command: bundle
        args: ["exec", "rails", "runner", "AppState.checkpoint!"]
        timeout: 30s
```

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
- **ready**: Executes after server starts (can be slower)
- **idle**: Delays suspension (balance speed vs completeness)
- **resume**: Blocks request handling (keep fast)
- **tenant.start**: Delays tenant availability
- **tenant.stop**: Delays tenant shutdown (usually acceptable)

### Recommendations

1. **Keep start/resume hooks fast** (<5s)
2. **Use ready for slow initialization** (cache warming, etc.)
3. **Balance idle hook completeness vs timeout** (users waiting)
4. **Run long operations asynchronously** (background jobs)
5. **Use conditional logic** (skip if already synced)

## See Also

- [YAML Configuration Reference](../configuration/yaml-reference.md#hooks) - Complete hooks syntax
- [Machine Suspend](machine-suspend.md) - Auto-suspend configuration
- [Use Cases: Machine Auto-Suspend](../use-cases.md#use-case-2-machine-auto-suspend-flyio) - Real-world examples
- [Examples: Suspend/Stop](../../examples/suspend-stop-example.yml) - Working configuration
