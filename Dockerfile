# Multi-stage build for optimal image size
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o stress-test ./main.go

# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl && \
    adduser -D -s /bin/sh -u 1001 appuser

# Set working directory
WORKDIR /home/appuser

# Copy binary from builder
COPY --from=builder /app/stress-test .

# Copy templates directory if it exists
# COPY --from=builder /app/templates ./templates/ 2>/dev/null || true

# Set proper permissions
RUN chown -R appuser:appuser /home/appuser && \
    chmod +x ./stress-test

# Switch to non-root user for security
USER appuser

# Expose application port
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Set environment variables
ENV GIN_MODE=release
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=8080

# Run the application
CMD ["./stress-test"]
