# Reference

Complete reference documentation for Navigator command-line interface, environment variables, signals, and system integration.

## Command-Line Interface

Navigator provides a simple command-line interface for starting the server, reloading configuration, and getting help.

### Basic Usage

```bash
# Start Navigator with default config
navigator

# Start with specific config file
navigator /path/to/config.yml

# Display help
navigator --help

# Show version information
navigator --version
```

[Complete CLI Reference](cli.md)

## Configuration Files

Navigator uses YAML configuration files with a specific structure and syntax.

### File Locations

Navigator looks for configuration files in this order:

1. **Command line argument**: `navigator /path/to/config.yml`
2. **Default location**: `config/navigator.yml`
3. **Current directory**: `navigator.yml`

### Environment Variable Substitution

Use `${VAR}` syntax for environment variable substitution:

```yaml
applications:
  global_env:
    DATABASE_URL: "${DATABASE_URL}"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"
```

[Complete YAML Reference](../configuration/yaml-reference.md)

## Environment Variables

Navigator recognizes several environment variables for configuration and behavior control.

### Navigator-Specific Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `NAVIGATOR_PID_FILE` | PID file location | `/tmp/navigator.pid` |

### Fly.io Variables

| Variable | Purpose | Required For |
|----------|---------|-------------|
| `FLY_APP_NAME` | Application name | Fly-Replay fallback |
| `FLY_MACHINE_ID` | Machine ID | Machine suspension |

[Complete Environment Reference](environment.md)

## Signals

Navigator responds to standard Unix signals for process management and configuration control.

### Supported Signals

| Signal | Action | Description |
|--------|--------|-------------|
| `SIGHUP` | Reload configuration | Live config reload without restart |
| `SIGTERM` | Graceful shutdown | Stop all processes cleanly |
| `SIGINT` | Graceful shutdown | Same as SIGTERM (Ctrl+C) |
| `SIGQUIT` | Immediate shutdown | Force shutdown all processes |

### Signal Usage

```bash
# Reload configuration
kill -HUP $(cat /tmp/navigator.pid)

# Graceful shutdown
kill -TERM $(cat /tmp/navigator.pid)

# Force shutdown
kill -QUIT $(cat /tmp/navigator.pid)
```

[Complete Signal Reference](signals.md)

## System Integration

Navigator integrates well with modern system service managers and container orchestrators.

### systemd Service

```ini title="/etc/systemd/system/navigator.service"
[Unit]
Description=Navigator Rails Server
After=network.target

[Service]
Type=simple
User=rails
WorkingDirectory=/var/www/app
ExecStart=/usr/local/bin/navigator /etc/navigator/config.yml
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Docker Integration

```dockerfile
FROM ruby:3.2-slim

# Install Navigator
COPY navigator /usr/local/bin/navigator
RUN chmod +x /usr/local/bin/navigator

# Setup Rails app
WORKDIR /app
COPY . .
RUN bundle install

# Navigator configuration
COPY navigator.yml /app/
EXPOSE 3000

# Start Navigator
CMD ["navigator", "/app/navigator.yml"]
```

### Process Management

Navigator creates and manages several types of processes:

- **Navigator main process**: HTTP server and process manager
- **Rails processes**: Application instances on dynamic ports
- **Managed processes**: External services (Redis, Sidekiq, etc.)

## API Reference

Navigator doesn't provide a formal API, but exposes several interfaces:

### HTTP Endpoints

| Endpoint | Purpose | Method |
|----------|---------|---------|
| `/*` | Application routing | ALL |
| `/up` | Health check | GET |

### Internal Interfaces

- **Process communication**: HTTP proxy to Rails processes
- **Signal handling**: Unix signals for control
- **File system**: Configuration files and PID files

## Error Codes

Navigator uses standard HTTP status codes and Unix exit codes.

### HTTP Status Codes

| Code | Meaning | Cause |
|------|---------|-------|
| `200` | Success | Normal request processing |
| `401` | Unauthorized | Authentication required |
| `403` | Forbidden | Authentication failed |
| `404` | Not Found | No matching route |
| `502` | Bad Gateway | Rails process unavailable |
| `503` | Service Unavailable | All processes busy |

### Exit Codes

| Code | Meaning | Cause |
|------|---------|-------|
| `0` | Success | Normal shutdown |
| `1` | General error | Configuration or runtime error |
| `2` | Usage error | Invalid command line arguments |

## File Formats

Navigator uses standard file formats for configuration and data storage.

### YAML Configuration

Navigator uses YAML 1.2 format with specific schema requirements:

```yaml
# Comments supported
server:
  listen: 3000  # Port number
  hostname: "localhost"  # String values

pools:
  max_size: 10  # Integer values
  idle_timeout: 300

applications:
  tenants:
    - name: app1  # Array of objects
      path: /
```

### PID Files

Standard Unix PID file format:
- Contains single process ID number
- Created at startup, removed at shutdown
- Used for signal delivery and process tracking

### htpasswd Files

Standard Apache htpasswd format:
- One user per line: `username:password_hash`
- Supports multiple hash formats (APR1, bcrypt, SHA)
- Compatible with Apache/nginx htpasswd files

## Logging Format

Navigator uses structured logging with the Go `slog` package.

### Log Levels

| Level | Usage | Example |
|-------|--------|---------|
| `DEBUG` | Detailed information | Request routing decisions |
| `INFO` | General information | Process starts/stops |
| `WARN` | Warning conditions | Configuration issues |
| `ERROR` | Error conditions | Process failures |

### Log Format

```
2024-09-02T17:20:42Z INFO Starting process app=main port=4001 pid=12345
2024-09-02T17:20:42Z DEBUG Request routed path=/api/users method=GET app=main
2024-09-02T17:20:45Z WARN Process idle timeout app=main idle_time=300s
```

## Performance Characteristics

Navigator is designed for efficiency and low resource usage.

### Resource Usage

| Resource | Typical Usage | Notes |
|----------|---------------|--------|
| **Memory** | 20-50 MB base | Plus Rails process memory |
| **CPU** | <5% idle | Scales with request volume |
| **File descriptors** | 10-50 base | Plus Rails process FDs |
| **Network ports** | 1 listen port | Plus Rails process ports |

### Throughput

- **Static files**: 1000+ requests/second
- **Rails proxy**: Depends on Rails app performance
- **Process startup**: 1-5 seconds per Rails process

### Limits

| Limit | Default | Configurable |
|-------|---------|-------------|
| **Max processes** | 10 | `pools.max_size` |
| **Port range** | 100 ports | `pools.start_port` |
| **File size** | No limit | OS/filesystem limits |
| **Request size** | No limit | May trigger fallback behavior |

## Compatibility

Navigator is designed to be compatible with modern systems and standards.

### Go Version

- **Minimum**: Go 1.21
- **Tested**: Go 1.21, 1.22, 1.23
- **Recommended**: Latest stable Go version

### Operating Systems

| OS | Support | Notes |
|----|---------|--------|
| **Linux** | Full | Primary development platform |
| **macOS** | Full | Apple Silicon and Intel |
| **FreeBSD** | Basic | Community tested |
| **Windows** | Limited | WSL recommended |

### Rails Versions

| Rails Version | Support | Notes |
|---------------|---------|--------|
| **7.x** | Full | Primary target |
| **6.x** | Full | Well tested |
| **5.x** | Basic | Limited testing |

### Ruby Versions

Navigator works with Rails applications running on:
- Ruby 3.0+
- Ruby 2.7+ (with Rails 6.x)
- JRuby 9.3+
- TruffleRuby (experimental)

## Standards Compliance

Navigator follows established standards and conventions:

- **HTTP/1.1**: RFC 7230-7237 compliance
- **Unix signals**: POSIX signal handling
- **File formats**: Standard YAML, htpasswd formats
- **Process management**: Unix process model
- **Logging**: Structured logging with slog

## See Also

- [CLI Reference](cli.md) - Complete command-line documentation
- [Environment Variables](environment.md) - All environment variables
- [Signal Handling](signals.md) - Unix signal reference
- [Configuration Reference](../configuration/yaml-reference.md) - YAML configuration
- [Examples](../examples/index.md) - Working configuration examples