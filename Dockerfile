# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Enable automatic toolchain download for newer Go versions
ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apk add --no-cache git ca-certificates nodejs npm

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build frontend
RUN cd web && npm install && npm run build

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /cryptojackal ./cmd/cryptojackal
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /demo ./cmd/demo

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 cryptojackal

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /cryptojackal /usr/local/bin/cryptojackal
COPY --from=builder /demo /usr/local/bin/demo

# Copy built frontend so Go static file serving works
COPY --from=builder /app/web/dist ./web/dist

# Copy configuration
COPY .env.example .env.example

# Create directories
RUN mkdir -p /app/logs /app/data && \
    chown -R cryptojackal:cryptojackal /app

USER cryptojackal

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1

# Default command
CMD ["cryptojackal"]
