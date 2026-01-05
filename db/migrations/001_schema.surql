-- =============================================================================
-- LUCID LOGS - COMPLETE DATABASE SCHEMA
-- =============================================================================
-- 
-- Single consolidated migration with all tables, indexes, and relations.
-- This is the ONLY migration file needed (besides seed data).
--
-- Run: surreal sql -e http://localhost:8000 -u root -p root -ns lucid -db logs < db/migrations/001_schema.surql
--
-- =============================================================================


-- =============================================================================
-- CORE TABLES
-- =============================================================================

-- USERS (SCHEMAFULL - only table with strict schema)
DEFINE TABLE IF NOT EXISTS users SCHEMAFULL PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS email ON users TYPE string ASSERT string::is::email($value);
DEFINE FIELD IF NOT EXISTS pass ON users TYPE string;
DEFINE FIELD IF NOT EXISTS is_admin ON users TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS created_at ON users TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON users TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_users_email ON TABLE users COLUMNS email UNIQUE;

-- CATEGORIES
DEFINE TABLE IF NOT EXISTS categories PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_categories_user_name ON TABLE categories COLUMNS created_by, name UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_categories_user_active ON TABLE categories COLUMNS created_by, deleted_at;

-- TASKS
DEFINE TABLE IF NOT EXISTS tasks PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_date ON TABLE tasks COLUMNS created_by, start_date;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_active ON TABLE tasks COLUMNS created_by, deleted_at;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_completed ON TABLE tasks COLUMNS created_by, completed;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_end_date ON TABLE tasks COLUMNS created_by, end_date;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_priority ON TABLE tasks COLUMNS created_by, priority;

-- Full-text search for tasks
DEFINE ANALYZER IF NOT EXISTS task_search_analyzer TOKENIZERS class FILTERS lowercase, ascii, snowball(english);
DEFINE INDEX IF NOT EXISTS idx_tasks_title_fts ON TABLE tasks COLUMNS title SEARCH ANALYZER task_search_analyzer BM25 HIGHLIGHTS;
DEFINE INDEX IF NOT EXISTS idx_tasks_journal_fts ON TABLE tasks COLUMNS journal SEARCH ANALYZER task_search_analyzer BM25 HIGHLIGHTS;
DEFINE INDEX IF NOT EXISTS idx_tasks_note_fts ON TABLE tasks COLUMNS note SEARCH ANALYZER task_search_analyzer BM25 HIGHLIGHTS;

-- GOALS (with denormalized streaks)
-- Key fields: current_streak, longest_streak, last_completed_date (updated on completion)
DEFINE TABLE IF NOT EXISTS goals PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_goals_user ON TABLE goals COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_goals_status ON TABLE goals COLUMNS created_by, status;
DEFINE INDEX IF NOT EXISTS idx_goals_deleted ON TABLE goals COLUMNS created_by, deleted_at;
DEFINE INDEX IF NOT EXISTS idx_goals_priority ON TABLE goals COLUMNS created_by, priority;

-- GOAL_ENTRIES (daily habit tracking)
DEFINE TABLE IF NOT EXISTS goal_entries PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_goal_entries_goal_date ON TABLE goal_entries COLUMNS goal_id, date UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_goal_entries_goal ON TABLE goal_entries COLUMNS goal_id;

-- GOAL_SNAPSHOTS (point-in-time state for history)
DEFINE TABLE IF NOT EXISTS goal_snapshots PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_goal_snapshots_goal ON TABLE goal_snapshots COLUMNS goal_id;

-- TEMPLATES
DEFINE TABLE IF NOT EXISTS templates PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_templates_user ON TABLE templates COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_templates_quick_log ON TABLE templates COLUMNS created_by, is_quick_log;
DEFINE INDEX IF NOT EXISTS idx_templates_deleted ON TABLE templates COLUMNS created_by, deleted_at;

-- UNITS (measurement units)
DEFINE TABLE IF NOT EXISTS units PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_units_system ON TABLE units COLUMNS is_system;
DEFINE INDEX IF NOT EXISTS idx_units_user ON TABLE units COLUMNS created_by;

-- RETROSPECTIVES
DEFINE TABLE IF NOT EXISTS retrospectives PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_retros_user_date ON TABLE retrospectives COLUMNS created_by, start_date;
DEFINE INDEX IF NOT EXISTS idx_retros_type ON TABLE retrospectives COLUMNS created_by, retro_type;
DEFINE INDEX IF NOT EXISTS idx_retros_deleted ON TABLE retrospectives COLUMNS created_by, deleted_at;

-- AGG_DAILY (pre-computed daily metrics - materialized view pattern)
DEFINE TABLE IF NOT EXISTS agg_daily PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_agg_daily ON TABLE agg_daily COLUMNS user_id, date UNIQUE;


-- =============================================================================
-- REFERENCE DATA
-- =============================================================================

-- EMOTIONS (seeded separately in 002_seed_emotions.surql)
DEFINE TABLE IF NOT EXISTS emotions SCHEMALESS PERMISSIONS FULL;


-- =============================================================================
-- RELATION TABLES (Graph Edges)
-- =============================================================================

-- TASK_EMOTIONS (tasks -> emotions)
DEFINE TABLE IF NOT EXISTS task_emotions TYPE RELATION IN tasks OUT emotions PERMISSIONS FULL;
DEFINE FIELD type ON task_emotions TYPE string ASSERT $value IN ["primary", "positive", "negative"];
DEFINE FIELD text ON task_emotions TYPE option<string>;
DEFINE FIELD created_at ON task_emotions TYPE datetime DEFAULT time::now();
DEFINE INDEX idx_task_emotions_out ON TABLE task_emotions COLUMNS out;
DEFINE INDEX idx_task_emotions_type ON TABLE task_emotions COLUMNS type;

-- TASK_GOALS (tasks -> goals) with impact and milestone tracking
DEFINE TABLE IF NOT EXISTS task_goals TYPE RELATION IN tasks OUT goals PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS impact_type ON task_goals TYPE string ASSERT $value IN ["positive", "negative", "neutral"];
DEFINE FIELD IF NOT EXISTS impact_magnitude ON task_goals TYPE int ASSERT $value >= 1 AND $value <= 5;
DEFINE FIELD IF NOT EXISTS quantity_value ON task_goals TYPE option<float>;
DEFINE FIELD IF NOT EXISTS unit_id ON task_goals TYPE option<string>;
DEFINE FIELD IF NOT EXISTS is_milestone ON task_goals TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS milestone_label ON task_goals TYPE option<string>;
DEFINE FIELD IF NOT EXISTS milestone_order ON task_goals TYPE option<int>;
DEFINE FIELD IF NOT EXISTS notes ON task_goals TYPE option<string>;
DEFINE FIELD IF NOT EXISTS source ON task_goals TYPE string DEFAULT "manual";
DEFINE FIELD IF NOT EXISTS created_at ON task_goals TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_task_goals_goal ON TABLE task_goals COLUMNS out;
DEFINE INDEX IF NOT EXISTS idx_task_goals_impact ON TABLE task_goals COLUMNS out, impact_type;
DEFINE INDEX IF NOT EXISTS idx_task_goals_milestone ON TABLE task_goals COLUMNS out, is_milestone;

-- GOAL_LOGS (goals -> goal_snapshots) event history
DEFINE TABLE IF NOT EXISTS goal_logs TYPE RELATION IN goals OUT goal_snapshots PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS event_type ON goal_logs TYPE string;
DEFINE FIELD IF NOT EXISTS changes ON goal_logs TYPE option<object>;
DEFINE FIELD IF NOT EXISTS triggered_by_task_id ON goal_logs TYPE option<string>;
DEFINE FIELD IF NOT EXISTS created_at ON goal_logs TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_goal_logs_goal ON TABLE goal_logs COLUMNS in, created_at;
DEFINE INDEX IF NOT EXISTS idx_goal_logs_event ON TABLE goal_logs COLUMNS event_type;


-- =============================================================================
-- AUTO-CREATED RELATIONS (no schema needed - created on RELATE)
-- =============================================================================
-- in_category:     any -> categories     (universal category linking)
-- goal_children:   goals -> goals        (parent-child hierarchy)
-- created_from:    tasks -> templates    (task origin tracking)
-- template_goals:  templates -> goals    (template-goal linking)


-- =============================================================================
-- END SCHEMA
-- =============================================================================
