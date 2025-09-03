# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY . .
RUN go mod download && \
    GOOS=linux GOARCH=amd64 go build -mod=readonly -o navigator cmd/navigator/main.go

# Final stage - minimal image with just the binary
FROM scratch
COPY --from=builder /build/navigator /navigator
ENTRYPOINT ["/navigator"]