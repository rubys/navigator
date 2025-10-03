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
		_, _ = w.Write(responseBodyBytes)

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
		portStr := strings.TrimPrefix(config.Server.Listen, ":")
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
