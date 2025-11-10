package process

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/utils"
)

// ManagedProcess represents a managed external process
type ManagedProcess struct {
	Name        string
	Command     string
	Args        []string
	WorkingDir  string
	Env         map[string]string
	AutoRestart bool
	StartDelay  time.Duration
	Process     *exec.Cmd
	Cancel      context.CancelFunc
	Running     bool
	Stopping    bool // Flag to prevent multiple stop attempts
	mutex       sync.RWMutex
}

// Manager manages external processes
type Manager struct {
	processes []*ManagedProcess
	config    *config.Config
	mutex     sync.RWMutex
	wg        sync.WaitGroup
}

// NewManager creates a new process manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		processes: make([]*ManagedProcess, 0),
		config:    cfg,
	}
}

// buildManagedProcessConfigs returns the complete list of managed processes,
// including automatically injected processes like Vector
func buildManagedProcessConfigs(cfg *config.Config) []config.ManagedProcessConfig {
	processes := make([]config.ManagedProcessConfig, 0, len(cfg.ManagedProcesses)+1)

	// Add Vector as first process if enabled (high priority)
	if cfg.Logging.Vector.Enabled && cfg.Logging.Vector.Config != "" {
		vectorProc := config.ManagedProcessConfig{
			Name:        "vector",
			Command:     "vector",
			Args:        []string{"--config", cfg.Logging.Vector.Config},
			AutoRestart: true,
		}
		processes = append(processes, vectorProc)
	}

	// Add user-configured managed processes
	processes = append(processes, cfg.ManagedProcesses...)

	return processes
}

// StartManagedProcesses starts all configured managed processes
func (m *Manager) StartManagedProcesses() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Get complete list including Vector if enabled
	allProcesses := buildManagedProcessConfigs(m.config)

	for _, procConfig := range allProcesses {
		// Parse start delay
		startDelay := utils.ParseDurationWithContext(procConfig.StartDelay, 0, map[string]interface{}{
			"process": procConfig.Name,
		})

		process := &ManagedProcess{
			Name:        procConfig.Name,
			Command:     procConfig.Command,
			Args:        procConfig.Args,
			WorkingDir:  procConfig.WorkingDir,
			Env:         procConfig.Env,
			AutoRestart: procConfig.AutoRestart,
			StartDelay:  startDelay,
		}

		m.processes = append(m.processes, process)

		// Start the process with delay if configured
		if startDelay > 0 {
			m.wg.Add(1)
			go func(p *ManagedProcess) {
				defer m.wg.Done()
				time.Sleep(p.StartDelay)
				if err := m.startProcess(p); err != nil {
					slog.Error("Failed to start managed process after delay",
						"process", p.Name,
						"error", err)
				}
			}(process)
		} else {
			if err := m.startProcess(process); err != nil {
				slog.Error("Failed to start managed process",
					"process", process.Name,
					"error", err)
			}
		}
	}

	return nil
}

// startProcess starts a single managed process
func (m *Manager) startProcess(proc *ManagedProcess) error {
	proc.mutex.Lock()
	defer proc.mutex.Unlock()

	if proc.Running {
		return fmt.Errorf("process %s is already running", proc.Name)
	}

	// Clean up Vector's Unix socket before starting (if this is Vector)
	if proc.Name == "vector" && m.config.Logging.Vector.Socket != "" {
		socketPath := m.config.Logging.Vector.Socket
		// Check if socket exists first
		if _, err := os.Stat(socketPath); err == nil {
			// Socket exists, remove it
			if err := os.Remove(socketPath); err != nil {
				slog.Warn("Failed to remove stale Vector socket",
					"socket", socketPath,
					"error", err)
			} else {
				slog.Info("Removed stale Vector socket", "socket", socketPath)
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	proc.Cancel = cancel

	cmd := exec.CommandContext(ctx, proc.Command, proc.Args...)
	if proc.WorkingDir != "" {
		cmd.Dir = proc.WorkingDir
	}

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range proc.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create log writers for the process output
	stdout := CreateLogWriter(proc.Name, "stdout", m.config.Logging)
	stderr := CreateLogWriter(proc.Name, "stderr", m.config.Logging)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	proc.Process = cmd

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process %s: %w", proc.Name, err)
	}

	proc.Running = true
	proc.Stopping = false // Reset stopping flag since we're starting
	slog.Info("Starting managed process", "name", proc.Name, "command", proc.Command, "args", proc.Args)

	// Monitor process
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		err := cmd.Wait()

		proc.mutex.Lock()
		proc.Running = false
		proc.Stopping = false // Reset stopping flag on exit
		wasAutoRestart := proc.AutoRestart
		proc.mutex.Unlock()

		if err != nil {
			slog.Error("Process exited with error",
				"name", proc.Name,
				"error", err)
		} else {
			slog.Info("Process exited normally", "name", proc.Name)
		}

		// Auto-restart if configured and not being explicitly stopped
		proc.mutex.Lock()
		isBeingStopped := proc.Stopping
		proc.mutex.Unlock()

		if wasAutoRestart && err != nil && !isBeingStopped {
			slog.Info("Auto-restarting process in 5 seconds", "name", proc.Name)
			time.Sleep(5 * time.Second) // Longer delay to ensure port cleanup

			// Double-check we're still supposed to restart
			proc.mutex.Lock()
			stillShouldRestart := proc.AutoRestart && !proc.Stopping
			proc.mutex.Unlock()

			if stillShouldRestart {
				if err := m.startProcess(proc); err != nil {
					slog.Error("Failed to restart managed process",
						"name", proc.Name,
						"error", err)
				}
			}
		}
	}()

	return nil
}

// StopManagedProcesses stops all managed processes
func (m *Manager) StopManagedProcesses() {
	m.StopManagedProcessesWithContext(context.Background())
}

// StopManagedProcessesWithContext stops all managed processes with context deadline
func (m *Manager) StopManagedProcessesWithContext(ctx context.Context) {
	m.mutex.RLock()
	processesCopy := make([]*ManagedProcess, len(m.processes))
	copy(processesCopy, m.processes)
	m.mutex.RUnlock()

	// First pass: mark all processes as stopping and disable auto-restart
	for _, proc := range processesCopy {
		proc.mutex.Lock()
		if proc.Running && !proc.Stopping {
			slog.Info("Stopping process", "name", proc.Name)
			proc.AutoRestart = false // Prevent auto-restart
			proc.Stopping = true     // Mark as being stopped
			if proc.Cancel != nil {
				proc.Cancel()
			}
		}
		proc.mutex.Unlock()
	}

	// Wait for all processes to finish with context awareness
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	// Use context timeout if available, otherwise fall back to default timeout
	timeout := config.ProcessStopTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	select {
	case <-done:
		slog.Info("All managed processes stopped")
	case <-ctx.Done():
		slog.Warn("Context deadline exceeded during managed process shutdown")
	case <-time.After(timeout):
		slog.Warn("Timeout waiting for managed processes to stop")
	}
}

// UpdateManagedProcesses updates managed processes after configuration reload
func (m *Manager) UpdateManagedProcesses(newConfig *config.Config) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Create maps for comparison
	oldProcs := make(map[string]*ManagedProcess)
	for _, proc := range m.processes {
		oldProcs[proc.Name] = proc
	}

	// Get complete list including Vector if enabled
	allNewProcesses := buildManagedProcessConfigs(newConfig)

	newProcs := make(map[string]*config.ManagedProcessConfig)
	for i := range allNewProcesses {
		proc := &allNewProcesses[i]
		newProcs[proc.Name] = proc
	}

	// Stop removed processes
	for name, proc := range oldProcs {
		if _, exists := newProcs[name]; !exists {
			slog.Info("Stopping removed managed process", "name", name)
			proc.mutex.Lock()
			if proc.Running && proc.Cancel != nil {
				proc.AutoRestart = false
				proc.Stopping = true
				proc.Cancel()
			}
			proc.mutex.Unlock()
		}
	}

	// Start new processes
	for name, procConfig := range newProcs {
		if _, exists := oldProcs[name]; !exists {
			slog.Info("Starting new managed process", "name", name)

			var startDelay time.Duration
			if procConfig.StartDelay != "" {
				startDelay, _ = time.ParseDuration(procConfig.StartDelay)
			}

			process := &ManagedProcess{
				Name:        procConfig.Name,
				Command:     procConfig.Command,
				Args:        procConfig.Args,
				WorkingDir:  procConfig.WorkingDir,
				Env:         procConfig.Env,
				AutoRestart: procConfig.AutoRestart,
				StartDelay:  startDelay,
			}

			m.processes = append(m.processes, process)
			if err := m.startProcess(process); err != nil {
				slog.Error("Failed to start new managed process",
					"process", process.Name,
					"error", err)
			}
		}
	}

	// Update configuration
	m.config = newConfig
}
