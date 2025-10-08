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
			yamlConfig: func() YAMLConfig {
				cfg := YAMLConfig{}
				cfg.Server.Listen = 3001
				cfg.Server.Static.PublicDir = "/var/www/public"
				cfg.Auth.Enabled = true
				cfg.Auth.HTPasswd = "/etc/htpasswd"
				return cfg
			}(),
			wantListen: "3001",
			wantPublic: "/var/www/public",
			wantAuth:   "/etc/htpasswd",
		},
		{
			name: "string listen port",
			yamlConfig: func() YAMLConfig {
				cfg := YAMLConfig{}
				cfg.Server.Listen = ":8080"
				return cfg
			}(),
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
			if config.Server.Static.PublicDir != tt.wantPublic {
				t.Errorf("PublicDir = %v, want %v", config.Server.Static.PublicDir, tt.wantPublic)
			}
			if config.Auth.HTPasswd != tt.wantAuth {
				t.Errorf("Authentication = %v, want %v", config.Auth.HTPasswd, tt.wantAuth)
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
	yamlConfig := func() YAMLConfig {
		cfg := YAMLConfig{}
		cfg.Routes.Redirects = []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		}{
			{From: "^/old", To: "/new"},
		}
		cfg.Routes.Rewrites = []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		}{
			{From: "^/api/(.*)", To: "/v1/api/$1"},
		}
		return cfg
	}()

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
	yamlConfig := func() YAMLConfig {
		cfg := YAMLConfig{}
		cfg.Hooks.Server.Start = []HookConfig{{Command: "/usr/local/bin/start.sh"}}
		cfg.Hooks.Tenant.Start = []HookConfig{{Command: "/usr/local/bin/tenant-start.sh"}}
		return cfg
	}()

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

func TestConfigParser_ParseAuthConfig(t *testing.T) {
	yamlConfig := func() YAMLConfig {
		cfg := YAMLConfig{}
		cfg.Auth.Enabled = true
		cfg.Auth.Realm = "Test Realm"
		cfg.Auth.HTPasswd = "/etc/htpasswd"
		cfg.Auth.PublicPaths = []string{"/public", "/health"}
		return cfg
	}()

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check that auth is enabled
	if config.Auth.HTPasswd != "/etc/htpasswd" {
		t.Errorf("Expected htpasswd file '/etc/htpasswd', got %q", config.Auth.HTPasswd)
	}

	// Check that public paths were set in Auth.PublicPaths
	if len(config.Auth.PublicPaths) < 2 {
		t.Errorf("Expected at least 2 auth public paths, got %d", len(config.Auth.PublicPaths))
	}

	// Verify the public paths are in Auth.PublicPaths
	expectedPaths := []string{"/public", "/health"}
	for _, expectedPath := range expectedPaths {
		found := false
		for _, publicPath := range config.Auth.PublicPaths {
			if publicPath == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %q in Auth.PublicPaths, but it was not found", expectedPath)
		}
	}
}

func TestConfigParser_ParseFlyReplayRoutes(t *testing.T) {
	yamlConfig := func() YAMLConfig {
		cfg := YAMLConfig{}
		cfg.Routes.Fly.Replay = []struct {
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
		}
		return cfg
	}()

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
	yamlConfig := func() YAMLConfig {
		cfg := YAMLConfig{}
		cfg.Routes.Fly.StickySession.Enabled = true
		cfg.Routes.Fly.StickySession.CookieName = "_nav_session"
		cfg.Routes.Fly.StickySession.CookieMaxAge = "2h"
		cfg.Routes.Fly.StickySession.CookieSecure = true
		cfg.Routes.Fly.StickySession.CookieHTTPOnly = true
		cfg.Routes.Fly.StickySession.CookieSameSite = "Lax"
		cfg.Routes.Fly.StickySession.CookiePath = "/"
		cfg.Routes.Fly.StickySession.Paths = []string{"/app/*"}
		return cfg
	}()

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	ss := config.StickySession
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
  static:
    public_dir: /var/www/public
  idle:
    action: suspend
    timeout: 20m

routes:
  redirects:
    - from: "^/old"
      to: "/new"
  reverse_proxies:
    - name: action-cable
      path: "^/cable"
      target: "http://localhost:28080"
      websocket: true
  fly:
    replay:
      - path: "^/admin"
        region: "lax"
        status: 307
    sticky_sessions:
      enabled: true
      cookie_name: "_navigator_session"
      cookie_max_age: "1h"

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
	// Verify fly-replay was converted to rewrite rules
	flyReplayFound := false
	for _, rule := range config.Server.RewriteRules {
		if strings.HasPrefix(rule.Flag, "fly-replay:") {
			flyReplayFound = true
			break
		}
	}
	if !flyReplayFound {
		t.Error("Expected fly-replay rewrite rule not found")
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

func TestNormalizePathWithTrailingSlash(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "root path",
			input: "/",
			want:  "/",
		},
		{
			name:  "path without trailing slash",
			input: "/showcase",
			want:  "/showcase/",
		},
		{
			name:  "path with trailing slash",
			input: "/showcase/",
			want:  "/showcase/",
		},
		{
			name:  "nested path without trailing slash",
			input: "/showcase/2025/raleigh",
			want:  "/showcase/2025/raleigh/",
		},
		{
			name:  "nested path with trailing slash",
			input: "/showcase/2025/raleigh/",
			want:  "/showcase/2025/raleigh/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePathWithTrailingSlash(tt.input)
			if got != tt.want {
				t.Errorf("normalizePathWithTrailingSlash(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrailingSlashRedirects(t *testing.T) {
	yamlConfig := `
server:
  listen: 3000
  root_path: /showcase

applications:
  tenants:
    - path: /showcase/2025/raleigh
      root: /app/raleigh
    - path: /showcase/2025/boston/
      root: /app/boston
`

	// Parse YAML
	config, err := ParseYAML([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	// Verify root_path was normalized
	if config.Server.RootPath != "/showcase/" {
		t.Errorf("RootPath = %q, want %q", config.Server.RootPath, "/showcase/")
	}

	// Verify tenant paths were normalized
	if len(config.Applications.Tenants) != 2 {
		t.Fatalf("Expected 2 tenants, got %d", len(config.Applications.Tenants))
	}

	if config.Applications.Tenants[0].Path != "/showcase/2025/raleigh/" {
		t.Errorf("Tenant 0 Path = %q, want %q", config.Applications.Tenants[0].Path, "/showcase/2025/raleigh/")
	}

	if config.Applications.Tenants[1].Path != "/showcase/2025/boston/" {
		t.Errorf("Tenant 1 Path = %q, want %q", config.Applications.Tenants[1].Path, "/showcase/2025/boston/")
	}

	// Verify automatic redirects were created
	// Should have: 1 for root_path + 2 for tenants = 3 redirect rules
	redirectCount := 0
	for _, rule := range config.Server.RewriteRules {
		if rule.Flag == "redirect" {
			redirectCount++
		}
	}

	if redirectCount < 3 {
		t.Errorf("Expected at least 3 redirect rules, got %d", redirectCount)
	}

	// Verify specific redirects
	testCases := []struct {
		from string
		to   string
		desc string
	}{
		{from: "/showcase", to: "/showcase/", desc: "root_path redirect"},
		{from: "/showcase/2025/raleigh", to: "/showcase/2025/raleigh/", desc: "tenant 0 redirect"},
		{from: "/showcase/2025/boston", to: "/showcase/2025/boston/", desc: "tenant 1 redirect"},
	}

	for _, tc := range testCases {
		found := false
		for _, rule := range config.Server.RewriteRules {
			if rule.Flag == "redirect" && rule.Replacement == tc.to {
				// Test if the pattern matches the "from" path
				if rule.Pattern.MatchString(tc.from) {
					found = true
					// Verify it doesn't match the path WITH trailing slash
					if rule.Pattern.MatchString(tc.to) {
						t.Errorf("%s: pattern should not match %q (with trailing slash)", tc.desc, tc.to)
					}
					break
				}
			}
		}
		if !found {
			t.Errorf("%s: redirect from %q to %q not found", tc.desc, tc.from, tc.to)
		}
	}
}

func TestTrailingSlashRedirectsEmptyRootPath(t *testing.T) {
	yamlConfig := `
server:
  listen: 3000
  root_path: ""

applications:
  tenants:
    - path: /2025/raleigh
      root: /app/raleigh
`

	// Parse YAML
	config, err := ParseYAML([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	// Verify empty root_path stays empty
	if config.Server.RootPath != "" {
		t.Errorf("RootPath = %q, want empty string", config.Server.RootPath)
	}

	// Verify tenant path was normalized
	if config.Applications.Tenants[0].Path != "/2025/raleigh/" {
		t.Errorf("Tenant Path = %q, want %q", config.Applications.Tenants[0].Path, "/2025/raleigh/")
	}

	// Should only have redirect for tenant (not root_path since it's empty)
	redirectCount := 0
	for _, rule := range config.Server.RewriteRules {
		if rule.Flag == "redirect" {
			redirectCount++
		}
	}

	// Should have 1 redirect for the tenant
	if redirectCount < 1 {
		t.Errorf("Expected at least 1 redirect rule, got %d", redirectCount)
	}
}

func TestTrailingSlashRedirectsRootSlash(t *testing.T) {
	yamlConfig := `
server:
  listen: 3000
  root_path: /

applications:
  tenants:
    - path: /2025/raleigh
      root: /app/raleigh
`

	// Parse YAML
	config, err := ParseYAML([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	// Verify root "/" stays as "/"
	if config.Server.RootPath != "/" {
		t.Errorf("RootPath = %q, want %q", config.Server.RootPath, "/")
	}

	// Should only have redirect for tenant (not root_path since it's just "/")
	redirectCount := 0
	rootPathRedirect := false
	for _, rule := range config.Server.RewriteRules {
		if rule.Flag == "redirect" {
			redirectCount++
			if rule.Replacement == "/" {
				rootPathRedirect = true
			}
		}
	}

	// Should have 1 redirect for the tenant, but NOT for root_path
	if redirectCount != 1 {
		t.Errorf("Expected exactly 1 redirect rule, got %d", redirectCount)
	}

	if rootPathRedirect {
		t.Error("Should not create redirect for root_path when it is '/'")
	}
}
