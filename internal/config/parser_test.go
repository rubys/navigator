package config

import (
	"strings"
	"testing"
)

func TestConfigParser_ParseServerConfig(t *testing.T) {
	tests := []struct {
		name       string
		yamlConfig YAMLConfig
		wantListen string
		wantPublic string
		wantAuth   string
	}{
		{
			name: "basic server config",
			yamlConfig: YAMLConfig{
				Server: struct {
					Listen         interface{} `yaml:"listen"`
					Hostname       string      `yaml:"hostname"`
					PublicDir      string      `yaml:"public_dir"`
					RootPath       string      `yaml:"root_path"`
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
						Action  string `yaml:"action"`
						Timeout string `yaml:"timeout"`
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
				}{
					Listen:         3001,
					PublicDir:      "/var/www/public",
					Authentication: "/etc/htpasswd",
				},
			},
			wantListen: "3001",
			wantPublic: "/var/www/public",
			wantAuth:   "/etc/htpasswd",
		},
		{
			name: "string listen port",
			yamlConfig: YAMLConfig{
				Server: struct {
					Listen         interface{} `yaml:"listen"`
					Hostname       string      `yaml:"hostname"`
					PublicDir      string      `yaml:"public_dir"`
					RootPath       string      `yaml:"root_path"`
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
						Action  string `yaml:"action"`
						Timeout string `yaml:"timeout"`
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
				}{
					Listen: ":8080",
				},
			},
			wantListen: ":8080",
			wantPublic: "",
			wantAuth:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewConfigParser(&tt.yamlConfig)
			config, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if config.Server.Listen != tt.wantListen {
				t.Errorf("Listen = %v, want %v", config.Server.Listen, tt.wantListen)
			}
			if config.Server.PublicDir != tt.wantPublic {
				t.Errorf("PublicDir = %v, want %v", config.Server.PublicDir, tt.wantPublic)
			}
			if config.Server.Authentication != tt.wantAuth {
				t.Errorf("Authentication = %v, want %v", config.Server.Authentication, tt.wantAuth)
			}
		})
	}
}

func TestConfigParser_ParseApplicationConfig(t *testing.T) {
	yamlConfig := `
applications:
  env:
    DB_NAME: "myapp_${environment}"
    DB_HOST: "localhost"
  tenants:
    - path: "/showcase/2025/raleigh/"
      root: "/app/raleigh"
      var:
        environment: "production"
`

	// Parse YAML
	config, err := ParseYAML([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	// Check tenant environment expansion
	if len(config.Applications.Tenants) != 1 {
		t.Fatalf("Expected 1 tenant, got %d", len(config.Applications.Tenants))
	}

	tenant := config.Applications.Tenants[0]

	// Check that variables were expanded
	expectedDBName := "myapp_production"
	if tenant.Env["DB_NAME"] != expectedDBName {
		t.Errorf("Tenant env DB_NAME = %v, want %v", tenant.Env["DB_NAME"], expectedDBName)
	}

	if tenant.Env["DB_HOST"] != "localhost" {
		t.Errorf("Tenant env DB_HOST = %v, want %v", tenant.Env["DB_HOST"], "localhost")
	}
}

func TestConfigParser_ParseRoutesConfig(t *testing.T) {
	yamlConfig := YAMLConfig{
		Routes: RoutesConfig{
			Redirects: []struct {
				From string `yaml:"from"`
				To   string `yaml:"to"`
			}{
				{From: "^/old", To: "/new"},
			},
			Rewrites: []struct {
				From string `yaml:"from"`
				To   string `yaml:"to"`
			}{
				{From: "^/api/(.*)", To: "/v1/api/$1"},
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check that rewrites were converted to rewrite rules
	if len(config.Server.RewriteRules) < 2 {
		t.Errorf("Expected at least 2 rewrite rules, got %d", len(config.Server.RewriteRules))
	}

	// Check redirect rule
	found := false
	for _, rule := range config.Server.RewriteRules {
		if rule.Flag == "redirect" && rule.Replacement == "/new" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected redirect rewrite rule not found")
	}

	// Check rewrite rule
	found = false
	for _, rule := range config.Server.RewriteRules {
		if rule.Flag == "last" && rule.Replacement == "/v1/api/$1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected last rewrite rule not found")
	}
}

func TestConfigParser_ParseManagedProcesses(t *testing.T) {
	yamlConfig := YAMLConfig{
		ManagedProcesses: []ManagedProcessConfig{
			{
				Name:        "redis",
				Command:     "redis-server",
				Args:        []string{"--port", "6379"},
				AutoRestart: true,
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(config.ManagedProcesses) != 1 {
		t.Fatalf("Expected 1 managed process, got %d", len(config.ManagedProcesses))
	}

	proc := config.ManagedProcesses[0]
	if proc.Name != "redis" {
		t.Errorf("Name = %v, want redis", proc.Name)
	}
	if proc.Command != "redis-server" {
		t.Errorf("Command = %v, want redis-server", proc.Command)
	}
	if !proc.AutoRestart {
		t.Error("Expected AutoRestart to be true")
	}
}

func TestConfigParser_ParseLoggingConfig(t *testing.T) {
	yamlConfig := YAMLConfig{
		Logging: LogConfig{
			Format: "json",
			File:   "/var/log/navigator.log",
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if config.Logging.Format != "json" {
		t.Errorf("Format = %v, want json", config.Logging.Format)
	}
	if config.Logging.File != "/var/log/navigator.log" {
		t.Errorf("File = %v, want /var/log/navigator.log", config.Logging.File)
	}
}

func TestConfigParser_ParseHooksConfig(t *testing.T) {
	yamlConfig := YAMLConfig{
		Hooks: struct {
			Server struct {
				Start  []HookConfig `yaml:"start"`
				Ready  []HookConfig `yaml:"ready"`
				Resume []HookConfig `yaml:"resume"`
				Idle   []HookConfig `yaml:"idle"`
			} `yaml:"server"`
			Tenant struct {
				Start []HookConfig `yaml:"start"`
				Stop  []HookConfig `yaml:"stop"`
			} `yaml:"tenant"`
		}{
			Server: struct {
				Start  []HookConfig `yaml:"start"`
				Ready  []HookConfig `yaml:"ready"`
				Resume []HookConfig `yaml:"resume"`
				Idle   []HookConfig `yaml:"idle"`
			}{
				Start: []HookConfig{
					{Command: "/usr/local/bin/start.sh"},
				},
			},
			Tenant: struct {
				Start []HookConfig `yaml:"start"`
				Stop  []HookConfig `yaml:"stop"`
			}{
				Start: []HookConfig{
					{Command: "/usr/local/bin/tenant-start.sh"},
				},
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(config.Hooks.Start) != 1 {
		t.Fatalf("Expected 1 server start hook, got %d", len(config.Hooks.Start))
	}
	if config.Hooks.Start[0].Command != "/usr/local/bin/start.sh" {
		t.Errorf("Start hook command = %v, want /usr/local/bin/start.sh", config.Hooks.Start[0].Command)
	}

	if len(config.Applications.Hooks.Start) != 1 {
		t.Fatalf("Expected 1 tenant start hook, got %d", len(config.Applications.Hooks.Start))
	}
	if config.Applications.Hooks.Start[0].Command != "/usr/local/bin/tenant-start.sh" {
		t.Errorf("Tenant start hook command = %v, want /usr/local/bin/tenant-start.sh", config.Applications.Hooks.Start[0].Command)
	}
}

func TestConfigParser_ParseStaticConfig(t *testing.T) {
	yamlConfig := YAMLConfig{
		Static: struct {
			Directories []struct {
				Path  string `yaml:"path"`
				Dir   string `yaml:"dir"`
				Cache string `yaml:"cache"`
			} `yaml:"directories"`
			Extensions []string `yaml:"extensions"`
			TryFiles   struct {
				Enabled  bool     `yaml:"enabled"`
				Suffixes []string `yaml:"suffixes"`
				Fallback string   `yaml:"fallback"`
			} `yaml:"try_files"`
		}{
			Extensions: []string{".html", ".css", ".js"},
			TryFiles: struct {
				Enabled  bool     `yaml:"enabled"`
				Suffixes []string `yaml:"suffixes"`
				Fallback string   `yaml:"fallback"`
			}{
				Enabled:  true,
				Suffixes: []string{".html"},
				Fallback: "/index.html",
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(config.Static.Extensions) != 3 {
		t.Errorf("Expected 3 extensions, got %d", len(config.Static.Extensions))
	}
	if !config.Static.TryFiles.Enabled {
		t.Error("Expected TryFiles to be enabled")
	}
	if config.Static.TryFiles.Fallback != "/index.html" {
		t.Errorf("Fallback = %v, want /index.html", config.Static.TryFiles.Fallback)
	}
}

func TestConfigParser_ParseAuthConfig(t *testing.T) {
	yamlConfig := YAMLConfig{
		Auth: struct {
			Enabled         bool     `yaml:"enabled"`
			Realm           string   `yaml:"realm"`
			HTPasswd        string   `yaml:"htpasswd"`
			PublicPaths     []string `yaml:"public_paths"`
			ExcludePatterns []struct {
				Pattern     string `yaml:"pattern"`
				Description string `yaml:"description"`
			} `yaml:"exclude_patterns"`
		}{
			Enabled:  true,
			Realm:    "Test Realm",
			HTPasswd: "/etc/htpasswd",
			PublicPaths: []string{
				"/public",
				"/health",
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check that auth patterns were created
	if len(config.Server.AuthPatterns) < 2 {
		t.Errorf("Expected at least 2 auth patterns, got %d", len(config.Server.AuthPatterns))
	}

	// Verify one of the public path patterns
	foundPublic := false
	for _, pattern := range config.Server.AuthPatterns {
		if pattern.Action == "off" {
			foundPublic = true
			break
		}
	}
	if !foundPublic {
		t.Error("Expected to find public path pattern with action 'off'")
	}
}

func TestConfigParser_ParseFlyReplayRoutes(t *testing.T) {
	yamlConfig := YAMLConfig{
		Routes: RoutesConfig{
			FlyReplay: []struct {
				Path   string `yaml:"path"`
				App    string `yaml:"app"`
				Region string `yaml:"region"`
				Status int    `yaml:"status"`
			}{
				{
					Path:   "^/admin",
					Region: "lax",
					Status: 307,
				},
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check that fly-replay route was converted to rewrite rule
	found := false
	for _, rule := range config.Server.RewriteRules {
		if strings.HasPrefix(rule.Flag, "fly-replay:") {
			found = true
			if !strings.Contains(rule.Flag, "lax") {
				t.Error("Expected fly-replay flag to contain region 'lax'")
			}
			if !strings.Contains(rule.Flag, "307") {
				t.Error("Expected fly-replay flag to contain status '307'")
			}
		}
	}
	if !found {
		t.Error("Expected fly-replay rewrite rule not found")
	}
}

func TestConfigParser_ParseStickySessionConfig(t *testing.T) {
	yamlConfig := YAMLConfig{
		Server: struct {
			Listen         interface{} `yaml:"listen"`
			Hostname       string      `yaml:"hostname"`
			PublicDir      string      `yaml:"public_dir"`
			RootPath       string      `yaml:"root_path"`
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
				Action  string `yaml:"action"`
				Timeout string `yaml:"timeout"`
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
		}{
			StickySession: struct {
				Enabled        bool     `yaml:"enabled"`
				CookieName     string   `yaml:"cookie_name"`
				CookieMaxAge   string   `yaml:"cookie_max_age"`
				CookieSecure   bool     `yaml:"cookie_secure"`
				CookieHTTPOnly bool     `yaml:"cookie_httponly"`
				CookieSameSite string   `yaml:"cookie_samesite"`
				CookiePath     string   `yaml:"cookie_path"`
				Paths          []string `yaml:"paths"`
			}{
				Enabled:        true,
				CookieName:     "_nav_session",
				CookieMaxAge:   "2h",
				CookieSecure:   true,
				CookieHTTPOnly: true,
				CookieSameSite: "Lax",
				CookiePath:     "/",
				Paths:          []string{"/app/*"},
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	ss := config.Server.StickySession
	if !ss.Enabled {
		t.Error("Expected sticky sessions to be enabled")
	}
	if ss.CookieName != "_nav_session" {
		t.Errorf("CookieName = %v, want _nav_session", ss.CookieName)
	}
	if ss.CookieMaxAge != "2h" {
		t.Errorf("CookieMaxAge = %v, want 2h", ss.CookieMaxAge)
	}
	if !ss.CookieSecure {
		t.Error("Expected CookieSecure to be true")
	}
	if !ss.CookieHTTPOnly {
		t.Error("Expected CookieHTTPOnly to be true")
	}
	if ss.CookieSameSite != "Lax" {
		t.Errorf("CookieSameSite = %v, want Lax", ss.CookieSameSite)
	}
	if len(ss.Paths) != 1 || ss.Paths[0] != "/app/*" {
		t.Errorf("Paths = %v, want [/app/*]", ss.Paths)
	}
}

func TestConfigParser_ParseCompleteConfig(t *testing.T) {
	yamlConfig := `
server:
  listen: 3000
  hostname: localhost
  public_dir: /var/www/public
  idle:
    action: suspend
    timeout: 20m
  sticky_sessions:
    enabled: true
    cookie_name: "_navigator_session"
    cookie_max_age: "1h"

routes:
  redirects:
    - from: "^/old"
      to: "/new"
  reverse_proxies:
    - name: action-cable
      path: "^/cable"
      target: "http://localhost:28080"
      websocket: true
  fly_replay:
    - path: "^/admin"
      region: "lax"
      status: 307

applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000
  tenants:
    - path: "/showcase/2025/raleigh/"
      root: "/app/raleigh"

managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true

logging:
  format: json
  file: /var/log/navigator.log
`

	// Parse YAML
	config, err := ParseYAML([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	// Verify all components were parsed
	if len(config.Routes.ReverseProxies) != 1 {
		t.Errorf("Expected 1 reverse proxy, got %d", len(config.Routes.ReverseProxies))
	}
	// StandaloneServers removed - using Routes.ReverseProxies instead
	if len(config.Routes.FlyReplay) != 1 {
		t.Errorf("Expected 1 fly replay route, got %d", len(config.Routes.FlyReplay))
	}
	if len(config.Server.RewriteRules) < 1 {
		t.Error("Expected at least 1 rewrite rule from redirect")
	}
}

func TestConfigParser_ParseTrackWebSocketsConfig(t *testing.T) {
	tests := []struct {
		name                 string
		yamlConfig           string
		wantGlobal           bool
		wantTenant1Override  *bool
		wantTenant2Override  *bool
		wantEffectiveTenant1 bool
		wantEffectiveTenant2 bool
	}{
		{
			name: "global enabled, no tenant overrides",
			yamlConfig: `
applications:
  track_websockets: true
  tenants:
    - path: "/showcase/2025/raleigh/"
      root: "/app/raleigh"
    - path: "/showcase/2025/boston/"
      root: "/app/boston"
`,
			wantGlobal:           true,
			wantTenant1Override:  nil,
			wantTenant2Override:  nil,
			wantEffectiveTenant1: true,
			wantEffectiveTenant2: true,
		},
		{
			name: "global enabled, one tenant disables",
			yamlConfig: `
applications:
  track_websockets: true
  tenants:
    - path: "/showcase/2025/raleigh/"
      root: "/app/raleigh"
      track_websockets: false
    - path: "/showcase/2025/boston/"
      root: "/app/boston"
`,
			wantGlobal:           true,
			wantTenant1Override:  boolPtr(false),
			wantTenant2Override:  nil,
			wantEffectiveTenant1: false,
			wantEffectiveTenant2: true,
		},
		{
			name: "global disabled (default), tenant enables",
			yamlConfig: `
applications:
  tenants:
    - path: "/showcase/2025/raleigh/"
      root: "/app/raleigh"
      track_websockets: true
    - path: "/showcase/2025/boston/"
      root: "/app/boston"
`,
			wantGlobal:           true, // defaults to true in parser
			wantTenant1Override:  boolPtr(true),
			wantTenant2Override:  nil,
			wantEffectiveTenant1: true,
			wantEffectiveTenant2: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse YAML
			config, err := ParseYAML([]byte(tt.yamlConfig))
			if err != nil {
				t.Fatalf("ParseYAML() error = %v", err)
			}

			// Check global setting
			if config.Applications.TrackWebSockets != tt.wantGlobal {
				t.Errorf("Global TrackWebSockets = %v, want %v",
					config.Applications.TrackWebSockets, tt.wantGlobal)
			}

			// Check tenant overrides
			if len(config.Applications.Tenants) < 2 {
				t.Fatalf("Expected at least 2 tenants, got %d", len(config.Applications.Tenants))
			}

			tenant1 := config.Applications.Tenants[0]
			tenant2 := config.Applications.Tenants[1]

			if !compareBoolPtr(tenant1.TrackWebSockets, tt.wantTenant1Override) {
				t.Errorf("Tenant1 TrackWebSockets override = %v, want %v",
					formatBoolPtr(tenant1.TrackWebSockets), formatBoolPtr(tt.wantTenant1Override))
			}

			if !compareBoolPtr(tenant2.TrackWebSockets, tt.wantTenant2Override) {
				t.Errorf("Tenant2 TrackWebSockets override = %v, want %v",
					formatBoolPtr(tenant2.TrackWebSockets), formatBoolPtr(tt.wantTenant2Override))
			}
		})
	}
}

// Helper functions for testing pointer values
func boolPtr(b bool) *bool {
	return &b
}

func compareBoolPtr(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func formatBoolPtr(b *bool) string {
	if b == nil {
		return "nil"
	}
	if *b {
		return "true"
	}
	return "false"
}
