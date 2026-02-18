-- =============================================================================
-- GOAL ANALYTICS OPTIMIZATION MIGRATION - SIMPLIFIED
-- =============================================================================
-- Creates goal_daily_stats table for O(1) progress lookups.
-- Table is SCHEMALESS for maximum compatibility with SurrealDB 2.6.
--
-- Note: Stats are updated via Go code in tasks/repository.go when task_goals
-- edges are created/updated. Database events were removed due to instability.
-- =============================================================================

-- Simple SCHEMALESS table for goal daily stats
DEFINE TABLE IF NOT EXISTS goal_daily_stats SCHEMALESS PERMISSIONS FULL;

-- Indexes for efficient queries
DEFINE INDEX IF NOT EXISTS idx_goal_daily_unique ON goal_daily_stats COLUMNS goal_id, date UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_goal_daily_user_date ON goal_daily_stats COLUMNS created_by, date;

-- Enhanced index for task_goals lookups
DEFINE INDEX IF NOT EXISTS idx_task_goals_goal_completed ON task_goals COLUMNS out, in;
