package utils

import (
	"log/slog"
	"os"
	"time"
)

// ReloadDecision contains the result of checking whether config should be reloaded
type ReloadDecision struct {
	ShouldReload  bool
	Reason        string
	NewConfigFile string
}

// ShouldReloadConfig determines if configuration should be reloaded after a command execution
// Returns true if:
// 1. reloadConfigPath is different from currentConfigPath, OR
// 2. The config file was modified during command execution (after startTime)
//
// Parameters:
//   - reloadConfigPath: Path specified in reload_config field (empty = no reload requested)
//   - currentConfigPath: Currently loaded config file path
//   - startTime: When the command started executing
//
// This is used by both hooks and CGI scripts to share consistent reload logic.
func ShouldReloadConfig(reloadConfigPath, currentConfigPath string, startTime time.Time) ReloadDecision {
	// No reload requested
	if reloadConfigPath == "" {
		return ReloadDecision{ShouldReload: false}
	}

	// Check if config file path is different from current
	if reloadConfigPath != currentConfigPath {
		slog.Info("Reload config specifies different config file",
			"current", currentConfigPath,
			"new", reloadConfigPath)
		return ReloadDecision{
			ShouldReload:  true,
			Reason:        "different config file",
			NewConfigFile: reloadConfigPath,
		}
	}

	// Check if config file was modified during command execution
	info, err := os.Stat(reloadConfigPath)
	if err != nil {
		slog.Warn("Cannot stat reload_config file",
			"file", reloadConfigPath,
			"error", err)
		return ReloadDecision{ShouldReload: false}
	}

	if info.ModTime().After(startTime) {
		slog.Info("Reload config file was modified during execution",
			"file", reloadConfigPath,
			"modTime", info.ModTime(),
			"startTime", startTime)
		return ReloadDecision{
			ShouldReload:  true,
			Reason:        "config file modified",
			NewConfigFile: reloadConfigPath,
		}
	}

	// Config file unchanged - no reload needed
	slog.Debug("Reload config skipped - file unchanged",
		"file", reloadConfigPath,
		"modTime", info.ModTime(),
		"startTime", startTime)
	return ReloadDecision{ShouldReload: false}
}
