# Database Migrations

Simple migration files for SurrealDB. All migrations are idempotent (safe to run multiple times).

## Files

| File | Description |
|------|-------------|
| 001_core.surql | Users, categories, tasks tables + indexes |
| 002_task_emotions.surql | Edge table for emotion analytics |

## Run Migrations

```bash
# Run all migrations (same namespace/database as Go app)
cat db/migrations/*.surql | surreal sql \
  --endpoint http://localhost:8000 \
  --username <username> --password <pass> \
  --namespace daily_journal --database core \
  --hide-welcome

# Or run specific migration
cat db/migrations/001_core.surql | surreal sql \
  --endpoint http://localhost:8000 \
  --username <username> --password <pass> \
  --namespace daily_journal --database core \
  --hide-welcome
```

## Architecture

**Schemaless Tables**: We only define indexes for performance - data itself is flexible.

**Record Links**: Tasks link to categories via `task.category = categories:abc123`

**PERMISSIONS FULL**: The Go service handles authorization in code via `created_by` filters.

## Query Examples

```sql
-- Task with category
SELECT *, category.* FROM tasks:id FETCH category

-- Task's emotions
SELECT ->task_emotions.* FROM tasks:abc

-- All tasks with emotion E16
SELECT <-task_emotions<-tasks.* FROM "E16"

-- Emotion frequency
SELECT out as emotion, count() FROM task_emotions 
WHERE in.created_by = $user GROUP BY out
```

