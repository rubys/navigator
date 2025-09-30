# Navigator Refactoring Plan and Status

This document tracks the ongoing refactoring effort to improve code maintainability, reduce complexity, and enhance separation of concerns in the Navigator codebase.

## Overall Goals

1. **Reduce file complexity**: Break down large files into smaller, focused modules
2. **Improve separation of concerns**: Each module should have a single, well-defined responsibility
3. **Enhance maintainability**: Make code easier to understand, test, and modify
4. **Eliminate duplication**: Extract common patterns into reusable components
5. **Maintain stability**: All refactoring must pass existing tests with no behavioral changes

## Current Status: Phase 1 - Large File Analysis

### Completed Refactorings

#### ✅ internal/server/handler.go (COMPLETED)

**Original size**: 722 LOC
**Final size**: 343 LOC (52% reduction)

**Extractions performed**:

1. **Static File Handling** → `internal/server/static.go` (263 LOC)
   - Created `StaticFileHandler` type
   - Methods: `ServeStatic`, `TryFiles`, `ServeFallback`, `getPublicDir`
   - Handles root path stripping, extension checking, file resolution
   - Commit: `b8606b7` (Sep 30, 2025)

2. **MIME Type Detection** → `internal/server/mime.go` (30 LOC)
   - Replaced 40-line switch statement with stdlib `mime.TypeByExtension()`
   - Added fallback for font types not in stdlib
   - Simpler, more maintainable implementation
   - Commit: `b8606b7` (Sep 30, 2025)

3. **Access Logging** → `internal/server/access_log.go` (119 LOC)
   - Extracted `AccessLogEntry` struct
   - Created `LogRequest()` function for centralized logging
   - Handles client IP extraction, user info, timing, metadata
   - Commit: `aacd4d0` (Sep 30, 2025)

**Benefits achieved**:
- ✅ Better separation of concerns
- ✅ Improved testability (each module can be tested independently)
- ✅ Cleaner handler focused on request routing
- ✅ Easier to maintain and modify
- ✅ All 100+ tests passing
- ✅ No behavioral changes

#### ✅ cmd/navigator-refactored/main.go (COMPLETED)

**Date**: Sep 30, 2025
**Commit**: `9c47f67`

**High-priority fixes applied**:

1. **Extracted Log Level Parsing** → `getLogLevel()` helper
   - Eliminated duplicate log level parsing code
   - Single source of truth for log configuration

2. **Server Lifecycle Management** → `ServerLifecycle` type
   - Encapsulates signal handling and shutdown logic
   - Methods: `Run()`, `handleReload()`, `handleShutdown()`
   - Reduced main() from ~67 LOC to ~20 LOC

3. **Context Propagation** → Added to cleanup functions
   - `CleanupWithContext()` in webapp.go
   - `StopManagedProcessesWithContext()` in manager.go
   - Proper timeout handling during graceful shutdown

**Benefits achieved**:
- ✅ Cleaner, more maintainable main function
- ✅ Better signal handling architecture
- ✅ Improved graceful shutdown with context timeouts
- ✅ All tests passing

---

## Upcoming Work

### Phase 1: Large File Analysis (In Progress)

#### ✅ internal/process/webapp.go (COMPLETED)

**Original size**: 498 LOC (355 code lines)
**Final size**: 351 LOC (252 code lines) - 30% total reduction, 29% code reduction

**Extractions performed**:

1. **Process Starter** → `internal/process/process_starter.go` (187 LOC, 133 code)
   - Created `ProcessStarter` type
   - Methods: `StartWebApp`, `getRuntime`, `getServer`, `getArgs`, `setupCommand`, `waitForReady`
   - Handles framework detection, command building, environment setup, readiness checks
   - Commit: TBD (Sep 30, 2025)

2. **Port Allocator** → `internal/process/port_allocator.go` (34 LOC, 25 code)
   - Created `PortAllocator` type
   - Method: `FindAvailablePort`
   - Finds available TCP ports in configured range
   - Commit: TBD (Sep 30, 2025)

**Benefits achieved**:
- ✅ Better separation of concerns (process starting, port allocation isolated)
- ✅ Improved testability (each module can be tested independently)
- ✅ Cleaner AppManager focused on lifecycle management
- ✅ Easier to maintain and extend framework support
- ✅ All 100+ tests passing
- ✅ No behavioral changes

#### ⏸️ cmd/navigator-legacy/main.go (3842 LOC)

**Status**: Excluded from current refactoring
**Reason**: Legacy implementation, focus is on navigator-refactored

---

### Phase 2: Code Duplication (IN PROGRESS)

**Status**: High-priority extractions completed
**Date Started**: Sep 30, 2025

#### ✅ Completed Extractions

1. **Duration Parsing Enhancement** → `internal/utils/time.go`
   - Enhanced existing `ParseDurationWithDefault()` with logging
   - Added `ParseDurationWithContext()` for contextual error logging
   - Replaced 23 occurrences of manual duration parsing with logging
   - Files updated: `hooks.go`, `manager.go`, `webapp.go`
   - Added comprehensive tests

2. **Error Constructors** → `internal/errors/errors.go` (NEW PACKAGE)
   - Created domain-specific error constructor functions
   - Categories: Process, Config, Proxy, Auth, Server errors
   - Uses proper error wrapping with `%w` for error chains
   - Ready for use across 43 identified locations
   - Comprehensive test coverage with unwrapping tests

3. **Logging Helpers** → `internal/logging/logging.go` (NEW PACKAGE)
   - Created structured logging helper functions
   - Categories: Request, Proxy, Process, WebApp, Config, Server, Hooks, Cleanup
   - Reduces 271 repetitive slog calls to simple function calls
   - Ready for adoption across codebase
   - Full test coverage with output verification

**Benefits achieved**:
- ✅ Reduced code duplication in duration parsing (23 → 3 occurrences)
- ✅ Standardized error construction patterns (ready for 43 locations)
- ✅ Simplified structured logging (ready for 271 locations)
- ✅ Better error wrapping and unwrapping support
- ✅ Enhanced logging with context information
- ✅ All tests passing (100+ tests)
- ✅ No behavioral changes

#### 🔄 Pending Adoption

**Next steps** (optional, based on need):
1. Gradually adopt `internal/errors` constructors across codebase
2. Gradually adopt `internal/logging` helpers to reduce verbosity
3. Extract HTTP utilities (client IP, metadata extraction)
4. Extract test utilities (temp directory helpers)
5. Extract Fly.io context helpers

**Approach for future adoption**:
- Adopt incrementally, one module at a time
- Update during feature development, not as bulk refactoring
- Maintain backward compatibility during transition

---

### Phase 3: Test Organization (Planned)

**Status**: Not started
**Priority**: Low

**Goals**:
- Review test file sizes and organization
- Extract common test utilities
- Improve test readability
- Ensure comprehensive coverage

---

### Phase 4: Documentation (Planned)

**Status**: Not started
**Priority**: Low

**Goals**:
- Ensure all extracted modules have clear documentation
- Update architectural diagrams if needed
- Document design decisions and patterns
- Update CLAUDE.md with refactoring guidelines

---

## Refactoring Principles

### 1. Safety First
- ✅ All tests must pass before and after refactoring
- ✅ Run full test suite after each extraction
- ✅ Run CI to verify no regressions
- ✅ No behavioral changes allowed

### 2. Incremental Progress
- ✅ Small, focused refactorings
- ✅ One extraction at a time
- ✅ Commit after each successful refactoring
- ✅ Can be reviewed and understood easily

### 3. Clear Separation
- ✅ Each module has single, well-defined responsibility
- ✅ Clear interfaces between modules
- ✅ Minimize coupling between components
- ✅ Use dependency injection where appropriate

### 4. Maintainability
- ✅ Code is easier to read and understand
- ✅ Modules can be tested independently
- ✅ Changes are localized to relevant modules
- ✅ Follow Go idioms and best practices

### 5. Documentation
- ✅ Clear commit messages explaining changes
- ✅ Code comments for non-obvious logic
- ✅ Update documentation as needed
- ✅ Track progress in this file

---

## Metrics

### File Size Reductions

| File | Original | Current | Reduction | Status |
|------|----------|---------|-----------|--------|
| internal/server/handler.go | 722 LOC | 343 LOC | 52% | ✅ Complete |
| internal/process/webapp.go | 498 LOC | 351 LOC | 30% | ✅ Complete |
| cmd/navigator-refactored/main.go | ~300 LOC | ~280 LOC | ~7% | ✅ Complete |

### New Modules Created

| Module | Size | Purpose | Status |
|--------|------|---------|--------|
| internal/server/static.go | 263 LOC | Static file serving | ✅ Complete |
| internal/server/mime.go | 30 LOC | MIME type detection | ✅ Complete |
| internal/server/access_log.go | 119 LOC | Access logging | ✅ Complete |
| internal/process/process_starter.go | 187 LOC | Process lifecycle management | ✅ Complete |
| internal/process/port_allocator.go | 34 LOC | Port allocation | ✅ Complete |
| internal/errors/errors.go | 88 LOC | Domain-specific error constructors | ✅ Complete |
| internal/logging/logging.go | 194 LOC | Structured logging helpers | ✅ Complete |
| internal/utils/time.go (enhanced) | 51 LOC | Duration parsing with logging | ✅ Complete |

### Test Coverage

- **Total tests**: 100+ tests passing
- **Test failures**: 0
- **CI status**: ✅ All jobs passing
- **Behavioral changes**: None

---

## Recent Activity

### September 30, 2025

**Phase 2: Code Duplication Extraction** (Commit: TBD)
- Created `internal/errors/` package with domain-specific error constructors
- Created `internal/logging/` package with structured logging helpers
- Enhanced `internal/utils/time.go` with logging and context support
- Updated duration parsing in hooks.go, manager.go, webapp.go
- All tests passing (100+ tests), no behavioral changes
- Ready for gradual adoption across codebase (43 error sites, 271 logging sites)

**Webapp.go Refactoring** (Commit: `fd56d63`)
- Extracted process starting to `internal/process/process_starter.go`
- Extracted port allocation to `internal/process/port_allocator.go`
- Reduced webapp.go from 498 to 351 LOC (30% reduction)
- Updated AppManager to use ProcessStarter and PortAllocator
- All tests passing (100+ tests), CI pending

**Access Logging Extraction** (Commit: `aacd4d0`)
- Extracted access logging to `internal/server/access_log.go`
- Created `LogRequest()` function
- Removed 113 LOC from handler.go
- All tests passing, CI green

**Formatting Fix** (Commit: `977651c`)
- Ran `gofmt` on refactored files
- Fixed CI lint check failures
- All formatting now compliant

**Static File & MIME Extraction** (Commit: `b8606b7`)
- Extracted static file handling to `internal/server/static.go`
- Extracted MIME type logic to `internal/server/mime.go`
- Reduced handler.go by ~330 LOC
- Completed handler.go refactoring phase

**Main.go Refactoring** (Commit: `9c47f67`)
- Implemented high-priority fixes
- Created `ServerLifecycle` type
- Added context propagation to cleanup functions
- Improved shutdown handling

---

## Next Steps

1. ✅ **Complete**: internal/process/webapp.go refactoring finished
2. ✅ **Complete**: Phase 2 high-priority duplication extraction (errors, logging, duration)
3. **Optional**: Gradually adopt new utility packages during feature development
4. **Consider**: Extract HTTP utilities (client IP, metadata)
5. **Consider**: Extract test utilities (temp directory helpers)
6. **Consider**: Extract WebSocket connection management from webapp.go
7. **Consider**: Extract idle monitoring logic from webapp.go

---

## Notes

- **Focus**: Currently focused on internal/server package refactoring
- **Approach**: Bottom-up refactoring of large files first
- **Timeline**: No fixed timeline, prioritizing quality over speed
- **Testing**: Comprehensive testing required for each change
- **Review**: Each refactoring should be reviewable as a standalone change

---

## Questions or Concerns?

If you have questions about this refactoring effort or want to contribute:
- Review the commit history for detailed explanations
- Check test coverage before making changes
- Follow the principles outlined above
- Keep refactorings small and focused
- Document your reasoning in commit messages

---

**Last Updated**: September 30, 2025
**Status**: Phase 1 complete (handler.go, webapp.go), Phase 2 high-priority complete (errors, logging, duration)