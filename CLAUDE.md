# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the Navigator project.

## Project Overview

Navigator is a Go-based web server that replaces nginx/Passenger for multi-tenant Rails applications. It provides intelligent request routing, dynamic process management, authentication, static file serving, and managed external processes.

## Current Implementation Status

✅ **Single-File Go Implementation**: Simple, self-contained Go application in `cmd/navigator/main.go`
✅ **YAML Configuration Support**: Modern YAML-based configuration with automatic generation from Rails
✅ **Managed Processes**: External process management (Redis, Sidekiq, etc.)
✅ **Dynamic Port Allocation**: Finds available ports instead of sequential assignment
✅ **PID File Management**: Automatic cleanup of stale PID files
✅ **Graceful Shutdown**: Proper SIGTERM/SIGINT handling
✅ **Static File Serving**: Direct filesystem serving with try_files behavior
✅ **Authentication**: htpasswd support with multiple hash formats

## Architecture

### Single-File Implementation

The entire Navigator implementation is contained in `cmd/navigator/main.go`. This design provides:
- **Simplicity**: No complex internal package structure
- **Easy deployment**: Single binary with minimal dependencies
- **Clear dependencies**: Only essential external Go packages
- **Maintainability**: All logic in one place for this focused use case

### Key Components

1. **Configuration Loading** (`LoadConfig`, `ParseYAML`)
   - Auto-detects YAML vs nginx configuration formats
   - Supports template variable substitution for tenant configuration
   - Nginx format deprecated but still supported for backward compatibility

2. **Process Management** (`AppManager`, `ProcessManager`)
   - **Rails Apps**: On-demand startup with dynamic port allocation
   - **Managed Processes**: External process lifecycle management
   - **PID File Handling**: Automatic cleanup of stale processes
   - **Graceful Shutdown**: Clean termination of all processes

3. **HTTP Handler** (`CreateHandler`)
   - **Rewrite Rules**: nginx-style URL rewriting and redirects
   - **Authentication**: Pattern-based auth exclusions with htpasswd
   - **Static Files**: Direct filesystem serving with caching
   - **Try Files**: nginx-style file resolution for public content
   - **Rails Proxy**: Reverse proxy to Rails applications

4. **Static File Serving** (`serveStaticFile`, `tryFiles`)
   - **Performance**: Bypasses Rails for static content
   - **Try Files**: Attempts multiple file extensions before Rails fallback
   - **Content Types**: Automatic MIME type detection
   - **Caching**: Configurable cache headers

## Configuration

### YAML Configuration (Primary Method)

Navigator uses YAML configuration files that you create and maintain:

```bash
# Run with default config location
./bin/navigator  # Looks for config/navigator.yml

# Run with custom config file
./bin/navigator /path/to/config.yml
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

### 1. Managed Processes (New)

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

### 2. Process Management Improvements

- **PID file cleanup**: Removes stale PID files before starting Rails apps
- **Dynamic port allocation**: Finds available ports in range 4000-4099
- **Graceful shutdown**: SIGINT/SIGTERM handling with proper cleanup
- **Environment inheritance**: Rails apps inherit parent environment variables

### 3. Static File Optimization

- **Direct serving**: Static files served without Rails overhead
- **Try files**: nginx-style file resolution with multiple extensions
- **Content-Type detection**: Automatic MIME type setting
- **Public routes**: Serves studios, regions, docs without authentication

### 4. Configuration Template System

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

## Configuration Migration

### From nginx to YAML

To migrate from nginx configuration to YAML:

1. **Create YAML**: Convert your nginx configuration to YAML format
2. **Test configuration**: Start Navigator with new YAML configuration
3. **Update deployment**: Switch production to use YAML configuration
4. **Remove nginx**: Deprecated nginx support will be removed in future versions

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

## Future Development

### Completed Improvements ✅
- Single-file Go implementation
- Managed processes feature
- PID file handling
- Dynamic port allocation
- Graceful shutdown

### Planned Enhancements
- **Hot reload**: Configuration file watching
- **Metrics**: Prometheus/OpenTelemetry integration
- **SSL termination**: Optional HTTPS support
- **Load balancing**: Multiple Rails backends per tenant

## Contributing Guidelines

1. **Single file approach**: Keep all logic in `cmd/navigator/main.go`
2. **Minimal dependencies**: Only add essential external packages
3. **YAML configuration**: Create clear, maintainable YAML configuration examples
4. **Testing**: Verify both YAML and nginx configuration formats work
5. **Documentation**: Update both README.md and this file

## Important Notes

- **YAML configuration**: Create and maintain your own YAML configuration files
- **Single file design**: All logic in one Go file for simplicity
- **Nginx deprecated**: YAML is the preferred configuration format
- **Process management**: Navigator handles both Rails apps and external processes
- **Graceful shutdown**: All processes cleaned up properly on termination