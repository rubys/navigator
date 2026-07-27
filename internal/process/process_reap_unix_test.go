//go:build unix

package process

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
)

// pidReaped reports whether pid has been reaped. A zombie still occupies its
// PID, so kill(pid, 0) succeeds against one and only starts failing with
// ESRCH once the parent has waited on it. That is the production symptom
// this guards: navigator accumulated 1613 zombies over 41 days because
// StartWebApp never called Wait().
func pidReaped(pid int) bool {
	err := syscall.Kill(pid, 0)
	return errors.Is(err, syscall.ESRCH)
}

// waitForReap polls until pid is reaped or the deadline passes.
func waitForReap(t *testing.T, pid int, within time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if pidReaped(pid) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return pidReaped(pid)
}

// reapTestConfig builds a single-tenant config running the given command.
func reapTestConfig(name, runtime, server string, args []string) *config.Config {
	return &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{MaxSize: 5, StartPort: 4000, Timeout: "5m"},
			Tenants: []config.Tenant{{
				Name:    name,
				Root:    "/tmp",
				Runtime: runtime,
				Server:  server,
				Args:    args,
			}},
		},
	}
}

func startReapTestApp(t *testing.T, cfg *config.Config, port int) *WebApp {
	t.Helper()
	t.Setenv("NAVIGATOR_TEST_SKIP_READINESS", "true")

	appManager := NewAppManager(cfg)
	app := &WebApp{
		Port:          port,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		wsConnections: make(map[string]interface{}),
		readyChan:     make(chan struct{}),
	}
	if err := appManager.processStarter.StartWebApp(app, &cfg.Applications.Tenants[0]); err != nil {
		t.Fatalf("StartWebApp: %v", err)
	}
	if app.Process == nil || app.Process.Process == nil {
		t.Fatal("StartWebApp did not record a started process")
	}
	return app
}

// A web app that exits on its own must be reaped rather than left a zombie.
func TestStartWebAppReapsSelfExitedProcess(t *testing.T) {
	cfg := reapTestConfig("reap-self-exit", "true", "", nil)
	// `true` ignores argv and exits 0 immediately.
	cfg.Applications.Tenants[0].Args = []string{"ignored"}

	app := startReapTestApp(t, cfg, 4101)
	pid := app.Process.Process.Pid
	defer func() {
		if app.cancel != nil {
			app.cancel()
		}
	}()

	if !waitForReap(t, pid, 5*time.Second) {
		t.Fatalf("pid %d still exists after exiting — it was never reaped (missing cmd.Wait)", pid)
	}
}

// The production path: idle shutdown cancels the context, which kills the
// child but does NOT reap it. Every start/stop cycle leaked a PID this way.
func TestStartWebAppReapsCancelledProcess(t *testing.T) {
	// `sh -c 'sleep 30'` — args must be non-empty or getArgs substitutes the
	// default Rails server args, which would make this exit immediately and
	// silently test the self-exit path instead of cancellation.
	cfg := reapTestConfig("reap-cancelled", "sh", "-c", []string{"sleep 30"})

	app := startReapTestApp(t, cfg, 4102)
	pid := app.Process.Process.Pid

	// Confirm it is genuinely running before cancelling — otherwise this
	// test would pass for the wrong reason.
	time.Sleep(150 * time.Millisecond)
	if pidReaped(pid) {
		t.Fatalf("pid %d exited on its own; test cannot exercise the cancel path", pid)
	}

	// What monitorAppIdleTimeout does when an app goes idle.
	if app.cancel == nil {
		t.Fatal("StartWebApp did not record a cancel func")
	}
	app.cancel()

	if !waitForReap(t, pid, 5*time.Second) {
		t.Fatalf("pid %d still exists after cancellation — killed but never reaped", pid)
	}
}
