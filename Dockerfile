# Build stage
FROM golang:1.21-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./


# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .


# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o rss-aggregator

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS support
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/rss-aggregator .

# Copy the template file (if any)
# COPY --from=builder /app/templates ./templates

# Expose the application port
EXPOSE 8080

# Command to run the executable
CMD ["./rss-aggregator"]
