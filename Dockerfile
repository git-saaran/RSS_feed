# Build stage
FROM golang:1.21-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy everything first
COPY . .

# Initialize Go module and download dependencies
RUN go mod init rss-aggregator 2>/dev/null || true && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o rss-aggregator

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS support and set timezone
RUN apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Kolkata /etc/localtime && \
    echo "Asia/Kolkata" > /etc/timezone && \
    apk del tzdata

# Set the working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/rss-aggregator .

# Expose the application port
EXPOSE 8080

# Command to run the executable
CMD ["./rss-aggregator"]
