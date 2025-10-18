package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logging"
	"github.com/rubys/navigator/internal/utils"
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
		logging.LogStickySessionsDisabled()
		return false
	}

	// Get or set sticky session cookie
	targetMachine := ""
	cookie, err := r.Cookie(config.StickySession.CookieName)
	if err == nil {
		targetMachine = cookie.Value
		logging.LogStickySessionFound(targetMachine, currentMachineID)
	}

	// If no cookie or current machine, set cookie and continue
	if targetMachine == "" || targetMachine == currentMachineID {
		SetStickySessionCookie(w, currentMachineID, config)
		return false // Continue processing normally
	}

	// Check for retry header indicating target machine is down
	if r.Header.Get("X-Navigator-Retry") == "true" {
		logging.LogStickySessionUnavailable(targetMachine, currentMachineID)
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
		duration := utils.ParseDurationWithDefault(config.StickySession.CookieMaxAge, time.Hour)
		maxAge = int(duration.Seconds())
	}

	cookiePath := config.StickySession.CookiePath
	if cookiePath == "" {
		cookiePath = "/" // Default to root path
	}

	cookie := &http.Cookie{
		Name:     config.StickySession.CookieName,
		Value:    machineID,
		Path:     cookiePath,
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
	logging.LogStickySessionSet(machineID, maxAge)
}
