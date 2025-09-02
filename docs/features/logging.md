# Logging and Observability

Navigator provides comprehensive logging capabilities using Go's structured logging (`slog`) with configurable levels, formats, and output destinations.

## Log Levels

Navigator supports four log levels via the `LOG_LEVEL` environment variable:

### Debug Level
```bash
LOG_LEVEL=debug navigator config.yml
```

**Output includes**:
- Request routing decisions
- Process management operations
- Configuration parsing details
- Static file serving attempts
- Authentication checks

**Example output**:
```
2024-09-02T17:20:42Z DEBUG Request received path=/api/users method=GET
2024-09-02T17:20:42Z DEBUG Route matched app=main pattern=/
2024-09-02T17:20:42Z DEBUG Starting process app=main port=4001
2024-09-02T17:20:42Z DEBUG Proxying request to localhost:4001
2024-09-02T17:20:42Z DEBUG Static file check path=/favicon.ico root=./public
2024-09-02T17:20:42Z DEBUG Auth check passed path=/assets/app.css public_path=*.css
```

### Info Level (Default)
```bash
LOG_LEVEL=info navigator config.yml
# or simply
navigator config.yml
```

**Output includes**:
- Navigator startup and shutdown
- Process lifecycle events  
- Configuration reloads
- Major operational events

**Example output**:
```
2024-09-02T17:20:42Z INFO Starting Navigator listen=:3000
2024-09-02T17:20:42Z INFO Process started app=main port=4001 pid=12345
2024-09-02T17:20:45Z INFO Configuration reloaded
2024-09-02T17:20:50Z INFO Process stopped app=main pid=12345
```

### Warn Level
```bash
LOG_LEVEL=warn navigator config.yml
```

**Output includes**:
- Configuration issues
- Process timeout warnings
- Performance warnings
- Non-critical failures

**Example output**:
```
2024-09-02T17:20:45Z WARN Process idle timeout app=main idle_time=300s
2024-09-02T17:20:46Z WARN Configuration file not found, using defaults path=config/navigator.yml
2024-09-02T17:20:47Z WARN High memory usage app=main memory_mb=512
```

### Error Level  
```bash
LOG_LEVEL=error navigator config.yml
```

**Output includes**:
- Critical failures
- Process crashes
- Configuration errors
- System errors

**Example output**:
```
2024-09-02T17:20:50Z ERROR Failed to start process app=main error="port unavailable"
2024-09-02T17:20:51Z ERROR Configuration invalid: missing required field server.listen
2024-09-02T17:20:52Z ERROR Cannot bind to port port=3000 error="permission denied"
```

## Structured Logging Format

Navigator uses structured logging with consistent key-value pairs:

```
TIMESTAMP LEVEL MESSAGE key1=value1 key2=value2 ...
```

### Common Log Fields

| Field | Description | Example |
|-------|-------------|---------|
| `timestamp` | ISO 8601 timestamp | `2024-09-02T17:20:42Z` |
| `level` | Log level | `INFO`, `DEBUG`, `WARN`, `ERROR` |
| `message` | Human-readable message | `Process started` |
| `app` | Application/tenant name | `main`, `api`, `admin` |
| `pid` | Process ID | `12345` |
| `port` | Port number | `4001` |
| `path` | Request path | `/api/users` |
| `method` | HTTP method | `GET`, `POST` |
| `status` | HTTP status code | `200`, `404`, `500` |
| `duration` | Request duration | `45ms` |
| `error` | Error message | `connection refused` |

## Configuration Examples

### Development Logging

```yaml title="config/navigator-dev.yml"
# Enable debug logging for development
```

```bash
# Set via environment
export LOG_LEVEL=debug
navigator config/navigator-dev.yml
```

**Benefits**:
- Detailed request tracing
- Configuration debugging
- Process management insights
- Static file resolution details

### Production Logging

```yaml title="config/navigator-prod.yml"
# Standard info level logging
```

```bash
# Set via environment or systemd
export LOG_LEVEL=info
navigator config/navigator-prod.yml
```

**Benefits**:
- Essential operational information
- Performance monitoring data
- Error tracking
- Process lifecycle events

### Minimal Logging

```bash
# Only warnings and errors
LOG_LEVEL=warn navigator config.yml
```

**Use Cases**:
- High-traffic environments
- Minimal disk usage requirements
- When external monitoring is primary

## Log Output Destinations

### Standard Output (Default)

Navigator logs to stdout by default, which integrates well with container orchestrators and process managers:

```bash
# Logs to stdout/stderr
navigator config.yml

# Capture logs
navigator config.yml > navigator.log 2>&1

# Pipe to logging service
navigator config.yml | logger -t navigator
```

### systemd Journal Integration

When running under systemd, logs are automatically captured by the journal:

```bash
# View Navigator logs
sudo journalctl -u navigator -f

# Search logs
sudo journalctl -u navigator | grep ERROR

# Export logs
sudo journalctl -u navigator --since yesterday --output json > navigator.log
```

### File Logging via systemd

```ini title="/etc/systemd/system/navigator.service"
[Service]
StandardOutput=append:/var/log/navigator.log
StandardError=append:/var/log/navigator.log
ExecStart=/usr/local/bin/navigator /etc/navigator/config.yml
```

### Syslog Integration

```bash
# Configure rsyslog to capture Navigator logs
sudo tee /etc/rsyslog.d/navigator.conf << 'EOF'
# Navigator logs
if $programname == "navigator" then /var/log/navigator.log
& stop
EOF

sudo systemctl restart rsyslog
```

## Log Analysis and Monitoring

### Real-Time Monitoring

```bash
# Follow Navigator logs in real-time
tail -f /var/log/navigator.log

# Monitor specific log levels
tail -f /var/log/navigator.log | grep -E "(WARN|ERROR)"

# Monitor request activity
tail -f /var/log/navigator.log | grep -E "(Request|Proxying)"

# Monitor process events
tail -f /var/log/navigator.log | grep -E "(started|stopped)"
```

### Log Analysis Scripts

```bash title="scripts/analyze-logs.sh"
#!/bin/bash
# Navigator log analysis

LOG_FILE="${1:-/var/log/navigator.log}"

echo "Navigator Log Analysis"
echo "======================"

# Request statistics
echo -n "Total requests today: "
grep "$(date '+%Y-%m-%d')" "$LOG_FILE" | grep -c "Request received"

echo -n "Error count today: "
grep "$(date '+%Y-%m-%d')" "$LOG_FILE" | grep -c "ERROR"

echo -n "Warning count today: "
grep "$(date '+%Y-%m-%d')" "$LOG_FILE" | grep -c "WARN"

# Process events
echo -n "Process starts today: "
grep "$(date '+%Y-%m-%d')" "$LOG_FILE" | grep -c "Process started"

echo -n "Process stops today: "
grep "$(date '+%Y-%m-%d')" "$LOG_FILE" | grep -c "Process stopped"

# Top error messages
echo -e "\nTop errors:"
grep "$(date '+%Y-%m-%d')" "$LOG_FILE" | grep "ERROR" | \
    awk -F 'ERROR ' '{print $2}' | sort | uniq -c | sort -nr | head -5

# Response time analysis
echo -e "\nAverage response times:"
grep "$(date '+%Y-%m-%d')" "$LOG_FILE" | grep "duration=" | \
    sed 's/.*duration=\([0-9]*\)ms.*/\1/' | \
    awk '{sum+=$1; count++} END {print "Average:", sum/count "ms"}'
```

### Performance Metrics from Logs

```bash title="scripts/metrics-from-logs.sh"
#!/bin/bash
# Extract metrics from Navigator logs

LOG_FILE="${1:-/var/log/navigator.log}"
TIME_RANGE="${2:-1 hour ago}"

# Request rate (requests per minute)
requests_per_min=$(journalctl -u navigator --since "$TIME_RANGE" | \
    grep "Request received" | wc -l)
echo "navigator_requests_per_minute $requests_per_min"

# Error rate
error_count=$(journalctl -u navigator --since "$TIME_RANGE" | \
    grep "ERROR" | wc -l)
echo "navigator_errors_total $error_count"

# Process restarts
restarts=$(journalctl -u navigator --since "$TIME_RANGE" | \
    grep "Process started" | wc -l)
echo "navigator_process_restarts $restarts"

# Memory warnings
memory_warnings=$(journalctl -u navigator --since "$TIME_RANGE" | \
    grep "High memory usage" | wc -l)
echo "navigator_memory_warnings $memory_warnings"
```

## Integration with Monitoring Systems

### Prometheus Integration

```bash title="scripts/navigator-exporter.sh"
#!/bin/bash
# Navigator metrics exporter for Prometheus

METRICS_FILE="/var/lib/prometheus/node-exporter/navigator.prom"
LOG_FILE="/var/log/navigator.log"

{
    echo "# HELP navigator_requests_total Total requests processed"
    echo "# TYPE navigator_requests_total counter"
    requests=$(grep -c "Request received" "$LOG_FILE")
    echo "navigator_requests_total $requests"

    echo "# HELP navigator_errors_total Total errors logged"
    echo "# TYPE navigator_errors_total counter"
    errors=$(grep -c "ERROR" "$LOG_FILE")
    echo "navigator_errors_total $errors"

    echo "# HELP navigator_processes_started Total processes started"
    echo "# TYPE navigator_processes_started counter"
    starts=$(grep -c "Process started" "$LOG_FILE")
    echo "navigator_processes_started $starts"
    
} > "$METRICS_FILE.tmp" && mv "$METRICS_FILE.tmp" "$METRICS_FILE"
```

### ELK Stack Integration

```yaml title="filebeat.yml"
# Filebeat configuration for Navigator logs
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/navigator.log
  fields:
    service: navigator
    environment: production
  fields_under_root: true
  multiline.pattern: '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z'
  multiline.negate: true
  multiline.match: after

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "navigator-logs-%{+yyyy.MM.dd}"

processors:
  - dissect:
      tokenizer: "%{timestamp} %{level} %{message}"
      field: "message"
```

### Fluentd Integration

```ruby title="fluent.conf"
# Fluentd configuration for Navigator logs
<source>
  @type tail
  path /var/log/navigator.log
  pos_file /var/log/fluentd/navigator.log.pos
  tag navigator
  format /^(?<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z) (?<level>\w+) (?<message>.*)/
  time_format %Y-%m-%dT%H:%M:%SZ
</source>

<match navigator>
  @type elasticsearch
  host elasticsearch
  port 9200
  index_name navigator-logs
  type_name navigator
</match>
```

## Application-Level Logging

Navigator integrates with Rails application logging:

### Rails Log Configuration

```ruby title="config/environments/production.rb"
# Configure Rails logging to work with Navigator
Rails.application.configure do
  # Use stdout for logs (captured by Navigator/systemd)
  config.logger = ActiveSupport::Logger.new(STDOUT)
  
  # Structured logging format
  config.log_formatter = proc do |severity, datetime, progname, msg|
    "#{datetime.strftime('%Y-%m-%dT%H:%M:%SZ')} #{severity} [Rails] #{msg}\n"
  end
  
  # Log level matching Navigator
  config.log_level = ENV.fetch('RAILS_LOG_LEVEL', 'info').to_sym
end
```

### Custom Logging Middleware

```ruby title="config/initializers/navigator_logging.rb"
# Custom middleware to enhance Navigator request logging
class NavigatorLoggingMiddleware
  def initialize(app)
    @app = app
  end

  def call(env)
    start_time = Time.current
    request = ActionDispatch::Request.new(env)
    
    # Log request start
    Rails.logger.info({
      event: 'request_start',
      method: request.method,
      path: request.path,
      ip: request.ip,
      user_agent: request.user_agent
    }.to_json)
    
    status, headers, response = @app.call(env)
    
    # Log request completion  
    duration = ((Time.current - start_time) * 1000).round(2)
    Rails.logger.info({
      event: 'request_complete',
      method: request.method,
      path: request.path,
      status: status,
      duration_ms: duration
    }.to_json)
    
    [status, headers, response]
  rescue => e
    # Log request error
    Rails.logger.error({
      event: 'request_error',
      method: request.method,
      path: request.path,
      error: e.class.name,
      message: e.message
    }.to_json)
    
    raise
  end
end

# Add middleware
Rails.application.config.middleware.use NavigatorLoggingMiddleware
```

## Log Rotation and Retention

### logrotate Configuration

```bash title="/etc/logrotate.d/navigator"
/var/log/navigator.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 navigator navigator
    sharedscripts
    postrotate
        /usr/bin/systemctl reload navigator
    endscript
}
```

### Automated Cleanup

```bash title="scripts/log-cleanup.sh"
#!/bin/bash
# Clean up old Navigator logs

LOG_DIR="/var/log"
RETENTION_DAYS=30

# Clean old log files
find "$LOG_DIR" -name "navigator*.log*" -mtime +$RETENTION_DAYS -delete

# Clean systemd journal logs older than 30 days
journalctl --vacuum-time=30d

echo "Log cleanup completed"
```

## Troubleshooting with Logs

### Common Debugging Scenarios

#### Process Startup Issues

```bash
# Check process startup logs
journalctl -u navigator | grep -E "(started|stopped|error)"

# Debug process failures
LOG_LEVEL=debug navigator config.yml 2>&1 | grep -E "(Process|error)"
```

#### Request Routing Problems

```bash
# Trace request routing
LOG_LEVEL=debug navigator config.yml 2>&1 | grep -E "(Request|Route|Proxying)"

# Monitor specific paths
tail -f /var/log/navigator.log | grep "path=/api"
```

#### Configuration Issues

```bash
# Check configuration parsing
LOG_LEVEL=debug navigator --validate config.yml

# Monitor configuration reloads
journalctl -u navigator | grep -E "(reload|configuration)"
```

#### Performance Analysis

```bash
# Monitor slow requests
journalctl -u navigator | grep "duration=" | \
    awk -F'duration=' '{print $2}' | awk '{print $1}' | \
    sort -nr | head -10

# Check memory usage patterns
journalctl -u navigator | grep -E "(memory|Memory)"
```

## Best Practices

### 1. Log Level Management

```bash
# Development - verbose logging
LOG_LEVEL=debug

# Staging - balanced logging  
LOG_LEVEL=info

# Production - minimal logging
LOG_LEVEL=warn
```

### 2. Structured Logging

```ruby
# Use structured logging in Rails
Rails.logger.info({
  event: 'user_action',
  user_id: user.id,
  action: 'login',
  timestamp: Time.current
}.to_json)
```

### 3. Log Retention

```bash
# Implement proper log rotation
# Keep 30 days for production
# Keep 7 days for development
```

### 4. Monitoring Integration

```bash
# Set up automated log monitoring
# Alert on ERROR level messages
# Monitor request rates and performance
```

### 5. Security

```bash
# Never log sensitive information
# Sanitize user input in logs
# Use appropriate log levels for security events
```

## See Also

- [Monitoring Setup](../deployment/monitoring.md)
- [Production Deployment](../deployment/production.md)
- [Configuration Reference](../configuration/yaml-reference.md)