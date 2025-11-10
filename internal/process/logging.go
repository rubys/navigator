package process

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rubys/navigator/internal/config"
)

// LogWriter wraps output streams to add source identification
type LogWriter struct {
	source string // app name or process name
	stream string // "stdout" or "stderr"
	output io.Writer
}

// Write implements io.Writer interface, prefixing each line with source metadata
func (w *LogWriter) Write(p []byte) (n int, err error) {
	// Split input into lines
	lines := bytes.Split(p, []byte("\n"))
	for i, line := range lines {
		// Skip empty lines at the end
		if len(line) == 0 && i == len(lines)-1 {
			continue
		}
		// Write prefixed line
		prefix := fmt.Sprintf("[%s.%s] ", w.source, w.stream)
		_, _ = w.output.Write([]byte(prefix))
		_, _ = w.output.Write(line)
		_, _ = w.output.Write([]byte("\n"))
	}
	return len(p), nil
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string `json:"@timestamp"`
	Source    string `json:"source"`
	Stream    string `json:"stream"`
	Message   string `json:"message"`
	Tenant    string `json:"tenant,omitempty"`
}

// JSONLogWriter writes structured JSON log entries
type JSONLogWriter struct {
	source string
	stream string
	tenant string
	output io.Writer
}

// Write implements io.Writer interface, outputting JSON log entries
func (w *JSONLogWriter) Write(p []byte) (n int, err error) {
	lines := bytes.Split(p, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Check if the line is already valid JSON from the Rails app
		if json.Valid(line) {
			// Check if it contains Rails JSON log markers
			if bytes.Contains(line, []byte(`"@timestamp"`)) && bytes.Contains(line, []byte(`"severity"`)) {
				// This looks like Rails JSON output - pass it through directly
				_, _ = w.output.Write(line)
				_, _ = w.output.Write([]byte("\n"))
				continue
			}
		}

		// Not JSON or not Rails JSON format - wrap it in our JSON structure
		entry := LogEntry{
			Timestamp: time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
			Source:    w.source,
			Stream:    w.stream,
			Message:   string(line),
			Tenant:    w.tenant,
		}
		data, _ := json.Marshal(entry)
		_, _ = w.output.Write(data)
		_, _ = w.output.Write([]byte("\n"))
	}
	return len(p), nil
}

// MultiLogWriter writes to multiple outputs simultaneously
type MultiLogWriter struct {
	outputs []io.Writer
}

// Write implements io.Writer interface, writing to all configured outputs
func (m *MultiLogWriter) Write(p []byte) (n int, err error) {
	for _, output := range m.outputs {
		_, _ = output.Write(p)
	}
	return len(p), nil
}

// VectorWriter writes logs to Vector via Unix socket
type VectorWriter struct {
	socket string
	conn   net.Conn
	mutex  sync.Mutex
}

// NewVectorWriter creates a new Vector writer
func NewVectorWriter(socket string) *VectorWriter {
	return &VectorWriter{socket: socket}
}

// Write implements io.Writer interface for Vector output
func (v *VectorWriter) Write(p []byte) (n int, err error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Lazy connection - connect on first write
	if v.conn == nil {
		v.conn, err = net.Dial("unix", v.socket)
		if err != nil {
			// Silently fail if Vector isn't running - graceful degradation
			return len(p), nil
		}
	}

	// Try to write to Vector
	n, err = v.conn.Write(p)
	if err != nil {
		// Connection failed, close and reset
		v.conn.Close()
		v.conn = nil
		// Return success to avoid breaking the log pipeline
		return len(p), nil
	}

	return n, nil
}

// Close closes the Vector connection
func (v *VectorWriter) Close() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if v.conn != nil {
		err := v.conn.Close()
		v.conn = nil
		return err
	}
	return nil
}

// CreateAccessLogWriter creates a writer for Navigator's HTTP access logs
// This sends logs to stdout and optionally to Vector for aggregation
func CreateAccessLogWriter(logConfig config.LogConfig, stdout io.Writer) io.Writer {
	outputs := []io.Writer{stdout}

	// Add Vector output if configured
	if logConfig.Vector.Enabled && logConfig.Vector.Socket != "" {
		vectorWriter := NewVectorWriter(logConfig.Vector.Socket)
		outputs = append(outputs, vectorWriter)
		slog.Debug("Access logs will be sent to Vector", "socket", logConfig.Vector.Socket)
	}

	// Return appropriate writer
	if len(outputs) == 1 {
		return outputs[0]
	}
	return &MultiLogWriter{outputs: outputs}
}

// createFileWriter creates a file writer with the specified path
func createFileWriter(path string, appName string) (io.Writer, error) {
	// Replace {{app}} template with actual app name
	logPath := strings.ReplaceAll(path, "{{app}}", appName)

	// Create directory if it doesn't exist
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
	}

	// Open file for append (create if doesn't exist)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logPath, err)
	}

	return file, nil
}

// CreateLogWriter creates appropriate log writer based on configuration
func CreateLogWriter(source, stream string, logConfig config.LogConfig) io.Writer {
	var outputs []io.Writer

	// Always include console output
	if logConfig.Format == "json" {
		outputs = append(outputs, &JSONLogWriter{
			source: source,
			stream: stream,
			output: os.Stdout,
		})
	} else {
		outputs = append(outputs, &LogWriter{
			source: source,
			stream: stream,
			output: os.Stdout,
		})
	}

	// Add file output if configured
	if logConfig.File != "" {
		if fileWriter, err := createFileWriter(logConfig.File, source); err == nil {
			if logConfig.Format == "json" {
				outputs = append(outputs, &JSONLogWriter{
					source: source,
					stream: stream,
					output: fileWriter,
				})
			} else {
				outputs = append(outputs, &LogWriter{
					source: source,
					stream: stream,
					output: fileWriter,
				})
			}
		}
	}

	// Add Vector output if configured
	if logConfig.Vector.Enabled && logConfig.Vector.Socket != "" {
		vectorWriter := NewVectorWriter(logConfig.Vector.Socket)
		outputs = append(outputs, &JSONLogWriter{
			source: source,
			stream: stream,
			output: vectorWriter,
		})
	}

	// Return appropriate writer
	if len(outputs) == 1 {
		return outputs[0]
	}
	return &MultiLogWriter{outputs: outputs}
}
