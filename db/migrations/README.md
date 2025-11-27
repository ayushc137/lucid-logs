# Database Migrations

This directory contains versioned database migrations for the Daily Journal app.

## Architecture: Graph-First Design

We use SurrealDB's **graph relations** for flexible relationships:

```
task -[in_category]-> category
```

### Why Graph Relations?

| Feature | Foreign Keys | Graph Relations ✅ |
|---------|-------------|-------------------|
| One-to-Many | ✅ | ✅ |
| Many-to-Many | Needs junction table | ✅ Native |
| Relationship metadata | ❌ | ✅ (assigned_at, assigned_by) |
| Bidirectional queries | Manual | ✅ `<->` operator |
| Complex traversals | Joins | ✅ `->edge->node` |
| Analytics | GROUP BY | ✅ Graph aggregations |

### Query Examples

```sql
-- Get task with its category
SELECT *, ->in_category->categories[0] AS category FROM tasks:id

-- Get all tasks in a category (reverse traversal)
SELECT <-in_category<-tasks FROM categories:work

-- Find tasks by category name
SELECT <-in_category<-tasks FROM categories WHERE name = "Work"

-- Analytics: tasks per category
SELECT out.name AS category, count() AS tasks 
FROM in_category 
WHERE in.created_by = $user 
GROUP BY out
```

## Current Migrations

| Version | Name | Description |
|---------|------|-------------|
| 000 | init | Migration tracking table |
| 001 | auth | Authentication scope & users |
| 002 | categories | User-owned categories |
| 003 | tasks | Tasks table (no foreign keys) |
| 004 | task_relations | Graph edges (in_category) |

## Commands

```bash
# Apply pending migrations
task rust:migrate

# Show migration status
task rust:migrate:status

# Create new migration
task rust:migrate:new -- add_field

# Preview migrations (dry run)
task rust:migrate:dry-run

# Validate migration checksums
task rust:migrate:validate

# Reset database (DESTRUCTIVE!)
task rust:schema:reset
```

## Adding New Relations

### Step 1: Create the node table (if needed)

```sql
-- Example: Adding tags
DEFINE TABLE tags SCHEMAFULL
  PERMISSIONS ...;

DEFINE FIELD name ON tags TYPE string;
DEFINE FIELD color ON tags TYPE option<string>;
DEFINE FIELD created_by ON tags TYPE string;
```

### Step 2: Create the edge table

```sql
-- Edge: task -[has_tag]-> tag (many-to-many)
DEFINE TABLE has_tag SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR in.created_by = $auth.id
    ...;

DEFINE FIELD in ON has_tag TYPE record<tasks>;
DEFINE FIELD out ON has_tag TYPE record<tags>;
DEFINE FIELD added_at ON has_tag TYPE datetime VALUE time::now();

-- For many-to-many: unique on (in, out) pair
DEFINE INDEX idx_has_tag_unique ON TABLE has_tag COLUMNS in, out UNIQUE;
```

### Step 3: Use RELATE in code

```sql
-- Create relation
RELATE tasks:123 -> has_tag -> tags:important;

-- Query with traversal
SELECT *, ->has_tag->tags AS tags FROM tasks:123;

-- Remove relation
DELETE has_tag WHERE in = tasks:123 AND out = tags:important;
```

## Migration Best Practices

### DO:
- ✅ Create new migrations for schema changes
- ✅ Use graph relations for relationships
- ✅ Add metadata to edges (timestamps, who created)
- ✅ Test on fresh database before deploying
- ✅ Use `DEFINE ... IF NOT EXISTS` for idempotency

### DON'T:
- ❌ Modify applied migrations
- ❌ Use foreign key fields for relationships
- ❌ Delete migration files
- ❌ Skip version numbers

## Development vs Production

### Development
- Edit `db/schema.surql` for quick iteration
- Auto-applies on server startup with hot-reload
- Run `task rust:schema:apply` to manually apply

### Production
- Use versioned migrations only
- Run `task rust:migrate` before deploying
- Validate with `task rust:migrate:validate`
- Back up data before migrating
