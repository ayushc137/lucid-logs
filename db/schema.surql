-- =============================================================================
-- DAILY JOURNAL APP - Database Schema (SurrealDB)
-- =============================================================================
-- 
-- This file is the CONSOLIDATED SCHEMA for development and reference.
-- 
-- ARCHITECTURE: Schemaless with Record Links
-- ==========================================
-- We use SurrealDB's schemaless tables with direct record links for relations.
-- This is simpler than graph edges and works well for one-to-many.
--
-- Pattern: task.category = categories:work123
-- Query:   SELECT *, category.* FROM tasks FETCH category
--
-- Benefits:
--   - Simpler than graph edges
--   - Direct field access with dot notation
--   - FETCH hydrates linked records automatically
--   - Easy to understand and debug
--
-- PERMISSIONS: Service Account Model
-- ===================================
-- Tables use PERMISSIONS FULL because:
--   - The Axum service runs as root/service account
--   - Multi-tenancy is enforced in Rust via created_by filters
--   - This avoids $auth.id mismatch issues with service accounts
--
-- If you need per-request auth later, issue Surreal tokens per request
-- instead of using a shared root session.
-- 
-- MIGRATION FILES: db/migrations/
-- ================================
-- 000_init.surql       - Migration tracking
-- 001_auth.surql       - Authentication
-- 002_categories.surql - Categories table
-- 003_tasks.surql      - Tasks table with category link
-- 004_relax_permissions.surql - Service account permissions
--
-- =============================================================================

-- -----------------------------------------------------------------------------
-- MIGRATION TRACKING (000_init)
-- -----------------------------------------------------------------------------
DEFINE TABLE IF NOT EXISTS _migrations SCHEMAFULL;
DEFINE FIELD IF NOT EXISTS version ON _migrations TYPE int;
DEFINE FIELD IF NOT EXISTS name ON _migrations TYPE string;
DEFINE FIELD IF NOT EXISTS applied_at ON _migrations TYPE datetime VALUE time::now();
DEFINE FIELD IF NOT EXISTS checksum ON _migrations TYPE option<string>;
DEFINE INDEX IF NOT EXISTS idx_migrations_version ON TABLE _migrations COLUMNS version UNIQUE;

-- -----------------------------------------------------------------------------
-- AUTHENTICATION (001_auth)
-- -----------------------------------------------------------------------------
DEFINE ACCESS IF NOT EXISTS account ON DATABASE TYPE RECORD 
  SIGNUP ( CREATE users SET email = $user, pass = crypto::argon2::generate($pass) ) 
  SIGNIN ( SELECT * FROM users WHERE email = $user AND crypto::argon2::compare(pass, $pass) ) 
  WITH JWT ALGORITHM HS512 KEY "${JWT_SECRET}" 
  DURATION FOR TOKEN 1h, FOR SESSION 1d;

DEFINE TABLE IF NOT EXISTS users SCHEMAFULL;
DEFINE FIELD IF NOT EXISTS email ON users TYPE string ASSERT string::is::email($value);
DEFINE FIELD IF NOT EXISTS pass ON users TYPE string;
DEFINE FIELD IF NOT EXISTS is_admin ON users TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS created_at ON users TYPE datetime VALUE $before.created_at ?? time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON users TYPE datetime VALUE time::now();
DEFINE INDEX IF NOT EXISTS idx_users_email ON TABLE users COLUMNS email UNIQUE;

-- -----------------------------------------------------------------------------
-- CATEGORIES (002_categories) - SCHEMALESS + FULL PERMISSIONS
-- -----------------------------------------------------------------------------
-- Service account model: permissions are FULL, tenancy enforced in application
DEFINE TABLE IF NOT EXISTS categories PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS created_at ON categories TYPE datetime VALUE $before.created_at ?? time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON categories TYPE datetime VALUE time::now();

-- Indexes for performance
-- Composite index for the most common query pattern: user's categories by name
DEFINE INDEX IF NOT EXISTS idx_categories_user_name ON TABLE categories COLUMNS created_by, name UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_categories_user ON TABLE categories COLUMNS created_by;
-- Composite index for active records lookup (excludes soft-deleted)
DEFINE INDEX IF NOT EXISTS idx_categories_user_active ON TABLE categories COLUMNS created_by, deleted_at;

-- -----------------------------------------------------------------------------
-- TASKS (003_tasks) - SCHEMALESS with category record link + FULL PERMISSIONS
-- -----------------------------------------------------------------------------
-- Service account model: permissions are FULL, tenancy enforced in application
-- category field holds a record link: categories:abc123
DEFINE TABLE IF NOT EXISTS tasks PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS created_at ON tasks TYPE datetime VALUE $before.created_at ?? time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON tasks TYPE datetime VALUE time::now();

-- Indexes for performance
-- Primary composite index: user's tasks ordered by date (most common query)
DEFINE INDEX IF NOT EXISTS idx_tasks_user_date ON TABLE tasks COLUMNS created_by, start_date;
-- User's active tasks (excludes soft-deleted)
DEFINE INDEX IF NOT EXISTS idx_tasks_user_active ON TABLE tasks COLUMNS created_by, deleted_at;
-- Filter by completion status
DEFINE INDEX IF NOT EXISTS idx_tasks_user_completed ON TABLE tasks COLUMNS created_by, completed;
-- Category relationship index
DEFINE INDEX IF NOT EXISTS idx_tasks_category ON TABLE tasks COLUMNS category;
-- Legacy indexes (kept for compatibility)
DEFINE INDEX IF NOT EXISTS idx_tasks_user ON TABLE tasks COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_tasks_time ON TABLE tasks COLUMNS start_date;
DEFINE INDEX IF NOT EXISTS idx_tasks_status ON TABLE tasks COLUMNS completed;

-- -----------------------------------------------------------------------------
-- DATABASE FUNCTIONS (Reusable Query Logic)
-- -----------------------------------------------------------------------------
-- 
-- Server-side functions for complex, reusable queries.
-- Only functions that are actively used by the Rust backend are defined here.
--
-- Benefits:
--   - Run server-side for better performance
--   - Single source of truth for business logic
--   - Reusable across different clients
--
-- Naming Convention:
--   fn::<table>::<action>  e.g., fn::task::with_category

-- Get a task with its category populated
-- Used by: TaskRepository::fetch_task_with_category()
-- Usage: RETURN fn::task::with_category(tasks:abc123)
DEFINE FUNCTION IF NOT EXISTS fn::task::with_category($id: record<tasks>) {
    RETURN SELECT *, category.* FROM $id FETCH category
};

-- Get task count for a user (excludes soft-deleted)
-- Used by: TaskRepository::find_by_user_paginated()
-- Usage: RETURN fn::task::count_for_user("user:xyz")
DEFINE FUNCTION IF NOT EXISTS fn::task::count_for_user($user_id: string) {
    RETURN (SELECT count() FROM tasks WHERE created_by = $user_id AND deleted_at = NONE GROUP ALL)[0].count OR 0
};

-- Get category count for a user (excludes soft-deleted)
-- Used by: CategoryRepository::find_by_user_paginated()
-- Usage: RETURN fn::category::count_for_user("user:xyz")
DEFINE FUNCTION IF NOT EXISTS fn::category::count_for_user($user_id: string) {
    RETURN (SELECT count() FROM categories WHERE created_by = $user_id AND deleted_at = NONE GROUP ALL)[0].count OR 0
};

-- -----------------------------------------------------------------------------
-- QUERY EXAMPLES
-- -----------------------------------------------------------------------------
-- 
-- Get task with category populated:
--   SELECT *, category.* FROM tasks:id FETCH category
--   -- OR using function:
--   RETURN fn::task::with_category(tasks:id)
--
-- Get all tasks with categories:
--   SELECT *, category.* FROM tasks WHERE created_by = $user FETCH category
--
-- Filter by category:
--   SELECT * FROM tasks WHERE category = categories:work123
--
-- Get category with task count:
--   SELECT *, count(<-tasks) AS task_count FROM categories
--
-- -----------------------------------------------------------------------------

-- -----------------------------------------------------------------------------
-- FUTURE TABLES (Schemaless, record links)
-- -----------------------------------------------------------------------------

-- Tags (many-to-many via array of record links)
-- DEFINE TABLE IF NOT EXISTS tags PERMISSIONS FULL;
-- Task would have: tags = [tags:tag1, tags:tag2]
-- Query: SELECT *, tags.* FROM tasks FETCH tags

-- Moods
-- DEFINE TABLE IF NOT EXISTS moods PERMISSIONS FULL;
-- Task would have: mood = moods:happy
-- Query: SELECT *, mood.* FROM tasks FETCH mood
