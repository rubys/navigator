package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

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
	p.parseStaticConfig()
	p.parseRoutesConfig()
	p.parseApplicationConfig()
	p.parseManagedProcesses()
	p.parseLoggingConfig()
	p.parseHooksConfig()

	return p.config, nil
}

// parseServerConfig parses server-level configuration
func (p *ConfigParser) parseServerConfig() {
	p.config.Server.Hostname = p.yamlConfig.Server.Hostname
	p.config.Server.PublicDir = p.yamlConfig.Server.PublicDir
	p.config.Server.RootPath = p.yamlConfig.Server.RootPath
	p.config.Server.NamedHosts = p.yamlConfig.Server.NamedHosts
	p.config.Server.Root = p.yamlConfig.Server.Root
	p.config.Server.TryFiles = p.yamlConfig.Server.TryFiles
	p.config.Server.Authentication = p.yamlConfig.Server.Authentication
	p.config.Server.AuthExclude = p.yamlConfig.Server.AuthExclude

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

	// Parse sticky sessions
	p.parseStickySessionConfig()
}

// parseStickySessionConfig parses sticky session configuration
func (p *ConfigParser) parseStickySessionConfig() {
	ss := &p.config.Server.StickySession
	yamlSS := &p.yamlConfig.Server.StickySession

	ss.Enabled = yamlSS.Enabled
	ss.CookieName = yamlSS.CookieName
	ss.CookieMaxAge = yamlSS.CookieMaxAge
	ss.CookieSecure = yamlSS.CookieSecure
	ss.CookieHTTPOnly = yamlSS.CookieHTTPOnly
	ss.CookieSameSite = yamlSS.CookieSameSite
	ss.CookiePath = yamlSS.CookiePath
	ss.Paths = yamlSS.Paths

	// Parse sticky session max age duration
	if ss.CookieMaxAge != "" {
		if duration, err := time.ParseDuration(ss.CookieMaxAge); err == nil {
			ss.cookieMaxAge = duration
		}
	}
}

// parseAuthConfig parses authentication configuration
func (p *ConfigParser) parseAuthConfig() {
	if p.yamlConfig.Auth.Enabled {
		p.config.Server.Authentication = p.yamlConfig.Auth.HTPasswd
		p.config.Server.AuthExclude = p.yamlConfig.Auth.PublicPaths
	}

	// Note: AuthExclude patterns are handled as glob patterns in auth.go's ShouldExcludeFromAuth()
	// We don't compile them as regex here since they use glob syntax (e.g., *.css, *.js)
}

// parseStaticConfig parses static file configuration
func (p *ConfigParser) parseStaticConfig() {
	// Copy static directories
	for _, yamlDir := range p.yamlConfig.Static.Directories {
		dir := StaticDir{
			Path:  yamlDir.Path,
			Dir:   yamlDir.Dir,
			Cache: yamlDir.Cache,
		}
		p.config.Static.Directories = append(p.config.Static.Directories, dir)
	}

	// Copy extensions and try_files
	p.config.Static.Extensions = p.yamlConfig.Static.Extensions
	p.config.Static.TryFiles = p.yamlConfig.Static.TryFiles
}


// parseApplicationConfig parses application pool and tenant configuration
func (p *ConfigParser) parseApplicationConfig() {
	apps := &p.config.Applications
	yamlApps := &p.yamlConfig.Applications

	// Copy pool settings
	apps.Pools = yamlApps.Pools

	// Copy environment templates
	apps.Env = yamlApps.Env

	// Process tenants
	for _, yamlTenant := range yamlApps.Tenants {
		// Extract tenant name from path (e.g., "/showcase/2025/raleigh/" -> "2025/raleigh")
		tenantName := strings.TrimPrefix(yamlTenant.Path, "/showcase/")
		tenantName = strings.TrimSuffix(tenantName, "/")

		tenant := Tenant{
			Name:      tenantName,
			Root:      yamlTenant.Root,
			PublicDir: yamlTenant.PublicDir,
			Framework: yamlTenant.Framework,
			Runtime:   yamlTenant.Runtime,
			Server:    yamlTenant.Server,
			Args:      yamlTenant.Args,
			Var:       yamlTenant.Var,
			Hooks:     yamlTenant.Hooks,
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
	p.config.Hooks = p.yamlConfig.Hooks
}

// parseRoutesConfig parses routes configuration
func (p *ConfigParser) parseRoutesConfig() {
	p.config.Routes = p.yamlConfig.Routes

	// Convert routes to rewrite rules if needed
	for _, redirect := range p.config.Routes.Redirects {
		if pattern, err := regexp.Compile(redirect.From); err == nil {
			p.config.Server.RewriteRules = append(p.config.Server.RewriteRules, RewriteRule{
				Pattern:     pattern,
				Replacement: redirect.To,
				Flag:        "redirect",
			})
		}
	}

	for _, rewrite := range p.config.Routes.Rewrites {
		if pattern, err := regexp.Compile(rewrite.From); err == nil {
			p.config.Server.RewriteRules = append(p.config.Server.RewriteRules, RewriteRule{
				Pattern:     pattern,
				Replacement: rewrite.To,
				Flag:        "last",
			})
		}
	}

	// Convert fly-replay routes to rewrite rules
	for _, flyReplay := range p.config.Routes.FlyReplay {
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

