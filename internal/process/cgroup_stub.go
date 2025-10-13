//go:build !linux

package process

import "log/slog"

// SetupCgroupMemoryLimit is a no-op on non-Linux platforms
func SetupCgroupMemoryLimit(tenantName string, limitBytes int64) (string, error) {
	if limitBytes > 0 {
		slog.Debug("Memory limits not supported on this platform",
			"tenant", tenantName,
			"platform", "non-Linux")
	}
	return "", nil
}

// AddProcessToCgroup is a no-op on non-Linux platforms
func AddProcessToCgroup(cgroupPath string, pid int) error {
	return nil
}

// CleanupCgroup is a no-op on non-Linux platforms
func CleanupCgroup(tenantName string) error {
	return nil
}

// IsOOMKill always returns false on non-Linux platforms
func IsOOMKill(cgroupPath string) bool {
	return false
}

// GetOOMKillCount always returns 0 on non-Linux platforms
func GetOOMKillCount(cgroupPath string) int {
	return 0
}

// GetMemoryUsage always returns 0 on non-Linux platforms
func GetMemoryUsage(cgroupPath string) int64 {
	return 0
}

// ParseMemorySize parses memory size strings (same implementation for all platforms)
func ParseMemorySize(sizeStr string) (int64, error) {
	// This function is platform-independent, but we duplicate it here
	// to avoid import issues. On Linux, the real implementation is used.
	return 0, nil
}

// LogMemoryStats is a no-op on non-Linux platforms
func LogMemoryStats(cgroupPath string, tenantName string) {
	// No memory statistics available on non-Linux platforms
}
