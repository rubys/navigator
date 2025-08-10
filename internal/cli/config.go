package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete Navigator configuration
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Rails   RailsConfig   `mapstructure:"rails"`
	Manager ManagerConfig `mapstructure:"manager"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Listen    string `mapstructure:"listen"`
	URLPrefix string `mapstructure:"url_prefix"`
}

// RailsConfig holds Rails application configuration
type RailsConfig struct {
	Root      string `mapstructure:"root"`
	Showcases string `mapstructure:"showcases"`
	DbPath    string `mapstructure:"db_path"`
	Storage   string `mapstructure:"storage"`
}

// ManagerConfig holds process manager configuration
type ManagerConfig struct {
	MaxPuma     int           `mapstructure:"max_puma"`
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	HtpasswdFile string `mapstructure:"htpasswd_file"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

// LoadConfig loads configuration from file, environment variables, and command line flags
func LoadConfig(configFile string) (*Config, error) {
	v := viper.GetViper() // Use global viper instance instead of creating new one

	// Set configuration defaults
	setDefaults(v)

	// Set environment variable settings
	v.SetEnvPrefix("NAVIGATOR")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Manually bind environment variables for proper recognition
	v.BindEnv("rails.root", "NAVIGATOR_RAILS_ROOT")
	v.BindEnv("logging.level", "NAVIGATOR_LOGGING_LEVEL") 
	v.BindEnv("server.listen", "NAVIGATOR_SERVER_LISTEN")
	v.BindEnv("server.url_prefix", "NAVIGATOR_SERVER_URL_PREFIX")
	v.BindEnv("manager.max_puma", "NAVIGATOR_MANAGER_MAX_PUMA")
	v.BindEnv("manager.idle_timeout", "NAVIGATOR_MANAGER_IDLE_TIMEOUT") 
	v.BindEnv("auth.htpasswd_file", "NAVIGATOR_AUTH_HTPASSWD_FILE")

	// Load configuration file if specified
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		// Try to find config file in common locations
		v.SetConfigName("navigator")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config") // Only look in project config directory
		v.AddConfigPath("/etc/navigator")

		// Read config file if found (ignore if not found or corrupted)
		if err := v.ReadInConfig(); err != nil {
			// Only return error if explicitly set config file, otherwise just warn and continue
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				// Config file found but has errors - warn but continue
				fmt.Printf("Warning: found config file but couldn't parse it: %v\n", err)
			}
		}
	}

	// Unmarshal configuration into struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate and resolve paths
	if err := config.validateAndResolvePaths(); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.listen", ":3000")
	v.SetDefault("server.url_prefix", "/showcase")

	// Rails defaults
	v.SetDefault("rails.showcases", "config/tenant/showcases.yml")
	v.SetDefault("rails.db_path", "db")
	v.SetDefault("rails.storage", "storage")

	// Manager defaults
	v.SetDefault("manager.max_puma", 10)
	v.SetDefault("manager.idle_timeout", "5m")

	// Logging defaults
	v.SetDefault("logging.level", "info")
}

// validateAndResolvePaths validates required fields and resolves relative paths
func (c *Config) validateAndResolvePaths() error {
	// Validate required fields
	if c.Rails.Root == "" {
		return fmt.Errorf("rails.root is required")
	}

	// Resolve Rails root to absolute path
	absRailsRoot, err := filepath.Abs(c.Rails.Root)
	if err != nil {
		return fmt.Errorf("error resolving rails.root path: %w", err)
	}
	c.Rails.Root = absRailsRoot

	// Resolve relative paths to be relative to Rails root
	if !filepath.IsAbs(c.Rails.DbPath) {
		c.Rails.DbPath = filepath.Join(absRailsRoot, c.Rails.DbPath)
	}
	if !filepath.IsAbs(c.Rails.Storage) {
		c.Rails.Storage = filepath.Join(absRailsRoot, c.Rails.Storage)
	}
	if c.Auth.HtpasswdFile != "" && !filepath.IsAbs(c.Auth.HtpasswdFile) {
		c.Auth.HtpasswdFile = filepath.Join(absRailsRoot, c.Auth.HtpasswdFile)
	}

	return nil
}

// GetShowcasesPath returns the full path to the showcases.yml file
func (c *Config) GetShowcasesPath() string {
	if filepath.IsAbs(c.Rails.Showcases) {
		return c.Rails.Showcases
	}
	return filepath.Join(c.Rails.Root, c.Rails.Showcases)
}
