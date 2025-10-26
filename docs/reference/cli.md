# Command-Line Interface

Navigator provides a simple command-line interface for starting the server, managing configuration, and controlling process behavior.

## Basic Usage

```bash
navigator [OPTIONS] [CONFIG_FILE]
```

### Examples

```bash
# Start with default configuration
navigator

# Start with specific config file
navigator /path/to/navigator.yml

# Show help
navigator --help

# Show version
navigator --version

# Reload running instance
navigator -s reload
```

## Command-Line Options

### Configuration Options

#### `CONFIG_FILE`
Path to YAML configuration file.

```bash
navigator /etc/navigator/production.yml
```

**Default behavior**: Navigator looks for configuration in this order:
1. Command-line argument
2. `config/navigator.yml`
3. `navigator.yml` in current directory

#### `--config`, `-c`
Alternative way to specify configuration file:

```bash
navigator --config /path/to/config.yml
navigator -c /path/to/config.yml
```

### Information Options

#### `--help`, `-h`
Display help information and exit:

```bash
navigator --help
```

Output:
```
Navigator - Go Web Server for Rails Applications

USAGE:
    navigator [OPTIONS] [CONFIG_FILE]

OPTIONS:
    -h, --help       Show this help message
    -V, --version    Show version information
    -s, --signal     Send signal to running process
    -c, --config     Configuration file path
    -v, --verbose    Enable verbose logging
    -q, --quiet      Suppress non-error output

EXAMPLES:
    navigator                           # Use default config
    navigator config/production.yml     # Use specific config
    navigator -s reload                 # Reload configuration
    navigator --version                 # Show version
```

#### `--version`, `-V`
Display version information and exit:

```bash
navigator --version
```

Output:
```
Navigator v0.16.0
Built with Go 1.24.0
```

### Control Options

#### `--signal`, `-s`
Send signal to running Navigator process:

```bash
navigator -s reload    # Reload configuration (SIGHUP)
navigator -s stop      # Graceful shutdown (SIGTERM)
navigator -s quit      # Immediate shutdown (SIGQUIT)
```

**Available signals**:
- `reload` - Reload configuration without restart
- `stop` - Graceful shutdown
- `quit` - Immediate shutdown

### Logging Options

#### `--verbose`, `-v`
Enable verbose debug logging:

```bash
navigator -v config.yml
# Equivalent to: LOG_LEVEL=debug navigator config.yml
```

#### `--quiet`, `-q`
Suppress non-error output:

```bash
navigator -q config.yml
# Only shows errors and warnings
```

### Validation Options

#### `--validate`, `--check`
Validate configuration file without starting:

```bash
navigator --validate config.yml
navigator --check config.yml
```

**Exit codes**:
- `0` - Configuration valid
- `1` - Configuration invalid
- `2` - File not found or read error

## Configuration File Handling

### File Discovery

Navigator searches for configuration files in this order:

1. **Command-line argument**:
   ```bash
   navigator /explicit/path/config.yml
   ```

2. **Default Rails location**:
   ```bash
   # Looks for: config/navigator.yml
   navigator
   ```

3. **Current directory**:
   ```bash
   # Looks for: ./navigator.yml
   navigator
   ```

### File Validation

Navigator validates configuration on startup:

```bash
# Valid configuration
navigator config.yml
# Output: Starting Navigator on :3000

# Invalid configuration  
navigator bad-config.yml
# Output: Error: invalid configuration...
# Exit code: 1
```

### Environment-Specific Configs

Use different configurations for different environments:

```bash
# Development
navigator config/navigator.dev.yml

# Staging
navigator config/navigator.staging.yml

# Production
navigator config/navigator.prod.yml
```

## Signal Management

### Reload Configuration

Reload configuration without restarting:

```bash
# Using CLI
navigator -s reload

# Using Unix signals
kill -HUP $(cat /tmp/navigator.pid)

# Using systemd
systemctl reload navigator
```

**Behavior**:
- Reloads YAML configuration
- Updates routing rules
- Starts new managed processes
- Existing Rails processes continue running
- No downtime for active requests

### Graceful Shutdown

Stop Navigator cleanly:

```bash
# Using CLI
navigator -s stop

# Using Unix signals
kill -TERM $(cat /tmp/navigator.pid)

# Using systemd
systemctl stop navigator
```

**Behavior**:
- Stops accepting new requests
- Waits for active requests to complete
- Stops Rails processes gracefully
- Stops managed processes
- Removes PID files

### Immediate Shutdown

Force immediate shutdown:

```bash
# Using CLI
navigator -s quit

# Using Unix signals
kill -QUIT $(cat /tmp/navigator.pid)
```

**Behavior**:
- Immediately stops all processes
- No graceful request handling
- May cause connection errors
- Use only when necessary

## Exit Codes

Navigator uses standard Unix exit codes:

| Code | Meaning | Examples |
|------|---------|----------|
| `0` | Success | Normal startup/shutdown |
| `1` | General error | Invalid config, port in use |
| `2` | Usage error | Invalid command line args |
| `130` | Interrupted | Ctrl+C (SIGINT) |

### Examples

```bash
# Check exit code
navigator config.yml
echo $?  # 0 = success, non-zero = error

# Use in scripts
if navigator --validate config.yml; then
    echo "Configuration valid"
    navigator config.yml
else
    echo "Configuration invalid"
    exit 1
fi
```

## Environment Variables

### Navigator-Specific

| Variable | Purpose | CLI Equivalent |
|----------|---------|---------------|
| `LOG_LEVEL` | Set logging level | `--verbose`, `--quiet` |
| `NAVIGATOR_CONFIG` | Default config file | First argument |
| `NAVIGATOR_PID_FILE` | PID file location | N/A |

### Usage Examples

```bash
# Set log level
LOG_LEVEL=debug navigator config.yml

# Set default config
NAVIGATOR_CONFIG=/etc/navigator.yml navigator

# Custom PID file location
NAVIGATOR_PID_FILE=/var/run/navigator.pid navigator
```

## Integration Examples

### systemd Service

```ini title="/etc/systemd/system/navigator.service"
[Unit]
Description=Navigator Web Server
After=network.target

[Service]
Type=simple
User=navigator
Group=navigator
WorkingDirectory=/var/www/app
ExecStart=/usr/local/bin/navigator /etc/navigator/config.yml
ExecReload=/usr/local/bin/navigator -s reload
Restart=always
RestartSec=5

# Environment
Environment=LOG_LEVEL=info
Environment=RAILS_ENV=production

[Install]
WantedBy=multi-user.target
```

### Docker Container

```dockerfile
FROM ruby:3.2-slim

# Install Navigator
COPY navigator /usr/local/bin/
RUN chmod +x /usr/local/bin/navigator

# Rails application
WORKDIR /app
COPY . .
RUN bundle install

# Configuration
COPY navigator.yml /app/
EXPOSE 3000

# Health check using CLI
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD navigator --validate /app/navigator.yml || exit 1

# Start Navigator
CMD ["navigator", "/app/navigator.yml"]
```

### Process Management Scripts

```bash
#!/bin/bash
# Navigator management script

PIDFILE="/tmp/navigator.pid"
CONFIG="/etc/navigator/config.yml"

case "$1" in
  start)
    if [ -f "$PIDFILE" ]; then
      echo "Navigator already running (PID: $(cat $PIDFILE))"
      exit 1
    fi
    echo "Starting Navigator..."
    navigator "$CONFIG"
    ;;
  
  stop)
    if [ ! -f "$PIDFILE" ]; then
      echo "Navigator not running"
      exit 1
    fi
    echo "Stopping Navigator..."
    navigator -s stop
    ;;
  
  restart)
    $0 stop
    sleep 2
    $0 start
    ;;
  
  reload)
    if [ ! -f "$PIDFILE" ]; then
      echo "Navigator not running"
      exit 1
    fi
    echo "Reloading Navigator configuration..."
    navigator -s reload
    ;;
  
  status)
    if [ -f "$PIDFILE" ]; then
      PID=$(cat "$PIDFILE")
      if kill -0 "$PID" 2>/dev/null; then
        echo "Navigator running (PID: $PID)"
      else
        echo "Navigator not running (stale PID file)"
        rm -f "$PIDFILE"
      fi
    else
      echo "Navigator not running"
    fi
    ;;
  
  validate)
    echo "Validating configuration..."
    navigator --validate "$CONFIG"
    ;;
  
  *)
    echo "Usage: $0 {start|stop|restart|reload|status|validate}"
    exit 2
    ;;
esac
```

## Debugging and Troubleshooting

### Enable Debug Logging

```bash
# Method 1: CLI flag
navigator -v config.yml

# Method 2: Environment variable
LOG_LEVEL=debug navigator config.yml

# Method 3: In configuration
# Set LOG_LEVEL=debug in environment before starting
```

### Validate Configuration

```bash
# Check configuration syntax
navigator --validate config.yml

# Check with verbose output
LOG_LEVEL=debug navigator --validate config.yml
```

### Test Signal Handling

```bash
# Start Navigator
navigator config.yml &
NAVIGATOR_PID=$!

# Test reload
navigator -s reload

# Test graceful shutdown
navigator -s stop

# Check if process stopped
kill -0 $NAVIGATOR_PID 2>/dev/null || echo "Process stopped"
```

### Check Process Status

```bash
# Check if Navigator is running
if [ -f "/tmp/navigator.pid" ]; then
  PID=$(cat /tmp/navigator.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "Navigator running (PID: $PID)"
  else
    echo "Stale PID file"
  fi
else
  echo "Navigator not running"
fi

# Check ports
netstat -tlnp | grep -E "(3000|400[0-9])"
```

## Common Usage Patterns

### Development Workflow

```bash
# Start in foreground for development
navigator config/dev.yml

# Or with debug logging
LOG_LEVEL=debug navigator config/dev.yml

# Reload after config changes
navigator -s reload
```

### Production Deployment

```bash
# Validate before deployment
navigator --validate /etc/navigator/prod.yml

# Start production server
navigator /etc/navigator/prod.yml

# Zero-downtime config updates
navigator -s reload
```

### CI/CD Integration

```bash
# In deployment script
echo "Validating Navigator configuration..."
if ! navigator --validate config/production.yml; then
  echo "Invalid configuration, aborting deployment"
  exit 1
fi

echo "Deploying application..."
# ... deployment steps ...

echo "Reloading Navigator..."
navigator -s reload
```

## See Also

- [Configuration Reference](../configuration/yaml-reference.md)
- [Environment Variables](environment.md)  
- [Signal Handling](signals.md)
- [Examples](../examples/index.md)