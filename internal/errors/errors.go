package errors

import "fmt"

// Process-related errors

// ErrTenantNotFound returns an error indicating the tenant was not found
func ErrTenantNotFound(name string) error {
	return fmt.Errorf("tenant %s not found", name)
}

// ErrPIDFileRead returns an error for PID file read failures
func ErrPIDFileRead(path string, err error) error {
	return fmt.Errorf("error reading PID file %s: %w", path, err)
}

// ErrPIDFileRemove returns an error for PID file removal failures
func ErrPIDFileRemove(path string, err error) error {
	return fmt.Errorf("error removing PID file %s: %w", path, err)
}

// ErrProcessStart returns an error for process start failures
func ErrProcessStart(name string, err error) error {
	return fmt.Errorf("failed to start process %s: %w", name, err)
}

// ErrWebAppStart returns an error for web app start failures
func ErrWebAppStart(err error) error {
	return fmt.Errorf("failed to start web app: %w", err)
}

// ErrNoAvailablePorts returns an error when no ports are available in range
func ErrNoAvailablePorts(minPort, maxPort int) error {
	return fmt.Errorf("no available ports in range %d-%d", minPort, maxPort)
}

// Config-related errors

// ErrConfigParse returns an error for configuration parsing failures
func ErrConfigParse(err error) error {
	return fmt.Errorf("failed to parse configuration: %w", err)
}

// ErrConfigLoad returns an error for configuration loading failures
func ErrConfigLoad(path string, err error) error {
	return fmt.Errorf("failed to load config from %s: %w", path, err)
}

// ErrConfigValidation returns an error for configuration validation failures
func ErrConfigValidation(msg string) error {
	return fmt.Errorf("configuration validation failed: %s", msg)
}

// Proxy-related errors

// ErrProxyConnection returns an error for proxy connection failures
func ErrProxyConnection(target string, err error) error {
	return fmt.Errorf("failed to connect to %s: %w", target, err)
}

// ErrProxyRequest returns an error for proxy request failures
func ErrProxyRequest(err error) error {
	return fmt.Errorf("proxy request failed: %w", err)
}

// Auth-related errors

// ErrAuthFileLoad returns an error for htpasswd file loading failures
func ErrAuthFileLoad(path string, err error) error {
	return fmt.Errorf("failed to load htpasswd file %s: %w", path, err)
}

// ErrAuthInvalid returns an error for invalid authentication
func ErrAuthInvalid(user string) error {
	return fmt.Errorf("invalid authentication for user %s", user)
}

// Server-related errors

// ErrServerStart returns an error for server start failures
func ErrServerStart(err error) error {
	return fmt.Errorf("failed to start server: %w", err)
}

// ErrServerShutdown returns an error for server shutdown failures
func ErrServerShutdown(err error) error {
	return fmt.Errorf("server shutdown error: %w", err)
}
