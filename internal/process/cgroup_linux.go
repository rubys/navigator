//go:build linux

package process

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"log/slog"
)

const cgroupRoot = "/sys/fs/cgroup"

// SetupCgroupMemoryLimit creates a cgroup and sets memory limit for a tenant.
// Returns cgroup path or empty string if not running as root or if limit is 0.
// On Linux, this uses cgroups v2 to enforce memory limits.
func SetupCgroupMemoryLimit(tenantName string, limitBytes int64) (string, error) {
	if limitBytes == 0 {
		return "", nil // No limit requested
	}

	if os.Geteuid() != 0 {
		slog.Debug("Not running as root, skipping cgroup setup",
			"tenant", tenantName)
		return "", nil
	}

	// Use default name "app" if tenant name is empty (framework mode)
	cgroupName := tenantName
	if cgroupName == "" {
		cgroupName = "app"
	}

	// Create tenant-specific cgroup under navigator/
	cgroupPath := filepath.Join(cgroupRoot, "navigator", sanitizeCgroupName(cgroupName))

	// Create cgroup directory
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create cgroup: %w", err)
	}

	// Enable memory controller if not already enabled
	if err := enableMemoryController(cgroupPath); err != nil {
		slog.Warn("Failed to enable memory controller, continuing anyway",
			"tenant", tenantName,
			"error", err)
	}

	// Write memory.max limit
	memMaxPath := filepath.Join(cgroupPath, "memory.max")
	if err := os.WriteFile(memMaxPath, []byte(strconv.FormatInt(limitBytes, 10)), 0644); err != nil {
		return "", fmt.Errorf("failed to set memory.max: %w", err)
	}

	slog.Info("Memory limit configured for tenant",
		"tenant", tenantName,
		"limit", formatBytes(limitBytes),
		"cgroup", cgroupPath)

	return cgroupPath, nil
}

// AddProcessToCgroup moves a PID into the specified cgroup
func AddProcessToCgroup(cgroupPath string, pid int) error {
	if cgroupPath == "" {
		return nil // No cgroup configured
	}

	procsPath := filepath.Join(cgroupPath, "cgroup.procs")
	if err := os.WriteFile(procsPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	slog.Debug("Process added to cgroup",
		"pid", pid,
		"cgroup", cgroupPath)

	return nil
}

// CleanupCgroup removes a tenant's cgroup directory
// This should only be called when the cgroup is empty (no processes)
func CleanupCgroup(tenantName string) error {
	// Use default name "app" if tenant name is empty (framework mode)
	cgroupName := tenantName
	if cgroupName == "" {
		cgroupName = "app"
	}

	cgroupPath := filepath.Join(cgroupRoot, "navigator", sanitizeCgroupName(cgroupName))

	if err := os.Remove(cgroupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cgroup: %w", err)
	}

	slog.Debug("Cgroup removed",
		"tenant", tenantName,
		"cgroup", cgroupPath)

	return nil
}

// IsOOMKill checks if a process exit was due to OOM kill
func IsOOMKill(cgroupPath string) bool {
	if cgroupPath == "" {
		return false
	}

	// Check memory.events for oom_kill counter
	eventsPath := filepath.Join(cgroupPath, "memory.events")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		return false
	}

	// Look for "oom_kill N" where N > 0
	// Format: "oom_kill 1" or "oom_kill 2", etc.
	re := regexp.MustCompile(`oom_kill\s+(\d+)`)
	matches := re.FindSubmatch(data)
	if len(matches) < 2 {
		return false
	}

	count, _ := strconv.Atoi(string(matches[1]))
	return count > 0
}

// GetOOMKillCount returns the number of OOM kills for a cgroup
func GetOOMKillCount(cgroupPath string) int {
	if cgroupPath == "" {
		return 0
	}

	eventsPath := filepath.Join(cgroupPath, "memory.events")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		return 0
	}

	re := regexp.MustCompile(`oom_kill\s+(\d+)`)
	matches := re.FindSubmatch(data)
	if len(matches) < 2 {
		return 0
	}

	count, _ := strconv.Atoi(string(matches[1]))
	return count
}

// GetMemoryUsage returns current memory usage for a cgroup in bytes
func GetMemoryUsage(cgroupPath string) int64 {
	if cgroupPath == "" {
		return 0
	}

	currentPath := filepath.Join(cgroupPath, "memory.current")
	data, err := os.ReadFile(currentPath)
	if err != nil {
		return 0
	}

	usage, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	return usage
}

// enableMemoryController enables the memory controller for a cgroup
func enableMemoryController(cgroupPath string) error {
	// Get parent cgroup path
	parentPath := filepath.Dir(cgroupPath)
	if parentPath == cgroupRoot {
		// Already at root, can't enable controller here
		return nil
	}

	// Write to parent's cgroup.subtree_control
	subtreeControlPath := filepath.Join(parentPath, "cgroup.subtree_control")

	// Read current controllers
	data, err := os.ReadFile(subtreeControlPath)
	if err != nil {
		return err
	}

	controllers := string(data)
	if strings.Contains(controllers, "memory") {
		// Already enabled
		return nil
	}

	// Enable memory controller
	if err := os.WriteFile(subtreeControlPath, []byte("+memory"), 0644); err != nil {
		// If it fails, it might already be enabled or we don't have permission
		// Don't treat this as fatal
		return err
	}

	return nil
}

// sanitizeCgroupName ensures the cgroup name is safe for use as a directory name
func sanitizeCgroupName(name string) string {
	// Replace slashes and other problematic characters with underscores
	re := regexp.MustCompile(`[/\\:*?"<>|]`)
	return re.ReplaceAllString(name, "_")
}

// ParseMemorySize parses memory size strings like "512M", "1G", "2048M"
func ParseMemorySize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))

	// Match number and optional unit
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([KMGT]I?B?)?$`)
	matches := re.FindStringSubmatch(sizeStr)
	if matches == nil {
		return 0, fmt.Errorf("invalid memory size format: %s", sizeStr)
	}

	// Parse number
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in memory size: %s", sizeStr)
	}

	// Parse unit
	unit := matches[2]
	if unit == "" {
		// No unit, assume bytes
		return int64(value), nil
	}

	// Normalize unit (remove 'I' and 'B' if present)
	unit = strings.TrimSuffix(unit, "IB")
	unit = strings.TrimSuffix(unit, "B")

	multiplier := int64(1)
	switch unit {
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit in memory size: %s", unit)
	}

	return int64(value * float64(multiplier)), nil
}
