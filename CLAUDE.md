# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the Navigator project.

## Project Overview

Navigator is a Go-based web server for multi-tenant web applications. It provides framework independence, intelligent request routing, dynamic process management, authentication, static file serving, managed external processes, and support for deployment patterns like Microsoft Azure's Deployment Stamps and Fly.io's preferred instance routing.

## Production Status

**IMPORTANT**: The refactored Navigator (`cmd/navigator-refactored/`) is now in production and is the active development target.

- **Active Development**: All new features and bug fixes should target `cmd/navigator-refactored/`
- **Legacy Reference**: `cmd/navigator-legacy/` is preserved for reference only and should NOT be modified
- **Transition Period**: The legacy version remains until we have sufficient production experience with the refactored version
- **Do Not Touch**: Leave `cmd/navigator-legacy/` unchanged - it serves as a reference implementation only

## Current Implementation Status

✅ **Framework Independence**: Support for Rails, Django, Node.js, and other web frameworks
✅ **Modular Architecture**: Well-organized internal packages in `cmd/navigator-refactored/`
✅ **YAML Configuration Support**: Modern YAML-based configuration
✅ **Managed Processes**: External process management (Redis, Sidekiq, workers, etc.)
✅ **Dynamic Port Allocation**: Finds available ports instead of sequential assignment
✅ **PID File Management**: Automatic cleanup of stale PID files with /tmp/navigator.pid
✅ **Graceful Shutdown**: Proper SIGTERM/SIGINT handling
✅ **Static File Serving**: Direct filesystem serving with try_files behavior
✅ **Authentication**: htpasswd support with multiple hash formats
✅ **Configuration Reload**: Live reload via SIGHUP signal without restart
✅ **Machine Idle Management**: Fly.io machine auto-suspend/stop after idle timeout
✅ **Intelligent Fly-Replay**: Smart routing with automatic fallback for large requests
✅ **WebSocket Support**: Full WebSocket connection support with standalone servers
✅ **High Reliability**: Automatic retry with exponential backoff for proxy failures
✅ **Simple Configuration**: Flexible variable substitution system for multi-tenant apps
✅ **Structured Logging**: Source-identified output with configurable JSON format for all processes
✅ **Lifecycle Hooks**: Server and tenant hooks for custom integration and automation
✅ **Comprehensive Documentation**: Complete documentation site at https://rubys.github.io/navigator/
✅ **Sticky Sessions**: Machine-based session affinity for Fly.io with cross-region support

## Architecture

### Code Organization

Navigator has two implementations:

1. **`cmd/navigator-refactored/`** - **PRODUCTION VERSION** ✅
   - Modular, well-organized codebase
   - Uses shared `internal/` packages
   - **All development happens here**
   - Currently deployed in production

2. **`cmd/navigator-legacy/`** - **REFERENCE ONLY** 🔒
   - Original single-file implementation
   - **DO NOT MODIFY** - frozen for reference
   - Kept until refactored version proves stable in production
   - Will be removed once transition is complete

### Modular Package Structure

The refactored navigator uses focused internal packages for maintainability:
- **internal/server/**: HTTP handling, routing, static files, proxying
- **internal/process/**: Web app and managed process lifecycle
- **internal/config/**: Configuration loading and validation
- **internal/auth/**: Authentication (htpasswd)
- **internal/proxy/**: Reverse proxy and Fly-Replay logic
- **internal/idle/**: Fly.io machine idle management
- **internal/errors/**: Domain-specific error constructors
- **internal/logging/**: Structured logging helpers
- **internal/utils/**: Common utilities (duration parsing, environment, etc.)

**Design principles**:
- **Focused modules**: Each package has a single, clear responsibility
- **Easy deployment**: Single binary with minimal dependencies
- **Clear dependencies**: Only essential external Go packages
- **Maintainability**: Well-organized code with good separation of concerns
- **Testability**: Each module can be tested independently

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

3. **HTTP Handler** (`CreateHandler`, `handleStickySession`)
   - **Sticky Sessions**: Cookie-based machine affinity for Fly.io deployments
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

5. **Idle Manager** (`NewIdleManager`)
   - **Idle Detection**: Monitors request activity
   - **Auto-Suspend/Stop**: Suspends or stops Fly.io machines after idle timeout
   - **Auto-Wake**: Machines wake automatically on incoming requests

6. **Lifecycle Hooks** (`executeHooks`, `executeServerHooks`, `executeTenantHooks`)
   - **Server Hooks**: Execute at Navigator lifecycle events (start, ready, idle)
   - **Tenant Hooks**: Execute at tenant lifecycle events (start, stop)
   - **Environment Propagation**: Tenant hooks receive full tenant environment
   - **Sequential Execution**: Hooks run in order with configurable timeouts

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

### Configuration Structure

```yaml
server:
  listen: 3000
  hostname: localhost
  public_dir: public

  # Static file configuration (modern approach)
  cache_control:
    overrides:
      - path: /assets/
        max_age: 24h
  allowed_extensions: [html, css, js, png, jpg]  # optional, omit to allow all
  try_files: [index.html, .html, .htm]           # enables try_files feature

  idle:
    action: suspend    # "suspend" or "stop" for Fly.io machines
    timeout: 20m       # Duration format: "30s", "5m", "1h30m"

applications:
  pools:
    max_size: 10
    timeout: 5m        # App process idle timeout (duration format)
    start_port: 4000
  tenants:
    - name: tenant1
      root: /path/to/app
```

## Development Commands

### Building and Running

**IMPORTANT**: Always work with the refactored navigator (`cmd/navigator-refactored/`).

```bash
# Build refactored Navigator (production version)
go build -o bin/navigator cmd/navigator-refactored

# Run with configuration file
./bin/navigator config/navigator.yml

# Run with default config (looks for config/navigator.yml)
./bin/navigator

# DO NOT build legacy navigator - it is reference only
# go build cmd/navigator-legacy  ← DO NOT USE
```

### Development Workflow

```bash
# Install dependencies
go mod download
go mod tidy

# Format and check code
go fmt ./...
go vet ./...

# Build for different platforms (always use refactored version)
GOOS=linux GOARCH=amd64 go build -o navigator-linux cmd/navigator-refactored
GOOS=darwin GOARCH=arm64 go build -o navigator-darwin-arm64 cmd/navigator-refactored
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
- **Server-based configuration**: All static file settings in `server` section
- **Try files**: File resolution with multiple extensions (configured via `try_files` array)
- **Flexible extensions**: Serve specific file types or allow all files
- **Content-Type detection**: Automatic MIME type setting
- **Cache control**: Per-path cache header customization
- **Public routes**: Serves studios, regions, docs without authentication

### 4. Machine Idle Management (Fly.io)

- **Idle actions**: Supports both "suspend" and "stop" actions
- **Idle timeout**: Configurable inactivity period using duration format (e.g., "20m", "1h30m")
- **Request tracking**: Monitors active requests
- **Automatic wake**: Machines resume on incoming requests
- **Zero-downtime**: Seamless suspend/resume or stop/start cycles

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

### 6. Sticky Sessions (Fly.io)

Navigator provides machine-based session affinity for Fly.io deployments:

- **Cookie-Based Affinity**: Uses HTTP cookies to maintain session state
- **Cross-Region Support**: Works across all Fly.io regions globally
- **Automatic Failover**: Serves maintenance page when target machine unavailable
- **Large Request Handling**: Falls back to reverse proxy for requests >1MB
- **Path-Specific Sessions**: Optional configuration for specific URL paths
- **Configurable Duration**: Session lifetime using Go duration format (e.g., "1h", "30m")
- **Security Options**: Full cookie security with HTTPOnly, Secure, and SameSite flags

Configuration example:
```yaml
sticky_sessions:
  enabled: true
  cookie_name: "_navigator_machine"
  cookie_max_age: "2h"
  cookie_secure: true
  cookie_httponly: true
  paths:
    - "/app/*"
    - "/dashboard/*"
```

### 7. Lifecycle Hooks

Navigator supports hooks for custom integration at key lifecycle events:

**Server Hooks**:
- **start**: Executes before Navigator starts accepting requests
- **ready**: Executes after Navigator is ready and listening
- **resume**: Executes once on first request after machine suspension (Fly.io)
- **idle**: Executes before machine suspension (Fly.io)

**Tenant Hooks**:
- **start**: Executes after a tenant web app starts
- **stop**: Executes before a tenant web app stops
- **Environment**: Tenant hooks receive the same environment variables as the tenant app

Configuration example:
```yaml
hooks:
  server:
    start:
      - command: /usr/local/bin/prepare-server.sh
        timeout: 30
    ready:
      - command: curl
        args: ["-X", "POST", "http://monitoring.example.com/ready"]
    resume:
      - command: /usr/local/bin/notify-resume.sh
        args: ["Machine resumed from suspend"]
    idle:
      - command: /usr/local/bin/prepare-suspend.sh
  tenant:  # Default hooks for all tenants
    start:
      - command: /usr/local/bin/notify-tenant-start.sh

applications:
  tenants:
    - name: "2025/boston"
      hooks:  # Tenant-specific hooks
        start:
          - command: /usr/local/bin/boston-setup.sh
```

### 8. Configuration Template System

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

**CRITICAL IMPORTANCE**: Comprehensive test coverage is essential for maintaining Navigator's reliability in production.

### Test Organization

Navigator's test suite is organized by package:
- **internal/proxy/** - Proxy retry logic, WebSocket handling, response buffering
- **internal/server/** - Request routing, authentication, static files, fly-replay
- **internal/process/** - Web app lifecycle, managed processes, graceful shutdown
- **internal/config/** - Configuration parsing, validation, variable substitution
- **internal/auth/** - htpasswd authentication with multiple hash formats
- **internal/idle/** - Fly.io machine idle management and auto-suspend
- **internal/logging/** - Structured logging helpers
- **internal/utils/** - Duration parsing and utility functions

### Running Tests

```bash
# Run all tests with coverage
go test -cover ./...

# Run tests for specific package
go test -v ./internal/proxy/...

# Run tests with race detection
go test -race ./...

# Run short tests only (skip slow integration tests)
go test -short ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

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

## Code Quality and CI Requirements

**CRITICAL**: All code must pass CI checks. GitHub Actions will reject pushes that fail.

### CI Checks (Required)

1. ✓ **Formatting** - `gofmt -s -l .` returns no files
2. ✓ **Linting** - `golangci-lint run` passes (0 errors)
3. ✓ **Vet** - `go vet ./...` passes
4. ✓ **Tests** - `go test -race -cover ./...` passes
5. ✓ **Build** - Both navigators build successfully

### Quick Pre-Push Validation

```bash
# Run all checks locally before pushing
gofmt -s -l . && \
golangci-lint run && \
go vet ./... && \
go test -race -cover -timeout=3m ./... && \
go build ./cmd/navigator-legacy && \
go build ./cmd/navigator-refactored && \
echo "✓ All CI checks passed!"
```

### Test-Driven Development

**Every bug is a missing test.**
1. Write a failing test that reproduces the bug
2. Fix the bug with minimal changes
3. Verify the test passes and covers edge cases

**Every feature requires tests.**
1. Write tests before implementing the feature
2. Cover success, failure, and edge cases
3. Verify integration with existing components

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
- **Output Formats**: 
  - Text mode (default): Prefixed with `[source.stream]` format (e.g., `[2025/boston.stdout]`, `[redis.stderr]`)
  - JSON mode: Structured JSON with timestamp, source, stream, message, and optional tenant fields
- **Multiple Destinations**: Logs can be written to console and files simultaneously (Phase 3)
- **File Output**: Configurable file paths with `{{app}}` template variable support
- **Configuration**: Set via `logging:` section in YAML config:
  - `format`: text or json (defaults to text)
  - `file`: optional file path with `{{app}}` template support
  - `vector`: Vector integration configuration (enabled, socket, config)
- **Vector Integration**: Professional log aggregation with automatic Vector process management
- **Current Status**: Phases 1-4 complete (basic capture + JSON output + multiple destinations + Vector integration). Phase 5+ available for advanced features
- **Implementation Plan**: See `docs/logging-implementation-plan.md` for complete phased approach and current capabilities

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

## Utility Packages

Navigator provides reusable utility packages to reduce code duplication:

### internal/errors/

Domain-specific error constructors with proper error wrapping:
```go
import "github.com/rubys/navigator/internal/errors"

// Use standardized error constructors
return errors.ErrTenantNotFound(tenantName)
return errors.ErrConfigLoad(path, err)
return errors.ErrProxyConnection(target, err)
```

**Benefits**: Consistent error messages, proper error chaining with `%w`, better debugging context.

### internal/logging/

Structured logging helpers for common patterns:
```go
import "github.com/rubys/navigator/internal/logging"

// Instead of multi-line slog calls
logging.LogWebAppStart(tenant, port, runtime, server, args)
logging.LogProcessExit(name, err)
logging.LogConfigReload()
```

**Benefits**: Concise one-line calls, consistent structured logging format, easier maintenance.

### internal/utils/

Common utilities including enhanced duration parsing:
```go
import "github.com/rubys/navigator/internal/utils"

// Duration parsing with automatic logging of errors
timeout := utils.ParseDurationWithDefault(cfg.Timeout, 5*time.Minute)

// Duration parsing with contextual logging
delay := utils.ParseDurationWithContext(cfg.Delay, 0, map[string]interface{}{
    "process": procName,
})
```

**Benefits**: Eliminates duplicate duration parsing code, provides helpful error logging.

## Contributing Guidelines

**CRITICAL**: All development must target `cmd/navigator-refactored/` and its internal packages. Do NOT modify `cmd/navigator-legacy/`.

1. **Target refactored navigator**: All changes go to `cmd/navigator-refactored/` and `internal/` packages
2. **Leave legacy alone**: `cmd/navigator-legacy/` is frozen - do not modify it under any circumstances
3. **Modular design**: Place new functionality in appropriate internal packages
4. **Use utility packages**: Adopt error constructors and logging helpers for consistency
5. **Minimal dependencies**: Only add essential external packages
6. **Testing**: Write tests for new functionality, ensure all tests pass
7. **Documentation**: Update README.md, CLAUDE.md, and docs/ as needed
8. **Release process**: Use annotated tags for GitHub Actions releases

## Refactoring Guidelines

See `REFACTORING.md` for detailed refactoring status and guidelines. Key principles:

1. **Safety first**: All tests must pass before and after refactoring
2. **Incremental progress**: Small, focused refactorings with clear commits
3. **Clear separation**: Each module has a single, well-defined responsibility
4. **Maintainability**: Code should be easier to read, test, and modify
5. **No behavioral changes**: Refactoring should not change functionality

## Important Notes

- **YAML configuration**: YAML is the only supported configuration format
- **Modular architecture**: Well-organized internal packages for maintainability
- **Process management**: Navigator handles both web apps and external processes
- **Graceful shutdown**: All processes cleaned up properly on termination
- **Configuration reload**: Update configuration without restart using SIGHUP
- **Production ready**: Used in production with 75+ dance studios across 8 countries