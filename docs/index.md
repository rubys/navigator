# Navigator

A lightweight, fast Go-based web server for multi-tenant web applications with on-demand process management. Framework-independent with built-in support for Rails, Django, Node.js, and other frameworks.

!!! success "Latest Release: v0.11.0"
    **Major refactoring release** with modular architecture, enhanced test coverage (81.2%), and full cross-platform support (Linux, macOS, Windows).

    [View Release Notes](https://github.com/rubys/navigator/releases/tag/v0.11.0)

## What is Navigator?

Navigator is a modern alternative to nginx + Passenger, designed for multi-tenant web applications across different frameworks. It provides intelligent request routing, dynamic process management, and built-in support for modern deployment patterns like Fly.io, Azure Deployment Stamps, and container orchestration.

<div class="grid cards" markdown>

-   :rocket: **Fast & Lightweight**

    ---

    Single binary with minimal dependencies. Lower memory footprint than nginx/Passenger.

-   :gear: **Process Management**

    ---

    Starts web apps on-demand, manages Redis/Sidekiq, automatic cleanup of stale processes. Framework-agnostic with configurable commands.

-   :shield: **Production Ready**

    ---

    Used in production serving 75+ dance studios across 8 countries.

-   :arrows_counterclockwise: **Hot Reload**

    ---

    Update configuration without restart using SIGHUP signal.

</div>

## Quick Start

=== "Download Binary"

    ```bash
    # Download latest release
    curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz | tar xz
    
    # Run with your config
    ./navigator config/navigator.yml
    ```

=== "Build from Source"

    ```bash
    # Clone and build
    git clone https://github.com/rubys/navigator.git
    cd navigator
    make build
    
    # Run Navigator
    ./bin/navigator config/navigator.yml
    ```

=== "Docker"

    ```bash
    # Run with Docker
    docker run -v $(pwd)/config:/config \
               -v $(pwd)/app:/app \
               -p 3000:3000 \
               rubys/navigator /config/navigator.yml
    ```

## Simple Example

Here's a minimal configuration to serve a Rails application:

```yaml title="config/navigator.yml"
server:
  listen: 3000
  public_dir: ./public

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/rails/app
```

## Key Features

### :zap: Intelligent Routing
- URL rewriting with redirects and rewrites
- Try-files behavior for static content
- WebSocket and Action Cable support
- Fly-Replay for regional routing

### :lock: Authentication
- Full htpasswd support (APR1, bcrypt, SHA)
- Pattern-based exclusions for public paths
- Per-path authentication realms

### :file_folder: Static File Serving
- Direct filesystem serving bypasses Rails
- Configurable cache headers
- Automatic MIME type detection
- Try multiple file extensions

### :cloud: Cloud Native
- Fly.io machine auto-suspend
- Multi-region deployment support
- Health check endpoints
- Graceful shutdown handling

## Why Navigator?

### Compared to nginx + Passenger

| Feature | Navigator | nginx + Passenger |
|---------|-----------|------------------|
| Configuration | Simple YAML | Complex nginx.conf |
| Memory Usage | ~20MB base | ~100MB+ base |
| Process Management | Built-in | Requires Passenger |
| Hot Reload | ✅ Native | ❌ Restart required |
| Multi-tenant | ✅ Native | ⚠️ Complex setup |

### Perfect For

- **Multi-tenant SaaS** - Each customer gets isolated database/instance
- **Regional deployments** - Deploy closer to users with Fly.io
- **Development environments** - Replace complex nginx setups
- **Resource-constrained servers** - Lower memory footprint

## Architecture

Navigator v0.11.0+ uses a **modular package structure** for improved maintainability and testing:

- **internal/server/** - HTTP handling, routing, static files, proxying
- **internal/process/** - Web app and managed process lifecycle
- **internal/config/** - Configuration loading and validation
- **internal/auth/** - Authentication (htpasswd)
- **internal/proxy/** - Reverse proxy and Fly-Replay logic
- **internal/idle/** - Fly.io machine idle management
- **internal/errors/** - Domain-specific error constructors
- **internal/logging/** - Structured logging helpers
- **internal/utils/** - Common utilities (duration parsing, environment)

**Benefits of the refactored architecture**:

- ✅ **81.2% test coverage** - Comprehensive testing of all components
- ✅ **Cross-platform** - Native support for Linux, macOS, and Windows
- ✅ **Maintainable** - Clear separation of concerns, focused modules
- ✅ **Single binary** - Still deploys as one self-contained executable

See [REFACTORING.md](https://github.com/rubys/navigator/blob/main/REFACTORING.md) for technical details.

## Real-World Use Cases

Navigator powers production applications including:

- Dance studio management systems with 75+ tenants
- Regional PDF generation services
- Multi-database Rails applications
- WebSocket-enabled real-time apps

## Next Steps

<div class="grid cards" markdown>

-   :books: **[Getting Started](getting-started/index.md)**

    ---

    Install Navigator and deploy your first Rails app in 5 minutes

-   :wrench: **[Configuration Guide](configuration/index.md)**

    ---

    Learn about YAML configuration options and best practices

-   :bulb: **[Examples](examples/index.md)**

    ---

    Copy-paste ready configurations for common scenarios

-   :question: **[Reference](reference/index.md)**

    ---

    Complete CLI options, environment variables, and signals

</div>

## Community

- [GitHub Issues](https://github.com/rubys/navigator/issues) - Report bugs or request features
- [Discussions](https://github.com/rubys/navigator/discussions) - Ask questions and share experiences
- [Releases](https://github.com/rubys/navigator/releases) - Download binaries and view changelog