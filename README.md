# Navigator

Navigator is a modern Go-based web server that replaces nginx/Passenger for multi-tenant Rails applications. It provides intelligent request routing, dynamic process management, authentication, HTTP/2 support, and high-performance caching with flexible configuration options.

## Features

- **Multi-tenant Support**: Routes requests to appropriate Rails instances based on URL patterns
- **Dynamic Process Management**: Starts and stops Puma processes on-demand with configurable idle timeout
- **HTTP Caching**: 100MB memory cache with LRU eviction for static assets (70% performance improvement)
- **Structured Logging**: JSON logs with contextual fields using logrus
- **HTTP/2 Support**: Modern HTTP/2 protocol with h2c (cleartext) support
- **Automatic Process Recovery**: Detects crashed processes and restarts them automatically
- **Authentication**: HTTP Basic authentication support via htpasswd files
- **Smart Asset Serving**: Different cache TTLs for fingerprinted vs regular assets
- **Compression**: Gzip compression for text-based content
- **Flexible Configuration**: YAML files, environment variables, and command-line flags
- **Modern CLI**: Cobra-powered CLI with subcommands and comprehensive help

## Architecture

Built with modern Go libraries:
- **[Chi Router](https://github.com/go-chi/chi)**: Fast HTTP router with middleware support
- **[Logrus](https://github.com/sirupsen/logrus)**: Structured logging with JSON output
- **[HTTP Cache](https://github.com/victorspringer/http-cache)**: Memory-based caching with LRU eviction
- **[Cobra](https://github.com/spf13/cobra)**: Modern CLI framework with subcommands
- **[Viper](https://github.com/spf13/viper)**: Configuration management with YAML/ENV support

## Installation

```bash
# Clone the repository
git clone https://github.com/rubys/navigator.git
cd navigator

# Install dependencies
go mod download

# Build the binary
go build -o navigator cmd/navigator/main.go
```

## Quick Start

```bash
# Start Navigator with command-line flags
./navigator serve --root /path/to/rails/app --listen :3000

# Or start in the current directory (uses '.' as default)
./navigator serve

# Or use environment variables
NAVIGATOR_RAILS_ROOT=/path/to/rails/app ./navigator serve

# Or use a configuration file (automatically looks for config/navigator.yml)
./navigator serve

# Or specify a custom configuration file
./navigator serve --config /path/to/navigator.yaml

# Get help and see all available commands
./navigator --help
./navigator serve --help
```

## Configuration

Navigator supports three configuration methods (in order of precedence):

### 1. Command-Line Flags (Highest Priority)

```bash
./navigator serve \
  --root /path/to/rails/app \
  --listen :3000 \
  --url-prefix /showcase \
  --max-puma 20 \
  --idle-timeout 10m \
  --log-level debug \
  --htpasswd /path/to/htpasswd
```

### 2. Environment Variables

All configuration can be set via environment variables with the `NAVIGATOR_` prefix:

```bash
export NAVIGATOR_RAILS_ROOT="/path/to/rails/app"
export NAVIGATOR_SERVER_LISTEN=":3000"
export NAVIGATOR_SERVER_URL_PREFIX="/showcase"
export NAVIGATOR_MANAGER_MAX_PUMA=20
export NAVIGATOR_MANAGER_IDLE_TIMEOUT="10m"
export NAVIGATOR_AUTH_HTPASSWD_FILE="/path/to/htpasswd"
export NAVIGATOR_LOGGING_LEVEL="debug"

./navigator serve
```

### 3. YAML Configuration File (Lowest Priority)

Navigator automatically looks for `config/navigator.yml` relative to the root directory. You can also create a custom `navigator.yaml` file:

```yaml
server:
  listen: ":3000"
  url_prefix: "/showcase"

rails:
  root: "/path/to/rails/app"
  showcases: "config/tenant/showcases.yml"
  db_path: "db"
  storage: "storage"

manager:
  max_puma: 20
  idle_timeout: "10m"

auth:
  htpasswd_file: "/path/to/htpasswd"

logging:
  level: "debug"
```

Then run:
```bash
# Uses config/navigator.yml if present in root directory
./navigator serve

# Or specify a custom config file
./navigator serve --config navigator.yaml
```

### Configuration Options

| Option | CLI Flag | Environment Variable | Default | Description |
|--------|----------|---------------------|---------|-------------|
| Rails Root | `--root` | `NAVIGATOR_RAILS_ROOT` | `.` | Application root directory |
| Listen Address | `--listen` | `NAVIGATOR_SERVER_LISTEN` | `:3000` | HTTP server bind address |
| URL Prefix | `--url-prefix` | `NAVIGATOR_SERVER_URL_PREFIX` | `/showcase` | URL prefix to strip |
| Max Puma | `--max-puma` | `NAVIGATOR_MANAGER_MAX_PUMA` | `10` | Max concurrent Puma processes |
| Idle Timeout | `--idle-timeout` | `NAVIGATOR_MANAGER_IDLE_TIMEOUT` | `5m` | Process idle timeout |
| Htpasswd File | `--htpasswd` | `NAVIGATOR_AUTH_HTPASSWD_FILE` | `` | Authentication file path |
| Log Level | `--log-level` | `NAVIGATOR_LOGGING_LEVEL` | `info` | Log level (debug/info/warn/error) |

## Performance

Navigator provides significant performance improvements:

- **Static Asset Caching**: 70% faster response times (1.49ms → 0.45ms)
- **Memory Cache**: 100MB LRU cache with 1-hour TTL
- **Smart TTL**: 24-hour cache for fingerprinted assets
- **HTTP/2**: Modern protocol support with multiplexing
- **Compression**: Automatic gzip compression

## Structured Logging

All logs are output in JSON format for easy parsing:

```json
{
  "level": "info",
  "message": "HTTP request",
  "method": "GET",
  "path": "/studios/",
  "status": 200,
  "duration_ms": 45,
  "remote_addr": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "request_id": "req-123",
  "timestamp": "2025-08-10T12:00:00.000Z"
}
```

Enable debug logging for detailed tenant routing:
```bash
./navigator serve --rails-root /path/to/app --log-level debug
```

## CLI Commands

Navigator provides several commands for different operations:

### `navigator serve`
Start the Navigator server (main command):
```bash
./navigator serve --rails-root /path/to/rails/app
```

### `navigator config validate`
Validate configuration and display resolved settings:
```bash
./navigator config validate --rails-root /path/to/app
./navigator config validate --config navigator.yaml
```

This command will:
- Load and validate configuration from all sources
- Check that required files exist (Rails root, showcases.yml)
- Display the final resolved configuration
- Show configured tenants

### `navigator version`
Display version information:
```bash
./navigator version
```

### `navigator --help`
Show comprehensive help:
```bash
./navigator --help           # Show all commands
./navigator serve --help     # Show serve command options
./navigator config --help    # Show config subcommands
```

## Rails Integration

### showcases.yml Format

Navigator expects a `showcases.yml` file with Ruby-style symbols:

```yaml
"2025":
  raleigh:
    :name: "Raleigh Studio"
    :region: "us-east"
    :events:
      disney:
        :name: "Disney Theme"
        :date: "2025-03-15"
      summer:
        :name: "Summer Showcase"
        :date: "2025-07-20"
```

### Environment Variables Set by Navigator

For each Rails instance, Navigator sets:

- `RAILS_APP_DB`: Tenant database identifier (e.g., "2025-raleigh-disney")
- `RAILS_APP_OWNER`: Tenant owner name
- `RAILS_APP_SCOPE`: URL scope (e.g., "2025/raleigh/disney")
- `DATABASE_URL`: SQLite database URL
- `RAILS_STORAGE`: Storage path for the tenant
- `RAILS_ENV`: Always "production"
- `RAILS_PROXY_HOST`: Proxy host for URL generation

## Multi-Tenant Routing

Navigator routes requests based on URL patterns:

- `/` → Redirects to `/studios/`
- `/studios/` → `index` tenant
- `/2025/raleigh/disney/` → `2025-raleigh-disney` tenant
- Static assets served directly with caching

## Process Management

- **On-Demand Startup**: Puma processes start on first request
- **Idle Cleanup**: Processes stopped after configurable timeout (default: 5 minutes)
- **Automatic Recovery**: Crashed processes detected and restarted automatically
- **Port Management**: Dynamic port allocation starting from 4000
- **Health Monitoring**: Continuous process health checks

## Authentication

HTTP Basic authentication via htpasswd files:

```bash
# Create htpasswd file
htpasswd -c /path/to/htpasswd username

# Add users
htpasswd /path/to/htpasswd another_user

# Use with Navigator
./navigator -rails-root /path/to/app -htpasswd /path/to/htpasswd
```

Supports APR1, Bcrypt, SHA, and Crypt hash formats.

## Development

### Project Structure

```
navigator/
├── cmd/navigator/          # Main application entry point
├── internal/
│   ├── cli/               # Cobra CLI commands and Viper configuration
│   ├── config/            # Configuration and YAML parsing  
│   ├── logger/            # Structured logging wrapper
│   ├── manager/           # Puma process management
│   ├── proxy/             # HTTP routing, caching, auth
│   └── server/            # HTTP/2 server implementation
├── config/                # Example configuration files
├── go.mod                 # Go module definition
├── README.md              # This file
└── LICENSE               # MIT license
```

### Building

```bash
# Standard build
go build -o navigator cmd/navigator/main.go

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o navigator cmd/navigator/main.go

# Build with version info
go build -ldflags "-X main.version=1.0.0" -o navigator cmd/navigator/main.go
```

### Testing

```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...

# With race detection  
go test -race ./...

# Manual testing
curl http://localhost:3000/studios/
curl http://localhost:3000/2025/raleigh/disney/
```

## Deployment

### Systemd Service

Create `/etc/systemd/system/navigator.service`:

```ini
[Unit]
Description=Navigator Rails Proxy
After=network.target

[Service]
Type=simple
User=rails
WorkingDirectory=/opt/rails/app
ExecStart=/usr/local/bin/navigator serve --rails-root /opt/rails/app --listen :3000
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable navigator
sudo systemctl start navigator
sudo journalctl -u navigator -f  # View logs
```

### Docker

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o navigator cmd/navigator/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/navigator .
EXPOSE 3000
CMD ["./navigator", "serve", "--rails-root", "/app"]
```

## Monitoring

Navigator provides health check endpoints:

- `GET /up` - Basic health check
- `GET /health` - Detailed health check

All operations are logged with structured JSON including:
- Request details (method, path, status, duration)
- Process management events
- Cache hit/miss statistics  
- Error conditions with context

## Troubleshooting

### Common Issues

**Port conflicts**: Navigator manages ports 4000+ automatically
```bash
./navigator serve --max-puma 50  # Increase port range
```

**Authentication failures**: Check htpasswd format
```bash
./navigator serve --log-level debug  # See auth details
```

**Tenant not found**: Verify showcases.yml symbols
```bash
# Correct: :name
# Incorrect: name
```

**High memory usage**: Adjust cache size by editing source:
```go
memory.AdapterWithCapacity(50000000)  // 50MB instead of 100MB
```

### Performance Tuning

1. **Cache Size**: Adjust memory cache based on available RAM
2. **Process Limits**: Set `max-puma` based on CPU cores
3. **Idle Timeout**: Balance resource usage vs startup latency
4. **Log Level**: Use "warn" or "error" in production for performance

## Dependencies

- **Go 1.22+**: Modern Go features and performance
- **Chi v5**: HTTP router with middleware support
- **Logrus**: Structured JSON logging
- **HTTP-Cache**: Memory caching with LRU eviction
- **Cobra**: Modern CLI framework with subcommands
- **Viper**: Configuration management (YAML/ENV/flags)
- **htpasswd**: Multi-format password file support
- **YAML v3**: Configuration file parsing

## Contributing

Pull requests welcome! Please ensure:
- Code follows Go conventions (`go fmt`, `go vet`)
- Tests pass (`go test ./...`)
- Documentation is updated
- Commit messages are descriptive

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- **Issues**: Use GitHub issue tracker
- **Documentation**: See [CLAUDE.md](CLAUDE.md) for development details
- **Performance**: All requests logged with timing information