package config

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

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

// substituteVars replaces template variables with tenant values
func substituteVars(template string, tenant *Tenant) string {
	result := template
	// Replace ${var} with values from the Var map
	if tenant.Var != nil {
		for key, value := range tenant.Var {
			// Convert interface{} to string
			strValue := ""
			switch v := value.(type) {
			case string:
				strValue = v
			default:
				strValue = fmt.Sprintf("%v", v)
			}
			result = strings.ReplaceAll(result, "${"+key+"}", strValue)
		}
	}
	return result
}

// ParseYAML parses the new YAML configuration format
func ParseYAML(content []byte) (*Config, error) {
	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(content, &yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert YAML config to internal Config structure
	config := &Config{}

	// Set server configuration
	config.Server.Hostname = yamlConfig.Server.Hostname
	config.Server.PublicDir = yamlConfig.Server.PublicDir
	config.Server.NamedHosts = yamlConfig.Server.NamedHosts
	config.Server.Root = yamlConfig.Server.Root
	config.Server.TryFiles = yamlConfig.Server.TryFiles
	config.Server.Authentication = yamlConfig.Server.Authentication
	config.Server.AuthExclude = yamlConfig.Server.AuthExclude

	// Parse auth section and map public_paths to AuthExclude (like original navigator)
	if yamlConfig.Auth.Enabled {
		config.Server.Authentication = yamlConfig.Auth.HTPasswd
		config.Server.AuthExclude = yamlConfig.Auth.PublicPaths
		// TODO: Handle exclude patterns if needed
	}

	// Parse listen port
	switch v := yamlConfig.Server.Listen.(type) {
	case int:
		config.Server.Listen = fmt.Sprintf("%d", v)
	case string:
		config.Server.Listen = v
	default:
		config.Server.Listen = "3000"
	}

	// Set idle configuration
	config.Server.Idle.Action = yamlConfig.Server.Idle.Action
	config.Server.Idle.Timeout = yamlConfig.Server.Idle.Timeout

	// Set sticky sessions configuration
	config.Server.StickySession.Enabled = yamlConfig.Server.StickySession.Enabled
	config.Server.StickySession.CookieName = yamlConfig.Server.StickySession.CookieName
	config.Server.StickySession.CookieMaxAge = yamlConfig.Server.StickySession.CookieMaxAge
	config.Server.StickySession.CookieSecure = yamlConfig.Server.StickySession.CookieSecure
	config.Server.StickySession.CookieHTTPOnly = yamlConfig.Server.StickySession.CookieHTTPOnly
	config.Server.StickySession.CookieSameSite = yamlConfig.Server.StickySession.CookieSameSite
	config.Server.StickySession.CookiePath = yamlConfig.Server.StickySession.CookiePath
	config.Server.StickySession.Paths = yamlConfig.Server.StickySession.Paths

	// Parse sticky session max age
	if config.Server.StickySession.CookieMaxAge != "" {
		if duration, err := time.ParseDuration(config.Server.StickySession.CookieMaxAge); err == nil {
			config.Server.StickySession.cookieMaxAge = duration
		}
	}

	// Convert rewrite rules from server section
	for _, rewrite := range yamlConfig.Server.Rewrites {
		if re, err := regexp.Compile(rewrite.Pattern); err == nil {
			config.Server.RewriteRules = append(config.Server.RewriteRules, RewriteRule{
				Pattern:     re,
				Replacement: rewrite.Replacement,
				Flag:        rewrite.Flag,
				Methods:     rewrite.Methods,
			})
		} else {
			slog.Warn("Invalid rewrite pattern", "pattern", rewrite.Pattern, "error", err)
		}
	}

	// Convert redirect rules from routes section (redirects with 302 status)
	for _, redirect := range yamlConfig.Routes.Redirects {
		if re, err := regexp.Compile(redirect.From); err == nil {
			config.Server.RewriteRules = append(config.Server.RewriteRules, RewriteRule{
				Pattern:     re,
				Replacement: redirect.To,
				Flag:        "redirect", // Use "redirect" flag for 302 redirects
			})
		} else {
			slog.Warn("Invalid routes redirect pattern", "from", redirect.From, "error", err)
		}
	}

	// Convert rewrite rules from routes section (internal rewrites)
	for _, rewrite := range yamlConfig.Routes.Rewrites {
		if re, err := regexp.Compile(rewrite.From); err == nil {
			config.Server.RewriteRules = append(config.Server.RewriteRules, RewriteRule{
				Pattern:     re,
				Replacement: rewrite.To,
				Flag:        "last", // Default to "last" for internal rewrites
			})
		} else {
			slog.Warn("Invalid routes rewrite pattern", "from", rewrite.From, "error", err)
		}
	}

	// Convert locations
	for _, loc := range yamlConfig.Locations {
		location := Location{
			Path:               loc.Path,
			PublicDir:          loc.PublicDir,
			TryFiles:           loc.TryFiles,
			ProxyPass:          loc.ProxyPass,
			ProxyMethod:        loc.ProxyMethod,
			ProxyExcludeMethod: loc.ProxyExcludeMethod,
			Alias:              loc.Alias,
		}

		// Convert location rewrite rules
		for _, rewrite := range loc.Rewrites {
			if re, err := regexp.Compile(rewrite.Pattern); err == nil {
				location.RewriteRules = append(location.RewriteRules, RewriteRule{
					Pattern:     re,
					Replacement: rewrite.Replacement,
					Flag:        rewrite.Flag,
					Methods:     rewrite.Methods,
				})
			}
		}

		config.Locations = append(config.Locations, location)
	}

	// Convert applications
	config.Applications.Pools.MaxSize = yamlConfig.Applications.Pools.MaxSize
	config.Applications.Pools.Timeout = yamlConfig.Applications.Pools.Timeout
	config.Applications.Pools.StartPort = yamlConfig.Applications.Pools.StartPort

	if config.Applications.Pools.StartPort == 0 {
		config.Applications.Pools.StartPort = DefaultStartPort
	}

	config.Applications.Env = yamlConfig.Applications.Env
	config.Applications.Runtime = yamlConfig.Applications.Runtime
	config.Applications.Server = yamlConfig.Applications.Server
	config.Applications.Args = yamlConfig.Applications.Args

	// Convert hooks
	config.Applications.Hooks.Start = yamlConfig.Applications.Hooks.Start
	config.Applications.Hooks.Stop = yamlConfig.Applications.Hooks.Stop

	// Convert tenants
	for _, t := range yamlConfig.Applications.Tenants {
		tenant := Tenant{
			Name:      t.Name,
			Root:      t.Root,
			PublicDir: t.PublicDir,
			Env:       make(map[string]string),
			Framework: t.Framework,
			Runtime:   t.Runtime,
			Server:    t.Server,
			Args:      t.Args,
			Var:       t.Var,
		}

		// Process environment variables with substitution
		if config.Applications.Env != nil {
			for key, template := range config.Applications.Env {
				value := substituteVars(template, &tenant)
				tenant.Env[key] = value
			}
		}

		// Override with tenant-specific env
		for key, value := range t.Env {
			tenant.Env[key] = value
		}

		tenant.Hooks.Start = t.Hooks.Start
		tenant.Hooks.Stop = t.Hooks.Stop

		config.Applications.Tenants = append(config.Applications.Tenants, tenant)
	}

	// Set managed processes
	config.ManagedProcesses = yamlConfig.ManagedProcesses

	// Set logging configuration
	config.Logging = yamlConfig.Logging

	// Set server hooks
	config.Hooks.Start = yamlConfig.Hooks.Start
	config.Hooks.Ready = yamlConfig.Hooks.Ready
	config.Hooks.Resume = yamlConfig.Hooks.Resume
	config.Hooks.Idle = yamlConfig.Hooks.Idle

	// Set static configuration
	for _, dir := range yamlConfig.Static.Directories {
		staticDir := StaticDir{
			Path:   dir.Path,
			Prefix: dir.Root,
			Cache:  dir.Cache,
		}
		config.Static.Directories = append(config.Static.Directories, staticDir)
	}
	config.Static.Extensions = yamlConfig.Static.Extensions
	config.Static.TryFiles.Enabled = yamlConfig.Static.TryFiles.Enabled
	config.Static.TryFiles.Suffixes = yamlConfig.Static.TryFiles.Suffixes
	config.Static.TryFiles.Fallback = yamlConfig.Static.TryFiles.Fallback

	// Set routes configuration
	config.Routes.Redirects = yamlConfig.Routes.Redirects
	config.Routes.Rewrites = yamlConfig.Routes.Rewrites

	// Convert standalone servers
	for _, server := range yamlConfig.StandaloneServers {
		config.StandaloneServers = append(config.StandaloneServers, ProxyRoute{
			Name:      server.Name,
			Prefix:    server.Prefix,
			Target:    server.Target,
			StripPath: server.StripPath,
			Headers:   server.Headers,
		})
	}

	return config, nil
}

// UpdateConfig updates configuration dynamically
func UpdateConfig(currentConfig *Config, newConfig *Config) {
	currentConfig.LocationConfigMutex.Lock()
	defer currentConfig.LocationConfigMutex.Unlock()

	// Update server configuration
	currentConfig.Server = newConfig.Server
	currentConfig.Locations = newConfig.Locations
	currentConfig.Applications = newConfig.Applications
	currentConfig.ManagedProcesses = newConfig.ManagedProcesses
	currentConfig.Logging = newConfig.Logging
	currentConfig.Hooks = newConfig.Hooks
	currentConfig.StandaloneServers = newConfig.StandaloneServers

	slog.Info("Configuration updated successfully")
}