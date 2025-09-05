# Navigator Logging Implementation Plan

## Overview

This document outlines an incremental approach to add structured logging capabilities to Navigator. The implementation is designed to be rolled out in phases, with each phase providing immediate value while maintaining backward compatibility.

## Implementation Status

- ✅ **Phase 1: Basic Structured Log Capture** - Completed in commit 906cad2
- ✅ **Phase 2: JSON Output Mode** - Completed in commit 2e2e73a  
- ✅ **Phase 3: Multiple Output Destinations** - Completed
- ✅ **Phase 4: Vector Integration** - Completed
- ⏳ **Phase 5: Future Enhancements** - Not implemented

## Current Capabilities (Phases 1-4)

Navigator now provides enterprise-grade logging with Vector integration for advanced log processing and routing:

### Text Format (Default)
```yaml
# No configuration needed
```
Output example:
```
[redis.stdout] Ready to accept connections
[2025/boston.stderr] Error: Connection refused
time=2025-09-05T00:03:07.492Z level=INFO msg="Serving static file" path=/assets/app.js
```

### JSON Format
```yaml
logging:
  format: json
```
Output example:
```json
{"@timestamp":"2025-09-05T00:03:07Z","source":"redis","stream":"stdout","message":"Ready to accept connections"}
{"@timestamp":"2025-09-05T00:03:07Z","source":"2025/boston","stream":"stderr","message":"Error: Connection refused","tenant":"boston"}
{"time":"2025-09-05T00:03:07.492Z","level":"INFO","msg":"Serving static file","path":"/assets/app.js"}
```

### File Output (Phase 3 - NEW)
```yaml
logging:
  format: json  # or text
  file: /var/log/navigator/{{app}}.log
```
- Logs written to both console AND file simultaneously
- Template support: `{{app}}` replaced with application/process name
- Automatic directory creation
- File opened in append mode for persistence across restarts

Output example (file):
```
# /var/log/navigator/redis.log
[redis.stdout] Ready to accept connections

# /var/log/navigator/2025-boston.log  
[2025/boston.stderr] Error: Connection refused
```

### Vector Integration (Phase 4 - NEW)
```yaml
logging:
  format: json
  file: /var/log/navigator/{{app}}.log    # Optional: direct file output
  vector:
    enabled: true
    socket: /tmp/navigator-vector.sock     # Unix socket for Vector
    config: /etc/vector/vector.toml        # Path to Vector config
```
- **Professional Log Aggregation**: Vector automatically started as managed process
- **Multiple Sinks**: Send logs to Elasticsearch, S3, NATS, Kafka, etc.
- **Advanced Processing**: Transform, filter, and enrich logs with Vector's VRL
- **Graceful Degradation**: Continues working if Vector fails or isn't installed
- **Unix Socket Communication**: High-performance local logging transport
- **Automatic Integration**: No manual Vector process management required

### Key Features
- **Source Identification**: All child process output tagged with application/process name
- **Stream Separation**: Stdout and stderr clearly identified
- **Tenant Context**: Multi-tenant applications include tenant information in JSON logs
- **Consistent Format**: Navigator's own logs and child process logs use same format when JSON is enabled
- **Multiple Destinations**: Write to console, files, and Vector simultaneously (Phases 3-4)
- **Template Support**: Dynamic file paths using `{{app}}` variable (Phase 3)
- **Enterprise Logging**: Vector integration for professional log processing (Phase 4)
- **Zero Application Changes**: Works with any framework (Rails, Django, Node.js, etc.)

## Phase 1: Basic Structured Log Capture ✅ COMPLETED

**Goal:** Capture and tag output from child processes with metadata.

**Status:** Completed in commit 906cad2

**Implementation:**

```go
// Add to main.go
type LogWriter struct {
    source string  // app name or process name
    stream string  // "stdout" or "stderr"
    output io.Writer
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
    // For now, just prefix with metadata
    prefix := fmt.Sprintf("[%s.%s] ", w.source, w.stream)
    lines := bytes.Split(p, []byte("\n"))
    for _, line := range lines {
        if len(line) > 0 {
            w.output.Write([]byte(prefix))
            w.output.Write(line)
            w.output.Write([]byte("\n"))
        }
    }
    return len(p), nil
}

// Update StartApp() and startManagedProcess()
cmd.Stdout = &LogWriter{source: appName, stream: "stdout", output: os.Stdout}
cmd.Stderr = &LogWriter{source: appName, stream: "stderr", output: os.Stderr}
```

**Output Example:**
```
[2025-boston.stdout] Started GET "/" for 127.0.0.1
[2025-boston.stderr] Error: Connection refused
[redis.stdout] Ready to accept connections
```

**Benefits:**
- Minimal code changes
- Immediately identifies log source
- Maintains backward compatibility
- No configuration required

## Phase 2: JSON Output Mode ✅ COMPLETED

**Goal:** Add structured JSON output as a configuration option.

**Status:** Completed in commit 2e2e73a

**Key Changes Made:**
- Added `LogConfig` struct with `format` field
- Created `JSONLogWriter` and `LogEntry` for structured output
- Updated YAML parser to support `logging:` configuration section
- Navigator's slog switches to JSON format after config load
- Both managed processes and web apps use appropriate format based on config
- Added tenant extraction from environment variables

**Key Learning:** Logging format is set at startup for Navigator's own logs. Changing the format requires a restart, not just config reload, because the HTTP server and other components continue using their initial logger format.

**Original Implementation Plan:**

```go
type LogEntry struct {
    Timestamp string `json:"@timestamp"`
    Source    string `json:"source"`
    Stream    string `json:"stream"`
    Message   string `json:"message"`
    Tenant    string `json:"tenant,omitempty"`
}

type JSONLogWriter struct {
    source string
    stream string
    tenant string
    output io.Writer
}

func (w *JSONLogWriter) Write(p []byte) (n int, err error) {
    lines := bytes.Split(p, []byte("\n"))
    for _, line := range lines {
        if len(line) == 0 {
            continue
        }
        entry := LogEntry{
            Timestamp: time.Now().Format(time.RFC3339),
            Source:    w.source,
            Stream:    w.stream,
            Message:   string(line),
            Tenant:    w.tenant,
        }
        data, _ := json.Marshal(entry)
        w.output.Write(data)
        w.output.Write([]byte("\n"))
    }
    return len(p), nil
}

// Configuration
type LogConfig struct {
    Format string `yaml:"format"` // "text" or "json"
}

// In StartApp()
var stdout, stderr io.Writer
if config.Logging.Format == "json" {
    stdout = &JSONLogWriter{source: appName, stream: "stdout", tenant: tenant, output: os.Stdout}
    stderr = &JSONLogWriter{source: appName, stream: "stderr", tenant: tenant, output: os.Stderr}
} else {
    stdout = &LogWriter{source: appName, stream: "stdout", output: os.Stdout}
    stderr = &LogWriter{source: appName, stream: "stderr", output: os.Stderr}
}
cmd.Stdout = stdout
cmd.Stderr = stderr
```

**Configuration:**
```yaml
logging:
  format: json  # or "text" (default)
```

**Benefits:**
- Structured logs for parsing
- Easy integration with log aggregation tools
- Backward compatible (text mode is default)
- Tenant identification in logs

## Phase 3: Multiple Output Destinations ✅ COMPLETED

**Goal:** Enable writing logs to multiple destinations simultaneously.

**Status:** Completed - logs can now be written to both console and files simultaneously

**Implementation:**

```go
type MultiLogWriter struct {
    outputs []io.Writer
}

func (m *MultiLogWriter) Write(p []byte) (n int, err error) {
    for _, output := range m.outputs {
        output.Write(p)
    }
    return len(p), nil
}

// Add file output
func createFileWriter(path string) (io.Writer, error) {
    // Create with rotation support later
    return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

// In StartApp()
outputs := []io.Writer{os.Stdout}

// Add file output if configured
if config.Logging.File != "" {
    logFile := strings.ReplaceAll(config.Logging.File, "{{app}}", appName)
    if file, err := createFileWriter(logFile); err == nil {
        outputs = append(outputs, file)
        defer file.Close()
    }
}

// Create the appropriate writer
var writer io.Writer
if config.Logging.Format == "json" {
    writer = &JSONLogWriter{
        source: appName,
        stream: "stdout", 
        output: &MultiLogWriter{outputs: outputs},
    }
} else {
    writer = &LogWriter{
        source: appName,
        stream: "stdout",
        output: &MultiLogWriter{outputs: outputs},
    }
}
cmd.Stdout = writer
```

**Configuration:**
```yaml
logging:
  format: json
  file: /var/log/navigator/{{app}}.log  # Optional file output
```

**Benefits:**
- Persistent log storage
- Multiple destinations (console + file)
- Template support for dynamic paths
- Debugging remains easy with console output

## Phase 4: Vector Integration ✅ COMPLETED

**Goal:** Add Vector as a managed process and output destination for advanced log routing.

**Status:** Completed - Navigator can now send logs to Vector for enterprise-grade log processing

**Key Features Implemented:**
- Vector automatically started as a managed process when enabled
- Unix socket communication to Vector for high performance
- VectorWriter with graceful degradation if Vector fails
- Logs sent to console, files, AND Vector simultaneously
- Complete integration with existing logging pipeline
- Zero manual Vector process management required

**Implementation:**

```go
// Unix socket writer for Vector
type VectorWriter struct {
    socket string
    conn   net.Conn
    mutex  sync.Mutex
}

func NewVectorWriter(socket string) *VectorWriter {
    return &VectorWriter{socket: socket}
}

func (v *VectorWriter) Write(p []byte) (n int, err error) {
    v.mutex.Lock()
    defer v.mutex.Unlock()
    
    // Lazy connection
    if v.conn == nil {
        v.conn, err = net.Dial("unix", v.socket)
        if err != nil {
            return 0, nil // Silently fail if Vector isn't running
        }
    }
    
    // Try to write
    n, err = v.conn.Write(p)
    if err != nil {
        v.conn.Close()
        v.conn = nil
    }
    return n, nil
}

// Configuration
type LogConfig struct {
    Format string `yaml:"format"`
    File   string `yaml:"file"`
    Vector struct {
        Enabled bool   `yaml:"enabled"`
        Socket  string `yaml:"socket"`
        Config  string `yaml:"config"` // Path to vector.toml
    } `yaml:"vector"`
}

// Start Vector if configured
if config.Logging.Vector.Enabled {
    addManagedProcess(ManagedProcess{
        Name:    "vector",
        Command: "vector",
        Args:    []string{"--config", config.Logging.Vector.Config},
        AutoRestart: true,
    })
    
    // Add Vector output
    outputs = append(outputs, NewVectorWriter(config.Logging.Vector.Socket))
}
```

**Configuration:**
```yaml
logging:
  format: json
  file: /var/log/navigator/{{app}}.log
  vector:
    enabled: true
    socket: /tmp/vector.sock
    config: /etc/vector/vector.toml

managed_processes:
  - name: vector
    command: vector
    args: ["--config", "/etc/vector/vector.toml"]
    auto_restart: true
    priority: -10  # Start before other processes
```

**Sample vector.toml:**
```toml
[sources.navigator]
type = "socket"
mode = "unix"
path = "/tmp/vector.sock"

[transforms.parse]
type = "remap"
inputs = ["navigator"]
source = '''
  . = parse_json!(.message)
  .region = get_env_var!("FLY_REGION")
'''

[sinks.console]
type = "console"
inputs = ["parse"]
encoding.codec = "json"

[sinks.nats]
type = "nats"
inputs = ["parse"]
url = "nats://localhost:4222"
subject = "logs.{{ source }}"
```

**Benefits:**
- Professional log aggregation
- Multiple sink options (NATS, S3, Elasticsearch, etc.)
- Buffering and retry logic
- Log transformation capabilities
- Graceful degradation if Vector unavailable

## Phase 5: Future Enhancements

**Potential additions for later phases:**

1. **Log Buffering:**
   - In-memory buffering for high-volume applications
   - Batch writes to reduce I/O

2. **File Rotation:**
   - Size-based rotation
   - Time-based rotation
   - Compression of old files

3. **Direct Remote Outputs:**
   - HTTP/HTTPS endpoints
   - NATS direct integration
   - Syslog protocol

4. **Metrics Extraction:**
   - Parse response times
   - Count errors
   - Export to Prometheus

5. **Smart Sampling:**
   - Sample logs during high volume
   - Always capture errors
   - Configurable sampling rates

## Migration Path

### From Current State to Phase 1
- No configuration changes required
- Deploy new Navigator binary
- Logs immediately show source tagging

### From Phase 1 to Phase 2
- Add `logging: format: json` to configuration
- Restart Navigator
- Logs now in JSON format

### From Phase 2 to Phase 3
- Add `logging: file:` path to configuration
- Ensure log directory exists with write permissions
- Restart Navigator
- Logs now written to both console and file

### From Phase 3 to Phase 4
- Install Vector binary
- Create vector.toml configuration
- Add Vector configuration to navigator.yml
- Restart Navigator
- Vector starts automatically and receives logs

## Testing Strategy

### Phase 1 Testing
```bash
# Start Navigator and verify prefixed output
./bin/navigator config/navigator.yml
# Check output shows [app.stdout] prefixes
```

### Phase 2 Testing
```bash
# Enable JSON format
echo "logging:\n  format: json" >> config/navigator.yml
./bin/navigator config/navigator.yml | jq .
# Verify valid JSON with source, stream, message fields
```

### Phase 3 Testing
```bash
# Add file output
echo "logging:\n  file: /tmp/navigator-{{app}}.log" >> config/navigator.yml
./bin/navigator config/navigator.yml
# Verify logs appear in both console and files
tail -f /tmp/navigator-*.log
```

### Phase 4 Testing
```bash
# Test Vector integration
vector validate /etc/vector/vector.toml
./bin/navigator config/navigator.yml
# Check Vector is running
ps aux | grep vector
# Verify Vector receives logs
vector top /etc/vector/vector.toml
```

## Rollback Procedures

Each phase can be rolled back independently:

- **Phase 1 → Original:** Deploy previous Navigator binary
- **Phase 2 → Phase 1:** Remove or comment out `logging:` configuration
- **Phase 3 → Phase 2:** Remove `file:` from logging configuration
- **Phase 4 → Phase 3:** Set `vector: enabled: false` or remove Vector configuration

## Success Metrics

### Phase 1
- All logs properly tagged with source
- No performance degradation
- Zero log loss

### Phase 2
- Valid JSON output
- Parseable by jq and log tools
- Tenant properly identified

### Phase 3
- Logs written to files successfully
- No file descriptor leaks
- Proper file permissions

### Phase 4
- Vector process starts and stays running
- Logs flow to Vector destinations
- Graceful handling when Vector unavailable

## Resource Considerations

### Memory Impact
- Phase 1: Negligible (< 1MB)
- Phase 2: Small (JSON encoding overhead)
- Phase 3: Moderate (file buffers)
- Phase 4: Vector uses 50-200MB depending on configuration

### CPU Impact
- Phase 1-3: Minimal (< 1% overhead)
- Phase 4: Vector may use 5-10% CPU depending on volume

### Disk Usage
- Phase 3+: Depends on retention policy
- Recommend log rotation at 100MB per file
- Keep 10 rotated files (1GB per app maximum)

## Configuration Examples

### Minimal Configuration (Phase 1)
```yaml
# No configuration needed - works out of the box
```

### Development Configuration (Phase 2)
```yaml
logging:
  format: text  # Human-readable for development
```

### Production Configuration (Phase 4)
```yaml
logging:
  format: json
  file: /var/log/navigator/{{app}}.log
  vector:
    enabled: true
    socket: /tmp/vector.sock
    config: /etc/vector/vector.toml
```

### High-Volume Configuration
```yaml
logging:
  format: json
  # Skip file output to reduce I/O
  vector:
    enabled: true
    socket: /tmp/vector.sock
    config: /etc/vector/vector-buffered.toml
```

## Conclusion

This incremental approach allows Navigator to evolve its logging capabilities without disrupting existing deployments. Each phase provides immediate value while maintaining simplicity and reliability. The implementation can stop at any phase based on actual needs, with Phase 4 (Vector integration) providing enterprise-grade log management when required.