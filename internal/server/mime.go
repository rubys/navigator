package server

import (
	"mime"
	"net/http"
	"path/filepath"
)

// SetContentType sets the appropriate Content-Type header based on file extension
func SetContentType(w http.ResponseWriter, fsPath string) {
	ext := filepath.Ext(fsPath)

	// Try standard library first
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
		return
	}

	// Fallback for extensions not in standard library
	switch ext {
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	case ".ttf":
		w.Header().Set("Content-Type", "font/ttf")
	case ".eot":
		w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
	}
}