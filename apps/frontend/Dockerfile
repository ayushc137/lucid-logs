# Lucid Logs SvelteKit Frontend - Dockerfile
# ============================================
# Multi-stage build: install → build → serve

# Build stage
FROM node:22-alpine AS builder

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .

# Build for production (node adapter)
ENV VITE_DEV_AUTH_BYPASS=false
RUN pnpm build

# Runtime stage
FROM node:22-alpine

RUN adduser -D -g '' appuser
WORKDIR /app

COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./
COPY --from=builder /app/node_modules ./node_modules

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 3000

ENV PORT=3000
ENV HOST=0.0.0.0
ENV ORIGIN=http://lucid-logs

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

CMD ["node", "build"]
