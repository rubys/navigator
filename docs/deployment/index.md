# Deployment

Navigator deployment guides for different environments and platforms.

## Deployment Options

Navigator supports multiple deployment strategies:

### Production Environments

- **[Production Deployment](production.md)**: Complete production setup with systemd, security, monitoring
- **[Fly.io Deployment](fly-io.md)**: Deploy on Fly.io with machine suspension and regional distribution
- **[Monitoring Setup](monitoring.md)**: Comprehensive monitoring and observability configuration

### Platform-Specific Features

- **Machine Suspension**: Automatic cost optimization through idle machine suspension
- **Regional Distribution**: Deploy closer to users with intelligent routing
- **Process Management**: Managed external processes (Redis, Sidekiq, workers)
- **Static File Optimization**: Direct filesystem serving with try_files behavior
- **WebSocket Support**: Full WebSocket and Action Cable integration

## Quick Start

### Local Development

```bash
# Build Navigator
make build

# Run with configuration
./bin/navigator config/navigator.yml
```

### Production Deployment

1. **Choose your platform**:
   - [Production servers](production.md) - Traditional Linux servers with systemd
   - [Fly.io](fly-io.md) - Modern cloud platform with machine suspension

2. **Configure Navigator**:
   - Create YAML configuration file
   - Set up environment variables and secrets
   - Configure static file serving and authentication

3. **Set up monitoring**:
   - [Configure monitoring](monitoring.md) with health checks
   - Set up log aggregation and alerting
   - Monitor performance and costs

## Deployment Patterns

### Single-Tenant Application

Deploy one application per Navigator instance:

```yaml
applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /var/www/myapp
```

### Multi-Tenant Application  

Deploy multiple tenants sharing infrastructure:

```yaml
applications:
  tenants:
    - name: tenant-a
      path: /tenant-a/
      var:
        database: tenant_a_production
    - name: tenant-b  
      path: /tenant-b/
      var:
        database: tenant_b_production
```

### Microservices Architecture

Route different paths to different applications:

```yaml
applications:
  tenants:
    - name: web-app
      path: /
      working_dir: /var/www/web
    - name: api-service
      path: /api/
      working_dir: /var/www/api
    - name: admin-panel
      path: /admin/
      working_dir: /var/www/admin
```

## Security Considerations

### Authentication

- **htpasswd Integration**: Pattern-based authentication with multiple hash formats
- **Public Route Exclusions**: Serve static assets without authentication
- **Secret Management**: Environment variable substitution for sensitive data

### Network Security

- **Interface Binding**: Bind to specific interfaces for security
- **SSL Termination**: Configure HTTPS with reverse proxy
- **Request Filtering**: Method-based routing and filtering

## Performance Optimization

### Static File Serving

- **Direct Filesystem Access**: Bypass Rails for static content
- **Try Files**: Multiple extension attempts before Rails fallback
- **Content-Type Detection**: Automatic MIME type setting
- **Cache Headers**: Configurable caching for performance

### Process Management

- **Dynamic Port Allocation**: Automatic port discovery in configured ranges
- **Process Pooling**: Configurable pool sizes for different workloads
- **Idle Timeout**: Automatic process cleanup during low traffic
- **Graceful Shutdown**: Clean termination of all managed processes

### Resource Management

- **Memory Optimization**: Lower footprint than traditional setups
- **CPU Efficiency**: Single binary with minimal overhead
- **Connection Pooling**: Efficient proxy connection management
- **Request Buffering**: Smart handling of large requests

## Monitoring and Observability

### Health Checks

```yaml
# Configure health endpoints
server:
  listen: 3000
  
# Monitor via HTTP
curl http://localhost:3000/up
```

### Logging

- **Structured Logging**: JSON-formatted logs with consistent fields
- **Log Levels**: Configurable verbosity (debug, info, warn, error)  
- **Request Tracing**: Detailed request routing and performance data
- **Error Tracking**: Comprehensive error reporting and alerting

### Metrics

- **Process Metrics**: Memory, CPU, connection counts
- **Application Metrics**: Request rates, response times, error rates
- **Business Metrics**: Custom metrics via managed processes
- **Infrastructure Metrics**: System resources and health

## Best Practices

### 1. Configuration Management

```bash
# Use environment-specific configs
navigator config/production.yml    # Production
navigator config/staging.yml       # Staging  
navigator config/development.yml   # Development
```

### 2. Process Supervision

```bash
# Use systemd for production
systemctl enable navigator
systemctl start navigator
systemctl status navigator
```

### 3. Log Management

```bash
# Configure log rotation
journalctl -u navigator -f        # Follow logs
journalctl -u navigator --since yesterday  # Historical logs
```

### 4. Backup and Recovery

```bash
# Backup configuration
cp config/navigator.yml config/navigator.yml.backup

# Test configuration changes
navigator --validate config/navigator.yml
```

### 5. Scaling Strategy

```bash
# Start small and scale up
# Monitor resource usage
# Add machines/processes as needed
```

## Troubleshooting

### Common Issues

1. **Port Conflicts**: Use dynamic port allocation
2. **Process Startup**: Check PID file cleanup and port availability  
3. **Authentication**: Verify htpasswd file permissions and patterns
4. **Static Files**: Confirm file paths and permissions
5. **Proxy Errors**: Check Rails application startup and health

### Debugging Tools

```bash
# Check process status
ps aux | grep navigator

# View configuration
navigator --validate config.yml

# Monitor connections
netstat -tlnp | grep :3000

# Check logs
journalctl -u navigator -n 50
```

### Performance Issues

1. **High Memory Usage**: Adjust pool sizes and idle timeouts
2. **Slow Response**: Check static file serving and try_files configuration  
3. **Connection Errors**: Verify proxy retry and backoff settings
4. **Process Crashes**: Review managed process configuration and auto-restart

## Migration Guides

### From nginx/Passenger

1. **Static File Configuration**: Map nginx locations to Navigator static directories
2. **Authentication**: Convert nginx auth to htpasswd files
3. **Reverse Proxy**: Replace upstream blocks with tenant configuration
4. **SSL Termination**: Configure external SSL termination or load balancer

### From Other Platforms

1. **Process Management**: Replace supervisord/systemd process management
2. **Configuration Format**: Convert to YAML-based configuration
3. **Environment Variables**: Use Navigator's template system
4. **Health Checks**: Map health endpoints to Navigator patterns

## Support and Community

- **Documentation**: Complete guides and examples
- **Issue Tracking**: GitHub issue tracker for bug reports
- **Configuration Examples**: Real-world production configurations
- **Best Practices**: Community-driven recommendations

## See Also

- [Configuration Reference](../configuration/index.md)
- [Features Overview](../features/index.md)
- [Examples](../examples/index.md)