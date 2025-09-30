package server

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/rubys/navigator/internal/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now (configure as needed)
		return true
	},
}

// handleReverseProxies checks and handles reverse proxy routes
func (h *Handler) handleReverseProxies(w http.ResponseWriter, r *http.Request) bool {
	if h.config.Routes.ReverseProxies == nil {
		return false
	}

	for _, proxy := range h.config.Routes.ReverseProxies {
		matched := false

		// Check path pattern match
		if proxy.Path != "" {
			if pattern, err := regexp.Compile(proxy.Path); err == nil {
				if pattern.MatchString(r.URL.Path) {
					matched = true
				}
			}
		} else if proxy.Prefix != "" {
			// Simple prefix matching
			if strings.HasPrefix(r.URL.Path, proxy.Prefix) {
				matched = true
			}
		}

		if !matched {
			continue
		}

		slog.Debug("Matched reverse proxy route",
			"path", r.URL.Path,
			"target", proxy.Target,
			"websocket", proxy.WebSocket)

		// Handle the proxy
		if proxy.WebSocket && isWebSocketRequest(r) {
			h.handleWebSocketProxy(w, r, &proxy)
		} else {
			h.handleHTTPProxy(w, r, &proxy)
		}
		return true
	}

	return false
}

// handleHTTPProxy handles regular HTTP reverse proxy
func (h *Handler) handleHTTPProxy(w http.ResponseWriter, r *http.Request, route *config.ProxyRoute) {
	targetURL, err := url.Parse(route.Target)
	if err != nil {
		slog.Error("Invalid proxy target URL", "target", route.Target, "error", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the director to modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Apply custom headers
		for key, value := range route.Headers {
			// Replace variables
			headerValue := strings.ReplaceAll(value, "$remote_addr", getClientIP(req))
			headerValue = strings.ReplaceAll(headerValue, "$scheme", getScheme(req))
			headerValue = strings.ReplaceAll(headerValue, "$host", req.Host)
			req.Header.Set(key, headerValue)
		}

		// Strip path if configured
		if route.StripPath {
			if route.Prefix != "" {
				// Simple prefix stripping
				req.URL.Path = strings.TrimPrefix(req.URL.Path, route.Prefix)
				if !strings.HasPrefix(req.URL.Path, "/") {
					req.URL.Path = "/" + req.URL.Path
				}
			} else if route.Path != "" {
				// Regex-based path stripping using capture groups
				if pattern, err := regexp.Compile(route.Path); err == nil {
					matches := pattern.FindStringSubmatch(req.URL.Path)
					if len(matches) > 1 {
						// Use first capture group as the new path
						req.URL.Path = "/" + matches[1]
					}
				}
			}
		}

		slog.Debug("Proxying HTTP request",
			"method", req.Method,
			"original_path", r.URL.Path,
			"target_url", req.URL.String())
	}

	// Handle errors
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("Proxy error", "error", err, "target", route.Target)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

// handleWebSocketProxy handles WebSocket reverse proxy
func (h *Handler) handleWebSocketProxy(w http.ResponseWriter, r *http.Request, route *config.ProxyRoute) {
	targetURL, err := url.Parse(route.Target)
	if err != nil {
		slog.Error("Invalid WebSocket target URL", "target", route.Target, "error", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Adjust scheme for WebSocket
	switch targetURL.Scheme {
	case "http":
		targetURL.Scheme = "ws"
	case "https":
		targetURL.Scheme = "wss"
	}

	// Build target WebSocket URL
	var targetPath string
	if route.StripPath && route.Prefix != "" {
		targetPath = strings.TrimPrefix(r.URL.Path, route.Prefix)
	} else {
		targetPath = r.URL.Path
	}

	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}

	targetURL.Path = targetPath
	targetURL.RawQuery = r.URL.RawQuery

	slog.Debug("Proxying WebSocket connection",
		"original_path", r.URL.Path,
		"target_url", targetURL.String())

	// Connect to backend WebSocket server
	backendHeader := http.Header{}
	for key, values := range r.Header {
		// Skip hop-by-hop headers and WebSocket-specific headers
		if isHopByHopHeader(key) || isWebSocketHeader(key) {
			continue
		}
		for _, value := range values {
			backendHeader.Add(key, value)
		}
	}

	// Apply custom headers
	for key, value := range route.Headers {
		headerValue := strings.ReplaceAll(value, "$remote_addr", getClientIP(r))
		headerValue = strings.ReplaceAll(headerValue, "$scheme", getScheme(r))
		headerValue = strings.ReplaceAll(headerValue, "$host", r.Host)
		backendHeader.Set(key, headerValue)
	}

	backendConn, backendResp, err := websocket.DefaultDialer.Dial(targetURL.String(), backendHeader)
	if err != nil {
		slog.Error("Failed to connect to backend WebSocket",
			"target", targetURL.String(),
			"error", err)
		if backendResp != nil {
			slog.Debug("Backend response", "status", backendResp.StatusCode)
		}
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	// Upgrade client connection
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade client connection", "error", err)
		return
	}
	defer clientConn.Close()

	slog.Info("WebSocket proxy established",
		"client", getClientIP(r),
		"target", targetURL.String())

	// Proxy messages between client and backend
	errc := make(chan error, 2)

	// Copy from backend to client
	go func() {
		for {
			messageType, message, err := backendConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := clientConn.WriteMessage(messageType, message); err != nil {
				errc <- err
				return
			}
		}
	}()

	// Copy from client to backend
	go func() {
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := backendConn.WriteMessage(messageType, message); err != nil {
				errc <- err
				return
			}
		}
	}()

	// Wait for error or connection close
	err = <-errc
	if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		slog.Debug("WebSocket proxy ended with error", "error", err)
	} else {
		slog.Debug("WebSocket proxy closed normally")
	}
}

// isWebSocketRequest checks if the request is a WebSocket upgrade request
func isWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// isHopByHopHeader checks if a header is hop-by-hop
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	header = strings.ToLower(header)
	for _, h := range hopByHopHeaders {
		if strings.ToLower(h) == header {
			return true
		}
	}
	return false
}

// isWebSocketHeader checks if a header is WebSocket-specific
func isWebSocketHeader(header string) bool {
	wsHeaders := []string{
		"Sec-WebSocket-Key",
		"Sec-WebSocket-Version",
		"Sec-WebSocket-Extensions",
		"Sec-WebSocket-Accept",
		"Sec-WebSocket-Protocol",
	}

	header = strings.ToLower(header)
	for _, h := range wsHeaders {
		if strings.ToLower(h) == header {
			return true
		}
	}
	return false
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

// getScheme determines the request scheme
func getScheme(r *http.Request) string {
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
