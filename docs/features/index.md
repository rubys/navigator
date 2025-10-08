# Features

Navigator provides a comprehensive set of features for multi-tenant web applications, from basic process management to advanced regional routing.

!!! tip "What's New in v0.12.0"
    - **Configuration Modernization**: Reorganized YAML structure for better clarity
    - **Reverse Proxy Enhancements**: Capture group substitution in proxy targets
    - **Configurable Health Checks**: Global and per-tenant health check endpoints
    - **Startup Timeout Control**: Configurable maintenance page timing
    - **Per-Tenant WebSocket Tracking**: Fine-grained WebSocket connection control
    - **Reliability Improvements**: Fixed 502 errors during cold starts and shutdowns
    - **Better Status Codes**: 503 for maintenance, 499 for client disconnects

## Core Features

### :zap: Process Management
- **On-demand startup** - Web applications start when first requested
- **Dynamic port allocation** - Automatically finds available ports
- **Graceful shutdown** - Clean termination with proper cleanup
- **Auto-restart** - Crashed processes automatically restart
- **Resource limits** - Configurable pool sizes and timeouts

[Learn more about Process Management](process-management.md)

### :file_folder: Static File Serving
- **Direct filesystem serving** - Bypass application for static content
- **Configurable caching** - Set cache headers for different content types
- **Try files behavior** - Flexible file resolution with fallbacks
- **MIME type detection** - Automatic content type headers
- **Performance optimization** - 10-40x faster than application serving

[Learn more about Static Files](../configuration/static-files.md)

### :shield: Authentication
- **Multiple hash formats** - APR1, bcrypt, SHA support
- **Pattern-based exclusions** - Flexible public path configuration
- **Per-application auth** - Different authentication per app
- **htpasswd files** - Standard htpasswd format support

[Learn more about Authentication](../configuration/authentication.md)

### :arrows_counterclockwise: Hot Configuration Reload
- **Zero-downtime updates** - Reload config without restart
- **Signal-based** - SIGHUP or command-line trigger
- **Atomic changes** - All changes applied together
- **Process preservation** - Existing Rails processes continue running

[Learn more about Hot Reload](hot-reload.md)

### :electric_plug: WebSocket Support
- **Action Cable integration** - Built-in Rails Action Cable support
- **Standalone servers** - Manage WebSocket servers as separate processes
- **Connection upgrades** - Automatic WebSocket protocol upgrades
- **Long-lived connections** - Efficient handling of persistent connections

[Learn more about WebSocket Support](websocket-support.md)

### :hook: Lifecycle Hooks
- **Server hooks** - Execute commands at Navigator lifecycle events (start, ready, idle, resume)
- **Tenant hooks** - Execute commands when tenants start or stop
- **Variable substitution** - Use tenant variables in hook commands
- **Flexible integration** - Integrate with external services, monitoring, backups

[Learn more about Lifecycle Hooks](lifecycle-hooks.md)

## Advanced Features

### :globe_with_meridians: Fly-Replay Routing
- **Multi-target routing** - Route to regions, apps, or specific machines
- **Smart fallback** - Automatic reverse proxy for large requests
- **Pattern matching** - Flexible URL pattern configuration
- **Method filtering** - Apply rules to specific HTTP methods

[Learn more about Fly-Replay](fly-replay.md)

### :sleeping: Machine Suspension
- **Auto-suspend** - Fly.io machines suspend after idle timeout
- **Request tracking** - Monitor active requests for idle detection
- **Automatic wake** - Machines wake on incoming requests
- **Zero-downtime** - Seamless suspend/resume cycles

[Learn more about Machine Suspension](machine-suspend.md)

### :cookie: Sticky Sessions
- **Cookie-based affinity** - Route users to the same machine
- **Cross-region support** - Works across all Fly.io regions
- **Automatic failover** - Graceful handling of unavailable machines
- **Path-specific sessions** - Configure sticky sessions for specific paths

[Learn more about Sticky Sessions](sticky-sessions.md)

### :link: Try Files Behavior
- **Flexible resolution** - Try multiple file paths and extensions
- **Multiple suffixes** - Try different file extensions
- **Index file support** - Automatic index.html resolution
- **Application fallback** - Proxy to application when no file found

[Learn more about Try Files](try-files.md)

### :scroll: Structured Logging
- **slog integration** - Modern Go logging with structured output
- **Configurable levels** - Debug, info, warn, error levels
- **Request tracking** - Detailed request flow logging
- **Performance metrics** - Response times and process information

[Learn more about Logging](logging.md)

## Performance Features

### Smart Resource Management
- **Dynamic allocation** - Processes start/stop based on demand
- **Connection pooling** - Efficient connection reuse
- **Memory optimization** - Minimal memory footprint
- **CPU efficiency** - Low CPU overhead

### Caching Strategies
- **Static file caching** - Long-lived cache headers
- **Process caching** - Keep Rails processes warm
- **Configuration caching** - Cached config parsing

### Load Balancing
- **Round-robin** - Distribute requests across processes
- **Health checking** - Avoid sending requests to crashed processes
- **Graceful degradation** - Handle process failures elegantly

## Security Features

### Access Control
- **HTTP Basic Auth** - Industry standard authentication
- **Pattern-based exclusions** - Flexible public path configuration
- **Per-application security** - Different security policies per app

### Process Isolation
- **Separate processes** - Rails apps run in separate processes
- **Working directories** - Each app has its own working directory
- **Environment isolation** - Separate environment variables

### Configuration Security
- **Environment variables** - Secrets from environment, not config files
- **File permissions** - Secure configuration file handling
- **No hardcoded secrets** - Template-based secret management

## Monitoring and Observability

### Request Tracking
- **Structured logging** - JSON-formatted logs with request context
- **Response time tracking** - Monitor application performance
- **Error tracking** - Detailed error information

### Process Monitoring
- **Process lifecycle** - Track process starts, stops, crashes
- **Resource usage** - Monitor memory and CPU usage
- **Health status** - Process health checking

### System Integration
- **systemd compatibility** - Works well with systemd
- **Log aggregation** - Compatible with log management systems
- **Metrics export** - Structured data for monitoring systems

## Platform-Specific Features

### Cross-Platform Support (v0.11.0+)
- **Linux**: AMD64, ARM64 - Full feature support with SIGTSTP/SIGTERM
- **macOS**: Intel (AMD64), Apple Silicon (ARM64) - Native macOS signal handling
- **Windows**: AMD64, ARM64 - Graceful shutdown with os.Exit
- **Platform-specific optimizations** - Uses native OS primitives where appropriate

[Learn more about Architecture](../architecture.md)

### Fly.io Integration
- **Machine suspension** - Auto-suspend idle machines
- **Fly-Replay routing** - Intelligent request routing
- **Regional deployment** - Deploy close to users
- **Internal networking** - Secure inter-service communication

### General Cloud Support
- **Health check endpoints** - Standard health checking
- **Graceful shutdown** - Proper signal handling
- **Environment configuration** - 12-factor app compliance
- **Container friendly** - Works well in containers

## Feature Roadmap

### Short Term (Next Release)
- **Metrics endpoint** - Prometheus-compatible metrics
- **SSL termination** - Optional HTTPS support
- **Rate limiting** - Built-in rate limiting

### Medium Term
- **Auto-scaling** - Dynamic process scaling
- **Circuit breakers** - Failure isolation
- **Distributed tracing** - OpenTelemetry integration

### Long Term
- **Multi-server support** - Clustering capabilities
- **Advanced routing** - Content-based routing
- **Plugin system** - Extensible architecture

## Getting Started with Features

1. **Start Simple** - Begin with basic process management
2. **Add Static Files** - Optimize performance with direct serving
3. **Configure Authentication** - Secure your applications
4. **Enable Hot Reload** - Streamline configuration updates
5. **Explore Advanced Features** - Try Fly-Replay and machine suspension

## See Also

- [Configuration Guide](../configuration/index.md)
- [Examples](../examples/index.md)
- [Deployment Guide](../deployment/index.md)
