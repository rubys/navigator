# Rails with Redis

Configure Navigator to manage both your Rails application and Redis server for caching, sessions, and data storage.

## Use Case

- Rails application caching
- Session storage
- Real-time features
- Background job queuing
- Data caching and storage

## Complete Configuration

```yaml title="navigator.yml"
server:
  listen: 3000
  public_dir: ./public

# Redis process managed by Navigator
managed_processes:
  - name: redis
    command: redis-server
    args: 
      - --port 6379
      - --appendonly yes
      - --save 900 1
      - --save 300 10
      - --save 60 10000
      - --maxmemory 256mb
      - --maxmemory-policy allkeys-lru
    working_dir: /var/lib/redis
    auto_restart: true
    start_delay: 0

# Static files
static:
  directories:
    - path: /assets/
      root: public/assets/
      cache: 86400
  extensions: [css, js, png, jpg, gif]

# Rails application with Redis configuration
applications:
  global_env:
    RAILS_ENV: production
    REDIS_URL: redis://localhost:6379/0
    CACHE_REDIS_URL: redis://localhost:6379/1
    SESSION_REDIS_URL: redis://localhost:6379/2
    
  tenants:
    - name: myapp
      path: /
      working_dir: /var/www/app
```

## Rails Configuration

### 1. Add Redis to Gemfile

```ruby title="Gemfile"
gem 'redis', '~> 5.0'
gem 'redis-rails', '~> 5.0'  # For Rails integration
gem 'hiredis', '~> 0.6'      # Optional: faster Redis driver
```

### 2. Configure Redis Connection

```ruby title="config/initializers/redis.rb"
# Redis connection configuration
redis_config = {
  url: ENV.fetch('REDIS_URL', 'redis://localhost:6379/0'),
  reconnect_attempts: 3,
  reconnect_delay: 0.1,
  reconnect_delay_max: 0.5,
  timeout: 1
}

# Create Redis connection
$redis = Redis.new(redis_config)

# Test connection on startup
begin
  $redis.ping
  Rails.logger.info "Redis connected successfully"
rescue Redis::CannotConnectError => e
  Rails.logger.error "Redis connection failed: #{e.message}"
end
```

### 3. Configure Rails Cache

```ruby title="config/environments/production.rb"
Rails.application.configure do
  # Use Redis for caching
  config.cache_store = :redis_cache_store, {
    url: ENV.fetch('CACHE_REDIS_URL', 'redis://localhost:6379/1'),
    pool_size: 5,
    pool_timeout: 5,
    reconnect_attempts: 3,
    
    # Cache options
    expires_in: 1.hour,
    compress: true,
    compress_threshold: 1.kilobyte,
    namespace: Rails.application.class.module_parent_name.downcase
  }
  
  # Other production settings...
end
```

### 4. Configure Session Store

```ruby title="config/initializers/session_store.rb"
Rails.application.config.session_store :redis_store,
  servers: [ENV.fetch('SESSION_REDIS_URL', 'redis://localhost:6379/2')],
  expire_after: 2.weeks,
  key: "_#{Rails.application.class.module_parent_name.downcase}_session",
  secure: Rails.env.production?,
  same_site: :lax,
  httponly: true
```

## Redis Usage Examples

### Basic Caching

```ruby
class ProductsController < ApplicationController
  def index
    @products = Rails.cache.fetch("products", expires_in: 1.hour) do
      Product.includes(:category).published.order(:name)
    end
  end
  
  def show
    @product = Rails.cache.fetch("product_#{params[:id]}", expires_in: 30.minutes) do
      Product.find(params[:id])
    end
  end
end
```

### Custom Redis Operations

```ruby
class StatsService
  def self.increment_page_view(page)
    $redis.incr("page_views:#{page}")
  end
  
  def self.get_page_views(page)
    $redis.get("page_views:#{page}").to_i
  end
  
  def self.set_user_online(user_id)
    $redis.setex("user_online:#{user_id}", 5.minutes, "1")
  end
  
  def self.user_online?(user_id)
    $redis.exists?("user_online:#{user_id}")
  end
end
```

### Real-time Features

```ruby
class NotificationService
  def self.publish(channel, message)
    $redis.publish(channel, message.to_json)
  end
  
  def self.subscribe(channel, &block)
    $redis.subscribe(channel) do |on|
      on.message do |channel, message|
        data = JSON.parse(message)
        block.call(data)
      end
    end
  end
end

# Usage
NotificationService.publish("user_#{user_id}", {
  type: 'message',
  content: 'New message received'
})
```

## Advanced Configurations

### Redis Cluster Setup

```yaml title="navigator.yml"
managed_processes:
  # Redis cluster nodes
  - name: redis-node-1
    command: redis-server
    args: 
      - --port 7001
      - --cluster-enabled yes
      - --cluster-config-file nodes-7001.conf
      - --cluster-node-timeout 5000
    working_dir: /var/lib/redis/node1
    auto_restart: true
    
  - name: redis-node-2
    command: redis-server
    args: 
      - --port 7002
      - --cluster-enabled yes
      - --cluster-config-file nodes-7002.conf
      - --cluster-node-timeout 5000
    working_dir: /var/lib/redis/node2
    auto_restart: true
    
  - name: redis-node-3
    command: redis-server
    args:
      - --port 7003
      - --cluster-enabled yes
      - --cluster-config-file nodes-7003.conf
      - --cluster-node-timeout 5000
    working_dir: /var/lib/redis/node3
    auto_restart: true

applications:
  global_env:
    REDIS_URL: redis://localhost:7001,localhost:7002,localhost:7003
```

### Redis with Sentinel

```yaml title="navigator.yml"
managed_processes:
  # Redis master
  - name: redis-master
    command: redis-server
    args: [--port, "6379"]
    working_dir: /var/lib/redis/master
    auto_restart: true
    
  # Redis slaves
  - name: redis-slave-1
    command: redis-server
    args: [--port, "6380", --replicaof, localhost, "6379"]
    working_dir: /var/lib/redis/slave1
    auto_restart: true
    start_delay: 2
    
  # Sentinel processes
  - name: sentinel-1
    command: redis-sentinel
    args: [/etc/redis/sentinel1.conf]
    auto_restart: true
    start_delay: 5
    
  - name: sentinel-2
    command: redis-sentinel
    args: [/etc/redis/sentinel2.conf]
    auto_restart: true
    start_delay: 5

applications:
  global_env:
    REDIS_SENTINELS: "localhost:26379,localhost:26380"
    REDIS_MASTER_NAME: "mymaster"
```

### Multi-Database Configuration

```yaml title="navigator.yml"
applications:
  global_env:
    # Different Redis databases for different purposes
    CACHE_REDIS_URL: redis://localhost:6379/0       # Rails cache
    SESSION_REDIS_URL: redis://localhost:6379/1     # Sessions
    SIDEKIQ_REDIS_URL: redis://localhost:6379/2     # Background jobs
    REALTIME_REDIS_URL: redis://localhost:6379/3    # Real-time features
    ANALYTICS_REDIS_URL: redis://localhost:6379/4   # Analytics data
```

## Performance Optimization

### Redis Configuration Tuning

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args:
      # Memory management
      - --maxmemory 2gb
      - --maxmemory-policy allkeys-lru
      
      # Persistence
      - --save 900 1      # Save if at least 1 key changed in 900 seconds
      - --save 300 10     # Save if at least 10 keys changed in 300 seconds
      - --save 60 10000   # Save if at least 10000 keys changed in 60 seconds
      - --appendonly yes  # Enable AOF
      - --appendfsync everysec
      
      # Performance
      - --tcp-keepalive 60
      - --timeout 0
      - --tcp-backlog 511
      
      # Logging
      - --loglevel notice
      - --logfile /var/log/redis/redis.log
    working_dir: /var/lib/redis
```

### Connection Pool Configuration

```ruby title="config/initializers/redis.rb"
# Configure connection pool
redis_pool = ConnectionPool.new(size: 10, timeout: 5) do
  Redis.new(
    url: ENV.fetch('REDIS_URL', 'redis://localhost:6379/0'),
    timeout: 1,
    reconnect_attempts: 3
  )
end

# Use connection pool
class CacheService
  def self.get(key)
    redis_pool.with { |redis| redis.get(key) }
  end
  
  def self.set(key, value, expires_in: 1.hour)
    redis_pool.with { |redis| redis.setex(key, expires_in, value) }
  end
end
```

## Security Configuration

### Redis Authentication

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args:
      - --requirepass "${REDIS_PASSWORD}"
      - --protected-mode yes
      - --bind 127.0.0.1  # Only local connections
    env:
      REDIS_PASSWORD: "${REDIS_PASSWORD}"

applications:
  global_env:
    REDIS_URL: "redis://:${REDIS_PASSWORD}@localhost:6379/0"
```

### TLS Configuration

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args:
      - --tls-port 6380
      - --port 0  # Disable non-TLS port
      - --tls-cert-file /etc/redis/redis.crt
      - --tls-key-file /etc/redis/redis.key
      - --tls-ca-cert-file /etc/redis/ca.crt

applications:
  global_env:
    REDIS_URL: "rediss://localhost:6380/0"  # rediss:// for TLS
```

## Monitoring and Debugging

### Redis Health Checks

```ruby title="lib/redis_health_check.rb"
class RedisHealthCheck
  def self.healthy?
    $redis.ping == "PONG"
  rescue Redis::BaseError
    false
  end
  
  def self.info
    $redis.info
  rescue Redis::BaseError
    {}
  end
  
  def self.memory_usage
    info = $redis.info('memory')
    {
      used_memory: info['used_memory_human'],
      used_memory_peak: info['used_memory_peak_human'],
      fragmentation_ratio: info['mem_fragmentation_ratio']
    }
  end
end
```

### Monitoring Endpoint

```ruby title="config/routes.rb"
Rails.application.routes.draw do
  get '/health/redis', to: 'health#redis'
end
```

```ruby title="app/controllers/health_controller.rb"
class HealthController < ApplicationController
  def redis
    if RedisHealthCheck.healthy?
      render json: { 
        status: 'healthy',
        memory: RedisHealthCheck.memory_usage 
      }
    else
      render json: { status: 'unhealthy' }, status: :service_unavailable
    end
  end
end
```

## Testing

### Test Redis Connection

```bash
# Test Redis is running
redis-cli ping

# Check Redis info
redis-cli info server

# Monitor Redis commands
redis-cli monitor

# Check memory usage
redis-cli info memory
```

### Rails Integration Test

```ruby
# Test caching
Rails.cache.write('test_key', 'test_value')
Rails.cache.read('test_key')  # Should return 'test_value'

# Test custom Redis operations
$redis.set('custom_key', 'custom_value')
$redis.get('custom_key')  # Should return 'custom_value'
```

## Troubleshooting

### Redis Connection Issues

1. **Check Redis is running**:
   ```bash
   ps aux | grep redis-server
   netstat -tlnp | grep 6379
   ```

2. **Test connection**:
   ```bash
   redis-cli ping
   # Should return PONG
   ```

3. **Check Rails connection**:
   ```ruby
   rails console
   > $redis.ping
   # Should return "PONG"
   ```

### Memory Issues

1. **Check memory usage**:
   ```bash
   redis-cli info memory
   ```

2. **Configure memory limits**:
   ```yaml
   managed_processes:
     - name: redis
       command: redis-server
       args: [--maxmemory, 512mb, --maxmemory-policy, allkeys-lru]
   ```

### Performance Issues

1. **Monitor slow queries**:
   ```bash
   redis-cli config set slowlog-log-slower-than 10000
   redis-cli slowlog get 10
   ```

2. **Check connection pool**:
   ```ruby
   # In Rails console
   Rails.cache.redis.connection_pool.size
   Rails.cache.redis.connection_pool.available
   ```

## Production Considerations

### Persistence Strategy

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args:
      # RDB + AOF for maximum durability
      - --save 900 1
      - --appendonly yes
      - --appendfsync everysec
      - --auto-aof-rewrite-percentage 100
      - --auto-aof-rewrite-min-size 64mb
```

### Backup Strategy

```bash
#!/bin/bash
# Redis backup script
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/redis"

# Create backup directory
mkdir -p $BACKUP_DIR

# Copy RDB file
cp /var/lib/redis/dump.rdb $BACKUP_DIR/dump_$DATE.rdb

# Copy AOF file
cp /var/lib/redis/appendonly.aof $BACKUP_DIR/appendonly_$DATE.aof

# Compress backups older than 1 day
find $BACKUP_DIR -name "*.rdb" -mtime +1 -exec gzip {} \;
find $BACKUP_DIR -name "*.aof" -mtime +1 -exec gzip {} \;

# Remove backups older than 30 days
find $BACKUP_DIR -name "*.gz" -mtime +30 -delete
```

### Resource Limits

```bash
# Set system limits before starting Navigator
ulimit -n 65536  # File descriptors
echo 'vm.overcommit_memory = 1' >> /etc/sysctl.conf
echo 'net.core.somaxconn = 65535' >> /etc/sysctl.conf
sysctl -p
```

## See Also

- [Sidekiq Integration](with-sidekiq.md)
- [Managed Processes](../configuration/processes.md)
- [Multi-Tenant Setup](multi-tenant.md)
- [Performance Optimization](../features/process-management.md)