# Action Cable and WebSockets

Configure Navigator to support Rails Action Cable for real-time WebSocket connections, including standalone cable servers and multi-tenant WebSocket routing.

## Use Cases

- Real-time chat applications
- Live notifications and updates
- Collaborative editing
- Live dashboards and analytics
- Real-time gaming features
- Live streaming data

## Basic Action Cable Setup

```yaml title="navigator.yml"
server:
  listen: 3000
  public_dir: ./public

# Redis for Action Cable
managed_processes:
  - name: redis
    command: redis-server
    args: [--port, "6379"]
    auto_restart: true

# Static files
static:
  directories:
    - path: /assets/
      root: public/assets/
      cache: 86400
  extensions: [css, js, png, jpg, gif]

# Applications
applications:
  global_env:
    RAILS_ENV: production
    REDIS_URL: redis://localhost:6379/0
    CABLE_REDIS_URL: redis://localhost:6379/5  # Separate DB for Cable
    
  tenants:
    # Main Rails application
    - name: main
      path: /
      working_dir: /var/www/app
      
    # Action Cable server (WebSocket connections)
    - name: cable
      path: /cable
      working_dir: /var/www/app
      force_max_concurrent_requests: 0  # Unlimited for WebSockets
```

## Standalone Action Cable Server

Run Action Cable as a separate process for better performance:

```yaml title="navigator.yml"
server:
  listen: 3000

managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    
  # Standalone Action Cable server
  - name: action-cable
    command: bundle
    args: [exec, puma, -p, "28080", cable/config.ru]
    working_dir: /var/www/app
    env:
      RAILS_ENV: production
      REDIS_URL: redis://localhost:6379/5
      PORT: "28080"
    auto_restart: true
    start_delay: 2

applications:
  global_env:
    RAILS_ENV: production
    ACTION_CABLE_URL: ws://localhost:3000/cable
    
  tenants:
    # Main Rails app
    - name: main
      path: /
      working_dir: /var/www/app
      
    # Proxy /cable to standalone server
    - name: cable
      path: /cable
      standalone_server: "localhost:28080"
```

### Standalone Cable Configuration

```ruby title="cable/config.ru"
# Standalone Action Cable server
require_relative '../config/environment'

# Configure Action Cable
ActionCable.server.config.cable = {
  adapter: 'redis',
  url: ENV.fetch('REDIS_URL', 'redis://localhost:6379/5'),
  channel_prefix: Rails.application.class.module_parent_name.underscore
}

# Disable request forgery protection for WebSocket connections
ActionCable.server.config.disable_request_forgery_protection = true

# Allow connections from any origin in development
if Rails.env.development?
  ActionCable.server.config.allowed_request_origins = [/.*/]
end

run ActionCable.server
```

## Rails Action Cable Configuration

### 1. Configure Action Cable

```ruby title="config/cable.yml"
development:
  adapter: redis
  url: <%= ENV.fetch('CABLE_REDIS_URL', 'redis://localhost:6379/5') %>
  channel_prefix: myapp_development

production:
  adapter: redis
  url: <%= ENV.fetch('CABLE_REDIS_URL', 'redis://localhost:6379/5') %>
  channel_prefix: myapp_production
  
test:
  adapter: test
```

### 2. Configure Routes

```ruby title="config/routes.rb"
Rails.application.routes.draw do
  # Mount Action Cable server
  mount ActionCable.server => '/cable'
  
  # Other routes...
end
```

### 3. Configure Connection

```ruby title="app/channels/application_cable/connection.rb"
module ApplicationCable
  class Connection < ActionCable::Connection::Base
    identified_by :current_user
    
    def connect
      self.current_user = find_verified_user
    end
    
    private
    
    def find_verified_user
      # Get user from session or token
      if verified_user = User.find_by(id: session[:user_id])
        verified_user
      else
        # For API authentication, check Authorization header
        if token = request.headers['Authorization']&.remove('Bearer ')
          User.find_by(api_token: token)
        else
          reject_unauthorized_connection
        end
      end
    end
  end
end
```

## WebSocket Channel Examples

### Chat Channel

```ruby title="app/channels/chat_channel.rb"
class ChatChannel < ApplicationCable::Channel
  def subscribed
    # Subscribe to room-specific stream
    stream_from "chat_#{params[:room]}"
    
    # Track user presence
    Redis.current.sadd("chat_users_#{params[:room]}", current_user.id)
    broadcast_user_joined
  end
  
  def unsubscribed
    # Remove user from presence tracking
    Redis.current.srem("chat_users_#{params[:room]}", current_user.id)
    broadcast_user_left
  end
  
  def speak(data)
    # Validate and save message
    message = Message.create!(
      user: current_user,
      room: params[:room],
      content: data['message']
    )
    
    # Broadcast to all room subscribers
    ActionCable.server.broadcast(
      "chat_#{params[:room]}",
      {
        type: 'message',
        message: message.content,
        user: current_user.name,
        timestamp: message.created_at.iso8601
      }
    )
  end
  
  private
  
  def broadcast_user_joined
    ActionCable.server.broadcast(
      "chat_#{params[:room]}",
      {
        type: 'user_joined',
        user: current_user.name,
        users_count: Redis.current.scard("chat_users_#{params[:room]}")
      }
    )
  end
  
  def broadcast_user_left
    ActionCable.server.broadcast(
      "chat_#{params[:room]}",
      {
        type: 'user_left',
        user: current_user.name,
        users_count: Redis.current.scard("chat_users_#{params[:room]}")
      }
    )
  end
end
```

### Notification Channel

```ruby title="app/channels/notification_channel.rb"
class NotificationChannel < ApplicationCable::Channel
  def subscribed
    # Subscribe to user-specific notifications
    stream_from "notifications_#{current_user.id}"
  end
  
  def mark_read(data)
    notification = current_user.notifications.find(data['id'])
    notification.update(read_at: Time.current)
  end
end
```

### Live Updates Channel

```ruby title="app/channels/live_updates_channel.rb"
class LiveUpdatesChannel < ApplicationCable::Channel
  def subscribed
    # Subscribe to model updates
    stream_from "updates_#{params[:model]}_#{params[:id]}"
  end
end

# Trigger updates from models
class Post < ApplicationRecord
  after_update_commit do
    ActionCable.server.broadcast(
      "updates_post_#{id}",
      {
        type: 'update',
        model: 'post',
        id: id,
        attributes: slice(:title, :content, :updated_at)
      }
    )
  end
end
```

## JavaScript Client Setup

### Basic Connection

```javascript
// app/assets/javascripts/cable.js
import { createConsumer } from "@rails/actioncable"

// Create WebSocket connection
const cable = createConsumer("/cable")

// Subscribe to chat channel
const chatChannel = cable.subscriptions.create(
  { channel: "ChatChannel", room: "general" },
  {
    connected() {
      console.log("Connected to chat channel")
    },
    
    disconnected() {
      console.log("Disconnected from chat channel")
    },
    
    received(data) {
      switch(data.type) {
        case 'message':
          addMessageToDOM(data)
          break
        case 'user_joined':
          updateUserCount(data.users_count)
          showNotification(`${data.user} joined`)
          break
        case 'user_left':
          updateUserCount(data.users_count)
          showNotification(`${data.user} left`)
          break
      }
    },
    
    speak(message) {
      this.perform('speak', { message: message })
    }
  }
)

// Send message
function sendMessage() {
  const input = document.getElementById('message-input')
  const message = input.value.trim()
  
  if (message) {
    chatChannel.speak(message)
    input.value = ''
  }
}
```

### Authentication Token

```javascript
// For API-based authentication
import { createConsumer } from "@rails/actioncable"

const token = localStorage.getItem('auth_token')
const cable = createConsumer(`/cable?token=${token}`)
```

### React Integration

```javascript
// React hook for Action Cable
import { useEffect, useState } from 'react'
import { createConsumer } from '@rails/actioncable'

export function useActionCable(channelName, params = {}) {
  const [cable] = useState(() => createConsumer('/cable'))
  const [connected, setConnected] = useState(false)
  const [subscription, setSubscription] = useState(null)
  
  useEffect(() => {
    const sub = cable.subscriptions.create(
      { channel: channelName, ...params },
      {
        connected: () => setConnected(true),
        disconnected: () => setConnected(false),
        received: (data) => {
          // Handle received data
          console.log('Received:', data)
        }
      }
    )
    
    setSubscription(sub)
    
    return () => {
      sub.unsubscribe()
    }
  }, [cable, channelName, params])
  
  return { connected, subscription }
}

// Usage in component
function ChatRoom({ roomId }) {
  const { connected, subscription } = useActionCable('ChatChannel', { room: roomId })
  
  const sendMessage = (message) => {
    subscription?.perform('speak', { message })
  }
  
  return (
    <div>
      <div>Status: {connected ? 'Connected' : 'Disconnected'}</div>
      {/* Chat UI */}
    </div>
  )
}
```

## Multi-Tenant WebSocket Configuration

### Tenant-Specific Channels

```yaml title="navigator.yml"
applications:
  tenants:
    # Tenant-specific cable endpoints
    - name: tenant1-cable
      path: /tenant1/cable
      match_pattern: "/tenant1/cable"
      working_dir: /var/www/app
      env:
        TENANT_ID: tenant1
        CABLE_REDIS_DB: "6"
      force_max_concurrent_requests: 0
      
    - name: tenant2-cable
      path: /tenant2/cable
      match_pattern: "/tenant2/cable"
      working_dir: /var/www/app
      env:
        TENANT_ID: tenant2
        CABLE_REDIS_DB: "7"
      force_max_concurrent_requests: 0
```

### Dynamic Channel Routing

```ruby title="app/channels/application_cable/connection.rb"
module ApplicationCable
  class Connection < ActionCable::Connection::Base
    identified_by :current_user, :tenant_id
    
    def connect
      self.current_user = find_verified_user
      self.tenant_id = extract_tenant_id
    end
    
    private
    
    def extract_tenant_id
      # Extract tenant from path: /tenant1/cable
      path_segments = request.path.split('/')
      tenant_segment = path_segments[1]
      
      if Tenant.exists?(slug: tenant_segment)
        tenant_segment
      else
        reject_unauthorized_connection
      end
    end
  end
end
```

## Performance Optimization

### Connection Limits

```yaml title="navigator.yml"
applications:
  tenants:
    - name: cable
      path: /cable
      # Unlimited concurrent connections for WebSockets
      force_max_concurrent_requests: 0
      
      # But limit Rails processes to prevent resource exhaustion
      max_processes: 2
      idle_timeout: 300
```

### Redis Optimization

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args:
      # Optimize for pub/sub workload
      - --maxclients 10000
      - --timeout 0
      - --tcp-keepalive 60
      - --tcp-backlog 511
      
      # Pub/sub specific optimizations
      - --client-output-buffer-limit "pubsub 32mb 8mb 60"
      - --hz 10
```

### Action Cable Configuration

```ruby title="config/environments/production.rb"
Rails.application.configure do
  # Action Cable configuration
  config.action_cable.url = ENV.fetch('ACTION_CABLE_URL', 'ws://localhost:3000/cable')
  config.action_cable.allowed_request_origins = [
    ENV.fetch('ALLOWED_ORIGINS', 'https://example.com').split(',')
  ].flatten
  
  # Disable request forgery protection for API usage
  config.action_cable.disable_request_forgery_protection = ENV['DISABLE_CABLE_CSRF'] == 'true'
  
  # Connection pool settings
  config.action_cable.connection_class = -> { ApplicationCable::Connection }
  config.action_cable.worker_pool_size = ENV.fetch('CABLE_WORKER_POOL_SIZE', 4).to_i
end
```

## Monitoring and Debugging

### WebSocket Health Check

```ruby title="app/controllers/health_controller.rb"
class HealthController < ApplicationController
  def cable
    # Test Action Cable connection
    begin
      ActionCable.server.broadcast('health_check', { timestamp: Time.current })
      render json: { status: 'healthy', cable: 'connected' }
    rescue => e
      render json: { 
        status: 'unhealthy', 
        cable: 'disconnected',
        error: e.message 
      }, status: :service_unavailable
    end
  end
end
```

### Connection Monitoring

```ruby title="app/channels/application_cable/connection.rb"
module ApplicationCable
  class Connection < ActionCable::Connection::Base
    def connect
      Rails.logger.info "WebSocket connection attempt from #{request.remote_ip}"
      self.current_user = find_verified_user
      Rails.logger.info "WebSocket connected for user #{current_user.id}"
    end
    
    def disconnect
      Rails.logger.info "WebSocket disconnected for user #{current_user&.id}"
    end
  end
end
```

### Redis Monitoring

```bash
# Monitor Redis pub/sub activity
redis-cli monitor | grep -E "(PUBLISH|SUBSCRIBE)"

# Check Action Cable activity
redis-cli pubsub channels "*"
redis-cli pubsub numsub "chat_general"
```

## Security Considerations

### Origin Validation

```ruby title="config/environments/production.rb"
Rails.application.configure do
  config.action_cable.allowed_request_origins = [
    'https://myapp.com',
    'https://www.myapp.com',
    /https:\/\/.*\.myapp\.com/
  ]
end
```

### Token Authentication

```ruby title="app/channels/application_cable/connection.rb"
module ApplicationCable
  class Connection < ActionCable::Connection::Base
    def connect
      self.current_user = find_verified_user
    end
    
    private
    
    def find_verified_user
      # Check for token in query params or headers
      token = request.params[:token] || request.headers['Authorization']&.remove('Bearer ')
      
      if token && (user = User.find_by(api_token: token))
        user
      else
        reject_unauthorized_connection
      end
    end
  end
end
```

### Rate Limiting

```ruby title="app/channels/chat_channel.rb"
class ChatChannel < ApplicationCable::Channel
  def speak(data)
    # Rate limiting: max 10 messages per minute per user
    rate_limit_key = "chat_rate_limit:#{current_user.id}"
    current_count = Redis.current.get(rate_limit_key).to_i
    
    if current_count >= 10
      # Send error to client
      transmit({ error: 'Rate limit exceeded. Please slow down.' })
      return
    end
    
    # Increment counter with expiration
    Redis.current.multi do |multi|
      multi.incr(rate_limit_key)
      multi.expire(rate_limit_key, 60)
    end
    
    # Process message...
  end
end
```

## Troubleshooting

### Connection Issues

1. **Check WebSocket upgrade**:
   ```bash
   curl -i -N -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Key: test" \
        -H "Sec-WebSocket-Version: 13" \
        http://localhost:3000/cable
   ```

2. **Verify Redis pub/sub**:
   ```bash
   # Terminal 1
   redis-cli subscribe test_channel
   
   # Terminal 2  
   redis-cli publish test_channel "hello"
   ```

3. **Check Navigator logs**:
   ```bash
   tail -f /var/log/navigator.log | grep -E "(cable|websocket)"
   ```

### Performance Issues

1. **Monitor connection count**:
   ```ruby
   # In Rails console
   ActionCable.server.connections.count
   ```

2. **Check Redis memory usage**:
   ```bash
   redis-cli info memory | grep used_memory_human
   ```

3. **Monitor message throughput**:
   ```bash
   redis-cli --latency-history -i 1
   ```

## Testing

### Test WebSocket Connection

```ruby title="test/channels/chat_channel_test.rb"
require 'test_helper'

class ChatChannelTest < ActionCable::Channel::TestCase
  test "subscribes to stream" do
    user = users(:alice)
    
    subscribe(room: "general")
    assert subscription.confirmed?
    assert_has_stream "chat_general"
  end
  
  test "speaks message" do
    user = users(:alice)
    
    subscribe(room: "general")
    
    perform :speak, message: "Hello world"
    
    assert_broadcast_on("chat_general", {
      type: 'message',
      message: "Hello world",
      user: user.name
    })
  end
end
```

### Integration Testing

```ruby title="test/integration/action_cable_test.rb"
require 'test_helper'

class ActionCableTest < ActionDispatch::IntegrationTest
  test "cable server is accessible" do
    get "/cable"
    assert_response :success
  end
end
```

## See Also

- [Redis Integration](with-redis.md)
- [Multi-Tenant Setup](multi-tenant.md)
- [Process Management](../features/process-management.md)
- [WebSocket Feature](../features/websocket-support.md)