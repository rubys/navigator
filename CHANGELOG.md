# Changelog

All notable changes to Navigator will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2025-11-21

### Added
- **Immutable directive support** for optimal asset caching - fingerprinted assets can now use `immutable` flag for better browser caching
- **Response headers support** in reverse proxy - configure custom response headers to add/modify in proxied responses
- **Response write logging** - new debug logging for response writes with optional compression disabling via `disable_compression` config
- **trust_proxy configuration** - proper handling of X-Forwarded-For and X-Real-IP headers when behind proxies
- **Vector integration enhancements** - Navigator access logs now sent to Vector for centralized log aggregation
- **Broadcast logging** - enhanced with message content for better debugging

### Fixed
- **Duration parsing** - now correctly supports documented extended formats (y, w, d) in addition to standard Go formats
- **Cache control** - properly handles `max-age=0` for HTML revalidation while using long-lived caching for assets
- **Vector socket cleanup** - improved cleanup during config reload to prevent stale sockets
- **Tenant logs** - fixed issue where tenant logs weren't being sent to Vector/NATS
- **trust_proxy parsing** - corrected configuration parsing for proxy trust settings

### Improved
- **WebSocket hardening** - applied security recommendations from @palkan for more robust WebSocket handling
- **Vector socket management** - cleaner startup with automatic stale socket cleanup and better logging visibility
- **Debug logging** - added comprehensive debug output for trust_proxy troubleshooting
- **Documentation** - improved Navigator logging documentation

## [1.1.0] - 2025-01-15

### Added
- Vector logging integration with automatic process management
- NATS support for distributed logging
- Structured logging with JSON output format
- Multiple log destinations (console + file)

### Improved
- Enhanced error handling and retry logic
- Better process lifecycle management

## [1.0.0] - 2025-01-08

### Initial Production Release
- Multi-tenant application support with isolated processes
- On-demand process management with auto-restart
- TurboCable WebSocket support (89% memory savings)
- Static file serving with try-files
- htpasswd authentication with flexible exclusions
- Hot configuration reload (SIGHUP)
- Regional routing with Fly-Replay
- Machine auto-suspend on Fly.io
- Lifecycle hooks for server and tenant events
- CGI script execution
- Managed external processes (Redis, Sidekiq, etc.)
- Comprehensive documentation site

**Production Status**: Battle-tested with 75+ customers across 8 countries, 81.2% test coverage

[1.2.0]: https://github.com/rubys/navigator/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/rubys/navigator/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/rubys/navigator/releases/tag/v1.0.0
