# Navigator - Go Web Server

Navigator is a Go-based replacement for nginx + Phusion Passenger that provides multi-tenant Rails application hosting with on-demand process management.

## Overview

Navigator supports both nginx-style and modern YAML configuration formats:
- **Multi-tenant hosting**: Manages multiple Rails applications with separate databases
- **On-demand process management**: Starts Rails apps when needed, stops after idle timeout
- **Managed processes**: Start and stop additional processes alongside Navigator (Redis, workers, etc.)
- **Static file serving**: Serves assets, images, and static content directly from filesystem with configurable caching
- **Authentication**: Full htpasswd support (APR1, bcrypt, SHA, etc.) with pattern-based exclusions
- **URL rewriting**: Nginx-style rewrite rules with redirect and last flags
- **Reverse proxy**: Forwards dynamic requests to Rails applications
- **Dual configuration**: Supports both nginx config files (deprecated) and modern YAML format

## Installation

Download the latest release from [GitHub Releases](https://github.com/rubys/navigator/releases) or build from source:

```bash
# Clone the repository
git clone https://github.com/rubys/navigator.git
cd navigator

# Build the navigator
make build
# Or build directly with Go
go build -mod=readonly -o bin/navigator cmd/navigator/main.go
```

## Quick Start

```bash
# Run with YAML configuration (default looks for config/navigator.yml)
./bin/navigator
# Or specify a custom config file
./bin/navigator /path/to/navigator.yml
```

The navigator will:
- Auto-detect configuration format (YAML vs nginx)
- Start listening on the configured port (default: 9999 for local, 3000 for production)
- Dynamically allocate ports for Rails applications (4000-4099)
- Clean up stale PID files before starting apps
- Handle graceful shutdown on interrupt signals

## Configuration

### YAML Configuration (Recommended)

Create a YAML configuration file with your application settings:

```yaml
server:
  listen: 3000
  hostname: localhost
  root_path: /showcase
  public_dir: /path/to/public

pools:
  max_size: 22
  idle_timeout: 300
  start_port: 4000

auth:
  enabled: true
  realm: Showcase
  htpasswd: /path/to/htpasswd
  public_paths:
    - /showcase/assets/
    - /showcase/docs/
    - "*.css"
    - "*.js"

static:
  directories:
    - path: /showcase/assets/
      root: assets/
      cache: 86400
  extensions: [html, htm, css, js, png, jpg, gif]
  try_files:
    enabled: true
    suffixes: ["index.html", ".html", ".htm", ".txt", ".xml", ".json"]
    fallback: rails

applications:
  global_env:
    RAILS_RELATIVE_URL_ROOT: /showcase
  
  # Standard environment variables applied to all tenants (except special ones)
  standard_vars:
    RAILS_APP_DB: "${tenant.database}"
    RAILS_APP_OWNER: "${tenant.owner}"  # Studio name only
    RAILS_STORAGE: "/path/to/storage"   # Root storage path (not tenant-specific)
    RAILS_APP_SCOPE: "${tenant.scope}"
    PIDFILE: "/path/to/pids/${tenant.database}.pid"
  
  tenants:
    - name: 2025-boston
      path: /showcase/2025/boston/
      group: showcase-2025-boston
      database: 2025-boston
      owner: "Boston Dance Studio"
      storage: "/path/to/storage/2025-boston"
      scope: "2025/boston"
      env:
        SHOWCASE_LOGO: "boston-logo.png"
    
    # Special tenants that don't use standard_vars
    - name: cable
      path: /cable
      group: showcase-cable
      special: true
      force_max_concurrent_requests: 0

managed_processes:
  - name: redis
    command: redis-server
    args: []
    working_dir: /path/to/app
    env:
      REDIS_PORT: "6379"
    auto_restart: true
    start_delay: 0
    
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /path/to/app
    env:
      RAILS_ENV: production
    auto_restart: true
    start_delay: 2
```

## Key Features

### Managed Processes

Navigator can manage additional processes that should run alongside the web server. These processes are:
- **Started automatically** when Navigator starts
- **Stopped gracefully** when Navigator shuts down (after Rails apps to maintain dependencies)
- **Monitored and restarted** if they crash (when auto_restart is enabled)
- **Started with delays** to ensure proper initialization order

Common use cases:
- **Redis server**: Cache and session storage
- **Sidekiq/Resque**: Background job processors
- **WebSocket servers**: Additional real-time communication servers
- **Monitoring scripts**: Health check and metrics collection

### Performance Optimizations
- **Static file serving**: Bypasses Rails for assets and static content
- **Try files optimization**: Serves public content (studios, regions, docs) without Rails
- **Process pooling**: Reuses Rails processes across requests
- **Concurrent handling**: Multiple requests processed simultaneously
- **Zero Rails overhead**: Public routes serve static files instantly

### Process Management
- **On-demand startup**: Rails apps start when first requested
- **Idle timeout**: Apps automatically shut down after 5 minutes of inactivity (configurable)
- **Dynamic port allocation**: Finds available ports in range 4000-4099 instead of sequential assignment
- **PID file management**: Cleans up stale PID files before starting and after stopping apps
- **Graceful shutdown**: Handles SIGINT/SIGTERM signals to cleanly stop all Rails apps

### Authentication
- **Multiple formats**: Full htpasswd support via go-htpasswd library (APR1, bcrypt, SHA, MD5-crypt, etc.)
- **Pattern-based exclusions**: Simple glob patterns and regex patterns for public paths
- **Basic Auth**: Standard HTTP Basic Authentication

## Testing

```bash
# Test static asset serving
curl -I http://localhost:9999/showcase/assets/application.js

# Test try_files behavior (non-authenticated routes)
curl -I http://localhost:9999/showcase/studios/raleigh        # → raleigh.html
curl -I http://localhost:9999/showcase/regions/dfw           # → dfw.html

# Test authentication
curl -u username:password http://localhost:9999/protected/path

# Test Rails proxy (authenticated routes)
curl -u test:secret http://localhost:9999/showcase/2025/boston/
```

## Development

### File Structure
- `cmd/navigator/main.go` - Main application entry point
- `Makefile` - Build configuration
- `go.mod`, `go.sum` - Go module dependencies

### Building
```bash
# Standard build
make build

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o navigator cmd/navigator/main.go

# Install dependencies
go mod download
```

## Deployment

Navigator is designed to replace nginx + Passenger in production environments:
- **Single binary**: No external dependencies
- **Configuration compatibility**: Uses existing nginx config files (deprecated) or modern YAML
- **Resource efficiency**: Lower memory footprint than full nginx/Passenger stack
- **Monitoring**: Built-in logging for requests, static files, and process management

## Release Process

Navigator uses GitHub Actions for automated releases. To create a new release:

```bash
# Create a new release with release notes
git tag -a v0.3.0 -m "Release v0.3.0

## New Features
- Added managed processes support
- Improved PID file handling
- Dynamic port allocation

## Bug Fixes  
- Fixed graceful shutdown
- Resolved port conflicts"

git push origin v0.3.0
```

The release workflow will automatically:
1. Run tests to ensure code quality
2. Build binaries for multiple platforms (Linux, macOS, Windows on AMD64/ARM64)
3. Create compressed archives for distribution
4. Generate GitHub release with release notes from tag annotation
5. Upload all release assets

## License

MIT License - see [LICENSE](LICENSE) file for details.

For more detailed information, see [NAVIGATOR.md](NAVIGATOR.md).