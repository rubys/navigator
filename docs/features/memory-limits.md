# Per-Tenant Memory Limits

Navigator provides per-tenant memory limiting using Linux cgroups v2, ensuring that memory-hungry tenants cannot impact other tenants on the same machine.

## How It Works

Navigator uses Linux control groups (cgroups) v2 to enforce hard memory limits for each tenant application:

1. **Cgroup creation** - Navigator creates a dedicated cgroup for each tenant under `/sys/fs/cgroup/navigator/<tenant>`
2. **Memory limit** - Sets `memory.max` to the configured limit (e.g., 512MB, 1GB)
3. **Kernel enforcement** - Linux kernel tracks memory usage and enforces limits
4. **OOM handling** - When limit is exceeded, kernel OOM kills only that tenant
5. **Auto-restart** - Tenant restarts automatically on next incoming request

## Platform Requirements

**Linux Only:**
- Linux operating system with kernel 4.5+ (cgroups v1) or 5.4+ (cgroups v2)
- Navigator running as root
- Debian 12+ (Bookworm), Ubuntu 22.04+, or equivalent
- Automatic detection and support for both cgroups v1 and v2

**Graceful Degradation:**
- macOS: Configuration ignored, logged at debug level
- Windows: Configuration ignored, logged at debug level
- Non-root: Configuration ignored, logged at debug level

**Cgroups Version Support:**
- **cgroups v2** (preferred): Modern unified hierarchy, better resource control
- **cgroups v1** (legacy): Supported with automatic fallback for older systems
- Navigator automatically detects and uses the available cgroups version

## Configuration

### Basic Setup

Set a default memory limit for all tenants:

```yaml
applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000
    default_memory_limit: "512M"  # Default for all tenants
```

### Per-Tenant Overrides

Configure different limits for specific tenants:

```yaml
applications:
  pools:
    default_memory_limit: "512M"  # Default for most tenants

  tenants:
    - name: 2025/boston
      path: /2025/boston/
      memory_limit: "384M"        # Small event

    - name: 2025/newyork
      path: /2025/newyork/
      memory_limit: "768M"        # Large event

    - name: 2025/chicago
      path: /2025/chicago/
      # Uses default (512M)
```

### Memory Size Formats

Navigator supports human-readable memory sizes:

| Format | Description | Bytes |
|--------|-------------|-------|
| `512M` | 512 megabytes | 536,870,912 |
| `1G` | 1 gigabyte | 1,073,741,824 |
| `1.5G` | 1.5 gigabytes | 1,610,612,736 |
| `2048M` | 2048 megabytes | 2,147,483,648 |

**Supported units:** K (kilobytes), M (megabytes), G (gigabytes), T (terabytes)

**Alternate formats:** `512MB`, `1GB`, `1GiB` are also accepted

## User and Group Isolation

Run tenant processes as non-root users for enhanced security:

```yaml
applications:
  pools:
    default_memory_limit: "512M"
    user: "rails"   # Default user for all tenants
    group: "rails"  # Default group for all tenants

  tenants:
    - name: special-tenant
      path: /special/
      memory_limit: "1G"
      user: "app"     # Override: run as different user
      group: "app"    # Override: run as different group
```

**Security Benefits:**
- Limits tenant process permissions
- Prevents tenant code from accessing Navigator internals
- Isolates tenants from each other at OS level

**Requirements:**
- Unix-like OS (Linux, macOS with limitations)
- Navigator running as root
- User and group must exist on system

## Capacity Planning

### Rails 8 + Puma Baseline

A typical Rails 8 application with Puma and 3 threads uses:

- **Baseline memory**: 300-400MB
- **Recommended limit**: 512MB
- **Margin**: ~150MB for request handling

### Machine Sizing

Example for a 2GB Fly.io machine:

```yaml
applications:
  pools:
    default_memory_limit: "512M"  # 512MB per tenant

# Capacity: ~3 active tenants per 2GB machine
# - 3 × 512MB = 1,536MB for tenants
# - ~500MB for system + Navigator
```

### Per-Tenant Sizing

Adjust limits based on actual usage:

```yaml
applications:
  pools:
    default_memory_limit: "512M"

  tenants:
    # Small events (~50 attendees)
    - name: 2025/smalltown
      memory_limit: "384M"

    # Medium events (~200 attendees)
    - name: 2025/boston
      memory_limit: "512M"  # Use default

    # Large events (~500 attendees)
    - name: 2025/newyork
      memory_limit: "768M"
```

## OOM Kill Behavior

When a tenant exceeds its memory limit:

### Detection and Logging

Navigator monitors cgroup `memory.events` for OOM kills:

```log
ERROR Tenant OOM killed by kernel tenant=2025/boston limit=512.0 MiB oomCount=1
```

### Automatic Restart

1. **Detection**: Navigator periodically detects OOM via cgroup events
2. **Cleanup**: Removes tenant from process registry
3. **Next request**: Incoming request triggers restart via `GetOrStartApp()`
4. **Fresh start**: New process starts with same memory limit

**No auto-restart loop**: Tenant only restarts when a request actually arrives, preventing rapid restart cycles.

### Cgroup Persistence

Cgroups remain in place after OOM kills:

- **Idle timeout**: Cgroup persists when tenant goes idle
- **OOM kill**: Cgroup persists for reuse on restart
- **Navigator shutdown**: Cgroups cleaned up when Navigator stops

**Benefits:**
- Faster tenant restart (cgroup already configured)
- Preserves OOM statistics across restarts
- No cgroup churn during normal operation

## Monitoring

### Log Messages

Navigator logs memory limit events:

```log
# Cgroup setup
INFO Memory limit configured for tenant tenant=2025/boston limit=512.0 MiB cgroup=/sys/fs/cgroup/navigator/2025_boston

# OOM kill
ERROR Tenant OOM killed by kernel tenant=2025/boston limit=512.0 MiB oomCount=1

# Repeated OOM (investigate tenant)
ERROR Tenant OOM killed by kernel tenant=2025/boston limit=512.0 MiB oomCount=5
```

### Memory Statistics on Shutdown

Navigator automatically logs detailed memory statistics when tenants stop (either from idle timeout or Navigator shutdown):

```log
# Tenant shutdown statistics (cgroups v2)
INFO Memory statistics tenant=2025/boston peak=487.3 MiB current=412.8 MiB limit=512.0 MiB utilization=95.2% failcnt=0 oomKills=0

# Tenant shutdown statistics (cgroups v1)
INFO Memory statistics tenant=2025/newyork peak=623.1 MiB current=598.2 MiB limit=768.0 MiB utilization=81.1% failcnt=2 oomKills=0
```

**Statistics provided:**
- **Peak usage**: Maximum memory used since Navigator started (or cgroup creation)
- **Current usage**: Memory in use at shutdown time
- **Limit**: Configured memory limit for this tenant
- **Utilization**: Peak usage as percentage of limit
- **Failcnt**: Number of times the memory limit was hit (cgroups v1 only)
- **OOM kills**: Number of times kernel killed the tenant for exceeding limit

**Use cases:**
- Identify right-sizing opportunities (consistently low utilization = reduce limit)
- Detect memory growth patterns (increasing peak usage over time)
- Validate capacity planning (ensure tenants stay within limits)
- Track memory efficiency across tenant restarts

### Checking Memory Usage

On Linux, query cgroup memory stats:

```bash
# Current memory usage
cat /sys/fs/cgroup/navigator/2025_boston/memory.current

# Memory limit
cat /sys/fs/cgroup/navigator/2025_boston/memory.max

# OOM kill count
grep oom_kill /sys/fs/cgroup/navigator/2025_boston/memory.events
```

### OOM Statistics

Navigator tracks OOM kills per tenant:

- **OOMCount**: Total number of OOM kills for this tenant
- **LastOOMTime**: Timestamp of most recent OOM kill

**Use cases:**
- Identify tenants that need higher limits
- Detect memory leaks in tenant code
- Track capacity planning effectiveness

## Fly.io Deployment

### Firecracker VMs

Fly.io uses Firecracker VMs, not Docker containers, which simplifies cgroup setup:

- **Direct kernel access**: No Docker nesting issues
- **Full cgroups v2**: Native support in Debian Trixie
- **Root execution**: Navigator runs as root in VM

See [Fly.io's Docker without Docker](https://fly.io/blog/docker-without-docker/) for architecture details.

### Example Configuration

```yaml
# navigator.yml for Fly.io
applications:
  pools:
    default_memory_limit: "512M"
    user: "rails"
    group: "rails"

  tenants:
    - name: 2025/boston
      path: /2025/boston/
      memory_limit: "512M"

    - name: 2025/newyork
      path: /2025/newyork/
      memory_limit: "768M"
```

## Kamal Deployment

For Kamal (Docker-based deployments), enable privileged mode:

```yaml
# config/deploy.yml
service: myapp

image: myapp/navigator

servers:
  web:
    hosts:
      - 192.0.2.10
    options:
      privileged: true        # Required for cgroup access
      cgroupns: host          # Use host cgroup namespace
```

**Limitations:**
- Some hosting providers restrict privileged containers
- Cloud platforms (AWS ECS, Google Cloud Run) may not allow `--privileged`
- VPS and bare metal deployments typically support privileged mode

## Troubleshooting

### Memory Limits Not Working

Check if Navigator is running as root:

```bash
# Check Navigator process user
ps aux | grep navigator

# Should show:
# root  1234  ... /usr/local/bin/navigator
```

If not running as root:

```bash
# Run Navigator as root
sudo /usr/local/bin/navigator /etc/navigator/navigator.yml

# Or configure systemd to run as root
# /etc/systemd/system/navigator.service
[Service]
User=root
```

### Cgroup Not Created

Check which cgroups version is available:

```bash
# Check cgroup version
mount | grep cgroup

# cgroups v2 (preferred):
# cgroup2 on /sys/fs/cgroup type cgroup2 (rw,nosuid,nodev,noexec,relatime)

# cgroups v1 (supported):
# tmpfs on /sys/fs/cgroup type tmpfs (ro,nosuid,nodev,noexec,mode=755)
# cgroup on /sys/fs/cgroup/memory type cgroup (rw,nosuid,nodev,noexec,relatime,memory)
```

Navigator automatically detects and uses the available version. Both v1 and v2 are fully supported.

**Preference for v2**: While both versions work, cgroups v2 provides better resource control and is the modern standard. Consider upgrading to Ubuntu 22.04+, Debian 12+, or equivalent for v2 support.

### Permission Denied Errors

Check directory permissions:

```bash
# Navigator needs write access to /sys/fs/cgroup
ls -la /sys/fs/cgroup

# Should be writable by root
drwxr-xr-x 2 root root ... /sys/fs/cgroup
```

### User/Group Not Found

Verify user exists:

```bash
# Check if rails user exists
id rails

# Create user if needed
sudo useradd -r -s /bin/false rails
```

### Repeated OOM Kills

If a tenant repeatedly hits memory limit:

1. **Increase limit**:
   ```yaml
   tenants:
     - name: problematic-tenant
       memory_limit: "768M"  # Increase from 512M
   ```

2. **Investigate memory usage**:
   ```bash
   # Monitor tenant memory
   watch cat /sys/fs/cgroup/navigator/problematic-tenant/memory.current
   ```

3. **Check for memory leaks**:
   ```ruby
   # Use memory profiling tools for Rails
   # Add to Gemfile:
   gem 'memory_profiler'
   gem 'derailed_benchmarks'

   # Profile memory in production:
   # bundle exec derailed bundle:mem
   # bundle exec derailed exec perf:mem
   ```

## Security Considerations

### Process Isolation

Running tenants as non-root users provides defense-in-depth:

- **Principle of least privilege**: Tenants run with minimal permissions
- **Filesystem isolation**: Limited access to system files
- **Process isolation**: Cannot signal or debug other processes

### Root Requirement

Navigator must run as root to:

1. Create and manage cgroups (requires `CAP_SYS_ADMIN`)
2. Set process credentials via `setuid`/`setgid`

**Mitigation:**
- Tenant code runs as non-root user (e.g., `rails`)
- Navigator only uses root for process management
- No tenant code executes with root privileges

## Best Practices

### Start Conservative

Begin with default limits and adjust based on actual usage:

```yaml
applications:
  pools:
    default_memory_limit: "512M"  # Start here

  # Monitor OOM events, then adjust:
  tenants:
    - name: high-usage-tenant
      memory_limit: "768M"  # Increase if needed
```

### Monitor OOM Events

Set up alerting for repeated OOM kills:

```bash
# Count OOM kills per tenant
for dir in /sys/fs/cgroup/navigator/*/; do
  tenant=$(basename "$dir")
  oom_count=$(grep oom_kill "$dir/memory.events" | awk '{print $2}')
  echo "$tenant: $oom_count OOM kills"
done
```

### Plan for Growth

Leave headroom for tenant growth:

```yaml
# For a 2GB machine:
applications:
  pools:
    default_memory_limit: "512M"  # Conservative

# Capacity:
# - 3 tenants × 512M = 1,536M
# - System overhead: ~500M
# - Total: ~2GB (no room for spikes)

# Better:
applications:
  pools:
    default_memory_limit: "384M"  # More headroom

# Capacity:
# - 4 tenants × 384M = 1,536M
# - System overhead: ~500M
# - Total: ~2GB with better burst capacity
```

### Test Limits

Verify limits work before production deployment:

```bash
# On a test machine, trigger OOM:
# 1. Set low limit (e.g., 128M)
# 2. Load tenant that uses >128M
# 3. Verify OOM kill and restart

# Check logs for:
# ERROR Tenant OOM killed by kernel
```

## Implementation Details

### Cgroup Hierarchy

Navigator creates cgroups under `/sys/fs/cgroup/navigator/`:

```
/sys/fs/cgroup/
├── navigator/                    # Navigator's top-level cgroup
│   ├── 2025_boston/             # Tenant cgroup (sanitized name)
│   │   ├── cgroup.procs         # PIDs in this cgroup
│   │   ├── memory.max           # Memory limit (bytes)
│   │   ├── memory.current       # Current usage (bytes)
│   │   └── memory.events        # OOM event counters
│   ├── 2025_newyork/
│   └── ...
```

**Name sanitization**: Slashes and special characters replaced with underscores (`2025/boston` → `2025_boston`)

### Memory Controller

Navigator enables the memory controller via `cgroup.subtree_control`:

```bash
# Navigator writes:
echo "+memory" > /sys/fs/cgroup/navigator/cgroup.subtree_control
```

This allows child cgroups (tenants) to use memory limiting.

### Process Assignment

After starting a tenant process, Navigator adds it to the cgroup:

```go
// Pseudocode
cmd.Start()                                    // Start process
pid := cmd.Process.Pid                         // Get PID
os.WriteFile(cgroupPath + "/cgroup.procs", pid) // Add to cgroup
```

All child processes inherit the cgroup and memory limit.

## See Also

- [YAML Configuration Reference](../configuration/yaml-reference.md)
- [Process Management](process-management.md)
- [Capacity Planning Blog Post](https://intertwingly.net/blog/2025/10/10/Multi-Tenant-Multi-Region-Web-Applications)
- [Fly.io Machine Sizing](https://fly.io/docs/about/pricing/#machines)
