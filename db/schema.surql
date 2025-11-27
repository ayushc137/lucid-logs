-- =============================================================================
-- DAILY JOURNAL APP - Database Schema (SurrealDB)
-- =============================================================================
-- 
-- This file is the CONSOLIDATED SCHEMA for development and reference.
-- 
-- ARCHITECTURE: Graph-First Design
-- ================================
-- We use SurrealDB's graph relations for flexible relationships:
--   task -[in_category]-> category
--   (future: task -[has_tag]-> tag, task -[has_mood]-> mood, etc.)
--
-- Benefits:
--   - Easy to add many-to-many later
--   - Relationship metadata (when, by whom)
--   - Powerful graph traversals for analytics
--   - Natural for time-series and journaling data
-- 
-- MIGRATION FILES: db/migrations/
-- ================================
-- 000_init.surql       - Migration tracking
-- 001_auth.surql       - Authentication
-- 002_categories.surql - Categories table
-- 003_tasks.surql      - Tasks table
-- 004_task_relations.surql - Graph edges
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
-- CATEGORIES (002_categories)
-- -----------------------------------------------------------------------------
DEFINE TABLE categories SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

DEFINE FIELD name ON categories TYPE string ASSERT string::len(string::trim($value)) > 0;
DEFINE FIELD color ON categories TYPE string ASSERT string::len(string::trim($value)) > 0;
DEFINE FIELD created_at ON categories TYPE option<datetime> VALUE $value OR time::now();
DEFINE FIELD updated_at ON categories TYPE option<datetime> VALUE time::now();
DEFINE FIELD deleted_at ON categories TYPE option<datetime> VALUE $value ?? NONE;
DEFINE FIELD created_by ON categories TYPE string ASSERT string::len($value) > 0;
DEFINE FIELD updated_by ON categories TYPE string ASSERT string::len($value) > 0;
DEFINE INDEX idx_categories_user_name ON TABLE categories COLUMNS created_by, name UNIQUE;
DEFINE INDEX idx_categories_user ON TABLE categories COLUMNS created_by;

-- -----------------------------------------------------------------------------
-- TASKS (003_tasks) - Core entity, NO foreign keys
-- -----------------------------------------------------------------------------
DEFINE TABLE tasks SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

DEFINE FIELD title ON tasks TYPE string ASSERT string::len(string::trim($value)) > 0;
DEFINE FIELD journal ON tasks TYPE option<string>;
DEFINE FIELD start_date ON tasks TYPE datetime ASSERT $value != NONE;
DEFINE FIELD end_date ON tasks TYPE datetime ASSERT $value >= start_date;
DEFINE FIELD completed ON tasks TYPE bool DEFAULT false;
DEFINE FIELD priority ON tasks TYPE number DEFAULT 0;
DEFINE FIELD source ON tasks TYPE string VALUE $value ?? "manual" ASSERT $value = "manual";
DEFINE FIELD note ON tasks TYPE option<string>;
DEFINE FIELD positives ON tasks TYPE array<string> DEFAULT [];
DEFINE FIELD negatives ON tasks TYPE array<string> DEFAULT [];
DEFINE FIELD created_at ON tasks TYPE option<datetime> VALUE $value OR time::now();
DEFINE FIELD updated_at ON tasks TYPE option<datetime> VALUE time::now();
DEFINE FIELD deleted_at ON tasks TYPE option<datetime> VALUE $value ?? NONE;
DEFINE FIELD created_by ON tasks TYPE string ASSERT string::len($value) > 0;
DEFINE FIELD updated_by ON tasks TYPE string ASSERT string::len($value) > 0;
DEFINE INDEX idx_tasks_user ON TABLE tasks COLUMNS created_by;
DEFINE INDEX idx_tasks_time ON TABLE tasks COLUMNS start_date;
DEFINE INDEX idx_tasks_status ON TABLE tasks COLUMNS completed;

-- -----------------------------------------------------------------------------
-- GRAPH RELATIONS (004_task_relations)
-- -----------------------------------------------------------------------------
-- Edge: task -[in_category]-> category
-- Query: SELECT *, ->in_category->categories AS category FROM tasks
DEFINE TABLE in_category SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR in.created_by = $auth.id
    FOR create WHERE $auth = NONE OR in.created_by = $auth.id
    FOR update WHERE $auth = NONE OR in.created_by = $auth.id
    FOR delete WHERE $auth = NONE OR in.created_by = $auth.id;

DEFINE FIELD in ON in_category TYPE record<tasks>;
DEFINE FIELD out ON in_category TYPE record<categories>;
DEFINE FIELD assigned_at ON in_category TYPE datetime VALUE time::now();
DEFINE FIELD assigned_by ON in_category TYPE option<string>;

-- One task -> one category (for now - remove for many-to-many)
DEFINE INDEX idx_in_category_unique_task ON TABLE in_category COLUMNS in UNIQUE;
DEFINE INDEX idx_in_category_out ON TABLE in_category COLUMNS out;

-- -----------------------------------------------------------------------------
-- GRAPH QUERY EXAMPLES
-- -----------------------------------------------------------------------------
-- 
-- Get task with category:
--   SELECT *, ->in_category->categories[0] AS category FROM tasks:id
--
-- Get all tasks in a category:
--   SELECT <-in_category<-tasks FROM categories:id
--
-- Find tasks by category name:
--   SELECT <-in_category<-tasks FROM categories WHERE name = "Work"
--
-- Complex traversal (future):
--   SELECT 
--     *,
--     ->in_category->categories AS category,
--     ->has_mood->moods AS mood,
--     ->has_tag->tags AS tags
--   FROM tasks WHERE created_by = $user
--
-- Analytics - tasks per category:
--   SELECT 
--     out.name AS category,
--     count() AS task_count
--   FROM in_category 
--   WHERE in.created_by = $user 
--   GROUP BY out
--
-- -----------------------------------------------------------------------------

-- -----------------------------------------------------------------------------
-- FUTURE TABLES (Graph-ready)
-- -----------------------------------------------------------------------------

-- Tags (many-to-many with tasks)
-- DEFINE TABLE tags SCHEMAFULL;
-- DEFINE FIELD name ON tags TYPE string;
-- DEFINE FIELD color ON tags TYPE option<string>;
-- DEFINE FIELD created_by ON tags TYPE string;
-- 
-- DEFINE TABLE has_tag SCHEMAFULL;  -- Edge: task -[has_tag]-> tag
-- DEFINE FIELD in ON has_tag TYPE record<tasks>;
-- DEFINE FIELD out ON has_tag TYPE record<tags>;

-- Moods (one-to-one with tasks)
-- DEFINE TABLE moods SCHEMAFULL;
-- DEFINE FIELD name ON moods TYPE string;
-- DEFINE FIELD emoji ON moods TYPE string;
-- DEFINE FIELD energy_level ON moods TYPE number;
--
-- DEFINE TABLE has_mood SCHEMAFULL;  -- Edge: task -[has_mood]-> mood
-- DEFINE FIELD in ON has_mood TYPE record<tasks>;
-- DEFINE FIELD out ON has_mood TYPE record<moods>;
-- DEFINE FIELD recorded_at ON has_mood TYPE datetime;
