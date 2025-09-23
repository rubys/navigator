package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/proxy"
)

const (
	// MaxFlyReplaySize is the maximum request size for Fly-Replay (1MB)
	MaxFlyReplaySize = 1024 * 1024
)

// ShouldUseFlyReplay determines if a request should use fly-replay based on content length
// Fly replay can handle any method as long as the content length is less than 1MB
func ShouldUseFlyReplay(r *http.Request) bool {
	// If Content-Length is explicitly set and >= 1MB, use reverse proxy
	if r.ContentLength >= MaxFlyReplaySize {
		slog.Debug("Using reverse proxy due to large content length",
			"method", r.Method,
			"contentLength", r.ContentLength)
		return false
	}

	// If Content-Length is missing (-1) on methods that typically require content
	// (POST, PUT, PATCH), be conservative and use reverse proxy
	if r.ContentLength == -1 {
		methodsRequiringContent := []string{"POST", "PUT", "PATCH"}
		for _, method := range methodsRequiringContent {
			if r.Method == method {
				slog.Debug("Using reverse proxy due to missing content length on body method",
					"method", r.Method)
				return false
			}
		}
	}

	// For GET, HEAD, DELETE, OPTIONS and other methods without content, or
	// methods with content < 1MB, use fly-replay
	return true
}

// HandleFlyReplay handles Fly-Replay rewrite rules
func HandleFlyReplay(w http.ResponseWriter, r *http.Request, target string, status string, config *config.Config) bool {
	if ShouldUseFlyReplay(r) {
		// Check if this is a retry
		if r.Header.Get("X-Navigator-Retry") == "true" {
			// Retry detected, serve maintenance page
			slog.Info("Retry detected, serving maintenance page",
				"path", r.URL.Path,
				"target", target,
				"method", r.Method,
				"navigatorRetry", r.Header.Get("X-Navigator-Retry"))

			ServeMaintenancePage(w, r, config)
			return true
		}

		w.Header().Set("Content-Type", "application/vnd.fly.replay+json")
		statusCode := http.StatusTemporaryRedirect
		if code, err := strconv.Atoi(status); err == nil {
			statusCode = code
		}

		// Get current app name to determine if we're replaying to a different app
		currentAppName := os.Getenv("FLY_APP_NAME")

		// Parse target to determine if it's machine, app, or region
		var responseMap map[string]interface{}
		if strings.HasPrefix(target, "machine=") {
			// Machine-based fly-replay: machine=machine_id:app_name
			machineAndApp := strings.TrimPrefix(target, "machine=")
			parts := strings.Split(machineAndApp, ":")
			if len(parts) == 2 {
				machineID := parts[0]
				appName := parts[1]

				responseMap = map[string]interface{}{
					"app":             appName,
					"prefer_instance": machineID,
				}

				// Only add retry header if staying within the same app
				if currentAppName == appName {
					responseMap["transform"] = map[string]interface{}{
						"set_headers": []map[string]string{
							{"name": "X-Navigator-Retry", "value": "true"},
						},
					}
				}
			}
		} else if strings.HasPrefix(target, "app=") {
			// App-based fly-replay
			appName := strings.TrimPrefix(target, "app=")

			responseMap = map[string]interface{}{
				"app": appName,
			}

			// Only add retry header if staying within the same app
			if currentAppName == appName {
				responseMap["transform"] = map[string]interface{}{
					"set_headers": []map[string]string{
						{"name": "X-Navigator-Retry", "value": "true"},
					},
				}
			}
		} else {
			// Region-based fly-replay (same app, different region)
			responseMap = map[string]interface{}{
				"region": target + ",any",
				"transform": map[string]interface{}{
					"set_headers": []map[string]string{
						{"name": "X-Navigator-Retry", "value": "true"},
					},
				},
			}
		}

		w.WriteHeader(statusCode)

		responseBodyBytes, err := json.Marshal(responseMap)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return true
		}
		slog.Debug("Fly replay response body", "body", string(responseBodyBytes))
		w.Write(responseBodyBytes)

		// Set metadata for fly-replay response
		if recorder, ok := w.(*ResponseRecorder); ok {
			recorder.SetMetadata("response_type", "fly-replay")
			recorder.SetMetadata("destination", target)
		}

		return true
	} else {
		// Automatically reverse proxy instead of fly-replay
		return HandleFlyReplayFallback(w, r, target, config)
	}
}

// HandleFlyReplayFallback automatically reverse proxies the request when fly-replay isn't suitable
// Constructs the target URL based on target type:
// - Machine: http://<machine_id>.vm.<appname>.internal:<port><path>
// - App: http://<appname>.internal:<port><path>
// - Region: http://<region>.<FLY_APP_NAME>.internal:<port><path>
func HandleFlyReplayFallback(w http.ResponseWriter, r *http.Request, target string, config *config.Config) bool {
	flyAppName := os.Getenv("FLY_APP_NAME")
	if flyAppName == "" {
		slog.Debug("FLY_APP_NAME not set, cannot construct fallback proxy URL")
		return false
	}

	// Construct the target URL based on target type
	listenPort := 3000 // Default port
	if config.Server.Listen != "" {
		// Parse port from config.Server.Listen (could be ":3000" or "3000")
		portStr := config.Server.Listen
		if strings.HasPrefix(portStr, ":") {
			portStr = portStr[1:]
		}
		if port, err := strconv.Atoi(portStr); err == nil {
			listenPort = port
		}
	}

	var targetURL string
	if strings.HasPrefix(target, "machine=") {
		// Machine-based: http://<machine_id>.vm.<appname>.internal:<port><path>
		machineAndApp := strings.TrimPrefix(target, "machine=")
		parts := strings.Split(machineAndApp, ":")
		if len(parts) == 2 {
			machineID := parts[0]
			appName := parts[1]
			targetURL = fmt.Sprintf("http://%s.vm.%s.internal:%d%s", machineID, appName, listenPort, r.URL.Path)
		} else {
			slog.Debug("Invalid machine target format", "target", target)
			return false
		}
	} else if strings.HasPrefix(target, "app=") {
		// App-based: http://<appname>.internal:<port><path>
		appName := strings.TrimPrefix(target, "app=")
		targetURL = fmt.Sprintf("http://%s.internal:%d%s", appName, listenPort, r.URL.Path)
	} else {
		// Region-based: http://<region>.<FLY_APP_NAME>.internal:<port><path>
		targetURL = fmt.Sprintf("http://%s.%s.internal:%d%s", target, flyAppName, listenPort, r.URL.Path)
	}

	_, err := url.Parse(targetURL)
	if err != nil {
		slog.Error("Failed to parse fly-replay fallback URL",
			"url", targetURL,
			"error", err)
		return false
	}

	// Set forwarding headers
	r.Header.Set("X-Forwarded-Host", r.Host)

	slog.Info("Using automatic reverse proxy fallback for fly-replay",
		"originalPath", r.URL.Path,
		"targetURL", targetURL,
		"target", target,
		"method", r.Method,
		"contentLength", r.ContentLength)

	// Use the existing retry proxy logic
	proxy.HandleProxyWithRetry(w, r, targetURL, 3*time.Second)
	return true
}

// ServeMaintenancePage serves a maintenance/503 page
func ServeMaintenancePage(w http.ResponseWriter, r *http.Request, config *config.Config) {
	// Set metadata for maintenance page
	if recorder, ok := w.(*ResponseRecorder); ok {
		recorder.SetMetadata("response_type", "maintenance")
	}

	// Serve a simple maintenance page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)

	// Try to serve custom 503.html if available
	publicDir := "public" // Default fallback
	if config.Server.PublicDir != "" {
		publicDir = config.Server.PublicDir
	}

	// Check for custom 503.html
	maintenancePage := fmt.Sprintf("%s/503.html", publicDir)
	if _, err := os.Stat(maintenancePage); err == nil {
		http.ServeFile(w, r, maintenancePage)
		slog.Debug("Served custom maintenance page", "file", maintenancePage)
		return
	}

	// Serve fallback maintenance page
	fallbackHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Service Temporarily Unavailable</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            padding: 20px;
        }
        .container {
            text-align: center;
            max-width: 600px;
        }
        h1 {
            font-size: 3rem;
            margin-bottom: 0.5rem;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
        }
        p {
            font-size: 1.25rem;
            margin-bottom: 2rem;
            opacity: 0.9;
        }
        .status-code {
            font-size: 6rem;
            font-weight: bold;
            margin-bottom: 1rem;
            text-shadow: 3px 3px 6px rgba(0,0,0,0.3);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="status-code">503</div>
        <h1>Service Temporarily Unavailable</h1>
        <p>We're performing maintenance or experiencing high traffic. Please check back in a few moments.</p>
    </div>
</body>
</html>`

	w.Write([]byte(fallbackHTML))
	slog.Debug("Served fallback maintenance page")
}