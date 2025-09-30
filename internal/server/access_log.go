package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// AccessLogEntry represents a structured access log entry matching nginx format
type AccessLogEntry struct {
	Timestamp     string `json:"@timestamp"`
	ClientIP      string `json:"client_ip"`
	RemoteUser    string `json:"remote_user"`
	Method        string `json:"method"`
	URI           string `json:"uri"`
	Protocol      string `json:"protocol"`
	Status        int    `json:"status"`
	BodyBytesSent int    `json:"body_bytes_sent"`
	RequestID     string `json:"request_id"`
	RequestTime   string `json:"request_time"`
	Referer       string `json:"referer"`
	UserAgent     string `json:"user_agent"`
	FlyRequestID  string `json:"fly_request_id"`
	Tenant        string `json:"tenant,omitempty"`
	ResponseType  string `json:"response_type,omitempty"` // Type of response: proxy, static, redirect, fly-replay, auth-failure, error
	Destination   string `json:"destination,omitempty"`   // For fly-replay or redirect responses
	ProxyBackend  string `json:"proxy_backend,omitempty"` // For proxy responses
	FilePath      string `json:"file_path,omitempty"`     // For static file responses
	ErrorMessage  string `json:"error_message,omitempty"` // For error responses
}

// LogRequest logs an HTTP request in JSON format matching nginx/legacy navigator format
func LogRequest(req *http.Request, statusCode, bodySize int, startTime time.Time, metadata map[string]interface{}, disableLog bool) {
	// Skip logging if disabled (e.g., during tests)
	if disableLog {
		return
	}

	// Get client IP (prefer X-Forwarded-For if available)
	clientIP := req.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = req.RemoteAddr
	}
	// Clean up client IP (remove port if present)
	if idx := strings.LastIndex(clientIP, ":"); idx > 0 && strings.Count(clientIP, ":") == 1 {
		clientIP = clientIP[:idx]
	}

	// Get remote user from basic auth or headers
	remoteUser := "-"
	if user, _, ok := req.BasicAuth(); ok && user != "" {
		remoteUser = user
	} else if user := req.Header.Get("X-Remote-User"); user != "" {
		remoteUser = user
	}

	// Calculate request duration
	duration := time.Since(startTime)
	requestTime := fmt.Sprintf("%.3f", duration.Seconds())

	// Get request ID from headers
	requestID := req.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = req.Header.Get("X-Amzn-Trace-Id")
	}

	// Get Fly request ID
	flyRequestID := req.Header.Get("Fly-Request-Id")

	// Build URI including query string
	uri := req.URL.Path
	if req.URL.RawQuery != "" {
		uri += "?" + req.URL.RawQuery
	}

	// Create access log entry
	entry := AccessLogEntry{
		Timestamp:     time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
		ClientIP:      clientIP,
		RemoteUser:    remoteUser,
		Method:        req.Method,
		URI:           uri,
		Protocol:      req.Proto,
		Status:        statusCode,
		BodyBytesSent: bodySize,
		RequestID:     requestID,
		RequestTime:   requestTime,
		Referer:       req.Header.Get("Referer"),
		UserAgent:     req.Header.Get("User-Agent"),
		FlyRequestID:  flyRequestID,
	}

	// Add metadata from the recorder
	if tenant, ok := metadata["tenant"].(string); ok {
		entry.Tenant = tenant
	}
	if responseType, ok := metadata["response_type"].(string); ok {
		entry.ResponseType = responseType
	}
	if destination, ok := metadata["destination"].(string); ok {
		entry.Destination = destination
	}
	if proxyBackend, ok := metadata["proxy_backend"].(string); ok {
		entry.ProxyBackend = proxyBackend
	}
	if filePath, ok := metadata["file_path"].(string); ok {
		entry.FilePath = filePath
	}
	if errorMessage, ok := metadata["error_message"].(string); ok {
		entry.ErrorMessage = errorMessage
	}

	// Output JSON log entry (matching nginx/rails format)
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stdout, string(data))
}
