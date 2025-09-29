package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/rubys/navigator/internal/config"
)

// GenerateRequestID generates a random request ID similar to nginx $request_id
func GenerateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ExtractTenantName extracts the tenant name from a URL path
// Examples: "/showcase/2025/livermore/district-showcase/" -> "livermore-district-showcase"
//
//	"/2025/adelaide/adelaide-combined/" -> "adelaide-combined"
func ExtractTenantName(path string) string {
	// Remove leading/trailing slashes and split by '/'
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	// Skip empty parts
	var validParts []string
	for _, part := range parts {
		if part != "" {
			validParts = append(validParts, part)
		}
	}

	if len(validParts) < 2 {
		return ""
	}

	// Pattern 1: /showcase/YEAR/TENANT1/TENANT2/...
	if validParts[0] == "showcase" && len(validParts) >= 4 {
		// Skip "showcase" and year, join tenant parts
		return strings.Join(validParts[2:4], "-")
	}

	// Pattern 2: /YEAR/TENANT1/TENANT2/...
	if len(validParts) >= 3 {
		// Skip year, join tenant parts
		return strings.Join(validParts[1:3], "-")
	}

	// Pattern 3: /YEAR/TENANT
	if len(validParts) >= 2 {
		// Skip year, return tenant
		return validParts[1]
	}

	return ""
}

// WritePIDFile writes the current process PID to a file
func WritePIDFile(pidFile string) error {
	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// RemovePIDFile removes the PID file
func RemovePIDFile(pidFile string) {
	os.Remove(pidFile)
}

// SendReloadSignal sends a HUP signal to the running navigator process
func SendReloadSignal(pidFile string) error {
	// Read PID from file
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("navigator is not running (PID file not found)")
		}
		return fmt.Errorf("failed to read PID file: %v", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return fmt.Errorf("invalid PID in file: %v", err)
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %v", pid, err)
	}

	// Send HUP signal
	if err := process.Signal(syscall.SIGHUP); err != nil {
		// Check if process exists
		if err.Error() == "os: process already finished" {
			// Clean up stale PID file
			RemovePIDFile(pidFile)
			return fmt.Errorf("navigator is not running (process %d not found)", pid)
		}
		return fmt.Errorf("failed to send signal to process %d: %v", pid, err)
	}

	return nil
}

// GetDefaultMaintenancePage returns the maintenance page content
func GetDefaultMaintenancePage() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>503 Service Temporarily Unavailable</title>
    <style>
        body {
            width: 35em;
            margin: 0 auto;
            font-family: Tahoma, Verdana, Arial, sans-serif;
        }
    </style>
</head>
<body>
    <h1>Service Temporarily Unavailable</h1>
    <p>The server is temporarily unable to service your request due to maintenance downtime or capacity problems. Please try again later.</p>
</body>
</html>`
}

// GetPidFilePath extracts the PID file path from tenant environment variables
func GetPidFilePath(tenant *config.Tenant) string {
	if tenant == nil || tenant.Env == nil {
		return ""
	}
	if pidfile, ok := tenant.Env["PIDFILE"]; ok {
		return pidfile
	}
	return ""
}
