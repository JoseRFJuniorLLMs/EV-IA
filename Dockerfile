# Build stage
FROM golang:1.22-alpine AS builder

ARG VERSION=dev

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /app/server ./cmd/server

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 sigec && \
    adduser -u 1000 -G sigec -s /bin/sh -D sigec

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Copy configuration files if needed
# COPY --from=builder /app/config /app/config

# Change ownership
RUN chown -R sigec:sigec /app

# Switch to non-root user
USER sigec

# Expose ports
EXPOSE 8080 8081 9000 50051

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the server
ENTRYPOINT ["/app/server"]
