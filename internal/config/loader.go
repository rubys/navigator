package config

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	slog.Debug("Loading YAML configuration")
	return ParseYAML(content)
}

// ParseYAML parses the new YAML configuration format
func ParseYAML(content []byte) (*Config, error) {
	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(content, &yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Use the new parser to convert YAML config to internal Config structure
	parser := NewConfigParser(&yamlConfig)
	return parser.Parse()
}

// UpdateConfig updates configuration dynamically
func UpdateConfig(currentConfig *Config, newConfig *Config) {
	currentConfig.LocationConfigMutex.Lock()
	defer currentConfig.LocationConfigMutex.Unlock()

	// Update server configuration
	currentConfig.Server = newConfig.Server
	currentConfig.Static = newConfig.Static
	currentConfig.Routes = newConfig.Routes
	currentConfig.Applications = newConfig.Applications
	currentConfig.ManagedProcesses = newConfig.ManagedProcesses
	currentConfig.Logging = newConfig.Logging
	currentConfig.Hooks = newConfig.Hooks

	slog.Info("Configuration updated successfully")
}
