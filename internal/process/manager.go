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

// StartManagedProcesses starts all configured managed processes
func (m *Manager) StartManagedProcesses() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, procConfig := range m.config.ManagedProcesses {
		// Parse start delay
		var startDelay time.Duration
		if procConfig.StartDelay != "" {
			var err error
			startDelay, err = time.ParseDuration(procConfig.StartDelay)
			if err != nil {
				slog.Warn("Invalid start_delay format, using 0",
					"process", procConfig.Name,
					"delay", procConfig.StartDelay,
					"error", err)
				startDelay = 0
			}
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
	slog.Info("Started managed process", "name", proc.Name, "pid", cmd.Process.Pid)

	// Monitor process
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		err := cmd.Wait()

		proc.mutex.Lock()
		proc.Running = false
		wasAutoRestart := proc.AutoRestart
		proc.mutex.Unlock()

		if err != nil {
			slog.Error("Managed process exited with error",
				"name", proc.Name,
				"error", err)
		} else {
			slog.Info("Managed process exited cleanly", "name", proc.Name)
		}

		// Auto-restart if configured
		if wasAutoRestart && err != nil {
			slog.Info("Auto-restarting managed process", "name", proc.Name)
			time.Sleep(2 * time.Second) // Brief delay before restart
			if err := m.startProcess(proc); err != nil {
				slog.Error("Failed to restart managed process",
					"name", proc.Name,
					"error", err)
			}
		}
	}()

	return nil
}

// StopManagedProcesses stops all managed processes
func (m *Manager) StopManagedProcesses() {
	m.mutex.RLock()
	processesCopy := make([]*ManagedProcess, len(m.processes))
	copy(processesCopy, m.processes)
	m.mutex.RUnlock()

	for _, proc := range processesCopy {
		proc.mutex.Lock()
		if proc.Running && proc.Cancel != nil {
			slog.Info("Stopping managed process", "name", proc.Name)
			proc.AutoRestart = false // Prevent auto-restart
			proc.Cancel()
		}
		proc.mutex.Unlock()
	}

	// Wait for all processes to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("All managed processes stopped")
	case <-time.After(config.ProcessStopTimeout):
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

	newProcs := make(map[string]*config.ManagedProcessConfig)
	for i := range newConfig.ManagedProcesses {
		proc := &newConfig.ManagedProcesses[i]
		newProcs[proc.Name] = proc
	}

	// Stop removed processes
	for name, proc := range oldProcs {
		if _, exists := newProcs[name]; !exists {
			slog.Info("Stopping removed managed process", "name", name)
			proc.mutex.Lock()
			if proc.Running && proc.Cancel != nil {
				proc.AutoRestart = false
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