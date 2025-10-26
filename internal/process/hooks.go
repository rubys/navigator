package process

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/utils"
)

// ExecuteHooks executes a list of hook commands with the given environment
func ExecuteHooks(hooks []config.HookConfig, env map[string]string, hookType string) error {
	for i, hook := range hooks {
		if hook.Command == "" {
			continue
		}

		// Parse timeout
		timeout := utils.ParseDurationWithContext(hook.Timeout, 0, map[string]interface{}{
			"hookType": hookType,
			"index":    i,
		})

		// Create command with or without timeout
		var cmd *exec.Cmd
		if timeout > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			cmd = exec.CommandContext(ctx, hook.Command, hook.Args...)
		} else {
			cmd = exec.Command(hook.Command, hook.Args...)
		}

		// Set environment if provided
		if env != nil {
			cmd.Env = os.Environ()
			for key, value := range env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
			}
		}

		// Log command execution
		slog.Info("Executing hook",
			"type", hookType,
			"command", hook.Command,
			"args", hook.Args,
			"timeout", timeout)

		// Execute and wait
		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("Hook execution failed",
				"type", hookType,
				"command", hook.Command,
				"error", err,
				"output", string(output))
			return fmt.Errorf("hook %s failed: %w", hookType, err)
		}

		if len(output) > 0 {
			slog.Debug("Hook output",
				"type", hookType,
				"command", hook.Command,
				"output", string(output))
		}
	}
	return nil
}

// HookResult contains the result of executing hooks, including reload decision
type HookResult struct {
	Error          error
	ReloadDecision utils.ReloadDecision
}

// ExecuteServerHooks executes server lifecycle hooks
func ExecuteServerHooks(hooks []config.HookConfig, hookType string) error {
	return ExecuteHooks(hooks, nil, fmt.Sprintf("server.%s", hookType))
}

// ExecuteServerHooksWithReload executes server lifecycle hooks and checks for reload_config
// Returns HookResult containing any error and reload decision
func ExecuteServerHooksWithReload(hooks []config.HookConfig, hookType, currentConfigFile string) HookResult {
	// Record start time before executing hooks
	startTime := time.Now()

	// Execute all hooks
	err := ExecuteHooks(hooks, nil, fmt.Sprintf("server.%s", hookType))

	// Check if any hook specified reload_config
	var reloadConfigPath string
	for _, hook := range hooks {
		if hook.ReloadConfig != "" {
			reloadConfigPath = hook.ReloadConfig
			break // Use first non-empty reload_config
		}
	}

	// Determine if config should be reloaded
	reloadDecision := utils.ShouldReloadConfig(reloadConfigPath, currentConfigFile, startTime)

	return HookResult{
		Error:          err,
		ReloadDecision: reloadDecision,
	}
}

// ExecuteTenantHooks executes tenant lifecycle hooks
func ExecuteTenantHooks(defaultHooks, specificHooks []config.HookConfig, env map[string]string, tenantName, hookType string) error {
	// Execute default hooks first
	if err := ExecuteHooks(defaultHooks, env, fmt.Sprintf("tenant.%s.default", hookType)); err != nil {
		return err
	}

	// Then execute tenant-specific hooks
	if err := ExecuteHooks(specificHooks, env, fmt.Sprintf("tenant.%s.%s", hookType, tenantName)); err != nil {
		return err
	}

	return nil
}
