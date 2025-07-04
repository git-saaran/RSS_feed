# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk --no-cache add git ca-certificates tzdata

# Set the working directory
WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o rss-aggregator .

# Final stage - minimal image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget curl && \
    cp /usr/share/zoneinfo/Asia/Kolkata /etc/localtime && \
    echo "Asia/Kolkata" > /etc/timezone && \
    apk del tzdata

# Create non-root user for security
RUN addgroup -g 1001 appgroup && \
    adduser -u 1001 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/rss-aggregator .

# Change ownership and make executable
RUN chown appuser:appgroup rss-aggregator && \
    chmod +x rss-aggregator

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check for enhanced features
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/status || exit 1

# Run the application
CMD ["./rss-aggregator"]
