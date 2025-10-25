package process

import (
	"fmt"
	"net"
	"sync"
)

// PortAllocator handles finding available ports for web applications
type PortAllocator struct {
	minPort        int
	maxPort        int
	allocatedPorts map[int]bool
	mutex          sync.Mutex
}

// NewPortAllocator creates a new port allocator
func NewPortAllocator(minPort, maxPort int) *PortAllocator {
	return &PortAllocator{
		minPort:        minPort,
		maxPort:        maxPort,
		allocatedPorts: make(map[int]bool),
	}
}

// AllocatePort finds and reserves an available port in the configured range
func (pa *PortAllocator) AllocatePort() (int, error) {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	for port := pa.minPort; port <= pa.maxPort; port++ {
		// Skip already allocated ports
		if pa.allocatedPorts[port] {
			continue
		}

		// Try to listen on the port to verify it's available
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			// Port is available - close the test listener and mark as allocated
			listener.Close()
			pa.allocatedPorts[port] = true
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", pa.minPort, pa.maxPort)
}

// ReleasePort releases a previously allocated port back to the pool
func (pa *PortAllocator) ReleasePort(port int) {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()
	delete(pa.allocatedPorts, port)
}

// FindAvailablePort is deprecated - use AllocatePort instead
// Kept for backward compatibility
func (pa *PortAllocator) FindAvailablePort() (int, error) {
	return pa.AllocatePort()
}
