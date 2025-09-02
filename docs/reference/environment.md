# Environment Variables

Navigator recognizes environment variables for configuration, behavior control, and integration with hosting platforms.

## Navigator-Specific Variables

### LOG_LEVEL
Controls logging verbosity level.

**Values**: `debug`, `info`, `warn`, `error`  
**Default**: `info`

```bash
# Debug logging (very verbose)
LOG_LEVEL=debug navigator config.yml

# Only errors and warnings
LOG_LEVEL=warn navigator config.yml

# Production logging (recommended)
LOG_LEVEL=info navigator config.yml
```

**Log Output Examples**:

=== "debug"
    ```
    DEBUG Request received path=/api/users method=GET
    DEBUG Route matched app=main pattern=/
    DEBUG Starting process app=main port=4001
    INFO Process started app=main pid=12345
    DEBUG Proxying request to localhost:4001
    ```

=== "info"
    ```
    INFO Starting Navigator listen=3000
    INFO Process started app=main pid=12345
    INFO Configuration reloaded
    ```

=== "warn"
    ```
    WARN Process idle timeout app=main
    WARN Configuration file not found, using defaults
    ```

=== "error"
    ```
    ERROR Failed to start process app=main error="port unavailable"
    ERROR Configuration invalid: missing required field
    ```

### NAVIGATOR_PID_FILE
Location for Navigator's PID file.

**Default**: `/tmp/navigator.pid`

```bash
# Custom PID file location
NAVIGATOR_PID_FILE=/var/run/navigator.pid navigator config.yml

# In systemd service
Environment=NAVIGATOR_PID_FILE=/var/run/navigator/navigator.pid
```

### NAVIGATOR_CONFIG
Default configuration file path.

**Default**: Searches `config/navigator.yml`, then `navigator.yml`

```bash
# Set default config location
NAVIGATOR_CONFIG=/etc/navigator/production.yml navigator

# Override with command line
NAVIGATOR_CONFIG=/etc/navigator/production.yml navigator /path/to/other.yml
```

## Rails Application Variables

Navigator passes environment variables to Rails applications. These variables affect Rails behavior:

### RAILS_ENV
Rails environment mode.

**Common values**: `development`, `test`, `staging`, `production`

```yaml
# In Navigator configuration
applications:
  global_env:
    RAILS_ENV: production
```

```bash
# Set for Navigator process (inherited by Rails)
RAILS_ENV=production navigator config.yml
```

### SECRET_KEY_BASE
Rails secret key for encryption and signing.

```yaml
applications:
  global_env:
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
```

```bash
# Generate secret key
SECRET_KEY_BASE=$(openssl rand -hex 64)
export SECRET_KEY_BASE

# Start Navigator
navigator config.yml
```

### DATABASE_URL
Database connection string.

```yaml
applications:
  global_env:
    DATABASE_URL: "${DATABASE_URL}"
```

**Examples**:
```bash
# PostgreSQL
DATABASE_URL="postgresql://user:pass@localhost/myapp_production"

# MySQL
DATABASE_URL="mysql2://user:pass@localhost/myapp_production"

# SQLite
DATABASE_URL="sqlite3:///path/to/database.sqlite3"
```

### REDIS_URL
Redis connection string for caching and background jobs.

```yaml
applications:
  global_env:
    REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"
```

**Examples**:
```bash
# Local Redis
REDIS_URL="redis://localhost:6379"

# Remote Redis with auth
REDIS_URL="redis://user:pass@redis.example.com:6379"

# Redis with SSL
REDIS_URL="rediss://user:pass@redis.example.com:6380"
```

### RAILS_SERVE_STATIC_FILES
Control Rails static file serving.

**Recommended**: `false` (let Navigator serve static files)

```yaml
applications:
  global_env:
    RAILS_SERVE_STATIC_FILES: "false"
```

## Platform-Specific Variables

### Fly.io Variables

#### FLY_APP_NAME
Fly.io application name, required for regional routing fallback.

**Set by**: Fly.io platform  
**Used for**: Fly-Replay fallback proxy URLs

```yaml
# Navigator uses this for fallback URLs like:
# http://fra.${FLY_APP_NAME}.internal:3000/path
```

#### FLY_MACHINE_ID
Current machine identifier.

**Set by**: Fly.io platform  
**Used for**: Machine suspension feature

```bash
# Automatically set by Fly.io
echo $FLY_MACHINE_ID  # e48e123abc456def
```

#### FLY_REGION
Current region code.

**Set by**: Fly.io platform  
**Used for**: Regional routing logic

```bash
# Examples
FLY_REGION=ord  # Chicago
FLY_REGION=fra  # Frankfurt  
FLY_REGION=syd  # Sydney
```

### Heroku Variables

#### PORT
HTTP port for binding (Heroku sets this automatically).

```yaml
server:
  listen: "${PORT:-3000}"
```

#### DYNO
Dyno identifier (informational).

**Set by**: Heroku platform

### Docker Variables

Standard Docker environment variables:

#### HOSTNAME
Container hostname.

#### HOME
User home directory.

## Configuration Variable Substitution

Navigator supports environment variable substitution in YAML configuration:

### Basic Substitution

```yaml
applications:
  global_env:
    DATABASE_URL: "${DATABASE_URL}"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
```

### Default Values

```yaml
applications:
  global_env:
    # Use Redis URL or default to localhost
    REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"
    
    # Use Rails environment or default to production
    RAILS_ENV: "${RAILS_ENV:-production}"
    
    # Use custom port or default to 3000
    CUSTOM_PORT: "${CUSTOM_PORT:-3000}"
```

### Nested Substitution

```yaml
applications:
  global_env:
    # Combine environment variables
    DATABASE_URL: "postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}/${DB_NAME}"
    
    # Use environment in paths
    LOG_FILE: "/var/log/navigator-${RAILS_ENV}.log"
    
    # Conditional configuration
    STORAGE_PATH: "/storage/${RAILS_ENV}/${TENANT_ID}"
```

## Setting Environment Variables

### Development

#### Using Shell

```bash
# Set for current session
export RAILS_ENV=development
export DATABASE_URL=postgres://localhost/myapp_dev

# Start Navigator
navigator config.yml
```

#### Using .env File

```bash
# Create .env file (don't commit to git)
cat > .env << EOF
RAILS_ENV=development
DATABASE_URL=postgres://localhost/myapp_dev
SECRET_KEY_BASE=$(openssl rand -hex 64)
EOF

# Load and start Navigator
set -a  # Export all variables
source .env
set +a
navigator config.yml
```

#### Using direnv

```bash
# .envrc file
export RAILS_ENV=development
export DATABASE_URL=postgres://localhost/myapp_dev
export SECRET_KEY_BASE=$(openssl rand -hex 64)

# Auto-load with direnv
direnv allow
navigator config.yml
```

### Production

#### systemd Service

```ini title="/etc/systemd/system/navigator.service"
[Unit]
Description=Navigator Web Server

[Service]
# Environment variables
Environment=RAILS_ENV=production
Environment=LOG_LEVEL=info
Environment=DATABASE_URL=postgresql://user:pass@localhost/app_prod
EnvironmentFile=-/etc/navigator/environment

ExecStart=/usr/local/bin/navigator /etc/navigator/config.yml

[Install]
WantedBy=multi-user.target
```

```bash title="/etc/navigator/environment"
# Environment file for systemd
SECRET_KEY_BASE=your-secret-key-here
REDIS_URL=redis://localhost:6379
```

#### Docker

```dockerfile
# In Dockerfile
ENV RAILS_ENV=production
ENV LOG_LEVEL=info

# Or in docker run
docker run -e RAILS_ENV=production \
           -e SECRET_KEY_BASE=secret \
           navigator:latest
```

```yaml title="docker-compose.yml"
services:
  navigator:
    image: navigator:latest
    environment:
      - RAILS_ENV=production
      - SECRET_KEY_BASE=${SECRET_KEY_BASE}
      - DATABASE_URL=${DATABASE_URL}
    env_file:
      - .env.production
```

## Security Considerations

### Secret Management

**Never** hardcode secrets in configuration files:

```yaml
# DON'T DO THIS
applications:
  global_env:
    SECRET_KEY_BASE: "hardcoded-secret-here"  # NEVER!
    
# DO THIS INSTEAD
applications:
  global_env:
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
```

### Environment File Security

```bash
# Secure environment files
chmod 600 .env
chmod 600 /etc/navigator/environment

# Don't commit to version control
echo ".env" >> .gitignore
echo ".env.*" >> .gitignore
```

### Variable Validation

Validate required environment variables:

```bash
#!/bin/bash
# Check required variables before starting

required_vars=(
  "SECRET_KEY_BASE"
  "DATABASE_URL"  
  "RAILS_ENV"
)

for var in "${required_vars[@]}"; do
  if [ -z "${!var}" ]; then
    echo "Error: $var environment variable is required"
    exit 1
  fi
done

# Start Navigator
navigator config.yml
```

## Platform Integration Examples

### Fly.io

```yaml title="fly.toml"
[env]
  RAILS_ENV = "production"
  LOG_LEVEL = "info"

[processes]
web = "navigator config/production.yml"
```

### Heroku

```bash
# Set environment variables
heroku config:set RAILS_ENV=production
heroku config:set SECRET_KEY_BASE=$(rails secret)
heroku config:set LOG_LEVEL=info

# Procfile
web: navigator config/heroku.yml
```

### DigitalOcean App Platform

```yaml title=".do/app.yaml"
name: navigator-app
services:
- name: web
  environment_slug: ruby
  instance_count: 1
  instance_size_slug: basic-xxs
  
  envs:
  - key: RAILS_ENV
    value: production
  - key: LOG_LEVEL
    value: info
  - key: SECRET_KEY_BASE
    type: SECRET
    value: your-secret-key
    
  run_command: navigator config/production.yml
```

## Environment Variable Debugging

### List All Variables

```bash
# Show all environment variables
env | sort

# Show Rails-related variables
env | grep -E "RAILS|DATABASE|SECRET" | sort

# Show Navigator-related variables
env | grep -E "NAVIGATOR|LOG_LEVEL" | sort
```

### Test Variable Substitution

```bash
# Test variable expansion
echo "Database: ${DATABASE_URL:-not set}"
echo "Rails env: ${RAILS_ENV:-not set}"
echo "Log level: ${LOG_LEVEL:-not set}"
```

### Validate Configuration with Variables

```bash
# Check configuration with current environment
navigator --validate config.yml

# Test with specific environment
RAILS_ENV=production \
DATABASE_URL=postgres://localhost/test \
navigator --validate config.yml
```

## Common Issues

### Missing Required Variables

**Error**: Rails fails to start with environment-related errors

**Solution**: Ensure all required variables are set:
```bash
# Check if variables are set
echo "SECRET_KEY_BASE: ${SECRET_KEY_BASE:-NOT SET}"
echo "DATABASE_URL: ${DATABASE_URL:-NOT SET}"
```

### Variable Not Substituted

**Error**: Configuration contains literal `${VAR}` instead of value

**Cause**: Variable not set or incorrect syntax

**Solution**:
```bash
# Check variable is set
echo $DATABASE_URL

# Use default values in configuration
applications:
  global_env:
    DATABASE_URL: "${DATABASE_URL:-postgres://localhost/myapp}"
```

### Permission Denied Reading Environment File

**Error**: Cannot read environment file

**Solution**:
```bash
# Fix file permissions
chmod 600 /etc/navigator/environment
chown navigator:navigator /etc/navigator/environment
```

## See Also

- [Configuration Reference](../configuration/yaml-reference.md)
- [CLI Reference](cli.md)
- [Signal Handling](signals.md)
- [Examples](../examples/index.md)