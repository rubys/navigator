# systemd Service Example

This example shows how to run Navigator as a systemd service for production deployments with automatic startup, logging, and process management.

## Quick Start

```bash
# 1. Install Navigator
sudo cp navigator /usr/local/bin/
sudo chmod +x /usr/local/bin/navigator

# 2. Create service file
sudo tee /etc/systemd/system/navigator.service << 'EOF'
[Unit]
Description=Navigator Rails Server
After=network.target

[Service]
Type=simple
User=navigator
ExecStart=/usr/local/bin/navigator /etc/navigator/config.yml
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# 3. Enable and start
sudo systemctl enable navigator
sudo systemctl start navigator
```

## Complete Service Configuration

### Basic Service File

```ini title="/etc/systemd/system/navigator.service"
[Unit]
Description=Navigator Rails Proxy Server
Documentation=https://rubys.github.io/navigator/
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=navigator
Group=navigator
WorkingDirectory=/var/www/app

# Main command
ExecStart=/usr/local/bin/navigator /etc/navigator/production.yml

# Configuration reload without restart
ExecReload=/usr/local/bin/navigator -s reload

# Process management
Restart=always
RestartSec=10
KillSignal=SIGTERM
TimeoutStopSec=60
KillMode=mixed

# Environment variables
Environment=RAILS_ENV=production
Environment=LOG_LEVEL=info
EnvironmentFile=-/etc/navigator/environment

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/www/app/log /var/www/app/tmp /var/www/app/storage
PrivateTmp=true

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

[Install]
WantedBy=multi-user.target
```

### Environment File

```bash title="/etc/navigator/environment"
# Database configuration
DATABASE_URL=postgresql://user:password@localhost/app_production

# Rails secrets  
SECRET_KEY_BASE=your-very-long-secret-key-here
RAILS_MASTER_KEY=your-rails-master-key

# Redis configuration
REDIS_URL=redis://localhost:6379

# Monitoring services
NEW_RELIC_LICENSE_KEY=your-license-key
HONEYBADGER_API_KEY=your-api-key

# Custom application settings
RAILS_MAX_THREADS=5
WEB_CONCURRENCY=2
```

## Setup Steps

### 1. Create Navigator User

```bash
# Create system user
sudo useradd --system --no-create-home --shell /bin/false navigator

# Create group
sudo groupadd navigator
sudo usermod -a -G navigator navigator

# Set up application directory
sudo mkdir -p /var/www/app
sudo chown -R navigator:navigator /var/www/app
sudo chmod -R 755 /var/www/app
```

### 2. Install Navigator Binary

```bash
# Download and install
wget https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz
tar xzf navigator-linux-amd64.tar.gz
sudo mv navigator /usr/local/bin/
sudo chmod +x /usr/local/bin/navigator

# Verify installation
navigator --version
```

### 3. Create Configuration

```bash
# Create configuration directory
sudo mkdir -p /etc/navigator
sudo chown root:navigator /etc/navigator
sudo chmod 750 /etc/navigator

# Create production config
sudo tee /etc/navigator/production.yml << 'EOF'
server:
  listen: 3000
  public_dir: /var/www/app/public

pools:
  max_size: 10
  idle_timeout: 300

applications:
  global_env:
    RAILS_ENV: production
    RAILS_SERVE_STATIC_FILES: "false"
  
  tenants:
    - name: production
      path: /
      working_dir: /var/www/app
EOF

# Secure configuration
sudo chmod 640 /etc/navigator/production.yml
```

### 4. Set Up Environment File

```bash
# Create environment file
sudo touch /etc/navigator/environment
sudo chown root:navigator /etc/navigator/environment
sudo chmod 640 /etc/navigator/environment

# Add environment variables (edit with your values)
sudo tee /etc/navigator/environment << 'EOF'
DATABASE_URL=postgresql://user:password@localhost/app_production
SECRET_KEY_BASE=your-secret-key-here
REDIS_URL=redis://localhost:6379
EOF
```

### 5. Install and Enable Service

```bash
# Copy service file (using the complete version above)
sudo systemctl daemon-reload

# Enable service for auto-start
sudo systemctl enable navigator

# Start service
sudo systemctl start navigator

# Check status
sudo systemctl status navigator
```

## Service Management Commands

### Basic Operations

```bash
# Start Navigator
sudo systemctl start navigator

# Stop Navigator  
sudo systemctl stop navigator

# Restart Navigator
sudo systemctl restart navigator

# Reload configuration without restart
sudo systemctl reload navigator

# Enable auto-start on boot
sudo systemctl enable navigator

# Disable auto-start
sudo systemctl disable navigator
```

### Status and Monitoring

```bash
# Check service status
sudo systemctl status navigator

# View recent logs
sudo journalctl -u navigator -n 50

# Follow logs in real-time
sudo journalctl -u navigator -f

# View logs from today
sudo journalctl -u navigator --since today

# View error logs only
sudo journalctl -u navigator -p err
```

## Configuration Examples

### High-Availability Setup

```ini title="/etc/systemd/system/navigator.service"
[Unit]
Description=Navigator Rails Server (HA)
After=network.target postgresql.service redis.service
Requires=postgresql.service redis.service

[Service]
Type=simple
User=navigator
Group=navigator
WorkingDirectory=/var/www/app
ExecStart=/usr/local/bin/navigator /etc/navigator/production.yml
ExecReload=/usr/local/bin/navigator -s reload

# High availability settings
Restart=always
RestartSec=5
StartLimitInterval=0

# Health checking
ExecStartPre=/usr/local/bin/health-check.sh
TimeoutStartSec=30
TimeoutStopSec=120

# Resource management
LimitNOFILE=65536
LimitNPROC=32768
OOMScoreAdjust=-500

[Install]
WantedBy=multi-user.target
```

### Development Setup

```ini title="/etc/systemd/system/navigator-dev.service"
[Unit]
Description=Navigator Rails Server (Development)
After=network.target

[Service]
Type=simple
User=developer
Group=developer
WorkingDirectory=/home/developer/myapp
ExecStart=/usr/local/bin/navigator config/navigator.yml

# Development settings
Restart=on-failure
RestartSec=2
Environment=RAILS_ENV=development
Environment=LOG_LEVEL=debug

# Allow user to see logs easily
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### Multi-Instance Setup

```ini title="/etc/systemd/system/navigator@.service"
[Unit]
Description=Navigator Rails Server (%i)
After=network.target

[Service]
Type=simple
User=navigator
Group=navigator
WorkingDirectory=/var/www/%i
ExecStart=/usr/local/bin/navigator /etc/navigator/%i.yml
ExecReload=/usr/local/bin/navigator -s reload

Restart=always
RestartSec=10

# Instance-specific environment
EnvironmentFile=/etc/navigator/%i.env

[Install]
WantedBy=multi-user.target
```

```bash
# Start multiple instances
sudo systemctl enable navigator@app1
sudo systemctl enable navigator@app2
sudo systemctl start navigator@app1
sudo systemctl start navigator@app2
```

## Security Hardening

### Enhanced Security Configuration

```ini title="/etc/systemd/system/navigator.service"
[Service]
# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true

# File system access
ReadWritePaths=/var/www/app/log /var/www/app/tmp /var/www/app/storage
ReadOnlyPaths=/var/www/app

# Network restrictions
IPAddressDeny=any
IPAddressAllow=localhost
IPAddressAllow=127.0.0.1/8
IPAddressAllow=10.0.0.0/8

# System call filtering
SystemCallArchitectures=native
SystemCallFilter=@system-service
SystemCallFilter=~@debug @mount @cpu-emulation @obsolete
```

## Logging Configuration

### Log Management

```bash
# Create log directory
sudo mkdir -p /var/log/navigator
sudo chown navigator:navigator /var/log/navigator

# Configure rsyslog for Navigator logs
sudo tee /etc/rsyslog.d/navigator.conf << 'EOF'
# Navigator logs
if $programname == 'navigator' then /var/log/navigator/navigator.log
& stop
EOF

# Restart rsyslog
sudo systemctl restart rsyslog
```

### Log Rotation

```bash
# Configure log rotation
sudo tee /etc/logrotate.d/navigator << 'EOF'
/var/log/navigator/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 navigator navigator
    sharedscripts
    postrotate
        /usr/bin/systemctl reload navigator
    endscript
}
EOF
```

## Monitoring and Health Checks

### Health Check Script

```bash title="/usr/local/bin/navigator-health-check.sh"
#!/bin/bash
# Navigator health check for systemd

# Check process is running
if ! systemctl is-active --quiet navigator; then
    echo "Navigator service not active"
    exit 1
fi

# Check HTTP response
if ! curl -f -s http://localhost:3000/up > /dev/null; then
    echo "Navigator not responding to HTTP requests"
    exit 1
fi

# Check log for recent errors
if journalctl -u navigator --since "5 minutes ago" -p err | grep -q "ERROR"; then
    echo "Recent errors found in Navigator logs"
    exit 1
fi

echo "Navigator health check passed"
exit 0
```

```bash
# Make executable and test
sudo chmod +x /usr/local/bin/navigator-health-check.sh
/usr/local/bin/navigator-health-check.sh
```

### Systemd Monitoring Integration

```ini
# Add to [Service] section
ExecStartPre=/usr/local/bin/navigator-health-check.sh
WatchdogSec=60
NotifyAccess=all
```

## Troubleshooting

### Common Issues

#### Service Fails to Start

```bash
# Check detailed status
sudo systemctl status navigator -l

# Check logs for errors
sudo journalctl -u navigator -n 20

# Validate configuration
sudo -u navigator navigator --validate /etc/navigator/production.yml

# Check file permissions
ls -la /etc/navigator/
ls -la /usr/local/bin/navigator
```

#### Service Stops Unexpectedly

```bash
# Check why service stopped
sudo journalctl -u navigator --since "1 hour ago"

# Look for system events
sudo journalctl --since "1 hour ago" | grep navigator

# Check resource usage
sudo systemctl show navigator --property=MemoryUsage,CPUUsage
```

#### Configuration Reload Fails

```bash
# Test reload manually
sudo systemctl reload navigator

# Check reload logs
sudo journalctl -u navigator | grep reload

# Validate configuration
sudo -u navigator navigator --validate /etc/navigator/production.yml
```

### Service Debugging

```bash
# Run Navigator manually for debugging
sudo systemctl stop navigator
sudo -u navigator /usr/local/bin/navigator /etc/navigator/production.yml

# Enable debug logging
sudo systemctl edit navigator
# Add:
[Service]
Environment=LOG_LEVEL=debug

# View service environment
sudo systemctl show-environment
sudo systemctl show navigator --property=Environment
```

## Integration Examples

### nginx + Navigator

```nginx title="/etc/nginx/sites-available/navigator"
upstream navigator {
    server 127.0.0.1:3000 fail_timeout=0;
}

server {
    listen 80;
    server_name myapp.com;
    
    location / {
        proxy_pass http://navigator;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect off;
    }
}
```

### PostgreSQL Dependencies

```ini
# Add to [Unit] section
After=postgresql.service
Wants=postgresql.service

# Add to [Service] section
ExecStartPre=/usr/bin/pg_isready -h localhost -p 5432
```

### Load Balancer Health Checks

```ini
# Add health check endpoint
[Service]
ExecStartPost=/bin/bash -c 'until curl -f http://localhost:3000/up; do sleep 1; done'
```

## Best Practices

### 1. Security
- Run as dedicated non-root user
- Use environment files for secrets
- Enable security hardening options
- Restrict file system access

### 2. Reliability  
- Configure automatic restart
- Set appropriate timeouts
- Monitor health with watchdog
- Use dependency management

### 3. Operations
- Centralized logging
- Log rotation
- Health monitoring
- Graceful configuration reloads

### 4. Performance
- Optimize resource limits
- Use appropriate restart policies
- Monitor memory and CPU usage
- Scale with multiple instances

## See Also

- [Production Deployment](../deployment/production.md)
- [Configuration Reference](../configuration/yaml-reference.md)
- [Process Management](../features/process-management.md)
- [Monitoring Setup](../deployment/monitoring.md)