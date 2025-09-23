package config

import (
	"regexp"
	"sync"
	"time"
)

// Constants for configuration defaults and limits
const (
	// Timeout constants
	DefaultIdleTimeout  = 10 * time.Minute
	RailsStartupTimeout = 30 * time.Second
	ProxyRetryTimeout   = 3 * time.Second
	ProcessStopTimeout  = 10 * time.Second
	RailsStartupDelay   = 5 * time.Second

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
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	Timeout string   `yaml:"timeout"` // Duration string like "30s", "5m", 0 for no timeout
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

// StaticConfig represents static file configuration
type StaticConfig struct {
	Directories []StaticDir `yaml:"directories"`
	Extensions  []string    `yaml:"extensions"`
	TryFiles    struct {
		Enabled  bool     `yaml:"enabled"`
		Suffixes []string `yaml:"suffixes"`
		Fallback string   `yaml:"fallback"`
	} `yaml:"try_files"`
}

// RoutesConfig represents routes configuration
type RoutesConfig struct {
	Rewrites []struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	} `yaml:"rewrites"`
}

// Config represents the main configuration
type Config struct {
	Server struct {
		Listen           string   `yaml:"listen"`
		Hostname         string   `yaml:"hostname"`
		PublicDir        string   `yaml:"public_dir"`
		NamedHosts       []string `yaml:"named_hosts"`
		Root             string   `yaml:"root"`
		TryFiles         []string `yaml:"try_files"`
		Authentication   string   `yaml:"authentication"`
		AuthExclude      []string `yaml:"auth_exclude"`
		RewriteRules     []RewriteRule
		AuthPatterns     []AuthPattern
		Idle             struct {
			Action  string `yaml:"action"`  // "suspend" or "stop"
			Timeout string `yaml:"timeout"` // Duration string like "30s", "5m"
		} `yaml:"idle"`
		StickySession struct {
			Enabled        bool     `yaml:"enabled"`
			CookieName     string   `yaml:"cookie_name"`
			CookieMaxAge   string   `yaml:"cookie_max_age"`   // Duration format: "1h", "30m", etc.
			CookieSecure   bool     `yaml:"cookie_secure"`
			CookieHTTPOnly bool     `yaml:"cookie_httponly"`
			CookieSameSite string   `yaml:"cookie_samesite"`
			CookiePath     string   `yaml:"cookie_path"`
			Paths          []string `yaml:"paths"`
			cookieMaxAge   time.Duration
		} `yaml:"sticky_sessions"`
	} `yaml:"server"`
	Static              StaticConfig           `yaml:"static"`
	Routes              RoutesConfig           `yaml:"routes"`
	Locations           []Location             `yaml:"locations"`
	Applications        Applications           `yaml:"applications"`
	ManagedProcesses    []ManagedProcessConfig `yaml:"managed_processes"`
	Logging             LogConfig              `yaml:"logging"`
	Hooks               ServerHooks            `yaml:"hooks"`
	StandaloneServers   []ProxyRoute           `yaml:"standalone_servers"`
	LocationConfigMutex sync.RWMutex
}

// Applications represents application configuration
type Applications struct {
	Pools    Pools               `yaml:"pools"`
	Tenants  []Tenant            `yaml:"tenants"`
	Env      map[string]string   `yaml:"env"`
	Hooks    TenantHooks         `yaml:"hooks"`
	Defaults map[string]Tenant   // For framework-specific defaults
	Runtime  map[string]string   `yaml:"runtime"`  // Framework runtime commands
	Server   map[string]string   `yaml:"server"`   // Framework server commands
	Args     map[string][]string `yaml:"args"`     // Framework command arguments
}

// Pools represents application pool configuration
type Pools struct {
	MaxSize   int    `yaml:"max_size"`
	Timeout   string `yaml:"timeout"` // Duration string like "5m", "10m"
	StartPort int    `yaml:"start_port"`
}

// StaticDir represents static directory configuration
type StaticDir struct {
	Path       string   `yaml:"path"`
	Prefix     string   `yaml:"prefix"`
	TryFiles   []string `yaml:"try_files"`
	CacheAge   int      `yaml:"cache_age"`
	AuthExclude []string `yaml:"auth_exclude"`
}

// ProxyRoute represents a proxy route configuration
type ProxyRoute struct {
	Name       string            `yaml:"name"`
	Prefix     string            `yaml:"prefix"`
	Target     string            `yaml:"target"`
	StripPath  bool              `yaml:"strip_path"`
	Headers    map[string]string `yaml:"headers"`
}

// Location represents a location block configuration
type Location struct {
	Path              string            `yaml:"path"`
	PublicDir         string            `yaml:"public_dir"`
	TryFiles          []string          `yaml:"try_files"`
	ProxyPass         string            `yaml:"proxy_pass"`
	ProxyMethod       []string          `yaml:"proxy_method"`
	ProxyExcludeMethod []string          `yaml:"proxy_exclude_method"`
	RewriteRules      []RewriteRule
	AuthPatterns      []AuthPattern
	Alias             string            `yaml:"alias"`
}

// WebApp represents a web application
type WebApp struct {
	URL              string
	Process          interface{}
	Tenant           *Tenant
	Port             int
	StartTime        time.Time
	LastActivity     time.Time
	mutex            sync.Mutex
	wsConnections    map[string]interface{}
	wsConnectionsMux sync.RWMutex
}

// FrameworkConfig represents framework-specific configuration
type FrameworkConfig struct {
	Runtime string   `yaml:"runtime"`
	Server  string   `yaml:"server"`
	Args    []string `yaml:"args"`
}

// Tenant represents a tenant configuration
type Tenant struct {
	Name       string                 `yaml:"name"`
	Root       string                 `yaml:"root"`
	PublicDir  string                 `yaml:"public_dir"`
	Env        map[string]string      `yaml:"env"`
	Framework  string                 `yaml:"framework"`
	Runtime    string                 `yaml:"runtime"`
	Server     string                 `yaml:"server"`
	Args       []string               `yaml:"args"`
	AppManager interface{}            // Will be *AppManager
	Var        map[string]interface{} `yaml:"var"`
	Hooks      TenantHooks            `yaml:"hooks"`
}

// YAMLConfig represents the raw YAML configuration structure
type YAMLConfig struct {
	Server struct {
		Listen         interface{} `yaml:"listen"`
		Hostname       string      `yaml:"hostname"`
		PublicDir      string      `yaml:"public_dir"`
		NamedHosts     []string    `yaml:"named_hosts"`
		Root           string      `yaml:"root"`
		TryFiles       []string    `yaml:"try_files"`
		Authentication string      `yaml:"authentication"`
		AuthExclude    []string    `yaml:"auth_exclude"`
		Rewrites       []struct {
			Pattern     string   `yaml:"pattern"`
			Replacement string   `yaml:"replacement"`
			Flag        string   `yaml:"flag"`
			Methods     []string `yaml:"methods"`
		} `yaml:"rewrites"`
		Idle struct {
			Action  string `yaml:"action"`  // "suspend" or "stop"
			Timeout string `yaml:"timeout"` // Duration string like "30s", "5m"
		} `yaml:"idle"`
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
	} `yaml:"server"`
	Static struct {
		Directories []struct {
			Path  string `yaml:"path"`
			Root  string `yaml:"root"`
			Cache int    `yaml:"cache"`
		} `yaml:"directories"`
		Extensions []string `yaml:"extensions"`
		TryFiles   struct {
			Enabled  bool     `yaml:"enabled"`
			Suffixes []string `yaml:"suffixes"`
			Fallback string   `yaml:"fallback"`
		} `yaml:"try_files"`
	} `yaml:"static"`
	Routes struct {
		Rewrites []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		} `yaml:"rewrites"`
	} `yaml:"routes"`
	Locations []struct {
		Path               string   `yaml:"path"`
		PublicDir          string   `yaml:"public_dir"`
		TryFiles           []string `yaml:"try_files"`
		ProxyPass          string   `yaml:"proxy_pass"`
		ProxyMethod        []string `yaml:"proxy_method"`
		ProxyExcludeMethod []string `yaml:"proxy_exclude_method"`
		Alias              string   `yaml:"alias"`
		Rewrites           []struct {
			Pattern     string   `yaml:"pattern"`
			Replacement string   `yaml:"replacement"`
			Flag        string   `yaml:"flag"`
			Methods     []string `yaml:"methods"`
		} `yaml:"rewrites"`
	} `yaml:"locations"`
	Applications struct {
		Pools struct {
			MaxSize   int    `yaml:"max_size"`
			Timeout   string `yaml:"timeout"`
			StartPort int    `yaml:"start_port"`
		} `yaml:"pools"`
		Tenants []struct {
			Name      string                 `yaml:"name"`
			Root      string                 `yaml:"root"`
			PublicDir string                 `yaml:"public_dir"`
			Env       map[string]string      `yaml:"env"`
			Framework string                 `yaml:"framework"`
			Runtime   string                 `yaml:"runtime"`
			Server    string                 `yaml:"server"`
			Args      []string               `yaml:"args"`
			Var       map[string]interface{} `yaml:"var"`
			Hooks     struct {
				Start []HookConfig `yaml:"start"`
				Stop  []HookConfig `yaml:"stop"`
			} `yaml:"hooks"`
		} `yaml:"tenants"`
		Env     map[string]string   `yaml:"env"`
		Runtime map[string]string   `yaml:"runtime"`
		Server  map[string]string   `yaml:"server"`
		Args    map[string][]string `yaml:"args"`
		Hooks   struct {
			Start []HookConfig `yaml:"start"`
			Stop  []HookConfig `yaml:"stop"`
		} `yaml:"hooks"`
	} `yaml:"applications"`
	ManagedProcesses []ManagedProcessConfig `yaml:"managed_processes"`
	Logging          LogConfig              `yaml:"logging"`
	Hooks            struct {
		Start  []HookConfig `yaml:"start"`
		Ready  []HookConfig `yaml:"ready"`
		Resume []HookConfig `yaml:"resume"`
		Idle   []HookConfig `yaml:"idle"`
	} `yaml:"hooks"`
	StandaloneServers []struct {
		Name      string            `yaml:"name"`
		Prefix    string            `yaml:"prefix"`
		Target    string            `yaml:"target"`
		StripPath bool              `yaml:"strip_path"`
		Headers   map[string]string `yaml:"headers"`
	} `yaml:"standalone_servers"`
}