package config_test

import (
	"github.com/rubys/navigator/internal/config"
	"testing"
)

func TestHooksConfigParsing(t *testing.T) {
	yamlContent := `
server:
  listen: 3000
  hostname: localhost

hooks:
  server:
    start:
      - command: /bin/start
        args: ["arg1"]
        timeout: 5s
    ready:
      - command: /bin/ready
    idle:
      - command: /bin/idle
        timeout: 10m
    resume:
      - command: /bin/resume
  tenant:
    start:
      - command: /bin/tenant-start
        args: ["tenant", "starting"]
    stop:
      - command: /bin/tenant-stop
        timeout: 2m

applications:
  pools:
    max_size: 5
    timeout: 5m
    start_port: 4000
  tenants:
    - path: /test/
`

	cfg, err := config.ParseYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Verify server hooks
	if len(cfg.Hooks.Start) != 1 {
		t.Errorf("Expected 1 server start hook, got %d", len(cfg.Hooks.Start))
	}
	if cfg.Hooks.Start[0].Command != "/bin/start" {
		t.Errorf("Expected server start hook command '/bin/start', got '%s'", cfg.Hooks.Start[0].Command)
	}

	if len(cfg.Hooks.Ready) != 1 {
		t.Errorf("Expected 1 server ready hook, got %d", len(cfg.Hooks.Ready))
	}

	if len(cfg.Hooks.Idle) != 1 {
		t.Errorf("Expected 1 server idle hook, got %d", len(cfg.Hooks.Idle))
	}

	if len(cfg.Hooks.Resume) != 1 {
		t.Errorf("Expected 1 server resume hook, got %d", len(cfg.Hooks.Resume))
	}

	// Verify tenant default hooks
	if len(cfg.Applications.Hooks.Start) != 1 {
		t.Errorf("Expected 1 tenant start hook, got %d", len(cfg.Applications.Hooks.Start))
	}
	if cfg.Applications.Hooks.Start[0].Command != "/bin/tenant-start" {
		t.Errorf("Expected tenant start hook command '/bin/tenant-start', got '%s'", cfg.Applications.Hooks.Start[0].Command)
	}

	if len(cfg.Applications.Hooks.Stop) != 1 {
		t.Errorf("Expected 1 tenant stop hook, got %d", len(cfg.Applications.Hooks.Stop))
	}
	if cfg.Applications.Hooks.Stop[0].Command != "/bin/tenant-stop" {
		t.Errorf("Expected tenant stop hook command '/bin/tenant-stop', got '%s'", cfg.Applications.Hooks.Stop[0].Command)
	}
	if cfg.Applications.Hooks.Stop[0].Timeout != "2m" {
		t.Errorf("Expected tenant stop hook timeout '2m', got '%s'", cfg.Applications.Hooks.Stop[0].Timeout)
	}
}

func TestHooksReloadConfig(t *testing.T) {
	yamlContent := `
server:
  listen: 3000

hooks:
  server:
    ready:
      - command: /bin/init.sh
        timeout: 5m
        reload_config: config/navigator.yml
`

	cfg, err := config.ParseYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Verify ready hook has reload_config set
	if len(cfg.Hooks.Ready) != 1 {
		t.Fatalf("Expected 1 ready hook, got %d", len(cfg.Hooks.Ready))
	}

	hook := cfg.Hooks.Ready[0]
	if hook.ReloadConfig != "config/navigator.yml" {
		t.Errorf("Expected reload_config 'config/navigator.yml', got '%s'", hook.ReloadConfig)
	}
}

func TestHooksReloadConfigOptional(t *testing.T) {
	yamlContent := `
server:
  listen: 3000

hooks:
  server:
    ready:
      - command: /bin/ready.sh
`

	cfg, err := config.ParseYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Verify ready hook without reload_config works
	if len(cfg.Hooks.Ready) != 1 {
		t.Fatalf("Expected 1 ready hook, got %d", len(cfg.Hooks.Ready))
	}

	hook := cfg.Hooks.Ready[0]
	if hook.ReloadConfig != "" {
		t.Errorf("Expected empty reload_config, got '%s'", hook.ReloadConfig)
	}
}
