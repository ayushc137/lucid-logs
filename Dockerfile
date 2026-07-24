# syntax=docker/dockerfile:1

# ============================================================================
# Lucid Logs — all-in-one image
# Single container serving the SPA frontend + Go API + embedded libSQL (SQLite).
# No external database or reverse proxy required.
#
#   docker build -t lucid-logs .
#   docker run -p 8080:8080 \
#     -e ADMIN_SEED=true -e ADMIN_USERNAME=you@example.com -e ADMIN_PASSWORD=secret \
#     -e JWT_SECRET=$(openssl rand -hex 32) \
#     -v lucid-data:/data lucid-logs
# ============================================================================

# ---------------------------------------------------------------------------
# Stage 1 — build the frontend (static SPA)
# ---------------------------------------------------------------------------
FROM node:20-alpine AS frontend
# Install pnpm directly (corepack's bundled pnpm hits ERR_UNKNOWN_BUILTIN_MODULE
# on some node:20-alpine builds).
RUN npm install -g pnpm@9

WORKDIR /fe
COPY apps/frontend/package.json apps/frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY apps/frontend/ ./
# Build the SPA bundle (adapter-static). VITE_API_URL stays empty so the UI
# calls the same origin it's served from.
RUN VITE_SPA=1 pnpm build

# ---------------------------------------------------------------------------
# Stage 2 — build the Go backend with the frontend embedded
# ---------------------------------------------------------------------------
FROM golang:1.24-bookworm AS backend
# glibc toolchain so the CGO/turso binary matches the Debian runtime.
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY apps/go_backend/go.mod apps/go_backend/go.sum ./
RUN go mod download

COPY apps/go_backend/ ./
# Inject the built SPA so go:embed (internal/web/dist) bundles it into the binary.
RUN rm -rf internal/web/dist
COPY --from=frontend /fe/build ./internal/web/dist

# CGO must be enabled: the turso Go driver binds to a native library and forces
# a dynamically-linked (libc) binary. Building with CGO_ENABLED=0 would yield a
# broken artifact that fails to exec on a glibc runtime.
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-w -s" -o /app/bin/api ./cmd/api

# ---------------------------------------------------------------------------
# Stage 3 — minimal runtime
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim
# The embedded turso native library (libturso_sync_sdk_kit.so) is glibc-linked,
# so the runtime must be glibc-based (Alpine/musl cannot load it).
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/* && \
    useradd -m appuser

WORKDIR /app
COPY --from=backend /app/bin/api /app/api
COPY db/migrations ./db/migrations

# Persisted SQLite database lives here (mount a volume at /data).
RUN mkdir -p /data && chown -R appuser:appuser /data /app
USER appuser

ENV APP_ENV=production \
    HTTP_PORT=8080 \
    DATABASE_PATH=/data/lucid-logs.db \
    DATABASE_MIGRATIONS_PATH=/app/db/migrations \
    STATIC_ENABLED=true

EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/ || exit 1

ENTRYPOINT ["/app/api"]
