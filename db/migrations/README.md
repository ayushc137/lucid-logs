# Database Migrations

Versioned, plain-SQL migrations for the Turso/libSQL database (SQLite dialect).
They are applied automatically by the Go backend on startup
(`internal/shared/database/database.go`, `applyMigrations`), in lexical order,
each inside its own transaction. Applied files are recorded with a SHA-256
checksum in `schema_migrations(version, checksum, applied_at)` — re-running is
a no-op, and editing an already-applied file is a hard error.

The legacy SurrealDB migrations (`*.surql`) are kept for historical reference
only and are not applied by the backend.

## Files

| File | Description |
|------|-------------|
| `001_initial.sql` | Complete schema: all entity, relation, history, and analytics tables + indexes |
| `002_seed_reference.sql` | Reference data: 100 emotions + 17 system units (idempotent upserts) |
| `003_scheduler.sql` | Scheduler jobs/state tables and demo seed/reset state |

## Table coverage

Entities: `users`, `categories`, `units`, `emotions`, `tasks`, `goals`,
`activities`, `retrospectives`, `timer_sessions`.
Relations: `task_emotions`, `task_goals`, `created_from_activity`,
`activity_goals`, `goal_children`.
History/analytics: `activity_logs`, `goal_logs`, `goal_snapshots`,
`goal_daily_stats`, `goal_period_snapshots`, `streak_history`, `agg_daily`.
Infrastructure: `scheduler_jobs`, `scheduler_state`, `demo_seed_state`,
`schema_migrations`.

## Conventions

- Primary keys are the API `table:value` strings (e.g. `tasks:abc123`).
- Times are UTC RFC3339 text.
- Multi-tenant tables filter on `created_by`; soft delete via `deleted_at`.
- Nested/structured values are JSON text columns.
- Reference-data seeds use `INSERT ... ON CONFLICT DO UPDATE` so migrations
  are idempotent.

## Configuration

| Variable | Purpose |
|----------|---------|
| `LIBSQL_LOCAL_PATH` | Local database file (primary; `DATABASE_PATH` is the legacy alias) |
| `LIBSQL_URL` | Turso remote URL for sync (legacy alias `TURSO_DATABASE_URL`) |
| `LIBSQL_AUTH_TOKEN` | Turso auth token (legacy alias `TURSO_AUTH_TOKEN`) |
| `DATABASE_MIGRATIONS_PATH` | This directory (default `../../db/migrations`) |

## Verifying a fresh database

```bash
cd apps/go_backend
go test ./internal/shared/database/   # applies all migrations on temp DBs
```
