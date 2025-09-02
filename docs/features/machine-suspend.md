# Machine Suspension

Navigator can automatically suspend Fly.io machines after periods of inactivity, helping reduce costs while maintaining responsive applications.

## Overview

Machine suspension allows Navigator to:
- **Monitor request activity** across all tenants
- **Detect idle periods** when no requests are being processed
- **Automatically suspend machines** after configurable timeout
- **Wake machines instantly** when new requests arrive

This feature is specifically designed for Fly.io's machine architecture and provides significant cost savings for applications with variable traffic patterns.

## Configuration

### Basic Setup

```yaml
# Enable machine suspension
suspend:
  enabled: true
  idle_timeout: 300  # Suspend after 5 minutes of inactivity
```

### Production Configuration

```yaml
suspend:
  enabled: true
  idle_timeout: 600        # 10 minutes for production
  check_interval: 30       # Check every 30 seconds
  grace_period: 60         # Wait 60 seconds before suspend
```

### Development Configuration

```yaml
# Disable suspension in development
suspend:
  enabled: false
```

## How It Works

### Activity Tracking

Navigator tracks activity across all components:

1. **HTTP Requests**: Any incoming request resets idle timer
2. **Rails Processes**: Active Rails processes prevent suspension
3. **Managed Processes**: Running background jobs prevent suspension
4. **WebSocket Connections**: Open connections maintain activity

### Suspension Process

When idle timeout is reached:

1. **Final Activity Check**: Verify no active requests or processes
2. **Graceful Shutdown**: Stop Rails processes cleanly
3. **Resource Cleanup**: Clean up PID files and connections
4. **Machine Suspension**: Call Fly.io API to suspend machine

### Wake Process

When a request arrives at a suspended machine:

1. **Automatic Wake**: Fly.io automatically starts the machine
2. **Navigator Restart**: systemd or process manager restarts Navigator
3. **Process Recovery**: Rails processes start on-demand
4. **Request Processing**: Original request is processed normally

## Fly.io Integration

### Required Environment Variables

```bash
# Set by Fly.io automatically
export FLY_APP_NAME="myapp"
export FLY_MACHINE_ID="1234567890abcdef"
export FLY_API_TOKEN="fly_api_token_here"  # For machine control
```

### Fly.io Configuration

```toml title="fly.toml"
[machines]
  auto_start = true
  auto_stop = true
  min_machines_running = 0  # Allow all machines to suspend

[env]
  SUSPEND_ENABLED = "true"
  SUSPEND_TIMEOUT = "600"
```

## Use Cases

### Cost Optimization

**Scenario**: Development or staging environments with sporadic usage

```yaml
# Aggressive suspension for cost savings
suspend:
  enabled: true
  idle_timeout: 180     # 3 minutes
  check_interval: 15    # Check every 15 seconds
```

**Benefits**:
- Machines suspend during off-hours
- Wake instantly when needed
- Significant cost reduction for low-traffic environments

### Global Applications

**Scenario**: Applications deployed across multiple regions

```yaml
# Regional machines suspend independently
suspend:
  enabled: true
  idle_timeout: 900     # 15 minutes per region
  check_interval: 60
```

**Benefits**:
- Regional machines suspend during local off-hours
- Traffic patterns vary by timezone
- Optimal resource utilization globally

### Background Job Processing

**Scenario**: Applications with periodic background jobs

```yaml
# Allow suspension between job runs
suspend:
  enabled: true
  idle_timeout: 300     # 5 minutes between jobs
  
managed_processes:
  - name: cron-jobs
    command: bundle
    args: [exec, rake, cron:run]
    auto_restart: false  # Don't prevent suspension
```

## Monitoring Suspension

### Activity Logs

Navigator logs suspension activity:

```bash
# Monitor suspension events
tail -f /var/log/navigator.log | grep -E "(suspend|wake|idle)"
```

**Example log output**:
```
INFO Activity tracker started idle_timeout=300s
DEBUG Request activity recorded path=/users method=GET
DEBUG Idle check passed activity_within=45s
INFO No activity detected for 300s, preparing for suspension
INFO Gracefully stopping Rails processes for suspension
INFO Machine suspension initiated machine_id=1234567890abcdef
```

### Metrics Collection

```bash
#!/bin/bash
# Suspension metrics

# Count suspension events today
suspension_count=$(journalctl -u navigator --since today | grep -c "suspension initiated")
echo "machine_suspensions_today $suspension_count"

# Current idle time
last_activity=$(journalctl -u navigator | grep "activity recorded" | tail -1 | awk '{print $1" "$2}')
if [ -n "$last_activity" ]; then
    idle_seconds=$(( $(date +%s) - $(date -d "$last_activity" +%s) ))
    echo "machine_idle_seconds $idle_seconds"
fi

# Suspension status
if journalctl -u navigator --since "1 hour ago" | grep -q "suspension initiated"; then
    echo "machine_suspended 1"
else
    echo "machine_suspended 0"
fi
```

## Troubleshooting

### Machine Won't Suspend

**Issue**: Machine stays active despite no traffic

**Causes**:
- Active Rails processes
- Running managed processes
- Open WebSocket connections
- Ongoing background jobs

**Debug Steps**:
```bash
# Check active processes
ps aux | grep -E "(rails|ruby|navigator)"

# Check network connections
netstat -an | grep ESTABLISHED

# Check Navigator logs
journalctl -u navigator | grep -E "(idle|activity|suspend)"

# Manual activity check
LOG_LEVEL=debug navigator -s reload
```

### Suspension Takes Too Long

**Issue**: Machine doesn't suspend within expected timeframe

**Solutions**:
```yaml
# Reduce grace period
suspend:
  grace_period: 30      # Faster suspension

# More frequent checks
suspend:
  check_interval: 15    # Check more often
```

### Frequent Wake/Sleep Cycles

**Issue**: Machine suspends and wakes repeatedly

**Causes**:
- Health checks from load balancers
- Monitoring systems making requests
- Scheduled tasks running too frequently

**Solutions**:
```yaml
# Increase idle timeout
suspend:
  idle_timeout: 900     # 15 minutes

# Configure health checks to avoid suspension
# Or use Fly.io's built-in health checks
```

### Rails Processes Don't Stop

**Issue**: Rails processes prevent suspension

**Solutions**:
```bash
# Check for stuck processes
ps aux | grep rails

# Force cleanup if needed
pkill -f "rails server"

# Check for long-running requests
netstat -an | grep :400[0-9]
```

## Best Practices

### 1. Timeout Configuration

```yaml
# Development - aggressive suspension
suspend:
  idle_timeout: 180     # 3 minutes

# Staging - moderate suspension  
suspend:
  idle_timeout: 600     # 10 minutes

# Production - conservative suspension
suspend:
  idle_timeout: 1800    # 30 minutes
```

### 2. Health Check Coordination

```yaml
# Configure health checks to not prevent suspension
# Use Fly.io's health checks instead of external monitors

# In fly.toml
[http_service.checks.alive]
  path = "/up"
  interval = "30s"
  timeout = "5s"
```

### 3. Background Job Handling

```yaml
# Jobs that should not prevent suspension
managed_processes:
  - name: cleanup-job
    command: bundle
    args: [exec, rake, cleanup:run]
    auto_restart: false   # Allow suspension after completion

# Jobs that should prevent suspension  
managed_processes:
  - name: critical-worker
    command: bundle
    args: [exec, sidekiq]
    auto_restart: true    # Keep machine active
```

### 4. Monitoring Integration

```bash
# Alert on unexpected suspension patterns
if [ "$suspension_count" -gt 10 ]; then
    echo "WARNING: High suspension frequency - check for issues"
fi

# Monitor wake latency
wake_time=$(journalctl -u navigator | grep "wake" | tail -1 | awk '{print $NF}')
echo "machine_wake_time_seconds $wake_time"
```

## See Also

- [Fly.io Deployment](../deployment/fly-io.md)
- [Process Management](process-management.md)
- [Configuration Reference](../configuration/yaml-reference.md)
- [Monitoring Setup](../deployment/monitoring.md)