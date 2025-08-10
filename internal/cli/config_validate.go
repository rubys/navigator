package cli

import (
	"fmt"
	"os"

	showcasesConfig "github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logger"
	"github.com/spf13/cobra"
)

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long: `Validate the Navigator configuration file and display the resolved settings.

This command will:
  • Load configuration from file, environment variables, and flags
  • Validate all required fields and paths
  • Display the final resolved configuration
  • Check that showcases.yml can be loaded successfully

Examples:
  navigator config validate
  navigator config validate --config /etc/navigator/navigator.yaml
  navigator config validate --rails-root /path/to/rails/app`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := GetConfig()
		validateConfig(cfg)
	},
}

// configCmd represents the config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  "Commands for managing Navigator configuration files and settings.",
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)

	// Mark rails-root as required for config validate command
	configValidateCmd.MarkPersistentFlagRequired("rails-root")
}

// validateConfig validates and displays the current configuration
func validateConfig(cfg *Config) {
	// Initialize logger for validation output
	logger.Init(cfg.Logging.Level)

	fmt.Println("✓ Configuration loaded successfully")
	fmt.Printf("\nServer Configuration:\n")
	fmt.Printf("  Listen Address: %s\n", cfg.Server.Listen)
	fmt.Printf("  URL Prefix: %s\n", cfg.Server.URLPrefix)

	fmt.Printf("\nRails Configuration:\n")
	fmt.Printf("  Root Directory: %s\n", cfg.Rails.Root)
	fmt.Printf("  Showcases File: %s\n", cfg.GetShowcasesPath())
	fmt.Printf("  Database Path: %s\n", cfg.Rails.DbPath)
	fmt.Printf("  Storage Path: %s\n", cfg.Rails.Storage)

	fmt.Printf("\nProcess Manager Configuration:\n")
	fmt.Printf("  Max Puma Processes: %d\n", cfg.Manager.MaxPuma)
	fmt.Printf("  Idle Timeout: %v\n", cfg.Manager.IdleTimeout)

	fmt.Printf("\nAuthentication Configuration:\n")
	if cfg.Auth.HtpasswdFile != "" {
		fmt.Printf("  Htpasswd File: %s\n", cfg.Auth.HtpasswdFile)
		if _, err := os.Stat(cfg.Auth.HtpasswdFile); err != nil {
			fmt.Printf("  ⚠️  Warning: Htpasswd file does not exist\n")
		} else {
			fmt.Printf("  ✓ Htpasswd file exists\n")
		}
	} else {
		fmt.Printf("  Authentication: Disabled\n")
	}

	fmt.Printf("\nLogging Configuration:\n")
	fmt.Printf("  Log Level: %s\n", cfg.Logging.Level)

	// Validate Rails root directory exists
	if _, err := os.Stat(cfg.Rails.Root); err != nil {
		fmt.Printf("\n❌ Error: Rails root directory does not exist: %s\n", cfg.Rails.Root)
		os.Exit(1)
	}

	// Validate showcases.yml file exists and can be loaded
	showcasesPath := cfg.GetShowcasesPath()
	if _, err := os.Stat(showcasesPath); err != nil {
		fmt.Printf("\n❌ Error: Showcases file does not exist: %s\n", showcasesPath)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Rails root directory exists: %s\n", cfg.Rails.Root)

	// Try to load showcases to validate format
	showcases, err := showcasesConfig.LoadShowcases(showcasesPath)
	if err != nil {
		fmt.Printf("❌ Error loading showcases file: %v\n", err)
		os.Exit(1)
	}

	tenants := showcases.GetAllTenants()
	fmt.Printf("✓ Showcases file loaded successfully: %d tenants configured\n", len(tenants))

	// Display tenant information
	if len(tenants) > 0 {
		fmt.Printf("\nConfigured Tenants:\n")
		for _, tenant := range tenants {
			fmt.Printf("  • %s (scope: %s)\n", tenant.Label, tenant.Scope)
		}
	}

	fmt.Printf("\n✅ Configuration validation completed successfully\n")
}
