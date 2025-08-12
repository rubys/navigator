# Navigator Makefile

BINARY_NAME=navigator
VERSION?=$(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

.PHONY: build clean test lint fmt vet deps install release help

# Default target
build: ## Build the binary
	go build ${LDFLAGS} -o ${BINARY_NAME} cmd/navigator/main.go

# Development targets
dev: ## Build and run in development mode
	go run cmd/navigator/main.go serve --log-level debug

clean: ## Clean build artifacts
	rm -f ${BINARY_NAME}
	rm -f navigator-*
	go clean

test: ## Run tests
	go test -race -cover ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	go vet ./...

deps: ## Download dependencies
	go mod download
	go mod tidy

# Release targets
release: clean build-all ## Build release binaries for all platforms

build-all: ## Build for all platforms
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 cmd/navigator/main.go
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-arm64 cmd/navigator/main.go
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 cmd/navigator/main.go
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 cmd/navigator/main.go
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe cmd/navigator/main.go

install: build ## Install binary to GOPATH/bin
	go install ${LDFLAGS} ./cmd/navigator

docker: ## Build Docker image
	docker build -t navigator:${VERSION} .
	docker tag navigator:${VERSION} navigator:latest

# Documentation
docs: ## Generate documentation
	godoc -http=:6060

# Quality checks
check: fmt vet test lint ## Run all quality checks

# Development helpers
watch: ## Watch files and rebuild on changes (requires entr)
	find . -name '*.go' | entr -r make dev

profile: ## Run with profiling enabled
	go build ${LDFLAGS} -o ${BINARY_NAME} cmd/navigator/main.go
	./${BINARY_NAME} serve --log-level debug -cpuprofile cpu.prof -memprofile mem.prof

# Help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)