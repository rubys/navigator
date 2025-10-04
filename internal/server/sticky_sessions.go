package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rubys/navigator/internal/config"
)

// HandleStickySession handles sticky session routing using Fly-Replay
func HandleStickySession(w http.ResponseWriter, r *http.Request, config *config.Config) bool {
	if !config.StickySession.Enabled {
		return false
	}

	// Check if path requires sticky sessions
	if len(config.StickySession.Paths) > 0 {
		matched := false
		for _, pattern := range config.StickySession.Paths {
			if matched, _ = filepath.Match(pattern, r.URL.Path); matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	currentMachineID := os.Getenv("FLY_MACHINE_ID")
	appName := os.Getenv("FLY_APP_NAME")

	if currentMachineID == "" || appName == "" {
		slog.Debug("Sticky sessions require FLY_MACHINE_ID and FLY_APP_NAME")
		return false
	}

	// Get or set sticky session cookie
	targetMachine := ""
	cookie, err := r.Cookie(config.StickySession.CookieName)
	if err == nil {
		targetMachine = cookie.Value
		slog.Debug("Found sticky session cookie", "machine", targetMachine, "currentMachine", currentMachineID)
	}

	// If no cookie or current machine, set cookie and continue
	if targetMachine == "" || targetMachine == currentMachineID {
		SetStickySessionCookie(w, currentMachineID, config)
		return false // Continue processing normally
	}

	// Check for retry header indicating target machine is down
	if r.Header.Get("X-Navigator-Retry") == "true" {
		slog.Info("Sticky session target machine unavailable, serving from current",
			"targetMachine", targetMachine,
			"currentMachine", currentMachineID)
		SetStickySessionCookie(w, currentMachineID, config)
		ServeMaintenancePage(w, r, config)
		return true
	}

	// Use Fly-Replay for cross-region routing
	if ShouldUseFlyReplay(r) {
		// Construct the Fly-Replay target
		target := fmt.Sprintf("machine=%s:%s", targetMachine, appName)
		return HandleFlyReplay(w, r, target, "307", config)
	} else {
		// Large request, use reverse proxy fallback
		target := fmt.Sprintf("machine=%s:%s", targetMachine, appName)
		return HandleFlyReplayFallback(w, r, target, config)
	}
}

// SetStickySessionCookie sets a sticky session cookie for the given machine ID
func SetStickySessionCookie(w http.ResponseWriter, machineID string, config *config.Config) {
	maxAge := 3600 // Default 1 hour
	if config.StickySession.CookieMaxAge != "" {
		if duration, err := time.ParseDuration(config.StickySession.CookieMaxAge); err == nil {
			maxAge = int(duration.Seconds())
		}
	}

	cookie := &http.Cookie{
		Name:     config.StickySession.CookieName,
		Value:    machineID,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: config.StickySession.CookieHTTPOnly,
		Secure:   config.StickySession.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}

	if config.StickySession.CookieSameSite != "" {
		switch config.StickySession.CookieSameSite {
		case "strict":
			cookie.SameSite = http.SameSiteStrictMode
		case "none":
			cookie.SameSite = http.SameSiteNoneMode
		case "lax":
			cookie.SameSite = http.SameSiteLaxMode
		}
	}

	http.SetCookie(w, cookie)
	slog.Debug("Set sticky session cookie", "machine", machineID, "maxAge", maxAge)
}
