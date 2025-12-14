-- =============================================================================
-- GOAL ACTIONS MIGRATION
-- =============================================================================
-- 
-- This migration adds:
--   1. Goal Actions table (subtasks/action items for goals)
--   2. Updates goals table with completion_mode field
--
-- Run: surreal import --conn http://localhost:8000 --ns lucid --db logs db/migrations/005_goal_actions.surql
--
-- =============================================================================

-- =============================================================================
-- GOAL ACTIONS TABLE
-- =============================================================================
-- Goal actions are subtasks or action items within a goal.
-- For discrete goals: these are the steps to complete the goal.
-- For recurring goals: these define activities that count toward the goal.

DEFINE TABLE IF NOT EXISTS goal_actions PERMISSIONS FULL;

-- Indexes for query performance
DEFINE INDEX IF NOT EXISTS idx_goal_actions_goal ON TABLE goal_actions COLUMNS goal_id;
DEFINE INDEX IF NOT EXISTS idx_goal_actions_goal_order ON TABLE goal_actions COLUMNS goal_id, order;
DEFINE INDEX IF NOT EXISTS idx_goal_actions_user ON TABLE goal_actions COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_goal_actions_deleted ON TABLE goal_actions COLUMNS goal_id, deleted_at;

-- Goal action fields (schemaless, documented for reference):
-- {
--   id: "goal_actions:abc123",
--   goal_id: "goals:xyz",         // Parent goal ID (string, not record link for simplicity)
--   created_by: "users:abc",
--   
--   // Action definition
--   title: "Complete Module 1",
--   description: "Finish all lessons in Module 1",
--   order: 1,                     // Display order
--   
--   // Quantity (for measurable contributions)
--   quantity_value: 5.0,          // Optional: how much this action contributes
--   quantity_unit: "km",          // Optional: unit of measurement
--   
--   // Completion status (for discrete goals only)
--   completed: false,
--   completed_at: datetime,       // When marked complete
--   
--   // Metadata
--   created_at: datetime,
--   updated_at: datetime,
--   deleted_at: datetime          // Soft delete
-- }

-- =============================================================================
-- GOALS TABLE UPDATE: Add completion_mode
-- =============================================================================
-- completion_mode controls how epic goals auto-complete:
--   "all" (default): All child goals must be complete (AND logic)
--   "any": Any one child goal completes the epic (OR logic)

-- Note: SurrealDB schemaless - field will be added automatically on insert/update
-- This comment documents the expected field:
-- goals.completion_mode: "all" | "any" (default: "all" for epic goals)

-- =============================================================================
-- END MIGRATION
-- =============================================================================
