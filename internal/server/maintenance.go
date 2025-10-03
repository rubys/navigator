package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/rubys/navigator/internal/config"
)

// ServeMaintenancePage serves a maintenance/503 page
func ServeMaintenancePage(w http.ResponseWriter, r *http.Request, config *config.Config) {
	// Set metadata for maintenance page
	if recorder, ok := w.(*ResponseRecorder); ok {
		recorder.SetMetadata("response_type", "maintenance")
	}

	// Try to serve custom 503.html if available
	publicDir := "public" // Default fallback
	if config.Server.PublicDir != "" {
		publicDir = config.Server.PublicDir
	}

	// Check for custom 503.html
	maintenancePage := fmt.Sprintf("%s/503.html", publicDir)
	if content, err := os.ReadFile(maintenancePage); err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.WriteHeader(503) // http.StatusServiceUnavailable
		_, _ = w.Write(content)
		slog.Debug("Served custom maintenance page", "file", maintenancePage)
		return
	}

	// Serve fallback maintenance page with 503 status
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(503) // http.StatusServiceUnavailable

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
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            padding: 3rem;
            border-radius: 20px;
            box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
        }
        h1 {
            font-size: 3rem;
            margin: 0 0 1rem 0;
            font-weight: 700;
        }
        p {
            font-size: 1.1rem;
            line-height: 1.6;
            margin: 1rem 0;
            opacity: 0.9;
        }
        .status-code {
            font-size: 1rem;
            opacity: 0.7;
            margin-top: 2rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Service Temporarily Unavailable</h1>
        <p>The service you are trying to reach is currently unavailable.</p>
        <p>This may be due to maintenance or a temporary deployment.</p>
        <p>Please try again in a few moments.</p>
        <div class="status-code">Error 503</div>
    </div>
</body>
</html>`

	_, _ = w.Write([]byte(fallbackHTML))
	slog.Debug("Served fallback maintenance page")
}
