package config

import (
	"fmt"
	"regexp"
	"strings"
)

// normalizePathWithTrailingSlash ensures a path has a trailing slash
// Returns empty string for empty input
// Returns "/" for "/" input
// Returns "/path/" for "/path" or "/path/" input
func normalizePathWithTrailingSlash(path string) string {
	if path == "" {
		return ""
	}
	// Ensure it has a trailing slash
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}

// expandVariables expands ${var} placeholders in the given map using provided variables
func expandVariables(env map[string]string, vars map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range env {
		expanded := value
		for varName, varValue := range vars {
			// Convert interface{} to string
			strValue := ""
			switch v := varValue.(type) {
			case string:
				strValue = v
			default:
				strValue = fmt.Sprintf("%v", v)
			}
			expanded = strings.ReplaceAll(expanded, "${"+varName+"}", strValue)
		}
		result[key] = expanded
	}
	return result
}

// ConfigParser handles the parsing of YAML configuration into internal structures
type ConfigParser struct {
	yamlConfig *YAMLConfig
	config     *Config
}

// NewConfigParser creates a new configuration parser
func NewConfigParser(yamlConfig *YAMLConfig) *ConfigParser {
	return &ConfigParser{
		yamlConfig: yamlConfig,
		config:     &Config{},
	}
}

// Parse converts the YAML configuration to the internal Config structure
func (p *ConfigParser) Parse() (*Config, error) {
	p.parseServerConfig()
	p.parseAuthConfig()
	p.parseRoutesConfig()
	p.parseApplicationConfig()
	p.parseManagedProcesses()
	p.parseLoggingConfig()
	p.parseHooksConfig()
	p.parseMaintenanceConfig()

	// Add automatic trailing slash redirects after all other parsing
	p.addTrailingSlashRedirects()

	return p.config, nil
}

// parseServerConfig parses server-level configuration
func (p *ConfigParser) parseServerConfig() {
	p.config.Server.Hostname = p.yamlConfig.Server.Hostname
	// Normalize root_path to always have a trailing slash (unless empty)
	p.config.Server.RootPath = normalizePathWithTrailingSlash(p.yamlConfig.Server.RootPath)

	// Parse static file configuration
	p.config.Server.Static.PublicDir = p.yamlConfig.Server.Static.PublicDir
	p.config.Server.Static.TryFiles = p.yamlConfig.Server.Static.TryFiles
	p.config.Server.Static.AllowedExtensions = p.yamlConfig.Server.Static.AllowedExtensions

	// Parse cache control
	p.config.Server.Static.CacheControl.Default = p.yamlConfig.Server.Static.CacheControl.Default
	for _, override := range p.yamlConfig.Server.Static.CacheControl.Overrides {
		p.config.Server.Static.CacheControl.Overrides = append(p.config.Server.Static.CacheControl.Overrides, CacheControlOverride{
			Path:   override.Path,
			MaxAge: override.MaxAge,
		})
	}

	// Parse listen port
	switch v := p.yamlConfig.Server.Listen.(type) {
	case int:
		p.config.Server.Listen = fmt.Sprintf("%d", v)
	case string:
		p.config.Server.Listen = v
	default:
		p.config.Server.Listen = fmt.Sprintf("%d", DefaultListenPort)
	}

	// Set idle configuration
	p.config.Server.Idle.Action = p.yamlConfig.Server.Idle.Action
	p.config.Server.Idle.Timeout = p.yamlConfig.Server.Idle.Timeout

	// Copy health check configuration
	p.config.Server.HealthCheck = p.yamlConfig.Server.HealthCheck

	// Copy CGI scripts configuration
	p.config.Server.CGIScripts = p.yamlConfig.Server.CGIScripts
}

// parseAuthConfig parses authentication configuration
func (p *ConfigParser) parseAuthConfig() {
	p.config.Auth.Enabled = p.yamlConfig.Auth.Enabled
	p.config.Auth.Realm = p.yamlConfig.Auth.Realm
	p.config.Auth.HTPasswd = p.yamlConfig.Auth.HTPasswd

	// Only load public paths if auth is enabled
	if p.yamlConfig.Auth.Enabled {
		p.config.Auth.PublicPaths = p.yamlConfig.Auth.PublicPaths

		// Compile auth patterns (regex patterns for excluding paths from auth)
		for _, pattern := range p.yamlConfig.Auth.AuthPatterns {
			compiled, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				// Use slog since it's already imported via the package
				fmt.Printf("WARN: Invalid auth pattern, skipping: %s (error: %v)\n", pattern.Pattern, err)
				continue
			}
			p.config.Auth.AuthPatterns = append(p.config.Auth.AuthPatterns, AuthPattern{
				Pattern: compiled,
				Action:  pattern.Action,
			})
		}
	}

	// Note: PublicPaths patterns are handled as glob patterns in auth.go's ShouldExcludeFromAuth()
	// We don't compile them as regex here since they use glob syntax (e.g., *.css, *.js)
}

// parseMaintenanceConfig parses maintenance page configuration
func (p *ConfigParser) parseMaintenanceConfig() {
	p.config.Maintenance.Enabled = p.yamlConfig.Maintenance.Enabled
	p.config.Maintenance.Page = p.yamlConfig.Maintenance.Page
}

// parseApplicationConfig parses application pool and tenant configuration
func (p *ConfigParser) parseApplicationConfig() {
	apps := &p.config.Applications
	yamlApps := &p.yamlConfig.Applications

	// Copy pool settings
	apps.Pools = yamlApps.Pools

	// Copy environment templates
	apps.Env = yamlApps.Env

	// Copy framework-specific settings
	apps.Runtime = yamlApps.Runtime
	apps.Server = yamlApps.Server
	apps.Args = yamlApps.Args
	apps.HealthCheck = yamlApps.HealthCheck

	// Copy global track_websockets setting (default to true if not set)
	apps.TrackWebSockets = yamlApps.TrackWebSockets
	// If not explicitly set in YAML, default to true for backward compatibility
	if !yamlApps.TrackWebSockets {
		// Check if it was actually set to false or just defaulted
		// Since YAML unmarshals false as default, we assume true unless explicitly set
		// This is handled by setting default to true in documentation
		apps.TrackWebSockets = true
	}

	// Process tenants
	for _, yamlTenant := range yamlApps.Tenants {
		// Extract tenant name from path (e.g., "/showcase/2025/raleigh/" -> "2025/raleigh")
		tenantName := strings.TrimPrefix(yamlTenant.Path, "/showcase/")
		tenantName = strings.TrimSuffix(tenantName, "/")

		tenant := Tenant{
			Name:            tenantName,
			Path:            normalizePathWithTrailingSlash(yamlTenant.Path), // Normalize with trailing slash
			Root:            yamlTenant.Root,
			PublicDir:       yamlTenant.PublicDir,
			Framework:       yamlTenant.Framework,
			Runtime:         yamlTenant.Runtime,
			Server:          yamlTenant.Server,
			Args:            yamlTenant.Args,
			Var:             yamlTenant.Var,
			Hooks:           yamlTenant.Hooks,
			HealthCheck:     yamlTenant.HealthCheck,
			TrackWebSockets: yamlTenant.TrackWebSockets, // nil means use global setting
		}

		// Expand environment variables with tenant vars
		if apps.Env != nil {
			tenant.Env = expandVariables(apps.Env, tenant.Var)
		}

		// Merge with tenant-specific environment
		if yamlTenant.Env != nil {
			if tenant.Env == nil {
				tenant.Env = make(map[string]string)
			}
			for k, v := range yamlTenant.Env {
				tenant.Env[k] = v
			}
		}

		apps.Tenants = append(apps.Tenants, tenant)
	}
}

// parseManagedProcesses parses managed process configuration
func (p *ConfigParser) parseManagedProcesses() {
	p.config.ManagedProcesses = p.yamlConfig.ManagedProcesses
}

// parseLoggingConfig parses logging configuration
func (p *ConfigParser) parseLoggingConfig() {
	p.config.Logging = p.yamlConfig.Logging
}

// parseHooksConfig parses lifecycle hooks
func (p *ConfigParser) parseHooksConfig() {
	// Map server hooks from hooks.server to Config.Hooks
	p.config.Hooks.Start = p.yamlConfig.Hooks.Server.Start
	p.config.Hooks.Ready = p.yamlConfig.Hooks.Server.Ready
	p.config.Hooks.Resume = p.yamlConfig.Hooks.Server.Resume
	p.config.Hooks.Idle = p.yamlConfig.Hooks.Server.Idle

	// Map tenant default hooks from hooks.tenant to Config.Applications.Hooks
	p.config.Applications.Hooks.Start = p.yamlConfig.Hooks.Tenant.Start
	p.config.Applications.Hooks.Stop = p.yamlConfig.Hooks.Tenant.Stop
}

// parseRoutesConfig parses routes configuration
func (p *ConfigParser) parseRoutesConfig() {
	// Copy routes configuration
	p.config.Routes.Redirects = p.yamlConfig.Routes.Redirects
	p.config.Routes.Rewrites = p.yamlConfig.Routes.Rewrites
	p.config.Routes.ReverseProxies = p.yamlConfig.Routes.ReverseProxies

	// Convert routes to rewrite rules if needed
	for _, redirect := range p.yamlConfig.Routes.Redirects {
		if pattern, err := regexp.Compile(redirect.From); err == nil {
			p.config.Server.RewriteRules = append(p.config.Server.RewriteRules, RewriteRule{
				Pattern:     pattern,
				Replacement: redirect.To,
				Flag:        "redirect",
			})
		}
	}

	for _, rewrite := range p.yamlConfig.Routes.Rewrites {
		if pattern, err := regexp.Compile(rewrite.From); err == nil {
			p.config.Server.RewriteRules = append(p.config.Server.RewriteRules, RewriteRule{
				Pattern:     pattern,
				Replacement: rewrite.To,
				Flag:        "last",
			})
		}
	}

	// Convert fly-replay routes to rewrite rules
	flyReplays := p.yamlConfig.Routes.Fly.Replay
	for _, flyReplay := range flyReplays {
		if pattern, err := regexp.Compile(flyReplay.Path); err == nil {
			// Determine target format for fly-replay
			var target string
			if flyReplay.App != "" {
				target = fmt.Sprintf("app=%s", flyReplay.App)
			} else if flyReplay.Region != "" {
				target = flyReplay.Region
			} else {
				continue // Skip if no target specified
			}

			// Default status to 307 if not specified
			status := flyReplay.Status
			if status == 0 {
				status = 307
			}

			p.config.Server.RewriteRules = append(p.config.Server.RewriteRules, RewriteRule{
				Pattern:     pattern,
				Replacement: flyReplay.Path, // Keep original path for fly-replay
				Flag:        fmt.Sprintf("fly-replay:%s:%d", target, status),
			})
		}
	}
}

// addTrailingSlashRedirects adds automatic redirects from non-trailing-slash to trailing-slash versions
// for both root_path and all tenant paths
func (p *ConfigParser) addTrailingSlashRedirects() {
	// Add redirect for root_path (if not empty and not just "/")
	if p.config.Server.RootPath != "" && p.config.Server.RootPath != "/" {
		// Remove trailing slash to create the "from" pattern
		pathWithoutSlash := strings.TrimSuffix(p.config.Server.RootPath, "/")
		// Create regex pattern that matches the path without trailing slash at end of string
		pattern := regexp.MustCompile("^" + regexp.QuoteMeta(pathWithoutSlash) + "$")

		p.config.Server.RewriteRules = append(p.config.Server.RewriteRules, RewriteRule{
			Pattern:     pattern,
			Replacement: p.config.Server.RootPath, // The normalized version with trailing slash
			Flag:        "redirect",
		})
	}

	// Add redirects for all tenant paths
	for _, tenant := range p.config.Applications.Tenants {
		// Remove trailing slash to create the "from" pattern
		pathWithoutSlash := strings.TrimSuffix(tenant.Path, "/")
		// Create regex pattern that matches the path without trailing slash at end of string
		pattern := regexp.MustCompile("^" + regexp.QuoteMeta(pathWithoutSlash) + "$")

		p.config.Server.RewriteRules = append(p.config.Server.RewriteRules, RewriteRule{
			Pattern:     pattern,
			Replacement: tenant.Path, // The normalized version with trailing slash
			Flag:        "redirect",
		})
	}
}
