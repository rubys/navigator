# Navigator - Go Web Server

Navigator is a Go-based web server that provides multi-tenant web application hosting with on-demand process management. It supports multiple web frameworks through configurable settings.

## Overview

Navigator uses YAML configuration format for:
- **Multi-tenant hosting**: Manages multiple web applications with separate databases
- **Framework independence**: Supports Rails, Django, Node.js, and other frameworks via configuration
- **On-demand process management**: Starts web apps when needed, stops after idle timeout
- **Managed processes**: Start and stop additional processes alongside Navigator (Redis, workers, etc.)
- **Static file serving**: Serves assets, images, and static content directly from filesystem with configurable caching
- **Authentication**: Full htpasswd support (APR1, bcrypt, SHA, etc.) with pattern-based exclusions
- **URL rewriting**: Rewrite rules with redirect, last, and fly-replay flags for region-specific routing
- **Reverse proxy**: Forwards dynamic requests to web applications with method-based exclusions and custom headers
- **Machine idle management**: Auto-suspend or stop Fly.io machines after idle timeout with automatic wake on requests
- **Configuration reload**: Live configuration reload with SIGHUP signal (no restart needed)
- **WebSocket support**: Full support for WebSocket connections and standalone servers
- **Intelligent routing**: Smart Fly-Replay with automatic fallback to reverse proxy for large requests
- **High reliability**: Automatic retry with exponential backoff for proxy failures
- **Lifecycle hooks**: Server and tenant hooks for custom integration at key lifecycle events

## Installation

Modify your Dockerfile:

```Dockerfile
COPY --from=samruby/navigator:latest /navigator /usr/local/bin/navigator
CMD ["navigator", "/app/navigator.yml"]
```

Download the latest release from [GitHub Releases](https://github.com/rubys/navigator/releases) 

Build from source:

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
# Display help
./bin/navigator --help

# Run with YAML configuration (default looks for config/navigator.yml)
./bin/navigator
# Or specify a custom config file
./bin/navigator /path/to/navigator.yml

# Reload configuration without restart
./bin/navigator -s reload
# Or send SIGHUP signal directly
kill -HUP $(cat /tmp/navigator.pid)
```

The navigator will:
- Start listening on the configured port (default: 9999 for local, 3000 for production)
- Dynamically allocate ports for web applications (4000-4099)
- Clean up stale PID files before starting apps
- Handle graceful shutdown on interrupt signals

## Configuration

### YAML Configuration

Create a YAML configuration file with your application settings:

```yaml
server:
  listen: 3000
  hostname: localhost
  root_path: /showcase
  public_dir: /path/to/public
  idle:  # Machine idle management (Fly.io)
    action: suspend    # "suspend" or "stop"
    timeout: 20m       # Duration format: "30s", "5m", "1h30m"

auth:
  enabled: true
  realm: Showcase
  htpasswd: /path/to/htpasswd
  public_paths:
    - /showcase/assets/
    - /showcase/docs/
    - "*.css"
    - "*.js"

applications:
  pools:
    max_size: 22
    timeout: 5m        # App process idle timeout (duration format)
    start_port: 4000
  framework:
    runtime_executable: ruby
    server_executable: bin/rails
    server_command: server
    server_args: ["-p", "${port}"]
    app_directory: /rails
    port_env_var: PORT
    startup_delay: 5
  
  # Environment variables with template substitution  
  env:
    RAILS_RELATIVE_URL_ROOT: /showcase
    RAILS_APP_DB: "${database}"
    RAILS_APP_OWNER: "${owner}"  # Studio name only
    RAILS_STORAGE: "${storage}"
    RAILS_APP_SCOPE: "${scope}"
    PIDFILE: "/path/to/pids/${database}.pid"
  
  tenants:
    - path: /showcase/2025/boston/
      var:
        database: "2025-boston"
        owner: "Boston Dance Studio"
        storage: "/path/to/storage/2025-boston"
        scope: "2025/boston"
      env:
        SHOWCASE_LOGO: "boston-logo.png"
      # Tenant-specific hooks (optional)
      hooks:
        start:
          - command: /usr/local/bin/cache-warm.sh
            args: ["2025-boston"]
            timeout: 10
    
    # Special tenants that don't use variable substitution
    - path: /cable
      special: true
      force_max_concurrent_requests: 0
    
    # Tenants with pattern matching for WebSocket support
    - path: /cable-specific
      match_pattern: "*/cable"  # Matches any path ending with /cable
      special: true
    
    # Tenants with standalone servers (e.g., Action Cable)
    - path: /external/
      standalone_server: "localhost:28080"  # Proxy to standalone server instead of app

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

# Lifecycle hooks
hooks:
  server:
    start:  # Before accepting requests
      - command: /usr/local/bin/prepare-server.sh
        timeout: 30
    ready:  # After server is listening
      - command: curl
        args: ["-X", "POST", "http://monitoring.example.com/ready"]
    idle:   # Before machine suspension (Fly.io)
      - command: /usr/local/bin/cleanup.sh
  tenant:   # Default hooks for all tenants
    start:
      - command: echo
        args: ["Tenant started"]
    stop:
      - command: echo
        args: ["Tenant stopping"]

# Routing enhancements
routes:
  # Fly-replay support for multi-target routing
  fly_replay:
    # App-based routing (route to any instance of smooth-pdf app)
    - path: "^/showcase/.+\\.pdf$"
      app: smooth-pdf
      status: 307
    
    # Machine-based routing (route to specific machine instance)
    - path: "^/showcase/priority/.+\\.pdf$"
      machine: "48e403dc711e18"
      app: smooth-pdf
      status: 307
    
    # Region-based routing (route to specific region)
    - path: "^/showcase/2025/sydney/"
      region: syd
      status: 307
      methods: [GET, POST]  # Automatically uses reverse proxy for large POST requests
  
  # Reverse proxy with method exclusions
  reverse_proxies:
    - path: "/api/"
      target: "http://api.example.com"
      headers:
        X-API-Key: "secret"
      exclude_methods: [POST, DELETE]  # Don't proxy these methods

# Maintenance page configuration (optional)
maintenance:
  page: "/503.html"  # Path to custom maintenance page served during retry failures
```

## Key Features

### Machine Idle Management
- **Fly.io Integration**: Auto-suspend or stop machines after configurable idle timeout
- **Flexible Actions**: Choose between "suspend" (faster wake) or "stop" (releases resources)
- **Request Tracking**: Monitors active requests to determine idle state
- **Automatic Wake**: Machines wake automatically on incoming requests
- **Duration Format**: Set timeout using flexible duration strings (e.g., "20m", "1h30m")

### Configuration Reload
- **Live Reload**: Reload configuration without restart using SIGHUP signal
- **Reload Command**: Support for `navigator -s reload` command
- **PID File Management**: Writes PID file to /tmp/navigator.pid for signal management
- **Atomic Updates**: Configuration changes applied atomically with no downtime

### Intelligent Fly-Replay Support
- **Multi-Target Routing**: Support for three routing types:
  - **App-based**: Route to any instance of a specific app
  - **Machine-based**: Route to a specific machine instance using `prefer_instance`
  - **Region-based**: Route to a specific Fly.io region
- **Smart Fallback**: Automatically uses reverse proxy for requests >1MB that Fly.io replay can't handle
- **Pattern Matching**: Configure URL patterns for targeted routing
- **Status Codes**: Configurable HTTP status codes for replay responses
- **Method Filtering**: Apply replay rules only to specific HTTP methods
- **Internal Networking**: Support for `.internal`, `.vm.app.internal`, and regional `.internal` URLs

### Reverse Proxy Enhancements
- **Automatic Retry**: Connection failures are retried with exponential backoff (up to 3 seconds)
- **Method Exclusions**: Exclude specific HTTP methods from proxy routing
- **Custom Headers**: Add headers to proxied requests
- **Multiple Targets**: Support for multiple proxy configurations
- **High Reliability**: Graceful handling of backend failures with automatic recovery

### Standalone Server Support
- **External Services**: Proxy to standalone servers (e.g., Action Cable)
- **Pattern Matching**: Use wildcard patterns for location matching
- **WebSocket Support**: Full support for WebSocket connections

### Managed Processes

Navigator can manage additional processes that should run alongside the web server. These processes are:
- **Started automatically** when Navigator starts
- **Stopped gracefully** when Navigator shuts down (after Rails apps to maintain dependencies)
- **Monitored and restarted** if they crash (when auto_restart is enabled)
- **Started with delays** to ensure proper initialization order
- **Environment variables**: Custom environment variables for each process
- **Auto-restart capability**: Processes automatically restart if they crash

Common use cases:
- **Redis server**: Cache and session storage
- **Sidekiq/Resque**: Background job processors
- **WebSocket servers**: Additional real-time communication servers
- **Monitoring scripts**: Health check and metrics collection
- **File watchers**: Asset compilation or file synchronization

### Performance Optimizations
- **Static file serving**: Bypasses Rails for assets and static content
- **Try files optimization**: Serves public content (studios, regions, docs) without Rails
- **Process pooling**: Reuses Rails processes across requests
- **Concurrent handling**: Multiple requests processed simultaneously
- **Zero Rails overhead**: Public routes serve static files instantly

### Process Management
- **On-demand startup**: Rails apps start when first requested
- **Idle timeout**: Apps automatically shut down after 5 minutes of inactivity (configurable)
- **Dynamic port allocation**: Automatically finds available ports in range 4000-4099
- **PID file management**: Automatic cleanup of stale PID files before starting and after stopping apps
- **Graceful shutdown**: Handles SIGINT/SIGTERM signals to cleanly stop all Rails apps and managed processes
- **Environment inheritance**: Rails apps inherit parent process environment variables
- **Process cleanup**: Automatically removes PID files and kills stale processes

### Authentication
- **Multiple formats**: Full htpasswd support via go-htpasswd library (APR1, bcrypt, SHA, MD5-crypt, etc.)
- **Pattern-based exclusions**: Simple glob patterns and regex patterns for public paths
- **Basic Auth**: Standard HTTP Basic Authentication
- **Public paths**: Configure paths that bypass authentication entirely

### Lifecycle Hooks
- **Server hooks**: Execute commands at Navigator lifecycle events (start, ready, idle)
- **Tenant hooks**: Execute commands when tenants start or stop
- **Environment propagation**: Tenant hooks receive the same environment variables as the tenant app
- **Default and specific**: Default hooks apply to all tenants, with per-tenant overrides
- **Use cases**: Database migrations, cache warming, monitoring, cleanup tasks

## Testing

```bash
# Test static asset serving
curl -I http://localhost:9999/showcase/assets/application.js

# Test try_files behavior (non-authenticated routes)
curl -I http://localhost:9999/showcase/studios/raleigh        # → raleigh.html
curl -I http://localhost:9999/showcase/regions/dfw           # → dfw.html

# Test authentication
curl -u username:password http://localhost:9999/protected/path

# Test web app proxy (authenticated routes)
curl -u username:password http://localhost:9999/showcase/2025/boston/
```

## Documentation

📚 **Complete documentation is available at: https://rubys.github.io/navigator/**

The documentation includes:
- **Getting Started**: Installation and basic configuration
- **Configuration Reference**: Complete YAML options with examples
- **Working Examples**: Redis, Action Cable, multi-tenant setups
- **Feature Guides**: Process management, routing, authentication
- **Deployment**: Production deployment and best practices
- **CLI Reference**: Command-line options and usage

### Local Documentation Development

```bash
# Set up development environment
mise trust
mise settings experimental=true
uv venv
uv pip install -r pyproject.toml

# Serve documentation locally
mkdocs serve
# Open http://localhost:8000 in your browser
```

### Alternative Setup (without mise)

```bash
# Install dependencies directly
pip install -r requirements.txt

# Serve documentation
mkdocs serve
```

## Development

### File Structure
- `cmd/navigator/main.go` - Main application entry point
- `Makefile` - Build configuration
- `go.mod`, `go.sum` - Go module dependencies
- `docs/` - Documentation source files
- `mkdocs.yml` - Documentation configuration

### Logging
Navigator provides comprehensive logging for both its own operations and all managed processes:

**Navigator Logs** (via Go's `slog`):
- **Log Level**: Set via `LOG_LEVEL` environment variable (debug, info, warn, error)
- **Default Level**: Info level if not specified
- **Debug Output**: Includes detailed request routing, auth checks, and file serving attempts

**Process Output Capture**:
- All stdout/stderr from web apps and managed processes is captured with source identification
- **Text Format** (default): Output prefixed with `[source.stream]` (e.g., `[2025/boston.stdout]`)
- **JSON Format**: Structured logs with timestamp, source, stream, message, and tenant fields

**Output Destinations**:
- **Default**: Console output only (stdout) - no configuration required
- **File output**: Optional file logging with `{{app}}` template variable
- **Vector integration**: Professional log aggregation and processing
- **Multiple destinations**: Logs written to console, files, and Vector simultaneously

Configuration:
```yaml
# Default: text format to console only
# No configuration needed

# JSON format to console
logging:
  format: json

# Text format to both console and file
logging:
  file: /var/log/navigator/{{app}}.log

# JSON format to both console and file  
logging:
  format: json
  file: /var/log/navigator/{{app}}.log

# Vector integration for enterprise logging
logging:
  format: json
  vector:
    enabled: true
    socket: /tmp/navigator-vector.sock
    config: /etc/vector/vector.toml
```

**Template Variables**:
- `{{app}}` is replaced with the application or process name
- Creates separate log files per app: `redis.log`, `2025-boston.log`, etc.

**Vector Integration**:
- Vector automatically started as managed process when enabled
- Logs sent to Vector via high-performance Unix socket
- Supports all Vector sinks: Elasticsearch, S3, Kafka, NATS, etc.
- Graceful degradation if Vector fails to start

**Note**: Logging format is set at startup. To change the format, restart Navigator with the updated configuration. Configuration reload (SIGHUP) will apply the new format to newly started child processes, but Navigator's own logs will remain in their original format until restart.

Example JSON output:
```json
{"@timestamp":"2025-01-04T19:49:46-04:00","source":"redis","stream":"stdout","message":"Ready to accept connections"}
{"@timestamp":"2025-01-04T19:49:47-04:00","source":"2025/boston","stream":"stderr","message":"Error: Connection refused","tenant":"boston"}
```

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
- **YAML configuration**: Modern configuration format
- **Framework independence**: Support for Rails, Django, Node.js, and other web frameworks
- **Monitoring**: Built-in logging for requests, static files, and process management

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