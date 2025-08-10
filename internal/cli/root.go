package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	config  *Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "navigator",
	Short: "Navigator is a modern Go-based web server for multi-tenant Rails applications",
	Long: `Navigator is a high-performance HTTP/2 proxy server that replaces nginx/Passenger 
for multi-tenant Rails applications. It provides intelligent request routing, dynamic 
process management, authentication, HTTP caching, and automatic process recovery.

Features:
  • Multi-tenant routing based on URL patterns
  • Dynamic Puma process management  
  • HTTP/2 support with automatic upgrades
  • Built-in HTTP caching with smart TTL
  • Authentication via htpasswd files
  • Automatic process recovery on failures
  • Structured JSON logging`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Don't automatically initialize config for all commands
	// Let individual commands initialize config as needed

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is navigator.yaml)")

	// Server flags
	rootCmd.PersistentFlags().String("listen", ":3000", "address to listen on")
	rootCmd.PersistentFlags().String("url-prefix", "/showcase", "URL prefix to strip from requests")

	// Rails flags
	rootCmd.PersistentFlags().String("rails-root", "", "Rails application root directory (required)")
	rootCmd.PersistentFlags().String("showcases", "config/tenant/showcases.yml", "path to showcases.yml relative to rails-root")
	rootCmd.PersistentFlags().String("db-path", "db", "database directory path")
	rootCmd.PersistentFlags().String("storage", "storage", "storage directory path")

	// Manager flags
	rootCmd.PersistentFlags().Int("max-puma", 10, "maximum number of concurrent Puma processes")
	rootCmd.PersistentFlags().Duration("idle-timeout", 5*time.Minute, "idle timeout before stopping Puma process")

	// Auth flags
	rootCmd.PersistentFlags().String("htpasswd", "", "path to htpasswd file")

	// Logging flags
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")

	// Bind flags to viper
	bindFlags()

	// Don't mark flags as required at root level
	// Individual commands will validate required flags as needed
}

// bindFlags binds cobra flags to viper configuration
func bindFlags() {
	// Server flags
	viper.BindPFlag("server.listen", rootCmd.PersistentFlags().Lookup("listen"))
	viper.BindPFlag("server.url_prefix", rootCmd.PersistentFlags().Lookup("url-prefix"))

	// Rails flags
	viper.BindPFlag("rails.root", rootCmd.PersistentFlags().Lookup("rails-root"))
	viper.BindPFlag("rails.showcases", rootCmd.PersistentFlags().Lookup("showcases"))
	viper.BindPFlag("rails.db_path", rootCmd.PersistentFlags().Lookup("db-path"))
	viper.BindPFlag("rails.storage", rootCmd.PersistentFlags().Lookup("storage"))

	// Manager flags
	viper.BindPFlag("manager.max_puma", rootCmd.PersistentFlags().Lookup("max-puma"))
	viper.BindPFlag("manager.idle_timeout", rootCmd.PersistentFlags().Lookup("idle-timeout"))

	// Auth flags
	viper.BindPFlag("auth.htpasswd_file", rootCmd.PersistentFlags().Lookup("htpasswd"))

	// Logging flags
	viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level"))
}

// initConfig reads in config file and ENV variables
func initConfig() {
	var err error
	config, err = LoadConfig(cfgFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}
}

// GetConfig returns the loaded configuration, loading it if necessary
func GetConfig() *Config {
	if config == nil {
		initConfig()
	}
	return config
}
