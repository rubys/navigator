# Logging

Navigator provides structured logging with configurable levels and built-in Vector integration for log aggregation.

## Log Levels

Control logging verbosity with the `LOG_LEVEL` environment variable:

```bash
# Debug - detailed operational information
LOG_LEVEL=debug navigator config.yml

# Info (default) - startup, process lifecycle, configuration changes
LOG_LEVEL=info navigator config.yml

# Warn - warnings and potential issues
LOG_LEVEL=warn navigator config.yml

# Error - only errors
LOG_LEVEL=error navigator config.yml
```

## Structured Logging

Navigator uses Go's `slog` package for structured JSON logging:

```json
{
  "@timestamp": "2024-09-02T17:20:42Z",
  "level": "INFO",
  "msg": "Process started",
  "app": "main",
  "port": 4001,
  "pid": 12345
}
```

### Common Fields

- `@timestamp` - ISO 8601 timestamp
- `level` - Log level (DEBUG, INFO, WARN, ERROR)
- `msg` - Human-readable message
- Context-specific fields (app, port, pid, path, etc.)

## Process Output Capture

All stdout/stderr from managed processes and web apps is captured with source identification:

**Text format** (default):
```
[redis.stdout] Ready to accept connections
[2025-boston.stderr] Error: Connection refused
```

**JSON format**:
```yaml
logging:
  format: json
```

```json
{"@timestamp":"2025-01-04T19:49:46Z","source":"redis","stream":"stdout","message":"Ready to accept connections"}
{"@timestamp":"2025-01-04T19:49:47Z","source":"2025/boston","stream":"stderr","message":"Error: Connection refused","tenant":"boston"}
```

## Vector Integration

Modern applications generate logs from multiple sources across multiple machines. Navigator includes built-in [Vector](https://vector.dev/) integration for centralized log aggregation.

### Features

- **Automatic Process Management** - Starts and manages Vector as a high-priority process
- **Unix Socket Streaming** - Efficient log transfer without file I/O overhead
- **Source Identification** - Automatically tags logs with source (tenant, process name, stream)
- **Multiple Destinations** - Route logs to Elasticsearch, S3, Datadog, or any Vector sink
- **Graceful Degradation** - Continues operating if Vector is unavailable

### Configuration

**Navigator configuration:**

```yaml
logging:
  format: json  # Vector works best with JSON logs
  vector:
    enabled: true
    socket: /tmp/navigator.sock
    config: /etc/vector/vector.toml
```

**Vector configuration** (`/etc/vector/vector.toml`):

```toml
[sources.navigator]
type = "socket"
mode = "unix"
path = "/tmp/navigator.sock"
max_length = 102400

[transforms.parse_json]
type = "remap"
inputs = ["navigator"]
source = '''
  . = parse_json!(string!(.message))
'''

# Send to Elasticsearch
[sinks.elasticsearch]
type = "elasticsearch"
inputs = ["parse_json"]
endpoint = "http://elasticsearch:9200"
index = "navigator-logs-%Y.%m.%d"

# Send to file
[sinks.file]
type = "file"
inputs = ["parse_json"]
path = "/var/log/navigator/aggregated-%Y-%m-%d.log"
encoding.codec = "json"
```

### Benefits

- **Centralized logging** - All logs in one place across machines
- **Real-time processing** - Transform and route logs as they arrive
- **Multiple outputs** - Send to multiple destinations simultaneously
- **Performance** - Unix socket streaming is faster than file-based collection
- **Reliability** - Vector handles backpressure and retry logic

### Example Workflow

1. Navigator processes send logs to Unix socket
2. Vector receives logs via socket source
3. Vector transforms logs (parse JSON, add metadata, filter)
4. Vector routes to multiple destinations (Elasticsearch, S3, files)
5. Logs available for search, analysis, alerting

See [navigator-with-vector.yml](https://github.com/rubys/navigator/blob/main/examples/navigator-with-vector.yml) and [vector.toml](https://github.com/rubys/navigator/blob/main/examples/vector.toml) for complete examples.

## See Also

- [Monitoring](../deployment/monitoring.md) - Log analysis, retention, and alerting
- [Production Deployment](../deployment/production.md) - systemd journal integration
- [Configuration Reference](../configuration/yaml-reference.md) - Complete logging options