# Signal Handling

Navigator responds to standard Unix signals for process control, configuration management, and graceful shutdown. This enables seamless integration with system service managers and deployment workflows.

## Supported Signals

| Signal | Action | Description | Use Case |
|--------|--------|-------------|----------|
| `SIGHUP` | Reload configuration | Live config reload without restart | Configuration updates |
| `SIGTERM` | Graceful shutdown | Stop cleanly, finish active requests | Production shutdowns |
| `SIGINT` | Graceful shutdown | Same as SIGTERM (Ctrl+C) | Development/manual stop |
| `SIGQUIT` | Immediate shutdown | Force stop all processes | Emergency shutdown |

## Signal Usage

### Send Signals to Navigator

```bash
# Get Navigator's process ID
PID=$(cat /tmp/navigator.pid)

# Send signals using kill
kill -HUP $PID    # Reload configuration
kill -TERM $PID   # Graceful shutdown  
kill -INT $PID    # Graceful shutdown (same as TERM)
kill -QUIT $PID   # Immediate shutdown

# Or using signal names
kill -SIGHUP $PID
kill -SIGTERM $PID
kill -SIGINT $PID
kill -SIGQUIT $PID
```

### CLI Signal Commands

Navigator provides convenient CLI commands for signal operations:

```bash
# Reload configuration
navigator -s reload

# Graceful shutdown
navigator -s stop

# Immediate shutdown  
navigator -s quit
```

## SIGHUP - Configuration Reload

Reloads configuration without restarting Navigator or interrupting active requests.

### Behavior

1. **Parse new configuration** from YAML file
2. **Update routing rules** and application settings
3. **Start new managed processes** if added to config
4. **Keep existing Rails processes** running
5. **Apply changes** to new requests only

### Usage Examples

```bash
# Edit configuration
vim /etc/navigator/config.yml

# Reload without restart
kill -HUP $(cat /tmp/navigator.pid)

# Or using CLI
navigator -s reload

# Or using systemd
systemctl reload navigator
```

### What Gets Reloaded

✅ **Reloaded without restart**:
- Application routing rules
- Authentication settings  
- Static file configurations
- Managed process definitions
- Environment variables for new processes

❌ **Requires restart**:
- Server listen port
- Process pool limits (max_size)
- Working directories of existing processes

### Example Reload Workflow

```bash
# 1. Check current configuration
curl http://localhost:3000/health

# 2. Update configuration
cat >> config/navigator.yml << EOF
managed_processes:
  - name: new-worker
    command: ./worker.sh
    auto_restart: true
EOF

# 3. Reload configuration
navigator -s reload

# 4. Verify new process started
ps aux | grep worker
```

### Reload Logging

```bash
# Navigator logs reload events
tail -f /var/log/navigator.log

# Example output:
# INFO Received SIGHUP, reloading configuration
# INFO Configuration reloaded successfully  
# INFO Starting new managed process name=new-worker
```

## SIGTERM - Graceful Shutdown

Stops Navigator cleanly, allowing active requests to complete.

### Behavior

1. **Stop accepting new requests** on listen port
2. **Wait for active requests** to complete (with timeout)
3. **Stop Rails processes** gracefully (SIGTERM to each)
4. **Stop managed processes** in reverse order
5. **Clean up PID files** and resources
6. **Exit** with code 0

### Usage Examples

```bash
# Graceful shutdown
kill -TERM $(cat /tmp/navigator.pid)

# Or using CLI
navigator -s stop

# Or using systemd
systemctl stop navigator

# Or using Docker
docker stop navigator-container
```

### Shutdown Timeout

Navigator waits for processes to stop gracefully, with timeouts:

| Process Type | Timeout | Behavior After Timeout |
|--------------|---------|----------------------|
| **Rails processes** | 30 seconds | Send SIGKILL |
| **Managed processes** | 10 seconds | Send SIGKILL |
| **HTTP requests** | 5 seconds | Close connections |

### Example Graceful Shutdown

```bash
# Start Navigator
navigator config.yml &
NAVIGATOR_PID=$!

# Simulate some requests
curl http://localhost:3000/ &
curl http://localhost:3000/ &

# Graceful shutdown
kill -TERM $NAVIGATOR_PID

# Wait for shutdown
wait $NAVIGATOR_PID
echo "Navigator stopped gracefully"
```

### Shutdown Logging

```bash
# Example shutdown log output:
# INFO Received SIGTERM, starting graceful shutdown
# INFO Waiting for active requests to complete  
# INFO Stopping Rails processes
# INFO Process stopped app=main pid=12345
# INFO Stopping managed processes
# INFO Process stopped name=redis pid=12346
# INFO Cleanup complete, exiting
```

## SIGINT - Interactive Shutdown

Same as SIGTERM, triggered by Ctrl+C in interactive mode.

### Usage Examples

```bash
# Start Navigator in foreground
navigator config.yml

# Press Ctrl+C to trigger SIGINT
^C
# Output: Received SIGINT, starting graceful shutdown
```

### Development Workflow

```bash
# Start in development with debug logging
LOG_LEVEL=debug navigator config/dev.yml

# Make changes, then stop with Ctrl+C
^C

# Restart with new configuration
LOG_LEVEL=debug navigator config/dev.yml
```

## SIGQUIT - Immediate Shutdown

Forces immediate shutdown without waiting for requests or graceful process termination.

### Behavior

1. **Immediately stop** accepting requests
2. **Send SIGKILL** to all Rails processes
3. **Send SIGKILL** to all managed processes  
4. **Clean up PID files**
5. **Exit immediately** with code 130

### Usage Examples

```bash
# Force immediate shutdown
kill -QUIT $(cat /tmp/navigator.pid)

# Or using CLI
navigator -s quit

# Emergency stop (last resort)
kill -KILL $(cat /tmp/navigator.pid)
```

### When to Use SIGQUIT

- **Emergency situations** where graceful shutdown hangs
- **Development** when processes are stuck
- **Testing** signal handling behavior
- **CI/CD pipelines** with strict timeouts

⚠️ **Warning**: SIGQUIT may cause:
- Connection errors for active clients
- Data corruption in Rails processes
- Incomplete cleanup of resources

## Integration with System Services

### systemd Integration

```ini title="/etc/systemd/system/navigator.service"
[Unit]
Description=Navigator Web Server
After=network.target

[Service]
Type=simple
PIDFile=/var/run/navigator.pid
ExecStart=/usr/local/bin/navigator /etc/navigator/config.yml
ExecReload=/bin/kill -HUP $MAINPID
KillSignal=SIGTERM
TimeoutStopSec=60
KillMode=mixed

[Install]
WantedBy=multi-user.target
```

**systemd Commands**:
```bash
# Start service
systemctl start navigator

# Reload configuration (sends SIGHUP)
systemctl reload navigator

# Graceful stop (sends SIGTERM)
systemctl stop navigator

# Restart service
systemctl restart navigator
```

### Docker Integration

```dockerfile
# Dockerfile with proper signal handling
FROM ruby:3.2-slim

COPY navigator /usr/local/bin/
COPY config/ /app/config/

# Navigator handles signals properly
CMD ["navigator", "/app/config/production.yml"]
```

**Docker Commands**:
```bash
# Start container
docker run -d --name navigator-app navigator:latest

# Reload configuration (SIGHUP)
docker kill --signal=HUP navigator-app

# Graceful stop (SIGTERM) - default for docker stop
docker stop navigator-app

# Force stop (SIGKILL)
docker kill navigator-app
```

### Kubernetes Integration

```yaml title="deployment.yaml"
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: navigator
        image: navigator:latest
        
        # Graceful shutdown configuration
        lifecycle:
          preStop:
            exec:
              command: ["navigator", "-s", "stop"]
        
        # Signal handling
        terminationGracePeriodSeconds: 60
```

## Process Management Integration

### Supervisor Integration

```ini title="/etc/supervisor/conf.d/navigator.conf"
[program:navigator]
command=/usr/local/bin/navigator /etc/navigator/config.yml
user=navigator
autostart=true
autorestart=true
stopsignal=TERM
stopwaitsecs=60

# Reload configuration
killasgroup=false
stopasgroup=false
```

### PM2 Integration

```json title="ecosystem.config.js"
{
  "apps": [{
    "name": "navigator",
    "script": "/usr/local/bin/navigator",
    "args": "/etc/navigator/config.yml",
    "kill_timeout": 60000,
    "kill_retry_time": 5000
  }]
}
```

```bash
# PM2 commands
pm2 start ecosystem.config.js
pm2 reload navigator  # Graceful reload
pm2 stop navigator    # Graceful stop
```

## Signal Testing and Debugging

### Test Signal Handling

```bash
#!/bin/bash
# Test script for signal handling

# Start Navigator in background
navigator config.yml &
NAVIGATOR_PID=$!
echo "Started Navigator (PID: $NAVIGATOR_PID)"

# Wait for startup
sleep 2

# Test SIGHUP (reload)
echo "Testing SIGHUP (reload)..."
kill -HUP $NAVIGATOR_PID
sleep 1

# Test SIGTERM (graceful shutdown)
echo "Testing SIGTERM (graceful shutdown)..."
kill -TERM $NAVIGATOR_PID

# Wait for shutdown
wait $NAVIGATOR_PID
echo "Navigator shutdown complete"
```

### Monitor Signal Handling

```bash
# Monitor Navigator logs for signal events
tail -f /var/log/navigator.log | grep -E "(signal|shutdown|reload)"

# Check process status
ps aux | grep navigator

# Verify PID file cleanup
ls -la /tmp/navigator.pid
```

### Debug Signal Issues

```bash
# Check if Navigator responds to signals
PID=$(cat /tmp/navigator.pid)
kill -0 $PID && echo "Process running" || echo "Process not running"

# Send test signal and check response
kill -HUP $PID
sleep 1
ps -p $PID -o pid,ppid,cmd

# Check log output
tail -n 10 /var/log/navigator.log
```

## Common Issues and Solutions

### Signal Not Received

**Problem**: Navigator doesn't respond to signals

**Causes**:
- Invalid PID file
- Process already stopped
- Permission denied

**Solutions**:
```bash
# Check PID file exists and is readable
ls -la /tmp/navigator.pid
cat /tmp/navigator.pid

# Verify process is running
PID=$(cat /tmp/navigator.pid)
kill -0 $PID

# Check process owner
ps -o user,pid,cmd -p $PID

# Use correct permissions
sudo kill -HUP $PID  # If running as different user
```

### Graceful Shutdown Hangs

**Problem**: SIGTERM doesn't complete shutdown

**Causes**:
- Rails processes not responding
- Long-running requests
- Managed processes stuck

**Solutions**:
```bash
# Check what processes are still running
ps aux | grep -E "(rails|ruby|navigator)"

# Check active connections
netstat -an | grep :3000

# Force shutdown if necessary
navigator -s quit

# Or kill individual processes
pkill -f "rails server"
```

### Configuration Reload Fails

**Problem**: SIGHUP doesn't reload configuration

**Causes**:
- Invalid YAML syntax
- File permissions
- Missing configuration file

**Solutions**:
```bash
# Validate configuration first
navigator --validate config.yml

# Check file permissions
ls -la config.yml

# Check Navigator logs for error details
tail -f /var/log/navigator.log | grep -E "(error|reload)"
```

## Security Considerations

### Signal Security

- **PID file permissions**: Secure PID file to prevent unauthorized signal sending
- **Process ownership**: Run Navigator as dedicated user
- **Signal validation**: Navigator validates signals before processing

```bash
# Secure PID file
chmod 600 /tmp/navigator.pid
chown navigator:navigator /tmp/navigator.pid

# Run as dedicated user
sudo -u navigator navigator config.yml
```

### Production Recommendations

1. **Use systemd**: Better signal handling and process management
2. **Monitor signals**: Log and monitor signal events
3. **Test graceful shutdown**: Verify shutdown works under load
4. **Set appropriate timeouts**: Balance graceful shutdown vs. availability

## See Also

- [CLI Reference](cli.md)
- [Process Management](../features/process-management.md)
- [Hot Reload Feature](../features/hot-reload.md)
- [Production Deployment](../deployment/production.md)