# Features

Navigator provides a comprehensive set of features designed specifically for Rails applications, from basic process management to advanced regional routing.

## Core Features

### :zap: Process Management
- **On-demand startup** - Rails apps start when first requested
- **Dynamic port allocation** - Automatically finds available ports
- **Graceful shutdown** - Clean termination with proper cleanup
- **Auto-restart** - Crashed processes automatically restart
- **Resource limits** - Configurable pool sizes and timeouts

[Learn more about Process Management](process-management.md)

### :file_folder: Static File Serving
- **Direct filesystem serving** - Bypass Rails for static content
- **Configurable caching** - Set cache headers for different content types
- **Try files behavior** - nginx-style file resolution
- **MIME type detection** - Automatic content type headers
- **Performance optimization** - 10-40x faster than Rails serving

[Learn more about Static Files](../configuration/static-files.md)

### :shield: Authentication
- **Multiple hash formats** - APR1, bcrypt, SHA support
- **Pattern-based exclusions** - Flexible public path configuration
- **Per-application auth** - Different authentication per app
- **htpasswd compatibility** - Drop-in replacement for nginx auth

[Learn more about Authentication](../configuration/authentication.md)

### :arrows_counterclockwise: Hot Configuration Reload
- **Zero-downtime updates** - Reload config without restart
- **Signal-based** - SIGHUP or command-line trigger
- **Atomic changes** - All changes applied together
- **Process preservation** - Existing Rails processes continue running

[Learn more about Hot Reload](hot-reload.md)

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

### :link: Try Files Behavior
- **nginx compatibility** - Same behavior as nginx try_files
- **Multiple suffixes** - Try different file extensions
- **Index file support** - Automatic index.html resolution
- **Rails fallback** - Proxy to Rails when no file found

[Learn more about Try Files](try-files.md)

### :scroll: Structured Logging
- **slog integration** - Modern Go logging with structured output
- **Configurable levels** - Debug, info, warn, error levels
- **Request tracking** - Detailed request flow logging
- **Performance metrics** - Response times and process information

[Learn more about Logging](logging.md)

## Feature Comparison

### vs nginx + Passenger

| Feature | Navigator | nginx + Passenger |
|---------|-----------|------------------|
| **Configuration** | Single YAML file | Multiple config files |
| **Process management** | Built-in | Requires Passenger |
| **Hot reload** | ✅ SIGHUP | ❌ Restart required |
| **Multi-tenant** | ✅ Native support | ⚠️ Complex setup |
| **Memory usage** | ~20MB base | ~100MB+ base |
| **Setup complexity** | Simple | Complex |

### vs Standalone Puma

| Feature | Navigator | Standalone Puma |
|---------|-----------|----------------|
| **Static files** | ✅ Optimized serving | ❌ Rails overhead |
| **Process pooling** | ✅ Multiple processes | ⚠️ Single process |
| **Authentication** | ✅ Built-in | ❌ Application level |
| **Load balancing** | ✅ Automatic | ❌ External required |
| **Zero-downtime deploys** | ✅ Process rotation | ⚠️ Manual setup |

### vs Docker + Kubernetes

| Feature | Navigator | Docker + K8s |
|---------|-----------|-------------|
| **Resource usage** | ✅ Lightweight | ❌ Heavy overhead |
| **Deployment complexity** | ✅ Single binary | ❌ Complex orchestration |
| **Local development** | ✅ Same as production | ❌ Different environment |
| **Scaling** | ⚠️ Single server | ✅ Multi-server |
| **Service discovery** | ✅ Built-in routing | ✅ Native K8s |

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
