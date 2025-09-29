# Makefile for Navigator - Go web server replacement for nginx + Passenger

.PHONY: all build build-legacy build-refactored clean test test-fast test-full test-integration test-stress help

# Default target
all: build

build: build-legacy build-refactored

# Build the navigator executable
build-legacy:
	@echo "Building navigator-legacy..."
	@mkdir -p bin
	go build -mod=readonly -o bin/navigator-legacy cmd/navigator-legacy/main.go
	@echo "Navigator built successfully at bin/navigator-legacy"

# Build the refactored navigator executable
build-refactored:
	@echo "Building navigator-refactored..."
	@mkdir -p bin
	go build -mod=readonly -o bin/navigator-refactored cmd/navigator-refactored/main.go
	@echo "Navigator-refactored built successfully at bin/navigator-refactored"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bin/navigator-legacy bin/navigator-refactored
	@echo "Clean complete"

# Run fast tests (default for development)
test: test-fast

# Run fast tests only (excludes integration and stress tests)
test-fast: build
	@echo "Running go vet on entire codebase..."
	go vet ./...
	@echo "Running fast tests..."
	go test ./... -v
	@echo "Fast tests and linting passed!"

# Run full comprehensive test suite (includes integration and stress tests)
test-full: build
	@echo "Running go vet on entire codebase..."
	go vet ./...
	@echo "Running comprehensive test suite (integration + stress tests)..."
	go test -tags="integration" ./... -v
	@echo "Full test suite and linting passed!"

# Run integration tests only
test-integration: build
	@echo "Running integration tests..."
	go test -tags="integration" ./... -v
	@echo "Integration tests passed!"

# Run stress tests only
test-stress: build
	@echo "Running stress tests..."
	go test -tags="stress" ./... -v
	@echo "Stress tests passed!"

# Install dependencies (if needed)
deps:
	@echo "Installing Go dependencies..."
	go mod download

# Show help
help:
	@echo "Navigator Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build             Build the navigator executables (default)"
	@echo "  build-legacy      Build the navigator-legacy executable"
	@echo "  build-refactored  Build the navigator-refactored executable"
	@echo "  clean             Remove build artifacts"
	@echo "  test              Run fast tests (default - excludes integration/stress tests)"
	@echo "  test-fast         Run fast tests only (~5 seconds)"
	@echo "  test-full         Run comprehensive test suite with integration and stress tests (~15+ seconds)"
	@echo "  test-integration  Run integration tests only (~11 seconds)"
	@echo "  test-stress       Run stress tests only"
	@echo "  deps              Download Go dependencies"
	@echo "  help              Show this help message"
	@echo ""
	@echo "Usage:"
	@echo "  make                    # Build the navigator"
	@echo "  make test               # Run fast tests (development)"
	@echo "  make test-full          # Run comprehensive tests (before release)"
	@echo "  make test-integration   # Run integration tests only"
	@echo "  make clean              # Clean build artifacts"