# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the Navigator project.

## Project Overview

Navigator is a Go-based web server for multi-tenant Rails applications. It provides intelligent request routing, dynamic process management, authentication, static file serving, managed external processes, and support for deployment patterns like Microsoft Azure's Deployment Stamps and Fly.io's preferred instance routing.

## Current Implementation Status

✅ **Single-File Go Implementation**: Simple, self-contained Go application in `cmd/navigator/main.go`
✅ **YAML Configuration Support**: Modern YAML-based configuration
✅ **Managed Processes**: External process management (Redis, Sidekiq, workers, etc.)
✅ **Dynamic Port Allocation**: Finds available ports instead of sequential assignment
✅ **PID File Management**: Automatic cleanup of stale PID files with /tmp/navigator.pid
✅ **Graceful Shutdown**: Proper SIGTERM/SIGINT handling
✅ **Static File Serving**: Direct filesystem serving with try_files behavior
✅ **Authentication**: htpasswd support with multiple hash formats
✅ **Configuration Reload**: Live reload via SIGHUP signal without restart
✅ **Machine Suspension**: Fly.io machine auto-suspend after idle timeout
✅ **Fly-Replay Support**: Region-specific routing for distributed deployments
✅ **WebSocket Support**: Full WebSocket connection support with standalone servers

## Architecture

### Single-File Implementation

The entire Navigator implementation is contained in `cmd/navigator/main.go`. This design provides:
- **Simplicity**: No complex internal package structure
- **Easy deployment**: Single binary with minimal dependencies
- **Clear dependencies**: Only essential external Go packages
- **Maintainability**: All logic in one place for this focused use case

### Key Components

1. **Configuration Loading** (`LoadConfig`, `ParseYAML`, `UpdateConfig`)
   - YAML configuration format (nginx format removed)
   - Supports template variable substitution for tenant configuration
   - Live configuration reload via SIGHUP signal

2. **Process Management** (`AppManager`, `ProcessManager`)
   - **Rails Apps**: On-demand startup with dynamic port allocation
   - **Managed Processes**: External process lifecycle management
   - **PID File Handling**: Automatic cleanup of stale processes
   - **Graceful Shutdown**: Clean termination of all processes

3. **HTTP Handler** (`CreateHandler`)
   - **Rewrite Rules**: URL rewriting with redirect, last, and fly-replay flags
   - **Authentication**: Pattern-based auth exclusions with htpasswd
   - **Static Files**: Direct filesystem serving with caching
   - **Try Files**: File resolution for public content with multiple extensions
   - **Rails Proxy**: Reverse proxy to Rails applications with method exclusions
   - **Standalone Servers**: Proxy support for external services (Action Cable, etc.)
   - **Suspend Tracking**: Request tracking for idle machine suspension

4. **Static File Serving** (`serveStaticFile`, `tryFiles`)
   - **Performance**: Bypasses Rails for static content
   - **Try Files**: Attempts multiple file extensions before Rails fallback
   - **Content Types**: Automatic MIME type detection
   - **Caching**: Configurable cache headers

5. **Suspend Manager** (`NewSuspendManager`)
   - **Idle Detection**: Monitors request activity
   - **Auto-Suspend**: Suspends Fly.io machines after idle timeout
   - **Auto-Wake**: Machines wake automatically on incoming requests

## Configuration

### YAML Configuration

Navigator uses YAML configuration files:

```bash
# Display help
./bin/navigator --help

# Run with default config location
./bin/navigator  # Looks for config/navigator.yml

# Run with custom config file
./bin/navigator /path/to/config.yml

# Reload configuration without restart
./bin/navigator -s reload
# Or send SIGHUP signal directly
kill -HUP $(cat /tmp/navigator.pid)
```

### Configuration Flow

1. **YAML configuration**: Create and maintain YAML configuration files
2. **Navigator loads**: YAML configuration with tenant template variables
3. **Environment variables**: Standard variables applied to each tenant
4. **Process startup**: Rails apps and managed processes started as needed

## Development Commands

### Building and Running

```bash
# Build Navigator
make build

# Or build directly
go build -mod=readonly -o bin/navigator cmd/navigator/main.go

# Run with configuration file
./bin/navigator config/navigator.yml

# Run with default config (looks for config/navigator.yml)
./bin/navigator
```

### Development Workflow

```bash
# Install dependencies
go mod download
go mod tidy

# Format and check code
go fmt ./...
go vet ./...

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o navigator-linux cmd/navigator/main.go
GOOS=darwin GOARCH=arm64 go build -o navigator-darwin-arm64 cmd/navigator/main.go
```

## Key Features

### 1. Managed Processes

Navigator can start and manage additional processes:

```yaml
managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /path/to/app
    start_delay: 2
```

Features:
- **Auto-restart**: Processes restart on crash if configured
- **Start delays**: Ensures proper initialization order
- **Environment variables**: Custom env for each process
- **Graceful shutdown**: Stopped after Rails apps to preserve dependencies
- **Configuration updates**: Managed processes updated on configuration reload

### 2. Process Management Improvements

- **PID file cleanup**: Removes stale PID files before starting Rails apps
- **Dynamic port allocation**: Finds available ports in range 4000-4099
- **Graceful shutdown**: SIGINT/SIGTERM handling with proper cleanup
- **Environment inheritance**: Rails apps inherit parent environment variables

### 3. Static File Optimization

- **Direct serving**: Static files served without Rails overhead
- **Try files**: File resolution with multiple extensions
- **Content-Type detection**: Automatic MIME type setting
- **Public routes**: Serves studios, regions, docs without authentication

### 4. Machine Suspension (Fly.io)

- **Idle timeout**: Configurable inactivity period before suspension
- **Request tracking**: Monitors active requests
- **Automatic wake**: Machines resume on incoming requests
- **Zero-downtime**: Seamless suspend/resume cycles

### 5. Region Routing (Fly-Replay)

- **Pattern matching**: Route specific paths to designated regions
- **Status codes**: Configurable HTTP response codes
- **Method filtering**: Apply rules to specific HTTP methods
- **Deployment stamps**: Support for distributed deployment patterns

### 6. Configuration Template System

YAML supports template variables for tenant configuration:

```yaml
standard_vars:
  RAILS_APP_DB: "${tenant.database}"
  RAILS_APP_OWNER: "${tenant.owner}"
  RAILS_STORAGE: "${tenant.storage}"
  PIDFILE: "pids/${tenant.database}.pid"
```

Variables are substituted for each tenant during configuration loading.

## Error Handling

### Process Recovery

Navigator handles Rails process failures:

1. **Detection**: Connection refused errors detected
2. **Cleanup**: Stale PID files and processes cleaned up
3. **Restart**: Process restarted via `GetOrStartApp()`
4. **Retry**: Original request retried after restart

### Common Issues

1. **Port conflicts**: Dynamic port allocation prevents conflicts
2. **Stale PID files**: Automatic cleanup before starting
3. **Process crashes**: Managed processes auto-restart if configured
4. **Authentication**: Pattern-based exclusions for public assets

## Testing

### Manual Testing

```bash
# Test configuration loading
./bin/navigator /path/to/navigator.yml

# Test static file serving
curl -I http://localhost:3000/assets/application.js

# Test try_files behavior
curl -I http://localhost:3000/studios/raleigh  # → raleigh.html

# Test Rails proxy
curl http://localhost:3000/2025/boston/
```

### Configuration Testing

```bash
# Validate YAML configuration
./bin/navigator config/navigator.yml  # Should start without errors

# Check process management
ps aux | grep -E '(redis|sidekiq|rails)'  # See managed processes
```

## Release Process

### Automatic Releases

GitHub Actions automatically builds releases when version tags are pushed:

```bash
# Create annotated tag with release notes
git tag -a v1.0.0 -m "Navigator v1.0.0: Major Release

## New Features
- Managed process support
- Dynamic port allocation
- Improved PID file handling

## Bug Fixes
- Fixed graceful shutdown
- Resolved port conflicts"

git push origin v1.0.0
```

### Release Assets

The workflow creates binaries for:
- Linux: AMD64, ARM64 (tar.gz)
- macOS: AMD64, ARM64 (tar.gz)
- Windows: AMD64, ARM64 (zip)

All binaries include version information and build metadata.

## Dependencies

Navigator uses minimal, focused dependencies:

- **Go 1.24+**: Modern Go features
- **github.com/tg123/go-htpasswd**: htpasswd file support (APR1, bcrypt, etc.)
- **gopkg.in/yaml.v3**: YAML configuration parsing

**No complex web frameworks** - uses Go standard library for HTTP handling.

## Logging

Navigator uses Go's `slog` package for structured logging:
- **Log Level**: Set via `LOG_LEVEL` environment variable (debug, info, warn, error)
- **Default Level**: Info level if not specified
- **Debug Output**: Includes detailed request routing, auth checks, and file serving attempts
- **Structured Format**: Text handler with consistent key-value pairs

## Deployment Considerations

### Production Deployment

1. **Single binary**: No external dependencies beyond htpasswd files
2. **YAML configuration**: Create and maintain YAML configuration files
3. **Process monitoring**: Navigator manages Rails and external processes
4. **Resource efficiency**: Lower memory footprint than nginx/Passenger

### Systemd Integration

```ini
[Unit]
Description=Navigator Rails Proxy
After=network.target

[Service]
Type=simple
User=rails
WorkingDirectory=/opt/rails/app
ExecStart=/usr/local/bin/navigator config/navigator.yml
Restart=always

[Install]
WantedBy=multi-user.target
```

## Vision and Roadmap

Navigator aims to simplify deployment of multi-tenant Rails applications by providing a single binary that handles:
- Application lifecycle management
- Request routing and authentication
- Static file serving
- Process management
- Regional distribution

### Use Cases
- **Multi-tenant SaaS**: Each customer gets their own database and instance
- **Regional deployments**: Deploy closer to users using Fly.io regions
- **Deployment stamps**: Microsoft Azure pattern for distributed applications
- **Development environments**: Replace complex nginx/Passenger setups

### Future Enhancements
- **Dynamic DNS checking**: Smart replay decisions based on machine availability
- **Dynamic machine startup**: Start new machines based on demand
- **Per-user machines**: One machine per user with auto-suspend
- **Metrics**: Prometheus/OpenTelemetry integration
- **SSL termination**: Optional HTTPS support for development
- **Docker Hub releases**: Easy inclusion via COPY --from=rubys/navigator

## Contributing Guidelines

1. **Single file approach**: Keep all logic in `cmd/navigator/main.go`
2. **Minimal dependencies**: Only add essential external packages
3. **YAML configuration**: Create clear, maintainable YAML configuration examples
4. **Testing**: Verify YAML configuration and all features work
5. **Documentation**: Update README.md, CLAUDE.md, and Roadmap.md as needed
6. **Release process**: Use annotated tags for GitHub Actions releases

## Important Notes

- **YAML configuration**: YAML is the only supported configuration format
- **Single file design**: All logic in one Go file for simplicity
- **Process management**: Navigator handles both Rails apps and external processes
- **Graceful shutdown**: All processes cleaned up properly on termination
- **Configuration reload**: Update configuration without restart using SIGHUP
- **Production ready**: Used in production with 75+ dance studios across 8 countries