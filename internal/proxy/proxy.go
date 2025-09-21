package proxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rubys/navigator/internal/config"
)

// HandleProxy handles proxying requests to a target URL
func HandleProxy(w http.ResponseWriter, r *http.Request, targetURL string, location *config.Location) {
	target, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Invalid proxy target", http.StatusInternalServerError)
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the director to modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Preserve X-Forwarded headers
		if req.Header.Get("X-Forwarded-For") == "" {
			req.Header.Set("X-Forwarded-For", r.RemoteAddr)
		}
		req.Header.Set("X-Forwarded-Host", req.Host)
		if req.Header.Get("X-Forwarded-Proto") == "" {
			req.Header.Set("X-Forwarded-Proto", "http")
		}

		// Rewrite path if needed
		if location != nil && location.Alias != "" {
			req.URL.Path = location.Alias + req.URL.Path
		}
	}

	// Set error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("Proxy error", "target", targetURL, "error", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Perform the proxy request
	proxy.ServeHTTP(w, r)
}

// HandleProxyWithRetry handles proxying with automatic retry on connection failures
func HandleProxyWithRetry(w http.ResponseWriter, r *http.Request, targetURL string, maxRetryDuration time.Duration) {
	target, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Invalid proxy target", http.StatusInternalServerError)
		return
	}

	// Only retry for safe methods
	canRetry := r.Method == "GET" || r.Method == "HEAD"

	// Use RetryResponseWriter for safe methods
	var responseWriter http.ResponseWriter
	var retryWriter *RetryResponseWriter
	if canRetry {
		retryWriter = NewRetryResponseWriter(w)
		responseWriter = retryWriter
	} else {
		responseWriter = w
	}

	// Create custom transport with shorter timeout for connection attempts
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout: 500 * time.Millisecond,
			}
			return dialer.DialContext(ctx, network, addr)
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport

	// Implement retry logic
	startTime := time.Now()
	attempt := 0
	initialDelay := 100 * time.Millisecond
	maxDelay := 500 * time.Millisecond

	for {
		attempt++

		// Reset buffer for retry if applicable
		if canRetry && attempt > 1 && retryWriter != nil {
			retryWriter.Reset()
		}

		// Try the proxy request
		success := tryProxy(proxy, responseWriter, r)
		if success {
			// Commit the buffered response if using retry writer
			if retryWriter != nil {
				retryWriter.Commit()
			}
			return
		}

		// If we can't retry, fail immediately
		if !canRetry {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// Check if we've exceeded max retry duration
		if time.Since(startTime) >= maxRetryDuration {
			slog.Error("Proxy failed after max retry duration",
				"target", targetURL,
				"attempts", attempt,
				"duration", time.Since(startTime))
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// Calculate exponential backoff delay
		delay := initialDelay * time.Duration(1<<uint(attempt-1))
		if delay > maxDelay {
			delay = maxDelay
		}

		slog.Debug("Proxy retry",
			"target", targetURL,
			"attempt", attempt,
			"delay", delay)

		time.Sleep(delay)
	}
}

// tryProxy attempts a single proxy request
func tryProxy(proxy *httputil.ReverseProxy, w http.ResponseWriter, r *http.Request) bool {
	// Use a custom response writer to capture errors
	recorder := &proxyRecorder{
		ResponseWriter: w,
		success:        true,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		recorder.success = false
		// Don't write error response yet - we might retry
	}

	proxy.ServeHTTP(recorder, r)
	return recorder.success
}

// proxyRecorder captures proxy success/failure
type proxyRecorder struct {
	http.ResponseWriter
	success bool
}

// IsWebSocketRequest checks if request is a WebSocket upgrade
func IsWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// WebSocketTracker wraps ResponseWriter to track WebSocket connections
type WebSocketTracker struct {
	http.ResponseWriter
	ActiveWebSockets *int32
	Cleaned          bool
}

// Hijack implements http.Hijacker interface for WebSocket support
func (w *WebSocketTracker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		conn, rw, err := hijacker.Hijack()
		if err != nil {
			// If hijack fails, decrement the counter
			w.cleanup()
			return nil, nil, err
		}

		// Wrap the connection to detect when it's closed
		return &webSocketConn{
			Conn:             conn,
			ActiveWebSockets: w.ActiveWebSockets,
			Cleaned:          false,
		}, rw, nil
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not support hijacking")
}

func (w *WebSocketTracker) cleanup() {
	if !w.Cleaned && w.ActiveWebSockets != nil {
		atomic.AddInt32(w.ActiveWebSockets, -1)
		w.Cleaned = true
		slog.Debug("WebSocket connection ended", "activeWebSockets", atomic.LoadInt32(w.ActiveWebSockets))
	}
}

// webSocketConn wraps net.Conn to track when WebSocket connection closes
type webSocketConn struct {
	net.Conn
	ActiveWebSockets *int32
	Cleaned          bool
}

func (c *webSocketConn) Close() error {
	if !c.Cleaned && c.ActiveWebSockets != nil {
		atomic.AddInt32(c.ActiveWebSockets, -1)
		c.Cleaned = true
		slog.Debug("WebSocket connection closed", "activeWebSockets", atomic.LoadInt32(c.ActiveWebSockets))
	}
	return c.Conn.Close()
}

// ProxyWithWebSocketSupport handles both HTTP and WebSocket proxying
func ProxyWithWebSocketSupport(w http.ResponseWriter, r *http.Request, targetURL string, activeWebSockets *int32) {
	target, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Invalid proxy target", http.StatusInternalServerError)
		return
	}

	// Check if this is a WebSocket request
	if IsWebSocketRequest(r) {
		// Track WebSocket connection
		if activeWebSockets != nil {
			atomic.AddInt32(activeWebSockets, 1)
			slog.Debug("WebSocket connection started", "activeWebSockets", atomic.LoadInt32(activeWebSockets))
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		// Wrap the response writer to detect when WebSocket closes
		if activeWebSockets != nil {
			w = &WebSocketTracker{
				ResponseWriter:   w,
				ActiveWebSockets: activeWebSockets,
				Cleaned:          false,
			}
		}

		proxy.ServeHTTP(w, r)
		return
	}

	// Regular HTTP proxy with retry
	HandleProxyWithRetry(w, r, targetURL, 3*time.Second)
}

// RetryResponseWriter buffers responses to enable retry on failure
type RetryResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	headers    http.Header
	written    bool
}

// NewRetryResponseWriter creates a new retry response writer
func NewRetryResponseWriter(w http.ResponseWriter) *RetryResponseWriter {
	return &RetryResponseWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		headers:        make(http.Header),
	}
}

// Header returns the header map
func (w *RetryResponseWriter) Header() http.Header {
	if w.written {
		return w.ResponseWriter.Header()
	}
	return w.headers
}

// WriteHeader captures the status code
func (w *RetryResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
	}
}

// Write captures the response body
func (w *RetryResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		return w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// Commit writes the buffered response to the underlying ResponseWriter
func (w *RetryResponseWriter) Commit() {
	if w.written {
		return
	}
	w.written = true

	// Copy headers
	for k, v := range w.headers {
		w.ResponseWriter.Header()[k] = v
	}

	// Write status code
	if w.statusCode != 0 {
		w.ResponseWriter.WriteHeader(w.statusCode)
	}

	// Write body
	if w.body.Len() > 0 {
		w.ResponseWriter.Write(w.body.Bytes())
	}
}

// Reset clears the buffer for retry
func (w *RetryResponseWriter) Reset() {
	w.statusCode = 0
	w.body.Reset()
	w.headers = make(http.Header)
}

// Hijack implements http.Hijacker interface
func (w *RetryResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not support hijacking")
}