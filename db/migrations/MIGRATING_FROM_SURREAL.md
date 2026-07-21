# Migrating Data from SurrealDB to libSQL/Turso

One-shot data migration from the legacy SurrealDB backend to the libSQL schema
produced by the migrations in this directory. The tool is an **offline
converter**: SurrealDB is never queried live — you produce a JSON export first,
then run the migrator against a fresh libSQL database file.

Tool: `apps/go_backend/cmd/migrate-surreal-to-libsql`

---

## 1. Export from SurrealDB

Start your SurrealDB instance (or connect to the existing one), then export
**every table** as JSON using the CLI's `--json` flag. The migrator accepts a
single "bundle" file; produce it with this shell loop:

```bash
#!/usr/bin/env bash
# Requires: surreal CLI + jq
NS=lucid DB=logs
CONN="ws://localhost:8000" USER=root PASS=root

TABLES="users categories units emotions goals activities tasks \
        retrospectives timer_sessions in_category task_emotions task_goals \
        created_from_activity activity_goals goal_children activity_logs \
        goal_snapshots goal_logs goal_daily_stats goal_period_snapshots \
        streak_history agg_daily"

echo '{"tables": {' > surreal-export.json
first=1
for t in $TABLES; do
  rows=$(surreal sql --json --conn "$CONN" --user "$USER" --pass "$PASS" \
         --ns "$NS" --db "$DB" "SELECT * FROM $t" | jq -c '.[0].result // []')
  [ "$first" = 0 ] && echo "," >> surreal-export.json
  first=0
  printf '"%s": %s' "$t" "$rows" >> surreal-export.json
done
echo '}}' >> surreal-export.json
```

The migrator also accepts these input shapes directly (useful for one-table
exports or debugging):

- Raw `surreal sql --json` output — `[{"status":"OK","result":[...]}]`
  (table inferred from record IDs)
- A bare JSON array of rows (table inferred from record IDs)
- Newline-delimited JSON (one row per line)
- `{"tables": {...}}` or top-level `{"tasks": [...], ...}` bundles

Note: `surreal export` (SurrealQL dump) is **not** accepted — the migrator
parses JSON only. The `surreal sql --json` route above is the supported path.

## 2. Run the migration

```bash
cd apps/go_backend

# 1. Dry run — parse + validate only, no writes
go run ./cmd/migrate-surreal-to-libsql \
  --input /path/to/surreal-export.json \
  --dry-run

# 2. Real import into a fresh database file
go run ./cmd/migrate-surreal-to-libsql \
  --input /path/to/surreal-export.json \
  --db /path/to/lucid.db \
  --migrations ../../db/migrations
```

The target database is created if missing and **all pending migrations are
applied automatically** before import. Always import into a fresh file (or one
previously created by this tool) — importing into a live database with
application data is not supported.

### Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--input` | (required) | SurrealDB JSON export path |
| `--db` | (required unless dry-run) | Target libSQL file path |
| `--migrations` | `db/migrations` | Migrations directory |
| `--dry-run` | false | Parse + validate only, no writes |
| `--limit N` | 0 | Import at most N rows per table (smoke tests) |
| `--tables a,b,c` | all | Import only the listed tables |
| `--v` | false | Verbose logging |

## 3. Verify

The tool prints two reports at the end:

1. **Import summary** — per-table source/inserted/skipped/failed counts.
   `skipped` = rows already present (safe re-runs). Non-zero `failed` aborts
   with exit code 1.
2. **Validation** — per-table source vs target row counts plus
   `PRAGMA foreign_key_check` output. All counts should read `OK` and FK
   integrity must show no violations.

Because all inserts are `INSERT OR IGNORE`, re-running the tool on the same
database is safe and idempotent (second run inserts 0 rows). To start over,
delete the database file and re-run.

## 4. Rollback

The import targets a **fresh** libSQL file. Rollback = delete the file:

```bash
rm /path/to/lucid.db /path/to/lucid.db-wal /path/to/lucid.db-shm
```

The SurrealDB instance is untouched throughout and remains the source of truth
until cutover. Do not decommission SurrealDB until the new backend has been
validated end-to-end against the migrated file.

## 5. Field mapping notes

| SurrealDB | libSQL | Notes |
|-----------|--------|-------|
| `tasks.quantity: {value, unit_id}` | `quantity_value`, `unit_id` | object unwrapped |
| `tasks.positives/negatives` (arrays) | JSON TEXT columns | preserved verbatim |
| `tasks.source`, `tasks.status` | `metadata.source`, `metadata.legacy_status` | no libSQL column; preserved in metadata JSON |
| `goals.streak: {current, longest, last_completed_date}` | flat columns | denormalized |
| `goals.target`/`recurrence` (objects) | JSON TEXT columns | preserved verbatim |
| `priority` int (1–3) | label | 1→`high`, 2→`medium`, 3→`low`; existing labels pass through |
| `in_category` edges | `category_id` on target row | edge table collapsed (per migration plan) |
| `goal_children.order` | `sort_order` | `order` is a reserved word in SurrealDB |
| `goal_logs` edge (`in`, `out`) | `goal_id`, `snapshot_id` | `out` may be null |
| `retrospectives.auto_summary` + `user_content` | merged into `responses` JSON | single JSON column in libSQL |
| `timer_sessions.break_time/total_active_time/pauses` | `counters` JSON | no direct columns |
| `activities.default_*`, `quantity_*` | packed into `default_task` JSON | activities schema was redesigned |
| record IDs (`tasks:abc123`) | preserved verbatim | API/JWT contract depends on the format |

Seeded tables (`emotions`, `units`) are pre-populated by migrations with
system rows; imported rows are merged in idempotently (existing IDs skipped).

## 6. Caveats

- **Passwords** are carried over as-is (`pass` hash). Users keep their
  existing credentials; no rehashing is performed.
- **Soft-deleted rows** (`deleted_at` set) are imported with their deletion
  timestamp intact — they stay invisible to the API but remain in the table.
- Rows with **dangling references** (e.g. a task whose `activity_id` was
  deleted in SurrealDB) will trip `foreign_key_check` in the validation
  report. Clean these up in SurrealDB before export, or fix the export JSON
  manually.
- The legacy **`in_category`** edges are applied via UPDATE after core
  entities import; edges pointing at non-existent rows are logged as warnings
  and skipped.
