-- =============================================================================
-- TASK APP - Database Schema (SurrealDB)
-- =============================================================================
-- 
-- This is the COMPLETE schema reference. Run db/migrations/*.surql files to
-- apply changes to the database.
-- 
-- ARCHITECTURE: Schemaless with Record Links
-- ==========================================
-- We use SurrealDB's schemaless tables with direct record links for relations.
-- Pattern: task.category = categories:work123
-- Query:   SELECT *, category.* FROM tasks FETCH category
--
-- PERMISSIONS: Service Account Model
-- ===================================
-- Tables use PERMISSIONS FULL because:
--   - The Go service connects as root/service account
--   - Multi-tenancy enforced in Go via created_by filters
--
-- =============================================================================

-- -----------------------------------------------------------------------------
-- USERS (Authentication - SCHEMAFULL)
-- -----------------------------------------------------------------------------
REMOVE TABLE IF EXISTS users;
DEFINE TABLE users SCHEMAFULL PERMISSIONS FULL;
DEFINE FIELD email ON users TYPE string ASSERT string::is::email($value);
DEFINE FIELD pass ON users TYPE string;
DEFINE FIELD is_admin ON users TYPE bool DEFAULT false;
DEFINE FIELD created_at ON users TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON users TYPE datetime DEFAULT time::now();
DEFINE INDEX idx_users_email ON TABLE users COLUMNS email UNIQUE;

-- -----------------------------------------------------------------------------
-- CATEGORIES (Schemaless + FULL Permissions)
-- -----------------------------------------------------------------------------
DEFINE TABLE IF NOT EXISTS categories PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_categories_user_name ON TABLE categories COLUMNS created_by, name UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_categories_user_active ON TABLE categories COLUMNS created_by, deleted_at;

-- -----------------------------------------------------------------------------
-- TASKS (Schemaless with category record link)
-- -----------------------------------------------------------------------------
DEFINE TABLE IF NOT EXISTS tasks PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_date ON TABLE tasks COLUMNS created_by, start_date;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_active ON TABLE tasks COLUMNS created_by, deleted_at;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_completed ON TABLE tasks COLUMNS created_by, completed;
DEFINE INDEX IF NOT EXISTS idx_tasks_category ON TABLE tasks COLUMNS category;

-- -----------------------------------------------------------------------------
-- EMOTIONS (Reference data - seeded by migration)
-- -----------------------------------------------------------------------------
-- 100 emotions with 6D vectors for mood tracking
DEFINE TABLE IF NOT EXISTS emotions SCHEMALESS PERMISSIONS FULL;
-- Fields: name, emoji, quadrant, x, y, valence, arousal, dominance, intensity, certainty, social, description

-- -----------------------------------------------------------------------------
-- TASK_EMOTIONS (Edge table for analytics)
-- -----------------------------------------------------------------------------
-- Links tasks to emotions: task -> task_emotions -> emotions:E16
-- Types: primary, positive, negative
DEFINE TABLE IF NOT EXISTS task_emotions TYPE RELATION IN tasks OUT emotions PERMISSIONS FULL;
DEFINE FIELD type ON task_emotions TYPE string ASSERT $value IN ["primary", "positive", "negative"];
DEFINE FIELD text ON task_emotions TYPE option<string>;
DEFINE FIELD created_at ON task_emotions TYPE datetime DEFAULT time::now();
DEFINE INDEX idx_task_emotions_out ON TABLE task_emotions COLUMNS out;
DEFINE INDEX idx_task_emotions_type ON TABLE task_emotions COLUMNS type;

-- -----------------------------------------------------------------------------
-- QUERY EXAMPLES
-- -----------------------------------------------------------------------------
-- 
-- Task with category:
--   SELECT *, category.* FROM tasks:id FETCH category
--
-- Get emotion by ID:
--   SELECT * FROM emotions:E16
--
-- Task's emotion edges:
--   SELECT ->task_emotions->emotions.* FROM tasks:abc
--
-- All tasks with specific emotion:
--   SELECT <-task_emotions<-tasks.* FROM emotions:E16
--
-- Emotion frequency per user:
--   SELECT out.name as emotion, count() 
--   FROM task_emotions WHERE in.created_by = $user 
--   GROUP BY out
--
-- Most used emotions this week:
--   SELECT out.name as emotion, out.emoji, count() as times
--   FROM task_emotions 
--   WHERE created_at > time::now() - 7d
--   GROUP BY out ORDER BY times DESC LIMIT 10
