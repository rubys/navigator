# Fly.io Deployment

Deploy Navigator on Fly.io with machine suspension, regional distribution, and intelligent routing features.

## Quick Start

```bash
# 1. Install flyctl
curl -L https://fly.io/install.sh | sh

# 2. Login and create app
flyctl auth login
flyctl apps create myapp

# 3. Deploy Navigator
flyctl deploy
```

## Dockerfile

Navigator works excellently with Fly.io's machine architecture:

```dockerfile title="Dockerfile"
FROM ruby:3.2-slim

# Copy Navigator binary from Docker Hub image
COPY --from=samruby/navigator:latest /navigator /usr/local/bin/navigator
RUN chmod +x /usr/local/bin/navigator

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    libpq-dev \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

# Set up Rails application
WORKDIR /app
COPY Gemfile Gemfile.lock ./
RUN bundle install

COPY . .

# Precompile assets
RUN bundle exec rails assets:precompile

# Navigator configuration
COPY config/navigator-fly.yml /app/navigator.yml

EXPOSE 3000

# Start Navigator
CMD ["navigator", "/app/navigator.yml"]
```

## Fly.io Configuration

### Basic fly.toml

```toml title="fly.toml"
app = "myapp"
primary_region = "ord"

[build]
  dockerfile = "Dockerfile"

[env]
  RAILS_ENV = "production"
  RAILS_SERVE_STATIC_FILES = "false"

[http_service]
  internal_port = 3000
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0

  [http_service.checks.alive]
    path = "/up"
    interval = "30s"
    timeout = "5s"

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 512
```

### Production fly.toml

```toml title="fly.toml"
app = "myapp-production"
primary_region = "ord"

[build]
  dockerfile = "Dockerfile"

[env]
  RAILS_ENV = "production"
  RAILS_SERVE_STATIC_FILES = "false"
  LOG_LEVEL = "info"

[http_service]
  internal_port = 3000
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 1  # Keep at least one machine running

  [http_service.checks.alive]
    path = "/up"
    interval = "30s"
    timeout = "5s"
    grace_period = "10s"

  [http_service.checks.readiness]
    path = "/up"
    interval = "30s"
    timeout = "5s"

[[vm]]
  cpu_kind = "shared"
  cpus = 2
  memory_mb = 1024

[machines]
  auto_start = true
  auto_stop = true
  min_machines_running = 1

# Multi-region deployment
[[regions]]
  name = "ord"  # Chicago
  
[[regions]]  
  name = "fra"  # Frankfurt
  
[[regions]]
  name = "nrt"  # Tokyo
```

## Navigator Configuration for Fly.io

### Basic Navigator Config

```yaml title="config/navigator-fly.yml"
server:
  listen: 3000

  idle:
    action: suspend
    timeout: 10m

  static:
    public_dir: /app/public

applications:
  pools:
    max_size: 10
    idle_timeout: 5m
    start_port: 4000

  global_env:
    RAILS_ENV: production
    RAILS_SERVE_STATIC_FILES: "false"
    DATABASE_URL: "${DATABASE_URL}"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"

  tenants:
    - name: production
      path: /
      working_dir: /app
```

### Multi-Region Navigator Config

```yaml title="config/navigator-fly.yml"
server:
  listen: 3000

  # Aggressive suspension for cost optimization
  idle:
    action: suspend
    timeout: 15m

  static:
    public_dir: /app/public

routes:
  # Fly-Replay routing for optimal performance
  fly:
    replay:
      # Route PDF generation to specific region
      - path: "^/api/pdf/"
        region: ord
        status: 307

      # Route European users to Frankfurt
      - path: "^/eu/"
        region: fra
        status: 307

      # Route Asian users to Tokyo
      - path: "^/asia/"
        region: nrt
        status: 307

applications:
  pools:
    max_size: 15
    idle_timeout: 10m  # Longer timeout for production

  global_env:
    RAILS_ENV: production
    DATABASE_URL: "${DATABASE_URL}"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    REDIS_URL: "${REDIS_URL}"

  tenants:
    - name: production
      path: /
      working_dir: /app
```

## Environment Variables

### Required Secrets

```bash
# Set via flyctl
flyctl secrets set SECRET_KEY_BASE="your-rails-secret-key"
flyctl secrets set DATABASE_URL="postgresql://user:pass@host/db"

# Optional
flyctl secrets set REDIS_URL="redis://host:port"
flyctl secrets set NEW_RELIC_LICENSE_KEY="your-key"
```

### Fly.io-Specific Variables

```bash
# Set automatically by Fly.io
FLY_APP_NAME="myapp"
FLY_REGION="ord"
FLY_MACHINE_ID="unique-machine-id"

# Optional for machine control
flyctl secrets set FLY_API_TOKEN="your-fly-api-token"
```

## Database Setup

### PostgreSQL on Fly.io

```bash
# Create Fly PostgreSQL
flyctl postgres create myapp-db

# Connect to your app
flyctl postgres attach myapp-db

# This sets DATABASE_URL automatically
```

### External Database

```bash
# For external PostgreSQL
flyctl secrets set DATABASE_URL="postgresql://user:pass@external-db.com/myapp_production"
```

## Redis Setup

### Redis on Fly.io

```bash
# Create Fly Redis
flyctl redis create myapp-redis

# Connect to your app  
flyctl redis connect myapp-redis

# This sets REDIS_URL automatically
```

## Deployment Commands

### Initial Deployment

```bash
# Deploy to Fly.io
flyctl deploy

# Monitor deployment
flyctl logs

# Check app status
flyctl status
```

### Updates and Scaling

```bash
# Deploy updates
flyctl deploy

# Scale machines
flyctl scale count 3

# Scale to specific regions
flyctl scale count 2 --region ord
flyctl scale count 1 --region fra

# Update machine size
flyctl scale vm shared-cpu-2x --memory 1024
```

## Machine Management

### Machine Suspension

Navigator integrates seamlessly with Fly.io's machine suspension:

```yaml
# Enable in Navigator config
server:
  idle:
    action: suspend
    timeout: 10m

# Allow in Fly.io config
[machines]
  auto_stop = true
  min_machines_running = 0
```

**Benefits**:
- Machines suspend automatically during idle periods
- Wake in 1-3 seconds on incoming requests  
- Significant cost savings for variable traffic

### Multi-Machine Deployment

```bash
# Deploy to multiple machines
flyctl deploy --ha=false  # Single machine per region

# Or enable high availability  
flyctl scale count 2      # Multiple machines for redundancy
```

## Regional Distribution

### Geographic Routing

```yaml title="Navigator configuration"
routes:
  fly:
    replay:
      # North America -> Chicago
      - path: "^/na/"
        region: ord
        status: 307

      # Europe -> Frankfurt
      - path: "^/eu/"
        region: fra
        status: 307

      # Asia Pacific -> Tokyo
      - path: "^/apac/"
        region: nrt
        status: 307
```

### Regional Deployment

```bash
# Deploy to multiple regions
flyctl regions add ord fra nrt

# Check regional status
flyctl regions list

# Scale by region
flyctl scale count 2 --region ord  # Primary region
flyctl scale count 1 --region fra  # Secondary regions  
flyctl scale count 1 --region nrt
```

## Monitoring

### Fly.io Monitoring

```bash
# View logs
flyctl logs

# Monitor performance
flyctl metrics

# Check machine status
flyctl machine list

# SSH into machine
flyctl ssh console
```

### Navigator-Specific Monitoring

```bash
# Check Navigator processes
flyctl ssh console -C "ps aux | grep navigator"

# View Navigator logs
flyctl ssh console -C "journalctl -u navigator -n 50"

# Check Rails processes
flyctl ssh console -C "ps aux | grep rails"
```

## Performance Optimization

### Machine Sizing

```toml
# Development
[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 512

# Production
[[vm]]  
  cpu_kind = "shared"
  cpus = 2
  memory_mb = 1024

# High-performance
[[vm]]
  cpu_kind = "performance"
  cpus = 4
  memory_mb = 2048
```

### Process Pool Tuning

```yaml
applications:
  pools:
    # Adjust for machine size
    max_size: 8      # 512MB machine
    # max_size: 15   # 1GB machine
    # max_size: 25   # 2GB machine
    idle_timeout: 10m
```

## Cost Optimization

### Machine Suspension Strategy

```yaml
# Aggressive suspension (development)
server:
  idle:
    action: suspend
    timeout: 3m

# Balanced suspension (staging)
server:
  idle:
    action: suspend
    timeout: 10m

# Conservative suspension (production)
server:
  idle:
    action: suspend
    timeout: 30m
```

### Regional Cost Management

```bash
# Use cheaper regions for development
flyctl regions add ord  # Chicago (cheaper)
flyctl regions remove sjc  # San Jose (more expensive)

# Monitor costs
flyctl dashboard
```

## Troubleshooting

### Deployment Issues

```bash
# Check deployment status
flyctl status

# View deployment logs
flyctl logs

# Debug specific machine
flyctl machine list
flyctl logs --instance machine-id
```

### Machine Won't Start

```bash
# Check machine status
flyctl machine list

# Start machine manually
flyctl machine start machine-id

# Check machine configuration
flyctl machine show machine-id
```

### Rails App Issues

```bash
# SSH into machine
flyctl ssh console

# Check Navigator status
ps aux | grep navigator

# Check Rails processes  
ps aux | grep rails

# View Navigator logs
journalctl -u navigator -f
```

### Suspension Issues

```bash
# Check suspension configuration
flyctl ssh console -C "cat /app/navigator.yml | grep -A5 idle"

# Monitor suspension activity
flyctl logs | grep -E "(suspend|idle|activity)"

# Check machine auto-stop settings
flyctl config show | grep auto_stop
```

## Advanced Configuration

### Custom Health Checks

```toml title="fly.toml"
[http_service.checks.navigator]
  path = "/up"
  interval = "30s" 
  timeout = "5s"
  method = "GET"
  headers = {X-Health-Check = "fly"}
```

### Volume Mounts

```toml title="fly.toml"
[[mounts]]
  source = "myapp_data"
  destination = "/data"
  
# Create volume
# flyctl volumes create myapp_data --size 10
```

### Custom Entrypoint

```dockerfile
# Custom startup script
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["navigator", "/app/navigator.yml"]
```

```bash title="docker-entrypoint.sh"
#!/bin/bash
set -e

# Database migrations
bundle exec rails db:migrate

# Start Navigator
exec "$@"
```

## Integration Examples

### CI/CD with GitHub Actions

```yaml title=".github/workflows/deploy.yml"
name: Deploy to Fly.io

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: superfly/flyctl-actions/setup-flyctl@master
        
      - name: Deploy to Fly.io
        run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

### Multi-Environment Setup

```bash
# Production
flyctl apps create myapp-prod
flyctl deploy --app myapp-prod

# Staging
flyctl apps create myapp-staging  
flyctl deploy --app myapp-staging
```

## Best Practices

### 1. Use Machine Suspension

```yaml
server:
  idle:
    action: suspend
    timeout: 10m  # Adjust based on traffic patterns
```

### 2. Optimize for Regions

```bash
# Deploy close to users
flyctl regions add ord fra nrt  # Major regions
flyctl regions remove unused-regions
```

### 3. Right-Size Machines

```bash
# Start small, scale as needed
flyctl scale vm shared-cpu-1x --memory 512  # Start here
flyctl scale vm shared-cpu-2x --memory 1024 # Scale up
```

### 4. Monitor Costs

```bash
# Regular cost monitoring
flyctl dashboard
flyctl usage --app myapp
```

### 5. Security

```bash
# Use secrets for sensitive data
flyctl secrets set SECRET_KEY_BASE="..."
flyctl secrets set DATABASE_PASSWORD="..."

# Never hardcode secrets in fly.toml
```

## See Also

- [Production Deployment](production.md)
- [Machine Suspension](../configuration/suspend.md)
- [Fly-Replay Routing](../features/fly-replay.md)
- [Examples](../examples/fly-deployment.md)