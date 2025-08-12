package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	// Test that root command has core expected subcommands
	// Note: completion and help are added automatically by cobra
	expectedCommands := []string{"serve", "config", "version"}
	
	actualCommands := []string{}
	for _, cmd := range rootCmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}

	for _, expected := range expectedCommands {
		found := false
		for _, actual := range actualCommands {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command '%s' not found in root command. Available: %v", expected, actualCommands)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	// Test version command structure
	if versionCmd.Use != "version" {
		t.Errorf("Expected command name 'version', got '%s'", versionCmd.Use)
	}
	
	if versionCmd.Short == "" {
		t.Error("Version command should have a short description")
	}
	
	// Test that version command has a run function
	if versionCmd.Run == nil {
		t.Error("Version command should have a Run function")
	}
	
	// We can't easily test the actual output since it goes to stdout
	// and the command might be writing directly to fmt.Print functions
	// This is sufficient to verify the command structure
}

func TestServeCommandFlags(t *testing.T) {
	// Test that serve command has all expected flags
	expectedFlags := []string{
		"root",
		"listen", 
		"url-prefix",
		"max-puma",
		"idle-timeout",
		"htpasswd",
		"log-level",
		"showcases",
		"db-path",
		"storage",
		"config",
	}

	for _, flagName := range expectedFlags {
		flag := serveCmd.Flags().Lookup(flagName)
		if flag == nil {
			// Check persistent flags if not found in local flags
			flag = serveCmd.PersistentFlags().Lookup(flagName)
		}
		if flag == nil {
			// Check inherited flags from root
			flag = rootCmd.PersistentFlags().Lookup(flagName)
		}
		
		if flag == nil {
			t.Errorf("Expected flag '%s' not found in serve command", flagName)
		}
	}
}

func TestConfigValidateCommandFlags(t *testing.T) {
	// Test that config validate command has required flags
	expectedFlags := []string{
		"root",
		"config",
	}

	for _, flagName := range expectedFlags {
		flag := configValidateCmd.Flags().Lookup(flagName)
		if flag == nil {
			// Check persistent flags
			flag = configValidateCmd.PersistentFlags().Lookup(flagName)
		}
		if flag == nil {
			// Check inherited flags from root
			flag = rootCmd.PersistentFlags().Lookup(flagName)
		}

		if flag == nil {
			t.Errorf("Expected flag '%s' not found in config validate command", flagName)
		}
	}
}

func TestCommandHelp(t *testing.T) {
	// Test that commands have help text
	commands := []*cobra.Command{rootCmd, serveCmd, versionCmd, configCmd, configValidateCmd}

	for _, cmd := range commands {
		if cmd.Short == "" {
			t.Errorf("Command '%s' missing short description", cmd.Name())
		}
		if cmd.Long == "" && cmd.Name() != "version" { // version cmd has minimal help
			t.Errorf("Command '%s' missing long description", cmd.Name())
		}
	}
}

func TestCommandExamples(t *testing.T) {
	// Test that serve command has examples in help text
	if !strings.Contains(serveCmd.Long, "Examples:") {
		t.Error("Serve command should include examples in help text")
	}
	if !strings.Contains(configValidateCmd.Long, "Examples:") {
		t.Error("Config validate command should include examples in help text")
	}
}

func TestRootCommandDescription(t *testing.T) {
	// Test that root command has proper description
	expectedPhrases := []string{
		"Navigator",
		"multi-tenant Rails",
		"HTTP/2",
		"intelligent request routing",
		"process management",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(rootCmd.Long, phrase) {
			t.Errorf("Root command description should contain '%s'", phrase)
		}
	}
}

// Test flag binding to viper configuration
func TestFlagBinding(t *testing.T) {
	// This test ensures flags are properly bound to viper
	// We test this indirectly by checking if the bind functions exist
	// and don't panic when called
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("bindFlags() panicked: %v", r)
		}
	}()
	
	// Call bindFlags to ensure it doesn't panic
	bindFlags()
}

func TestGetConfigInitialization(t *testing.T) {
	// Test that GetConfig initializes config if nil
	// Reset global config to nil for testing
	originalConfig := config
	config = nil
	
	defer func() {
		// Restore original config
		config = originalConfig
	}()

	// No longer need to set NAVIGATOR_RAILS_ROOT since root has a default value
	// The default '.' will be used

	// GetConfig should initialize config without panicking
	testConfig := GetConfig()
	if testConfig == nil {
		t.Error("GetConfig() returned nil config")
	}
}