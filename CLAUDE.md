# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the Navigator project.

## Project Overview

Navigator is a Go-based web server for multi-tenant web applications. It provides framework independence, intelligent request routing, dynamic process management, authentication, static file serving, managed external processes, and support for deployment patterns like Microsoft Azure's Deployment Stamps and Fly.io's preferred instance routing.

## Current Implementation Status

✅ **Framework Independence**: Support for Rails, Django, Node.js, and other web frameworks
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
✅ **Intelligent Fly-Replay**: Smart routing with automatic fallback for large requests
✅ **WebSocket Support**: Full WebSocket connection support with standalone servers
✅ **High Reliability**: Automatic retry with exponential backoff for proxy failures
✅ **Simple Configuration**: Flexible variable substitution system for multi-tenant apps
✅ **Structured Logging**: Source-identified output from all managed processes and web apps
✅ **Comprehensive Documentation**: Complete documentation site at https://rubys.github.io/navigator/

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
   - **Web Apps**: On-demand startup with dynamic port allocation
   - **Framework Configuration**: Configurable runtime and server executables
   - **Managed Processes**: External process lifecycle management
   - **PID File Handling**: Automatic cleanup of stale processes
   - **Graceful Shutdown**: Clean termination of all processes

3. **HTTP Handler** (`CreateHandler`)
   - **Rewrite Rules**: URL rewriting with redirect, last, and fly-replay flags
   - **Authentication**: Pattern-based auth exclusions with htpasswd
   - **Static Files**: Direct filesystem serving with caching
   - **Try Files**: File resolution for public content with multiple extensions
   - **Web App Proxy**: Reverse proxy to web applications with method exclusions
   - **Standalone Servers**: Proxy support for external services (Action Cable, etc.)
   - **Suspend Tracking**: Request tracking for idle machine suspension
   - **Proxy Retry**: Automatic retry logic with exponential backoff

4. **Static File Serving** (`serveStaticFile`, `tryFiles`)
   - **Performance**: Bypasses web framework for static content
   - **Try Files**: Attempts multiple file extensions before web app fallback
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
2. **Navigator loads**: YAML configuration with framework and tenant settings
3. **Framework configuration**: Runtime executable and server settings applied
4. **Environment variables**: Flexible variable substitution for each tenant
5. **Process startup**: Web apps and managed processes started as needed

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
- **Graceful shutdown**: Stopped after web apps to preserve dependencies
- **Configuration updates**: Managed processes updated on configuration reload

### 2. Process Management Improvements

- **PID file cleanup**: Removes stale PID files before starting web apps
- **Dynamic port allocation**: Finds available ports in range 4000-4099
- **Graceful shutdown**: SIGINT/SIGTERM handling with proper cleanup
- **Environment inheritance**: Web apps inherit parent environment variables

### 3. Static File Optimization

- **Direct serving**: Static files served without web framework overhead
- **Try files**: File resolution with multiple extensions
- **Content-Type detection**: Automatic MIME type setting
- **Public routes**: Serves studios, regions, docs without authentication

### 4. Machine Suspension (Fly.io)

- **Idle timeout**: Configurable inactivity period before suspension
- **Request tracking**: Monitors active requests
- **Automatic wake**: Machines resume on incoming requests
- **Zero-downtime**: Seamless suspend/resume cycles

### 5. Intelligent Region Routing (Fly-Replay)

- **Multi-Target Routing**: Support for three routing types:
  - **App-based**: Route to any instance of a specific app
  - **Machine-based**: Route to a specific machine instance
  - **Region-based**: Route to a specific Fly.io region
- **Smart Detection**: Automatically uses reverse proxy for requests >1MB
- **Automatic Fallback**: Uses reverse proxy for requests >1MB
- **Maintenance Pages**: Serves custom 503 page when targets unavailable
- **Pattern matching**: Route specific paths to designated regions
- **Status codes**: Configurable HTTP response codes
- **Method filtering**: Apply rules to specific HTTP methods
- **Deployment stamps**: Support for distributed deployment patterns
- **Automatic Fallback**: Constructs internal URLs for direct proxy when needed

### 6. Configuration Template System

YAML supports flexible variable substitution for tenant configuration:

```yaml
applications:
  env:
    RAILS_APP_DB: "${database}"
    RAILS_APP_OWNER: "${owner}"
    RAILS_STORAGE: "${storage}"
    PIDFILE: "pids/${database}.pid"
  
  tenants:
    - name: 2025-boston
      var:
        database: "2025-boston"
        owner: "Boston Dance Studio"
        storage: "/path/to/storage/2025-boston"
```

Variables defined in the `var` map are substituted using `${variable}` syntax in environment templates.

## Error Handling

### Process Recovery

Navigator handles web app process failures:

1. **Detection**: Connection refused errors detected
2. **Cleanup**: Stale PID files and processes cleaned up
3. **Restart**: Process restarted via `GetOrStartApp()`
4. **Retry**: Original request retried after restart

### Proxy Reliability

Navigator includes robust proxy error handling:

1. **Automatic Retry**: Failed proxy connections retry with exponential backoff
2. **Smart Timeouts**: Up to 3 seconds of retries for connection failures
3. **Request Preservation**: GET/HEAD requests safely retried
4. **Graceful Degradation**: Falls back to error response after max retries

### Common Issues

1. **Port conflicts**: Dynamic port allocation prevents conflicts
2. **Stale PID files**: Automatic cleanup before starting
3. **Process crashes**: Managed processes auto-restart if configured
4. **Authentication**: Pattern-based exclusions for public assets
5. **Machine unavailable**: Serves maintenance page during deployments
6. **Large uploads**: Automatically falls back to reverse proxy for >1MB requests

## Testing

### Manual Testing

```bash
# Test configuration loading
./bin/navigator /path/to/navigator.yml

# Test static file serving
curl -I http://localhost:3000/assets/application.js

# Test try_files behavior
curl -I http://localhost:3000/studios/raleigh  # → raleigh.html

# Test web app proxy
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

## Documentation

Navigator has comprehensive documentation hosted at **https://rubys.github.io/navigator/** including:

### Core Documentation
- **Getting Started**: Installation and basic configuration
- **Configuration Reference**: Complete YAML configuration options
- **Examples**: Working configurations for common scenarios (Redis, Action Cable, multi-tenant)
- **Features**: Detailed explanations of all Navigator capabilities
- **Deployment**: Production deployment guides and best practices
- **Reference**: CLI options, environment variables, signal handling

### Key Documentation Features
- **Live Examples**: All configuration examples are copy-paste ready
- **Step-by-Step Guides**: Clear instructions for setup and testing
- **Comprehensive Coverage**: 50+ pages covering all Navigator features
- **Search Functionality**: Full-text search across all documentation
- **Mobile Responsive**: Works perfectly on all devices
- **Automatic Updates**: Documentation deploys automatically via GitHub Actions

### Quick Reference Links
- **Home**: https://rubys.github.io/navigator/
- **YAML Reference**: https://rubys.github.io/navigator/configuration/yaml-reference/
- **Examples**: https://rubys.github.io/navigator/examples/
- **CLI Reference**: https://rubys.github.io/navigator/reference/cli/

The documentation source is in the `docs/` directory and uses MkDocs with Material theme for generation.

## Logging

Navigator uses Go's `slog` package for structured logging:
- **Log Level**: Set via `LOG_LEVEL` environment variable (debug, info, warn, error)
- **Default Level**: Info level if not specified
- **Debug Output**: Includes detailed request routing, auth checks, and file serving attempts
- **Structured Format**: Text handler with consistent key-value pairs
- **Process Output Capture**: All stdout/stderr from managed processes and web apps is captured with source identification
- **Log Prefixing**: Output from child processes is prefixed with `[source.stream]` format (e.g., `[2025/boston.stdout]`, `[redis.stderr]`)
- **Incremental Implementation**: See `docs/logging-implementation-plan.md` for the phased approach to enhanced logging capabilities

## Deployment Considerations

### Production Deployment

1. **Single binary**: No external dependencies beyond htpasswd files
2. **YAML configuration**: Create and maintain YAML configuration files
3. **Process monitoring**: Navigator manages web apps and external processes
4. **Resource efficiency**: Lower memory footprint than nginx/Passenger

### Systemd Integration

```ini
[Unit]
Description=Navigator Web Application Proxy
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

Navigator aims to simplify deployment of multi-tenant web applications by providing a single binary that handles:
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
- **Simplified configuration**: More flexible variable substitution system
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
- **Process management**: Navigator handles both web apps and external processes
- **Graceful shutdown**: All processes cleaned up properly on termination
- **Configuration reload**: Update configuration without restart using SIGHUP
- **Production ready**: Used in production with 75+ dance studios across 8 countries