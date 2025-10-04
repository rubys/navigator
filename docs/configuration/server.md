# Server Configuration

The `server` section defines how Navigator binds to network interfaces and serves HTTP requests.

## Basic Configuration

```yaml
server:
  listen: 3000              # Port to bind to (required)
  hostname: "localhost"     # Hostname for routing (optional)
  root_path: ""             # URL path prefix (optional)

  # Static file configuration
  static:
    public_dir: "./public"  # Public directory for static files
```

## Configuration Options

### listen

**Required**: Port number or address to bind to.

```yaml
# Port only (binds to all interfaces)
server:
  listen: 3000

# Specific interface and port
server:
  listen: "127.0.0.1:3000"

# IPv6 address
server:
  listen: "[::1]:3000"

# Environment variable
server:
  listen: "${PORT:-3000}"
```

**Examples by Environment**:
- **Development**: `3000` or `9999`
- **Production**: `3000`, `80`, or `8080`
- **Docker**: `3000` (expose via container ports)
- **Heroku**: `"${PORT}"` (dynamic port assignment)

### hostname

**Optional**: Expected hostname for Host header validation.

```yaml
server:
  hostname: "myapp.com"     # Production domain
  # or
  hostname: "localhost"     # Development
  # or  
  hostname: ""              # Accept any hostname (default)
```

**Use Cases**:
- **Production**: Set to your domain for security
- **Multi-domain**: Leave empty to accept all domains
- **Development**: Use `localhost` for local testing

### static.public_dir

**Optional**: Directory for serving static files.

```yaml
server:
  static:
    public_dir: "./public"           # Relative to working directory
    # or
    public_dir: "/var/www/public"    # Absolute path
    # or
    public_dir: "/app/public"        # Docker container path
```

**Default Behavior**: If not specified, Navigator looks for `public/` in the current working directory.

### root_path

**Optional**: URL prefix to strip from all requests before processing.

```yaml
server:
  root_path: "/myapp"       # Strip /myapp from all URLs
```

**Example**:
- Request: `GET /myapp/users/123`
- After root_path processing: `GET /users/123`
- Useful for deploying behind reverse proxies

## Environment-Specific Examples

### Development

```yaml
server:
  listen: 3000
  hostname: "localhost"
  static:
    public_dir: "./public"
```

**Characteristics**:
- Low port number for easy access
- Localhost-only for security
- Relative paths for portability

### Staging

```yaml
server:
  listen: 3000
  hostname: "staging.myapp.com"
  static:
    public_dir: "/var/www/app/public"
```

**Characteristics**:
- Production-like configuration
- Specific hostname for staging domain
- Absolute paths for production similarity

### Production

```yaml
server:
  listen: "${PORT:-3000}"
  hostname: "myapp.com"
  static:
    public_dir: "/var/www/app/public"
```

**Characteristics**:
- Environment variable for flexibility
- Production domain name
- Absolute paths for security

### Docker

```yaml
server:
  listen: 3000              # Internal container port
  hostname: ""              # Accept any hostname
  static:
    public_dir: "/app/public"
```

**Characteristics**:
- Fixed internal port (expose via Docker)
- Accept any hostname (Docker networking)
- Container-specific paths

### Behind Reverse Proxy

```yaml
server:
  listen: 3000              # Internal port
  hostname: ""              # Proxy handles hostname
  root_path: "/api"         # Strip /api prefix
  static:
    public_dir: "/app/public"
```

**nginx Configuration**:
```nginx
location /api/ {
    proxy_pass http://127.0.0.1:3000/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## Advanced Configuration

### Multi-Interface Binding

```yaml
# Development - bind to all interfaces
server:
  listen: "0.0.0.0:3000"

# Production - bind to specific interface
server:
  listen: "10.0.1.100:3000"
```

### Load Balancer Integration

```yaml
# Health check endpoint
server:
  listen: 3000
  hostname: ""              # Accept health check requests from any source

# In your Rails routes.rb:
# get '/up', to: 'health#show'
```

### Kubernetes Deployment

```yaml
server:
  listen: 8080              # Non-privileged port
  hostname: ""              # Pod networking
  static:
    public_dir: "/app/public"
```

```yaml title="deployment.yaml"
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: navigator
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
```

## Security Considerations

### Hostname Validation

```yaml
# Secure - validate hostname
server:
  hostname: "myapp.com"

# Less secure - accept any hostname  
server:
  hostname: ""
```

**Recommendation**: Always set hostname in production to prevent Host header attacks.

### Interface Binding

```yaml
# Secure - localhost only
server:
  listen: "127.0.0.1:3000"

# Less secure - all interfaces
server:
  listen: "0.0.0.0:3000"

# Most secure - specific interface
server:
  listen: "10.0.1.100:3000"
```

**Recommendation**: Bind to specific interfaces in production, use localhost in development.

## Performance Considerations

### Port Selection

- **Standard HTTP ports** (80, 8080): May require root privileges
- **High ports** (3000, 8000): No special privileges required
- **Ephemeral ports** (32768+): Avoid for servers

### File System Performance

```yaml
# Fast - SSD storage
server:
  static:
    public_dir: "/var/www/app/public"

# Slower - network storage
server:
  static:
    public_dir: "/nfs/shared/public"

# Fastest - memory filesystem (temporary files only)
server:
  static:
    public_dir: "/tmp/public"
```

### Connection Limits

Navigator handles connection limits at the OS level:

```bash
# Check current limits
ulimit -n

# Increase for production
echo "navigator soft nofile 65536" >> /etc/security/limits.conf
echo "navigator hard nofile 65536" >> /etc/security/limits.conf
```

## Testing Server Configuration

### Validate Configuration

```bash
# Test configuration syntax
navigator --validate config.yml

# Test with specific config
navigator --validate production.yml
```

### Network Testing

```bash
# Test port binding
netstat -tlnp | grep :3000

# Test hostname resolution
curl -H "Host: myapp.com" http://localhost:3000/

# Test public directory
curl -I http://localhost:3000/favicon.ico
```

### Load Testing

```bash
# Simple load test
ab -n 1000 -c 10 http://localhost:3000/

# WebSocket testing (if applicable)
wscat -c ws://localhost:3000/cable
```

## Common Issues

### Port Already in Use

**Error**: `bind: address already in use`

**Solutions**:
```bash
# Find process using port
lsof -i :3000
sudo fuser -k 3000/tcp

# Use different port
server:
  listen: 3001
```

### Permission Denied

**Error**: `bind: permission denied`

**Causes**: Trying to bind to ports < 1024 without root privileges

**Solutions**:
```bash
# Use unprivileged port
server:
  listen: 3000

# Or use capabilities (Linux)
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/navigator

# Or run as root (not recommended)
sudo navigator config.yml
```

### Public Directory Not Found

**Error**: Static files return 404

**Solutions**:
```bash
# Verify directory exists
ls -la /var/www/app/public/

# Check permissions
stat /var/www/app/public/

# Use absolute path
server:
  static:
    public_dir: "/var/www/app/public"
```

### Hostname Mismatch

**Error**: Requests rejected or routed incorrectly

**Solutions**:
```yaml
# Accept all hostnames (development)
server:
  hostname: ""

# Set correct hostname (production)
server:
  hostname: "myapp.com"

# Test with curl
curl -H "Host: myapp.com" http://localhost:3000/
```

## Integration Examples

### systemd Service

```ini title="/etc/systemd/system/navigator.service"
[Service]
Environment=PORT=3000
ExecStart=/usr/local/bin/navigator /etc/navigator/config.yml
```

### Docker Compose

```yaml title="docker-compose.yml"
services:
  navigator:
    ports:
      - "3000:3000"
    environment:
      - PORT=3000
    volumes:
      - ./public:/app/public:ro
```

### Heroku Deployment

```yaml
server:
  listen: "${PORT}"         # Heroku sets PORT dynamically
  public_dir: "./public"
```

## Best Practices

### 1. Environment Variables

```yaml
# Good - flexible configuration
server:
  listen: "${PORT:-3000}"
  hostname: "${HOSTNAME:-localhost}"

# Bad - hardcoded values
server:
  listen: 3000
  hostname: "localhost"
```

### 2. Security First

```yaml
# Production configuration
server:
  listen: "127.0.0.1:3000"  # Specific interface
  hostname: "myapp.com"     # Validate hostname
  static:
    public_dir: "/var/www/app/public"  # Absolute path
```

### 3. Path Management

```yaml
# Development - relative paths
server:
  static:
    public_dir: "./public"

# Production - absolute paths
server:
  static:
    public_dir: "/var/www/app/public"
```

### 4. Testing Strategy

```bash
# Always validate before deployment
navigator --validate config.yml

# Test connectivity
curl -I http://localhost:3000/up

# Verify static files
curl -I http://localhost:3000/favicon.ico
```

## See Also

- [YAML Reference](yaml-reference.md)
- [Static Files](static-files.md)
- [Applications](applications.md)
- [Production Deployment](../deployment/production.md)