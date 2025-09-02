# Production Deployment

This guide covers deploying Navigator in production environments with best practices for security, performance, and reliability.

## Quick Start

```bash
# 1. Install Navigator
wget https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz
tar xzf navigator-linux-amd64.tar.gz
sudo mv navigator /usr/local/bin/
sudo chmod +x /usr/local/bin/navigator

# 2. Create configuration
sudo mkdir -p /etc/navigator
sudo cp config.yml /etc/navigator/production.yml

# 3. Set up systemd service
sudo cp navigator.service /etc/systemd/system/
sudo systemctl enable navigator
sudo systemctl start navigator
```

## System Requirements

### Hardware Requirements

| Component | Minimum | Recommended | Notes |
|-----------|---------|-------------|-------|
| **CPU** | 1 core | 2+ cores | Per Rails process |
| **Memory** | 1GB | 4GB+ | Rails apps are memory-intensive |
| **Storage** | 5GB | 20GB+ | Logs, uploads, assets |
| **Network** | 100Mbps | 1Gbps+ | For high traffic sites |

### Software Requirements

| Software | Version | Purpose |
|----------|---------|---------|
| **Linux** | Ubuntu 20.04+, RHEL 8+ | Production OS |
| **Ruby** | 3.0+ | Rails applications |
| **Node.js** | 16+ | Asset compilation |
| **Database** | PostgreSQL 12+, MySQL 8+ | Application data |
| **Redis** | 6.0+ | Caching, sessions |

## Installation

### Binary Installation (Recommended)

```bash
# Download latest release
cd /tmp
wget https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz

# Extract and install
tar xzf navigator-linux-amd64.tar.gz
sudo mv navigator /usr/local/bin/
sudo chmod +x /usr/local/bin/navigator

# Verify installation
navigator --version
```

### Building from Source

```bash
# Install Go 1.21+
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone and build
git clone https://github.com/rubys/navigator.git
cd navigator
make build
sudo cp bin/navigator /usr/local/bin/
```

## Configuration

### Production Configuration

```yaml title="/etc/navigator/production.yml"
server:
  listen: 3000
  hostname: myapp.com
  public_dir: /var/www/app/public

pools:
  max_size: 20           # Scale based on server capacity
  idle_timeout: 600      # 10 minutes for production
  start_port: 4000

# Authentication for admin areas
auth:
  enabled: true
  realm: "Production Application"
  htpasswd: /etc/navigator/htpasswd
  public_paths:
    - /assets/
    - /robots.txt
    - /favicon.ico
    - "*.css"
    - "*.js"

# Optimized static file serving
static:
  directories:
    - path: /assets/
      root: /var/www/app/public/assets/
      cache: 31536000      # 1 year for fingerprinted assets
    - path: /images/
      root: /var/www/app/public/images/
      cache: 86400         # 1 day for images
  extensions: [css, js, png, jpg, gif, ico, svg, woff, woff2]

applications:
  global_env:
    RAILS_ENV: production
    RAILS_SERVE_STATIC_FILES: "false"
    RAILS_LOG_TO_STDOUT: "true"
    SECRET_KEY_BASE: "${SECRET_KEY_BASE}"
    DATABASE_URL: "${DATABASE_URL}"
    REDIS_URL: "${REDIS_URL}"
    
  tenants:
    - name: production
      path: /
      working_dir: /var/www/app

# Managed processes
managed_processes:
  - name: redis
    command: redis-server
    args: [/etc/redis/redis.conf]
    auto_restart: true
    
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /var/www/app
    env:
      RAILS_ENV: production
    auto_restart: true
    start_delay: 2
```

### Environment Variables

```bash title="/etc/navigator/environment"
# Database
DATABASE_URL=postgresql://user:password@localhost/app_production

# Rails secrets
SECRET_KEY_BASE=your-very-long-secret-key-here
RAILS_MASTER_KEY=your-rails-master-key

# Redis
REDIS_URL=redis://localhost:6379

# Monitoring
NEW_RELIC_LICENSE_KEY=your-license-key
HONEYBADGER_API_KEY=your-api-key
```

## systemd Service

### Service Configuration

```ini title="/etc/systemd/system/navigator.service"
[Unit]
Description=Navigator Rails Proxy Server
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=navigator
Group=navigator
WorkingDirectory=/var/www/app

# Main command
ExecStart=/usr/local/bin/navigator /etc/navigator/production.yml

# Reload configuration without restart
ExecReload=/usr/local/bin/navigator -s reload

# Process management
Restart=always
RestartSec=10
KillSignal=SIGTERM
TimeoutStopSec=60

# Environment
Environment=RAILS_ENV=production
Environment=LOG_LEVEL=info
EnvironmentFile=-/etc/navigator/environment

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/www/app/log /var/www/app/tmp /var/www/app/storage

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

[Install]
WantedBy=multi-user.target
```

### Service Management

```bash
# Enable and start service
sudo systemctl enable navigator
sudo systemctl start navigator

# Check status
sudo systemctl status navigator

# View logs
sudo journalctl -u navigator -f

# Reload configuration
sudo systemctl reload navigator

# Restart service
sudo systemctl restart navigator
```

## User and Permissions

### Create Navigator User

```bash
# Create dedicated user
sudo useradd --system --no-create-home --shell /bin/false navigator

# Create group
sudo groupadd navigator
sudo usermod -a -G navigator navigator

# Set up directories
sudo mkdir -p /var/www/app
sudo chown -R navigator:navigator /var/www/app
sudo chmod -R 755 /var/www/app

# Configuration permissions
sudo chown -R root:navigator /etc/navigator
sudo chmod -R 640 /etc/navigator
```

### File Permissions

```bash
# Application files
sudo chown -R navigator:navigator /var/www/app
sudo find /var/www/app -type f -exec chmod 644 {} \;
sudo find /var/www/app -type d -exec chmod 755 {} \;

# Writable directories
sudo chmod 755 /var/www/app/log
sudo chmod 755 /var/www/app/tmp
sudo chmod 755 /var/www/app/storage

# Configuration security
sudo chmod 600 /etc/navigator/production.yml
sudo chmod 600 /etc/navigator/environment
```

## Security

### Authentication Setup

```bash
# Create htpasswd file
sudo htpasswd -c /etc/navigator/htpasswd admin
sudo htpasswd /etc/navigator/htpasswd user1

# Secure htpasswd file
sudo chown root:navigator /etc/navigator/htpasswd
sudo chmod 640 /etc/navigator/htpasswd
```

### Firewall Configuration

```bash
# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow SSH (adjust port as needed)
sudo ufw allow 22/tcp

# Enable firewall
sudo ufw --force enable
```

### SSL/TLS Termination

For production, use a reverse proxy for SSL termination:

```nginx title="/etc/nginx/sites-available/navigator"
server {
    listen 80;
    server_name myapp.com www.myapp.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name myapp.com www.myapp.com;

    ssl_certificate /path/to/ssl/cert.pem;
    ssl_certificate_key /path/to/ssl/private.key;
    
    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!MD5:!DSS;
    ssl_prefer_server_ciphers off;
    
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Monitoring and Logging

### Log Configuration

```bash
# Create log directory
sudo mkdir -p /var/log/navigator
sudo chown navigator:navigator /var/log/navigator

# Configure log rotation
sudo tee /etc/logrotate.d/navigator << EOF
/var/log/navigator/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 navigator navigator
    postrotate
        /usr/local/bin/navigator -s reload
    endscript
}
EOF
```

### Health Checks

```bash
#!/bin/bash
# /usr/local/bin/health-check.sh

# Check Navigator process
if ! pgrep -f navigator > /dev/null; then
    echo "ERROR: Navigator not running"
    exit 1
fi

# Check HTTP response
if ! curl -f http://localhost:3000/up > /dev/null 2>&1; then
    echo "ERROR: Navigator not responding"
    exit 1
fi

echo "OK: Navigator healthy"
exit 0
```

```bash
# Make executable and test
sudo chmod +x /usr/local/bin/health-check.sh
/usr/local/bin/health-check.sh
```

### Monitoring Integration

#### New Relic

```yaml
# Add to applications.global_env
NEW_RELIC_LICENSE_KEY: "${NEW_RELIC_LICENSE_KEY}"
NEW_RELIC_APP_NAME: "MyApp Production"
```

#### Prometheus Metrics

```bash
# Custom metrics script
#!/bin/bash
# /usr/local/bin/navigator-metrics.sh

# Process count
echo "navigator_processes $(pgrep -f navigator | wc -l)"

# Memory usage  
echo "navigator_memory_bytes $(ps -o pid,rss -p $(pgrep -f navigator) | tail -n +2 | awk '{sum+=$2} END {print sum*1024}')"

# Request count (from logs)
echo "navigator_requests_total $(grep -c 'GET\|POST' /var/log/navigator/access.log)"
```

## Performance Tuning

### Resource Optimization

```yaml
# Optimize for server capacity
pools:
  max_size: 20          # 2GB RAM = ~10 processes, 4GB = ~20
  idle_timeout: 600     # Keep processes alive longer
  start_port: 4000

# Efficient static file serving
static:
  directories:
    - path: /assets/
      root: /var/www/app/public/assets/
      cache: 31536000    # Long cache for assets
```

### Database Optimization

```bash
# PostgreSQL connection pooling
DATABASE_URL=postgresql://user:pass@localhost/app?pool=25

# Redis optimization
REDIS_URL=redis://localhost:6379/0?pool_size=10
```

### System Tuning

```bash
# Increase file descriptor limits
echo "navigator soft nofile 65536" >> /etc/security/limits.conf
echo "navigator hard nofile 65536" >> /etc/security/limits.conf

# Kernel network tuning
echo 'net.core.somaxconn = 32768' >> /etc/sysctl.conf
echo 'net.ipv4.ip_local_port_range = 1024 65535' >> /etc/sysctl.conf
sysctl -p
```

## Backup and Recovery

### Configuration Backup

```bash
#!/bin/bash
# /usr/local/bin/backup-navigator.sh

BACKUP_DIR="/backup/navigator/$(date +%Y-%m-%d)"
mkdir -p "$BACKUP_DIR"

# Backup configuration
cp -r /etc/navigator "$BACKUP_DIR/"

# Backup application
tar czf "$BACKUP_DIR/app.tar.gz" /var/www/app

echo "Backup completed: $BACKUP_DIR"
```

### Disaster Recovery

```bash
#!/bin/bash
# /usr/local/bin/restore-navigator.sh

BACKUP_DIR="$1"

if [[ -z "$BACKUP_DIR" ]]; then
    echo "Usage: $0 /path/to/backup"
    exit 1
fi

# Stop Navigator
sudo systemctl stop navigator

# Restore configuration
sudo cp -r "$BACKUP_DIR/navigator" /etc/

# Restore application
tar xzf "$BACKUP_DIR/app.tar.gz" -C /

# Fix permissions
sudo chown -R navigator:navigator /var/www/app

# Start Navigator
sudo systemctl start navigator
```

## Troubleshooting

### Common Issues

#### Navigator Won't Start

```bash
# Check configuration
navigator --validate /etc/navigator/production.yml

# Check permissions
ls -la /etc/navigator/
sudo -u navigator navigator /etc/navigator/production.yml

# Check systemd logs
sudo journalctl -u navigator -n 50
```

#### High Memory Usage

```bash
# Check Rails processes
ps aux | grep -E "(navigator|rails|ruby)" | sort -k 4 -nr

# Monitor memory over time
while true; do
  ps -o pid,rss,cmd -p $(pgrep -f rails) | tail -n +2 | awk '{sum+=$2} END {print sum/1024 " MB"}'
  sleep 5
done
```

#### Poor Performance

```bash
# Check process count
ps aux | grep rails | wc -l

# Monitor response times
tail -f /var/log/navigator.log | grep -E "completed|duration"

# Check database connections
sudo -u postgres psql -c "SELECT count(*) FROM pg_stat_activity WHERE datname='app_production';"
```

### Performance Monitoring

```bash
# Request monitoring
tail -f /var/log/navigator.log | grep -E "(GET|POST)" | head -20

# Process monitoring  
watch 'ps aux | grep -E "(navigator|rails)" | head -10'

# Memory monitoring
watch 'free -h'
```

## Best Practices

### 1. Configuration Management

- Version control all configuration files
- Use environment variables for secrets
- Validate configuration before deployment
- Test configuration changes in staging first

### 2. Security

- Run Navigator as dedicated non-root user
- Use strong authentication for admin areas
- Keep secrets in environment files, not config
- Regular security updates

### 3. Monitoring

- Set up health checks
- Monitor logs for errors
- Track resource usage
- Alert on service failures

### 4. Deployment

- Use blue-green deployments
- Test thoroughly in staging
- Have rollback procedures ready
- Automate common tasks

## See Also

- [Configuration Reference](../configuration/yaml-reference.md)
- [systemd Integration](../examples/systemd.md)
- [Monitoring Guide](monitoring.md)
- [Security Best Practices](../security/index.md)