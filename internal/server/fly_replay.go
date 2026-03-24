package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logging"
)

const (
	// MaxFlyReplaySize is the maximum request size for Fly-Replay (1MB)
	MaxFlyReplaySize = 1024 * 1024

	// DefaultFlyReplayTimeout is the default timeout for Fly-Replay attempts
	DefaultFlyReplayTimeout = "5s"

	// DefaultFlyReplayFallback is the default fallback strategy for Fly-Replay
	DefaultFlyReplayFallback = "force_self"
)

// ShouldUseFlyReplay determines if a request should use fly-replay based on content length
// Fly replay can handle any method as long as the content length is less than 1MB
func ShouldUseFlyReplay(r *http.Request) bool {
	// If Content-Length is explicitly set and >= 1MB, use reverse proxy
	if r.ContentLength >= MaxFlyReplaySize {
		logging.LogFlyReplayLargeContent(r.ContentLength, r.Method)
		return false
	}

	// If Content-Length is missing (-1) on methods that typically require content
	// (POST, PUT, PATCH), be conservative and use reverse proxy
	if r.ContentLength == -1 {
		methodsRequiringContent := []string{"POST", "PUT", "PATCH"}
		for _, method := range methodsRequiringContent {
			if r.Method == method {
				logging.LogFlyReplayMissingContentLength(r.Method)
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
	if !ShouldUseFlyReplay(r) {
		return false
	}

	// Check if this is a failed replay (Fly couldn't reach the target and fell back to us)
	if failedHeader := r.Header.Get("fly-replay-failed"); failedHeader != "" {
		logging.LogFlyReplayFailed(failedHeader, target)
		ServeMaintenancePage(w, r, config)
		return true
	}

	// Check if this is a retry (request already went through fly-replay once)
	if r.Header.Get("X-Navigator-Retry") == "true" {
		logging.LogFlyReplayRetryDetected(target)
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

	// Build transform headers that preserve Authorization
	// The Authorization header must be explicitly preserved during fly-replay
	// to prevent authentication failures on the target machine
	buildTransformHeaders := func() []map[string]string {
		headers := []map[string]string{
			{"name": "X-Navigator-Retry", "value": "true"},
		}

		// Explicitly preserve Authorization header if present
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			headers = append(headers, map[string]string{
				"name":  "Authorization",
				"value": authHeader,
			})
		}

		return headers
	}

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
				"instance": machineID,
				"timeout":         DefaultFlyReplayTimeout,
				"fallback":        DefaultFlyReplayFallback,
			}

			// Only add transform if staying within the same app
			if currentAppName == appName {
				responseMap["transform"] = map[string]interface{}{
					"set_headers": buildTransformHeaders(),
				}
			}
		}
	} else if strings.HasPrefix(target, "app=") {
		// App-based fly-replay
		appName := strings.TrimPrefix(target, "app=")

		responseMap = map[string]interface{}{
			"app":      appName,
			"timeout":  DefaultFlyReplayTimeout,
			"fallback": DefaultFlyReplayFallback,
		}

		// Only add transform if staying within the same app
		if currentAppName == appName {
			responseMap["transform"] = map[string]interface{}{
				"set_headers": buildTransformHeaders(),
			}
		}
	} else {
		// Region-based fly-replay (same app, different region)
		responseMap = map[string]interface{}{
			"region":   target,
			"timeout":  DefaultFlyReplayTimeout,
			"fallback": DefaultFlyReplayFallback,
			"transform": map[string]interface{}{
				"set_headers": buildTransformHeaders(),
			},
		}
	}

	w.WriteHeader(statusCode)

	responseBodyBytes, err := json.Marshal(responseMap)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return true
	}
	logging.LogFlyReplayResponseBody(responseBodyBytes)
	_, _ = w.Write(responseBodyBytes)

	// Set metadata for fly-replay response
	if recorder, ok := w.(*ResponseRecorder); ok {
		recorder.SetMetadata("response_type", "fly-replay")
		recorder.SetMetadata("destination", target)
	}

	return true
}
