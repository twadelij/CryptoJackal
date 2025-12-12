# =============================================================================
# CryptoJackal Multi-Stage Docker Build
# =============================================================================
# Optimized for development, testing, and production deployments

# Build Stage
FROM rust:latest AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy Cargo files
COPY Cargo.toml Cargo.lock ./

# Create dummy main.rs to cache dependencies
RUN mkdir src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

# Copy source code
COPY src ./src
COPY abi ./abi

# Build the application
RUN touch src/main.rs && cargo build --release

# Runtime Stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libssl3 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 1000 cryptojackal

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/target/release/cryptojackal /usr/local/bin/cryptojackal
COPY --from=builder /app/target/release/demo /usr/local/bin/demo

# Copy configuration files
COPY .env.example .env.example
COPY assets ./assets

# Create necessary directories
RUN mkdir -p /app/logs /app/data && \
    chown -R cryptojackal:cryptojackal /app

# Switch to non-root user
USER cryptojackal

# Expose ports
EXPOSE 8080 8081 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

# Default command
CMD ["cryptojackal"]
