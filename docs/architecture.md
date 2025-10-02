# Architecture

Navigator v0.11.0 represents a significant architectural evolution from a single-file implementation to a **modular, well-tested Go application** with clear separation of concerns.

## Design Principles

Navigator's architecture is built on these core principles:

1. **Focused Modules** - Each package has a single, well-defined responsibility
2. **High Test Coverage** - 81.2% test coverage across all internal packages
3. **Cross-Platform** - Native support for Unix, Linux, macOS, and Windows
4. **Single Binary** - Despite modular structure, still deploys as one executable
5. **Clear Dependencies** - Minimal external dependencies, well-organized internal packages

## Package Structure

### HTTP Layer

#### `internal/server/`
**Purpose**: HTTP request handling, routing, and response generation

**Key Components**:

- `handler.go` (343 LOC) - Main HTTP handler and routing logic
- `static.go` (263 LOC) - Static file serving with try_files behavior
- `mime.go` (30 LOC) - MIME type detection
- `access_log.go` (119 LOC) - Structured access logging

**Responsibilities**:

- Request routing and dispatching
- Static file serving with caching
- WebSocket connection upgrades
- Access logging with metadata

**Test Coverage**: 84.5%

---

### Process Management

#### `internal/process/`
**Purpose**: Web application and managed process lifecycle

**Key Components**:

- `webapp.go` (351 LOC) - Web app instance management and idle monitoring
- `process_starter.go` (187 LOC) - Framework detection and process startup
- `port_allocator.go` (34 LOC) - Dynamic TCP port allocation
- `manager.go` - External process management (Redis, Sidekiq, workers)
- `hooks.go` - Lifecycle hook execution

**Responsibilities**:

- On-demand web app startup
- Framework detection (Rails, Django, Node.js)
- Port allocation in configured range
- Process health monitoring
- Idle timeout and cleanup
- Managed process supervision
- WebSocket connection tracking

**Test Coverage**: 75.0%

---

### Reverse Proxy

#### `internal/proxy/`
**Purpose**: Reverse proxy with Fly.io integration

**Key Components**:

- Proxy request forwarding to tenant apps and reverse proxies
- Fly-Replay header generation for regional routing
- Sticky session support with cookies
- Large request detection and fallback for Fly-Replay
- WebSocket support with optional connection tracking

**Responsibilities**:

- Proxying requests to web apps (no retry - health checks ensure readiness)
- Smart region routing with retry fallback (Fly-Replay)
- Session affinity (sticky sessions)
- Maintenance page serving

**Test Coverage**: 88.1%

---

### Configuration

#### `internal/config/`
**Purpose**: YAML configuration loading and validation

**Key Components**:

- Configuration file parsing
- Template variable substitution
- Validation and defaults
- Hot reload support (SIGHUP)

**Responsibilities**:

- YAML file parsing
- Environment variable expansion
- Tenant-specific configuration
- Configuration validation
- Live reload handling

**Test Coverage**: 83.9%

---

### Authentication

#### `internal/auth/`
**Purpose**: HTTP Basic Authentication with htpasswd

**Key Components**:

- htpasswd file parsing (APR1, bcrypt, SHA)
- Pattern-based authentication exclusions
- Realm management
- Authorization header processing

**Responsibilities**:

- Credential verification
- Pattern matching for auth exclusions
- HTTP 401 response generation
- Multiple hash format support

**Test Coverage**: 78.4%

---

### Idle Management

#### `internal/idle/`
**Purpose**: Fly.io machine auto-suspend/stop

**Key Components**:

- `manager.go` - Idle detection and action triggering
- `signals_unix.go` - Unix/Linux/macOS signal handling (SIGTSTP, SIGTERM)
- `signals_windows.go` - Windows graceful shutdown (os.Exit)

**Responsibilities**:

- Request activity tracking
- Idle timeout detection
- Machine suspend (Unix: SIGTSTP)
- Machine stop (Unix: SIGTERM, Windows: os.Exit)
- Resume hook execution
- Platform-specific signal handling

**Test Coverage**: 74.3%

---

### Utility Packages

#### `internal/errors/`
**Purpose**: Domain-specific error constructors

**Components**:

- Error constructor functions for all domain areas
- Proper error wrapping with `%w` for error chains
- Consistent error message formatting

**Usage Example**:
```go
return errors.ErrTenantNotFound("tenant-name")
return errors.ErrPIDFileRead("/path/to/file", err)
return errors.ErrNoAvailablePorts(4000, 4099)
```

**Test Coverage**: 86.7%

---

#### `internal/logging/`
**Purpose**: Structured logging helper functions

**Components**:

- 24 logging helper functions
- Consistent slog-based structured logging
- Categorized by domain (Request, Proxy, Process, WebApp, etc.)

**Usage Example**:
```go
logging.LogWebAppStart(tenant, port, runtime, server, args)
logging.LogProxyRequest(method, path, target)
logging.LogServerReady(host, port)
```

**Benefits**:

- Reduces multi-line slog calls to single-line functions
- Consistent logging format across codebase
- Easier to maintain and update logging patterns

**Test Coverage**: 100%

---

#### `internal/utils/`
**Purpose**: Common utility functions

**Components**:

- Duration parsing with automatic error logging
- Environment variable handling
- Fly.io context detection
- Time utilities

**Key Functions**:
```go
ParseDurationWithDefault(input, defaultDuration)
ParseDurationWithContext(input, default, context)
```

**Test Coverage**: 77.9%

---

## Request Flow

Here's how Navigator handles an incoming HTTP request:

```
1. HTTP Request
   │
   ├──> [internal/server/handler.go]
   │    │
   │    ├──> Check authentication
   │    │    └──> [internal/auth/] - Verify credentials
   │    │
   │    ├──> Apply rewrite rules
   │    │    └──> Check Fly-Replay requirements
   │    │
   │    ├──> Try static files
   │    │    └──> [internal/server/static.go] - Serve from public/
   │    │
   │    └──> Proxy to web app
   │         │
   │         ├──> [internal/process/webapp.go] - Get or start app
   │         │    │
   │         │    └──> [internal/process/process_starter.go] - Start if needed
   │         │         │
   │         │         ├──> [internal/process/port_allocator.go] - Find port
   │         │         └──> Execute framework-specific command
   │         │
   │         └──> [internal/proxy/] - Forward request
   │              │
   │              ├──> Add Fly-Replay headers if needed
   │              ├──> Handle sticky sessions
   │              └──> Retry on connection errors
   │
   └──> [internal/server/access_log.go] - Log request
```

## Lifecycle Management

### Startup Sequence

```
1. Parse command-line arguments
2. Load configuration [internal/config/]
3. Initialize idle manager [internal/idle/]
4. Initialize app manager [internal/process/]
5. Start managed processes [internal/process/manager.go]
6. Execute server start hooks
7. Start HTTP server [internal/server/]
8. Execute server ready hooks
9. Begin accepting requests
```

### Shutdown Sequence

```
1. Receive SIGTERM/SIGINT
2. Stop accepting new requests
3. Wait for active requests (graceful shutdown)
4. Execute server idle hooks (if Fly.io machine idle)
5. Stop web applications [internal/process/webapp.go]
   └──> Execute tenant stop hooks
6. Stop managed processes [internal/process/manager.go]
7. Clean up PID files
8. Exit
```

### Configuration Reload (SIGHUP)

```
1. Receive SIGHUP signal
2. Parse new configuration [internal/config/]
3. Update app manager [internal/process/webapp.go]
   └──> Update idle timeout, port range
4. Update managed processes
   └──> Start new processes
   └──> Keep existing processes running
5. Update routing rules [internal/server/]
6. Continue serving requests (zero downtime)
```

## Concurrency Model

Navigator uses Go's concurrency primitives for safe multi-threaded operation:

### Request Handling

- **Goroutine per request** - Standard Go HTTP server model
- **No shared mutable state** in handler path
- **Read-only configuration** after load

### Process Management

- **Mutex-protected maps** - App registry (`internal/process/webapp.go`)
- **Atomic counters** - Active request tracking (`internal/idle/manager.go`)
- **Channel-based cleanup** - Graceful shutdown coordination

### Idle Monitoring

- **Per-app goroutines** - Each web app has dedicated idle monitor
- **Time-based timers** - Idle timeout detection
- **Condition variables** - Resume hook synchronization

## Testing Strategy

Navigator achieves 81.2% test coverage through:

### Unit Tests

- **Package-level tests** - Each package has `*_test.go` files
- **Mock-based testing** - Test mode flags prevent dangerous operations
- **Table-driven tests** - Comprehensive scenario coverage

### Integration Tests

- **HTTP round-trip tests** - Full request/response cycles
- **Process lifecycle tests** - App startup and shutdown
- **Configuration reload tests** - SIGHUP handling

### Platform-Specific Tests

- **Build tags** - Unix vs Windows test separation
- **Signal handling tests** - Test mode prevents actual signals
- **Cross-compilation** - CI tests all platforms

## Performance Characteristics

### Memory Usage

- **Base process**: ~20MB (vs ~100MB+ for nginx + Passenger)
- **Per web app**: ~50-200MB (Rails app dependent)
- **Per managed process**: Varies by process type

### Latency

- **Static files**: <1ms (direct filesystem serving)
- **Proxy overhead**: ~1-2ms (reverse proxy to web app)
- **App startup**: 2-10s (depends on framework)

### Scalability

- **Concurrent requests**: Thousands (Go HTTP server)
- **Web apps**: 10-100 per Navigator instance (config dependent)
- **Managed processes**: Limited by system resources

## Evolution from v0.7 to v0.11

### Before (v0.7.1) - Single File

```
cmd/navigator/main.go (3842 LOC)
├── All HTTP handling
├── All process management
├── All configuration
├── All authentication
└── All proxy logic
```

**Issues**:
- Hard to test specific components
- Difficult to maintain
- Low test coverage (~65%)
- Code duplication

### After (v0.11.0) - Modular

```
cmd/navigator-refactored/main.go (280 LOC)
├── internal/server/ (84.5% coverage)
├── internal/process/ (75.0% coverage)
├── internal/proxy/ (88.1% coverage)
├── internal/config/ (83.9% coverage)
├── internal/auth/ (78.4% coverage)
├── internal/idle/ (74.3% coverage)
├── internal/errors/ (86.7% coverage)
├── internal/logging/ (100% coverage)
└── internal/utils/ (77.9% coverage)
```

**Benefits**:
- ✅ Higher test coverage (81.2% overall)
- ✅ Easier to maintain and extend
- ✅ Clear separation of concerns
- ✅ Reduced code duplication
- ✅ Better error handling
- ✅ Cross-platform support

### Key Metrics

| Metric | v0.7.1 | v0.11.0 | Improvement |
|--------|--------|---------|-------------|
| Test Coverage | ~65% | 81.2% | +16.2% |
| Main File Size | 3842 LOC | 280 LOC | -92.7% |
| Packages | 1 | 9 | +8 |
| Platform Support | Unix only | Unix + Windows | Full cross-platform |
| Largest File | 3842 LOC | 351 LOC | -90.9% |

## Design Decisions

### Why Internal Packages?

- **Encapsulation**: Internal packages cannot be imported by external code
- **Clear API**: Only exported types and functions are public
- **Refactoring freedom**: Internal implementation can change without breaking imports

### Why Platform-Specific Files?

- **Build constraints**: `//go:build unix` and `//go:build windows`
- **Native behavior**: Use appropriate OS primitives
- **Clean compilation**: No platform-specific code in shared files

### Why Helper Packages (errors, logging)?

- **DRY principle**: Eliminate 271 repetitive logging calls
- **Consistency**: Standardized error and log formats
- **Maintainability**: Update format in one place

## Future Architecture Considerations

Potential areas for further improvement:

1. **Plugin system** - Dynamic framework support
2. **Metrics package** - Prometheus/OpenTelemetry integration
3. **Health check package** - Dedicated health endpoint handling
4. **Cache layer** - In-memory caching for static content
5. **Rate limiting** - Per-tenant request rate limiting

## See Also

- [REFACTORING.md](https://github.com/rubys/navigator/blob/main/REFACTORING.md) - Detailed refactoring history
- [CLAUDE.md](https://github.com/rubys/navigator/blob/main/CLAUDE.md) - Development guidelines
- [Configuration Reference](configuration/yaml-reference.md) - YAML configuration options
- [Getting Started](getting-started/index.md) - Installation and first app