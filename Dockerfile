# Build stage
FROM golang:1.25-alpine AS builder

# Set build arguments
ARG VERSION=dev
ARG GOOS=linux
ARG GOARCH=amd64

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod ./
COPY go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o task-breaker ./cmd/chat.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appuser && \
    adduser -u 1001 -S appuser -G appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/task-breaker .

# Create directory for config and context files
RUN mkdir -p /app/data && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose any potential ports (if needed for future features)
EXPOSE 8080

# Set environment variables
ENV PATH="/app:${PATH}"
ENV CONFIG_PATH="/app/data/.task-breaker-config.json"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/app/task-breaker", "--help"]

# Default command
ENTRYPOINT ["/app/task-breaker"]