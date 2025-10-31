# Makefile for Navigator - Go web server replacement for nginx + Passenger

.PHONY: all build clean lint test test-fast test-full test-integration test-stress help

# Version info for build
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "none")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.buildTime=$(BUILD_TIME)'

# Default target
all: build

# Build the navigator executable
build:
	@echo "Building navigator..."
	@mkdir -p bin
	go build -mod=readonly -ldflags="$(LDFLAGS)" -o bin/navigator cmd/navigator/main.go
	@echo "Navigator built successfully at bin/navigator"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bin/navigator
	@echo "Clean complete"

# Run all linting checks (matches CI lint job)
lint:
	@echo "Running go mod verify..."
	go mod verify
	@echo "Running go vet..."
	go vet ./...
	@echo "Running gofmt check..."
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "The following files are not properly formatted:"; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "Running golangci-lint..."
	golangci-lint run
	@echo "All lint checks passed!"

# Run fast tests (default for development)
test: test-fast

# Run fast tests only (excludes integration and stress tests)
test-fast:
	@echo "Running fast tests..."
	go test ./...
	@echo "Fast tests passed!"

# Run full comprehensive test suite (includes integration and stress tests)
test-full: build
	@echo "Running comprehensive test suite (integration + stress tests)..."
	go test -tags="integration" ./... -v
	@echo "Full test suite passed!"

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
	@echo "  build             Build the navigator executable (default)"
	@echo "  clean             Remove build artifacts"
	@echo "  lint              Run all linting checks (matches CI)"
	@echo "  test              Run fast tests (default - excludes integration/stress tests)"
	@echo "  test-fast         Run fast tests only (~6 seconds, or 0.3s with cache)"
	@echo "  test-full         Run comprehensive test suite with integration and stress tests (~86 seconds)"
	@echo "  test-integration  Run integration tests only (~50 seconds)"
	@echo "  test-stress       Run stress tests only"
	@echo "  deps              Download Go dependencies"
	@echo "  help              Show this help message"
	@echo ""
	@echo "Usage:"
	@echo "  make                    # Build the navigator"
	@echo "  make lint               # Run all linting checks"
	@echo "  make test               # Run fast tests (development)"
	@echo "  make test-full          # Run comprehensive tests (before release)"
	@echo "  make test-integration   # Run integration tests only"
	@echo "  make clean              # Clean build artifacts"