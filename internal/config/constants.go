package config

import "time"

// Additional constants for common configurations
const (
	// Default directories
	DefaultPublicDir = "public"
	DefaultTmpDir    = "tmp"
	DefaultLogDir    = "log"

	// Stream types
	StreamStdout = "stdout"
	StreamStderr = "stderr"

	// Common HTTP headers
	HeaderRequestID    = "X-Request-Id"
	HeaderFlyReplay    = "fly-replay"
	HeaderFlyMachineID = "fly-machine-id"

	// Auth realms
	DefaultAuthRealm = "Restricted"

	// Process states
	ProcessStateStarting = "starting"
	ProcessStateRunning  = "running"
	ProcessStateStopping = "stopping"
	ProcessStateStopped  = "stopped"

	// Hook timeout defaults
	DefaultHookTimeout = 30 * time.Second

	// Buffer sizes
	DefaultBufferSize    = 4096
	MaxRetryBufferSize   = 1024 * 1024 // 1MB
	DefaultLogBufferSize = 8192
)

// Static file extensions that should be served directly
var StaticFileExtensions = []string{
	"js", "css", "png", "jpg", "jpeg", "gif", "svg",
	"ico", "pdf", "txt", "xml", "json", "woff", "woff2",
	"ttf", "eot", "webp", "mp4", "webm", "ogg", "mp3",
	"wav", "flac", "aac", "wasm", "map",
}

// Common MIME types
var MIMETypes = map[string]string{
	".html": "text/html; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".js":   "application/javascript",
	".json": "application/json",
	".xml":  "application/xml",
	".pdf":  "application/pdf",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".ico":  "image/x-icon",
	".woff": "font/woff",
	".woff2": "font/woff2",
	".ttf":  "font/ttf",
	".eot":  "application/vnd.ms-fontobject",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".ogg":  "audio/ogg",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
	".aac":  "audio/aac",
	".wasm": "application/wasm",
	".map":  "application/json",
}