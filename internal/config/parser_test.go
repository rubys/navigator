package config

import (
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
				Path      string                 `yaml:"path"`
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
				Path      string                 `yaml:"path"`
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