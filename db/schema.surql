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
-- MIGRATION FILES: db/migrations/
-- ================================
-- 000_init.surql       - Migration tracking
-- 001_auth.surql       - Authentication
-- 002_categories.surql - Categories table
-- 003_tasks.surql      - Tasks table with category link
--
-- =============================================================================

-- -----------------------------------------------------------------------------
-- MIGRATION TRACKING (000_init)
-- -----------------------------------------------------------------------------
DEFINE TABLE _migrations SCHEMAFULL;
DEFINE FIELD version ON _migrations TYPE int;
DEFINE FIELD name ON _migrations TYPE string;
DEFINE FIELD applied_at ON _migrations TYPE datetime VALUE time::now();
DEFINE FIELD checksum ON _migrations TYPE option<string>;
DEFINE INDEX idx_migrations_version ON TABLE _migrations COLUMNS version UNIQUE;

-- -----------------------------------------------------------------------------
-- AUTHENTICATION (001_auth)
-- -----------------------------------------------------------------------------
DEFINE ACCESS account ON DATABASE TYPE RECORD 
  SIGNUP ( CREATE user SET email = $user, pass = crypto::argon2::generate($pass) ) 
  SIGNIN ( SELECT * FROM user WHERE email = $user AND crypto::argon2::compare(pass, $pass) ) 
  WITH JWT ALGORITHM HS512 KEY "${JWT_SECRET}" 
  DURATION FOR TOKEN 1h, FOR SESSION 1d;

DEFINE TABLE user SCHEMAFULL;
DEFINE FIELD email ON user TYPE string ASSERT string::is::email($value);
DEFINE FIELD pass ON user TYPE string;
DEFINE INDEX idx_user_email ON TABLE user COLUMNS email UNIQUE;

-- -----------------------------------------------------------------------------
-- CATEGORIES (002_categories) - SCHEMALESS
-- -----------------------------------------------------------------------------
-- Schemaless allows flexibility while still having permissions
DEFINE TABLE categories
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

-- Indexes for performance (don't require SCHEMAFULL)
DEFINE INDEX idx_categories_user_name ON TABLE categories COLUMNS created_by, name UNIQUE;
DEFINE INDEX idx_categories_user ON TABLE categories COLUMNS created_by;

-- -----------------------------------------------------------------------------
-- TASKS (003_tasks) - SCHEMALESS with category record link
-- -----------------------------------------------------------------------------
-- Schemaless: fields are not enforced, just store what you send
-- category field holds a record link: categories:abc123
DEFINE TABLE tasks
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

-- Indexes for performance
DEFINE INDEX idx_tasks_user ON TABLE tasks COLUMNS created_by;
DEFINE INDEX idx_tasks_time ON TABLE tasks COLUMNS start_date;
DEFINE INDEX idx_tasks_status ON TABLE tasks COLUMNS completed;
DEFINE INDEX idx_tasks_category ON TABLE tasks COLUMNS category;

-- -----------------------------------------------------------------------------
-- QUERY EXAMPLES
-- -----------------------------------------------------------------------------
-- 
-- Get task with category populated:
--   SELECT *, category.* FROM tasks:id FETCH category
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
-- DEFINE TABLE tags;
-- Task would have: tags = [tags:tag1, tags:tag2]
-- Query: SELECT *, tags.* FROM tasks FETCH tags

-- Moods
-- DEFINE TABLE moods;
-- Task would have: mood = moods:happy
-- Query: SELECT *, mood.* FROM tasks FETCH mood
