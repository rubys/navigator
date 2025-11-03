# WebSocket Support

Navigator provides comprehensive WebSocket support for real-time applications, including built-in TurboCable, Action Cable integration, standalone WebSocket servers, and connection management.

## Overview

Navigator supports WebSocket connections through multiple approaches:

- **Built-in TurboCable**: Native WebSocket support with zero external dependencies (recommended)
- **Rails Action Cable**: Standard Rails WebSocket integration with Redis/Solid Cable
- **Standalone WebSocket Servers**: External WebSocket services
- **Pattern-Based Routing**: Flexible WebSocket endpoint matching
- **Connection Management**: Proper connection lifecycle handling

## Built-in TurboCable Support (Recommended)

Navigator includes native TurboCable/WebSocket support, eliminating the need for external dependencies:

```yaml
# Zero configuration needed - enabled by default!
cable:
  enabled: true                   # Default: true
  path: "/cable"                  # WebSocket endpoint
  broadcast_path: "/_broadcast"   # Broadcast endpoint (localhost-only)
```

### Benefits

- **89% memory reduction**: 163 MB (Action Cable + Redis) → 18 MB (built-in)
- **Zero external dependencies**: No Redis, no Solid Cable, no managed processes
- **Zero configuration**: Works automatically with sensible defaults
- **Perfect for Turbo Streams**: Server → client broadcasts (HTML and JSON)
- **Secure by default**: Authenticated WebSocket, localhost-only broadcasts

### Quick Start

1. **Add TurboCable to Rails**:
   ```ruby
   # Gemfile
   gem 'turbo_cable', github: 'rubys/turbo_cable'
   ```

2. **Configure broadcast URL**:
   ```yaml
   # config/navigator.yml
   applications:
     env:
       TURBO_CABLE_BROADCAST_URL: "http://localhost:3000/_broadcast"
   ```

3. **Use in views**:
   ```erb
   <%= turbo_stream_from "live-scores" %>
   ```

That's it! No Redis, no Action Cable server, no additional configuration.

### When to Use Built-in TurboCable

**Perfect for:**
- Single-server applications
- Process-per-tenant multi-tenancy (each tenant isolated)
- Server → client real-time updates
- Turbo Streams applications
- Memory-constrained deployments

**Not appropriate for:**
- Horizontally scaled apps with load balancing
- Bidirectional WebSocket (client actions)
- Chat applications requiring client → server messages

**See**: [TurboCable Documentation](turbocable.md) for complete usage, examples, and migration guide.

## Action Cable Integration

### Basic Action Cable Setup

```yaml
applications:
  tenants:
    - name: main-app
      path: /
      working_dir: /var/www/app
      
    # Dedicated Action Cable tenant
    - name: cable
      path: /cable
      working_dir: /var/www/app
      force_max_concurrent_requests: 0  # Unlimited connections for WebSockets
```

### Rails Configuration

```ruby title="config/cable.yml"
production:
  adapter: redis
  url: <%= ENV['REDIS_URL'] %>
  channel_prefix: myapp_production

development:
  adapter: redis
  url: redis://localhost:6379/1
```

```ruby title="config/routes.rb"
Rails.application.routes.draw do
  # Action Cable endpoint
  mount ActionCable.server => '/cable'
  
  # Your other routes
  resources :messages
end
```

### Navigator Configuration for Action Cable

```yaml
server:
  listen: 3000
  static:
    public_dir: /var/www/app/public

pools:
  max_size: 10
  idle_timeout: 600        # Longer timeout for persistent connections

applications:
  global_env:
    RAILS_ENV: production
    REDIS_URL: "${REDIS_URL}"
    ACTION_CABLE_ALLOWED_REQUEST_ORIGINS: "https://myapp.com"
    
  tenants:
    # Main HTTP application
    - name: web
      path: /
      working_dir: /var/www/app
      
    # Action Cable WebSocket connections
    - name: cable
      path: /cable
      working_dir: /var/www/app
      special: true    # Skip environment variable substitution
      env:
        RAILS_ENV: production
        ACTION_CABLE_URL: "ws://localhost:3000/cable"
      force_max_concurrent_requests: 0  # Allow unlimited WebSocket connections

managed_processes:
  - name: redis
    command: redis-server
    args: [/etc/redis/redis.conf]
    auto_restart: true
```

## Standalone WebSocket Servers

### External WebSocket Service

```yaml
applications:
  tenants:
    - name: main-app
      path: /
      working_dir: /var/www/app
      
    # Proxy to external WebSocket server
    - name: websocket-service
      path: /ws/
      standalone_server: "localhost:8080"  # External WebSocket server
```

### Multiple WebSocket Endpoints

```yaml
applications:
  tenants:
    - name: web-app
      path: /
      
    # Different WebSocket services
    - name: chat-service
      path: /chat/
      standalone_server: "chat-server:9000"
      
    - name: notifications
      path: /notifications/
      standalone_server: "notification-server:9001"
      
    - name: live-updates
      path: /live/
      standalone_server: "update-server:9002"
```

## Pattern-Based WebSocket Routing

### Flexible Path Matching

```yaml
applications:
  tenants:
    # Match any path ending with /socket
    - name: websocket-wildcard
      path: /api/
      match_pattern: "*/socket"
      working_dir: /var/www/app
      force_max_concurrent_requests: 0
      
    # Match tenant-specific WebSocket paths
    - name: tenant-websockets
      path: /tenants/
      match_pattern: "/tenants/*/ws"
      working_dir: /var/www/app
```

### Multi-Tenant WebSocket Setup

```yaml
applications:
  env:
    TENANT_ID: "${tenant_id}"
    WEBSOCKET_PATH: "/tenants/${tenant_id}/ws"
    
  tenants:
    - name: tenant-a-ws
      path: /tenants/a/ws
      working_dir: /var/www/app
      var:
        tenant_id: "tenant_a"
      force_max_concurrent_requests: 0
      
    - name: tenant-b-ws  
      path: /tenants/b/ws
      working_dir: /var/www/app
      var:
        tenant_id: "tenant_b"
      force_max_concurrent_requests: 0
```

## WebSocket Connection Tracking

Navigator can track active WebSocket connections to prevent apps from shutting down during idle timeouts while connections are active. This feature is **enabled by default** but can be configured globally or per-tenant.

### When to Disable Tracking

Disable WebSocket tracking for tenants that:
- Proxy WebSockets to standalone servers (e.g., separate Action Cable service)
- Don't handle WebSocket connections directly
- Have minimal memory/performance requirements

### Global Configuration

```yaml
applications:
  track_websockets: true  # Default: true (enabled)

  tenants:
    - name: web-app
      # Inherits global setting (true)
      path: /

    - name: api-app
      # Inherits global setting (true)
      path: /api
```

### Per-Tenant Override

```yaml
applications:
  track_websockets: true  # Global default

  tenants:
    # This app proxies WebSockets elsewhere - disable tracking
    - name: web-app
      path: /
      track_websockets: false  # Override: disable tracking

    # This app handles WebSockets directly - keep tracking enabled
    - name: chat-app
      path: /chat
      track_websockets: true   # Override: explicitly enable

    # This app doesn't specify - inherits global (true)
    - name: api-app
      path: /api
```

### Example: Reverse Proxy to Action Cable

```yaml
applications:
  track_websockets: true  # Global default

  tenants:
    # Main Rails app - proxies /cable to standalone server
    - name: web
      path: /
      track_websockets: false  # Don't track (proxies WebSockets elsewhere)

# Reverse proxy to standalone Action Cable server
routes:
  reverse_proxies:
    - path: "^/cable"
      target: "http://localhost:28080"
      websocket: true  # Enable WebSocket support for proxy
```

### How Tracking Works

When `track_websockets: true` (default):
- Navigator counts active WebSocket connections per tenant
- Apps with active connections won't stop during idle timeout
- Prevents unexpected WebSocket disconnections
- Small memory overhead for connection tracking

When `track_websockets: false`:
- No WebSocket connection counting
- App stops normally after idle timeout
- Slightly lower memory usage
- Use when app doesn't handle WebSockets directly

## Connection Management

### Connection Limits

```yaml
pools:
  max_size: 20             # More processes for WebSocket connections
  idle_timeout: 1800       # Longer timeout (30 minutes)

applications:
  tenants:
    # Regular HTTP requests - limited concurrency
    - name: web
      path: /
      max_concurrent_requests: 5
      
    # WebSocket connections - unlimited
    - name: websockets
      path: /ws/
      force_max_concurrent_requests: 0  # No limit for persistent connections
```

### Connection Monitoring

```yaml
# Enable debug logging for connection tracking
applications:
  global_env:
    LOG_LEVEL: debug
    ACTION_CABLE_LOG_LEVEL: debug
```

## Production Configuration

### High-Performance WebSocket Setup

```yaml
server:
  listen: 3000
  static:
    public_dir: /var/www/app/public

pools:
  max_size: 30             # More processes for high connection count
  idle_timeout: 3600       # 1 hour - keep connections alive
  start_port: 4000

# Disable suspension for WebSocket servers
suspend:
  enabled: false           # WebSockets prevent proper suspension

applications:
  global_env:
    RAILS_ENV: production
    REDIS_URL: "${REDIS_URL}"
    ACTION_CABLE_ALLOWED_REQUEST_ORIGINS: "${ALLOWED_ORIGINS}"
    
    # WebSocket-specific settings
    ACTION_CABLE_WORKER_POOL_SIZE: "10"
    ACTION_CABLE_MAX_REQUEST_SIZE: "1048576"  # 1MB
    
  tenants:
    - name: web
      path: /
      working_dir: /var/www/app
      max_concurrent_requests: 10
      
    - name: websockets
      path: /cable
      working_dir: /var/www/app
      force_max_concurrent_requests: 0
      env:
        ACTION_CABLE_MOUNT_PATH: "/cable"
        
managed_processes:
  - name: redis
    command: redis-server
    args: [--maxmemory, 512mb, --maxmemory-policy, allkeys-lru]
    auto_restart: true
```

## Security Configuration

### Origin Validation

```yaml
applications:
  global_env:
    # Restrict WebSocket origins in production
    ACTION_CABLE_ALLOWED_REQUEST_ORIGINS: "https://myapp.com,https://www.myapp.com"
    
    # Development - allow all origins
    # ACTION_CABLE_ALLOWED_REQUEST_ORIGINS: "*"
```

```ruby title="config/environments/production.rb"
# Action Cable security
config.action_cable.allowed_request_origins = [
  'https://myapp.com',
  'https://www.myapp.com'
]

# Use secure WebSocket URLs
config.action_cable.url = 'wss://myapp.com/cable'
```

### Authentication Integration

```ruby title="app/channels/application_cable/connection.rb"
module ApplicationCable
  class Connection < ActionCable::Connection::Base
    identified_by :current_user

    def connect
      self.current_user = find_verified_user
    end

    private

    def find_verified_user
      # Authentication via session
      if verified_user = User.find_by(id: session[:user_id])
        verified_user
      else
        reject_unauthorized_connection
      end
    end
  end
end
```

## Load Balancing and Scaling

### Multi-Instance WebSocket Setup

```yaml
server:
  listen: 3000
  static:
    public_dir: /var/www/app/public

pools:
  max_size: 30             # More processes for high connection count
  idle_timeout: 3600       # 1 hour - keep connections alive
  start_port: 4000

# Disable suspension for WebSocket servers
suspend:
  enabled: false           # WebSockets prevent proper suspension

### Load Balancer Configuration

```nginx title="nginx.conf - WebSocket Load Balancing"
upstream websocket_backend {
    # Enable session persistence for WebSockets
    ip_hash;
    
    server 127.0.0.1:3001;
    server 127.0.0.1:3002;
}

server {
    listen 80;
    server_name myapp.com;
    
    # WebSocket proxy
    location /cable {
        proxy_pass http://websocket_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket specific timeouts
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
    
    # Regular HTTP traffic
    location / {
        proxy_pass http://websocket_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Monitoring WebSocket Connections

### Connection Metrics

```bash
#!/bin/bash
# WebSocket connection monitoring

# Count active WebSocket connections
websocket_connections=$(netstat -an | grep :3000 | grep ESTABLISHED | wc -l)
echo "websocket_connections $websocket_connections"

# Monitor Rails processes handling WebSockets
rails_processes=$(pgrep -f "rails server" | wc -l)
echo "websocket_rails_processes $rails_processes"

# Check Redis connections (for Action Cable)
redis_connections=$(redis-cli info clients | grep connected_clients | cut -d: -f2)
echo "websocket_redis_connections $redis_connections"

# Memory usage of WebSocket processes
websocket_memory=$(ps aux | grep -E "(rails|navigator)" | awk '{sum+=$6} END {print sum}')
echo "websocket_memory_kb $websocket_memory"
```

### Action Cable Monitoring

```ruby title="config/initializers/action_cable_monitoring.rb"
# Monitor Action Cable performance
ActionCable.server.config.logger = Rails.logger

# Custom metrics for Action Cable
Rails.application.config.after_initialize do
  ActionCable.server.config.connection_class.prepend(Module.new do
    def connect
      Rails.logger.info "ActionCable connection established: #{request.remote_ip}"
      super
    end
    
    def disconnect
      Rails.logger.info "ActionCable connection closed: #{request.remote_ip}"
      super
    end
  end)
end
```

## Troubleshooting

### WebSocket Connection Issues

**Issue**: WebSocket connections fail to establish

**Debug Steps**:
```bash
# Check Navigator WebSocket support
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: test" \
     http://localhost:3000/cable

# Check Rails Action Cable
rails console
> ActionCable.server.config.logger = Logger.new(STDOUT)

# Test WebSocket connection
wscat -c ws://localhost:3000/cable
```

### High Memory Usage

**Issue**: WebSocket connections cause high memory usage

**Solutions**:
```yaml
# Limit connection lifetime
applications:
  global_env:
    ACTION_CABLE_CONNECTION_TIMEOUT: "300"  # 5 minutes
    
# Use Redis for connection state
applications:
  global_env:
    REDIS_URL: "${REDIS_URL}"
    ACTION_CABLE_ADAPTER: "redis"
```

### Connection Drops

**Issue**: WebSocket connections drop frequently

**Solutions**:
```yaml
# Increase timeouts
pools:
  idle_timeout: 3600       # 1 hour

# Disable suspension for WebSocket servers
suspend:
  enabled: false

# Configure keepalive
applications:
  global_env:
    ACTION_CABLE_KEEPALIVE: "30"  # 30 seconds
```

## Integration Examples

### Real-Time Chat Application

```ruby title="app/channels/chat_channel.rb"
class ChatChannel < ApplicationCable::Channel
  def subscribed
    stream_from "chat_#{params[:room]}"
  end

  def receive(data)
    ActionCable.server.broadcast("chat_#{params[:room]}", {
      message: data['message'],
      user: current_user.name,
      timestamp: Time.current
    })
  end
end
```

```yaml title="navigator.yml"
applications:
  tenants:
    - name: web
      path: /
      
    - name: chat
      path: /cable
      force_max_concurrent_requests: 0
      env:
        ACTION_CABLE_ALLOWED_REQUEST_ORIGINS: "https://myapp.com"
```

### Live Dashboard Updates

```ruby title="app/channels/dashboard_channel.rb"
class DashboardChannel < ApplicationCable::Channel
  def subscribed
    stream_from "dashboard_#{current_user.id}"
  end
end

# Broadcast updates from controllers
class MetricsController < ApplicationController
  def update
    # Update metrics logic
    
    ActionCable.server.broadcast("dashboard_#{current_user.id}", {
      metrics: updated_metrics,
      timestamp: Time.current
    })
  end
end
```

### Notification System

```ruby title="app/channels/notifications_channel.rb"
class NotificationsChannel < ApplicationCable::Channel
  def subscribed
    stream_from "notifications_#{current_user.id}"
  end
end

# Background job to send notifications
class NotificationJob < ApplicationJob
  def perform(user_id, notification)
    ActionCable.server.broadcast("notifications_#{user_id}", notification)
  end
end
```

## Best Practices

### 1. Connection Limits

```yaml
# Set appropriate limits
applications:
  tenants:
    - name: websockets
      force_max_concurrent_requests: 0  # Unlimited for WebSockets
    - name: web
      max_concurrent_requests: 10       # Limited for HTTP
```

### 2. Resource Management

```yaml
# Allocate sufficient resources
pools:
  max_size: 25             # More processes for connections
  idle_timeout: 3600       # Longer timeouts for persistent connections
```

### 3. Security

```yaml
# Validate origins in production
applications:
  global_env:
    ACTION_CABLE_ALLOWED_REQUEST_ORIGINS: "https://yourdomain.com"
```

### 4. Monitoring

```bash
# Monitor connection health
netstat -an | grep :3000 | grep ESTABLISHED | wc -l
redis-cli info clients
```

### 5. Graceful Degradation

```javascript
// Client-side reconnection logic
const cable = ActionCable.createConsumer('/cable');

cable.subscriptions.create('ChatChannel', {
  connected() {
    console.log('WebSocket connected');
  },
  
  disconnected() {
    console.log('WebSocket disconnected - attempting reconnect');
    // Implement exponential backoff reconnection
  },
  
  received(data) {
    // Handle received data
  }
});
```

## See Also

- [Action Cable Guide](../examples/action-cable.md)
- [Process Management](process-management.md)
- [Configuration Reference](../configuration/yaml-reference.md)
- [Production Deployment](../deployment/production.md)