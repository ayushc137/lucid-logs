# Lucid Logs Go Backend - Dockerfile
# =====================================
# Multi-stage build for minimal production image

# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the API binary (platform-agnostic)
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o /app/bin/api \
    ./cmd/api

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/bin/api /app/api

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/ || exit 1

ENTRYPOINT ["/app/api"]
