package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

func TestResponseRecorder(t *testing.T) {
	// Create a test ResponseWriter
	recorder := httptest.NewRecorder()

	// Create ResponseRecorder
	respRecorder := NewResponseRecorder(recorder, nil)

	// Test WriteHeader
	respRecorder.WriteHeader(http.StatusCreated)
	if respRecorder.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, respRecorder.statusCode)
	}

	// Test Write
	testData := []byte("test response")
	n, err := respRecorder.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(testData))
	}
	if respRecorder.size != len(testData) {
		t.Errorf("Size = %d, expected %d", respRecorder.size, len(testData))
	}

	// Test SetMetadata
	respRecorder.SetMetadata("test_key", "test_value")
	if respRecorder.metadata["test_key"] != "test_value" {
		t.Errorf("Metadata not set correctly")
	}

	// Verify underlying recorder received the data
	if recorder.Code != http.StatusCreated {
		t.Errorf("Underlying recorder code = %d, expected %d", recorder.Code, http.StatusCreated)
	}
	if recorder.Body.String() != string(testData) {
		t.Errorf("Underlying recorder body = %q, expected %q", recorder.Body.String(), string(testData))
	}
}

func TestFindBestLocation(t *testing.T) {
	cfg := &config.Config{
		Locations: []config.Location{
			{
				Path:      "/api/*",
				ProxyPass: "http://localhost:4001",
			},
			{
				Path:      "/admin/*",
				ProxyPass: "http://localhost:4002",
			},
			{
				Path:      "/*/cable",
				ProxyPass: "http://localhost:4003",
			},
		},
	}

	handler := &Handler{config: cfg}

	tests := []struct {
		path     string
		expected string // Expected proxy pass URL
	}{
		{"/api/users", "http://localhost:4001"},
		{"/admin/dashboard", "http://localhost:4002"},
		{"/app/cable", "http://localhost:4003"},
		{"/unknown/path", ""}, // No match
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			location := handler.findBestLocation(tt.path)

			if tt.expected == "" {
				if location != nil {
					t.Errorf("Expected no location match, got %v", location)
				}
			} else {
				if location == nil {
					t.Errorf("Expected location match, got nil")
				} else if location.ProxyPass != tt.expected {
					t.Errorf("Expected proxy pass %s, got %s", tt.expected, location.ProxyPass)
				}
			}
		})
	}
}

func TestHealthCheckHandler(t *testing.T) {
	cfg := &config.Config{}
	handler := &Handler{config: cfg}

	req := httptest.NewRequest("GET", "/up", nil)
	recorder := httptest.NewRecorder()

	handler.handleHealthCheck(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	expectedBody := "OK"
	if recorder.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, recorder.Body.String())
	}

	expectedContentType := "text/html"
	if recorder.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("Expected Content-Type %q, got %q", expectedContentType, recorder.Header().Get("Content-Type"))
	}
}

func TestSetContentType(t *testing.T) {
	tests := []struct {
		filename    string
		expected    string
	}{
		{"test.html", "text/html; charset=utf-8"},
		{"test.css", "text/css"},
		{"test.js", "application/javascript"},
		{"test.json", "application/json; charset=utf-8"},
		{"test.png", "image/png"},
		{"test.jpg", "image/jpeg"},
		{"test.gif", "image/gif"},
		{"test.svg", "image/svg+xml"},
		{"test.pdf", "application/pdf"},
		{"test.txt", "text/plain; charset=utf-8"},
		{"test.unknown", ""}, // Default case returns empty
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			setContentType(recorder, tt.filename)

			contentType := recorder.Header().Get("Content-Type")
			if contentType != tt.expected {
				t.Errorf("For %s, expected %s, got %s", tt.filename, tt.expected, contentType)
			}
		})
	}
}

func TestLocationMatching(t *testing.T) {
	cfg := &config.Config{
		Locations: []config.Location{
			{Path: "/api/*", ProxyPass: "http://localhost:4001"},
			{Path: "/admin/*", ProxyPass: "http://localhost:4002"},
			{Path: "/*/cable", ProxyPass: "http://localhost:4003"},
		},
	}

	handler := &Handler{config: cfg}

	tests := []struct {
		path     string
		expected string
	}{
		{"/api/users", "/api/*"},
		{"/admin/dashboard", "/admin/*"},
		{"/2025/cable", "/*/cable"},
		{"/unknown/path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			location := handler.findBestLocation(tt.path)
			if tt.expected == "" {
				if location != nil {
					t.Errorf("For path %s, expected nil location, got %s", tt.path, location.Path)
				}
			} else {
				if location == nil {
					t.Errorf("For path %s, expected location %s, got nil", tt.path, tt.expected)
				} else if location.Path != tt.expected {
					t.Errorf("For path %s, expected location %s, got %s", tt.path, tt.expected, location.Path)
				}
			}
		})
	}
}

func BenchmarkResponseRecorder(b *testing.B) {
	recorder := httptest.NewRecorder()
	testData := []byte("benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		respRecorder := NewResponseRecorder(recorder, nil)
		respRecorder.WriteHeader(http.StatusOK)
		respRecorder.Write(testData)
		respRecorder.SetMetadata("test", "value")
	}
}

func BenchmarkFindBestLocation(b *testing.B) {
	cfg := &config.Config{
		Locations: []config.Location{
			{Path: "/api/*", ProxyPass: "http://localhost:4001"},
			{Path: "/admin/*", ProxyPass: "http://localhost:4002"},
			{Path: "/*/cable", ProxyPass: "http://localhost:4003"},
			{Path: "/assets/*", ProxyPass: "http://localhost:4004"},
			{Path: "/uploads/*", ProxyPass: "http://localhost:4005"},
		},
	}

	handler := &Handler{config: cfg}
	testPath := "/api/users/123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.findBestLocation(testPath)
	}
}