-- =============================================================================
-- LUCID LOGS - COMPLETE DATABASE SCHEMA
-- =============================================================================
--
-- Single consolidated schema with all tables, indexes, and relations.
-- This file serves as the reference documentation for the database structure.
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
DEFINE TABLE IF NOT EXISTS goals PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_goals_user ON TABLE goals COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_goals_status ON TABLE goals COLUMNS created_by, status;
DEFINE INDEX IF NOT EXISTS idx_goals_deleted ON TABLE goals COLUMNS created_by, deleted_at;
DEFINE INDEX IF NOT EXISTS idx_goals_priority ON TABLE goals COLUMNS created_by, priority;

-- ACTIVITIES (quick-log, scheduling, timer/flowmodoro)
DEFINE TABLE IF NOT EXISTS activities PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_activities_user ON TABLE activities COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_activities_pinned ON TABLE activities COLUMNS created_by, pinned;
DEFINE INDEX IF NOT EXISTS idx_activities_deleted ON TABLE activities COLUMNS created_by, deleted_at;

-- UNITS (measurement units)
DEFINE TABLE IF NOT EXISTS units PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_units_system ON TABLE units COLUMNS is_system;
DEFINE INDEX IF NOT EXISTS idx_units_user ON TABLE units COLUMNS created_by;

-- RETROSPECTIVES
DEFINE TABLE IF NOT EXISTS retrospectives PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_retros_user_date ON TABLE retrospectives COLUMNS created_by, start_date;
DEFINE INDEX IF NOT EXISTS idx_retros_type ON TABLE retrospectives COLUMNS created_by, retro_type;
DEFINE INDEX IF NOT EXISTS idx_retros_deleted ON TABLE retrospectives COLUMNS created_by, deleted_at;

-- TIMER_SESSIONS (Flowmodoro tracking)
DEFINE TABLE IF NOT EXISTS timer_sessions PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_timer_sessions_user ON TABLE timer_sessions COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_timer_sessions_active ON TABLE timer_sessions COLUMNS created_by, status;

-- EMOTIONS (seeded separately in 002_seed_emotions.surql)
DEFINE TABLE IF NOT EXISTS emotions SCHEMALESS PERMISSIONS FULL;

-- GOAL_SNAPSHOTS (point-in-time state for history)
DEFINE TABLE IF NOT EXISTS goal_snapshots PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS created_by ON goal_snapshots TYPE record<users>;
DEFINE INDEX IF NOT EXISTS idx_goal_snapshots_goal ON TABLE goal_snapshots COLUMNS goal_id;
DEFINE INDEX IF NOT EXISTS idx_goal_snapshots_user ON TABLE goal_snapshots COLUMNS created_by;

-- GOAL_LOGS (event history for goals)
DEFINE TABLE IF NOT EXISTS goal_logs PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS created_by ON goal_logs TYPE record<users>;
DEFINE FIELD IF NOT EXISTS event_type ON goal_logs TYPE string;
DEFINE FIELD IF NOT EXISTS changes ON goal_logs TYPE option<object>;
DEFINE FIELD IF NOT EXISTS triggered_by_task_id ON goal_logs TYPE option<string>;
DEFINE FIELD IF NOT EXISTS created_at ON goal_logs TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_goal_logs_goal ON TABLE goal_logs COLUMNS in, created_at;
DEFINE INDEX IF NOT EXISTS idx_goal_logs_user ON TABLE goal_logs COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_goal_logs_event ON TABLE goal_logs COLUMNS event_type;

-- AGG_DAILY (pre-computed daily metrics)
DEFINE TABLE IF NOT EXISTS agg_daily PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_agg_daily ON TABLE agg_daily COLUMNS user_id, date UNIQUE;

-- GOAL_DAILY_STATS (pre-computed daily goal progress for O(1) lookups)
DEFINE TABLE IF NOT EXISTS goal_daily_stats SCHEMAFULL PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS goal_id ON goal_daily_stats TYPE record<goals>;
DEFINE FIELD IF NOT EXISTS date ON goal_daily_stats TYPE datetime;
DEFINE FIELD IF NOT EXISTS created_by ON goal_daily_stats TYPE record<users>;
DEFINE FIELD IF NOT EXISTS daily_value ON goal_daily_stats TYPE float DEFAULT 0;
DEFINE FIELD IF NOT EXISTS cumulative_value ON goal_daily_stats TYPE float DEFAULT 0;
DEFINE FIELD IF NOT EXISTS contribution_count ON goal_daily_stats TYPE int DEFAULT 0;
DEFINE INDEX IF NOT EXISTS idx_goal_daily_unique ON goal_daily_stats COLUMNS goal_id, date UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_goal_daily_user_date ON goal_daily_stats COLUMNS created_by, date;

-- GOAL_PERIOD_SNAPSHOTS (week/month analytics snapshots)
DEFINE TABLE IF NOT EXISTS goal_period_snapshots SCHEMAFULL PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS goal_id ON goal_period_snapshots TYPE record<goals>;
DEFINE FIELD IF NOT EXISTS period_type ON goal_period_snapshots TYPE string;
DEFINE FIELD IF NOT EXISTS period_key ON goal_period_snapshots TYPE string;
DEFINE FIELD IF NOT EXISTS created_by ON goal_period_snapshots TYPE record<users>;
DEFINE INDEX IF NOT EXISTS idx_goal_period_unique ON goal_period_snapshots COLUMNS goal_id, period_type, period_key UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_goal_period_user ON goal_period_snapshots COLUMNS created_by, period_type, start_date;

-- STREAK_HISTORY (streak changes over time for analytics)
DEFINE TABLE IF NOT EXISTS streak_history SCHEMAFULL PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS goal_id ON streak_history TYPE record<goals>;
DEFINE FIELD IF NOT EXISTS created_by ON streak_history TYPE record<users>;
DEFINE FIELD IF NOT EXISTS date ON streak_history TYPE datetime;
DEFINE FIELD IF NOT EXISTS event ON streak_history TYPE string;
DEFINE INDEX IF NOT EXISTS idx_streak_goal_date ON streak_history COLUMNS goal_id, date;
DEFINE INDEX IF NOT EXISTS idx_streak_user ON streak_history COLUMNS created_by, date;


-- =============================================================================
-- RELATION TABLES (Graph Edges)
-- =============================================================================
-- These define the links between tables for the entity relationship diagram.
-- Format: DEFINE TABLE name TYPE RELATION IN source_table OUT target_table
--
-- Relation Summary:
--   tasks ──────────────────► emotions      (task_emotions)
--   tasks ──────────────────► goals         (task_goals)
--   tasks ──────────────────► activities    (created_from_activity)
--   tasks ──────────────────► categories    (in_category)
--   goals ──────────────────► goals         (goal_children)
--   goals ──────────────────► categories    (in_category)
--   activities ─────────────► goals         (activity_goals)
--   activities ─────────────► categories    (in_category)


-- TASK_EMOTIONS (tasks -> emotions)
DEFINE TABLE IF NOT EXISTS task_emotions TYPE RELATION IN tasks OUT emotions PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS type ON task_emotions TYPE string ASSERT $value IN ["primary", "positive", "negative"];
DEFINE FIELD IF NOT EXISTS text ON task_emotions TYPE option<string>;
DEFINE FIELD IF NOT EXISTS created_at ON task_emotions TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_task_emotions_out ON TABLE task_emotions COLUMNS out;
DEFINE INDEX IF NOT EXISTS idx_task_emotions_type ON TABLE task_emotions COLUMNS type;

-- TASK_GOALS (tasks -> goals)
DEFINE TABLE IF NOT EXISTS task_goals TYPE RELATION IN tasks OUT goals PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS impact_type ON task_goals TYPE string ASSERT $value IN ["positive", "negative", "neutral"];
DEFINE FIELD IF NOT EXISTS quantity_value ON task_goals TYPE option<float>;
DEFINE FIELD IF NOT EXISTS unit_id ON task_goals TYPE option<string>;
DEFINE FIELD IF NOT EXISTS is_milestone ON task_goals TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS milestone_label ON task_goals TYPE option<string>;
DEFINE FIELD IF NOT EXISTS milestone_order ON task_goals TYPE option<int>;
DEFINE FIELD IF NOT EXISTS notes ON task_goals TYPE option<string>;
DEFINE FIELD IF NOT EXISTS source ON task_goals TYPE string DEFAULT "manual";
DEFINE FIELD IF NOT EXISTS created_at ON task_goals TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_task_goals_out ON TABLE task_goals COLUMNS out;
DEFINE INDEX IF NOT EXISTS idx_task_goals_in ON TABLE task_goals COLUMNS in;

-- CREATED_FROM_ACTIVITY (tasks -> activities)
DEFINE TABLE IF NOT EXISTS created_from_activity TYPE RELATION IN tasks OUT activities PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS mode ON created_from_activity TYPE string DEFAULT "instant";
DEFINE FIELD IF NOT EXISTS created_at ON created_from_activity TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_created_from_activity_out ON TABLE created_from_activity COLUMNS out;
DEFINE INDEX IF NOT EXISTS idx_created_from_activity_in ON TABLE created_from_activity COLUMNS in;

-- ACTIVITY_GOALS (activities -> goals)
DEFINE TABLE IF NOT EXISTS activity_goals TYPE RELATION IN activities OUT goals PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS auto_link_tasks ON activity_goals TYPE bool DEFAULT true;
DEFINE FIELD IF NOT EXISTS quantity_multiplier ON activity_goals TYPE float DEFAULT 1.0;
DEFINE FIELD IF NOT EXISTS default_quantity ON activity_goals TYPE option<float>;
DEFINE FIELD IF NOT EXISTS default_impact ON activity_goals TYPE string DEFAULT "positive";
DEFINE FIELD IF NOT EXISTS created_at ON activity_goals TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_activity_goals_out ON TABLE activity_goals COLUMNS out;
DEFINE INDEX IF NOT EXISTS idx_activity_goals_in ON TABLE activity_goals COLUMNS in;

-- GOAL_CHILDREN (goals -> goals, parent-child hierarchy)
DEFINE TABLE IF NOT EXISTS goal_children TYPE RELATION IN goals OUT goals PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS order ON goal_children TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS required ON goal_children TYPE bool DEFAULT true;
DEFINE FIELD IF NOT EXISTS created_at ON goal_children TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_goal_children_in ON TABLE goal_children COLUMNS in;
DEFINE INDEX IF NOT EXISTS idx_goal_children_out ON TABLE goal_children COLUMNS out;

-- IN_CATEGORY (tasks | goals | activities -> categories)
DEFINE TABLE IF NOT EXISTS in_category TYPE RELATION IN tasks | goals | activities OUT categories PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS created_at ON in_category TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_in_category_out ON TABLE in_category COLUMNS out;
DEFINE INDEX IF NOT EXISTS idx_in_category_in ON TABLE in_category COLUMNS in;


-- =============================================================================
-- DEPRECATED TABLES (for cleanup reference)
-- =============================================================================
-- These tables have been replaced. Run the cleanup commands on existing databases.
--
-- REMOVE TABLE IF EXISTS templates;
-- REMOVE TABLE IF EXISTS template_goals;
-- REMOVE TABLE IF EXISTS created_from;


-- =============================================================================
-- END SCHEMA
-- =============================================================================
