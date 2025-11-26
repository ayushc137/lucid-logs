# Stage 1: Builder - Compiles the Rust application
FROM rust:1.83-alpine AS builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache musl-dev openssl-dev openssl-libs-static pkgconfig

# Create a new empty project for dependency caching
RUN cargo new --bin rust_backend
WORKDIR /app/rust_backend

# Copy manifests
COPY Cargo.toml Cargo.lock ./

# Build dependencies only (this layer will be cached)
RUN cargo build --release && rm -rf src

# Copy source code
COPY src ./src

# Build the actual application
RUN touch src/main.rs && cargo build --release

# Stage 2: Development - Hot reload with cargo-watch
FROM rust:1.83-alpine AS dev
WORKDIR /app

# Install development dependencies
RUN apk add --no-cache musl-dev openssl-dev pkgconfig

# Install cargo-watch for hot reloading
RUN cargo install cargo-watch

# Copy source code
COPY . .

# Download dependencies
RUN cargo fetch

# Expose port
EXPOSE 8080

# Start with hot reload
CMD ["cargo", "watch", "-x", "run"]

# Stage 3: Production - Minimal runtime image
FROM alpine:latest AS prod
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy the built binary from builder
COPY --from=builder /app/rust_backend/target/release/rust_backend ./main

# Create a non-root user for security
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8080

# Set environment
ENV APP_ENV=production
ENV RUST_LOG=info

# Run the application
CMD ["./main"]

