# Multi-Tenant Setup

Configure Navigator to serve multiple Rails applications with isolated databases and configurations.

## Use Cases

- SaaS with customer isolation
- Multiple applications on one server
- Different apps for different regions
- Development environments for multiple projects

## Path-Based Multi-Tenancy

Different applications on different URL paths:

```yaml title="navigator.yml"
server:
  listen: 3000
  public_dir: ./public

applications:
  # Shared environment variables
  global_env:
    RAILS_ENV: production
    RAILS_LOG_TO_STDOUT: "true"
    
  tenants:
    # Customer 1: Boston
    - name: boston
      path: /boston/
      working_dir: /var/www/app
      env:
        DATABASE_NAME: app_boston
        TENANT_NAME: "Boston Studio"
        STORAGE_PATH: /storage/boston
    
    # Customer 2: Chicago  
    - name: chicago
      path: /chicago/
      working_dir: /var/www/app
      env:
        DATABASE_NAME: app_chicago
        TENANT_NAME: "Chicago Studio"
        STORAGE_PATH: /storage/chicago
    
    # Customer 3: Dallas
    - name: dallas
      path: /dallas/
      working_dir: /var/www/app
      env:
        DATABASE_NAME: app_dallas
        TENANT_NAME: "Dallas Studio"
        STORAGE_PATH: /storage/dallas

# Serve static files for all tenants
static:
  directories:
    - path: /assets/
      root: /var/www/app/public/assets/
      cache: 86400
  extensions: [css, js, png, jpg, gif]

# Resource limits
pools:
  max_size: 15  # 5 processes per tenant max
  idle_timeout: 300
  start_port: 4000
```

## Database Isolation Pattern

Using template variables for cleaner configuration:

```yaml title="navigator.yml"
server:
  listen: 3000
  public_dir: /var/www/app/public

applications:
  # Template for all tenants
  env:
    RAILS_ENV: production
    DATABASE_URL: "postgresql://localhost/${database}"
    TENANT_ID: "${tenant_id}"
    TENANT_NAME: "${tenant_name}"
    STORAGE_ROOT: "/storage/${tenant_id}"
    
  tenants:
    - name: tenant-001
      path: /tenant/001/
      var:
        database: "app_tenant_001"
        tenant_id: "001"
        tenant_name: "Acme Corp"
    
    - name: tenant-002
      path: /tenant/002/
      var:
        database: "app_tenant_002"
        tenant_id: "002"
        tenant_name: "TechStart Inc"
    
    - name: tenant-003
      path: /tenant/003/
      var:
        database: "app_tenant_003"
        tenant_id: "003"
        tenant_name: "GlobalTrade LLC"
```

## Subdomain-Based Multi-Tenancy

Using a reverse proxy to route subdomains:

```yaml title="navigator.yml"
server:
  listen: 3000
  hostname: "*.myapp.com"
  public_dir: /var/www/app/public

applications:
  global_env:
    RAILS_ENV: production
    
  tenants:
    # Route based on Host header
    - name: boston
      path: /
      match_header: "Host: boston.myapp.com"
      working_dir: /var/www/app
      env:
        DATABASE_NAME: app_boston
        SUBDOMAIN: boston
    
    - name: chicago
      path: /
      match_header: "Host: chicago.myapp.com"
      working_dir: /var/www/app
      env:
        DATABASE_NAME: app_chicago
        SUBDOMAIN: chicago
```

## Year-Based Event System

Perfect for annual events or conferences:

```yaml title="navigator.yml"
server:
  listen: 3000
  public_dir: /var/www/showcase/public

applications:
  env:
    RAILS_ENV: production
    RAILS_APP_DB: "${database}"
    RAILS_APP_YEAR: "${year}"
    RAILS_APP_CITY: "${city}"
    EVENT_LOGO: "/logos/${year}-${city}.png"
    
  tenants:
    # 2024 Events
    - name: 2024-boston
      path: /2024/boston/
      var:
        database: "showcase_2024_boston"
        year: "2024"
        city: "boston"
    
    - name: 2024-seattle
      path: /2024/seattle/
      var:
        database: "showcase_2024_seattle"
        year: "2024"
        city: "seattle"
    
    # 2025 Events
    - name: 2025-boston
      path: /2025/boston/
      var:
        database: "showcase_2025_boston"
        year: "2025"
        city: "boston"
    
    - name: 2025-miami
      path: /2025/miami/
      var:
        database: "showcase_2025_miami"
        year: "2025"
        city: "miami"
```

## Mixed Application Types

Different applications with different requirements:

```yaml title="navigator.yml"
server:
  listen: 3000

applications:
  tenants:
    # Main application
    - name: main
      path: /
      working_dir: /var/www/main-app
      env:
        RAILS_ENV: production
        DATABASE_URL: postgresql://localhost/main
    
    # Admin panel (different codebase)
    - name: admin
      path: /admin/
      working_dir: /var/www/admin-app
      env:
        RAILS_ENV: production
        DATABASE_URL: postgresql://localhost/admin
      auth_realm: "Admin Area"  # Separate authentication
    
    # API (REST/GraphQL)
    - name: api
      path: /api/
      working_dir: /var/www/api-app
      env:
        RAILS_ENV: production
        DATABASE_URL: postgresql://localhost/api
        RAILS_API_ONLY: "true"
    
    # Legacy application
    - name: legacy
      path: /v1/
      working_dir: /var/www/legacy-app
      env:
        RAILS_ENV: production
        DATABASE_URL: mysql2://localhost/legacy

# Different auth for different apps
auth:
  enabled: true
  patterns:
    - path: /admin/
      htpasswd: /etc/navigator/admin.htpasswd
    - path: /api/
      action: "off"  # No auth for API
    - path: /
      htpasswd: /etc/navigator/users.htpasswd
```

## Database Setup

### PostgreSQL Setup

```bash
# Create databases for each tenant
for tenant in boston chicago dallas; do
  createdb "app_${tenant}"
  
  # Run migrations
  DATABASE_NAME="app_${tenant}" \
    RAILS_ENV=production \
    bundle exec rails db:migrate
    
  # Seed data (optional)
  DATABASE_NAME="app_${tenant}" \
    RAILS_ENV=production \
    bundle exec rails db:seed
done
```

### MySQL Setup

```sql
-- Create databases
CREATE DATABASE app_tenant_001;
CREATE DATABASE app_tenant_002;
CREATE DATABASE app_tenant_003;

-- Create users with limited permissions
CREATE USER 'tenant001'@'localhost' IDENTIFIED BY 'password';
GRANT ALL ON app_tenant_001.* TO 'tenant001'@'localhost';
```

## Tenant Management

### Adding a New Tenant

1. Create database:
```bash
createdb app_newtenant
```

2. Add to configuration:
```yaml
- name: newtenant
  path: /newtenant/
  var:
    database: "app_newtenant"
    tenant_name: "New Tenant"
```

3. Reload Navigator:
```bash
navigator -s reload
```

### Removing a Tenant

1. Remove from configuration
2. Reload Navigator
3. Archive/delete database:
```bash
pg_dump app_oldtenant > backup_oldtenant.sql
dropdb app_oldtenant
```

## Resource Management

### Per-Tenant Limits

```yaml
applications:
  tenants:
    # High-traffic tenant
    - name: premium
      path: /premium/
      max_processes: 5
      idle_timeout: 600
    
    # Low-traffic tenant
    - name: basic
      path: /basic/
      max_processes: 2
      idle_timeout: 120
```

### Shared Resources

```yaml
# Shared Redis for all tenants
managed_processes:
  - name: redis
    command: redis-server
    args: [--port, "6379"]

applications:
  global_env:
    REDIS_URL: redis://localhost:6379
    
  tenants:
    - name: tenant1
      path: /tenant1/
      env:
        REDIS_NAMESPACE: tenant1
```

## Monitoring

### Check Active Tenants

```bash
# See all Rails processes
ps aux | grep -E 'rails.*tenant'

# Check ports
netstat -tlnp | grep navigator

# Monitor logs
tail -f /var/log/navigator.log | grep tenant
```

### Per-Tenant Metrics

```bash
# Create monitoring script
cat > check_tenants.sh << 'EOF'
#!/bin/bash
for port in $(seq 4000 4020); do
  if nc -z localhost $port 2>/dev/null; then
    echo "Port $port: Active"
  fi
done
EOF
```

## Common Patterns

### Shared Codebase, Different Configs

All tenants use same code but different configurations:

```yaml
applications:
  tenants:
    - name: production
      path: /
      env:
        APP_MODE: production
        FEATURES: "basic,advanced"
    
    - name: staging
      path: /staging/
      env:
        APP_MODE: staging
        FEATURES: "basic,advanced,experimental"
    
    - name: demo
      path: /demo/
      env:
        APP_MODE: demo
        FEATURES: "basic"
        DEMO_RESET: "hourly"
```

### Regional Deployment

```yaml
routes:
  fly_replay:
    - path: "^/asia/"
      region: sin  # Singapore
    - path: "^/europe/"
      region: fra  # Frankfurt
    - path: "^/americas/"
      region: ord  # Chicago
```

## Troubleshooting

### Tenant Not Starting

```bash
# Check specific tenant logs
LOG_LEVEL=debug navigator navigator.yml 2>&1 | grep tenant-name

# Verify database connection
DATABASE_NAME=app_tenant rails console
```

### Cross-Tenant Data Leak

- Ensure DATABASE_NAME is properly set per tenant
- Use TENANT_ID in Rails for additional isolation
- Clear Rails cache between tenant requests

### Performance Issues

- Adjust `max_size` based on traffic patterns
- Use `min_instances` for frequently accessed tenants
- Consider separate servers for high-traffic tenants

## Next Steps

- Add [background jobs per tenant](with-sidekiq.md)
- Implement [tenant-specific authentication](../configuration/authentication.md)
- Set up [monitoring](../deployment/monitoring.md)
- Configure [regional routing](../features/fly-replay.md)