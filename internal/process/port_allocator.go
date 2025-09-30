package process

import (
	"fmt"
	"net"
)

// PortAllocator handles finding available ports for web applications
type PortAllocator struct {
	minPort int
	maxPort int
}

// NewPortAllocator creates a new port allocator
func NewPortAllocator(minPort, maxPort int) *PortAllocator {
	return &PortAllocator{
		minPort: minPort,
		maxPort: maxPort,
	}
}

// FindAvailablePort finds an available port in the configured range
func (pa *PortAllocator) FindAvailablePort() (int, error) {
	for port := pa.minPort; port <= pa.maxPort; port++ {
		// Try to listen on the port
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			// Port is available
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", pa.minPort, pa.maxPort)
}
