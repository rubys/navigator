# Fly.io Deployment Example

Complete example of deploying a Rails application with Navigator on Fly.io, including machine suspension, multi-region setup, and production optimizations.

## Project Structure

```
myapp/
├── Dockerfile                    # Fly.io container
├── fly.toml                      # Fly.io configuration  
├── config/
│   ├── navigator.yml             # Navigator configuration
│   └── database.yml              # Rails database config
├── app/                          # Rails application
├── Gemfile
└── .github/
    └── workflows/
        └── deploy.yml            # CI/CD workflow
```

## Complete Example Files

### Dockerfile

```dockerfile
FROM ruby:3.2-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    libpq-dev \
    nodejs \
    npm \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Navigator
RUN curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz | \
    tar xzf - -C /usr/local/bin && \
    chmod +x /usr/local/bin/navigator

# Set up Rails app
WORKDIR /app
COPY Gemfile Gemfile.lock ./
RUN bundle install

# Copy application code
COPY . .

# Precompile assets
RUN bundle exec rails assets:precompile

# Create required directories
RUN mkdir -p tmp/pids log storage

# Expose port
EXPOSE 3000

# Start Navigator
CMD ["navigator", "/app/config/navigator.yml"]
```

### fly.toml

```toml
app = "myapp-production"
primary_region = "ord"
kill_signal = "SIGTERM"
kill_timeout = "60s"

[build]

[deploy]
  release_command = "bundle exec rails db:migrate"

[env]
  RAILS_ENV = "production"
  LOG_LEVEL = "info"

[http_service]
  internal_port = 3000
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 1
  processes = ["app"]

  [[http_service.checks]]
    grace_period = "10s"
    interval = "30s"
    method = "GET"
    path = "/up"
    port = 3000
    timeout = "5s"
    type = "http"

# VM configuration
[[vm]]
  cpu_kind = "shared"
  cpus = 2
  memory_mb = 1024

# Persistent volumes
[[mounts]]
  source = "myapp_storage"
  destination = "/app/storage"
  initial_size = "10GB"

# Secrets (set via flyctl secrets)
# DATABASE_URL
# SECRET_KEY_BASE  
# REDIS_URL
```

### Navigator Configuration

```yaml title="config/navigator.yml"
server:
  listen: 3000
  public_dir: /app/public

pools:
  max_size: 15             # Optimized for 1GB machine
  idle_timeout: 600        # 10 minutes
  start_port: 4000

# Machine suspension for cost optimization
suspend:
  enabled: true
  idle_timeout: 900        # 15 minutes - conservative for production
  check_interval: 60       # Check every minute
  grace_period: 120        # 2 minute grace period

# Serve static files directly
static:
  directories:
    - path: /assets/
      root: /app/public/assets/
      cache: 31536000       # 1 year for fingerprinted assets
    - path: /images/  
      root: /app/public/images/
      cache: 86400          # 1 day for images
    - path: /favicon.ico
      root: /app/public/favicon.ico
      cache: 86400
  extensions: [css, js, png, jpg, gif, ico, svg, woff, woff2, map]
  try_files:
    enabled: true
    suffixes: [".html", ".htm"]
    fallback: rails

# Fly-Replay for optimal regional routing
routes:
  fly_replay:
    # Route PDF generation to primary region (more CPU)
    - path: "^/api/pdf/"
      region: ord
      status: 307
      
    # Route image processing to primary region
    - path: "^/api/images/"  
      region: ord
      status: 307
      
    # Route European traffic to Frankfurt
    - path: "^/eu/"
      region: fra
      status: 307
      
    # Route Asian traffic to Tokyo
    - path: "^/asia/"
      region: nrt  
      status: 307

applications:
  global_env:
    RAILS_ENV: production
    RAILS_LOG_TO_STDOUT: "1"
    RAILS_SERVE_STATIC_FILES: "false"
    DATABASE_URL: "${DATABASE_URL}"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    REDIS_URL: "${REDIS_URL}"
    
    # Performance settings
    RAILS_MAX_THREADS: "15"
    WEB_CONCURRENCY: "1"
    
    # Feature flags
    RAILS_FORCE_SSL: "true"
    RAILS_LOG_LEVEL: "info"
    
  tenants:
    - name: production
      path: /
      working_dir: /app

# Managed processes for background jobs
managed_processes:
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /app
    env:
      RAILS_ENV: production
      SIDEKIQ_CONCURRENCY: "10"
    auto_restart: true
    start_delay: 5
```

## Step-by-Step Deployment

### 1. Prerequisites

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login to Fly.io
flyctl auth login

# Verify installation
flyctl version
```

### 2. Create Rails Application

```bash
# Create new Rails app (or use existing)
rails new myapp --database=postgresql
cd myapp

# Add Navigator configuration
mkdir -p config
# Copy navigator.yml from above
```

### 3. Set Up Fly.io App

```bash
# Create Fly app
flyctl apps create myapp-production

# Create PostgreSQL database
flyctl postgres create myapp-production-db --region ord

# Attach database
flyctl postgres attach myapp-production-db --app myapp-production

# Create Redis (optional)
flyctl redis create myapp-production-redis --region ord
flyctl redis connect myapp-production-redis --app myapp-production

# Create volume for storage
flyctl volumes create myapp_storage --region ord --size 10
```

### 4. Configure Secrets

```bash
# Generate and set Rails secret
SECRET_KEY=$(rails secret)
flyctl secrets set SECRET_KEY_BASE="$SECRET_KEY"

# Set other secrets
flyctl secrets set RAILS_MASTER_KEY="$(cat config/master.key)"

# Optional monitoring
flyctl secrets set NEW_RELIC_LICENSE_KEY="your-license-key"
flyctl secrets set HONEYBADGER_API_KEY="your-api-key"
```

### 5. Deploy Application

```bash
# Create Dockerfile and fly.toml (from examples above)

# Initial deployment
flyctl deploy

# Monitor deployment
flyctl logs

# Check status
flyctl status
```

### 6. Verify Deployment

```bash
# Check app is running
curl https://myapp-production.fly.dev/up

# Check Navigator status
flyctl ssh console -C "ps aux | grep navigator"

# Check Rails processes
flyctl ssh console -C "ps aux | grep rails"

# View logs
flyctl logs --app myapp-production
```

## Multi-Region Deployment

### Deploy to Multiple Regions

```bash
# Add regions
flyctl regions add ord fra nrt

# Scale per region
flyctl scale count 2 --region ord  # Primary region - 2 machines
flyctl scale count 1 --region fra  # Secondary regions - 1 machine each
flyctl scale count 1 --region nrt

# Check distribution
flyctl status
flyctl machine list
```

### Regional Configuration

```yaml
# Update navigator.yml for regional routing
routes:
  fly_replay:
    # Geographic routing examples
    - path: "^/americas/"
      region: ord
      status: 307
      
    - path: "^/europe/"  
      region: fra
      status: 307
      
    - path: "^/apac/"
      region: nrt
      status: 307
      
    # Load-balancing heavy operations
    - path: "^/api/heavy/"
      region: ord      # Route to primary region with more resources
      status: 307
```

## Production Optimizations

### Resource Optimization

```toml title="fly.toml - Production VM"
[[vm]]
  cpu_kind = "performance"  # Better performance than shared
  cpus = 4
  memory_mb = 2048
```

```yaml title="navigator.yml - Production pools"
pools:
  max_size: 25             # More processes for higher traffic
  idle_timeout: 1800       # 30 minutes - keep processes alive longer
  start_port: 4000
```

### Monitoring Setup

```yaml title="navigator.yml - Add monitoring"
applications:
  global_env:
    # Performance monitoring
    NEW_RELIC_LICENSE_KEY: "${NEW_RELIC_LICENSE_KEY}"
    HONEYBADGER_API_KEY: "${HONEYBADGER_API_KEY}"
    
    # Logging
    RAILS_LOG_LEVEL: "info"
    LOG_LEVEL: "info"
```

### Database Optimizations

```yaml title="navigator.yml - Database config"
applications:
  global_env:
    # Connection pool settings
    DATABASE_POOL: "25"
    DATABASE_TIMEOUT: "5000"
    
    # Performance settings
    RAILS_CACHE_STORE: "redis_cache_store"
    SESSION_STORE: "redis_session_store"
```

## CI/CD Integration

### GitHub Actions Workflow

```yaml title=".github/workflows/deploy.yml"
name: Deploy to Fly.io

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
          
    steps:
      - uses: actions/checkout@v4
      
      - uses: ruby/setup-ruby@v1
        with:
          bundler-cache: true
          
      - name: Setup test database
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/test
          RAILS_ENV: test
        run: |
          bundle exec rails db:create
          bundle exec rails db:migrate
          
      - name: Run tests
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/test
          RAILS_ENV: test
        run: bundle exec rails test

      - name: Validate Navigator config
        run: |
          curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz | tar xzf - 
          ./navigator --validate config/navigator.yml

  deploy:
    if: github.ref == 'refs/heads/main'
    needs: test
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: superfly/flyctl-actions/setup-flyctl@master
        
      - name: Deploy to production
        run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
          
      - name: Verify deployment
        run: |
          sleep 30  # Wait for deployment
          curl -f https://myapp-production.fly.dev/up
```

## Monitoring and Alerting

### Health Monitoring Script

```bash title="scripts/health-check.sh"
#!/bin/bash
# Health monitoring script

APP_NAME="myapp-production"
HEALTH_URL="https://${APP_NAME}.fly.dev/up"
SLACK_WEBHOOK="${SLACK_WEBHOOK_URL}"

check_health() {
    if ! curl -f --max-time 10 "$HEALTH_URL" > /dev/null 2>&1; then
        echo "Health check failed for $APP_NAME"
        
        # Send alert
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"🚨 Health check failed for $APP_NAME\"}" \
            "$SLACK_WEBHOOK"
            
        exit 1
    fi
    
    echo "Health check passed for $APP_NAME"
}

check_machines() {
    running_machines=$(flyctl machine list --app "$APP_NAME" --json | jq -r '.[] | select(.state == "started") | .id' | wc -l)
    
    if [ "$running_machines" -eq 0 ]; then
        echo "No running machines for $APP_NAME"
        exit 1
    fi
    
    echo "$running_machines machines running for $APP_NAME"
}

# Run checks
check_health
check_machines

echo "All checks passed for $APP_NAME"
```

### Automated Monitoring

```bash
# Set up cron job for monitoring
crontab -e

# Add line:
*/5 * * * * /path/to/scripts/health-check.sh
```

## Troubleshooting

### Common Deployment Issues

```bash
# Check deployment logs
flyctl logs --app myapp-production

# Check specific machine
flyctl machine list
flyctl logs --app myapp-production --instance machine-id

# SSH into machine for debugging
flyctl ssh console --app myapp-production

# Check Navigator status inside machine
ps aux | grep navigator
journalctl -u navigator -n 50
```

### Performance Issues

```bash
# Monitor resource usage
flyctl metrics --app myapp-production

# Check machine specs
flyctl machine show machine-id

# Scale up if needed  
flyctl scale vm performance-2x --memory 2048
flyctl scale count 3
```

### Database Issues

```bash
# Check database connection
flyctl postgres connect myapp-production-db

# Monitor database performance
flyctl postgres list
flyctl postgres show myapp-production-db
```

## Cost Management

### Optimize for Cost

```yaml
# Aggressive suspension for cost savings
suspend:
  enabled: true
  idle_timeout: 300      # 5 minutes - aggressive
  check_interval: 30     # Frequent checks
```

```toml
# Use shared CPU for cost savings
[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 512

[machines]
  min_machines_running = 0  # Allow full suspension
```

### Monitor Costs

```bash
# Check usage and billing
flyctl dashboard

# Monitor per-app costs
flyctl usage --app myapp-production

# Optimize regions (some are more expensive)
flyctl regions list
flyctl regions remove expensive-region
```

## Best Practices

### 1. Environment Separation

```bash
# Separate apps for different environments
flyctl apps create myapp-production   # Production
flyctl apps create myapp-staging      # Staging  
flyctl apps create myapp-review-pr123 # Review apps
```

### 2. Database Management

```bash
# Use separate databases
flyctl postgres create myapp-prod-db      # Production
flyctl postgres create myapp-staging-db   # Staging
```

### 3. Secrets Management

```bash
# Environment-specific secrets
flyctl secrets set --app myapp-production SECRET_KEY_BASE="prod-secret"
flyctl secrets set --app myapp-staging SECRET_KEY_BASE="staging-secret"
```

### 4. Monitoring

```bash
# Set up comprehensive monitoring
flyctl secrets set --app myapp-production NEW_RELIC_LICENSE_KEY="..."
flyctl secrets set --app myapp-production HONEYBADGER_API_KEY="..."
```

### 5. Scaling Strategy

```bash
# Start small, scale as needed
flyctl scale count 1           # Start with 1 machine
flyctl scale count 3           # Scale up for traffic
flyctl scale vm shared-cpu-2x  # Upgrade machine size
```

## See Also

- [Fly.io Deployment Guide](../deployment/fly-io.md)
- [Machine Suspension](../features/machine-suspend.md)  
- [Configuration Reference](../configuration/yaml-reference.md)
- [Production Deployment](../deployment/production.md)