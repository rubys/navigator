# Logging

Navigator provides comprehensive structured logging with built-in Vector integration for enterprise-grade log aggregation.

## Overview

Navigator captures and structures logs from three sources:

1. **HTTP Access Logs** - All incoming HTTP requests with timing, status codes, client IPs
2. **Managed Process Logs** - stdout/stderr from external processes (Redis, Sidekiq, workers)
3. **Web Application Logs** - stdout/stderr from Rails, Django, Node.js applications

All logs can be sent to Vector for centralized aggregation, transformation, and routing to multiple destinations.

## Log Levels

Control Navigator's operational logging verbosity with the `LOG_LEVEL` environment variable:

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

## HTTP Access Logs

Navigator logs all HTTP requests in JSON format with comprehensive metadata:

```json
{
  "@timestamp": "2025-01-10T18:30:45.123-05:00",
  "client_ip": "192.168.1.100",
  "remote_user": "john",
  "method": "GET",
  "uri": "/showcase/studios/",
  "protocol": "HTTP/1.1",
  "status": 200,
  "body_bytes_sent": 15234,
  "request_id": "a1b2c3d4e5f6",
  "request_time": "0.045",
  "referer": "https://example.com/",
  "user_agent": "Mozilla/5.0...",
  "fly_request_id": "01ABC123XYZ",
  "tenant": "2025/boston",
  "response_type": "proxy",
  "proxy_backend": "tenant:",
  "file_path": "/path/to/static/file.html"
}
```

### Access Log Fields

- `@timestamp` - ISO 8601 timestamp with timezone
- `client_ip` - Client IP (from X-Forwarded-For if present)
- `remote_user` - Authenticated username or "-"
- `method` - HTTP method (GET, POST, etc.)
- `uri` - Full URI including query parameters
- `protocol` - HTTP protocol version
- `status` - HTTP status code
- `body_bytes_sent` - Response body size in bytes
- `request_id` - Unique request identifier
- `request_time` - Request duration in seconds
- `referer` - HTTP Referer header
- `user_agent` - User-Agent string
- `fly_request_id` - Fly.io request ID (if running on Fly.io)
- `tenant` - Tenant name for multi-tenant apps (optional)
- `response_type` - How request was handled: `proxy`, `static`, `redirect`, `fly-replay`, `auth-failure`, `error`
- `proxy_backend` - Backend that handled proxied request (optional)
- `file_path` - Path to served static file (optional)
- `destination` - Fly-replay or redirect destination (optional)
- `error_message` - Error description for failed requests (optional)

## Process Output Capture

Navigator captures all stdout/stderr from managed processes and web applications with source identification:

**JSON format**:
```json
{"@timestamp":"2025-01-10T18:30:45Z","source":"redis","stream":"stdout","message":"Ready to accept connections"}
{"@timestamp":"2025-01-10T18:30:46Z","source":"2025/boston","stream":"stderr","message":"Error: Connection refused","tenant":"boston"}
```

### Process Log Fields

- `@timestamp` - ISO 8601 timestamp
- `source` - Process or application name
- `stream` - `stdout` or `stderr`
- `message` - Log message text
- `tenant` - Tenant name for web applications (optional)

## Vector Integration

Navigator includes professional-grade log aggregation through [Vector](https://vector.dev/), enabling centralized logging across multiple machines and services.

### What Gets Sent to Vector

When Vector integration is enabled, Navigator sends:

- ✅ **HTTP Access Logs** - All incoming HTTP requests
- ✅ **Managed Process Logs** - Redis, Sidekiq, worker stdout/stderr
- ✅ **Web Application Logs** - Rails, Django, Node.js stdout/stderr
- ❌ **Navigator Operational Logs** - Navigator's own slog messages (use process supervisor for these)

### Features

- **Automatic Process Management** - Navigator starts and manages Vector as a high-priority process
- **Unix Socket Streaming** - Efficient log transfer without file I/O overhead
- **Complete Log Aggregation** - HTTP access logs and all process logs in one place
- **Source Identification** - Automatically tags logs with source (tenant, process name, stream)
- **Multiple Destinations** - Route logs to NATS, Elasticsearch, S3, Datadog, or any Vector sink
- **Graceful Degradation** - Continues operating if Vector is unavailable
- **Zero Application Changes** - Works with any framework without code modifications

### Configuration

**Navigator configuration** (`config/navigator.yml`):

```yaml
logging:
  format: json  # Vector requires JSON logs
  vector:
    enabled: true
    socket: /tmp/navigator-vector.sock
    config: /etc/vector/vector.toml
```

**Vector configuration** (`/etc/vector/vector.toml`):

```toml
# Global settings
data_dir = "/var/lib/vector"

# Receive logs from Navigator via Unix socket
[sources.navigator]
type = "socket"
mode = "unix"
path = "/tmp/navigator-vector.sock"

# Parse JSON logs from Navigator
[transforms.parse_json]
type = "remap"
inputs = ["navigator"]
source = '''
  . = parse_json!(string!(.message))
'''

# Send to NATS for real-time distribution
[sinks.nats]
type = "nats"
inputs = ["parse_json"]
url = "nats://localhost:4222"
subject = "logs.{{ source }}"
encoding.codec = "json"

# Send to Elasticsearch for search and analysis
[sinks.elasticsearch]
type = "elasticsearch"
inputs = ["parse_json"]
endpoint = "http://elasticsearch:9200"
index = "navigator-logs-%Y.%m.%d"

# Write to local files for backup
[sinks.file]
type = "file"
inputs = ["parse_json"]
path = "/var/log/navigator/aggregated-%Y-%m-%d.log"
encoding.codec = "json"
```

### Vector Startup

Navigator automatically starts Vector when `logging.vector.enabled: true`:

1. Vector is injected as the first managed process (high priority)
2. Stale Unix socket cleaned up before starting
3. Vector connects to Navigator's socket
4. Navigator begins streaming logs to Vector
5. Vector transforms and routes logs to configured sinks

If Vector fails or is unavailable, Navigator continues operating normally and logs to stdout.

### Example Workflow

```
┌─────────────┐
│   Clients   │
└──────┬──────┘
       │ HTTP Requests
       ▼
┌─────────────────────────────────────────┐
│           Navigator                      │
│  ┌────────────────────────────────────┐ │
│  │  HTTP Access Logs (JSON)           │ │
│  │  Managed Process Logs (JSON)       │ │
│  │  Web Application Logs (JSON)       │ │
│  └────────┬───────────────────────────┘ │
└───────────┼─────────────────────────────┘
            │ Unix Socket
            ▼
     ┌─────────────┐
     │   Vector    │
     │  Transform  │
     │   & Route   │
     └──────┬──────┘
            │
    ┌───────┼────────┐
    │       │        │
    ▼       ▼        ▼
┌──────┐ ┌────┐  ┌─────┐
│ NATS │ │ S3 │  │ ELK │
└──────┘ └────┘  └─────┘
```

1. Navigator receives HTTP requests and generates access logs
2. Managed processes and web apps send stdout/stderr to Navigator
3. Navigator streams all logs to Vector via Unix socket
4. Vector parses JSON, adds metadata, filters if needed
5. Vector routes logs to multiple destinations (NATS, S3, Elasticsearch)
6. Logs available for real-time monitoring, search, analysis, alerting

### Benefits

- **Centralized logging** - All logs from all machines in one place
- **Real-time processing** - Transform and route logs as they arrive
- **Multiple outputs** - Send to multiple destinations simultaneously
- **Performance** - Unix socket streaming is faster than file-based collection
- **Reliability** - Vector handles backpressure, buffering, and retry logic
- **Flexibility** - Vector supports 50+ sources, transforms, and sinks
- **Production-ready** - Battle-tested by companies processing billions of events

## Configuration Examples

### Minimal Configuration (JSON logs only)

```yaml
logging:
  format: json
```

Logs to stdout in JSON format. Suitable for:
- Development
- Container environments with external log collection (Docker, Kubernetes)
- systemd with journal forwarding

### Production Configuration (Vector aggregation)

```yaml
logging:
  format: json
  vector:
    enabled: true
    socket: /tmp/navigator-vector.sock
    config: /etc/vector/vector.toml
```

Complete log aggregation setup. Suitable for:
- Production deployments
- Multi-machine environments
- Centralized logging infrastructure
- Compliance and auditing requirements

### Configuration Reload

Logging configuration can be reloaded without restarting Navigator:

```bash
# Send reload signal
navigator -s reload

# Or use kill command
kill -HUP $(cat /tmp/navigator.pid)
```

Changes to `logging.vector` settings take effect immediately, including:
- Starting Vector if newly enabled
- Stopping Vector if disabled
- Updating Vector configuration path

## See Also

- [Monitoring](../deployment/monitoring.md) - Log analysis, retention, and alerting
- [Configuration Reference](../configuration/yaml-reference.md) - Complete logging options
- [Managed Processes](managed-processes.md) - External process management
- [Vector Documentation](https://vector.dev/docs/) - Complete Vector documentation
