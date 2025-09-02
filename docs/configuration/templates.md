# Variable Templates and Substitution

Navigator supports flexible variable substitution in YAML configuration, enabling efficient multi-tenant and multi-environment setups with shared templates and tenant-specific variables.

## How It Works

Variable substitution allows you to define templates in the `env` section and populate them with tenant-specific values from the `var` section:

```yaml
applications:
  # Template with variables
  env:
    DATABASE_URL: "postgresql://user:pass@localhost/${database}"
    STORAGE_PATH: "/storage/${tenant_id}"
    
  # Tenant-specific values
  tenants:
    - name: client-a
      var:
        database: "client_a_prod"
        tenant_id: "client_a"
```

**Result**: Client A gets `DATABASE_URL=postgresql://user:pass@localhost/client_a_prod`

## Variable Syntax

### Basic Substitution

```yaml
# Simple variable substitution
env:
  DATABASE_NAME: "${database}"
  TENANT_NAME: "${tenant}"
```

### Default Values

```yaml
# Use default if variable not defined
env:
  DATABASE_URL: "postgresql://localhost/${database:-default_db}"
  LOG_LEVEL: "${log_level:-info}"
  PORT: "${port:-3000}"
```

### Nested Variables

```yaml
# Combine multiple variables
env:
  FULL_DATABASE_URL: "postgresql://${db_user}:${db_pass}@${db_host}/${database}"
  STORAGE_PATH: "/storage/${environment}/${tenant_id}"
  LOG_FILE: "/var/log/${app_name}-${tenant_id}.log"
```

## Configuration Structure

### Complete Template Example

```yaml
applications:
  # Global environment (applied to all tenants)
  global_env:
    RAILS_ENV: production
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    
  # Template with variable placeholders
  env:
    # Database configuration
    DATABASE_URL: "postgresql://${db_user}:${db_pass}@${db_host}/${database}"
    DATABASE_POOL: "${db_pool:-25}"
    
    # Application settings
    RAILS_APP_NAME: "${app_name}"
    RAILS_APP_OWNER: "${owner}"
    RAILS_STORAGE: "${storage_path}"
    
    # Tenant-specific settings
    REDIS_NAMESPACE: "${tenant_prefix}"
    LOG_FILE: "logs/${tenant_id}.log"
    PIDFILE: "pids/${tenant_id}.pid"
    
    # Feature flags
    FEATURE_ANALYTICS: "${analytics_enabled:-false}"
    FEATURE_UPLOADS: "${uploads_enabled:-true}"
  
  # Individual tenant configurations
  tenants:
    - name: acme-corp
      path: /acme/
      working_dir: /var/www/app
      var:
        # Database variables
        database: "acme_production"
        db_user: "acme_user"
        db_pass: "acme_secure_password"
        db_host: "db-acme.internal.com"
        db_pool: "50"
        
        # Application variables
        app_name: "Acme Portal"
        owner: "Acme Corporation"
        tenant_id: "acme"
        tenant_prefix: "acme"
        storage_path: "/storage/acme"
        
        # Feature variables
        analytics_enabled: "true"
        uploads_enabled: "true"
    
    - name: widget-inc
      path: /widget/
      working_dir: /var/www/app
      var:
        # Different database setup
        database: "widget_prod"
        db_user: "widget_user" 
        db_pass: "widget_password"
        db_host: "localhost"
        # db_pool uses default (25)
        
        # Application variables
        app_name: "Widget Manager"
        owner: "Widget Inc"
        tenant_id: "widget"
        tenant_prefix: "widget"
        storage_path: "/storage/widget"
        
        # Different feature settings
        analytics_enabled: "false"
        # uploads_enabled uses default (true)
```

## Use Cases

### Multi-Tenant SaaS

Perfect for Software as a Service applications where each client needs their own database and configuration:

```yaml
applications:
  env:
    # Each tenant gets their own database
    DATABASE_URL: "postgresql://app_user:${DB_PASSWORD}@postgres.internal/${tenant_db}"
    
    # Separate Redis namespaces
    REDIS_URL: "redis://redis.internal:6379/0"
    REDIS_NAMESPACE: "app:${tenant_id}"
    
    # Tenant branding
    COMPANY_NAME: "${company_name}"
    COMPANY_LOGO: "${logo_filename}"
    THEME_COLOR: "${theme_color:-#007bff}"
    
    # Storage separation
    UPLOAD_PATH: "/storage/uploads/${tenant_id}"
    BACKUP_PATH: "/backup/${tenant_id}"
  
  tenants:
    - name: startup-alpha
      var:
        tenant_db: "startup_alpha_prod"
        tenant_id: "startup_alpha"
        company_name: "Startup Alpha Inc"
        logo_filename: "startup-alpha-logo.png"
        theme_color: "#ff6b6b"
        
    - name: enterprise-beta
      var:
        tenant_db: "enterprise_beta_prod"
        tenant_id: "enterprise_beta"
        company_name: "Enterprise Beta Corp"
        logo_filename: "enterprise-beta-logo.svg"
        theme_color: "#4ecdc4"
```

### Multi-Environment Deployment

Use the same configuration template across different environments:

```yaml
applications:
  env:
    DATABASE_URL: "postgresql://${db_user}:${db_pass}@${db_host}/${database}"
    REDIS_URL: "redis://${redis_host}:${redis_port}/0"
    S3_BUCKET: "${s3_bucket}"
    LOG_LEVEL: "${log_level:-info}"
    WORKER_PROCESSES: "${worker_count:-2}"
  
  tenants:
    - name: development
      path: /
      var:
        database: "myapp_development"
        db_user: "dev_user"
        db_pass: "dev_password"
        db_host: "localhost"
        redis_host: "localhost"
        redis_port: "6379"
        s3_bucket: "myapp-dev-uploads"
        log_level: "debug"
        worker_count: "1"
        
    - name: staging
      path: /
      var:
        database: "myapp_staging"
        db_user: "staging_user"
        db_pass: "${STAGING_DB_PASSWORD}"
        db_host: "staging-db.internal"
        redis_host: "staging-redis.internal"
        redis_port: "6379"
        s3_bucket: "myapp-staging-uploads"
        log_level: "info"
        worker_count: "3"
        
    - name: production
      path: /
      var:
        database: "myapp_production"
        db_user: "prod_user"
        db_pass: "${PRODUCTION_DB_PASSWORD}"
        db_host: "prod-db.internal"
        redis_host: "prod-redis.internal"
        redis_port: "6379"
        s3_bucket: "myapp-prod-uploads"
        log_level: "warn"
        worker_count: "10"
```

### Feature Flag Management

Control features per tenant using variables:

```yaml
applications:
  env:
    # Feature flags
    FEATURE_ADVANCED_ANALYTICS: "${analytics:-false}"
    FEATURE_API_ACCESS: "${api_access:-false}"
    FEATURE_CUSTOM_BRANDING: "${custom_branding:-false}"
    FEATURE_BULK_IMPORT: "${bulk_import:-false}"
    
    # Limits based on plan
    MAX_USERS: "${max_users:-10}"
    MAX_STORAGE_MB: "${max_storage:-1000}"
    API_RATE_LIMIT: "${api_rate_limit:-100}"
  
  tenants:
    - name: basic-plan-client
      var:
        analytics: "false"
        api_access: "false"
        custom_branding: "false"
        bulk_import: "false"
        max_users: "10"
        max_storage: "1000"
        api_rate_limit: "100"
        
    - name: premium-plan-client
      var:
        analytics: "true"
        api_access: "true"
        custom_branding: "true"
        bulk_import: "true"
        max_users: "100"
        max_storage: "10000"
        api_rate_limit: "1000"
```

## Advanced Patterns

### Conditional Configuration

```yaml
applications:
  env:
    # Use different services based on environment
    CACHE_URL: "${cache_type:-redis}://localhost:6379"
    SEARCH_URL: "${search_enabled:-false}"
    
    # Conditional database settings
    DATABASE_URL: "postgresql://user:pass@${db_host:-localhost}/${database}"
    DATABASE_SSL: "${db_ssl:-false}"
    
  tenants:
    - name: dev-local
      var:
        database: "myapp_dev"
        cache_type: "memory"
        search_enabled: "false"
        
    - name: production
      var:
        database: "myapp_prod"
        db_host: "prod-db.amazonaws.com"
        db_ssl: "true"
        cache_type: "redis"
        search_enabled: "elasticsearch://search.internal:9200"
```

### Secrets Management Integration

```yaml
applications:
  env:
    # Reference secrets from environment
    DATABASE_PASSWORD: "${DB_PASSWORD}"
    API_SECRET_KEY: "${API_SECRET_KEY}"
    ENCRYPTION_KEY: "${ENCRYPTION_KEY}"
    
    # Combine secrets with tenant info
    S3_BUCKET: "myapp-${environment}-${tenant_id}"
    
  tenants:
    - name: client-a
      var:
        tenant_id: "client_a"
        environment: "prod"
```

**Set secrets via environment**:
```bash
export DB_PASSWORD="secure-database-password"
export API_SECRET_KEY="api-secret-key-here"
export ENCRYPTION_KEY="encryption-key-here"
```

### Dynamic Service Discovery

```yaml
applications:
  env:
    # Service URLs based on environment
    DATABASE_URL: "postgresql://user:pass@${db_service}.${region}.internal/${database}"
    REDIS_URL: "redis://${redis_service}.${region}.internal:6379"
    S3_ENDPOINT: "https://s3.${region}.amazonaws.com"
    
  tenants:
    - name: us-east-client
      var:
        database: "client_prod"
        region: "us-east-1"
        db_service: "postgres-primary"
        redis_service: "redis-cluster"
        
    - name: eu-west-client
      var:
        database: "client_prod"
        region: "eu-west-1"
        db_service: "postgres-primary"
        redis_service: "redis-cluster"
```

## Special Tenant Types

### Special Tenants (No Variable Substitution)

Some tenants need fixed configuration without templates:

```yaml
applications:
  env:
    # Template applies to regular tenants
    DATABASE_URL: "postgresql://localhost/${database}"
    TENANT_ID: "${tenant_id}"
    
  tenants:
    - name: main-app
      path: /
      var:
        database: "main_prod"
        tenant_id: "main"
        
    # Special tenant - uses direct env, skips template
    - name: monitoring
      path: /monitoring/
      special: true
      env:
        DATABASE_URL: "postgresql://localhost/monitoring_db"
        MONITORING_MODE: "true"
        LOG_LEVEL: "debug"
        
    - name: health-check
      path: /health/
      special: true
      env:
        HEALTH_CHECK_ONLY: "true"
```

## Variable Validation

### Required Variables

Ensure critical variables are defined:

```yaml
# Navigator will fail to start if required variables are missing
applications:
  env:
    DATABASE_URL: "postgresql://localhost/${database}"  # Required
    SECRET_KEY: "${secret_key}"                         # Required
    OPTIONAL_FEATURE: "${optional_feature:-disabled}"  # Has default
    
  tenants:
    - name: client-a
      var:
        database: "client_a_prod"
        secret_key: "client-a-secret"
        # optional_feature will use default "disabled"
```

### Validation Script

```bash
#!/bin/bash
# validate-config.sh - Check if all required variables are defined

CONFIG_FILE="$1"
REQUIRED_VARS=("database" "secret_key" "tenant_id")

# Extract tenant configurations
for tenant in $(yq eval '.applications.tenants[].name' "$CONFIG_FILE"); do
    echo "Validating tenant: $tenant"
    
    for var in "${REQUIRED_VARS[@]}"; do
        if ! yq eval ".applications.tenants[] | select(.name == \"$tenant\") | .var.$var" "$CONFIG_FILE" | grep -q .; then
            echo "ERROR: Required variable '$var' missing for tenant '$tenant'"
            exit 1
        fi
    done
done

echo "All required variables present"
```

## Testing Templates

### Template Resolution Testing

```bash
#!/bin/bash
# Test variable substitution

# Create test config
cat > test-config.yml << 'EOF'
applications:
  env:
    DATABASE_URL: "postgresql://localhost/${database}"
    TENANT_NAME: "${name}"
    LOG_LEVEL: "${log_level:-info}"
    
  tenants:
    - name: test-tenant
      var:
        database: "test_db"
        name: "Test Tenant"
        # log_level should use default
EOF

# Test Navigator configuration parsing
navigator --validate test-config.yml

# Test with debug output
LOG_LEVEL=debug navigator --validate test-config.yml 2>&1 | grep -E "(DATABASE_URL|TENANT_NAME|LOG_LEVEL)"
```

### Environment Variable Testing

```bash
# Test template resolution in Rails
cd /var/www/app
RAILS_ENV=production bundle exec rails runner "
  puts 'DATABASE_URL: ' + ENV['DATABASE_URL'].inspect
  puts 'TENANT_NAME: ' + ENV['TENANT_NAME'].inspect
  puts 'LOG_LEVEL: ' + ENV['LOG_LEVEL'].inspect
"
```

## Best Practices

### 1. Template Organization

```yaml
# Good - organized by function
env:
  # Database settings
  DATABASE_URL: "postgresql://${db_user}:${db_pass}@${db_host}/${database}"
  DATABASE_POOL: "${db_pool:-25}"
  
  # Storage settings
  STORAGE_PATH: "/storage/${tenant_id}"
  BACKUP_PATH: "/backup/${tenant_id}"
  
  # Feature flags
  FEATURE_X: "${feature_x:-false}"
  FEATURE_Y: "${feature_y:-false}"

# Avoid - mixed concerns
env:
  DATABASE_URL: "postgresql://localhost/${database}"
  FEATURE_X: "${feature_x:-false}"
  STORAGE_PATH: "/storage/${tenant_id}"
  DATABASE_POOL: "${db_pool:-25}"
```

### 2. Default Values

```yaml
# Good - provide sensible defaults
env:
  LOG_LEVEL: "${log_level:-info}"
  MAX_CONNECTIONS: "${max_connections:-100}"
  TIMEOUT: "${timeout:-30}"

# Bad - no defaults for optional settings
env:
  LOG_LEVEL: "${log_level}"
  MAX_CONNECTIONS: "${max_connections}"
```

### 3. Variable Naming

```yaml
# Good - clear, consistent naming
var:
  tenant_id: "client_a"
  database_name: "client_a_prod"
  storage_path: "/storage/client_a"
  feature_analytics: "true"

# Avoid - inconsistent naming
var:
  id: "client_a"
  db: "client_a_prod"
  storage: "/storage/client_a"
  analytics: "true"
```

### 4. Security Considerations

```yaml
# Good - reference environment secrets
env:
  DATABASE_PASSWORD: "${DB_PASSWORD}"
  API_SECRET: "${API_SECRET}"

# Never hardcode secrets in templates
env:
  DATABASE_PASSWORD: "hardcoded-password"  # NEVER!
```

## Troubleshooting

### Variable Not Substituted

**Issue**: Variables appear as literal `${var}` in environment

**Causes**:
- Variable not defined in tenant's `var` section
- Typo in variable name
- Missing tenant configuration

**Solutions**:
```bash
# Check Navigator debug logs
LOG_LEVEL=debug navigator config.yml | grep "variable substitution"

# Validate tenant configuration
yq eval '.applications.tenants[] | select(.name == "my-tenant") | .var' config.yml

# Test with simple config
```

### Missing Required Variables

**Issue**: Navigator fails to start with variable errors

**Solution**:
```yaml
# Add missing variables to tenant
tenants:
  - name: my-tenant
    var:
      database: "required_db_name"
      secret_key: "required_secret"
```

### Template Resolution Order

Templates are resolved in this order:
1. `global_env` (applied to all tenants)
2. `env` template with variable substitution
3. Tenant-specific `env` (overrides template)

```yaml
applications:
  global_env:
    RAILS_ENV: production     # Applied to all
    
  env:
    DATABASE_URL: "postgresql://localhost/${database}"  # Template
    
  tenants:
    - name: special-tenant
      env:
        DATABASE_URL: "postgresql://special-host/special_db"  # Override
```

## See Also

- [Applications Configuration](applications.md)
- [YAML Reference](yaml-reference.md)
- [Multi-Tenant Example](../examples/multi-tenant.md)
- [Environment Variables](../reference/environment.md)