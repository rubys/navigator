package cgi

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/process"
	"github.com/rubys/navigator/internal/utils"
)

// Handler implements CGI script execution with user switching support
type Handler struct {
	Script          string
	User            string
	Group           string
	Env             map[string]string
	ReloadConfig    string
	Timeout         time.Duration
	CurrentConfigFn func() string // Function to get current config file path
	TriggerReloadFn func(string)  // Function to trigger config reload
}

// NewHandler creates a new CGI handler from configuration
func NewHandler(cfg *config.CGIScriptConfig, currentConfigFn func() string, triggerReloadFn func(string)) (*Handler, error) {
	// Validate script path
	if cfg.Script == "" {
		return nil, fmt.Errorf("CGI script path is required")
	}

	// Check if script exists and is executable
	info, err := os.Stat(cfg.Script)
	if err != nil {
		return nil, fmt.Errorf("CGI script not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("CGI script path is a directory: %s", cfg.Script)
	}
	// Check executable permission (Unix-specific check)
	if info.Mode()&0111 == 0 {
		slog.Warn("CGI script may not be executable", "script", cfg.Script, "mode", info.Mode())
	}

	// Parse timeout
	timeout := utils.ParseDurationWithContext(cfg.Timeout, 0, map[string]interface{}{
		"script": cfg.Script,
	})

	return &Handler{
		Script:          cfg.Script,
		User:            cfg.User,
		Group:           cfg.Group,
		Env:             cfg.Env,
		ReloadConfig:    cfg.ReloadConfig,
		Timeout:         timeout,
		CurrentConfigFn: currentConfigFn,
		TriggerReloadFn: triggerReloadFn,
	}, nil
}

// ServeHTTP implements the http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	slog.Info("Executing CGI script",
		"script", h.Script,
		"method", r.Method,
		"path", r.URL.Path,
		"user", h.User)

	// Create command with optional timeout
	var cmd *exec.Cmd
	var cancel context.CancelFunc
	if h.Timeout > 0 {
		ctx, cancelFn := context.WithTimeout(r.Context(), h.Timeout)
		cancel = cancelFn
		cmd = exec.CommandContext(ctx, h.Script)
	} else {
		cmd = exec.CommandContext(r.Context(), h.Script)
	}

	if cancel != nil {
		defer cancel()
	}

	// Set up CGI environment
	h.setupCGIEnvironment(cmd, r)

	// Set user credentials if specified (Unix only)
	if h.User != "" {
		cred, err := process.GetUserCredentials(h.User, h.Group)
		if err != nil {
			slog.Error("Failed to get user credentials for CGI script",
				"script", h.Script,
				"user", h.User,
				"error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if cred != nil {
			h.setProcessCredentials(cmd, cred)
		}
	}

	// Set working directory to script's directory
	cmd.Dir = filepath.Dir(h.Script)

	// Setup pipes for stdin/stdout/stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		slog.Error("Failed to create stdin pipe", "script", h.Script, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Failed to create stdout pipe", "script", h.Script, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Failed to create stderr pipe", "script", h.Script, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Start the CGI script
	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start CGI script", "script", h.Script, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Copy request body to script's stdin
	go func() {
		defer stdin.Close()
		if r.Body != nil {
			_, _ = io.Copy(stdin, r.Body)
		}
	}()

	// Read stderr in background
	stderrOutput := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(stderr)
		stderrOutput <- string(data)
	}()

	// Parse CGI response from stdout
	err = h.parseAndWriteCGIResponse(w, stdout)
	if err != nil {
		slog.Error("Failed to parse CGI response", "script", h.Script, "error", err)
		// Don't write error response here - may have already written headers
	}

	// Wait for command to finish
	cmdErr := cmd.Wait()

	// Log stderr if present
	select {
	case errOutput := <-stderrOutput:
		if errOutput != "" {
			slog.Debug("CGI script stderr", "script", h.Script, "stderr", errOutput)
		}
	default:
	}

	// Check command result
	if cmdErr != nil {
		slog.Error("CGI script execution failed",
			"script", h.Script,
			"error", cmdErr,
			"duration", time.Since(startTime))
		return
	}

	slog.Info("CGI script completed",
		"script", h.Script,
		"duration", time.Since(startTime))

	// Check if config should be reloaded
	if h.ReloadConfig != "" && h.CurrentConfigFn != nil && h.TriggerReloadFn != nil {
		currentConfig := h.CurrentConfigFn()
		decision := utils.ShouldReloadConfig(h.ReloadConfig, currentConfig, startTime)
		if decision.ShouldReload {
			slog.Info("CGI script triggered config reload",
				"script", h.Script,
				"reason", decision.Reason,
				"configFile", decision.NewConfigFile)
			h.TriggerReloadFn(decision.NewConfigFile)
		}
	}
}

// setupCGIEnvironment sets up standard CGI environment variables
func (h *Handler) setupCGIEnvironment(cmd *exec.Cmd, r *http.Request) {
	// Start with current environment
	cmd.Env = os.Environ()

	// Add custom environment variables from config
	for key, value := range h.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Standard CGI environment variables (RFC 3875)
	host := r.Host
	if host == "" {
		host = "localhost"
	}

	cgiEnv := map[string]string{
		"GATEWAY_INTERFACE": "CGI/1.1",
		"SERVER_PROTOCOL":   r.Proto,
		"SERVER_SOFTWARE":   "Navigator",
		"REQUEST_METHOD":    r.Method,
		"QUERY_STRING":      r.URL.RawQuery,
		"SCRIPT_NAME":       r.URL.Path,
		"PATH_INFO":         "",
		"SERVER_NAME":       host,
		"REMOTE_ADDR":       r.RemoteAddr,
		"CONTENT_TYPE":      r.Header.Get("Content-Type"),
		"CONTENT_LENGTH":    r.Header.Get("Content-Length"),
	}

	// Add CGI environment
	for key, value := range cgiEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add HTTP headers as HTTP_* variables
	for key, values := range r.Header {
		if key == "Content-Type" || key == "Content-Length" {
			continue // Already added above
		}
		headerKey := "HTTP_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", headerKey, strings.Join(values, ",")))
	}
}

// parseAndWriteCGIResponse parses CGI output and writes it to the response writer
func (h *Handler) parseAndWriteCGIResponse(w http.ResponseWriter, stdout io.Reader) error {
	reader := bufio.NewReader(stdout)

	// Read and parse headers
	statusCode := http.StatusOK
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading CGI headers: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")

		// Empty line marks end of headers
		if line == "" {
			break
		}

		// Parse header
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle special Status header
		if strings.EqualFold(key, "Status") {
			// Parse status code from "200 OK" format
			statusParts := strings.SplitN(value, " ", 2)
			if len(statusParts) > 0 {
				if code, err := strconv.Atoi(statusParts[0]); err == nil {
					statusCode = code
				}
			}
			continue
		}

		// Set header
		w.Header().Set(key, value)
	}

	// Write status code
	w.WriteHeader(statusCode)

	// Copy body
	_, err := io.Copy(w, reader)
	return err
}
