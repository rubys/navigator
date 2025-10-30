package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldReloadConfig(t *testing.T) {
	// Create temporary config files
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yml")
	if err := os.WriteFile(configPath, []byte("test: config\n"), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	differentConfigPath := filepath.Join(tmpDir, "different.yml")
	if err := os.WriteFile(differentConfigPath, []byte("test: different\n"), 0644); err != nil {
		t.Fatalf("Failed to create different config: %v", err)
	}

	tests := []struct {
		name              string
		reloadConfigPath  string
		currentConfigPath string
		modifyFile        bool
		modifyDelay       time.Duration
		wantReload        bool
		wantReason        string
	}{
		{
			name:              "No reload_config specified",
			reloadConfigPath:  "",
			currentConfigPath: configPath,
			wantReload:        false,
		},
		{
			name:              "Different config file",
			reloadConfigPath:  differentConfigPath,
			currentConfigPath: configPath,
			wantReload:        true,
			wantReason:        "different config file",
		},
		{
			name:              "Same config file, not modified",
			reloadConfigPath:  configPath,
			currentConfigPath: configPath,
			modifyFile:        false,
			wantReload:        false,
		},
		{
			name:              "Same config file, modified during execution",
			reloadConfigPath:  configPath,
			currentConfigPath: configPath,
			modifyFile:        true,
			modifyDelay:       50 * time.Millisecond,
			wantReload:        true,
			wantReason:        "config file modified",
		},
		{
			name:              "Same config file, modified before execution",
			reloadConfigPath:  configPath,
			currentConfigPath: configPath,
			modifyFile:        false,
			wantReload:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record start time
			startTime := time.Now()

			// Simulate command execution time
			if tt.modifyFile {
				time.Sleep(tt.modifyDelay)
				// Touch the file to update mtime
				if err := os.WriteFile(configPath, []byte("test: updated\n"), 0644); err != nil {
					t.Fatalf("Failed to modify config: %v", err)
				}
			}

			// Check reload decision
			decision := ShouldReloadConfig(tt.reloadConfigPath, tt.currentConfigPath, startTime)

			if decision.ShouldReload != tt.wantReload {
				t.Errorf("ShouldReload = %v, want %v", decision.ShouldReload, tt.wantReload)
			}

			if tt.wantReload && decision.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", decision.Reason, tt.wantReason)
			}

			if tt.wantReload && decision.NewConfigFile != tt.reloadConfigPath {
				t.Errorf("NewConfigFile = %q, want %q", decision.NewConfigFile, tt.reloadConfigPath)
			}
		})
	}
}

func TestShouldReloadConfig_NonExistentFile(t *testing.T) {
	startTime := time.Now()
	decision := ShouldReloadConfig("/nonexistent/config.yml", "/current/config.yml", startTime)

	// Should NOT reload if target file doesn't exist (prevents errors)
	if decision.ShouldReload {
		t.Error("Expected NO reload when target config file doesn't exist")
	}
}

func TestShouldReloadConfig_UnreadableFile(t *testing.T) {
	// Create config with same path for current and reload
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yml")
	if err := os.WriteFile(configPath, []byte("test: config\n"), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	startTime := time.Now()

	// Make file unreadable (Unix only)
	if err := os.Chmod(configPath, 0000); err != nil {
		t.Skip("Cannot change file permissions on this system")
	}
	defer func() {
		_ = os.Chmod(configPath, 0644) // Restore permissions
	}()

	decision := ShouldReloadConfig(configPath, configPath, startTime)

	// Should not reload if file can't be stat'd
	if decision.ShouldReload {
		t.Error("Expected no reload for unreadable file")
	}
}
