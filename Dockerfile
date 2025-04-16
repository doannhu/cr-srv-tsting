# Build stage
FROM golang:1.21-alpine AS builder

# Install protoc and required tools
RUN apk add --no-cache \
    protoc \
    protobuf-dev \
    git \
    make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate protobuf code
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/credit_enquiry.proto

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy necessary files
COPY --from=builder /app/proto ./proto
COPY --from=builder /app/internal ./internal

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 50051

# Set environment variables
ENV REDIS_HOST=localhost \
    REDIS_PORT=6379 \
    SERVER_PORT=50051 \
    SPANNER_PROJECT="" \
    SPANNER_INSTANCE="" \
    SPANNER_DATABASE="" \
    PUBSUB_PROJECT="" \
    PUBSUB_TOPIC="credit-enquiry-events"

# Run the application
CMD ["./server"] 