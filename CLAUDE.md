# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the Navigator project.

## Project Overview

Navigator is a modern Go-based web server that replaces nginx/Passenger for multi-tenant Rails applications. It provides intelligent request routing, dynamic process management, authentication, HTTP caching, and automatic process recovery for Rails applications with multiple tenants.

## Current Implementation Status

✅ **Chi Router Integration**: Replaced custom routing with github.com/go-chi/chi/v5
✅ **Structured Logging**: Implemented JSON logging with github.com/sirupsen/logrus  
✅ **HTTP Caching**: Added memory caching with github.com/victorspringer/http-cache
✅ **Viper Configuration**: YAML/ENV/CLI support with github.com/spf13/viper
✅ **Cobra CLI**: Full CLI with subcommands using github.com/spf13/cobra

## Architecture

### Core Components

1. **HTTP Server** (`internal/server/server.go`)
   - HTTP/2 support with h2c (HTTP/2 over cleartext)
   - Chi router integration with middleware stack
   - Graceful shutdown handling with context cancellation

2. **Request Router** (`internal/proxy/router.go`)
   - **Chi-based routing** with built-in middleware pipeline
   - **Multi-tenant routing** based on URL patterns and showcases.yml
   - **HTTP caching middleware** with 100MB LRU memory cache
   - **Smart cache TTL**: 1h for regular assets, 24h for fingerprinted assets  
   - **Static asset serving** with try_files logic and compression
   - **Automatic process recovery** on connection failures
   - **Authentication middleware** integration

3. **Process Manager** (`internal/manager/puma.go`)
   - **Dynamic Puma lifecycle** management with on-demand startup
   - **Port allocation** starting from 4000 with automatic management
   - **Idle process cleanup** with configurable timeouts (default 5min)
   - **Process health monitoring** and crash detection
   - **Graceful shutdown** with SIGTERM/SIGKILL progression

4. **Structured Logging** (`internal/logger/logger.go`)
   - **JSON log output** with contextual fields using logrus
   - **Request tracing** with unique request IDs
   - **Performance metrics** with duration tracking
   - **Configurable log levels**: debug, info, warn, error

5. **Configuration System** (`internal/config/showcases.go`)
   - **Ruby-style YAML parsing** (supports `:symbol` syntax)
   - **Multi-tenant configuration** from showcases.yml
   - **Environment variable generation** for Rails instances
   - **Tenant lookup** with longest-match routing logic

6. **Authentication** (`internal/proxy/auth.go`)
   - **HTTP Basic authentication** via htpasswd files
   - **Multi-format support**: APR1, Bcrypt, SHA, and Crypt hashes
   - **Public/private path** routing logic

## Development Commands

### Building and Running

```bash
# Build the Navigator binary
go build -o navigator cmd/navigator/main.go

# Run with command-line arguments
./navigator serve --rails-root /path/to/rails/app --listen :3000

# Run with configuration file
./navigator serve --config configs/navigator.yaml

# Run with environment variables
NAVIGATOR_RAILS_ROOT=/path/to/app ./navigator serve

# Build and run in one command
go build -o navigator cmd/navigator/main.go && ./navigator serve --rails-root /Users/rubys/git/showcase

# View help and available commands
./navigator --help
./navigator serve --help

# Validate configuration
./navigator config validate --rails-root /path/to/app
```

### Development Workflow

```bash
# Install dependencies
go mod download
go mod tidy

# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o navigator-linux cmd/navigator/main.go
GOOS=darwin GOARCH=arm64 go build -o navigator-darwin-arm64 cmd/navigator/main.go
```

## Key Features

### 1. Multi-Tenant Routing

Navigator routes requests to appropriate Rails instances based on URL patterns:

- `/studios/` → `index` tenant
- `/2025/raleigh/disney/` → `2025-raleigh-disney` tenant
- Static assets served directly with caching

### 2. Automatic Process Recovery

When a Puma process dies, Navigator automatically:
1. Detects connection refused errors
2. Identifies the affected tenant
3. Clears stale proxy connections
4. Restarts the process via `Manager.GetOrStart()`
5. Retries the original request

**Implementation**: See `proxyToTenant()` in `internal/proxy/router.go:162-215`

### 3. Process Management

- **Dynamic Startup**: Processes started on first request
- **Idle Cleanup**: Configurable timeout (default: 5 minutes)
- **Port Management**: Automatic port allocation starting from 4000
- **Health Monitoring**: Process crash detection and cleanup

### 4. Asset Optimization

- **Caching**: ETag-based caching with immutable cache headers
- **Compression**: Gzip compression for text-based assets
- **Try Files**: nginx-style file resolution (tries `.html`, `/index.html`)

## Configuration

### Configuration Methods

Navigator supports three configuration methods (in order of precedence):

1. **Command-line flags** (highest priority):
```bash
./navigator serve \
  --rails-root /path/to/rails/app \
  --listen :3000 \
  --url-prefix /showcase \
  --max-puma 20 \
  --idle-timeout 10m \
  --htpasswd /path/to/htpasswd
```

2. **Environment variables**:
```bash
export NAVIGATOR_RAILS_ROOT="/path/to/rails/app"
export NAVIGATOR_SERVER_LISTEN=":3000"
export NAVIGATOR_SERVER_URL_PREFIX="/showcase"
export NAVIGATOR_MANAGER_MAX_PUMA=20
export NAVIGATOR_MANAGER_IDLE_TIMEOUT="10m"
export NAVIGATOR_AUTH_HTPASSWD_FILE="/path/to/htpasswd"
./navigator serve
```

3. **YAML configuration file** (lowest priority):
```yaml
# config/navigator.yaml
server:
  listen: ":3000"
  url_prefix: "/showcase"

rails:
  root: "/path/to/rails/app"

manager:
  max_puma: 20
  idle_timeout: "10m"

auth:
  htpasswd_file: "/path/to/htpasswd"

logging:
  level: "info"
```

Then run: `./navigator serve --config config/navigator.yaml`

### Rails Integration

Navigator expects Rails applications with:

1. **Multi-tenant structure**: SQLite databases per tenant
2. **showcases.yml**: Tenant configuration with Ruby symbols
3. **Environment variables**: Set by Navigator for each Rails instance

Example environment variables set by Navigator:
- `RAILS_APP_DB=2025-raleigh-disney`
- `RAILS_APP_SCOPE=2025/raleigh/disney`
- `DATABASE_URL=sqlite3:db/2025-raleigh-disney.sqlite3`
- `RAILS_STORAGE=storage/2025-raleigh-disney`

## Error Handling

### Connection Errors

Navigator handles process failures gracefully:

```go
if strings.Contains(err.Error(), "connection refused") {
    // Clear stale proxy cache
    h.mu.Lock()
    delete(h.proxies, tenant.Label)
    h.mu.Unlock()
    
    // Restart process and retry request
    process, restartErr := h.config.Manager.GetOrStart(tenant)
    if restartErr == nil {
        proxy := h.getOrCreateProxy(tenant.Label, process.Port)
        proxy.ServeHTTP(w, r)
        return
    }
}
```

### Authentication Errors

- `401 Unauthorized`: Invalid credentials
- `403 Forbidden`: Valid credentials, insufficient permissions
- `404 Not Found`: Tenant not found

## Testing

### Manual Testing

```bash
# Test basic functionality
curl http://localhost:3000/studios/

# Test tenant routing
curl http://localhost:3000/2025/raleigh/disney/

# Test authentication
curl --user username:password http://localhost:3000/2025/raleigh/disney/

# Test process recovery (kill Puma and retry)
ps aux | grep puma | grep -v grep  # Find PID
kill <PID>                         # Kill process  
curl http://localhost:3000/studios/ # Should auto-recover
```

### Load Testing

```bash
# Install Apache Bench
brew install httpd

# Basic load test
ab -n 1000 -c 10 http://localhost:3000/studios/

# Authentication load test
ab -n 100 -c 5 -A username:password http://localhost:3000/2025/raleigh/disney/
```

## Debugging

### Enable Debug Logging

Navigator logs all requests and process management operations:

```bash
./navigator -rails-root /path/to/app 2>&1 | tee navigator.log
```

### Common Issues

1. **Port Conflicts**: Navigator manages ports 4000+ automatically
2. **Authentication Failures**: Check htpasswd file format and permissions
3. **Tenant Not Found**: Verify showcases.yml uses Ruby symbols (`:name`)
4. **Process Crashes**: Check Rails logs in `log/<tenant>.log`

### Monitoring

Navigator logs include:
- Request routing: `"Routing /studios/ to index tenant"`
- Process management: `"Started Puma for index on port 4000"`
- Recovery events: `"Connection refused for index, attempting to restart"`
- Performance: Response times and status codes

## Production Deployment

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
ExecStart=/usr/local/bin/navigator -config /etc/navigator/navigator.yml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Performance Tuning

1. **Process Limits**: Set `max_processes` based on available memory
2. **Idle Timeout**: Balance resource usage vs startup latency
3. **Cache Headers**: Tune cache duration for static assets
4. **Connection Pooling**: Configure HTTP transport settings

## Contributing Guidelines

1. **Code Style**: Follow Go conventions, use `go fmt`
2. **Error Handling**: Always check errors, provide context
3. **Logging**: Use structured logging with appropriate levels
4. **Testing**: Add tests for new features
5. **Documentation**: Update README.md and this file

## Dependencies

- **Go 1.22+**: Modern Go features and performance
- **gopkg.in/yaml.v3**: YAML parsing with Ruby symbol support
- **github.com/tg123/go-htpasswd**: APR1 password hash support
- **github.com/spf13/cobra**: CLI framework with subcommands
- **github.com/spf13/viper**: Configuration management with YAML/ENV support
- **github.com/go-chi/chi/v5**: HTTP router and middleware
- **github.com/sirupsen/logrus**: Structured JSON logging
- **github.com/victorspringer/http-cache**: HTTP caching middleware

## Security Considerations

1. **Path Traversal**: All file paths are cleaned and validated
2. **Authentication**: Supports industry-standard hash formats
3. **Process Isolation**: Each tenant runs in separate Puma process
4. **Header Security**: Adds security headers to responses

## Release Process

### Automatic Release Builds

Navigator uses GitHub Actions to automatically build and release binaries when version tags are pushed:

```bash
# Create a new release
git tag -a v0.3.0 -m "Release v0.3.0: Your release notes here"
git push origin v0.3.0
```

The release workflow will:
1. **Run all tests** to ensure code quality
2. **Build binaries** for multiple platforms:
   - Linux: AMD64, ARM64 (tar.gz)
   - macOS: AMD64, ARM64 (tar.gz)  
   - Windows: AMD64, ARM64 (zip)
3. **Inject version information** into binaries
4. **Create compressed archives** for distribution
5. **Generate release notes** from tag annotations
6. **Create GitHub release** with all assets
7. **Mark pre-releases** for versions with hyphens (e.g., v1.0.0-beta)

### Version Information

Binaries include build metadata:

```bash
./navigator version
# Navigator v0.3.0
# Git Commit: abc1234
# Build Date: 2025-08-10T14:52:00Z  
# Go Version: go1.22.0
# Platform: linux/amd64
```

### Release Notes

Use annotated tags for detailed release notes:

```bash
git tag -a v0.3.0 -m "Navigator v0.3.0: Feature Release

🚀 New Features:
- Added metrics endpoint with Prometheus support
- Implemented rate limiting per tenant
- Added TLS termination with automatic certificates

🐛 Bug Fixes:
- Fixed memory leak in process manager
- Improved error handling in proxy recovery

📈 Performance:
- 25% faster request routing
- Reduced memory usage by 15%"
```

### Manual Release Steps

For maintainers creating releases:

1. **Update version** in relevant files if needed
2. **Run tests** locally: `go test ./...`
3. **Create annotated tag** with release notes
4. **Push tag** to trigger automatic build
5. **Monitor workflow** for any issues
6. **Verify release** on GitHub with all assets

### Release Workflow Files

- `.github/workflows/release.yml` - Automatic release builds
- `.github/workflows/ci.yml` - Continuous integration tests
- `cmd/navigator/main.go` - Version information injection
- `internal/cli/version.go` - Version display command

## Future Enhancements

1. **Metrics**: Prometheus/OpenTelemetry integration
2. **TLS**: HTTPS support with automatic certificate management
3. **Rate Limiting**: Per-tenant request rate limiting
4. **Caching**: Redis-based shared cache for multi-instance deployments