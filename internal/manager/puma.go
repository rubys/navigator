package manager

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logger"
)

// PumaProcess represents a running Puma instance
type PumaProcess struct {
	Tenant      *config.Tenant
	Port        int
	Cmd         *exec.Cmd
	StartedAt   time.Time
	LastUsed    time.Time
	mu          sync.RWMutex
	stopChan    chan struct{}
	healthCheck *time.Timer
}

// Config for PumaManager
type Config struct {
	RailsRoot    string
	DbPath       string
	StoragePath  string
	LogPath      string
	MaxProcesses int
	IdleTimeout  time.Duration
	Region       string
	AppName      string
}

// PumaManager manages Puma processes for different tenants
type PumaManager struct {
	Config    Config                  // Exported for access by proxy
	processes map[string]*PumaProcess // key is tenant label
	portPool  *PortPool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// PortPool manages available ports for Puma processes
type PortPool struct {
	basePort  int
	maxPorts  int
	available []int
	inUse     map[int]bool
	mu        sync.Mutex
}

// NewPortPool creates a new port pool
func NewPortPool(basePort, maxPorts int) *PortPool {
	pool := &PortPool{
		basePort: basePort,
		maxPorts: maxPorts,
		inUse:    make(map[int]bool),
	}

	// Initialize available ports
	for i := 0; i < maxPorts; i++ {
		pool.available = append(pool.available, basePort+i)
	}

	return pool
}

// Get allocates a port from the pool
func (p *PortPool) Get() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.available) == 0 {
		return 0, fmt.Errorf("no available ports")
	}

	port := p.available[0]
	p.available = p.available[1:]
	p.inUse[port] = true

	return port, nil
}

// Release returns a port to the pool
func (p *PortPool) Release(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.inUse[port] {
		delete(p.inUse, port)
		p.available = append(p.available, port)
	}
}

// NewPumaManager creates a new Puma process manager
func NewPumaManager(cfg Config) *PumaManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &PumaManager{
		Config:    cfg,
		processes: make(map[string]*PumaProcess),
		portPool:  NewPortPool(4000, cfg.MaxProcesses), // Puma processes start at port 4000
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start idle process cleanup routine
	go manager.cleanupIdleProcesses()

	return manager
}

// GetOrStart gets an existing Puma process or starts a new one
func (m *PumaManager) GetOrStart(tenant *config.Tenant) (*PumaProcess, error) {
	m.mu.RLock()
	process, exists := m.processes[tenant.Label]
	m.mu.RUnlock()

	if exists {
		process.mu.Lock()
		process.LastUsed = time.Now()
		process.mu.Unlock()

		// Check if process is still alive
		if process.Cmd != nil && process.Cmd.Process != nil {
			if err := process.Cmd.Process.Signal(syscall.Signal(0)); err == nil {
				return process, nil
			}
		}

		// Process died, remove it
		m.mu.Lock()
		delete(m.processes, tenant.Label)
		m.mu.Unlock()
		m.portPool.Release(process.Port)
	}

	// Start new process
	return m.Start(tenant)
}

// Start starts a new Puma process for a tenant
func (m *PumaManager) Start(tenant *config.Tenant) (*PumaProcess, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if process, exists := m.processes[tenant.Label]; exists {
		return process, nil
	}

	// Allocate port
	port, err := m.portPool.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate port: %w", err)
	}

	// Set port on tenant
	tenant.PumaPort = port

	// Create process
	process := &PumaProcess{
		Tenant:    tenant,
		Port:      port,
		StartedAt: time.Now(),
		LastUsed:  time.Now(),
		stopChan:  make(chan struct{}),
	}

	// Build Puma command
	cmd := exec.Command(
		"bundle", "exec", "puma",
		"-p", fmt.Sprintf("%d", port),
		"-e", "production",
		"-t", "5:5", // 5 threads
		"-w", "2", // 2 workers
		"--pidfile", filepath.Join(m.Config.RailsRoot, "tmp/pids", fmt.Sprintf("%s.pid", tenant.Label)),
	)

	// Set working directory
	cmd.Dir = m.Config.RailsRoot

	// Set environment
	cmd.Env = append(os.Environ(), tenant.GetEnvironment(m.Config.RailsRoot, m.Config.DbPath, m.Config.StoragePath)...)

	// Set up logging
	logPath := m.Config.LogPath
	if logPath == "" {
		logPath = filepath.Join(m.Config.RailsRoot, "log")
	}
	logFile := filepath.Join(logPath, fmt.Sprintf("%s.log", tenant.Label))
	if file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		cmd.Stdout = file
		cmd.Stderr = file
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		m.portPool.Release(port)
		return nil, fmt.Errorf("failed to start Puma: %w", err)
	}

	process.Cmd = cmd

	// Wait for Puma to be ready
	if err := m.waitForPuma(port, 30*time.Second); err != nil {
		cmd.Process.Kill()
		m.portPool.Release(port)
		return nil, fmt.Errorf("Puma failed to start: %w", err)
	}

	// Store process
	m.processes[tenant.Label] = process

	logger.WithFields(map[string]interface{}{
		"tenant": tenant.Label,
		"port":   port,
	}).Info("Started Puma process")

	// Monitor process
	go m.monitorProcess(process)

	return process, nil
}

// Stop stops a Puma process
func (m *PumaManager) Stop(label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	process, exists := m.processes[label]
	if !exists {
		return nil
	}

	// Signal stop
	close(process.stopChan)

	// Try graceful shutdown first
	if process.Cmd != nil && process.Cmd.Process != nil {
		process.Cmd.Process.Signal(syscall.SIGTERM)

		// Wait up to 10 seconds for graceful shutdown
		done := make(chan struct{})
		go func() {
			process.Cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Graceful shutdown succeeded
		case <-time.After(10 * time.Second):
			// Force kill
			process.Cmd.Process.Kill()
		}
	}

	// Release port
	m.portPool.Release(process.Port)

	// Remove from map
	delete(m.processes, label)

	logger.WithField("tenant", label).Info("Stopped Puma process")

	return nil
}

// StopAll stops all Puma processes
func (m *PumaManager) StopAll() {
	m.cancel() // Cancel context to stop cleanup routine

	m.mu.RLock()
	labels := make([]string, 0, len(m.processes))
	for label := range m.processes {
		labels = append(labels, label)
	}
	m.mu.RUnlock()

	for _, label := range labels {
		m.Stop(label)
	}
}

// GetProcess returns the process for a tenant if running
func (m *PumaManager) GetProcess(label string) *PumaProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processes[label]
}

// ListProcesses returns all running processes
func (m *PumaManager) ListProcesses() []*PumaProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*PumaProcess, 0, len(m.processes))
	for _, p := range m.processes {
		list = append(list, p)
	}
	return list
}

// waitForPuma waits for Puma to start accepting connections
func (m *PumaManager) waitForPuma(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for Puma to start on port %d", port)
}

// monitorProcess monitors a Puma process for crashes
func (m *PumaManager) monitorProcess(process *PumaProcess) {
	select {
	case <-process.stopChan:
		// Intentional stop
		return
	default:
		// Wait for process to exit
		if process.Cmd != nil {
			process.Cmd.Wait()

			// Process crashed, clean up
			m.mu.Lock()
			if current, exists := m.processes[process.Tenant.Label]; exists && current == process {
				delete(m.processes, process.Tenant.Label)
				m.portPool.Release(process.Port)
				logger.WithField("tenant", process.Tenant.Label).Warn("Puma process crashed")
			}
			m.mu.Unlock()
		}
	}
}

// cleanupIdleProcesses periodically stops idle Puma processes
func (m *PumaManager) cleanupIdleProcesses() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			now := time.Now()
			toStop := []string{}

			for label, process := range m.processes {
				process.mu.RLock()
				idle := now.Sub(process.LastUsed)
				process.mu.RUnlock()

				if idle > m.Config.IdleTimeout {
					toStop = append(toStop, label)
				}
			}
			m.mu.RUnlock()

			for _, label := range toStop {
				logger.WithField("tenant", label).Info("Stopping idle Puma process")
				m.Stop(label)
			}
		}
	}
}

// Touch updates the last used time for a process
func (p *PumaProcess) Touch() {
	p.mu.Lock()
	p.LastUsed = time.Now()
	p.mu.Unlock()
}

// IsHealthy checks if the process is healthy
func (p *PumaProcess) IsHealthy() bool {
	if p.Cmd == nil || p.Cmd.Process == nil {
		return false
	}

	// Check if process is still running
	if err := p.Cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	// Check if port is listening
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", p.Port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}
