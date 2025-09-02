# Rails with Sidekiq Background Jobs

Configure Navigator to manage both your Rails application and Sidekiq workers for background job processing.

## Use Case

- Background job processing
- Email sending
- Report generation
- Data imports/exports
- Scheduled tasks

## Complete Configuration

```yaml title="navigator.yml"
server:
  listen: 3000
  public_dir: ./public

# Managed processes start before Rails
managed_processes:
  # Redis server for Sidekiq
  - name: redis
    command: redis-server
    args: [--port, "6379", --appendonly, "yes"]
    working_dir: /var/lib/redis
    auto_restart: true
    start_delay: 0
    
  # Sidekiq worker process
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq, -C, config/sidekiq.yml]
    working_dir: /var/www/app
    env:
      RAILS_ENV: production
      REDIS_URL: redis://localhost:6379/0
    auto_restart: true
    start_delay: 2  # Wait for Redis

# Rails application
applications:
  global_env:
    RAILS_ENV: production
    REDIS_URL: redis://localhost:6379/0
    DATABASE_URL: postgresql://localhost/myapp
    
  tenants:
    - name: myapp
      path: /
      working_dir: /var/www/app

# Static files
static:
  directories:
    - path: /assets/
      root: public/assets/
      cache: 86400
  extensions: [css, js, png, jpg, gif]
```

## Rails Setup

### 1. Add Sidekiq to Gemfile

```ruby title="Gemfile"
gem 'sidekiq', '~> 7.0'
gem 'redis', '~> 5.0'
```

### 2. Configure Sidekiq

```ruby title="config/initializers/sidekiq.rb"
Sidekiq.configure_server do |config|
  config.redis = { url: ENV.fetch('REDIS_URL', 'redis://localhost:6379/0') }
  
  # Server middleware
  config.server_middleware do |chain|
    chain.add SidekiqMiddleware::ServerMetrics
  end
end

Sidekiq.configure_client do |config|
  config.redis = { url: ENV.fetch('REDIS_URL', 'redis://localhost:6379/0') }
end
```

### 3. Create Sidekiq Configuration

```yaml title="config/sidekiq.yml"
:concurrency: 5
:max_retries: 3
:timeout: 25

:queues:
  - [critical, 4]
  - [default, 2]
  - [low, 1]
  - [mailers, 2]

production:
  :concurrency: 10
  :logfile: ./log/sidekiq.log

development:
  :concurrency: 2
```

### 4. Add Sidekiq Web UI (Optional)

```ruby title="config/routes.rb"
require 'sidekiq/web'

Rails.application.routes.draw do
  # Protect admin routes
  authenticate :user, ->(u) { u.admin? } do
    mount Sidekiq::Web => '/sidekiq'
  end
  
  # Or with basic auth
  Sidekiq::Web.use Rack::Auth::Basic do |username, password|
    username == ENV['SIDEKIQ_USERNAME'] && 
    password == ENV['SIDEKIQ_PASSWORD']
  end
  mount Sidekiq::Web => '/sidekiq'
end
```

## Job Examples

### Basic Job

```ruby title="app/jobs/welcome_email_job.rb"
class WelcomeEmailJob < ApplicationJob
  queue_as :mailers
  
  def perform(user_id)
    user = User.find(user_id)
    UserMailer.welcome(user).deliver_now
  end
end

# Usage
WelcomeEmailJob.perform_later(user.id)
```

### Scheduled Job

```ruby title="app/jobs/daily_report_job.rb"
class DailyReportJob < ApplicationJob
  queue_as :low
  
  def perform
    Report.generate_daily
  end
end

# Schedule with sidekiq-cron or whenever
```

## Advanced Configurations

### Multiple Workers

Run different workers for different queues:

```yaml title="navigator.yml"
managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    
  # Critical queue worker
  - name: sidekiq-critical
    command: bundle
    args: [exec, sidekiq, -q, critical, -c, "5"]
    working_dir: /var/www/app
    env:
      RAILS_ENV: production
      REDIS_URL: redis://localhost:6379/0
    auto_restart: true
    start_delay: 2
    
  # Default queues worker
  - name: sidekiq-default
    command: bundle
    args: [exec, sidekiq, -q, default, -q, low, -c, "10"]
    working_dir: /var/www/app
    env:
      RAILS_ENV: production
      REDIS_URL: redis://localhost:6379/0
    auto_restart: true
    start_delay: 2
    
  # Mailer queue worker
  - name: sidekiq-mailers
    command: bundle
    args: [exec, sidekiq, -q, mailers, -c, "3"]
    working_dir: /var/www/app
    env:
      RAILS_ENV: production
      REDIS_URL: redis://localhost:6379/0
    auto_restart: true
    start_delay: 2
```

### With Redis Sentinel

For high availability:

```yaml title="navigator.yml"
managed_processes:
  # Redis Sentinel setup
  - name: redis-master
    command: redis-server
    args: [--port, "6379"]
    auto_restart: true
    
  - name: redis-sentinel
    command: redis-sentinel
    args: [/etc/redis/sentinel.conf]
    auto_restart: true
    start_delay: 1
    
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    env:
      REDIS_SENTINELS: "localhost:26379"
      REDIS_MASTER_NAME: "mymaster"
    auto_restart: true
    start_delay: 3
```

### Multi-Tenant Sidekiq

Separate workers per tenant:

```yaml title="navigator.yml"
managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    
  # Tenant 1 worker
  - name: sidekiq-tenant1
    command: bundle
    args: [exec, sidekiq, -C, config/sidekiq_tenant1.yml]
    env:
      TENANT_ID: tenant1
      REDIS_URL: redis://localhost:6379/1
    auto_restart: true
    
  # Tenant 2 worker
  - name: sidekiq-tenant2
    command: bundle
    args: [exec, sidekiq, -C, config/sidekiq_tenant2.yml]
    env:
      TENANT_ID: tenant2
      REDIS_URL: redis://localhost:6379/2
    auto_restart: true
```

## Monitoring

### Check Process Status

```bash
# View all processes
ps aux | grep -E '(redis|sidekiq|rails)'

# Check Sidekiq is processing
redis-cli -n 0 INFO | grep connected_clients

# Monitor Sidekiq logs
tail -f log/sidekiq.log
```

### Sidekiq Stats

```ruby
# Rails console
require 'sidekiq/api'

# Get statistics
stats = Sidekiq::Stats.new
puts "Processed: #{stats.processed}"
puts "Failed: #{stats.failed}"
puts "Busy: #{stats.workers_size}"
puts "Enqueued: #{stats.enqueued}"

# Check queues
queues = Sidekiq::Queue.all
queues.each do |queue|
  puts "#{queue.name}: #{queue.size} jobs"
end

# Retry queue
retries = Sidekiq::RetrySet.new
puts "Retries: #{retries.size}"
```

### Health Check Endpoint

```ruby title="app/controllers/health_controller.rb"
class HealthController < ApplicationController
  def sidekiq
    stats = Sidekiq::Stats.new
    
    if stats.workers_size > 0
      render json: { 
        status: 'healthy',
        workers: stats.workers_size,
        queued: stats.enqueued,
        processed: stats.processed
      }
    else
      render json: { status: 'unhealthy' }, status: :service_unavailable
    end
  end
end
```

## Troubleshooting

### Sidekiq Not Starting

```bash
# Check Redis is running
redis-cli ping

# Test Sidekiq manually
cd /var/www/app
bundle exec sidekiq

# Check for configuration errors
bundle exec sidekiq --help
```

### Jobs Not Processing

```ruby
# Check Redis connection
Rails.console
> Sidekiq.redis { |r| r.ping }

# Check queues
> Sidekiq::Queue.all.map { |q| [q.name, q.size] }

# Clear all jobs (careful!)
> Sidekiq::Queue.all.each(&:clear)
```

### Memory Issues

```yaml
# Limit Sidekiq memory usage
managed_processes:
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq, -c, "2"]  # Reduce concurrency
    env:
      MALLOC_ARENA_MAX: "2"  # Reduce memory fragmentation
```

### Stuck Jobs

```ruby
# Find and remove stuck jobs
workers = Sidekiq::Workers.new
workers.each do |process_id, thread_id, work|
  if work['run_at'] < 1.hour.ago.to_i
    # Job running for over an hour
    puts "Stuck job: #{work}"
  end
end
```

## Performance Tuning

### Optimize Concurrency

```yaml title="config/sidekiq.yml"
# Match concurrency to CPU cores
:concurrency: <%= ENV.fetch('SIDEKIQ_CONCURRENCY', 5) %>

# Database pool must be >= concurrency + 1
production:
  :concurrency: 10
```

### Database Pool

```yaml title="config/database.yml"
production:
  pool: <%= ENV.fetch("RAILS_MAX_THREADS", 5).to_i + 10 %>
  # Account for: web workers + sidekiq concurrency
```

### Redis Configuration

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args:
      - --maxmemory 256mb
      - --maxmemory-policy allkeys-lru
      - --save 900 1  # Persist every 15 min if 1+ changes
      - --save 300 10
      - --save 60 10000
```

## Security

### Separate Redis Databases

```yaml
applications:
  global_env:
    REDIS_URL: redis://localhost:6379/0  # Rails cache
    SIDEKIQ_REDIS_URL: redis://localhost:6379/1  # Sidekiq only
```

### Authentication

```yaml
managed_processes:
  - name: redis
    command: redis-server
    args:
      - --requirepass mysecretpassword
    
  - name: sidekiq
    env:
      REDIS_URL: redis://:mysecretpassword@localhost:6379/0
```

## Next Steps

- Add [scheduled jobs with sidekiq-cron](action-cable.md)
- Set up [monitoring and alerts](../deployment/monitoring.md)
- Configure [auto-scaling for workers](../features/process-management.md)
- Implement [job prioritization strategies](../configuration/processes.md)