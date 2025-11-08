package config

import (
	"regexp"
	"sync"
	"time"
)

// Constants for configuration defaults and limits
const (
	// Timeout constants
	DefaultIdleTimeout    = 10 * time.Minute
	RailsStartupTimeout   = 30 * time.Second
	DefaultStartupTimeout = 5 * time.Second // Default timeout before showing maintenance page
	ProxyRetryTimeout     = 3 * time.Second // Match legacy navigator timeout
	ProcessStopTimeout    = 10 * time.Second
	RailsStartupDelay     = 5 * time.Second

	// Port configuration
	DefaultStartPort  = 4000
	MaxPortRange      = 100
	DefaultListenPort = 3000

	// Proxy configuration
	MaxFlyReplaySize       = 1000000 // 1MB
	ProxyRetryInitialDelay = 100 * time.Millisecond
	ProxyRetryMaxDelay     = 500 * time.Millisecond

	// File paths
	NavigatorPIDFile       = "/tmp/navigator.pid"
	DefaultMaintenancePage = "/503.html"
)

// ManagedProcessConfig represents configuration for a managed process
type ManagedProcessConfig struct {
	Name        string            `yaml:"name"`
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args"`
	WorkingDir  string            `yaml:"working_dir"`
	Env         map[string]string `yaml:"env"`
	AutoRestart bool              `yaml:"auto_restart"`
	StartDelay  string            `yaml:"start_delay"` // Duration string like "2s", "1m"
}

// RewriteRule represents a rewrite rule
type RewriteRule struct {
	Pattern     *regexp.Regexp
	Replacement string
	Flag        string   // redirect, last, fly-replay:region:status, etc.
	Methods     []string // Allowed methods for this rule
}

// AuthPattern represents an auth exclusion pattern
type AuthPattern struct {
	Pattern *regexp.Regexp
	Action  string // "off" or realm name
}

// LogConfig represents logging configuration
type LogConfig struct {
	Format string `yaml:"format"` // "text" or "json"
	File   string `yaml:"file"`   // Optional file output path (supports {{app}} template)
	Vector struct {
		Enabled bool   `yaml:"enabled"` // Enable Vector integration
		Socket  string `yaml:"socket"`  // Unix socket path for Vector
		Config  string `yaml:"config"`  // Path to vector.toml configuration
	} `yaml:"vector"`
}

// HookConfig represents a hook command configuration
type HookConfig struct {
	Command      string   `yaml:"command"`
	Args         []string `yaml:"args"`
	Timeout      string   `yaml:"timeout"`       // Duration string like "30s", "5m", 0 for no timeout
	ReloadConfig string   `yaml:"reload_config"` // Config file to reload after successful hook execution
}

// CGIScriptConfig represents a CGI script configuration
type CGIScriptConfig struct {
	Path         string            `yaml:"path"`          // URL path to match (e.g., "/showcase/index_update")
	Script       string            `yaml:"script"`        // Path to CGI script executable
	Method       string            `yaml:"method"`        // HTTP method (GET, POST, etc.) - empty means all methods
	User         string            `yaml:"user"`          // Unix user to run script as (empty = current user)
	Group        string            `yaml:"group"`         // Unix group to run script as (empty = user's primary group)
	AllowedUsers []string          `yaml:"allowed_users"` // Usernames allowed to access this script (empty = all authenticated users)
	Env          map[string]string `yaml:"env"`           // Additional environment variables
	ReloadConfig string            `yaml:"reload_config"` // Config file to reload after successful script execution
	Timeout      string            `yaml:"timeout"`       // Execution timeout (e.g., "30s", "5m") - 0 means no timeout
}

// ServerHooks represents server lifecycle hooks
type ServerHooks struct {
	Start  []HookConfig `yaml:"start"`
	Ready  []HookConfig `yaml:"ready"`
	Resume []HookConfig `yaml:"resume"`
	Idle   []HookConfig `yaml:"idle"`
}

// TenantHooks represents tenant lifecycle hooks
type TenantHooks struct {
	Start []HookConfig `yaml:"start"`
	Stop  []HookConfig `yaml:"stop"`
}

// RoutesConfig represents routes configuration
type RoutesConfig struct {
	Redirects []struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	} `yaml:"redirects"`
	Rewrites []struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	} `yaml:"rewrites"`
	ReverseProxies []ProxyRoute `yaml:"reverse_proxies"`
	Fly            struct {
		Replay []struct {
			Path   string `yaml:"path"`
			App    string `yaml:"app"`
			Region string `yaml:"region"`
			Status int    `yaml:"status"`
		} `yaml:"replay"`
	} `yaml:"fly"`
}

// CacheControlOverride represents cache control configuration for specific paths
type CacheControlOverride struct {
	Path   string `yaml:"path"`
	MaxAge string `yaml:"max_age"` // Duration format: "24h", "1h"
}

// CacheControl represents cache control configuration
type CacheControl struct {
	Default   string                 `yaml:"default"`   // Default cache duration
	Overrides []CacheControlOverride `yaml:"overrides"` // Path-specific overrides
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Realm        string        `yaml:"realm"`
	HTPasswd     string        `yaml:"htpasswd"`
	PublicPaths  []string      `yaml:"public_paths"`
	AuthPatterns []AuthPattern `yaml:"auth_patterns"`
}

// StaticConfig represents static file serving configuration
type StaticConfig struct {
	PublicDir                string   `yaml:"public_dir"`
	AllowedExtensions        []string `yaml:"allowed_extensions"`
	TryFiles                 []string `yaml:"try_files"`
	NormalizeTrailingSlashes bool     `yaml:"normalize_trailing_slashes"` // Automatically redirect paths without trailing slashes to include them
	CacheControl             CacheControl
}

// MaintenanceConfig represents maintenance page configuration
type MaintenanceConfig struct {
	Enabled bool   `yaml:"enabled"` // Enable maintenance mode (serve maintenance page for all requests)
	Page    string `yaml:"page"`
}

// BotDetectionConfig represents bot detection configuration
type BotDetectionConfig struct {
	Enabled bool   `yaml:"enabled"` // Enable bot detection
	Action  string `yaml:"action"`  // Action to take: "reject" (403), "ignore" (allow), "static-only" (allow static, reject dynamic)
}

// CableConfig represents TurboCable/WebSocket configuration
type CableConfig struct {
	Enabled       bool   `yaml:"enabled"`        // Enable built-in WebSocket/Cable support (default: true)
	Path          string `yaml:"path"`           // WebSocket endpoint path (default: "/cable")
	BroadcastPath string `yaml:"broadcast_path"` // Broadcast endpoint path (default: "/_broadcast")
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Path     string               `yaml:"path"`     // Health check path (e.g., "/up")
	Response *HealthCheckResponse `yaml:"response"` // Optional synthetic response (if nil, proxies to app)
}

// HealthCheckResponse represents a synthetic health check response
type HealthCheckResponse struct {
	Status  int               `yaml:"status"`  // HTTP status code (e.g., 200)
	Body    string            `yaml:"body"`    // Response body
	Headers map[string]string `yaml:"headers"` // Response headers
}

// Config represents the main configuration
type Config struct {
	Server struct {
		Listen       string `yaml:"listen"`
		Hostname     string `yaml:"hostname"`
		RootPath     string `yaml:"root_path"`
		TrustProxy   bool   `yaml:"trust_proxy"` // Trust X-Forwarded-* headers from upstream proxy
		RewriteRules []RewriteRule
		Static       StaticConfig
		BotDetection BotDetectionConfig `yaml:"bot_detection"`
		CGIScripts   []CGIScriptConfig  `yaml:"cgi_scripts"`
		HealthCheck  HealthCheckConfig  `yaml:"health_check"`
		Idle         struct {
			Action  string `yaml:"action"`  // "suspend" or "stop"
			Timeout string `yaml:"timeout"` // Duration string like "30s", "5m"
		} `yaml:"idle"`
	} `yaml:"server"`
	Cable               CableConfig
	Auth                AuthConfig
	Routes              RoutesConfig           `yaml:"routes"`
	Applications        Applications           `yaml:"applications"`
	ManagedProcesses    []ManagedProcessConfig `yaml:"managed_processes"`
	Logging             LogConfig              `yaml:"logging"`
	Hooks               ServerHooks            `yaml:"hooks"`
	Maintenance         MaintenanceConfig      `yaml:"maintenance"`
	LocationConfigMutex sync.RWMutex
}

// Applications represents application configuration
type Applications struct {
	Pools           Pools               `yaml:"pools"`
	Tenants         []Tenant            `yaml:"tenants"`
	Env             map[string]string   `yaml:"env"`
	Hooks           TenantHooks         `yaml:"hooks"`
	Defaults        map[string]Tenant   // For framework-specific defaults
	Runtime         map[string]string   `yaml:"runtime"`          // Framework runtime commands
	Server          map[string]string   `yaml:"server"`           // Framework server commands
	Args            map[string][]string `yaml:"args"`             // Framework command arguments
	HealthCheck     string              `yaml:"health_check"`     // Default health check endpoint (e.g., "/up")
	StartupTimeout  string              `yaml:"startup_timeout"`  // Default timeout before showing maintenance page (e.g., "5s")
	TrackWebSockets bool                `yaml:"track_websockets"` // Global default for WebSocket tracking (default: true)
}

// Pools represents application pool configuration
type Pools struct {
	MaxSize            int    `yaml:"max_size"`
	Timeout            string `yaml:"timeout"` // Duration string like "5m", "10m"
	StartPort          int    `yaml:"start_port"`
	DefaultMemoryLimit string `yaml:"default_memory_limit"` // Default memory limit for tenants (e.g., "512M", "1G")
	User               string `yaml:"user"`                 // Default user to run tenant processes as
	Group              string `yaml:"group"`                // Default group to run tenant processes as
}

// ProxyRoute represents a proxy route configuration
type ProxyRoute struct {
	Name      string            `yaml:"name"`
	Path      string            `yaml:"path"`   // Regex pattern for matching paths
	Prefix    string            `yaml:"prefix"` // Alternative to Path for simple prefix matching
	Target    string            `yaml:"target"`
	StripPath bool              `yaml:"strip_path"`
	Headers   map[string]string `yaml:"headers"`
	WebSocket bool              `yaml:"websocket"` // Enable WebSocket support
}

// WebApp represents a web application
type WebApp struct {
	URL          string
	Process      interface{}
	Tenant       *Tenant
	Port         int
	StartTime    time.Time
	LastActivity time.Time
	CgroupPath   string    // Cgroup path for memory limiting (Linux only)
	MemoryLimit  int64     // Memory limit in bytes (0 = no limit)
	OOMCount     int       // Number of times this tenant has been OOM killed
	LastOOMTime  time.Time // Timestamp of last OOM kill
}

// FrameworkConfig represents framework-specific configuration
type FrameworkConfig struct {
	Runtime string   `yaml:"runtime"`
	Server  string   `yaml:"server"`
	Args    []string `yaml:"args"`
}

// Tenant represents a tenant configuration
type Tenant struct {
	Name            string                 `yaml:"name"`
	Path            string                 `yaml:"path"` // URL path prefix for tenant matching
	Root            string                 `yaml:"root"`
	PublicDir       string                 `yaml:"public_dir"`
	Env             map[string]string      `yaml:"env"`
	Framework       string                 `yaml:"framework"`
	Runtime         string                 `yaml:"runtime"`
	Server          string                 `yaml:"server"`
	Args            []string               `yaml:"args"`
	AppManager      interface{}            // Will be *AppManager
	Var             map[string]interface{} `yaml:"var"`
	Hooks           TenantHooks            `yaml:"hooks"`
	HealthCheck     string                 `yaml:"health_check"`     // Override health check endpoint for this tenant
	StartupTimeout  string                 `yaml:"startup_timeout"`  // Override startup timeout for this tenant (e.g., "10s")
	TrackWebSockets *bool                  `yaml:"track_websockets"` // Override WebSocket tracking (nil = use global default)
	BotDetection    *BotDetectionConfig    `yaml:"bot_detection"`    // Override bot detection for this tenant (nil = use global default)
	MemoryLimit     string                 `yaml:"memory_limit"`     // Memory limit for this tenant (e.g., "512M", "1G") - Linux only
	User            string                 `yaml:"user"`             // User to run this tenant's process as
	Group           string                 `yaml:"group"`            // Group to run this tenant's process as
}

// YAMLConfig represents the raw YAML configuration structure
type YAMLConfig struct {
	Cable struct {
		Enabled       *bool  `yaml:"enabled"` // Pointer to distinguish unset from false
		Path          string `yaml:"path"`
		BroadcastPath string `yaml:"broadcast_path"`
	} `yaml:"cable"`
	Auth struct {
		Enabled      bool     `yaml:"enabled"`
		Realm        string   `yaml:"realm"`
		HTPasswd     string   `yaml:"htpasswd"`
		PublicPaths  []string `yaml:"public_paths"`
		AuthPatterns []struct {
			Pattern string `yaml:"pattern"`
			Action  string `yaml:"action"`
		} `yaml:"auth_patterns"`
	} `yaml:"auth"`
	Server struct {
		Listen     interface{}       `yaml:"listen"`
		Hostname   string            `yaml:"hostname"`
		RootPath   string            `yaml:"root_path"`
		CGIScripts []CGIScriptConfig `yaml:"cgi_scripts"`
		Static     struct {
			PublicDir                string   `yaml:"public_dir"`
			AllowedExtensions        []string `yaml:"allowed_extensions"`
			TryFiles                 []string `yaml:"try_files"`
			NormalizeTrailingSlashes bool     `yaml:"normalize_trailing_slashes"`
			CacheControl             struct {
				Default   string `yaml:"default"`
				Overrides []struct {
					Path   string `yaml:"path"`
					MaxAge string `yaml:"max_age"`
				} `yaml:"overrides"`
			} `yaml:"cache_control"`
		} `yaml:"static"`
		Idle struct {
			Action  string `yaml:"action"`  // "suspend" or "stop"
			Timeout string `yaml:"timeout"` // Duration string like "30s", "5m"
		} `yaml:"idle"`
		HealthCheck HealthCheckConfig `yaml:"health_check"`
	} `yaml:"server"`
	Routes struct {
		Redirects []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		} `yaml:"redirects"`
		Rewrites []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		} `yaml:"rewrites"`
		ReverseProxies []ProxyRoute `yaml:"reverse_proxies"`
		Fly            struct {
			StickySession struct {
				Enabled        bool     `yaml:"enabled"`
				CookieName     string   `yaml:"cookie_name"`
				CookieMaxAge   string   `yaml:"cookie_max_age"`
				CookieSecure   bool     `yaml:"cookie_secure"`
				CookieHTTPOnly bool     `yaml:"cookie_httponly"`
				CookieSameSite string   `yaml:"cookie_samesite"`
				CookiePath     string   `yaml:"cookie_path"`
				Paths          []string `yaml:"paths"`
			} `yaml:"sticky_sessions"`
			Replay []struct {
				Path   string `yaml:"path"`
				App    string `yaml:"app"`
				Region string `yaml:"region"`
				Status int    `yaml:"status"`
			} `yaml:"replay"`
		} `yaml:"fly"`
	} `yaml:"routes"`
	Applications struct {
		Pools struct {
			MaxSize            int    `yaml:"max_size"`
			Timeout            string `yaml:"timeout"`
			StartPort          int    `yaml:"start_port"`
			DefaultMemoryLimit string `yaml:"default_memory_limit"`
			User               string `yaml:"user"`
			Group              string `yaml:"group"`
		} `yaml:"pools"`
		Framework struct {
			Command      string   `yaml:"command"`
			Args         []string `yaml:"args"`
			AppDirectory string   `yaml:"app_directory"`
			PortEnvVar   string   `yaml:"port_env_var"`
			StartDelay   string   `yaml:"start_delay"`
		} `yaml:"framework"`
		Tenants []struct {
			Path            string                 `yaml:"path"`
			Root            string                 `yaml:"root"`
			PublicDir       string                 `yaml:"public_dir"`
			Env             map[string]string      `yaml:"env"`
			Framework       string                 `yaml:"framework"`
			Runtime         string                 `yaml:"runtime"`
			Server          string                 `yaml:"server"`
			Args            []string               `yaml:"args"`
			Var             map[string]interface{} `yaml:"var"`
			HealthCheck     string                 `yaml:"health_check"`
			StartupTimeout  string                 `yaml:"startup_timeout"`
			TrackWebSockets *bool                  `yaml:"track_websockets"`
			MemoryLimit     string                 `yaml:"memory_limit"`
			User            string                 `yaml:"user"`
			Group           string                 `yaml:"group"`
			Hooks           struct {
				Start []HookConfig `yaml:"start"`
				Stop  []HookConfig `yaml:"stop"`
			} `yaml:"hooks"`
		} `yaml:"tenants"`
		Env             map[string]string   `yaml:"env"`
		Runtime         map[string]string   `yaml:"runtime"`
		Server          map[string]string   `yaml:"server"`
		Args            map[string][]string `yaml:"args"`
		HealthCheck     string              `yaml:"health_check"`
		StartupTimeout  string              `yaml:"startup_timeout"`
		TrackWebSockets bool                `yaml:"track_websockets"`
		Hooks           struct {
			Start []HookConfig `yaml:"start"`
			Stop  []HookConfig `yaml:"stop"`
		} `yaml:"hooks"`
	} `yaml:"applications"`
	ManagedProcesses []ManagedProcessConfig `yaml:"managed_processes"`
	Logging          LogConfig              `yaml:"logging"`
	Hooks            struct {
		Server struct {
			Start  []HookConfig `yaml:"start"`
			Ready  []HookConfig `yaml:"ready"`
			Resume []HookConfig `yaml:"resume"`
			Idle   []HookConfig `yaml:"idle"`
			Stop   []HookConfig `yaml:"stop"`
		} `yaml:"server"`
		Tenant struct {
			Start []HookConfig `yaml:"start"`
			Stop  []HookConfig `yaml:"stop"`
		} `yaml:"tenant"`
	} `yaml:"hooks"`
	Maintenance struct {
		Enabled bool   `yaml:"enabled"`
		Page    string `yaml:"page"`
	} `yaml:"maintenance"`
}
