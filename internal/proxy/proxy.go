package proxy

import (
	"bufio"
	"bytes"
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
func HandleProxy(w http.ResponseWriter, r *http.Request, targetURL string) {
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

		// Path rewriting removed with legacy Location configuration
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

	// Use default transport - no custom connection timeout needed
	// The 500ms ProxyRetryMaxDelay is for retry backoff, not connection timeout
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Implement retry logic
	startTime := time.Now()
	attempt := 0
	initialDelay := config.ProxyRetryInitialDelay
	maxDelay := config.ProxyRetryMaxDelay

	for {
		attempt++

		// Reset buffer for retry if applicable
		if canRetry && attempt > 1 && retryWriter != nil {
			// If buffer limit was hit, disable further retries for this request
			if retryWriter.bufferLimitHit {
				slog.Debug("Disabling retry due to large response size",
					"target", targetURL,
					"attempt", attempt)
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
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
	success    bool
	statusCode int
}

// WriteHeader captures the status code and marks 502 as failure
// Note: Only propagates non-502 status codes to underlying writer
func (pr *proxyRecorder) WriteHeader(statusCode int) {
	pr.statusCode = statusCode
	if statusCode == http.StatusBadGateway {
		pr.success = false
		// Don't propagate 502 to underlying writer - we may retry
		return
	}
	// Propagate non-502 status codes (success or non-retryable errors)
	pr.ResponseWriter.WriteHeader(statusCode)
}

// Write blocks writes when a 502 error was detected
func (pr *proxyRecorder) Write(b []byte) (int, error) {
	if !pr.success {
		// Don't propagate response body for failed requests that may be retried
		return len(b), nil
	}
	return pr.ResponseWriter.Write(b)
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
	HandleProxyWithRetry(w, r, targetURL, config.ProxyRetryTimeout)
}

// RetryResponseWriter buffers responses to enable retry on failure
// Note: Only buffers responses up to MaxRetryBufferSize (64KB) to prevent memory issues
// Large responses automatically switch to streaming mode
type RetryResponseWriter struct {
	http.ResponseWriter
	statusCode     int
	body           *bytes.Buffer
	headers        http.Header
	written        bool
	bufferLimitHit bool
}

// MaxRetryBufferSize limits how much response data we buffer for retries
// Set to 64KB as most responses are smaller, and larger responses should stream immediately
const MaxRetryBufferSize = 64 * 1024 // 64KB

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

// Write captures the response body up to MaxRetryBufferSize
func (w *RetryResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		// Check if adding this data would exceed buffer limit
		if w.body.Len()+len(b) > MaxRetryBufferSize {
			// Calculate how much we can still buffer
			buffered := 0
			if !w.bufferLimitHit && w.body.Len() < MaxRetryBufferSize {
				remaining := MaxRetryBufferSize - w.body.Len()
				w.body.Write(b[:remaining])
				buffered = remaining
			}
			w.bufferLimitHit = true
			// Commit the buffer before switching to streaming mode
			w.Commit()
			// Now switch to direct writing for large responses
			w.written = true
			// Only write the portion that wasn't buffered
			if buffered < len(b) {
				n, err := w.ResponseWriter.Write(b[buffered:])
				if err != nil {
					return buffered + n, err
				}
				// All bytes processed successfully
				return len(b), nil
			}
			// All bytes were buffered
			return buffered, nil
		}
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
		_, _ = w.ResponseWriter.Write(w.body.Bytes())
	}
}

// Reset clears the buffer for retry
func (w *RetryResponseWriter) Reset() {
	w.statusCode = 0
	w.body.Reset()
	w.headers = make(http.Header)
	w.bufferLimitHit = false
}

// Hijack implements http.Hijacker interface
func (w *RetryResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not support hijacking")
}
