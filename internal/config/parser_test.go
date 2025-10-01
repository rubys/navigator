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
					PublicDir:      "dist",
					Authentication: "htpasswd",
				},
			},
			wantListen: "3001",
			wantPublic: "dist",
			wantAuth:   "htpasswd",
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
					Listen: "8080",
				},
			},
			wantListen: "8080",
			wantPublic: "",
			wantAuth:   "",
		},
		{
			name:       "default listen port",
			yamlConfig: YAMLConfig{},
			wantListen: "3000",
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
				t.Errorf("Server.Listen = %v, want %v", config.Server.Listen, tt.wantListen)
			}

			if config.Server.PublicDir != tt.wantPublic {
				t.Errorf("Server.PublicDir = %v, want %v", config.Server.PublicDir, tt.wantPublic)
			}

			if config.Server.Authentication != tt.wantAuth {
				t.Errorf("Server.Authentication = %v, want %v", config.Server.Authentication, tt.wantAuth)
			}
		})
	}
}

func TestConfigParser_ParseApplicationConfig(t *testing.T) {
	yamlConfig := YAMLConfig{
		Applications: struct {
			Pools struct {
				MaxSize   int    `yaml:"max_size"`
				Timeout   string `yaml:"timeout"`
				StartPort int    `yaml:"start_port"`
			} `yaml:"pools"`
			Tenants []struct {
				Path        string                 `yaml:"path"`
				Root        string                 `yaml:"root"`
				PublicDir   string                 `yaml:"public_dir"`
				Env         map[string]string      `yaml:"env"`
				Framework   string                 `yaml:"framework"`
				Runtime     string                 `yaml:"runtime"`
				Server      string                 `yaml:"server"`
				Args        []string               `yaml:"args"`
				Var         map[string]interface{} `yaml:"var"`
				HealthCheck string                 `yaml:"health_check"`
				Hooks       struct {
					Start []HookConfig `yaml:"start"`
					Stop  []HookConfig `yaml:"stop"`
				} `yaml:"hooks"`
			} `yaml:"tenants"`
			Env         map[string]string   `yaml:"env"`
			Runtime     map[string]string   `yaml:"runtime"`
			Server      map[string]string   `yaml:"server"`
			Args        map[string][]string `yaml:"args"`
			HealthCheck string              `yaml:"health_check"`
			Hooks       struct {
				Start []HookConfig `yaml:"start"`
				Stop  []HookConfig `yaml:"stop"`
			} `yaml:"hooks"`
		}{
			Pools: struct {
				MaxSize   int    `yaml:"max_size"`
				Timeout   string `yaml:"timeout"`
				StartPort int    `yaml:"start_port"`
			}{
				MaxSize:   10,
				Timeout:   "5m",
				StartPort: 4000,
			},
			Env: map[string]string{
				"DB_NAME": "${app}_${env}",
				"DB_HOST": "localhost",
			},
			Tenants: []struct {
				Path        string                 `yaml:"path"`
				Root        string                 `yaml:"root"`
				PublicDir   string                 `yaml:"public_dir"`
				Env         map[string]string      `yaml:"env"`
				Framework   string                 `yaml:"framework"`
				Runtime     string                 `yaml:"runtime"`
				Server      string                 `yaml:"server"`
				Args        []string               `yaml:"args"`
				Var         map[string]interface{} `yaml:"var"`
				HealthCheck string                 `yaml:"health_check"`
				Hooks       struct {
					Start []HookConfig `yaml:"start"`
					Stop  []HookConfig `yaml:"stop"`
				} `yaml:"hooks"`
			}{
				{
					Path: "/showcase/test-app/",
					Root: "/app/test",
					Var: map[string]interface{}{
						"app": "myapp",
						"env": "production",
					},
				},
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check pool settings
	if config.Applications.Pools.MaxSize != 10 {
		t.Errorf("Pools.MaxSize = %v, want %v", config.Applications.Pools.MaxSize, 10)
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
				CookieName:     "test_cookie",
				CookieMaxAge:   "2h",
				CookieSecure:   true,
				CookieHTTPOnly: true,
				Paths:          []string{"/app/*", "/api/*"},
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
		t.Error("StickySession should be enabled")
	}

	if ss.CookieName != "test_cookie" {
		t.Errorf("CookieName = %v, want %v", ss.CookieName, "test_cookie")
	}

	if ss.CookieMaxAge != "2h" {
		t.Errorf("CookieMaxAge = %v, want %v", ss.CookieMaxAge, "2h")
	}

	if !ss.CookieSecure {
		t.Error("CookieSecure should be true")
	}

	if !ss.CookieHTTPOnly {
		t.Error("CookieHTTPOnly should be true")
	}

	if len(ss.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(ss.Paths))
	}
}

func TestConfigParser_ParseAuthConfig(t *testing.T) {
	tests := []struct {
		name                  string
		yamlConfig            YAMLConfig
		wantAuthFile          string
		wantAuthExcludeCount  int
		wantAuthExclude       []string
		wantAuthPatternsCount int
	}{
		{
			name: "auth with glob patterns",
			yamlConfig: YAMLConfig{
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
					HTPasswd: "/etc/htpasswd",
					PublicPaths: []string{
						"*.css",
						"*.js",
						"*.png",
						"*.jpg",
						"*.gif",
						"/assets/",
						"/favicon.ico",
					},
				},
			},
			wantAuthFile:          "/etc/htpasswd",
			wantAuthExcludeCount:  7,
			wantAuthExclude:       []string{"*.css", "*.js", "*.png", "*.jpg", "*.gif", "/assets/", "/favicon.ico"},
			wantAuthPatternsCount: 0, // Glob patterns should NOT be compiled as regex
		},
		{
			name: "auth disabled",
			yamlConfig: YAMLConfig{
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
					Enabled:     false,
					HTPasswd:    "/etc/htpasswd",
					PublicPaths: []string{"*.css"},
				},
			},
			wantAuthFile:          "",
			wantAuthExcludeCount:  0,
			wantAuthPatternsCount: 0,
		},
		{
			name: "auth with mixed patterns",
			yamlConfig: YAMLConfig{
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
					HTPasswd: "/etc/htpasswd",
					PublicPaths: []string{
						"*.css",
						"/public/",
						"/health",
						"*.woff",
					},
				},
			},
			wantAuthFile:          "/etc/htpasswd",
			wantAuthExcludeCount:  4,
			wantAuthExclude:       []string{"*.css", "/public/", "/health", "*.woff"},
			wantAuthPatternsCount: 0, // None should be compiled as regex
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewConfigParser(&tt.yamlConfig)
			config, err := parser.Parse()

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if config.Server.Authentication != tt.wantAuthFile {
				t.Errorf("Server.Authentication = %v, want %v", config.Server.Authentication, tt.wantAuthFile)
			}

			if len(config.Server.AuthExclude) != tt.wantAuthExcludeCount {
				t.Errorf("Server.AuthExclude count = %v, want %v", len(config.Server.AuthExclude), tt.wantAuthExcludeCount)
			}

			if tt.wantAuthExclude != nil {
				for i, want := range tt.wantAuthExclude {
					if i >= len(config.Server.AuthExclude) || config.Server.AuthExclude[i] != want {
						t.Errorf("Server.AuthExclude[%d] = %v, want %v", i, config.Server.AuthExclude[i], want)
					}
				}
			}

			// Verify that glob patterns are NOT compiled as regex (AuthPatterns should be empty)
			if len(config.Server.AuthPatterns) != tt.wantAuthPatternsCount {
				t.Errorf("Server.AuthPatterns count = %v, want %v (glob patterns should not be compiled as regex)",
					len(config.Server.AuthPatterns), tt.wantAuthPatternsCount)
			}
		})
	}
}

func TestConfigParser_ParseRoutesConfig(t *testing.T) {
	yamlConfig := YAMLConfig{
		Routes: RoutesConfig{
			Redirects: []struct {
				From string `yaml:"from"`
				To   string `yaml:"to"`
			}{
				{From: "^/old/(.*)$", To: "/new/$1"},
			},
			Rewrites: []struct {
				From string `yaml:"from"`
				To   string `yaml:"to"`
			}{
				{From: "^/api/v1/(.*)$", To: "/v1/$1"},
			},
		},
	}

	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check that routes were converted to rewrite rules
	if len(config.Server.RewriteRules) != 2 {
		t.Errorf("Expected 2 rewrite rules, got %d", len(config.Server.RewriteRules))
	}

	// Check redirect rule
	redirectFound := false
	rewriteFound := false

	for _, rule := range config.Server.RewriteRules {
		if rule.Flag == "redirect" {
			redirectFound = true
			if rule.Replacement != "/new/$1" {
				t.Errorf("Redirect replacement = %v, want /new/$1", rule.Replacement)
			}
		}
		if rule.Flag == "last" {
			rewriteFound = true
			if rule.Replacement != "/v1/$1" {
				t.Errorf("Rewrite replacement = %v, want /v1/$1", rule.Replacement)
			}
		}
	}

	if !redirectFound {
		t.Error("Redirect rule not found")
	}

	if !rewriteFound {
		t.Error("Rewrite rule not found")
	}
}

// Tests for new proxy functionality

func TestParseReverseProxies(t *testing.T) {
	yamlConfig := &YAMLConfig{
		Routes: RoutesConfig{
			ReverseProxies: []ProxyRoute{
				{
					Path:      "^/api/",
					Target:    "http://backend:8080",
					WebSocket: false,
					Headers: map[string]string{
						"X-Forwarded-Proto": "$scheme",
					},
				},
				{
					Path:      "^/ws",
					Target:    "ws://ws-backend:9090",
					WebSocket: true,
				},
			},
		},
	}

	parser := NewConfigParser(yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(config.Routes.ReverseProxies) != 2 {
		t.Errorf("Expected 2 reverse proxies, got %d", len(config.Routes.ReverseProxies))
	}

	// Test first proxy (HTTP)
	httpProxy := config.Routes.ReverseProxies[0]
	if httpProxy.Path != "^/api/" {
		t.Errorf("Expected path '^/api/', got %s", httpProxy.Path)
	}
	if httpProxy.Target != "http://backend:8080" {
		t.Errorf("Expected target 'http://backend:8080', got %s", httpProxy.Target)
	}
	if httpProxy.WebSocket {
		t.Error("Expected WebSocket to be false for HTTP proxy")
	}
	if httpProxy.Headers["X-Forwarded-Proto"] != "$scheme" {
		t.Errorf("Expected header value '$scheme', got %s", httpProxy.Headers["X-Forwarded-Proto"])
	}

	// Test second proxy (WebSocket)
	wsProxy := config.Routes.ReverseProxies[1]
	if wsProxy.Path != "^/ws" {
		t.Errorf("Expected path '^/ws', got %s", wsProxy.Path)
	}
	if wsProxy.Target != "ws://ws-backend:9090" {
		t.Errorf("Expected target 'ws://ws-backend:9090', got %s", wsProxy.Target)
	}
	if !wsProxy.WebSocket {
		t.Error("Expected WebSocket to be true for WebSocket proxy")
	}
}

// TestParseStandaloneServers removed - use Routes.ReverseProxies instead

func TestParseFlyReplay(t *testing.T) {
	yamlConfig := &YAMLConfig{
		Routes: RoutesConfig{
			FlyReplay: []struct {
				Path   string `yaml:"path"`
				App    string `yaml:"app"`
				Region string `yaml:"region"`
				Status int    `yaml:"status"`
			}{
				{
					Path:   "^/pdf/",
					App:    "pdf-generator",
					Status: 307,
				},
				{
					Path:   "^/region-specific/",
					Region: "iad",
					Status: 302,
				},
			},
		},
	}

	parser := NewConfigParser(yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(config.Routes.FlyReplay) != 2 {
		t.Errorf("Expected 2 fly replay routes, got %d", len(config.Routes.FlyReplay))
	}

	// Test app-based routing
	appRoute := config.Routes.FlyReplay[0]
	if appRoute.Path != "^/pdf/" {
		t.Errorf("Expected path '^/pdf/', got %s", appRoute.Path)
	}
	if appRoute.App != "pdf-generator" {
		t.Errorf("Expected app 'pdf-generator', got %s", appRoute.App)
	}
	if appRoute.Status != 307 {
		t.Errorf("Expected status 307, got %d", appRoute.Status)
	}

	// Test region-based routing
	regionRoute := config.Routes.FlyReplay[1]
	if regionRoute.Path != "^/region-specific/" {
		t.Errorf("Expected path '^/region-specific/', got %s", regionRoute.Path)
	}
	if regionRoute.Region != "iad" {
		t.Errorf("Expected region 'iad', got %s", regionRoute.Region)
	}
	if regionRoute.Status != 302 {
		t.Errorf("Expected status 302, got %d", regionRoute.Status)
	}
}

func TestParseFlyReplayToRewriteRules(t *testing.T) {
	yamlConfig := &YAMLConfig{
		Routes: RoutesConfig{
			FlyReplay: []struct {
				Path   string `yaml:"path"`
				App    string `yaml:"app"`
				Region string `yaml:"region"`
				Status int    `yaml:"status"`
			}{
				{
					Path:   "^/pdf/",
					App:    "pdf-generator",
					Status: 307,
				},
				{
					Path:   "^/region-specific/",
					Region: "iad",
					Status: 302,
				},
				{
					Path:   "^/coquitlam/",
					Region: "sjc",
					Status: 0, // Should default to 307
				},
			},
		},
	}

	parser := NewConfigParser(yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check that fly_replay routes were converted to rewrite rules
	var flyReplayRules []RewriteRule
	for _, rule := range config.Server.RewriteRules {
		if strings.HasPrefix(rule.Flag, "fly-replay:") {
			flyReplayRules = append(flyReplayRules, rule)
		}
	}

	if len(flyReplayRules) != 3 {
		t.Errorf("Expected 3 fly-replay rewrite rules, got %d", len(flyReplayRules))
	}

	// Test app-based fly-replay rule
	appRule := flyReplayRules[0]
	if appRule.Pattern.String() != "^/pdf/" {
		t.Errorf("Expected pattern '^/pdf/', got %s", appRule.Pattern.String())
	}
	if appRule.Flag != "fly-replay:app=pdf-generator:307" {
		t.Errorf("Expected flag 'fly-replay:app=pdf-generator:307', got %s", appRule.Flag)
	}

	// Test region-based fly-replay rule
	regionRule := flyReplayRules[1]
	if regionRule.Pattern.String() != "^/region-specific/" {
		t.Errorf("Expected pattern '^/region-specific/', got %s", regionRule.Pattern.String())
	}
	if regionRule.Flag != "fly-replay:iad:302" {
		t.Errorf("Expected flag 'fly-replay:iad:302', got %s", regionRule.Flag)
	}

	// Test default status (should be 307)
	defaultRule := flyReplayRules[2]
	if defaultRule.Flag != "fly-replay:sjc:307" {
		t.Errorf("Expected flag 'fly-replay:sjc:307' (default status), got %s", defaultRule.Flag)
	}

	// Test pattern matching
	testPaths := []struct {
		path        string
		shouldMatch int // index of rule that should match, -1 if no match
	}{
		{"/pdf/document.pdf", 0},
		{"/region-specific/test", 1},
		{"/coquitlam/medal-ball/", 2},
		{"/other/path", -1},
	}

	for _, test := range testPaths {
		matchFound := false
		matchedRuleIndex := -1

		for i, rule := range flyReplayRules {
			if rule.Pattern.MatchString(test.path) {
				matchFound = true
				matchedRuleIndex = i
				break
			}
		}

		if test.shouldMatch == -1 {
			if matchFound {
				t.Errorf("Path %s should not match any fly-replay rule, but matched rule %d", test.path, matchedRuleIndex)
			}
		} else {
			if !matchFound {
				t.Errorf("Path %s should match fly-replay rule %d, but didn't match any", test.path, test.shouldMatch)
			} else if matchedRuleIndex != test.shouldMatch {
				t.Errorf("Path %s should match rule %d, but matched rule %d", test.path, test.shouldMatch, matchedRuleIndex)
			}
		}
	}
}

func TestParseEmptyProxyConfiguration(t *testing.T) {
	yamlConfig := &YAMLConfig{
		Routes: RoutesConfig{},
	}

	parser := NewConfigParser(yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(config.Routes.ReverseProxies) != 0 {
		t.Errorf("Expected 0 reverse proxies, got %d", len(config.Routes.ReverseProxies))
	}
	// StandaloneServers removed - using Routes.ReverseProxies instead
	if len(config.Routes.FlyReplay) != 0 {
		t.Errorf("Expected 0 fly replay routes, got %d", len(config.Routes.FlyReplay))
	}
}

func TestParseComplexProxyConfiguration(t *testing.T) {
	yamlConfig := &YAMLConfig{
		Routes: RoutesConfig{
			Redirects: []struct {
				From string `yaml:"from"`
				To   string `yaml:"to"`
			}{
				{From: "^/$", To: "/dashboard"},
			},
			Rewrites: []struct {
				From string `yaml:"from"`
				To   string `yaml:"to"`
			}{
				{From: "^/api/v1/(.*)", To: "/v1/$1"},
			},
			ReverseProxies: []ProxyRoute{
				{
					Path:   "^/api/",
					Target: "http://api-server:8080",
					Headers: map[string]string{
						"X-Forwarded-For": "$remote_addr",
					},
				},
			},
			FlyReplay: []struct {
				Path   string `yaml:"path"`
				App    string `yaml:"app"`
				Region string `yaml:"region"`
				Status int    `yaml:"status"`
			}{
				{
					Path:   "^/heavy-compute/",
					App:    "compute-app",
					Status: 307,
				},
			},
		},
		// StandaloneServers removed - using Routes.ReverseProxies instead
	}

	parser := NewConfigParser(yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
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
