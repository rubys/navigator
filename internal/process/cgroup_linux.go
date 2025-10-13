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

const (
	cgroupRoot   = "/sys/fs/cgroup"
	cgroupV1Root = "/sys/fs/cgroup/memory"
)

// cgroupVersion represents the detected cgroup version
type cgroupVersion int

const (
	cgroupNone cgroupVersion = iota
	cgroupV1
	cgroupV2
)

var (
	detectedVersion cgroupVersion
	cgroupV2Root    string // Dynamically determined cgroup v2 root path
)

// detectCgroupVersion determines which cgroup version is available and usable
func detectCgroupVersion() cgroupVersion {
	// Check for cgroup v2 with usable memory controller
	// Try both common mount points: /sys/fs/cgroup (unified) and /sys/fs/cgroup/unified
	v2Paths := []string{cgroupRoot, filepath.Join(cgroupRoot, "unified")}

	for _, v2Path := range v2Paths {
		controllersPath := filepath.Join(v2Path, "cgroup.controllers")
		if data, err := os.ReadFile(controllersPath); err == nil {
			controllers := strings.TrimSpace(string(data))

			// If cgroup.controllers is empty, v2 is not usable (hybrid mode with v1 active)
			if controllers == "" {
				slog.Debug("cgroup v2 path exists but cgroup.controllers is empty (v1 active)",
					"path", v2Path)
				continue
			}

			if strings.Contains(controllers, "memory") {
				// Memory controller is listed in cgroup.controllers, but we need to check
				// if it's enabled in cgroup.subtree_control or can be enabled
				subtreeControlPath := filepath.Join(v2Path, "cgroup.subtree_control")
				if subtreeData, err := os.ReadFile(subtreeControlPath); err == nil {
					subtreeControllers := strings.TrimSpace(string(subtreeData))

					// If memory is already enabled in subtree_control, v2 is usable
					if strings.Contains(subtreeControllers, "memory") {
						cgroupV2Root = v2Path
						slog.Debug("cgroup v2 memory controller enabled in subtree_control",
							"path", v2Path,
							"subtree_control", subtreeControllers)
						return cgroupV2
					}

					// Memory is not enabled. In hybrid mode (v1 active), subtree_control is empty
					// and we cannot enable controllers. Skip this v2 path and fall back to v1.
					slog.Debug("cgroup v2 memory in controllers but not in subtree_control (v1 active)",
						"path", v2Path,
						"controllers", controllers,
						"subtree_control", subtreeControllers)
					continue
				}
				// Could not read subtree_control, skip this v2 path
				slog.Debug("cgroup v2 memory in controllers but cannot read subtree_control",
					"path", v2Path,
					"controllers", controllers)
				continue
			}

			slog.Debug("cgroup v2 exists but no memory controller in cgroup.controllers",
				"path", v2Path,
				"controllers", controllers)
		}
	}

	// Check for cgroup v1 memory controller
	if _, err := os.Stat(cgroupV1Root); err == nil {
		// Verify we can read memory settings
		if _, err := os.ReadFile(filepath.Join(cgroupV1Root, "memory.limit_in_bytes")); err == nil {
			slog.Debug("Detected cgroup v1 with memory controller")
			return cgroupV1
		}
		slog.Debug("cgroup v1 path exists but cannot read memory.limit_in_bytes")
	}

	slog.Debug("No usable cgroup memory controller detected")
	return cgroupNone
}

// SetupCgroupMemoryLimit creates a cgroup and sets memory limit for a tenant.
// Returns cgroup path or empty string if not running as root or if limit is 0.
// On Linux, this uses cgroups v2 or v1 depending on what's available.
func SetupCgroupMemoryLimit(tenantName string, limitBytes int64) (string, error) {
	if limitBytes == 0 {
		return "", nil // No limit requested
	}

	if os.Geteuid() != 0 {
		slog.Debug("Not running as root, skipping cgroup setup",
			"tenant", tenantName)
		return "", nil
	}

	// Detect cgroup version on first use
	if detectedVersion == cgroupNone {
		detectedVersion = detectCgroupVersion()
		if detectedVersion == cgroupNone {
			slog.Warn("No usable cgroup memory controller found, memory limits will be ignored",
				"tenant", tenantName)
			return "", nil
		}
	}

	// Use default name "app" if tenant name is empty (framework mode)
	cgroupName := tenantName
	if cgroupName == "" {
		cgroupName = "app"
	}
	cgroupName = sanitizeCgroupName(cgroupName)

	// Setup based on detected version
	switch detectedVersion {
	case cgroupV2:
		return setupCgroupV2(cgroupName, limitBytes, tenantName)
	case cgroupV1:
		return setupCgroupV1(cgroupName, limitBytes, tenantName)
	default:
		return "", nil
	}
}

// setupCgroupV2 configures memory limits using cgroups v2
func setupCgroupV2(cgroupName string, limitBytes int64, tenantName string) (string, error) {
	// Create parent navigator cgroup first
	navigatorPath := filepath.Join(cgroupV2Root, "navigator")
	if err := os.MkdirAll(navigatorPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create navigator cgroup: %w", err)
	}

	// Enable memory controller in root for navigator cgroup
	if err := enableMemoryControllerInParent(cgroupV2Root, "navigator"); err != nil {
		slog.Warn("Failed to enable memory controller in root, continuing anyway",
			"error", err)
	}

	// Create tenant-specific cgroup under navigator/
	cgroupPath := filepath.Join(navigatorPath, cgroupName)

	// Create cgroup directory
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create cgroup: %w", err)
	}

	// Enable memory controller for tenant cgroup
	if err := enableMemoryControllerInParent(navigatorPath, cgroupName); err != nil {
		slog.Warn("Failed to enable memory controller in navigator, continuing anyway",
			"tenant", tenantName,
			"error", err)
	}

	// Write memory.max limit
	memMaxPath := filepath.Join(cgroupPath, "memory.max")
	if err := os.WriteFile(memMaxPath, []byte(strconv.FormatInt(limitBytes, 10)), 0644); err != nil {
		return "", fmt.Errorf("failed to set memory.max: %w", err)
	}

	slog.Info("Memory limit configured for tenant (cgroup v2)",
		"tenant", tenantName,
		"limit", formatBytes(limitBytes),
		"cgroup", cgroupPath)

	return cgroupPath, nil
}

// setupCgroupV1 configures memory limits using cgroups v1
func setupCgroupV1(cgroupName string, limitBytes int64, tenantName string) (string, error) {
	// Create navigator parent cgroup
	navigatorPath := filepath.Join(cgroupV1Root, "navigator")
	if err := os.MkdirAll(navigatorPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create navigator cgroup: %w", err)
	}

	// Create tenant-specific cgroup under navigator/
	cgroupPath := filepath.Join(navigatorPath, cgroupName)

	// Create cgroup directory
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create cgroup: %w", err)
	}

	// Write memory.limit_in_bytes (v1 format)
	memLimitPath := filepath.Join(cgroupPath, "memory.limit_in_bytes")
	if err := os.WriteFile(memLimitPath, []byte(strconv.FormatInt(limitBytes, 10)), 0644); err != nil {
		return "", fmt.Errorf("failed to set memory.limit_in_bytes: %w", err)
	}

	slog.Info("Memory limit configured for tenant (cgroup v1)",
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
	cgroupName = sanitizeCgroupName(cgroupName)

	// Try to remove from both v1 and v2 locations (one will fail, that's ok)
	var lastErr error

	// Try v2 location
	cgroupV2Path := filepath.Join(cgroupV2Root, "navigator", cgroupName)
	if err := os.Remove(cgroupV2Path); err != nil && !os.IsNotExist(err) {
		lastErr = err
	} else if err == nil {
		slog.Debug("Cgroup removed (v2)",
			"tenant", tenantName,
			"cgroup", cgroupV2Path)
		return nil
	}

	// Try v1 location
	cgroupV1Path := filepath.Join(cgroupV1Root, "navigator", cgroupName)
	if err := os.Remove(cgroupV1Path); err != nil && !os.IsNotExist(err) {
		lastErr = err
	} else if err == nil {
		slog.Debug("Cgroup removed (v1)",
			"tenant", tenantName,
			"cgroup", cgroupV1Path)
		return nil
	}

	// If we got a real error (not just "doesn't exist"), return it
	if lastErr != nil {
		return fmt.Errorf("failed to remove cgroup: %w", lastErr)
	}

	return nil
}

// IsOOMKill checks if a process exit was due to OOM kill
func IsOOMKill(cgroupPath string) bool {
	if cgroupPath == "" {
		return false
	}

	// Try v2 format first (memory.events)
	eventsPath := filepath.Join(cgroupPath, "memory.events")
	if data, err := os.ReadFile(eventsPath); err == nil {
		// Look for "oom_kill N" where N > 0
		re := regexp.MustCompile(`oom_kill\s+(\d+)`)
		matches := re.FindSubmatch(data)
		if len(matches) >= 2 {
			count, _ := strconv.Atoi(string(matches[1]))
			return count > 0
		}
	}

	// Try v1 format (memory.oom_control)
	oomControlPath := filepath.Join(cgroupPath, "memory.oom_control")
	if data, err := os.ReadFile(oomControlPath); err == nil {
		// Look for "oom_kill N" or "under_oom 1"
		re := regexp.MustCompile(`oom_kill\s+(\d+)`)
		matches := re.FindSubmatch(data)
		if len(matches) >= 2 {
			count, _ := strconv.Atoi(string(matches[1]))
			return count > 0
		}
	}

	return false
}

// GetOOMKillCount returns the number of OOM kills for a cgroup
func GetOOMKillCount(cgroupPath string) int {
	if cgroupPath == "" {
		return 0
	}

	// Try v2 format first (memory.events)
	eventsPath := filepath.Join(cgroupPath, "memory.events")
	if data, err := os.ReadFile(eventsPath); err == nil {
		re := regexp.MustCompile(`oom_kill\s+(\d+)`)
		matches := re.FindSubmatch(data)
		if len(matches) >= 2 {
			count, _ := strconv.Atoi(string(matches[1]))
			return count
		}
	}

	// Try v1 format (memory.oom_control)
	oomControlPath := filepath.Join(cgroupPath, "memory.oom_control")
	if data, err := os.ReadFile(oomControlPath); err == nil {
		re := regexp.MustCompile(`oom_kill\s+(\d+)`)
		matches := re.FindSubmatch(data)
		if len(matches) >= 2 {
			count, _ := strconv.Atoi(string(matches[1]))
			return count
		}
	}

	return 0
}

// GetMemoryUsage returns current memory usage for a cgroup in bytes
func GetMemoryUsage(cgroupPath string) int64 {
	if cgroupPath == "" {
		return 0
	}

	// Try v2 format first (memory.current)
	currentPath := filepath.Join(cgroupPath, "memory.current")
	if data, err := os.ReadFile(currentPath); err == nil {
		usage, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		return usage
	}

	// Try v1 format (memory.usage_in_bytes)
	usagePath := filepath.Join(cgroupPath, "memory.usage_in_bytes")
	if data, err := os.ReadFile(usagePath); err == nil {
		usage, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		return usage
	}

	return 0
}

// enableMemoryControllerInParent enables the memory controller in parent for child cgroup
func enableMemoryControllerInParent(parentPath, childName string) error {
	// Write to parent's cgroup.subtree_control
	subtreeControlPath := filepath.Join(parentPath, "cgroup.subtree_control")

	// Read current controllers
	data, err := os.ReadFile(subtreeControlPath)
	if err != nil {
		return fmt.Errorf("failed to read subtree_control: %w", err)
	}

	controllers := string(data)
	if strings.Contains(controllers, "memory") {
		// Already enabled
		return nil
	}

	// Enable memory controller
	if err := os.WriteFile(subtreeControlPath, []byte("+memory"), 0644); err != nil {
		return fmt.Errorf("failed to enable memory controller: %w", err)
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
