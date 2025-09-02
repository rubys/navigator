# Monitoring and Observability

Comprehensive monitoring setup for Navigator in production environments, including metrics, logging, alerting, and performance monitoring.

## Quick Start

```bash
# 1. Enable structured logging
export LOG_LEVEL=info

# 2. Set up health check endpoint
curl http://localhost:3000/up

# 3. Monitor with systemd
sudo journalctl -u navigator -f

# 4. Basic metrics collection
ps aux | grep navigator
```

## Health Monitoring

### Health Check Endpoint

Navigator applications typically expose a health check endpoint:

```bash
# Basic health check
curl http://localhost:3000/up

# With timeout and failure detection
curl -f --max-time 5 http://localhost:3000/up || echo "Health check failed"
```

**Rails setup** (add to `config/routes.rb`):
```ruby
Rails.application.routes.draw do
  get '/up', to: 'rails/health#show', as: :rails_health_check
end
```

### Process Health Monitoring

```bash
#!/bin/bash
# /usr/local/bin/navigator-health.sh

# Check Navigator process
if ! pgrep -f navigator > /dev/null; then
    echo "CRITICAL: Navigator process not running"
    exit 2
fi

# Check port binding
if ! netstat -tlnp | grep -q ":3000.*navigator"; then
    echo "CRITICAL: Navigator not listening on port 3000"
    exit 2
fi

# Check HTTP response
if ! curl -f -s --max-time 5 http://localhost:3000/up > /dev/null; then
    echo "WARNING: Navigator health check failed"
    exit 1
fi

# Check Rails processes
rails_count=$(pgrep -f "rails server" | wc -l)
if [ "$rails_count" -eq 0 ]; then
    echo "WARNING: No Rails processes running"
    exit 1
fi

echo "OK: Navigator healthy, $rails_count Rails processes"
exit 0
```

## Logging

### Structured Logging Configuration

Navigator uses Go's `slog` package for structured logging:

```bash
# Set log level
export LOG_LEVEL=info    # debug, info, warn, error

# Run Navigator with structured logging
navigator config.yml
```

**Log format example**:
```
2024-09-02T17:20:42Z INFO Starting Navigator listen=:3000
2024-09-02T17:20:42Z INFO Process started app=main port=4001 pid=12345
2024-09-02T17:20:45Z DEBUG Request routed path=/users method=GET app=main
2024-09-02T17:20:45Z WARN Process idle timeout app=main idle_time=300s
```

### Log Aggregation

#### systemd Journal Integration

```bash
# View Navigator logs
sudo journalctl -u navigator -f

# Search logs
sudo journalctl -u navigator | grep ERROR

# Export logs for analysis
sudo journalctl -u navigator --since yesterday --output json > navigator.log
```

#### rsyslog Configuration

```bash title="/etc/rsyslog.d/navigator.conf"
# Separate Navigator logs
:programname, isequal, "navigator" /var/log/navigator.log
& stop
```

#### Log Rotation

```bash title="/etc/logrotate.d/navigator"
/var/log/navigator.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 navigator navigator
    postrotate
        /usr/bin/systemctl reload navigator
    endscript
}
```

## Metrics Collection

### System Metrics

```bash
#!/bin/bash
# /usr/local/bin/navigator-metrics.sh

# Process metrics
echo "# Navigator process metrics"
echo "navigator_processes $(pgrep -f navigator | wc -l)"
echo "navigator_rails_processes $(pgrep -f 'rails server' | wc -l)"

# Memory usage (in bytes)
navigator_memory=$(ps -o pid,rss -p $(pgrep -f navigator) | tail -n +2 | awk '{sum+=$2} END {print sum*1024}')
echo "navigator_memory_bytes ${navigator_memory:-0}"

# CPU usage
navigator_cpu=$(ps -o pid,pcpu -p $(pgrep -f navigator) | tail -n +2 | awk '{sum+=$2} END {print sum}')
echo "navigator_cpu_percent ${navigator_cpu:-0}"

# Connection count
connections=$(netstat -an | grep :3000 | grep ESTABLISHED | wc -l)
echo "navigator_connections $connections"

# Port usage (4000-4099 range for Rails processes)
rails_ports=$(netstat -tlnp | grep -E ':40[0-9][0-9]' | wc -l)
echo "navigator_rails_ports_used $rails_ports"
```

### Application Metrics

Monitor Rails application performance:

```bash
#!/bin/bash
# Rails application metrics from logs

# Request rate (requests per minute)
requests_per_min=$(tail -n 1000 /var/log/navigator.log | grep "$(date '+%Y-%m-%dT%H:%M')" | grep -c 'method=GET\|method=POST')
echo "rails_requests_per_minute $requests_per_min"

# Response time analysis
tail -n 1000 /var/log/navigator.log | grep "completed" | awk '{print $NF}' | sed 's/ms//' | awk '
{
    sum+=$1; 
    count++; 
    if($1>max) max=$1; 
    if(min=="" || $1<min) min=$1
} 
END {
    print "rails_response_time_avg", (count>0 ? sum/count : 0)
    print "rails_response_time_max", (max ? max : 0)
    print "rails_response_time_min", (min ? min : 0)
}'
```

## Prometheus Integration

### Metrics Export

```bash title="/usr/local/bin/navigator-prometheus.sh"
#!/bin/bash
# Export Navigator metrics in Prometheus format

# Write metrics to file for node_exporter textfile collector
METRICS_FILE="/var/lib/prometheus/node-exporter/navigator.prom"

{
    echo "# HELP navigator_up Navigator process status"
    echo "# TYPE navigator_up gauge"
    if pgrep -f navigator > /dev/null; then
        echo "navigator_up 1"
    else
        echo "navigator_up 0"
    fi

    echo "# HELP navigator_processes Number of Navigator processes"
    echo "# TYPE navigator_processes gauge"
    echo "navigator_processes $(pgrep -f navigator | wc -l)"

    echo "# HELP navigator_rails_processes Number of Rails processes"
    echo "# TYPE navigator_rails_processes gauge"
    echo "navigator_rails_processes $(pgrep -f 'rails server' | wc -l)"

    echo "# HELP navigator_memory_bytes Navigator memory usage in bytes"
    echo "# TYPE navigator_memory_bytes gauge"
    memory=$(ps -o pid,rss -p $(pgrep -f navigator) | tail -n +2 | awk '{sum+=$2} END {print sum*1024}')
    echo "navigator_memory_bytes ${memory:-0}"

    echo "# HELP navigator_connections_total Active connections"
    echo "# TYPE navigator_connections_total gauge"
    connections=$(netstat -an | grep :3000 | grep ESTABLISHED | wc -l)
    echo "navigator_connections_total $connections"
} > "$METRICS_FILE.tmp" && mv "$METRICS_FILE.tmp" "$METRICS_FILE"
```

```bash
# Run metrics collection every minute
echo "* * * * * navigator /usr/local/bin/navigator-prometheus.sh" | sudo crontab -u navigator -
```

### Prometheus Configuration

```yaml title="prometheus.yml"
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'navigator-node'
    static_configs:
      - targets: ['localhost:9100']
    
  - job_name: 'navigator-health'
    metrics_path: '/up'
    static_configs:
      - targets: ['localhost:3000']
    scrape_interval: 30s
```

## Application Performance Monitoring

### New Relic Integration

```yaml title="Navigator configuration"
applications:
  global_env:
    NEW_RELIC_LICENSE_KEY: "${NEW_RELIC_LICENSE_KEY}"
    NEW_RELIC_APP_NAME: "Navigator Production"
    NEW_RELIC_DISTRIBUTED_TRACING_ENABLED: "true"
```

```ruby title="Rails: config/newrelic.yml"
production:
  license_key: <%= ENV['NEW_RELIC_LICENSE_KEY'] %>
  app_name: Navigator Rails App
  distributed_tracing:
    enabled: true
  transaction_tracer:
    enabled: true
  error_collector:
    enabled: true
```

### Honeybadger Error Tracking

```yaml title="Navigator configuration"
applications:
  global_env:
    HONEYBADGER_API_KEY: "${HONEYBADGER_API_KEY}"
    HONEYBADGER_ENV: "production"
```

### Custom Rails Monitoring

```ruby title="Rails: config/initializers/navigator_monitoring.rb"
# Custom middleware for Navigator-specific metrics
class NavigatorMonitoring
  def initialize(app)
    @app = app
  end

  def call(env)
    start_time = Time.current
    status, headers, response = @app.call(env)
    duration = (Time.current - start_time) * 1000

    # Log request metrics in Navigator-compatible format
    Rails.logger.info({
      event: 'request_completed',
      method: env['REQUEST_METHOD'],
      path: env['PATH_INFO'],
      status: status,
      duration_ms: duration.round(2),
      process_id: Process.pid
    }.to_json)

    [status, headers, response]
  rescue => e
    Rails.logger.error({
      event: 'request_error',
      error: e.class.name,
      message: e.message,
      path: env['PATH_INFO']
    }.to_json)
    raise
  end
end

Rails.application.config.middleware.use NavigatorMonitoring
```

## Alerting

### Basic Shell Script Alerts

```bash title="/usr/local/bin/navigator-alerts.sh"
#!/bin/bash
# Basic alerting script

ALERT_EMAIL="admin@example.com"
WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

send_alert() {
    local severity=$1
    local message=$2
    
    # Email alert
    echo "Navigator Alert [$severity]: $message" | mail -s "Navigator Alert" "$ALERT_EMAIL"
    
    # Slack webhook
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"Navigator Alert [$severity]: $message\"}" \
        "$WEBHOOK_URL"
}

# Check Navigator health
if ! /usr/local/bin/navigator-health.sh > /dev/null; then
    send_alert "CRITICAL" "Navigator health check failed"
fi

# Check memory usage
memory_usage=$(ps -o pid,pmem -p $(pgrep -f navigator) | tail -n +2 | awk '{sum+=$2} END {print sum}')
if (( $(echo "$memory_usage > 80" | bc -l) )); then
    send_alert "WARNING" "Navigator memory usage high: ${memory_usage}%"
fi

# Check log for errors
error_count=$(journalctl -u navigator --since "5 minutes ago" -p err | wc -l)
if [ "$error_count" -gt 0 ]; then
    send_alert "WARNING" "Navigator logged $error_count errors in last 5 minutes"
fi
```

### systemd Service Monitoring

```bash title="/etc/systemd/system/navigator-monitor.service"
[Unit]
Description=Navigator Monitoring
Requires=navigator.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/navigator-alerts.sh

[Install]
WantedBy=multi-user.target
```

```bash title="/etc/systemd/system/navigator-monitor.timer"
[Unit]
Description=Run Navigator monitoring every 5 minutes
Requires=navigator-monitor.service

[Timer]
OnCalendar=*:0/5
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
# Enable monitoring
sudo systemctl enable navigator-monitor.timer
sudo systemctl start navigator-monitor.timer
```

## Dashboard Setup

### Grafana Dashboard

```json title="navigator-dashboard.json"
{
  "dashboard": {
    "title": "Navigator Monitoring",
    "panels": [
      {
        "title": "Navigator Status",
        "type": "stat",
        "targets": [
          {
            "expr": "navigator_up",
            "legendFormat": "Navigator Up"
          }
        ]
      },
      {
        "title": "Active Processes",
        "type": "graph",
        "targets": [
          {
            "expr": "navigator_processes",
            "legendFormat": "Navigator Processes"
          },
          {
            "expr": "navigator_rails_processes", 
            "legendFormat": "Rails Processes"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "navigator_memory_bytes",
            "legendFormat": "Memory Usage"
          }
        ]
      },
      {
        "title": "Active Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "navigator_connections_total",
            "legendFormat": "Connections"
          }
        ]
      }
    ]
  }
}
```

### Simple HTML Dashboard

```html title="/var/www/monitor/index.html"
<!DOCTYPE html>
<html>
<head>
    <title>Navigator Status</title>
    <meta http-equiv="refresh" content="30">
</head>
<body>
    <h1>Navigator Status Dashboard</h1>
    
    <div id="status">
        <script>
            fetch('/cgi-bin/navigator-status.sh')
                .then(response => response.text())
                .then(data => {
                    document.getElementById('status').innerHTML = '<pre>' + data + '</pre>';
                });
        </script>
    </div>
</body>
</html>
```

## Performance Monitoring

### Response Time Monitoring

```bash
#!/bin/bash
# Monitor Navigator response times

measure_response_time() {
    local url=$1
    local name=$2
    
    time=$(curl -o /dev/null -s -w '%{time_total}\n' "$url")
    echo "response_time_seconds{endpoint=\"$name\"} $time"
}

# Monitor different endpoints
measure_response_time "http://localhost:3000/up" "health"
measure_response_time "http://localhost:3000/" "home"
measure_response_time "http://localhost:3000/api/users" "api"
```

### Load Testing Integration

```bash
#!/bin/bash
# Automated load testing with monitoring

# Run load test
ab -n 1000 -c 10 http://localhost:3000/ > /tmp/load_test.out

# Extract key metrics
requests_per_second=$(grep "Requests per second" /tmp/load_test.out | awk '{print $4}')
mean_time=$(grep "Time per request" /tmp/load_test.out | head -1 | awk '{print $4}')

# Log results
echo "load_test_rps $requests_per_second"
echo "load_test_mean_time $mean_time"

# Alert if performance degrades
if (( $(echo "$requests_per_second < 50" | bc -l) )); then
    echo "WARNING: Low request rate: $requests_per_second RPS"
fi
```

## Troubleshooting Monitoring

### Common Issues

#### No Metrics Being Collected

```bash
# Check if scripts are executable
ls -la /usr/local/bin/navigator-*.sh

# Verify cron jobs
crontab -l -u navigator

# Test metric collection manually
/usr/local/bin/navigator-metrics.sh
```

#### Health Checks Failing

```bash
# Test health check manually
curl -v http://localhost:3000/up

# Check Navigator process
ps aux | grep navigator

# Verify port binding
netstat -tlnp | grep :3000
```

#### High Memory Usage Alerts

```bash
# Check actual memory usage
ps aux --sort=-%mem | head -10

# Monitor Rails process memory
ps aux | grep rails | awk '{print $6}' | sort -nr

# Check for memory leaks
while true; do
    ps -o pid,rss,cmd -p $(pgrep -f rails)
    sleep 60
done
```

### Debug Logging

```bash
# Enable debug logging for troubleshooting
export LOG_LEVEL=debug
systemctl restart navigator

# Watch debug logs
journalctl -u navigator -f | grep DEBUG
```

## Best Practices

### 1. Monitoring Strategy
- Monitor both Navigator and Rails processes
- Track system resources (CPU, memory, disk)
- Set up both technical and business metrics
- Use multiple monitoring tools for redundancy

### 2. Alerting Guidelines
- Alert on symptoms, not just causes
- Use different severity levels appropriately
- Avoid alert fatigue with proper thresholds
- Include runbook information in alerts

### 3. Performance Monitoring
- Establish baseline performance metrics
- Monitor end-to-end response times
- Track error rates and types
- Set up synthetic monitoring

### 4. Log Management
- Use structured logging consistently
- Implement proper log rotation
- Centralize logs for analysis
- Include correlation IDs for tracing

## Integration Examples

### CloudWatch (AWS)

```bash
# Install CloudWatch agent
wget https://s3.amazonaws.com/amazoncloudwatch-agent/amazon_linux/amd64/latest/amazon-cloudwatch-agent.rpm
sudo rpm -U amazon-cloudwatch-agent.rpm

# Configure custom metrics
aws logs create-log-group --log-group-name navigator-logs
```

### Datadog Integration

```yaml
# Add to Navigator environment
applications:
  global_env:
    DD_API_KEY: "${DATADOG_API_KEY}"
    DD_SITE: "datadoghq.com"
    DD_SERVICE: "navigator"
    DD_VERSION: "1.0.0"
```

## See Also

- [Production Deployment](production.md)
- [systemd Integration](../examples/systemd.md)
- [Process Management](../features/process-management.md)
- [Configuration Reference](../configuration/yaml-reference.md)