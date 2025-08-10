# Navigator

Navigator is a modern Go-based web server that replaces nginx/Passenger for multi-tenant Rails applications. It provides intelligent request routing, dynamic process management, authentication, and high-performance caching.

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

## Architecture

Built with modern Go libraries:
- **[Chi Router](https://github.com/go-chi/chi)**: Fast HTTP router with middleware support
- **[Logrus](https://github.com/sirupsen/logrus)**: Structured logging with JSON output
- **[HTTP Cache](https://github.com/victorspringer/http-cache)**: Memory-based caching with LRU eviction

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
./navigator -rails-root /path/to/rails/app -listen :3000
```

## Configuration

Navigator uses command-line flags for configuration:

### Required Arguments

```bash
-rails-root string
    Rails application root directory (required)
```

### Optional Arguments

```bash
-listen string
    Address to listen on (default ":3000")
-showcases string
    Path to showcases.yml relative to rails-root (default "config/tenant/showcases.yml")
-db-path string
    Database directory path (default "db")
-storage string
    Storage directory path (default "storage")
-htpasswd string
    Path to htpasswd file
-url-prefix string
    URL prefix to strip from requests (default "/showcase")
-max-puma int
    Maximum number of concurrent Puma processes (default 10)
-idle-timeout duration
    Idle timeout before stopping Puma process (default 5m0s)
-log-level string
    Log level: debug, info, warn, error (default "info")
```

### Example Usage

```bash
# Basic usage
./navigator -rails-root /opt/rails/showcase

# With custom settings
./navigator \
  -rails-root /opt/rails/showcase \
  -listen :8080 \
  -max-puma 20 \
  -idle-timeout 10m \
  -log-level debug \
  -htpasswd /etc/navigator/htpasswd
```

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
./navigator -rails-root /path/to/app -log-level debug
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
│   ├── config/            # Configuration and YAML parsing  
│   ├── logger/            # Structured logging wrapper
│   ├── manager/           # Puma process management
│   ├── proxy/             # HTTP routing, caching, auth
│   └── server/            # HTTP/2 server implementation
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
ExecStart=/usr/local/bin/navigator -rails-root /opt/rails/app -listen :3000
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
CMD ["./navigator", "-rails-root", "/app"]
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
./navigator -max-puma 50  # Increase port range
```

**Authentication failures**: Check htpasswd format
```bash
./navigator -log-level debug  # See auth details
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

- **Go 1.19+**: Modern Go features
- **Chi v5**: HTTP router with middleware
- **Logrus**: Structured logging
- **HTTP-Cache**: Memory caching with LRU
- **htpasswd**: Password file support
- **YAML v3**: Configuration parsing

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