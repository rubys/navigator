# Per-Tenant Memory Limits - Implementation Status

## Branch: `feature/per-tenant-memory-limits`

This feature adds per-tenant memory limiting using Linux cgroups v2, with graceful degradation on non-Linux platforms.

## ✅ Completed

### 1. Configuration Fields (commit: e3d803d)
- Added `default_memory_limit`, `user`, `group` to `Pools` config
- Added `memory_limit`, `user`, `group` to `Tenant` config
- Added `CgroupPath`, `MemoryLimit`, `OOMCount`, `LastOOMTime` to `WebApp` struct
- Updated `YAMLConfig` to support new fields

### 2. Platform-Specific Cgroup Management (commit: dca7e46)
- **Linux** (`cgroup_linux.go`):
  - `SetupCgroupMemoryLimit()` - Creates cgroups and sets memory limits
  - `AddProcessToCgroup()` - Moves processes into cgroups
  - `IsOOMKill()` / `GetOOMKillCount()` - Detects OOM kills
  - `GetMemoryUsage()` - Queries current memory usage
  - `ParseMemorySize()` - Parses sizes like "512M", "1G"

- **Non-Linux** (`cgroup_stub.go`):
  - No-op implementations that log and return safely

- **Credentials**:
  - `credentials_unix.go` - User/group lookup for Unix
  - `credentials_windows.go` - Stub for Windows

## 🚧 Remaining Work

### 3. Integrate into Process Startup
**File**: `internal/process/process_starter.go`

Need to modify `StartWebApp()` to:
```go
// 1. Get memory limit (tenant override or default)
memLimit := getMemoryLimit(tenant, config.Applications.Pools)

// 2. Parse memory limit
limitBytes, err := ParseMemorySize(memLimit)

// 3. Setup cgroup before starting process
cgroupPath, err := SetupCgroupMemoryLimit(tenant.Name, limitBytes)
app.CgroupPath = cgroupPath
app.MemoryLimit = limitBytes

// 4. Get user/group credentials
user := tenant.User
if user == "" {
    user = config.Applications.Pools.User
}
group := tenant.Group
if group == "" {
    group = config.Applications.Pools.Group
}
cred, err := GetUserCredentials(user, group)

// 5. Start process with credentials
cmd.SysProcAttr = &syscall.SysProcAttr{
    Credential: cred, // Run as specified user/group
}

// 6. After cmd.Start(), add to cgroup
AddProcessToCgroup(cgroupPath, cmd.Process.Pid)
```

### 4. OOM Detection and Cleanup
**File**: `internal/process/webapp.go`

Modify process monitoring to detect OOM kills and clean up:
```go
// In goroutine that waits for process
go func() {
    err := cmd.Wait()

    // Check if OOM killed
    if IsOOMKill(app.CgroupPath) {
        slog.Error("Tenant OOM killed by kernel",
            "tenant", tenantName,
            "limit", formatBytes(app.MemoryLimit))

        // Remove from registry
        // Next request will trigger restart via GetOrStartApp()
        m.mutex.Lock()
        delete(m.apps, tenantName)
        m.mutex.Unlock()

        // Note: Keep cgroup in place for reuse on restart
        // Only cleanup cgroup on Navigator shutdown
    }
}()
```

**Behavior**: OOM killed tenants are removed from the app registry but their cgroup remains. The next incoming request will trigger a restart via the normal `GetOrStartApp()` flow.

### 5. Cleanup on Shutdown
**File**: `internal/process/webapp.go`

In `CleanupWithContext()`:
```go
for tenantName, app := range m.apps {
    // ... existing cleanup ...

    // Cleanup cgroup on shutdown
    if app.CgroupPath != "" {
        CleanupCgroup(tenantName)
    }
}
```

### 6. Update Documentation

#### A. YAML Reference (`docs/configuration/yaml-reference.md`)
Add to `applications.pools`:
```yaml
default_memory_limit: "512M"  # Default memory limit per tenant (Linux only)
user: "rails"                 # Default user to run tenants as
group: "rails"                # Default group to run tenants as
```

Add to `applications.tenants[]`:
```yaml
memory_limit: "1G"    # Override memory limit for this tenant
user: "app"           # Override user for this tenant
group: "app"          # Override group for this tenant
```

#### B. Feature Documentation (`docs/features/memory-limits.md`)
Create new page explaining:
- What per-tenant memory limits are
- Linux cgroups v2 requirement
- Configuration examples
- OOM kill behavior and auto-restart
- Monitoring OOM events
- Platform support (Linux vs others)

### 7. Tests
**File**: `internal/process/cgroup_test.go`

Test cases needed:
- `TestParseMemorySize()` - Parse various formats
- `TestSetupCgroupMemoryLimit()` - Mock cgroup creation
- `TestIsOOMKill()` - Parse memory.events
- `TestGetUserCredentials()` - User/group lookup

## Configuration Example

```yaml
applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000
    default_memory_limit: "512M"  # Default for all tenants
    user: "rails"                 # Run as rails user
    group: "rails"                # Run as rails group

  tenants:
    - name: 2025/boston
      root: /apps/boston
      memory_limit: "384M"  # Smaller event

    - name: 2025/newyork
      root: /apps/newyork
      memory_limit: "768M"  # Larger event

    - name: 2025/chicago
      root: /apps/chicago
      # Uses default (512M)
```

## Testing Plan

1. **Local Testing** (macOS):
   - Verify builds successfully
   - Verify stub functions are called
   - Verify no errors on non-Linux platform

2. **Fly.io Testing** (Linux/Trixie):
   - Deploy with memory limits configured
   - Verify cgroups are created
   - Trigger OOM condition (memory leak test)
   - Verify OOM kill and auto-restart
   - Check `memory.events` for OOM counters
   - Verify processes run as specified user

3. **Edge Cases**:
   - No memory limit specified (should work normally)
   - Invalid memory limit format
   - Non-root execution (should log and skip)
   - User doesn't exist
   - OOM restart loop (should stop after 3)

## Platform Support

| Platform | Memory Limits | User/Group | Status |
|----------|--------------|------------|--------|
| Linux (Debian Trixie) | ✅ Full cgroups v2 | ✅ syscall.Credential | Production ready |
| macOS | ⚠️ Logged, skipped | ⚠️ Logged, skipped | Graceful degradation |
| Windows | ⚠️ Logged, skipped | ⚠️ Not supported | Graceful degradation |

## Next Steps

1. Complete items 3-5 (integrate into process management)
2. Update documentation (items 6A-6B)
3. Write tests (item 7)
4. Test locally on macOS
5. Deploy to Fly.io for Linux testing
6. Merge to main after successful testing
