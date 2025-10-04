# Use Cases

Navigator excels at solving specific architectural challenges in multi-tenant and distributed applications. This page explores proven use cases with real-world examples and configuration patterns.

## Background

Navigator was designed to address challenges that arise in modern multi-tenant Rails applications, particularly those using SQLite for database-per-tenant architectures. While SQLite was the motivation for many features, Navigator is database-agnostic and works with any Rails backend.

**Key Resources:**
- [Multi-Tenant Rails: Everybody Gets a Database](https://www.youtube.com/watch?v=Sc4FJ0EZTAg) - Stephen Margheim
- [SQLite Replication with Beamer](https://www.youtube.com/watch?v=lcved9uEV5U) - Mike Dalessio
- [Kamal Geo Proxy](https://www.youtube.com/watch?v=gcwzWzC7gUA&t=3541s) - Kevin McConnell

Navigator is inspired by Basecamp's [thruster](https://github.com/basecamp/thruster) and implemented as a reverse proxy in Go.

## Quick Start

To use Navigator in your Dockerfile:

```dockerfile
COPY --from=samruby/navigator:latest /navigator /usr/local/bin/navigator
CMD ["navigator", "config/navigator.yml"]
```

---

## Use Case 1: Multi-Tenant and Monorepos

**Challenge**: Serve multiple tenants with isolated databases from a single codebase, potentially with different application components (API, admin, web).

**Solution**: Navigator launches separate instances of the same Rails application for each tenant, varying environment variables such as `DATABASE_URL`.

### Capabilities

- **Multi-tenant setup**: Multiple tenants sharing the same codebase with isolated databases using template variables
- **Monorepo structure**: API server, admin panel, and web server from different directories
- **Managed processes**: Background services like Sidekiq shared across tenants
- **Different tenant types**: Production, staging/demo, and development environments

### Configuration Example

```yaml
server:
  listen: 3000
  static:
    public_dir: public

applications:
  env:
    DATABASE_URL: "sqlite3:db/${tenant_id}.sqlite3"
    RAILS_ENV: "production"

  tenants:
    - name: acme
      path: /acme
      working_dir: /app
      var:
        tenant_id: acme_production

    - name: widgets-co
      path: /widgets
      working_dir: /app
      var:
        tenant_id: widgets_production
```

### Benefits

- **Isolation**: Each tenant has its own process and database
- **Resource efficiency**: Processes start on-demand and idle out
- **Simple deployment**: Single Docker image for all tenants
- **Template flexibility**: Customize environment per tenant

See [multi-tenant-monorepo-example.yml](https://github.com/rubys/navigator/blob/main/examples/multi-tenant-monorepo-example.yml) for a complete example.

---

## Use Case 2: Machine Auto-Suspend (Fly.io)

**Challenge**: Reduce costs and enable global distribution by automatically suspending or stopping machines when idle.

**Solution**: Navigator tracks active requests and triggers machine suspension or shutdown after a configurable idle timeout.

### Why This Matters

Fly.io supports [autostop](https://fly.io/docs/launch/autostop-autostart/) and [suspend](https://fly.io/docs/reference/suspend-resume/) for idle machines. This isn't just for cost savings—it makes it practical to distribute your application across hundreds of machines that only run when needed.

Manual control over when machines suspend/stop and what actions to take before/after state changes requires custom code to track requests. Navigator handles this automatically.

### Lifecycle Hooks

Navigator's lifecycle hooks enable powerful automation for machine state changes:

**Pre-suspension hooks** (`idle` event) could:
- Upload databases and files to S3 or cloud storage
- Send metrics to monitoring services
- Notify other services about pending suspension
- Checkpoint long-running computations

**Post-resume hooks** (`resume` event) could:
- Sync state from shared storage
- Reconnect to external services
- Notify monitoring systems of machine availability
- Restore application state

### Configuration Example

```yaml
server:
  idle:
    action: suspend  # or "stop"
    timeout: 20m

hooks:
  server:
    idle:
      - command: /usr/local/bin/backup-to-s3.sh
        timeout: 30s
    resume:
      - command: /usr/local/bin/restore-from-s3.sh
        timeout: 30s

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /app
```

See [suspend-stop-example.yml](https://github.com/rubys/navigator/blob/main/examples/suspend-stop-example.yml) for a complete example.

---

## Use Case 3: WebSocket Support

**Challenge**: Run WebSocket servers alongside Rails applications, especially with Rails 8's Solid Cable which requires Action Cable and Rails on the same machine.

**Solution**: Navigator acts as both reverse proxy and process manager, routing WebSocket requests to standalone servers.

### Background

WebSockets enable two-way communication between server and browser, with connections typically open for minutes or hours. Rails recommends running a separate [Standalone Cable Server](https://guides.rubyonrails.org/action_cable_overview.html#running-standalone-cable-servers) for performance, scalability, and stability.

**Rails 7 and earlier**: Three services (Rails, Redis, Action Cable)
**Rails 8 with Solid Cable**: Action Cable and Rails must run on the same machine

### How Navigator Helps

Traditionally, you'd need:
- Reverse proxy ([Nginx](https://nginx.org/) or [Traefik](https://traefik.io/traefik)) for routing
- Process manager ([Foreman](https://github.com/ddollar/foreman) or [Overmind](https://github.com/DarthSim/overmind)) for services

Navigator combines both without modifying your Rails configuration.

### Configuration Example

```yaml
server:
  listen: 3000
  static:
    public_dir: public

  # Reverse proxy WebSocket requests to standalone Action Cable server
  reverse_proxies:
    - name: action-cable
      path: "^/cable"
      target: "http://localhost:28080"
      websocket: true

applications:
  # Disable WebSocket tracking for main app (proxies WebSockets elsewhere)
  track_websockets: true  # Global default

  tenants:
    - name: main-app
      path: /
      working_dir: /app
      track_websockets: false  # Override: this app proxies WebSockets to standalone server
```

### WebSocket Connection Tracking

Navigator can track WebSocket connections to prevent apps from shutting down during idle timeouts. This is useful when apps handle WebSockets directly, but unnecessary when proxying to standalone servers.

**When to disable tracking:**
- Apps that proxy WebSockets to separate services (like example above)
- Apps that don't handle WebSocket connections at all
- When minimizing memory overhead is critical

See [websockets-example.yml](https://github.com/rubys/navigator/blob/main/examples/websockets-example.yml) and [WebSocket Support documentation](features/websocket-support.md) for complete examples.

---

## Use Case 4: Sticky Sessions

**Challenge**: Maintain session affinity so requests from the same client route to the same machine, crucial for WebSocket connections or locally stored data.

**Solution**: Navigator provides built-in sticky session support using HTTP cookies.

### Features

- **Cookie-based routing**: Stores machine ID in HTTP-only cookie
- **Cross-region support**: Works across all Fly.io regions
- **Automatic failover**: Serves maintenance page if target machine unavailable
- **Large request handling**: Falls back to reverse proxy for requests >1MB
- **Path-specific sessions**: Optional configuration for specific URL paths
- **Configurable duration**: Session lifetime using Go duration format

### Configuration Example

```yaml
routes:
  fly:
    sticky_sessions:
      enabled: true
      cookie_name: "_navigator_machine"
      cookie_max_age: "2h"
      cookie_secure: true
      cookie_httponly: true
      paths:
        - "/app/*"
        - "/dashboard/*"

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /app
```

### Benefits

- **Consistent routing**: Same user always hits same machine
- **WebSocket compatibility**: Maintains long-lived connections
- **Local cache hits**: Access machine-specific cached data
- **Simple configuration**: No external session store required

See [sticky-sessions-example.yml](https://github.com/rubys/navigator/blob/main/examples/sticky-sessions-example.yml) and [Sticky Sessions feature page](features/sticky-sessions.md) for more details.

---

## Use Case 5: Dynamic Routing with Fly-Replay

**Challenge**: Route requests to specific regions, apps, or machines while handling failures gracefully.

**Solution**: Navigator implements Fly.io's [Dynamic Routing](https://fly.io/docs/networking/dynamic-request-routing/) with smart fallback behavior.

### Fly-Replay Modes

**Prefer mode**: Requests that can't route to intended destination go to available server
**Force mode**: Failed routing returns error

### Navigator's Approach

Navigator takes a middle ground: when requests can't be routed to the intended destination, it shows a user-friendly maintenance page that can be searched in logs. This is better than silent failures and provides visibility for debugging.

### Configuration Example

```yaml
server:
  rewrite_rules:
    - pattern: "^/api/(.*)"
      rewrite: "/$1"
      fly_replay:
        app: api-backend
        status: 307

    - pattern: "^/admin/(.*)"
      rewrite: "/$1"
      fly_replay:
        region: ord  # Chicago
        status: 307

applications:
  tenants:
    - name: web-app
      path: /
      working_dir: /app
```

### Features

- **Multi-target routing**: Route to regions, apps, or specific machines
- **Smart fallback**: Automatic reverse proxy for requests >1MB
- **Maintenance pages**: User-friendly error pages
- **Pattern matching**: Flexible URL pattern configuration
- **Method filtering**: Apply rules to specific HTTP methods

See [routing-example.yml](https://github.com/rubys/navigator/blob/main/examples/routing-example.yml) for a complete example.

---

## Future Use Case Ideas

While Navigator currently focuses on the use cases above, there are several interesting directions worth exploring based on production usage patterns and community feedback.

### A/B Testing and Feature Flags

Route different users to different application instances based on cookies, headers, or URL parameters. This would enable:

- Testing new features with a subset of users
- Gradual feature rollouts
- Canary deployments at the proxy level
- User cohort experimentation

**Potential Configuration**:
```yaml
routing:
  ab_tests:
    - name: new-ui-test
      header: X-Feature-Flag
      value: new-ui
      target: /v2
      percentage: 10  # 10% of users
```

### Rate Limiting and Protection

Per-tenant or per-IP rate limiting to prevent abuse:

- Automatic blocking of suspicious traffic patterns
- Per-tenant request quotas
- Integration with fail2ban or similar tools
- DDoS protection at the application level

**Potential Configuration**:
```yaml
rate_limiting:
  global:
    requests_per_minute: 1000
  per_tenant:
    requests_per_minute: 100
  per_ip:
    requests_per_minute: 60
```

### Health Checks and Circuit Breakers

Automatic health monitoring with resilience patterns:

- Health checks for tenant apps before proxying (already implemented)
- Circuit breaker patterns to prevent cascading failures
- Graceful degradation when services are unavailable
- Automatic retry with exponential backoff for Fly-Replay fallback (already implemented)

**Potential Configuration**:
```yaml
health_checks:
  interval: 30s
  timeout: 5s
  unhealthy_threshold: 3

circuit_breaker:
  failure_threshold: 5
  timeout: 60s
```

### Enhanced Observability

Integration with observability platforms:

- Datadog, New Relic, Honeycomb integration
- OpenTelemetry distributed tracing
- Prometheus metrics endpoint (planned)
- Comprehensive monitoring across all tenants

**Potential Configuration**:
```yaml
observability:
  opentelemetry:
    enabled: true
    endpoint: http://collector:4318
  prometheus:
    enabled: true
    path: /metrics
```

### Request Transformation

Advanced request/response manipulation:

- Header injection and removal
- Request/response body transformation
- Content negotiation
- Protocol upgrades (HTTP/1.1 → HTTP/2)

### Geographic Routing Intelligence

Smarter geographic routing based on:

- Client IP geolocation
- Latency measurements
- Regional capacity
- Time-of-day patterns

These ideas represent potential evolution paths as usage patterns and requirements emerge from production deployments. Community feedback and real-world needs will guide prioritization.

---

## Choosing Your Use Case

| Your Scenario | Recommended Use Case | Key Features |
|---------------|---------------------|--------------|
| Multiple customers with separate DBs | Multi-Tenant (#1) | Process isolation, template variables |
| Global deployment on Fly.io | Auto-Suspend (#2) | Cost optimization, lifecycle hooks |
| Rails 8 with Solid Cable | WebSockets (#3) | Process management, routing |
| Stateful sessions across regions | Sticky Sessions (#4) | Cookie-based affinity |
| Multi-region deployment | Dynamic Routing (#5) | Fly-Replay, maintenance pages |

## See Also

- [Examples](examples/index.md) - Complete configuration examples
- [Features](features/index.md) - Detailed feature documentation
- [Configuration Reference](configuration/yaml-reference.md) - Complete YAML reference
- [Architecture](architecture.md) - Technical implementation details
- [Deployment](deployment/index.md) - Production deployment guides