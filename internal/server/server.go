package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rubys/navigator/internal/logger"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// ChiServer represents an HTTP server using chi router
type ChiServer struct {
	router     *chi.Mux
	httpServer *http.Server
	listenAddr string
}

// NewChiServer creates a new server with chi router
func NewChiServer(router *chi.Mux, listenAddr string) *ChiServer {
	// Setup HTTP/2 support
	h2s := &http2.Server{}
	
	// Create the HTTP server
	httpServer := &http.Server{
		Addr:         listenAddr,
		Handler:      h2c.NewHandler(router, h2s), // HTTP/2 without TLS
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &ChiServer{
		router:     router,
		httpServer: httpServer,
		listenAddr: listenAddr,
	}
}

// Start starts the HTTP server
func (s *ChiServer) Start() error {
	logger.WithField("address", s.listenAddr).Info("Starting HTTP server with HTTP/2 support")
	
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}
	
	return nil
}

// Shutdown gracefully shuts down the server
func (s *ChiServer) Shutdown(ctx context.Context) error {
	logger.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}