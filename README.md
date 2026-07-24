# Lucid Logs

> A self-hostable daily journal, task tracker, and goal-setting app — with AI-assisted retrospectives.

Lucid Logs is a single, dependency-free container: a **SvelteKit** frontend, a **Go (Gin)** API, and an **embedded libSQL/SQLite** database, all served from one binary. No external database, no reverse proxy, no microservices to babysit.

**Stack:** SvelteKit (static SPA) · Go (Gin) · libSQL/SQLite · Turso (optional sync)

---

## Features

- **Daily journal** — rich entries with mood/emotion tracking.
- **Tasks & goals** — hierarchical goals, tasks linked to goals, streaks, and progress stats.
- **Analytics** — activity heatmap, streaks, and per-metric dashboards.
- **AI retrospectives** — generate weekly/monthly retros and insights from your data (bring-your-own LLM key; works fully offline without one).
- **Categories & units** — custom categories and measurement units for whatever you track.
- **PWA** — installable, offline-capable single-page app.
- **Self-hosted & private** — your data lives in a single SQLite file you own.

---

## Quickstart (Docker)

```bash
docker run -d --name lucid-logs \
  -p 8080:8080 \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
  -e ADMIN_SEED=true \
  -e ADMIN_USERNAME=you@example.com \
  -e ADMIN_PASSWORD=change-me \
  -v lucid-data:/data \
  lucid-logs:latest
```

Open **http://localhost:8080** and log in with the admin credentials above. The admin account is created only when the database has no users yet, so it's safe to leave those variables set.

### Docker Compose

```bash
cd deploy
cp .env.example .env    # fill in JWT_SECRET, ADMIN_USERNAME, ADMIN_PASSWORD
docker compose up -d
```

---

## Configuration

All configuration is via environment variables.

| Variable | Default | Description |
|---|---|---|
| `HTTP_PORT` | `8080` | Port the app listens on (inside the container). |
| `JWT_SECRET` | — | **Required in production.** `openssl rand -hex 32` |
| `ADMIN_SEED` | `false` (prod) | Set `true` to create the first admin on an empty DB. |
| `ADMIN_USERNAME` | — | Admin email (used with `ADMIN_SEED`). |
| `ADMIN_PASSWORD` | — | Admin password (used with `ADMIN_SEED`). |
| `DATABASE_PATH` | `/data/lucid-logs.db` | SQLite file location (persist the `/data` volume). |
| `DATABASE_MIGRATIONS_PATH` | `/app/db/migrations` | SQL migrations directory. |
| `STATIC_ENABLED` | `true` | Serve the embedded SPA frontend. |
| `CORS_ALLOWED_ORIGINS` | — | Comma-separated origins (only needed for split-host dev). |
| `LLM_PROVIDER` / `LLM_BASE_URL` / `LLM_MODEL` / `LLM_API_KEY` | — | Optional: pre-fill the AI settings screen (bring-your-own-key). |

### Data persistence

All state lives in `/data` inside the container (a single SQLite file + WAL). Mount a volume to keep it:

```bash
-v lucid-data:/data
```

### Optional Turso sync

Lucid Logs uses embedded libSQL by default. If you'd like a managed replica or off-device backup, point it at [Turso](https://turso.tech) with `LIBSQL_URL` + `LIBSQL_AUTH_TOKEN` (see `env.example`).

---

## Development

The repo is a monorepo:

```
apps/
  frontend/    # SvelteKit SPA (adapter-static in all-in-one builds)
  go_backend/  # Go (Gin) API, embeds the frontend via go:embed
db/
  migrations/  # SQL migrations (golang-migrate)
deploy/        # all-in-one Docker Compose
```

### Run locally (hot reload)

Requires Go 1.24+ and pnpm.

```bash
# Backend (from apps/go_backend)
go run ./cmd/api

# Frontend (from apps/frontend, another terminal)
pnpm install && pnpm dev
```

The frontend dev server proxies API calls; set `VITE_API_URL` if running the backend on a different origin.

### Build the all-in-one image

```bash
docker build -t lucid-logs .
```

This multi-stage build compiles the SPA with `VITE_SPA=1` (adapter-static), embeds it into the Go binary via `go:embed`, and produces a minimal Alpine runtime image.

---

## License

[MIT](./LICENSE)
