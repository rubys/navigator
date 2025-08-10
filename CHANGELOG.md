# Changelog

All notable changes to Navigator will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - 2025-08-10

### Added
- **Multi-tenant Rails proxy** with dynamic process management
- **Chi router integration** with middleware pipeline
- **Structured JSON logging** using logrus with contextual fields
- **HTTP caching** with 100MB LRU memory cache (70% performance improvement)
- **HTTP/2 support** with h2c (cleartext HTTP/2)
- **Automatic process recovery** for crashed Puma processes
- **HTTP Basic authentication** via htpasswd files (APR1, Bcrypt, SHA, Crypt)
- **Dynamic port management** starting from port 4000
- **Smart asset serving** with different TTLs for fingerprinted assets
- **Gzip compression** for text-based content
- **Health check endpoints** (`/up`, `/health`)
- **Graceful shutdown** with proper cleanup
- **Ruby-style YAML parsing** for showcases.yml configuration
- **Request tracing** with unique request IDs
- **Performance metrics** with millisecond timing

### Technical Features
- Built with modern Go libraries (Chi, Logrus, HTTP-Cache)
- Command-line configuration with sensible defaults
- Configurable idle timeouts and process limits
- Directory traversal protection
- Security headers (X-Frame-Options, X-Content-Type-Options, X-XSS-Protection)
- Production-ready error handling and logging

### Infrastructure
- GitHub Actions CI/CD pipeline
- Multi-platform builds (Linux/macOS/Windows, AMD64/ARM64)
- Docker support with multi-stage builds
- Systemd service configuration
- Makefile with development targets
- Comprehensive documentation and examples

### Initial Release Notes
This is the first stable release of Navigator, ready for production use in multi-tenant Rails environments. The application successfully replaces nginx/Passenger with modern Go performance and reliability.

**Performance benchmarks:**
- Static asset caching: 70% faster response times (1.49ms → 0.45ms)
- Memory usage: 100MB cache with LRU eviction
- Automatic process management with 5-minute idle timeout
- HTTP/2 support with request multiplexing

**Deployment ready:**
- Systemd integration
- Docker containerization
- Health check endpoints
- Structured logging for monitoring
- Graceful shutdown handling

[v0.1.0]: https://github.com/rubys/navigator/releases/tag/v0.1.0