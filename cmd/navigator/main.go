package main

import (
	"github.com/rubys/navigator/internal/cli"
)

// Version information - set during build with -ldflags
var (
	version   = "dev"      // Version number (e.g., "v0.2.0")
	buildDate = "unknown"  // Build date (RFC3339 format)
	gitCommit = "unknown"  // Git commit hash
)

func main() {
	// Set version information for CLI
	cli.SetVersionInfo(version, buildDate, gitCommit)
	cli.Execute()
}
