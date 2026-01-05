# Database Migrations

This directory contains two migration files for Lucid Logs.

## Files

| File | Description |
|------|-------------|
| `001_schema.surql` | Complete schema: all tables, indexes, and relations |
| `002_seed_emotions.surql` | Seed 100 emotions reference data |

## How to Apply

### Apply all migrations:
```bash
surreal sql -e http://localhost:8000 -u root -p root -ns lucid -db logs < db/migrations/001_schema.surql
surreal sql -e http://localhost:8000 -u root -p root -ns lucid -db logs < db/migrations/002_seed_emotions.surql
```

### Or use the consolidated schema file:
```bash
surreal sql -e http://localhost:8000 -u root -p root -ns lucid -db logs < db/schema.surql
```

## Schema Design

### Schemaless + Essential Indexes
- Tables auto-create fields on insert (no schema definitions)
- Only define tables for permissions and indexes
- TYPE RELATION for graph edges (provides in/out validation)

### Denormalized Fields (Materialized View Pattern)
Pre-computed values stored on records, updated on write:

| Table | Fields | When Updated |
|-------|--------|--------------|
| `goals` | `current_streak`, `longest_streak`, `last_completed_date` | Goal entry marked met |

### Graph Relations

| Relation | Type | Notes |
|----------|------|-------|
| `task_emotions` | Explicit | Has type validation |
| `task_goals` | Explicit | Has impact/milestone fields |
| `goal_logs` | Explicit | Event history |
| `in_category` | Auto | Created on RELATE |
| `goal_children` | Auto | Created on RELATE |
| `created_from` | Auto | Created on RELATE |
| `template_goals` | Auto | Created on RELATE |

## Quick Reference

```sql
-- Goal with linked tasks
SELECT *, 
  (SELECT * FROM task_goals WHERE out = $parent.id) as linked_tasks 
FROM goals:abc

-- Task with category
SELECT *,
  (SELECT out FROM in_category WHERE in = $parent.id)[0].out as category
FROM tasks:xyz

-- All tasks for a goal
SELECT <-task_goals<-tasks.* FROM goals:abc
```
