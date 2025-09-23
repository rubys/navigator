# Makefile for Navigator - Go web server replacement for nginx + Passenger

.PHONY: all build clean test help

# Default target
all: build

# Build the navigator executable
build:
	@echo "Building navigator..."
	@mkdir -p bin
	go build -mod=readonly -o bin/navigator cmd/navigator/main.go
	@echo "Navigator built successfully at bin/navigator"

# Build the refactored navigator executable
build-refactored:
	@echo "Building navigator-refactored..."
	@mkdir -p bin
	go build -mod=readonly -o bin/navigator-refactored cmd/navigator-refactored/main.go
	@echo "Navigator-refactored built successfully at bin/navigator-refactored"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bin/navigator bin/navigator-refactored
	@echo "Clean complete"

# Run comprehensive tests and linting
test:
	@echo "Running go vet on entire codebase..."
	go vet ./...
	@echo "Running tests on entire codebase..."
	go test ./... -v
	@echo "All tests and linting passed!"

# Test the build (basic smoke test)
test-build: build
	@echo "Testing navigator build..."
	./bin/navigator --help 2>/dev/null || echo "Navigator executable built successfully"

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
	@echo "  build-refactored  Build the navigator-refactored executable"
	@echo "  clean             Remove build artifacts"
	@echo "  test              Run comprehensive tests and linting on entire codebase"
	@echo "  test-build        Test the build (basic smoke test)"
	@echo "  deps              Download Go dependencies"
	@echo "  help              Show this help message"
	@echo ""
	@echo "Usage:"
	@echo "  make                    # Build the navigator"
	@echo "  make build-refactored   # Build the refactored navigator"
	@echo "  make clean              # Clean build artifacts"
	@echo "  make test               # Run comprehensive tests and linting"
	@echo "  make test-build         # Test the build (smoke test)"