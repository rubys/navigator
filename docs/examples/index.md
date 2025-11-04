# Examples

Ready-to-use Navigator configurations for common scenarios. Each example is complete and tested.

!!! warning "Updated for v1.0.0"
    All examples have been updated for Navigator v1.0.0. See the [YAML Reference](../configuration/yaml-reference.md) for complete configuration details.

## Quick Reference

| Example | Use Case | Key Features |
|---------|----------|--------------|
| [Single Rails App](single-tenant.md) | Basic Rails application | Simple setup, static files |
| [Multi-Tenant](multi-tenant.md) | Multiple customers/databases | Isolated apps, path routing |
| [With Redis](with-redis.md) | Caching and sessions | Managed Redis process |
| [With Sidekiq](with-sidekiq.md) | Background jobs | Worker process management |
| [Action Cable](action-cable.md) | WebSockets | Real-time features |

## Choose Your Scenario

### I want to serve a single Rails application
→ Start with [Single Rails App](single-tenant.md)

### I have multiple customers with separate databases
→ See [Multi-Tenant Setup](multi-tenant.md)

### I need background job processing
→ Check [With Sidekiq](with-sidekiq.md)

### I need WebSocket support
→ Read [Action Cable](action-cable.md)

### I'm deploying to Fly.io
→ Follow [Fly.io Deployment](fly-deployment.md)

### I need a systemd service
→ Use [Systemd Service](systemd.md)

## Configuration Templates

All examples follow this structure:

```yaml
# Server configuration
server:
  listen: 3000
  static:
    public_dir: ./public

# Authentication (optional)
auth:
  enabled: false

# Applications
applications:
  tenants: []

# Managed processes (optional)
managed_processes: []

# Routing rules (optional)
routes:
  rewrites: []
  fly:
    replay: []
```

## Tips for Using Examples

1. **Start Simple**: Begin with the basic example closest to your needs
2. **Modify Gradually**: Make small changes and test each one
3. **Check Logs**: Use `LOG_LEVEL=debug` for troubleshooting
4. **Test Locally**: Verify configuration before deploying
5. **Use Absolute Paths**: In production, use absolute paths for directories

## Common Patterns

### Environment-Specific Configuration

Use environment variables for different environments:

```yaml
applications:
  global_env:
    RAILS_ENV: "${RAILS_ENV:-development}"
    DATABASE_URL: "${DATABASE_URL}"
```

### Path-Based Routing

Route different paths to different apps:

```yaml
applications:
  tenants:
    - name: api
      path: /api/
    - name: admin
      path: /admin/
    - name: main
      path: /
```

### Resource Limits

Control resource usage:

```yaml
pools:
  max_size: 10
  idle_timeout: 300
  start_port: 4000
```

## Need Help?

- Can't find your scenario? Check the [Configuration Guide](../configuration/index.md)
- Having issues? Check the documentation or [open an issue](https://github.com/rubys/navigator/issues)
- Want to contribute an example? [Open a PR](https://github.com/rubys/navigator/pulls)