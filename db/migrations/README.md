# Database Migrations

This directory contains versioned database migrations for the Daily Journal app.

## Architecture: Record Links

We use SurrealDB's **direct record links** for relationships:

```surql
-- Task stores a link to category
task.category = categories:work123

-- Query with FETCH hydrates the linked record
SELECT *, category.* FROM tasks FETCH category
```

### Why Record Links?

| Feature | Graph Edges | Record Links ✅ |
|---------|-------------|-----------------|
| Simplicity | Complex | Simple field |
| One-to-Many | Works | ✅ Perfect fit |
| Query syntax | `->edge->` | Dot notation |
| FETCH support | Yes | ✅ Yes |
| Performance | Good | ✅ Excellent |
| Schema clarity | Hidden in edges | ✅ Visible in record |

### Query Examples

```sql
-- Get task with its category populated
SELECT *, category.* FROM tasks:id FETCH category

-- Get all tasks with categories for a user
SELECT *, category.* FROM tasks WHERE created_by = $user FETCH category

-- Filter tasks by category
SELECT * FROM tasks WHERE category = categories:work123

-- Get category with task count (reverse lookup)
SELECT *, count(<-tasks) AS task_count FROM categories
```

## Permissions Model

Tables use `PERMISSIONS FULL` because:
- The Axum service runs as a root/service account
- Multi-tenancy is enforced in Rust via `created_by` filters
- This avoids `$auth.id` mismatch issues

If per-request auth is needed later, issue Surreal tokens per request
instead of using a shared root session.

## Current Migrations

| Version | Name | Description |
|---------|------|-------------|
| 000 | init | Migration tracking table |
| 001 | auth | Authentication access & users |
| 002 | categories | User-owned categories |
| 003 | tasks | Tasks table with category record link |
| 004 | relax_permissions | Service account permissions (FULL) |
| 005 | composite_indexes | Optimized composite indexes & DB functions |

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

## Query Patterns

All queries are centralized in `src/shared/db/queries.rs` for:
- **Maintainability**: Single source of truth
- **Consistency**: Reuse common patterns
- **Optimization**: Easy to review and tune

```rust
// Use centralized queries
use crate::shared::db::task_queries;

db.query(task_queries::LIST_BY_USER)
    .bind(("user", user_id))
    .bind(("limit", 25))
    .bind(("offset", 0))
    .await?;
```

## Adding New Tables

### Step 1: Create migration file

```bash
# Create new migration
touch db/migrations/00N_my_feature.surql
```

### Step 2: Define the table

```sql
-- db/migrations/00N_my_feature.surql
-- Migration: 00N_my_feature
-- Description: Add my_feature table

DEFINE TABLE IF NOT EXISTS my_feature PERMISSIONS FULL;

-- Indexes for performance
DEFINE INDEX IF NOT EXISTS idx_my_feature_user 
  ON TABLE my_feature COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_my_feature_user_active 
  ON TABLE my_feature COLUMNS created_by, deleted_at;
```

### Step 3: Add record links (if needed)

```sql
-- Task links to my_feature
-- In your repository, store as: my_feature = my_feature:abc123

-- Composite index on the link
DEFINE INDEX IF NOT EXISTS idx_tasks_my_feature 
  ON TABLE tasks COLUMNS my_feature;
```

### Step 4: Update consolidated schema

Add the same definitions to `db/schema.surql` for reference.

## Database Functions

SurrealDB functions run server-side for better performance:

```sql
-- Define a reusable function
DEFINE FUNCTION IF NOT EXISTS fn::task::count_for_user($user_id: string) {
    RETURN (SELECT count() FROM tasks 
            WHERE created_by = $user_id AND deleted_at = NONE 
            GROUP ALL)[0].count OR 0
};

-- Use in queries
RETURN fn::task::count_for_user($user);
```

## Type-Safe IDs

Use type-safe record ID wrappers in Rust:

```rust
use crate::shared::{TaskId, CategoryId};

// Type-safe creation
let task_id = TaskId::new("abc123");
let category_id = CategoryId::new("categories:work");

// Correct format guaranteed
assert_eq!(task_id.full_id(), "tasks:abc123");
assert_eq!(category_id.full_id(), "categories:work");
```

## Migration Best Practices

### DO:
- ✅ Create new migrations for schema changes
- ✅ Use record links for one-to-many relationships
- ✅ Add composite indexes for common query patterns
- ✅ Use `DEFINE ... IF NOT EXISTS` for idempotency
- ✅ Test on fresh database before deploying
- ✅ Centralize queries in `shared/db/queries.rs`

### DON'T:
- ❌ Modify applied migrations
- ❌ Delete migration files
- ❌ Skip version numbers
- ❌ Use format!() for table names in queries

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
