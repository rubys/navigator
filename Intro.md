# Introduction

There are many good reasons to like [SQLite](https://sqlite.org/), and this is one of the best reasons: [Multi-Tenant Rails: Everybody Gets a Database](https://www.youtube.com/watch?v=Sc4FJ0EZTAg). Some of the best work in this area is done by [Stephen Margheim](https://fractaledmind.com/) and [Mike Dalessio](https://mike.daless.io/), establishing Rails as a thought leader in this space.

I'm also a big fan of [fly.io](https://fly.io/) and [Kamal](https://kamal-deploy.org/), but truth be told neither is optimized for SQLite. In fact, no PaaS is.

Navigator was extracted from my [Showcase](https://github.com/rubys/showcase?tab=readme-ov-file#showcase) application to help with a number of issues, including but not limited to, the ones that arise from the use of SQLite. While SQLite was the motivation for a number of features in Navigator, nothing in Navigator is specific to SQLite.

The inspiration for Navigator was Basecamp's [thruster](https://github.com/basecamp/thruster?tab=readme-ov-file#thruster), and like thruster, is a reverse proxy written in Go. To use it, modify your Dockerfile:

```dockerfile
COPY --from=samruby/navigator:latest /navigator /usr/local/bin/navigator
CMD ["navigator", "config/navigator.yml"]
```

## Use Case 1: Multi-tenant and Monorepos

The multi-tenant work linked at the top of this page will enable a single Rails application to support multiple tenant databases. My showcase application takes a different approach: launching a separate instance of the same Rails application for each tenant, varying environment variables such as `DATABASE_URL`.

More generally, many projects employ a [monorepo](https://en.wikipedia.org/wiki/Monorepo) containing a number of servers, for example a web server and an API server.

Navigator supports both patterns with:
- **Multi-tenant setup**: Multiple tenants sharing the same codebase with isolated databases using template variables
- **Monorepo structure**: API server, admin panel, and web server from different directories
- **Managed processes**: Background services like Sidekiq shared across tenants
- **Environment templating**: Using `${variable}` syntax for tenant-specific configuration
- **Different tenant types**: Production, staging/demo, and development environments

See [multi-tenant-monorepo-example.yml](examples/multi-tenant-monorepo-example.yml) for a complete configuration example.

## Use Case 2: Suspend/Stop

Fly.io has the ability to [autostop](https://fly.io/docs/launch/autostop-autostart/) or [suspend](https://fly.io/docs/reference/suspend-resume/) a machine when idle. This isn't merely for broke college students, it also makes it practical to distribute your application across hundreds of machines. Instead of these machines running 24/7, they will only run when needed.

If you want control over when a machine is to be stopped or suspended, or want actions to be taken immediately prior to or immediately after a state change, you will need to write code. Doing so will require you to keep track of requests.

All of this is something a reverse proxy like Navigator can do. See [suspend-stop-example.yml](examples/suspend-stop-example.yml) for a complete configuration example.

## Use Case 3: WebSockets

WebSockets enable two-way communication between a server and a browser. A WebSocket connection is typically open for minutes or hours rather than tens or hundreds of milliseconds.

Rails recommends running a separate [Standalone Cable Server](https://guides.rubyonrails.org/action_cable_overview.html#running-standalone-cable-servers) for performance, scalability, and stability.

Prior to Rails 8, the recommended configuration was three services: Rails, Redis, and Action Cable. With Rails 8, the new default is [Solid Cable](https://github.com/rails/solid_cable?tab=readme-ov-file#solid-cable) with SQLite. This requires Action Cable and Rails to be run on the same machine.

This could be accomplished via a reverse proxy like [Nginx](https://nginx.org/) or [Traefik](https://traefik.io/traefik), and a process manager like [Foreman](https://github.com/ddollar/foreman?tab=readme-ov-file#foreman) or [Overmind](https://github.com/DarthSim/overmind/blob/master/README.md).

Navigator can be configured to do both tasks, without needing to modify the configuration of your Rails application. See [websockets-example.yml](examples/websockets-example.yml) for a complete configuration example.

## Use Case 4: Sticky Sessions

Navigator provides built-in sticky session support using HTTP cookies, ensuring requests from the same client are routed to the same machine. This is particularly useful for maintaining WebSocket connections or accessing locally stored data on specific machines. See [sticky-sessions-example.yml](examples/sticky-sessions-example.yml) for a complete configuration example.

## Use Case 5: Routing

Fly.io supports [Dynamic Routing](https://fly.io/docs/networking/dynamic-request-routing/), which has two modes: _prefer_ or _force_. With _prefer_ requests that can't be routed to the intended destination are routed to an available server. With _force_, such requests fail.

Having requests routed somewhere means that your application has an ability to detect, log, recover, or take other actions. For now, Navigator shows a maintenance page, which is more user friendly and something that can be searched for in logs. See [routing-example.yml](examples/routing-example.yml) for a complete configuration example.

## Use Case 6: Log Aggregation with Vector

Modern applications generate logs from multiple sources: web servers, background workers, databases, and the application itself. When running across multiple machines or regions, centralizing these logs becomes essential for debugging and monitoring. [Vector](https://vector.dev/) is a high-performance observability data pipeline that can collect, transform, and route logs to various destinations.

Navigator includes built-in Vector integration that:
- **Automatic Process Management**: Starts and manages Vector as a high-priority process
- **Unix Socket Streaming**: Efficient log transfer via Unix sockets without file I/O overhead
- **Structured Logging**: JSON format support for rich metadata and easy parsing
- **Source Identification**: Automatically tags logs with source (tenant, process name, stream)
- **Graceful Degradation**: Continues operating if Vector is unavailable
- **Multiple Destinations**: Route logs to files, Elasticsearch, S3, or any Vector-supported sink

This integration eliminates the need for complex log collection setups and provides a unified logging pipeline for all Navigator-managed processes. See [navigator-with-vector.yml](examples/navigator-with-vector.yml) and [vector.toml](examples/vector.toml) for complete configuration examples.

## Future Use Case Ideas

While Navigator currently focuses on the use cases above, there are several interesting directions worth exploring:

### A/B Testing and Feature Flags
Route different users to different application instances based on cookies, headers, or URL parameters. This would enable testing new features with a subset of users or gradual feature rollouts.

### Local Development Environment
Replace complex nginx/Docker setups for local development. A single Navigator configuration could manage frontend, backend, and supporting services with automatic SSL certificates for HTTPS testing.

### Rate Limiting and Protection
Per-tenant or per-IP rate limiting to prevent abuse, with automatic blocking of suspicious traffic patterns and potential integration with fail2ban or similar tools.

### Health Checks and Circuit Breakers
Automatic health monitoring of backend services with circuit breaker patterns to prevent cascading failures and enable graceful degradation when services are unavailable.

### Database Connection Management
PgBouncer-like functionality for Postgres connections or sophisticated SQLite connection management for multi-tenant setups, including read/write splitting for database replicas.

### Enhanced Observability
Integration with observability platforms (Datadog, New Relic, OpenTelemetry) for distributed tracing and comprehensive monitoring across all tenants.

These ideas represent potential evolution paths for Navigator as usage patterns and requirements emerge from production deployments.