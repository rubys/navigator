# Navigator

A lightweight web server for multi-tenant applications with on-demand process management. Deploy multiple customers or projects from a single configuration file.

!!! success "Latest Release: v1.2.0"
    **Enhanced Caching & Logging** - Navigator 1.2 adds immutable asset caching, response headers, trust_proxy support, and improved Vector integration.

    [View Release Notes](https://github.com/rubys/navigator/releases/tag/v1.2.0) | [Changelog](https://github.com/rubys/navigator/blob/main/CHANGELOG.md)

## Features

- Serve multiple tenants with isolated processes and databases
- Built-in TurboCable WebSocket support (89% memory savings vs Action Cable)
- Execute standalone CGI scripts without starting web apps
- Automatic machine suspension when idle (Fly.io)
- Regional routing with automatic fallback

Used in production serving 75+ customers across 8 countries.

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

## Common Use Cases

- Multi-tenant applications with isolated databases
  [See configuration →](use-cases.md#use-case-1-multi-tenant-and-monorepos)

- Auto-suspend idle machines on Fly.io
  [Learn more →](use-cases.md#use-case-2-machine-auto-suspend-flyio)

- Standalone Action Cable with Rails 8
  [View example →](use-cases.md#use-case-3-websocket-support)

- Regional routing with Fly-Replay
  [See example →](use-cases.md#use-case-4-dynamic-routing-with-fly-replay)

## Learn More

<div class="grid cards" markdown>

-   :books: **[Use Cases](use-cases.md)**

    ---

    Real-world examples and configuration patterns

-   :building_construction: **[Architecture](architecture.md)**

    ---

    Technical details and design decisions

-   :mag: **[Features](features/index.md)**

    ---

    Complete feature documentation

</div>

## Get Started

<div class="grid cards" markdown>

-   :rocket: **[Getting Started](getting-started/index.md)**

    ---

    Install and deploy your first app in 5 minutes

-   :bulb: **[Examples](examples/index.md)**

    ---

    Copy-paste ready configurations

-   :wrench: **[Configuration](configuration/index.md)**

    ---

    YAML configuration reference

-   :question: **[CLI Reference](reference/index.md)**

    ---

    Command-line options and signals

</div>