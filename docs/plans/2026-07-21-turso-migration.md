# Turso/libSQL migration plan

Date: 2026-07-21
Branch: `migration/turso-libsql`
Base: `c69a366`

## Decision

Use `database/sql` with `turso.tech/database/tursogo` (current official Turso recommendation) for the local database and optional local-plus-cloud sync. The driver uses prebuilt native libraries through purego and does not require CGO. Local development opens `DATABASE_PATH` (default `./data/lucid-logs.db`) with driver name `turso`. When `TURSO_DATABASE_URL` is set, configure `NewTursoSyncDb` with `DATABASE_PATH`, the remote URL, and `TURSO_AUTH_TOKEN`; all queries still run against the local file and synchronization is explicit. A future stateless deployment may use `github.com/tursodatabase/libsql-client-go/libsql` directly, but the stateful backend should prefer the lower-latency local replica.

Sources checked:

- https://docs.turso.tech/sdk/go/quickstart — recommends `tursogo`, `database/sql`, purego/no-CGO; documents `NewTursoSyncDb` for local/cloud sync.
- https://github.com/tursodatabase/libsql-client-go — pure-Go over-the-wire driver for direct remote access.

## Compatibility invariants

- Gin handlers, service interfaces, routes, request/response JSON, pagination semantics, JWT claims, and string IDs stay unchanged.
- API IDs retain the existing `table:value` shape. Relational primary keys store the complete string, so clients do not need an ID migration.
- All tenant-owned reads and writes include `created_by = ?`; soft-deletable entities also include `deleted_at IS NULL`.
- Times are UTC RFC3339 text. Scanner helpers accept RFC3339/RFC3339Nano and normalize to UTC.
- Structured values that do not benefit from relational querying remain JSON text. Relationships become relational tables with foreign keys.
- Foreign keys are enabled for every connection (`PRAGMA foreign_keys = ON`). WAL and a busy timeout are used for the local file.

## Schema mapping

### Core tables

`users`

- `id TEXT PRIMARY KEY` (`users:<uuid>`)
- `email TEXT NOT NULL COLLATE NOCASE UNIQUE`
- `pass TEXT NOT NULL` (Argon2id hash generated in Go)
- `is_admin INTEGER NOT NULL DEFAULT 0 CHECK (is_admin IN (0,1))`
- `preferences_json TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(preferences_json))`
- `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`

`categories`

- `id TEXT PRIMARY KEY`, `created_by TEXT NOT NULL REFERENCES users(id)`
- `name TEXT NOT NULL`, `color TEXT NOT NULL`
- `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, `deleted_at TEXT`
- partial unique index on `(created_by, name COLLATE NOCASE)` where `deleted_at IS NULL`

`units`

- `id TEXT PRIMARY KEY`, `created_by TEXT REFERENCES users(id)` (NULL for system units)
- `name TEXT NOT NULL`, `symbol TEXT NOT NULL`, `type TEXT NOT NULL`
- `is_system INTEGER NOT NULL DEFAULT 0`
- `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, `deleted_at TEXT`

`emotions`

- `id TEXT PRIMARY KEY`, `name TEXT NOT NULL UNIQUE`
- `emoji TEXT`, `description TEXT`, `category TEXT`, `quadrant TEXT`
- `valence REAL NOT NULL`, `arousal REAL NOT NULL`, `intensity REAL`
- remaining display/alias metadata in typed columns where used by API, otherwise `metadata_json TEXT NOT NULL DEFAULT '{}'`

`tasks`

- `id TEXT PRIMARY KEY`, `created_by TEXT NOT NULL REFERENCES users(id)`
- content: `title TEXT NOT NULL`, `note TEXT`, `journal TEXT`
- schedule: `start_date TEXT NOT NULL`, `end_date TEXT`, `duration INTEGER`
- state: `completed INTEGER NOT NULL DEFAULT 0`, `completed_at TEXT`, `priority TEXT`, `status TEXT`
- organization/origin: `category_id TEXT REFERENCES categories(id)`, `activity_id TEXT REFERENCES activities(id)`, `activity_mode TEXT`
- emotions/reflection: `emotion_id TEXT REFERENCES emotions(id)`, `positives_json TEXT`, `negatives_json TEXT`, `inferred_emotion_json TEXT`
- quantity and remaining optional API structures: typed scalar columns where queried plus JSON text for opaque arrays/objects
- metadata: `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, `deleted_at TEXT`
- indexes on `(created_by,start_date)`, `(created_by,completed)`, `(created_by,end_date)`, `(created_by,priority)`, `(created_by,deleted_at)`

`goals`

- `id TEXT PRIMARY KEY`, `created_by TEXT NOT NULL REFERENCES users(id)`
- identity: `title TEXT NOT NULL`, `description TEXT`, `icon TEXT`, `color TEXT`
- state: `status TEXT NOT NULL`, `priority TEXT`, `category_id TEXT REFERENCES categories(id)`
- structured definitions: `target_json TEXT`, `recurrence_json TEXT`, `schedule_json TEXT`
- denormalized streaks: `current_streak INTEGER NOT NULL DEFAULT 0`, `longest_streak INTEGER NOT NULL DEFAULT 0`, `last_completed_at TEXT`
- date/metadata: `start_date TEXT`, `end_date TEXT`, `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, `deleted_at TEXT`
- indexes on `(created_by,status)`, `(created_by,priority)`, `(created_by,deleted_at)`

`activities`

- `id TEXT PRIMARY KEY`, `created_by TEXT NOT NULL REFERENCES users(id)`
- identity: `name TEXT NOT NULL`, `description TEXT`, `icon TEXT`, `color TEXT`
- behavior: `mode TEXT NOT NULL`, `duration INTEGER`, `pinned INTEGER NOT NULL DEFAULT 0`
- inherited organization: `category_id TEXT REFERENCES categories(id)`, `priority TEXT`
- scheduling/timer defaults stored as typed queried scalars plus JSON for opaque options
- usage: `use_count INTEGER NOT NULL DEFAULT 0`, `last_used_at TEXT`
- metadata: `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, `deleted_at TEXT`

`activity_logs`

- `id TEXT PRIMARY KEY`, `activity_id TEXT NOT NULL REFERENCES activities(id)`, `created_by TEXT NOT NULL REFERENCES users(id)`
- `task_id TEXT REFERENCES tasks(id)`, `event_type TEXT NOT NULL`, `mode TEXT`
- `started_at TEXT`, `ended_at TEXT`, `duration INTEGER`, `quantity REAL`, `metadata_json TEXT`
- `created_at TEXT NOT NULL`

`retrospectives`

- `id TEXT PRIMARY KEY`, `created_by TEXT NOT NULL REFERENCES users(id)`
- `retro_type TEXT NOT NULL`, `start_date TEXT NOT NULL`, `end_date TEXT NOT NULL`
- response fields represented as JSON text without changing response JSON
- `status TEXT`, `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, `deleted_at TEXT`
- indexes on `(created_by,start_date)`, `(created_by,retro_type)`, `(created_by,deleted_at)`

### Relationship tables

`task_emotions`

- `id TEXT PRIMARY KEY`
- `task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE`
- `emotion_id TEXT NOT NULL REFERENCES emotions(id)`
- `type TEXT NOT NULL CHECK (type IN ('primary','positive','negative'))`, `text TEXT`, `created_at TEXT NOT NULL`
- unique `(task_id, emotion_id, type)`

`task_goals`

- `id TEXT PRIMARY KEY`
- `task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE`
- `goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`
- `impact_type TEXT NOT NULL CHECK (impact_type IN ('positive','negative','neutral'))`
- `quantity_value REAL`, `unit_id TEXT REFERENCES units(id)`, `is_milestone INTEGER NOT NULL DEFAULT 0`
- `milestone_label TEXT`, `milestone_order INTEGER`, `notes TEXT`, `source TEXT NOT NULL DEFAULT 'manual'`, `created_at TEXT NOT NULL`
- unique `(task_id, goal_id)` and indexes in both directions

`created_from_activity`

- `task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE`
- `activity_id TEXT NOT NULL REFERENCES activities(id)`, `mode TEXT NOT NULL DEFAULT 'instant'`, `created_at TEXT NOT NULL`

`activity_goals`

- `id TEXT PRIMARY KEY`, `activity_id TEXT NOT NULL REFERENCES activities(id) ON DELETE CASCADE`, `goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`
- `auto_link_tasks INTEGER NOT NULL DEFAULT 1`, `quantity_multiplier REAL NOT NULL DEFAULT 1`, `default_quantity REAL`, `default_impact TEXT NOT NULL DEFAULT 'positive'`, `created_at TEXT NOT NULL`
- unique `(activity_id, goal_id)`

`goal_children`

- `parent_goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`
- `child_goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`
- `sort_order INTEGER NOT NULL DEFAULT 0`, `required INTEGER NOT NULL DEFAULT 1`, `created_at TEXT NOT NULL`
- primary key `(parent_goal_id, child_goal_id)` and check parent != child

Category graph edges are intentionally collapsed into `category_id` foreign keys on tasks, goals, and activities because each entity has at most one active category. This preserves API semantics while enforcing ownership in repository transactions.

### Goal analytics/history

`goal_logs`

- `id TEXT PRIMARY KEY`, `goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`, `created_by TEXT NOT NULL REFERENCES users(id)`
- `event_type TEXT NOT NULL`, `changes_json TEXT`, `triggered_by_task_id TEXT REFERENCES tasks(id)`, `created_at TEXT NOT NULL`

`goal_snapshots`

- `id TEXT PRIMARY KEY`, `goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`, `created_by TEXT NOT NULL REFERENCES users(id)`
- point-in-time goal state as `snapshot_json TEXT NOT NULL`, `created_at TEXT NOT NULL`

`goal_daily_stats`

- `goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`, `date TEXT NOT NULL`, `created_by TEXT NOT NULL REFERENCES users(id)`
- `daily_value REAL NOT NULL DEFAULT 0`, `cumulative_value REAL NOT NULL DEFAULT 0`, `contribution_count INTEGER NOT NULL DEFAULT 0`
- `target_value REAL`, `streak_at_date INTEGER NOT NULL DEFAULT 0`, `status TEXT NOT NULL DEFAULT 'pending'`
- `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, primary key `(goal_id,date)`

`goal_period_snapshots`

- `id TEXT PRIMARY KEY`, `goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`, `created_by TEXT NOT NULL REFERENCES users(id)`
- `period_type TEXT NOT NULL`, `period_key TEXT NOT NULL`, `start_date TEXT NOT NULL`, `end_date TEXT NOT NULL`, `snapshot_json TEXT NOT NULL`
- unique `(goal_id,period_type,period_key)`

`streak_history`

- `id TEXT PRIMARY KEY`, `goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`, `created_by TEXT NOT NULL REFERENCES users(id)`
- `date TEXT NOT NULL`, `event TEXT NOT NULL`, `streak_value INTEGER`, `created_at TEXT NOT NULL`

`agg_daily`

- `user_id TEXT NOT NULL REFERENCES users(id)`, `date TEXT NOT NULL`, typed aggregate counters/values plus `metrics_json TEXT NOT NULL DEFAULT '{}'`
- primary key `(user_id,date)`

`timer_sessions`

- `id TEXT PRIMARY KEY`, `activity_id TEXT REFERENCES activities(id)`, `created_by TEXT NOT NULL REFERENCES users(id)`
- `status TEXT NOT NULL`, `started_at TEXT NOT NULL`, `ended_at TEXT`, timer counters/config JSON, `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`

## Repository operation mapping

- Surreal SDK `Select/Create/Merge/Delete` becomes parameterized `QueryRowContext`, `ExecContext`, and transactions.
- Surreal named variables become positional `?` parameters. Dynamic filters may concatenate only fixed SQL fragments; user values are always bound.
- `RELATE` becomes `INSERT ... ON CONFLICT ... DO UPDATE` into the join table.
- graph traversals become explicit joins.
- `NONE` becomes `NULL`; `IS NONE` becomes `IS NULL`.
- `time::now()` and record IDs are generated in Go.
- `crypto::argon2::*` is replaced by Go `golang.org/x/crypto/argon2` hashing/verification with a versioned encoded hash format.
- Surreal arrays/objects are marshalled with `encoding/json`; scanners unmarshal after row retrieval.
- multi-step writes (entity + category + emotions + goal links + stats) run in one SQL transaction.
- analytics use SQLite `COUNT`, `SUM`, `AVG`, `strftime`, conditional aggregates, and relational joins. Empty aggregates are normalized with `COALESCE` to preserve response types.

## Versioned migrations

- `db/migrations/001_initial.sql`: all core, relationship, history, analytics, and migration metadata tables; indexes and checks.
- `db/migrations/002_seed_reference.sql`: immutable emotions and system units using `INSERT ... ON CONFLICT DO UPDATE`.
- An embedded migration runner executes files in lexical order inside transactions and records checksums in `schema_migrations(version, checksum, applied_at)`. Changed checksums are an error.
- `task db:migrate` applies migrations; `task db:reset` deletes only the configured local development DB after an explicit environment guard, then migrates. Remote reset is refused.

## Seeder

The seed command calls the same repository/service/API paths used by the app. Stable seed IDs or natural-key upserts make repeated runs idempotent. `--reset` deletes only rows owned by the configured development admin, in a transaction, and is rejected outside development or against a remote-only target. Reference units/emotions are independently idempotent.

## Existing-data migration path

Do not connect to or mutate the running demo database during cutover. Provide two explicit offline commands:

1. `cmd/migrate-surreal export --input <Surreal JSON export> --output <neutral.json>` parses an operator-created export, converts record references to stable strings, validates every row, and writes a neutral versioned JSON bundle. It has no network client and therefore cannot modify the source.
2. `cmd/migrate-surreal import --input <neutral.json>` validates all foreign keys, imports in dependency order inside one libSQL transaction, and supports `--dry-run`. It aborts and rolls back on validation or constraint failure.

Demo-only data can be regenerated with the seeder, but this utility preserves a future production migration route.

## TDD vertical slices and evidence

Each slice must record the exact RED and GREEN command/result below before moving to the next:

1. database configuration selects local vs synced Turso and rejects incomplete remote credentials;
2. migrations apply once, preserve checksums, and enable FKs;
3. users/auth bootstrap and password verification;
4. categories, units, and emotions;
5. task CRUD/filtering plus category/emotion transactions;
6. goal CRUD, hierarchy, task-goal links, daily stats, and streak updates;
7. activities and activity logs;
8. retrospectives;
9. analytics parity;
10. idempotent seed/reset and neutral import validation;
11. HTTP smoke flow against a temporary local DB.

A RED is valid only when the test compiles and fails because behavior is missing. Baseline on 2026-07-21: `go test ./...` passed but reported `[no test files]` for every package. Frontend `pnpm check` and `pnpm build` passed with one pre-existing `GoalSearchModal.svelte` ARIA warning.

## Baseline status snapshot (2026-07-21, task t_eae4eef0)

After three prior implementation runs on the parent card, the tree state at the time this baseline task ran is:

- `go build ./...` — clean
- `go vet ./...` — clean
- `gofmt -l .` — clean (after a `gofmt -w .` pass during this task)
- `go test ./...` — 4 packages with tests, all passing:
  - `internal/config` (local/synced Turso config selection, credential rejection)
  - `internal/features/auth` (Argon2id hash/verify, register/login service against SQL)
  - `internal/shared/database` (migration runner, checksums, FK pragma, row scanning)
  - `internal/shared/recordid` (Surreal `table:id` string compatibility helpers)
- All other packages report `[no test files]` — per-repository tests are owned by the per-repo child tasks.
- Frontend was not re-run during this baseline task; last recorded state (above) had `pnpm check`/`pnpm build` green with one ARIA warning.

Committed branch state at this point:

- `926ccab` test: establish Turso migration plan and baseline (this plan + config/database tests)
- `e36faf3` feat: add Turso database core and SQL migrations (`tursogo` driver, `database/sql` core, `db/migrations/001_initial.sql`)
- `6c53f44` feat: port all repositories to database/sql libSQL (users/auth + RecordID compat + partial repository ports; commit message overstates — see remaining-surface inventory in the parent card comments)
- `wip` commit from this task preserving the mixed-tree partial work (remaining repository ports in progress across analytics, categories, goallogs, goals, retrospectives, taskgoals, tasks, units; bootstrap and seeder partial ports; Surreal SDK dependency removal)

No `.surql` files, Surreal schema assets, Docker services, or runtime/docs references have been deleted yet. The Surreal Go SDK module is removed from `go.mod` only where the ported code no longer imports it; remaining SurrealQL call sites inside repositories are inventoried in the parent card and assigned to per-repo child tasks.

## Cutover phases

1. Add tests and database/config/migration infrastructure without changing handlers.
2. Port repositories by dependency order: users/auth → reference tables → tasks → goals/links → activities/logs → retrospectives → analytics/history.
3. Port bootstrap/seeder and add neutral import/export.
4. Remove Surreal SDK, RecordID/time wrappers, SurrealQL and `.surql` runtime assets.
5. Replace environment variables with `DATABASE_PATH`, `TURSO_DATABASE_URL`, `TURSO_AUTH_TOKEN`, and sync policy settings. Remove the database container from Compose; mount the local DB directory for the backend.
6. Verify all static and runtime acceptance gates before deleting any old demo data.

## Final verification gates

- `gofmt`/`gofumpt` clean; `go vet ./...`; `go test -race ./...`.
- `pnpm check`; `pnpm build`.
- no runtime/dependency/config/deploy matches for `surreal`, `surql`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAMESPACE`, or `DB_DATABASE`; historical plan may mention the source system.
- start backend against a new temporary local DB; migrations and reference seeds run automatically.
- run seed twice and seed reset twice; row counts are stable.
- login as `admin@example.com` / `adminadmin`; exercise health, dashboard/analytics, task create/list/update/delete, goal create/list/link/progress, activities, and retrospectives.
- inspect `PRAGMA foreign_key_check` (zero rows) and `PRAGMA integrity_check` (`ok`).
- only after all gates pass may the deployment switch from the old demo database. Destruction of the old demo database is a separate, explicitly approved operation.
