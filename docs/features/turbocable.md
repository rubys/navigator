# TurboCable Integration

Built-in WebSocket support for Rails applications using TurboCable, eliminating the need for external dependencies.

## Overview

Navigator provides native TurboCable/WebSocket support, allowing Rails applications to use real-time Turbo Streams without:
- Separate Action Cable server process (~153 MB)
- Redis or Solid Cable for pub/sub (~13 MB)
- Additional configuration and dependencies

**Memory savings**: 89% reduction (163 MB → 18 MB per machine)

## How It Works

Navigator handles two endpoints:

1. **WebSocket Endpoint** (`/cable`): Client connections (requires authentication)
2. **Broadcast Endpoint** (`/_broadcast`): Rails app broadcasts (localhost-only, no auth)

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │ WebSocket /cable (authenticated)
       │
┌──────▼──────────────────────────────────┐
│           Navigator                      │
│  ┌────────────────────────────────────┐ │
│  │  Cable Handler (18 MB)             │ │
│  │  - Manages WebSocket connections   │ │
│  │  - Tracks subscriptions in-memory  │ │
│  │  - Distributes broadcasts          │ │
│  └────────────────────────────────────┘ │
└───────────────▲────────────────────────┘
                │ POST /_broadcast (localhost-only)
                │
      ┌─────────┴──────────┐
      │   Rails App        │
      │  (TurboCable gem)  │
      └────────────────────┘
```

## Configuration

### Default Configuration (Zero Config)

Navigator enables WebSocket support automatically with sensible defaults:

```yaml
# No configuration needed! These are the defaults:
cable:
  enabled: true
  path: "/cable"
  broadcast_path: "/_broadcast"
```

### Custom Endpoints

Customize the WebSocket and broadcast paths:

```yaml
cable:
  enabled: true
  path: "/websocket"
  broadcast_path: "/internal/broadcast"
```

### Disable Built-in WebSocket

To use traditional Action Cable instead:

```yaml
cable:
  enabled: false
```

## Rails Setup

### 1. Add TurboCable Gem

```ruby
# Gemfile
gem 'turbo_cable', github: 'rubys/turbo_cable'
```

```bash
bundle install
rails generate turbo_cable:install
```

### 2. Configure Broadcast URL

Set the environment variable to point broadcasts to Navigator:

```yaml
# config/navigator.yml
applications:
  env:
    TURBO_CABLE_BROADCAST_URL: "http://localhost:3000/_broadcast"
```

Or in your Rails configuration:

```ruby
# config/application.rb
config.turbo_cable_broadcast_url = ENV.fetch(
  'TURBO_CABLE_BROADCAST_URL',
  'http://localhost:3000/_broadcast'
)
```

### 3. Use in Views

TurboCable uses the same API as Action Cable:

```erb
<%# app/views/scores/index.html.erb %>
<div id="scores-board">
  <%= turbo_stream_from "live-scores" %>
  <%= render @scores %>
</div>
```

### 4. Broadcast Updates

```ruby
# app/models/score.rb
class Score < ApplicationRecord
  after_save do
    broadcast_replace_later_to "live-scores",
      partial: "scores/score",
      target: dom_id(self)
  end
end
```

## Security Model

### WebSocket Endpoint (`/cable`)

- **Requires authentication** (same as other protected endpoints)
- Checks htpasswd credentials before allowing WebSocket upgrade
- Honors `auth.public_paths` configuration
- Standard Navigator authentication flow

### Broadcast Endpoint (`/_broadcast`)

- **Localhost-only** for security (127.0.0.1, ::1, localhost)
- No authentication needed (internal communication)
- Returns 403 Forbidden for non-localhost requests
- Processed BEFORE authentication middleware

This security model ensures:
- Public clients cannot broadcast directly
- Rails apps on same machine can broadcast without credentials
- WebSocket clients are properly authenticated

## Memory Comparison

Real measurements from production (iad region, November 2025):

### Action Cable + Redis
```
navigator          1.0%    21 MB
puma (cable)       7.6%   153 MB
redis-server       0.6%    13 MB
─────────────────────────────────
Total WebSocket:   8.2%   163 MB
```

### TurboCable (Built-in)
```
navigator          0.9%    18 MB
─────────────────────────────────
Total WebSocket:   0.9%    18 MB
```

**Savings: 145 MB per machine (89% reduction)**

With 8 regional machines: **1.16 GB saved infrastructure-wide**

## Migration from Action Cable

### Step 1: Add TurboCable

```ruby
# Gemfile
gem 'turbo_cable', github: 'rubys/turbo_cable'
```

### Step 2: Update Configuration

```yaml
# config/navigator.yml
applications:
  env:
    TURBO_CABLE_BROADCAST_URL: "http://localhost:3000/_broadcast"

# Optional: Keep Action Cable running during migration
managed_processes:
  - name: redis
    command: redis-server
  - name: action-cable
    command: bundle
    args: [exec, puma, -p, "28080", cable/config.ru]
```

### Step 3: Test with One Tenant

Deploy and verify WebSocket functionality works with TurboCable.

### Step 4: Remove Action Cable

Once verified, remove managed processes:

```yaml
# config/navigator.yml - Remove these sections
# managed_processes:
#   - name: redis
#     command: redis-server
#   - name: action-cable
#     command: bundle
```

### Code Changes

For channels that only use `stream_from` (no custom actions):

**Before (Action Cable)**:
```ruby
# app/channels/scores_channel.rb
class ScoresChannel < ApplicationCable::Channel
  def subscribed
    stream_from "live-scores"
  end
end
```

**After (TurboCable)**:
```erb
<%# Just use turbo_stream_from in views, delete the channel file %>
<%= turbo_stream_from "live-scores" %>
```

If you have custom channel actions, continue using Action Cable or refactor to HTTP endpoints.

## Examples

### Live Dashboard Updates

```ruby
# app/jobs/update_dashboard_job.rb
class UpdateDashboardJob < ApplicationJob
  def perform(user_id)
    user = User.find(user_id)

    # Broadcast HTML update
    Turbo::StreamsChannel.broadcast_replace_to(
      "dashboard_#{user_id}",
      target: "metrics",
      partial: "dashboard/metrics",
      locals: { metrics: user.current_metrics }
    )
  end
end
```

```erb
<%# app/views/dashboard/show.html.erb %>
<div id="dashboard">
  <%= turbo_stream_from "dashboard_#{current_user.id}" %>

  <div id="metrics">
    <%= render "dashboard/metrics", metrics: @metrics %>
  </div>
</div>
```

### Progress Bar for Long Operations

```ruby
# app/jobs/export_job.rb
class ExportJob < ApplicationJob
  def perform(export_id)
    export = Export.find(export_id)

    # Broadcast progress updates
    (1..100).each do |progress|
      TurboCable::Broadcastable.broadcast_json(
        "export_#{export_id}",
        { progress: progress, status: 'processing' }
      )

      # Do work...
      sleep 0.1
    end

    # Broadcast completion
    TurboCable::Broadcastable.broadcast_json(
      "export_#{export_id}",
      { progress: 100, status: 'complete', url: export.download_url }
    )
  end
end
```

```javascript
// app/javascript/controllers/export_controller.js
import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
  connect() {
    this.subscription = this.element.dataset.stream
  }

  handleMessage(event) {
    const { stream, data } = event.detail
    if (stream === this.subscription) {
      this.updateProgress(data.progress, data.status)

      if (data.status === 'complete') {
        window.location.href = data.url
      }
    }
  }

  updateProgress(value, status) {
    this.element.querySelector('.progress-bar').style.width = `${value}%`
    this.element.querySelector('.status').textContent = status
  }
}
```

### Real-Time Notifications

```ruby
# app/models/notification.rb
class Notification < ApplicationRecord
  after_create_commit do
    broadcast_prepend_later_to(
      "notifications_#{user_id}",
      target: "notifications",
      partial: "notifications/notification"
    )
  end
end
```

```erb
<%# app/views/layouts/application.html.erb %>
<div id="notifications" data-controller="notifications">
  <%= turbo_stream_from "notifications_#{current_user.id}" %>
  <%= render current_user.notifications.recent %>
</div>
```

## Custom JSON Broadcasting

Beyond Turbo Stream HTML, you can broadcast custom JSON:

```ruby
# Broadcast custom data
TurboCable::Broadcastable.broadcast_json(
  "charts_#{user_id}",
  {
    type: 'chart_update',
    data: { sales: 1000, revenue: 50000 },
    timestamp: Time.current.iso8601
  }
)
```

```javascript
// Handle in Stimulus controller
handleMessage(event) {
  const { stream, data } = event.detail

  if (data.type === 'chart_update') {
    this.updateChart(data.data)
  }
}
```

## Limitations

### What TurboCable Does

- ✅ Server → Client broadcasts (Turbo Streams HTML)
- ✅ Server → Client JSON messages
- ✅ WebSocket connection management
- ✅ Stream subscriptions
- ✅ Automatic reconnection
- ✅ Ping/pong keepalive

### What TurboCable Doesn't Do

- ❌ Client → Server channel actions
- ❌ Bidirectional WebSocket communication
- ❌ Horizontal scaling (broadcasts within single process)

For client → server communication, use:
- Standard HTTP requests (fetch, forms)
- Turbo Frames and Streams
- Traditional REST endpoints

## When to Use TurboCable

**Perfect for:**
- ✅ Single-server applications
- ✅ Process-per-tenant multi-tenancy
- ✅ Server → client real-time updates
- ✅ Turbo Streams-based applications
- ✅ Development environments
- ✅ Memory-constrained deployments

**Not appropriate for:**
- ❌ Horizontally scaled apps with load balancing
- ❌ Multiple servers serving the same application
- ❌ Bidirectional WebSocket communication
- ❌ Real-time chat with client actions
- ❌ Collaborative editing requiring client messages

## Troubleshooting

### WebSocket Not Connecting

Check that the WebSocket endpoint is accessible:

```bash
# Test WebSocket upgrade
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: test" \
  http://localhost:3000/cable
```

Should return `101 Switching Protocols` if working.

### Broadcasts Not Received

1. Check that Rails app is broadcasting to correct URL:
   ```bash
   echo $TURBO_CABLE_BROADCAST_URL
   # Should be: http://localhost:3000/_broadcast
   ```

2. Check Navigator logs for broadcast endpoint hits:
   ```bash
   # Should see POST /_broadcast requests
   tail -f /var/log/navigator.log | grep broadcast
   ```

3. Verify subscription in browser console:
   ```javascript
   // Should see: Subscribed to stream_name
   ```

### Authentication Issues

If WebSocket connections are rejected:

1. Check if `/cable` is in `auth.public_paths`:
   ```yaml
   auth:
     public_paths:
       - "/cable"  # Add if WebSocket should be public
   ```

2. Or ensure users are authenticated before connecting

3. Check Navigator auth logs:
   ```bash
   tail -f /var/log/navigator.log | grep -E "(auth|cable)"
   ```

## Performance Considerations

### Connection Limits

TurboCable handles connections efficiently, but consider:

- **Memory**: ~1-2 KB per WebSocket connection
- **CPU**: Minimal (event-driven I/O)
- **Subscriptions**: Tracked in-memory hash maps

### Scaling

For high-traffic applications:

- **Single machine**: Handles thousands of concurrent connections
- **Process-per-tenant**: Each tenant isolated, no cross-process concerns
- **Multi-region**: Deploy Navigator in multiple Fly.io regions

### Monitoring

```bash
# Check WebSocket connections
netstat -an | grep :3000 | grep ESTABLISHED | wc -l

# Monitor Navigator memory
ps aux | grep navigator
```

## See Also

- [YAML Configuration Reference](../configuration/yaml-reference.md#cable)
- [WebSocket Support](websocket-support.md)
- [Action Cable Examples](../examples/action-cable.md)
- [Production Deployment](../deployment/production.md)
- [TurboCable Gem](https://github.com/rubys/turbo_cable)
