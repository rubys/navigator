package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestErrTenantNotFound(t *testing.T) {
	err := ErrTenantNotFound("test-tenant")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "test-tenant") {
		t.Errorf("Error message should contain tenant name, got: %v", err)
	}
}

func TestErrPIDFileRead(t *testing.T) {
	baseErr := errors.New("file not found")
	err := ErrPIDFileRead("/tmp/test.pid", baseErr)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "/tmp/test.pid") {
		t.Errorf("Error message should contain path, got: %v", err)
	}
	if !errors.Is(err, baseErr) {
		t.Errorf("Expected wrapped error to be unwrappable")
	}
}

func TestErrNoAvailablePorts(t *testing.T) {
	err := ErrNoAvailablePorts(4000, 4099)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "4000") || !strings.Contains(err.Error(), "4099") {
		t.Errorf("Error message should contain port range, got: %v", err)
	}
}

func TestErrConfigLoad(t *testing.T) {
	baseErr := errors.New("permission denied")
	err := ErrConfigLoad("/etc/navigator.yml", baseErr)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "/etc/navigator.yml") {
		t.Errorf("Error message should contain config path, got: %v", err)
	}
	if !errors.Is(err, baseErr) {
		t.Errorf("Expected wrapped error to be unwrappable")
	}
}

func TestErrProxyConnection(t *testing.T) {
	baseErr := errors.New("connection refused")
	err := ErrProxyConnection("http://localhost:4000", baseErr)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "localhost:4000") {
		t.Errorf("Error message should contain target, got: %v", err)
	}
	if !errors.Is(err, baseErr) {
		t.Errorf("Expected wrapped error to be unwrappable")
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that all wrapped errors can be unwrapped
	baseErr := errors.New("base error")

	wrappedErrors := []error{
		ErrPIDFileRead("/tmp/test.pid", baseErr),
		ErrPIDFileRemove("/tmp/test.pid", baseErr),
		ErrProcessStart("test-process", baseErr),
		ErrWebAppStart(baseErr),
		ErrConfigParse(baseErr),
		ErrConfigLoad("/config.yml", baseErr),
		ErrProxyConnection("http://target", baseErr),
		ErrProxyRequest(baseErr),
		ErrAuthFileLoad("/auth.htpasswd", baseErr),
		ErrServerStart(baseErr),
		ErrServerShutdown(baseErr),
	}

	for _, err := range wrappedErrors {
		if !errors.Is(err, baseErr) {
			t.Errorf("Error should wrap base error: %v", err)
		}
	}
}
