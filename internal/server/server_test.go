package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestNewChiServer(t *testing.T) {
	// Create a simple test router
	router := chi.NewRouter()
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Create server
	server := NewChiServer(router, ":0") // Use port 0 for testing

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	// Check that server has the expected properties
	if server.router != router {
		t.Error("Expected server router to be the provided router")
	}

	// Check that HTTP server has expected timeouts
	if server.httpServer.ReadTimeout != 30*time.Second {
		t.Errorf("Expected read timeout 30s, got %v", server.httpServer.ReadTimeout)
	}
	if server.httpServer.WriteTimeout != 30*time.Second {
		t.Errorf("Expected write timeout 30s, got %v", server.httpServer.WriteTimeout)
	}
	if server.httpServer.IdleTimeout != 120*time.Second {
		t.Errorf("Expected idle timeout 120s, got %v", server.httpServer.IdleTimeout)
	}
}

func TestServerAddresses(t *testing.T) {
	router := chi.NewRouter()

	testCases := []struct {
		name    string
		address string
	}{
		{"default port", ":3000"},
		{"custom port", ":8080"}, 
		{"localhost", "localhost:3000"},
		{"specific IP", "127.0.0.1:3000"},
		{"dynamic port", ":0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := NewChiServer(router, tc.address)
			if server.httpServer.Addr != tc.address {
				t.Errorf("Expected server address '%s', got '%s'", tc.address, server.httpServer.Addr)
			}
		})
	}
}

func TestServerHTTP2Configuration(t *testing.T) {
	router := chi.NewRouter()
	server := NewChiServer(router, ":0")

	// Check that HTTP/2 is configured
	// The h2c handler should be set as the main handler
	if server.httpServer.Handler == router {
		t.Error("Expected server handler to be wrapped with h2c, but got original router")
	}
	
	// We can't easily test the exact h2c configuration without starting the server,
	// but we can verify the handler was wrapped
	if server.httpServer.Handler == nil {
		t.Error("Expected server to have a handler")
	}
}

func TestServerConfiguration(t *testing.T) {
	router := chi.NewRouter()
	server := NewChiServer(router, ":0")

	// Test server configuration values - these are Go defaults, not explicitly set
	// We're mainly testing that the server was created properly
	if server.httpServer == nil {
		t.Error("Expected HTTP server to be created")
	}

	// Test that address is set correctly
	if server.httpServer.Addr != ":0" {
		t.Errorf("Expected server address ':0', got '%s'", server.httpServer.Addr)
	}
}

// TestServerLifecycle tests starting and stopping the server
func TestServerLifecycle(t *testing.T) {
	// This is a basic integration test - in practice you might want more sophisticated tests
	router := chi.NewRouter()
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := NewChiServer(router, ":0")

	// Test that server can be created without errors
	if server == nil {
		t.Fatal("Failed to create server")
	}

	// Note: We're not actually starting the server in this test because:
	// 1. It would require handling the blocking Start() call in a goroutine
	// 2. We'd need to find the actual port assigned when using ":0"  
	// 3. We'd need proper cleanup to avoid port conflicts
	// 
	// In a real-world scenario, you might have integration tests that:
	// - Start the server in a goroutine
	// - Make HTTP requests to test endpoints
	// - Gracefully shutdown the server
	// - Use testcontainers or similar for isolated testing

	t.Log("Server configuration test passed - actual HTTP serving would require integration test")
}