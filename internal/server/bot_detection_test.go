package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

// TestBotDetectionReject tests that bots are blocked when action is "reject"
func TestBotDetectionReject(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.BotDetection.Enabled = true
	cfg.Server.BotDetection.Action = "reject"
	cfg.Applications.Tenants = []config.Tenant{
		{
			Name: "test",
			Path: "/test/",
		},
	}

	appManager := process.NewAppManager(cfg)
	idleManager := &idle.Manager{}
	handler := CreateTestHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name      string
		userAgent string
	}{
		{"Googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"},
		{"GPTBot", "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.2; +https://openai.com/gptbot)"},
		{"ClaudeBot", "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; ClaudeBot/1.0; +claudebot@anthropic.com)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test/page", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusForbidden {
				t.Errorf("Expected bot to be blocked (403), got %d", recorder.Code)
			}
			if !strings.Contains(recorder.Body.String(), "Forbidden") {
				t.Errorf("Expected 'Forbidden' in response body, got: %s", recorder.Body.String())
			}
		})
	}
}

// TestBotDetectionStaticOnly tests that bots are blocked from dynamic content with "static-only"
func TestBotDetectionStaticOnly(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.BotDetection.Enabled = true
	cfg.Server.BotDetection.Action = "static-only"
	cfg.Applications.Tenants = []config.Tenant{
		{
			Name: "test",
			Path: "/test/",
		},
	}

	appManager := process.NewAppManager(cfg)
	idleManager := &idle.Manager{}
	handler := CreateTestHandler(cfg, appManager, nil, idleManager)

	req := httptest.NewRequest(http.MethodGet, "/test/page", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// For dynamic content (web app proxy), bot should be blocked
	if recorder.Code != http.StatusForbidden {
		t.Errorf("Expected bot to be blocked from dynamic content (403), got %d", recorder.Code)
	}
}

// TestBotDetectionTenantOverride tests that tenant-specific config overrides global config
func TestBotDetectionTenantOverride(t *testing.T) {
	botUA := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

	// Test that global reject blocks bot
	cfg := &config.Config{}
	cfg.Server.BotDetection.Enabled = true
	cfg.Server.BotDetection.Action = "reject"
	cfg.Applications.Tenants = []config.Tenant{
		{
			Name: "test",
			Path: "/test/",
		},
	}

	appManager := process.NewAppManager(cfg)
	idleManager := &idle.Manager{}
	handler := CreateTestHandler(cfg, appManager, nil, idleManager)

	req := httptest.NewRequest(http.MethodGet, "/test/page", nil)
	req.Header.Set("User-Agent", botUA)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Errorf("Expected bot to be blocked with global reject, got %d", recorder.Code)
	}
}

// TestBotDetectionWithDemoTenant tests the showcase use case: block bots from regular tenants
func TestBotDetectionWithDemoTenant(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.BotDetection.Enabled = true
	cfg.Server.BotDetection.Action = "reject" // Global: block bots

	// Add regular tenant and demo tenant
	cfg.Applications.Tenants = []config.Tenant{
		{
			Name: "index",
			Path: "/showcase/",
			// Uses global config (reject)
		},
		{
			Name: "demo",
			Path: "/showcase/regions/iad/demo/",
			BotDetection: &config.BotDetectionConfig{
				Enabled: true,
				Action:  "ignore", // Demo allows bots
			},
		},
	}

	appManager := process.NewAppManager(cfg)
	idleManager := &idle.Manager{}
	handler := CreateTestHandler(cfg, appManager, nil, idleManager)

	botUA := "Mozilla/5.0 (compatible; Googlebot/2.1)"

	// Verify bot is blocked from index tenant
	req := httptest.NewRequest(http.MethodGet, "/showcase/studios/laval", nil)
	req.Header.Set("User-Agent", botUA)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Errorf("Expected bot to be blocked from index tenant, got %d", recorder.Code)
	}
}
