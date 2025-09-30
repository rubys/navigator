# Navigator

A lightweight web server for multi-tenant applications with on-demand process management. Deploy multiple customers or projects from a single configuration file.

!!! success "Latest Release: v0.11.0"
    Enhanced test coverage (81.2%) and full cross-platform support (Linux, macOS, Windows).

    [View Release Notes](https://github.com/rubys/navigator/releases/tag/v0.11.0)

## What Can Navigator Do For You?

- **Serve multiple tenants** - Each customer gets their own database and isolated process
- **Save on hosting costs** - Automatic machine suspension when idle (Fly.io)
- **Simplify WebSocket deployments** - Built-in support for Rails Action Cable
- **Deploy globally** - Smart regional routing with automatic fallback

Trusted in production serving 75+ customers across 8 countries.

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

- **Multi-tenant SaaS** - Serve multiple customers with isolated databases
  [See configuration →](use-cases.md#use-case-1-multi-tenant-and-monorepos)

- **Cost optimization** - Auto-suspend idle machines on Fly.io
  [Learn more →](use-cases.md#use-case-2-machine-auto-suspend-flyio)

- **WebSocket support** - Standalone Action Cable with Rails 8
  [View example →](use-cases.md#use-case-3-websocket-support)

- **Sticky sessions** - Route users to the same machine
  [Read guide →](use-cases.md#use-case-4-sticky-sessions)

- **Regional routing** - Deploy closer to your users
  [Explore Fly-Replay →](use-cases.md#use-case-5-dynamic-routing-with-fly-replay)

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