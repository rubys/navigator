# Build stage
FROM golang:1.24-alpine AS builder

# Install git for version info
RUN apk add --no-cache git

WORKDIR /build
COPY . .

# Build with version info from git or build args
ARG VERSION
ARG COMMIT
ARG BUILD_TIME

RUN go mod download && \
    # Get version info from git or use build args
    if [ -z "$VERSION" ]; then \
        VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
    fi && \
    if [ -z "$COMMIT" ]; then \
        COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "none"); \
    fi && \
    if [ -z "$BUILD_TIME" ]; then \
        BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ); \
    fi && \
    GOOS=linux GOARCH=amd64 go build -mod=readonly \
        -ldflags="-X 'main.version=$VERSION' -X 'main.commit=$COMMIT' -X 'main.buildTime=$BUILD_TIME'" \
        -o navigator cmd/navigator/main.go

# Final stage - minimal image with just the binary
FROM scratch
COPY --from=builder /build/navigator /navigator
ENTRYPOINT ["/navigator"]