# Machine Suspension Configuration

Configure Navigator's machine suspension feature for automatic idle timeout and cost optimization on Fly.io and other cloud platforms.

## Basic Configuration

```yaml
suspend:
  enabled: true
  idle_timeout: 300     # Suspend after 5 minutes of inactivity
```

## Configuration Options

### enabled

**Type**: Boolean  
**Default**: `false`  
**Required**: No

Enable or disable machine suspension feature.

```yaml
# Enable suspension
suspend:
  enabled: true

# Disable suspension (default)
suspend:
  enabled: false
```

**Use Cases**:
- **Production**: Enable for cost optimization during low-traffic periods
- **Development**: Usually disabled for consistent availability  
- **Staging**: Enable for cost savings between testing sessions

### idle_timeout

**Type**: Integer (seconds)  
**Default**: `600` (10 minutes)  
**Required**: When suspension is enabled

Time in seconds to wait before suspending an idle machine.

```yaml
suspend:
  enabled: true
  idle_timeout: 300     # 5 minutes
  # or
  idle_timeout: 1800    # 30 minutes
  # or  
  idle_timeout: 60      # 1 minute (aggressive)
```

**Recommendations by Environment**:

| Environment | Timeout | Reason |
|------------|---------|--------|
| **Development** | 180-300s | Quick iteration, cost savings |
| **Staging** | 300-600s | Balance between cost and availability |
| **Production** | 900-1800s | Prioritize availability over cost |
| **Background Jobs** | 60-180s | Suspend quickly between job runs |

### check_interval

**Type**: Integer (seconds)  
**Default**: `30`  
**Required**: No

How often Navigator checks for idle conditions.

```yaml
suspend:
  enabled: true
  idle_timeout: 600
  check_interval: 30    # Check every 30 seconds
```

**Considerations**:
- **Smaller values**: More responsive suspension, slightly higher CPU usage
- **Larger values**: Less CPU overhead, less precise suspension timing
- **Recommended**: 15-60 seconds depending on idle_timeout

### grace_period

**Type**: Integer (seconds)  
**Default**: `60`  
**Required**: No

Additional time to wait after detecting idle state before actually suspending.

```yaml
suspend:
  enabled: true
  idle_timeout: 300
  grace_period: 60      # Wait extra 60 seconds before suspending
```

**Purpose**:
- Handle brief idle periods (user thinking, page loads)
- Avoid suspension during temporary quiet moments
- Provide buffer for cleanup operations

## Environment-Specific Examples

### Development Setup

```yaml
suspend:
  enabled: true
  idle_timeout: 180     # 3 minutes - save costs during breaks
  check_interval: 15    # Quick response
  grace_period: 30      # Short grace period
```

**Benefits**:
- Suspends quickly during development breaks
- Wakes instantly when you return to coding
- Significant cost savings for personal projects

### Staging Environment

```yaml
suspend:
  enabled: true
  idle_timeout: 600     # 10 minutes between test sessions
  check_interval: 30    # Standard checking
  grace_period: 60      # Standard grace period
```

**Benefits**:
- Suspends between testing sessions
- Available immediately when QA starts testing
- Balances cost and availability

### Production (Low Traffic)

```yaml
suspend:
  enabled: true
  idle_timeout: 1800    # 30 minutes for production safety
  check_interval: 60    # Less frequent checks
  grace_period: 120     # Extra grace for safety
```

**Benefits**:
- Suspends during genuine low-traffic periods
- Conservative timeouts prevent premature suspension
- Cost optimization for variable traffic patterns

### Production (High Availability)

```yaml
suspend:
  enabled: false        # Disable for maximum availability
```

**When to disable**:
- Critical applications requiring 24/7 availability
- High-traffic applications with minimal idle time
- Applications with long-running processes
- When cost is less important than response time

## Activity Detection

Navigator tracks multiple activity types:

### HTTP Requests

Any incoming HTTP request resets the idle timer:

```yaml
# All requests count as activity
GET /users
POST /api/data
WebSocket upgrade requests
Static file requests (if served by Navigator)
```

### Rails Processes

Active Rails processes prevent suspension:

```yaml
# Navigator monitors Rails process activity
ps aux | grep "rails server"  # Any running Rails process
```

### Managed Processes

Background processes can prevent suspension:

```yaml
managed_processes:
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    auto_restart: true      # Prevents suspension while running
    
  - name: cleanup-job
    command: bundle
    args: [exec, rake, cleanup:run]  
    auto_restart: false     # Allows suspension after completion
```

### WebSocket Connections

Open WebSocket connections maintain activity:

```yaml
applications:
  tenants:
    - name: cable
      path: /cable
      force_max_concurrent_requests: 0  # WebSocket support
```

## Platform Integration

### Fly.io Integration

Navigator automatically integrates with Fly.io when these environment variables are present:

```bash
# Set automatically by Fly.io
FLY_APP_NAME=myapp
FLY_MACHINE_ID=1234567890abcdef

# Optional: for machine control API
FLY_API_TOKEN=fly_api_token_here
```

**Fly.io Configuration**:
```toml title="fly.toml"
[machines]
  auto_start = true
  auto_stop = true
  min_machines_running = 0

[env]
  SUSPEND_ENABLED = "true"
  SUSPEND_TIMEOUT = "600"
```

### Other Platforms

For non-Fly.io platforms, suspension falls back to graceful shutdown:

```yaml
# Platform-agnostic suspension
suspend:
  enabled: true
  idle_timeout: 600
  # Machine will shutdown gracefully instead of suspend
```

## Monitoring and Debugging

### Enable Debug Logging

```bash
LOG_LEVEL=debug navigator config.yml
```

**Debug output**:
```
DEBUG Activity tracker initialized idle_timeout=300s
DEBUG HTTP request activity recorded method=GET path=/users
DEBUG Checking idle status last_activity=45s_ago threshold=300s
DEBUG Idle timeout not reached, continuing activity monitoring
DEBUG No activity detected for 300s, initiating suspension sequence
INFO Starting graceful shutdown for machine suspension
```

### Activity Monitoring

```bash
# Monitor suspension events
journalctl -u navigator | grep -E "(suspend|idle|activity)"

# Watch real-time activity
tail -f /var/log/navigator.log | grep -E "(activity|idle)"
```

### Metrics Collection

```bash
#!/bin/bash
# Suspension metrics

# Time since last activity
last_activity_log=$(journalctl -u navigator | grep "activity recorded" | tail -1)
if [ -n "$last_activity_log" ]; then
    # Extract timestamp and calculate idle time
    idle_seconds=$(( $(date +%s) - $(date -d "$(echo $last_activity_log | cut -d' ' -f1-2)" +%s) ))
    echo "navigator_idle_seconds $idle_seconds"
fi

# Suspension events today
suspensions_today=$(journalctl -u navigator --since today | grep -c "suspension")
echo "navigator_suspensions_today $suspensions_today"

# Current suspension status
if journalctl -u navigator --since "1 hour ago" | grep -q "suspension.*initiated"; then
    echo "navigator_machine_suspended 1"
else
    echo "navigator_machine_suspended 0"
fi
```

## Troubleshooting

### Machine Won't Suspend

**Issue**: Idle timeout reached but machine doesn't suspend

**Debug Steps**:
```bash
# Check for active processes
ps aux | grep -E "(rails|ruby|navigator|sidekiq)"

# Check network connections
netstat -an | grep ESTABLISHED | grep -E ":300[0-9]|:400[0-9]"

# Check Navigator logs
journalctl -u navigator | grep -E "(idle|activity|suspend)"

# Manual suspension test
LOG_LEVEL=debug navigator config.yml
```

**Common Causes**:
- Rails processes still running
- Active managed processes with `auto_restart: true`
- Open WebSocket connections
- Recent HTTP requests
- External health checks

### Suspension Happens Too Quickly

**Issue**: Machine suspends during legitimate use

**Solutions**:
```yaml
# Increase idle timeout
suspend:
  idle_timeout: 900     # 15 minutes instead of 5

# Add grace period
suspend:
  grace_period: 120     # Extra 2 minutes

# Less frequent checking
suspend:
  check_interval: 60    # Check every minute instead of 30s
```

### Suspension Logs Missing

**Issue**: Can't see suspension activity in logs

**Solutions**:
```bash
# Enable debug logging
export LOG_LEVEL=debug

# Check systemd logs
journalctl -u navigator -f

# Verify configuration
navigator --validate config.yml
```

### Wake Time Too Slow

**Issue**: Machine takes too long to respond after suspension

**This is typically a platform issue, not Navigator**:
- **Fly.io**: Usually wakes in 1-3 seconds
- **Other platforms**: May need to restart service instead of suspend
- **Network routing**: DNS/load balancer propagation delays

## Integration Examples

### CI/CD Pipeline

```yaml
# Suspend staging after deployments
suspend:
  enabled: true
  idle_timeout: 300     # Quick suspension after deployment tests

# Production - disable during business hours
suspend:
  enabled: false        # Override via environment variable
```

```bash
# In deployment script
if [ "$ENVIRONMENT" = "staging" ]; then
    export SUSPEND_ENABLED=true
    export SUSPEND_TIMEOUT=300
else
    export SUSPEND_ENABLED=false
fi
```

### Multi-Region Deployment

```yaml
# Different suspension patterns by region
suspend:
  enabled: true
  idle_timeout: "${SUSPEND_TIMEOUT:-600}"   # Environment-specific
```

```bash
# US East (business hours ET)
SUSPEND_TIMEOUT=1800    # 30 minutes

# Europe (business hours CET) 
SUSPEND_TIMEOUT=900     # 15 minutes

# Asia Pacific (business hours JST)
SUSPEND_TIMEOUT=600     # 10 minutes
```

### Background Job Integration

```yaml
suspend:
  enabled: true
  idle_timeout: 300

managed_processes:
  # Prevents suspension while running
  - name: critical-worker
    command: bundle
    args: [exec, sidekiq]
    auto_restart: true
    
  # Allows suspension after completion
  - name: hourly-cleanup
    command: bundle  
    args: [exec, rake, cleanup:hourly]
    auto_restart: false
```

## Best Practices

### 1. Environment-Appropriate Timeouts

```yaml
# Conservative for production
suspend:
  idle_timeout: 1800    # 30 minutes

# Aggressive for development
suspend:
  idle_timeout: 180     # 3 minutes
```

### 2. Monitor Suspension Patterns

```bash
# Weekly suspension report
journalctl -u navigator --since "1 week ago" | grep "suspension" | wc -l
```

### 3. Health Check Coordination

```yaml
# Use platform health checks instead of external monitoring
# to avoid preventing suspension

# fly.toml
[http_service.checks.alive]
  path = "/up"
  interval = "30s"
```

### 4. Process Management

```yaml
# Design processes to allow suspension
managed_processes:
  - name: batch-job
    auto_restart: false   # Completes and allows suspension
    
  - name: persistent-worker  
    auto_restart: true    # Keeps machine active
```

## See Also

- [Machine Suspension Feature](../features/machine-suspend.md)
- [Fly.io Deployment](../deployment/fly-io.md)
- [Process Management](processes.md)
- [YAML Reference](yaml-reference.md)