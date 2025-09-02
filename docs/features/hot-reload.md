# Configuration Hot Reload

Navigator supports live configuration reloading without restarting the server or interrupting active requests. This enables zero-downtime configuration updates in production.

## How It Works

When Navigator receives a `SIGHUP` signal, it:

1. **Parses new configuration** from the YAML file
2. **Updates routing rules** and application settings
3. **Starts new managed processes** if added to config
4. **Keeps existing Rails processes** running
5. **Applies changes** to new requests immediately

## Triggering Reload

### CLI Command (Recommended)
```bash
# Reload configuration
navigator -s reload
```

### Unix Signal
```bash
# Send SIGHUP signal directly
kill -HUP $(cat /tmp/navigator.pid)

# Or using process name
pkill -HUP navigator
```

### systemd Integration
```bash
# Reload via systemd
systemctl reload navigator
```

## What Gets Reloaded

### ✅ Reloaded Without Restart

- **Application routing rules**: Path mappings, tenant configurations
- **Authentication settings**: htpasswd files, public paths
- **Static file configurations**: Directories, caching, try files
- **Managed process definitions**: New processes started
- **Environment variables**: For new processes only
- **Fly-Replay rules**: Routing patterns and targets
- **Reverse proxy settings**: Target URLs and headers

### ❌ Requires Full Restart

- **Server listen port**: Cannot change port without restart
- **Process pool limits**: `max_size` changes require restart
- **Working directories**: Of existing Rails processes
- **PID file location**: Set at startup only

## Configuration Example

### Before Reload
```yaml
server:
  listen: 3000

applications:
  tenants:
    - name: app1
      path: /app1/
      working_dir: /var/www/app1

managed_processes:
  - name: redis
    command: redis-server
```

### After Configuration Update
```yaml
server:
  listen: 3000  # Same - no restart needed

applications:
  tenants:
    - name: app1
      path: /app1/
      working_dir: /var/www/app1
    
    # New tenant - will be available immediately
    - name: app2
      path: /app2/
      working_dir: /var/www/app2

managed_processes:
  - name: redis
    command: redis-server
  
  # New process - will be started
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /var/www/app1
```

### Reload Process
```bash
# 1. Update configuration file
vim /etc/navigator/config.yml

# 2. Reload Navigator
navigator -s reload

# 3. Verify new tenant is available
curl http://localhost:3000/app2/

# 4. Check new process is running
ps aux | grep sidekiq
```

## Use Cases

### 1. Adding New Tenants

Add new applications without downtime:

```bash
# Add tenant to config
echo "    - name: new-client
      path: /clients/new-client/
      working_dir: /var/www/new-client" >> config.yml

# Reload
navigator -s reload

# New tenant immediately available
curl http://localhost:3000/clients/new-client/
```

### 2. Authentication Updates

Update htpasswd files and reload:

```bash
# Add new user
htpasswd /etc/navigator/htpasswd newuser

# Update public paths in config
# ...

# Reload authentication settings
navigator -s reload
```

### 3. Static File Changes

Add new static file directories:

```yaml
# Add to config
static:
  directories:
    - path: /assets/
      root: public/assets/
    - path: /uploads/    # New directory
      root: storage/uploads/
      cache: 3600
```

```bash
# Reload to serve new directory
navigator -s reload
```

### 4. Managed Process Updates

Add background services without restart:

```yaml
managed_processes:
  - name: redis
    command: redis-server
  - name: worker-queue      # New worker
    command: ./worker.sh
    auto_restart: true
```

```bash
# Start new managed process
navigator -s reload

# Verify it's running
ps aux | grep worker
```

## Production Workflow

### Safe Deployment Pattern

```bash
#!/bin/bash
# production-reload.sh

# 1. Backup current config
cp /etc/navigator/config.yml /etc/navigator/config.yml.backup

# 2. Validate new configuration
if navigator --validate /tmp/new-config.yml; then
  echo "✓ Configuration valid"
else
  echo "✗ Configuration invalid, aborting"
  exit 1
fi

# 3. Update configuration
cp /tmp/new-config.yml /etc/navigator/config.yml

# 4. Reload Navigator
navigator -s reload

# 5. Verify reload succeeded
if ps aux | grep -q navigator; then
  echo "✓ Reload successful"
else
  echo "✗ Reload failed, restoring backup"
  cp /etc/navigator/config.yml.backup /etc/navigator/config.yml
  systemctl restart navigator
  exit 1
fi

# 6. Test new configuration
curl -f http://localhost:3000/health || {
  echo "✗ Health check failed"
  exit 1
}

echo "✓ Production reload completed successfully"
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
name: Deploy Configuration

on:
  push:
    paths:
      - 'config/production.yml'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Validate configuration
        run: navigator --validate config/production.yml
      
      - name: Deploy to server
        run: |
          scp config/production.yml server:/tmp/new-config.yml
          ssh server 'sudo ./production-reload.sh'
```

## Monitoring Reloads

### Logging

Navigator logs reload events:

```bash
# Monitor reload events
tail -f /var/log/navigator.log | grep -E "(reload|SIGHUP)"
```

**Example log output**:
```
INFO Received SIGHUP, reloading configuration
INFO Configuration reloaded successfully
INFO Starting new managed process name=sidekiq
INFO New tenant added name=app2 path=/app2/
```

### Health Checks

Verify reload success:

```bash
#!/bin/bash
# check-reload.sh

# Send reload signal
navigator -s reload

# Wait for reload to complete
sleep 2

# Check if Navigator is still running
if ! ps aux | grep -q navigator; then
  echo "ERROR: Navigator stopped after reload"
  exit 1
fi

# Test application response
if curl -f http://localhost:3000/health; then
  echo "SUCCESS: Reload completed"
else
  echo "ERROR: Application not responding after reload"
  exit 1
fi
```

### Metrics Collection

Track reload frequency and success:

```bash
# Count reloads in logs
grep "reloading configuration" /var/log/navigator.log | wc -l

# Check for reload errors
grep -E "(reload.*error|failed.*reload)" /var/log/navigator.log
```

## Troubleshooting

### Reload Doesn't Take Effect

**Problem**: Configuration changes not applied after reload

**Causes**:
- Invalid YAML syntax
- File permissions issues
- Navigator didn't receive signal

**Solutions**:
```bash
# 1. Validate configuration
navigator --validate config.yml

# 2. Check file permissions
ls -la config.yml

# 3. Verify Navigator is running
ps aux | grep navigator

# 4. Check PID file exists
cat /tmp/navigator.pid

# 5. Send signal manually
kill -HUP $(cat /tmp/navigator.pid)
```

### Process Fails to Start After Reload

**Problem**: New managed process doesn't start

**Causes**:
- Command not found
- Working directory doesn't exist
- Environment variables missing

**Solutions**:
```bash
# Test command manually
cd /working/directory
./command-name

# Check Navigator logs
tail -f /var/log/navigator.log

# Verify working directory exists
ls -la /working/directory
```

### Existing Requests Affected

**Problem**: Active requests fail after reload

**This should NOT happen** - reload is designed to be zero-downtime. If this occurs:

1. **Check configuration**: Ensure no breaking changes
2. **Review logs**: Look for error messages
3. **Consider restart**: May need full restart for certain changes

## Best Practices

### 1. Always Validate First
```bash
# Never reload without validation
navigator --validate config.yml && navigator -s reload
```

### 2. Use Configuration Management
```bash
# Version control your configs
git add config/production.yml
git commit -m "Add new tenant"
git push

# Deploy with automation
ansible-playbook deploy-config.yml
```

### 3. Monitor Reload Success
```bash
# Add to monitoring
curl -f http://localhost:3000/health || alert "Navigator health check failed"
```

### 4. Gradual Rollout
```yaml
# Add one tenant at a time
tenants:
  - name: existing-app
    path: /
  - name: new-app-pilot  # Test with limited users first
    path: /pilot/
```

## Limitations

- **Port changes**: Require full restart
- **Process pool size**: Changes need restart
- **Existing processes**: Keep running with old config
- **SSL certificates**: May need restart depending on implementation

## See Also

- [Signal Handling](../reference/signals.md)
- [Process Management](process-management.md)
- [Configuration Reference](../configuration/yaml-reference.md)
- [Production Deployment](../deployment/production.md)