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
// 2. The config file was modified since the last config load (after configLoadTime)
//
// Parameters:
//   - reloadConfigPath: Path specified in reload_config field (empty = no reload requested)
//   - currentConfigPath: Currently loaded config file path
//   - configLoadTime: When the current configuration was last loaded
//
// This is used by both hooks and CGI scripts to share consistent reload logic.
// Using configLoadTime (rather than command start time) ensures that changes made
// while the machine was suspended are detected when it resumes.
func ShouldReloadConfig(reloadConfigPath, currentConfigPath string, configLoadTime time.Time) ReloadDecision {
	// No reload requested
	if reloadConfigPath == "" {
		return ReloadDecision{ShouldReload: false}
	}

	// Check if config file path is different from current
	if reloadConfigPath != currentConfigPath {
		// Verify the new config file actually exists before attempting reload
		if _, err := os.Stat(reloadConfigPath); err != nil {
			slog.Error("Reload config file does not exist",
				"file", reloadConfigPath,
				"error", err)
			return ReloadDecision{ShouldReload: false}
		}

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

	if info.ModTime().After(configLoadTime) {
		slog.Info("Reload config file was modified since last load",
			"file", reloadConfigPath,
			"modTime", info.ModTime(),
			"configLoadTime", configLoadTime)
		return ReloadDecision{
			ShouldReload:  true,
			Reason:        "config file modified",
			NewConfigFile: reloadConfigPath,
		}
	}

	// Config file unchanged - no reload needed
	slog.Debug("Reload config skipped - file unchanged since last load",
		"file", reloadConfigPath,
		"modTime", info.ModTime(),
		"configLoadTime", configLoadTime)
	return ReloadDecision{ShouldReload: false}
}
