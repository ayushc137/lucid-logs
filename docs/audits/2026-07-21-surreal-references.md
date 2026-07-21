# SurrealDB Schema & Runtime Reference Audit

**Date:** 2026-07-21
**Commit:** `926ccab` (base `c69a366`)
**Scope:** Read-only inventory of SurrealDB persistence surface in Lucid Logs.
**Purpose:** Input for the Turso/libSQL migration implementer (`t_ba6bafe6`).

---

## 1. Tables & Fields

### Core entity tables (from `db/schema.surql` — authoritative; `db/migrations/001_schema.surql` is a subset missing 4 analytics tables)

| Table | Schema | Key fields (beyond `created_at`/`updated_at`/`deleted_at`/`created_by`) |
|---|---|---|
| `users` | SCHEMAFULL | `email` (unique), `pass` (argon2), `is_admin` bool |
| `categories` | schemaless | `name`, `color`; unique `(created_by,name)` |
| `tasks` | schemaless | `title`, `journal`, `note`, `start_date`, `end_date`, `completed`, `priority`, `source`, `positives[]`, `negatives[]`, `emotion_id`, `inferred_emotion{}`, `quantity{value,unit_id}` |
| `goals` | schemaless | `title`, `description`, `icon`, `target{value,operator,unit_id,track_completed_only}`, `recurrence{frequency,period,active_days,before_time,after_time,grace_days}`, `status`, `priority`, `start_date`, `deadline`, `completed_at`, denormalized streaks |
| `activities` | schemaless | `title`, `icon`, `description`, `default_*` (duration/emotion_id/priority/completed), `quantity_*`, `default_impact`, `pinned`, `sort_order`, `use_count`, `last_used_at` |
| `units` | schemaless | `name`, `symbol`, `type`, `is_system` |
| `retrospectives` | schemaless | `retro_type`, `start_date`, `end_date`, `auto_summary{}` (nested mood/habits/tasks/goals/categories), `user_content{}`, `status`, `generated_at` |
| `timer_sessions` | schemaless | `created_by`, `status`, session/break data |
| `emotions` | SCHEMALESS | `name`, `emoji`, `quadrant`, `x`, `y`, `valence`, `arousal`, `dominance`, `intensity`, `certainty`, `social`, `description` |
| `goal_snapshots` | schemaless | `goal_id`, `created_by`, point-in-time state |
| `goal_logs` | schemaless | `in` (goal), `out` (snapshot), `created_by`, `event_type`, `changes`, `triggered_by_task_id` |
| `agg_daily` | schemaless | `user_id`, `date` unique |
| `goal_daily_stats` | SCHEMAFULL | `goal_id`, `date`, `created_by`, `daily_value`, `cumulative_value`, `contribution_count` |
| `goal_period_snapshots` | SCHEMAFULL | `goal_id`, `period_type`, `period_key`, `created_by` |
| `streak_history` | SCHEMAFULL | `goal_id`, `created_by`, `date`, `event` |

### Graph relation tables (6) — `DEFINE TABLE TYPE RELATION IN … OUT …`

| Relation | In → Out | Edge fields |
|---|---|---|
| `task_emotions` | tasks→emotions | `type` (primary/positive/negative), `text` |
| `task_goals` | tasks→goals | `impact_type`, `quantity_value`, `unit_id`, `is_milestone`, `milestone_label/order`, `notes`, `source` |
| `created_from_activity` | tasks→activities | `mode` |
| `activity_goals` | activities→goals | `auto_link_tasks`, `quantity_multiplier`, `default_quantity`, `default_impact` |
| `goal_children` | goals→goals | `order`, `required` |
| `in_category` | tasks\|goals\|activities→categories | (none beyond `created_at`) |

### Code-only table (not in any `.surql` schema — must be added to libSQL)

- `activity_logs` — `entity_type`, `entity_id`, `event`, `changes`, `entity_title`, `entity_icon`, `created_by`, `created_at`

---

## 2. Repository Surface (12 repos, ~87 methods, ~5,300 LOC of SurrealQL)

| Repo | LOC | Notes |
|---|---|---|
| tasks | 1297 | Heaviest SurrealQL: `->in_category.out.*`, `<-task_goals`, `FETCH`, `time::floor`, `math::sum`, `count()`, `string::lowercase CONTAINS`, `type::string(id)`, `RETURN ... AS`, dynamic `ORDER BY` |
| goals | 1319 | Deepest graph queries: nested `IF recurrence IS NOT NONE THEN ... ?? ...`, `time::floor/week/month/year`, `->goal_children`, `goal_daily_stats` UPSERT, `GROUP ALL` |
| goals/analytics.go | — | `goal_daily_stats` UPSERT with `time::now()` |
| analytics | 696 | `math::sum(duration::secs(end_date - start_date))`, `time::format(time::floor(...),"%Y-%m-%d")`, quadrant distribution |
| activities | 703 | RELATE for `activity_goals`, `created_from_activity`, `in_category`; `MERGE` partial updates |
| goallogs | 407 | `RELATE $goal->goal_logs->$snapshot CONTENT $content` (nullable out) |
| taskgoals | 332 | `RELATE $task->task_goals->$goal SET ...` |
| categories | 398 | Standard CRUD + RELATE |
| retrospectives | 368 | `models.RecordID`, `SurrealTime` |
| users | 219 | `crypto::argon2::generate` on password update |
| units | 288 | System + user units, seeded |
| emotions | 124 | Read-mostly, cache-backed |
| activitylogs | 252 | Code-only table; plain inserts |

---

## 3. Surreal-Only Patterns Requiring SQL Translation

- `models.RecordID` (string form `table:id`) in every DB-facing struct + `database.ToStringID()` / `MustRecordID()` / `RecordIDFromString()` / `NewRecordID()`.
- `database.SurrealTime` custom CBOR datetime unmarshaler (handles `[seconds, nanoseconds]` arrays).
- `crypto::argon2::generate` / `crypto::argon2::compare` → Go `golang.org/x/crypto/argon2`.
- `time::now()`, `time::floor(d, 1d)`, `time::week()`, `time::month()`, `time::year()`, `time::format()` → SQLite `strftime`/`date`/Go-side computation.
- `math::sum()`, `count()`, `GROUP ALL`, `GROUP BY` → SQL aggregates.
- `string::lowercase(...) CONTAINS`, `string::is_email()`, `type::string(id)` → `LIKE`/Go validation/ explicit casts.
- `IS NONE` / `IS NOT NONE` → `IS NULL`.
- `MERGE` partial updates → `UPDATE ... SET` with COALESCE.
- `RELATE ... SET/CONTENT` → `INSERT ... ON CONFLICT DO UPDATE`.
- `UPSERT ... WHERE` → `INSERT ... ON CONFLICT(...) DO UPDATE`.
- `SELECT *, (subquery)[0] as field` → JOINs or multi-query.
- `RETURN (SELECT count() ...)[0].count OR 0` → `SELECT COUNT(*)`.
- Graph `->` / `<-` traversals → explicit JOINs on edge tables.

---

## 4. Runtime / Config / Deploy / Docs Assets Requiring Changes

### Go source (must rewrite)

- `apps/go_backend/internal/shared/database/database.go` (660 lines) — entire Surreal SDK wrapper.
- `apps/go_backend/internal/shared/database/surreal_time.go` — delete.
- `apps/go_backend/internal/shared/database/database_test.go` — **already targets libSQL** (`db.SQL()`, named params, `schema_migrations`) — keep as spec.
- `apps/go_backend/internal/features/*/repository.go` (12 files).
- `apps/go_backend/internal/features/*/models.go` — strip `models.RecordID` from `*DB` structs.
- `apps/go_backend/internal/features/auth/service.go` — argon2 + SurrealQL.
- `apps/go_backend/internal/features/goals/analytics.go` — `goal_daily_stats` UPSERT.
- `apps/go_backend/internal/bootstrap/admin.go` — SurrealQL CREATE.
- `apps/go_backend/internal/shared/middleware/auth.go` — `AuthenticatedUser.Namespace/.Database` fields; `users:` prefix normalization (L143) must stay.
- `apps/go_backend/internal/config/config.go` — **already migrated** to `DatabaseConfig{Path, URL, AuthToken, MigrationsPath}`.
- `apps/go_backend/internal/config/config_test.go` — **already Turso-targeting** — keep.
- `apps/go_backend/cmd/api/main.go` L83-90 — **COMPILE BREAK**: calls removed `cfg.Database.WebSocketURL()/.Namespace/.Database/.User/.Password`.
- `apps/go_backend/cmd/seed/main.go` (2715 lines) — 22 `database.*` calls + RELATE.
- `apps/go_backend/go.mod` / `go.sum` — remove `github.com/surrealdb/surrealdb.go v1.3.0` + transitive (`fxamacker/cbor`, `gorilla/websocket`, `gofrs/uuid`, `quic-go/*`).

### Schema / migrations (replace)

- `db/schema.surql` (211 lines) — authoritative source; archive then delete.
- `db/migrations/001_schema.surql` (171 lines) — subset; archive then delete.
- `db/migrations/002_seed_emotions.surql` — referenced by Taskfile/deploy.
- `db/scripts/backfill_daily_stats.surql` — SurrealQL with `LET`/`FOR`/`time::floor`/`math::sum`.
- `db/migrations/README.md` — SurrealDB CLI instructions.

### Deploy / infra

- `deploy/docker-compose.yml` — `surrealdb` service, `surrealdb_data` volume, `DB_HOST/DB_PORT/DB_SCHEMA_PATH` env, `depends_on: surrealdb`.
- `deploy/docker-compose.prod.yml` — same + `/home/enduser/configs/lucid-logs/surrealdb` mount.
- `.devcontainer/Dockerfile` L49-50 — `install.surrealdb.com`.
- `.devcontainer/docker-compose.yml` — `surrealdb` service + DB_* env.
- `.gitea/workflows/deploy.yml` — `mkdir surrealdb`, DB_* env, `/surreal sql`, `/surreal is-ready`.
- `Taskfile.yml` — `db:up`/`db:schema`/`db:down`/`db:reset` use Surreal HTTP `/sql` endpoint.
- `apps/go_backend/Makefile` — `db-start`/`db-start-file` (`surreal start ...`), setup hint.
- `env.example` — `DB_HOST/DB_PORT/DB_USER/DB_PASS/DB_NAMESPACE/DB_DATABASE/HOST_DB_PORT` block.
- `apps/frontend/Dockerfile` / `apps/go_backend/Dockerfile` — verify (likely no changes).

### Docs (rewrite or annotate as historical)

- `CONTRIBUTING.md` (511 lines) — extensive SurrealDB sections, DB_* env.
- `apps/go_backend/README.md` (370 lines) — "SurrealDB SDK" feature list, env table.
- `docs/DATA_MODEL.md` (621 lines) — SurrealDB design discussion.
- `docs/planning.md` (5000+ lines) — deeply Surreal-centric.
- `docs/NEXT_GEN_STACK.md`, `docs/review/*` — incidental.
- `history.txt` — historical changelog mentions.

### Frontend

- `apps/frontend/src/lib/api/auth.ts` — comment "User ID string like `users:xyz`" (**preserve** as ID format contract).
- No other frontend files hardcode RecordIDs; IDs flow as opaque strings.

---

## 5. Migration Risks (high-value)

1. **`table:id` string IDs are API + JWT contract.** Middleware normalizes `users:` prefix (L143 `auth.go`); frontend expects `users:xyz`. Preserve string PKs.
2. **Nested schemaless objects** (`target`, `recurrence`, `auto_summary`, `user_content`, `preferences`, `inferred_emotion`, `positives[]`, `negatives[]`) → JSON text columns.
3. **Graph traversals** → explicit JOINs; `in`/`out` edge columns become typed FK columns.
4. **Multi-step aggregate updates** (goal stats + daily_stats + streak + snapshot + log) — one SQL transaction.
5. **`activity_logs` table is code-only** — not in `.surql`; must be created.
6. **`001_schema.surql` missing 4 analytics tables** that `schema.surql` has — use `schema.surql` as authoritative.
7. **Scheduler is defined but not started** in `cmd/api/main.go`.
8. **Seed reset** uses `WHERE created_by = $user OR true` + swallows errors — replace with scoped transactional deletes.
9. **Config half-migrated** — `config.go`/`config_test.go` are Turso-native; `main.go` still references Surreal fields → guaranteed compile failure.
10. **Existing partial work to build on:**
    - `database_test.go` already targets libSQL API.
    - `docs/plans/2026-07-21-turso-migration.md` is a complete migration plan with full schema mapping.
    - `config.go` + `config_test.go` already done.

---

## 6. Tally for Implementer

- 12 repositories (~5,300 LOC SurrealQL)
- 1 DB wrapper (660 lines) + 1 helper file (`surreal_time.go`)
- 1 seeder (2,715 lines)
- 4 `.surql` files + 1 README
- 1 devcontainer Dockerfile, 3 compose/workflow files
- 2 task runners (Taskfile + Makefile)
- 1 env template
- ~7 doc files
- 1 Go dependency (`surrealdb.go`) + transitive
- Frontend: no ID-contract changes needed
