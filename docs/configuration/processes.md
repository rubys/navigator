# Managed Processes

Navigator can start and manage additional processes alongside your Rails applications, such as Redis, Sidekiq, background workers, and other services.

## Basic Configuration

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args: ["--port", "6379"]
    working_dir: /var/lib/redis
    auto_restart: true
    start_delay: 0
```

## Process Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Unique process identifier |
| `command` | string | ✓ | Command to execute |
| `args` | array | | Command line arguments |
| `working_dir` | string | | Working directory for the process |
| `env` | object | | Environment variables |
| `auto_restart` | boolean | | Restart process if it crashes |
| `start_delay` | integer | | Seconds to wait before starting |

## Common Process Examples

### Redis Server

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args: 
      - --port 6379
      - --appendonly yes
      - --save 900 1
      - --save 300 10
    working_dir: /var/lib/redis
    env:
      REDIS_LOG_LEVEL: notice
    auto_restart: true
    start_delay: 0
```

### Sidekiq Worker

```yaml
managed_processes:
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq, -C, config/sidekiq.yml]
    working_dir: /var/www/app
    env:
      RAILS_ENV: production
      REDIS_URL: redis://localhost:6379/0
    auto_restart: true
    start_delay: 2  # Wait for Redis
```

### Custom Background Worker

```yaml
managed_processes:
  - name: worker
    command: python
    args: [worker.py, --queue, high]
    working_dir: /var/www/workers
    env:
      PYTHONPATH: /var/www/workers
      LOG_LEVEL: info
    auto_restart: true
    start_delay: 1
```

### Monitoring Service

```yaml
managed_processes:
  - name: prometheus
    command: prometheus
    args:
      - --config.file=/etc/prometheus/prometheus.yml
      - --storage.tsdb.path=/var/lib/prometheus
      - --web.console.libraries=/etc/prometheus/console_libraries
    working_dir: /var/lib/prometheus
    auto_restart: true
    start_delay: 0
```

## Process Lifecycle

### Startup Order

Navigator starts processes in the order defined in configuration:

```yaml
managed_processes:
  # 1. Start Redis first
  - name: redis
    command: redis-server
    start_delay: 0
    
  # 2. Start Sidekiq after Redis  
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    start_delay: 2  # Wait for Redis to be ready
    
  # 3. Start monitoring last
  - name: monitor
    command: ./monitor.sh
    start_delay: 5  # Wait for other services
```

### Shutdown Order

Navigator stops processes in **reverse order**:

1. Rails applications stop first
2. Managed processes stop in reverse order (last started, first stopped)
3. This preserves dependencies (Rails → Sidekiq → Redis)

### Auto-Restart Behavior

```yaml
managed_processes:
  - name: critical-service
    command: ./critical
    auto_restart: true    # Restart on any exit
    
  - name: optional-service  
    command: ./optional
    auto_restart: false   # Don't restart (default)
```

Auto-restart triggers when:
- Process exits with non-zero status
- Process crashes or is killed
- Process exits unexpectedly

Auto-restart does NOT trigger when:
- Navigator is shutting down
- Process is manually stopped
- Process exits with status 0 (success)

## Environment Variables

### Global Environment

Environment variables are inherited from Navigator's environment:

```bash
export REDIS_URL=redis://localhost:6379
export RAILS_ENV=production
./navigator config.yml
```

### Process-Specific Environment

```yaml
managed_processes:
  - name: worker1
    command: ./worker
    env:
      WORKER_ID: "1"
      QUEUE: "high"
      REDIS_URL: redis://localhost:6379/1
      
  - name: worker2
    command: ./worker
    env:
      WORKER_ID: "2" 
      QUEUE: "low"
      REDIS_URL: redis://localhost:6379/2
```

### Environment Variable Substitution

Use Navigator's template system:

```yaml
managed_processes:
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    env:
      RAILS_ENV: "${RAILS_ENV:-production}"
      DATABASE_URL: "${DATABASE_URL}"
      REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"
```

## Multi-Process Patterns

### Multiple Workers

```yaml
managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    
  # High priority queue worker
  - name: sidekiq-high
    command: bundle
    args: [exec, sidekiq, -q, high, -c, "5"]
    env:
      REDIS_URL: redis://localhost:6379/0
    auto_restart: true
    start_delay: 2
    
  # Low priority queue worker
  - name: sidekiq-low
    command: bundle
    args: [exec, sidekiq, -q, low, -c, "2"]
    env:
      REDIS_URL: redis://localhost:6379/0
    auto_restart: true
    start_delay: 2
```

### Multi-Tenant Workers

```yaml
managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    
  # Tenant-specific workers
  - name: worker-tenant1
    command: bundle
    args: [exec, sidekiq, -q, tenant1]
    env:
      TENANT_ID: tenant1
      DATABASE_URL: postgres://localhost/tenant1
    auto_restart: true
    start_delay: 2
    
  - name: worker-tenant2
    command: bundle
    args: [exec, sidekiq, -q, tenant2]
    env:
      TENANT_ID: tenant2
      DATABASE_URL: postgres://localhost/tenant2
    auto_restart: true
    start_delay: 2
```

### Service Stack

```yaml
managed_processes:
  # Data layer
  - name: redis
    command: redis-server
    auto_restart: true
    
  - name: postgres
    command: postgres
    args: [-D, /var/lib/postgres/data]
    auto_restart: true
    
  # Application layer
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    auto_restart: true
    start_delay: 3
    
  # Monitoring layer
  - name: metrics
    command: ./metrics-collector
    auto_restart: true
    start_delay: 5
```

## Working Directories

### Absolute Paths (Recommended)

```yaml
managed_processes:
  - name: app-worker
    command: bundle
    args: [exec, sidekiq]
    working_dir: /var/www/app  # Absolute path
    
  - name: data-processor
    command: python
    args: [process.py]
    working_dir: /opt/processors  # Absolute path
```

### Relative Paths

```yaml
# Relative to Navigator's working directory
managed_processes:
  - name: local-worker
    command: ./scripts/worker.sh
    working_dir: ./workers  # Relative to Navigator
```

### Process-Specific Directories

```yaml
managed_processes:
  - name: redis
    command: redis-server
    working_dir: /var/lib/redis  # Redis data directory
    
  - name: log-processor
    command: ./process-logs
    working_dir: /var/log/app    # Log directory
```

## Security Considerations

### Process Isolation

```yaml
managed_processes:
  # Run as specific user
  - name: redis
    command: sudo
    args: [-u, redis, redis-server, /etc/redis/redis.conf]
    
  # Limit resources
  - name: worker
    command: nice
    args: [-n, "10", bundle, exec, sidekiq]  # Lower priority
```

### Secure Environment

```yaml
managed_processes:
  - name: secure-worker
    command: bundle
    args: [exec, worker]
    env:
      # Use environment variables for secrets
      DATABASE_PASSWORD: "${DATABASE_PASSWORD}"
      API_KEY: "${API_KEY}"
      # Never hard-code secrets in config
```

### File Permissions

```bash
# Ensure Navigator can execute commands
chmod +x /path/to/command

# Secure working directories
chmod 750 /var/lib/service
chown navigator:service /var/lib/service
```

## Monitoring and Logging

### Process Status

Navigator logs process events:

```bash
# Check process status in logs
grep "managed_processes" /var/log/navigator.log

# See process starts
grep "Starting process" /var/log/navigator.log

# See process crashes
grep "Process crashed" /var/log/navigator.log
```

### Health Checks

```yaml
managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    
  # Health check process
  - name: healthcheck
    command: ./health-check.sh
    args: [redis, sidekiq]
    start_delay: 10
```

### Log Management

```yaml
managed_processes:
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq, --logfile, /var/log/sidekiq.log]
    env:
      RAILS_LOG_TO_STDOUT: "false"  # Log to file instead
```

## Troubleshooting

### Process Won't Start

1. **Check command exists**:
   ```bash
   which redis-server
   /usr/bin/redis-server --version
   ```

2. **Verify working directory**:
   ```bash
   ls -la /var/lib/redis
   cd /var/lib/redis && redis-server --test-memory
   ```

3. **Test manually**:
   ```bash
   cd /var/www/app
   bundle exec sidekiq -C config/sidekiq.yml
   ```

### Process Keeps Crashing

1. **Check logs**:
   ```bash
   tail -f /var/log/navigator.log | grep process-name
   ```

2. **Disable auto-restart temporarily**:
   ```yaml
   managed_processes:
     - name: problem-process
       auto_restart: false  # Stop restart loop
   ```

3. **Add debugging**:
   ```yaml
   managed_processes:
     - name: debug-process
       command: bash
       args: [-c, "set -x; /path/to/actual-command"]
   ```

### Dependencies Not Ready

```yaml
managed_processes:
  - name: dependency
    command: service
    start_delay: 0
    
  - name: dependent  
    command: client
    start_delay: 5  # Increase delay
    
  # Or add health check
  - name: wait-for-redis
    command: bash
    args: [-c, "until redis-cli ping; do sleep 1; done"]
    start_delay: 2
    auto_restart: false
    
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    start_delay: 10  # Start after health check
```

## Performance Tuning

### Resource Limits

```bash
# Set limits before starting Navigator
ulimit -n 4096      # File descriptors  
ulimit -u 2048      # Processes
ulimit -m 2097152   # Memory (KB)

./navigator config.yml
```

### Process Concurrency

```yaml
managed_processes:
  # Scale workers based on CPU cores
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq, -c, "${SIDEKIQ_CONCURRENCY:-5}"]
    
  # Multiple instances for CPU-bound work
  - name: cpu-worker-1
    command: ./cpu-intensive-task
    
  - name: cpu-worker-2  
    command: ./cpu-intensive-task
```

### Memory Management

```yaml
managed_processes:
  # Limit memory usage
  - name: memory-limited
    command: systemd-run
    args: 
      - --scope
      - --property=MemoryLimit=512M
      - ./memory-hungry-process
```

## Migration Patterns

### From Systemd Services

```systemd
# /etc/systemd/system/sidekiq.service
[Unit]
Description=Sidekiq
After=redis.service

[Service]
ExecStart=/usr/local/bin/bundle exec sidekiq
WorkingDirectory=/var/www/app
User=deploy
```

Becomes:

```yaml
managed_processes:
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /var/www/app
    auto_restart: true
```

### From Docker Compose

```yaml
# docker-compose.yml
services:
  redis:
    image: redis:latest
    command: redis-server --appendonly yes
    
  sidekiq:
    build: .
    command: bundle exec sidekiq
    depends_on: [redis]
```

Becomes:

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args: [--appendonly, yes]
    
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    start_delay: 2
```

## See Also

- [Configuration Overview](index.md)
- [Applications](applications.md)  
- [Process Management Feature](../features/process-management.md)
- [Sidekiq Example](../examples/with-sidekiq.md)