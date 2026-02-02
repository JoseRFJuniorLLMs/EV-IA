# ============================================
# SIGEC-VE Enterprise - Multi-stage Dockerfile
# ============================================
# This Dockerfile creates a minimal, secure production image
# Final image size: ~20MB (vs. 1GB+ with standard Go image)

# ============================================
# Stage 1: Build Stage
# ============================================
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    make \
    protobuf \
    protobuf-dev

# Set working directory
WORKDIR /build

# Copy go mod files first (for better layer caching)
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Generate Protocol Buffers
RUN make proto-gen

# Build the application
# CGO_ENABLED=0: Creates a statically linked binary
# -ldflags: Reduces binary size by stripping debug info
# -trimpath: Removes file system paths from binary
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s \
    -X 'main.Version=${VERSION}' \
    -X 'main.BuildTime=${BUILD_TIME}' \
    -X 'main.GitCommit=${GIT_COMMIT}'" \
    -trimpath \
    -o /build/sigec-server \
    ./cmd/server/main.go

# Build the worker binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s" \
    -trimpath \
    -o /build/sigec-worker \
    ./cmd/worker/main.go

# ============================================
# Stage 2: Runtime Stage (Distroless)
# ============================================
# Distroless images are ultra-minimal, containing only:
# - Your application
# - Runtime dependencies
# - NO shell, package managers, or unnecessary binaries
# This drastically reduces attack surface

FROM gcr.io/distroless/static-debian12:nonroot

# Import timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /build/sigec-server /app/sigec-server

# Copy configuration files (will be overridden by ConfigMap in k8s)
COPY --from=builder /build/configs /app/configs

# Set working directory
WORKDIR /app

# Set timezone
ENV TZ=America/Sao_Paulo

# Expose ports
# 8080: HTTP/REST API
# 9000: OCPP WebSocket
# 50051: gRPC
EXPOSE 8080 9000 50051

# Use non-root user (from distroless base)
USER nonroot:nonroot

# Health check (note: distroless doesn't have shell, so this is for Docker only)
# In Kubernetes, use liveness/readiness probes instead
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/sigec-server", "healthcheck"] || exit 1

# Run the application
ENTRYPOINT ["/app/sigec-server"]

# ============================================
# Alternative: Alpine-based runtime (if you need shell for debugging)
# ============================================
FROM alpine:3.19 AS runtime-alpine

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/sigec-server /app/sigec-server
COPY --from=builder /build/configs /app/configs

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 8080 9000 50051

HEALTHCHECK --interval=30s --timeout=3s \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

ENTRYPOINT ["/app/sigec-server"]

# ============================================
# Build instructions:
# ============================================
# Development build:
#   docker build -t sigec-ve:dev .
#
# Production build with version:
#   docker build \
#     --build-arg VERSION=v1.0.0 \
#     --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
#     --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
#     -t sigec-ve:v1.0.0 .
#
# Multi-platform build (for ARM64 servers):
#   docker buildx build \
#     --platform linux/amd64,linux/arm64 \
#     -t gcr.io/your-project/sigec-ve:latest \
#     --push .
#
# Run locally:
#   docker run -p 8080:8080 -p 9000:9000 \
#     -e DATABASE_URL=postgres://... \
#     -e REDIS_URL=redis://... \
#     sigec-ve:dev
# ============================================
