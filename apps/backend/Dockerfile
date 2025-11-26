# Stage 1: Builder
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy modules manifests
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /main ./cmd/api

# Stage 2: Development (Air Hot Reload)
FROM golang:1.23-alpine AS dev
WORKDIR /app
RUN apk add --no-cache git
# Pin Air to a version compatible with Go 1.23
RUN go install github.com/air-verse/air@v1.52.3
# Install Swag for documentation generation inside Docker
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY go.mod go.sum* ./
RUN go mod download
COPY . .
CMD ["air"]

# Stage 3: Production
FROM alpine:latest AS prod
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /main .
# Create a non-root user for security
RUN adduser -D -g '' appuser
USER appuser
CMD ["./main"]
