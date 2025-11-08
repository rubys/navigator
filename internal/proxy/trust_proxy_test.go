package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustProxySettings(t *testing.T) {
	tests := []struct {
		name                string
		trustProxy          bool
		incomingForwardHost string
		expectedForwardHost string
	}{
		{
			name:                "trust_proxy=false with no incoming header sets current host",
			trustProxy:          false,
			incomingForwardHost: "",
			expectedForwardHost: "example.com:8080",
		},
		{
			name:                "trust_proxy=false with incoming header overwrites with current host",
			trustProxy:          false,
			incomingForwardHost: "upstream.proxy.com",
			expectedForwardHost: "example.com:8080",
		},
		{
			name:                "trust_proxy=true with no incoming header sets current host",
			trustProxy:          true,
			incomingForwardHost: "",
			expectedForwardHost: "example.com:8080",
		},
		{
			name:                "trust_proxy=true with incoming header preserves it",
			trustProxy:          true,
			incomingForwardHost: "upstream.proxy.com",
			expectedForwardHost: "upstream.proxy.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set trust_proxy for this test
			SetTrustProxy(tt.trustProxy)

			// Create a test backend that captures the forwarded headers
			var receivedForwardHost string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedForwardHost = r.Header.Get("X-Forwarded-Host")
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			// Create a test request
			req := httptest.NewRequest("GET", "http://example.com:8080/test", nil)
			if tt.incomingForwardHost != "" {
				req.Header.Set("X-Forwarded-Host", tt.incomingForwardHost)
			}

			// Proxy the request
			recorder := httptest.NewRecorder()
			HandleProxy(recorder, req, backend.URL)

			// Verify the X-Forwarded-Host header received by backend
			if receivedForwardHost != tt.expectedForwardHost {
				t.Errorf("Expected X-Forwarded-Host=%q, got %q", tt.expectedForwardHost, receivedForwardHost)
			}
		})
	}
}

func TestGetTrustProxy(t *testing.T) {
	// Test that GetTrustProxy returns the value set by SetTrustProxy
	SetTrustProxy(false)
	if GetTrustProxy() != false {
		t.Error("Expected GetTrustProxy() to return false")
	}

	SetTrustProxy(true)
	if GetTrustProxy() != true {
		t.Error("Expected GetTrustProxy() to return true")
	}
}
