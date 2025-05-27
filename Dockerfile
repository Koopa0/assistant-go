# GoAssistant Dockerfile
# Multi-stage build following Go 1.24+ best practices

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o goassistant \
    ./cmd/assistant

# Final stage
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/goassistant /goassistant

# Copy configuration files
COPY --from=builder /app/configs /configs

# Copy static files and templates (when they exist)
COPY --from=builder /app/internal/web/static /internal/web/static
COPY --from=builder /app/internal/web/templates /internal/web/templates

# Set environment variables
ENV APP_MODE=production
ENV LOG_LEVEL=info
ENV LOG_FORMAT=json
ENV SERVER_ADDRESS=:8080

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/goassistant", "health"] || exit 1

# Run as non-root user
USER 65534:65534

# Set entrypoint
ENTRYPOINT ["/goassistant"]

# Default command
CMD ["serve"]
