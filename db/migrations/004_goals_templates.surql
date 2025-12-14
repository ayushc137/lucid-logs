-- =============================================================================
-- GOALS & TEMPLATES MIGRATION
-- =============================================================================
-- 
-- This migration adds:
--   1. Goals table (unified: one-time + recurring/habits)
--   2. Goal Entries table (daily tracking for recurring goals)
--   3. Goal History table (versioning/audit trail)
--   4. Templates table (task blueprints for quick logging)
--   5. Updates to Tasks table (activity_key, template_id, quantity)
--
-- Run: surreal import --conn http://localhost:8000 --ns lucid --db logs db/migrations/004_goals_templates.surql
--
-- =============================================================================

-- =============================================================================
-- GOALS TABLE (Unified: one-time goals + recurring habits)
-- =============================================================================
-- Schemaless with FULL permissions (Go service handles multi-tenancy)
-- Key fields stored as schemaless JSON objects for flexibility

DEFINE TABLE IF NOT EXISTS goals PERMISSIONS FULL;

-- Core indexes for query performance
DEFINE INDEX IF NOT EXISTS idx_goals_user ON TABLE goals COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_goals_status ON TABLE goals COLUMNS created_by, status;
DEFINE INDEX IF NOT EXISTS idx_goals_activity_key ON TABLE goals COLUMNS created_by, activity_key UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_goals_deleted ON TABLE goals COLUMNS created_by, deleted_at;

-- Goal fields (schemaless, documented for reference):
-- {
--   id: "goals:abc123",
--   created_by: "users:xyz",
--   
--   // Core
--   title: "Drink 3L water daily",
--   description: "Stay hydrated throughout the day",
--   why: "Better energy and focus",
--   icon: "💧",
--   color: "#3B82F6",
--   
--   // Type & matching
--   goal_type: "discrete" | "measurable" | "epic" | "avoidance",
--   activity_key: "drink_3l_water_daily",  // Auto-generated from title
--   
--   // Recurrence (null = one-time, populated = recurring/habit)
--   recurrence: {
--     frequency: 1,           // times per period
--     period: "day" | "week" | "month",
--     active_days: ["mon", "tue", "wed", "thu", "fri"],  // optional
--     before_time: "22:00",   // optional
--     after_time: "06:00",    // optional
--     grace_days: 1           // optional
--   },
--   
--   // Target (for measurable goals)
--   target: {
--     value: 3,
--     unit: "L",
--     current_value: 1.5,     // Computed from linked tasks
--     per_period: true        // true = per day/week, false = total
--   },
--   
--   // Timeline
--   start_date: datetime,
--   deadline: datetime,       // optional
--   
--   // Status & progress
--   status: "active" | "completed" | "paused" | "abandoned",
--   completion_date: datetime,
--   
--   // Streaks (computed, for recurring goals)
--   current_streak: 0,
--   longest_streak: 0,
--   last_completed_date: datetime,
--   grace_days_used: 0,
--   
--   // Organization
--   priority: 1 | 2 | 3,
--   value_score: 1..5,
--   category: categories:work123,  // record link
--   parent_goal: goals:parent,     // for epics
--   life_domain: "health" | "work" | "learning" | ...,
--   
--   // Linked template (auto-created)
--   linked_template: templates:xyz,  // record link
--   
--   // Settings
--   is_private: false,
--   
--   // Metadata
--   created_at: datetime,
--   updated_at: datetime,
--   deleted_at: datetime
-- }

-- =============================================================================
-- GOAL ENTRIES TABLE (Daily tracking for recurring goals)
-- =============================================================================
-- Tracks daily/periodic completion of habits

DEFINE TABLE IF NOT EXISTS goal_entries PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_goal_entries_goal_date ON TABLE goal_entries COLUMNS goal_id, date UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_goal_entries_goal ON TABLE goal_entries COLUMNS goal_id;

-- Goal entry fields:
-- {
--   id: "goal_entries:abc123",
--   goal_id: goals:xyz,        // record link
--   date: datetime,            // The day this entry is for
--   
--   value: 2.5,                // Actual quantity achieved (optional)
--   met: true,                 // Was the target met?
--   
--   task_ids: ["tasks:a", "tasks:b"],  // Contributing tasks
--   notes: "Great day!",
--   
--   created_at: datetime
-- }

-- =============================================================================
-- GOAL HISTORY TABLE (Versioning & audit trail)
-- =============================================================================
-- Event-sourcing pattern for tracking all goal changes

DEFINE TABLE IF NOT EXISTS goal_history PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_goal_history_goal ON TABLE goal_history COLUMNS goal_id, created_at;
DEFINE INDEX IF NOT EXISTS idx_goal_history_period ON TABLE goal_history COLUMNS retro_period;

-- Goal history fields:
-- {
--   id: "goal_history:abc123",
--   goal_id: goals:xyz,        // record link
--   
--   event_type: "created" | "status_changed" | "deadline_changed" | 
--               "target_changed" | "title_changed" | "task_linked" |
--               "task_unlinked" | "progress_milestone" | "completed" |
--               "abandoned" | "paused" | "resumed" | "rolled_over",
--   
--   field_name: "deadline",    // Which field changed (optional)
--   old_value: object,         // Previous value (JSON)
--   new_value: object,         // New value (JSON)
--   delta: 5.0,                // For progress changes (optional)
--   
--   reason: "Holiday travel",  // User-provided reason (optional)
--   triggered_by: "user" | "system" | "rollover",
--   retro_period: "2024-W49",  // For retro grouping (optional)
--   
--   created_at: datetime
-- }

-- =============================================================================
-- TEMPLATES TABLE (Task blueprints for quick logging)
-- =============================================================================
-- Reusable task templates for one-tap logging

DEFINE TABLE IF NOT EXISTS templates PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_templates_user ON TABLE templates COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_templates_quick_log ON TABLE templates COLUMNS created_by, is_quick_log;
DEFINE INDEX IF NOT EXISTS idx_templates_activity_key ON TABLE templates COLUMNS created_by, activity_key;
DEFINE INDEX IF NOT EXISTS idx_templates_deleted ON TABLE templates COLUMNS created_by, deleted_at;

-- Template fields:
-- {
--   id: "templates:abc123",
--   created_by: "users:xyz",
--   
--   // Core
--   title: "Drink Water",
--   description: "Log water intake",
--   icon: "💧",
--   color: "#3B82F6",
--   
--   // Defaults
--   default_duration: 30,      // seconds
--   default_priority: 1,
--   default_category: categories:health,  // record link
--   
--   // Quick log settings
--   is_quick_log: true,
--   quick_log_order: 1,
--   
--   // Quantity settings
--   quantity_enabled: true,
--   quantity_default: 1,
--   quantity_unit: "glasses",
--   quantity_step: 1,
--   
--   // Emotion defaults
--   expected_quadrant: "green",
--   default_emotion_id: "emotions:E16",
--   
--   // Goal/Activity linking
--   activity_key: "drink_water",  // Inherited from goal
--   goal_id: goals:xyz,           // Source goal (optional)
--   
--   // Fields to show in UI
--   show_fields: {
--     journal: false,
--     duration: false,
--     quantity: true,
--     emotion: false,
--     positives_negatives: false,
--     notes: false
--   },
--   
--   // Source
--   is_default: false,
--   
--   // Usage tracking
--   use_count: 0,
--   last_used_at: datetime,
--   
--   // Metadata
--   created_at: datetime,
--   updated_at: datetime,
--   deleted_at: datetime
-- }

-- =============================================================================
-- TASKS TABLE UPDATES
-- =============================================================================
-- Add new fields for goal/template integration

-- Activity key for goal matching
DEFINE INDEX IF NOT EXISTS idx_tasks_activity_key ON TABLE tasks COLUMNS created_by, activity_key;

-- Template reference
DEFINE INDEX IF NOT EXISTS idx_tasks_template ON TABLE tasks COLUMNS template_id;

-- New task fields documented (schemaless):
-- {
--   // ... existing fields ...
--   
--   activity_key: "drink_water",  // For goal auto-linking
--   template_id: templates:xyz,   // Source template (record link)
--   quantity: {                   // For measurable tasks
--     value: 500,
--     unit: "ml"
--   }
-- }

-- =============================================================================
-- TASK_GOALS RELATION TABLE (Edge: tasks → goals)
-- =============================================================================
-- Links tasks to goals with impact tracking

DEFINE TABLE IF NOT EXISTS task_goals TYPE RELATION IN tasks OUT goals PERMISSIONS FULL;

DEFINE FIELD IF NOT EXISTS impact_type ON task_goals TYPE string 
    ASSERT $value IN ["positive", "negative", "neutral"];
DEFINE FIELD IF NOT EXISTS impact_magnitude ON task_goals TYPE int 
    ASSERT $value >= 1 AND $value <= 5;
DEFINE FIELD IF NOT EXISTS quantity_value ON task_goals TYPE option<float>;
DEFINE FIELD IF NOT EXISTS quantity_unit ON task_goals TYPE option<string>;
DEFINE FIELD IF NOT EXISTS notes ON task_goals TYPE option<string>;
DEFINE FIELD IF NOT EXISTS source ON task_goals TYPE string DEFAULT "manual";
DEFINE FIELD IF NOT EXISTS created_at ON task_goals TYPE datetime DEFAULT time::now();

DEFINE INDEX IF NOT EXISTS idx_task_goals_goal ON TABLE task_goals COLUMNS out;
DEFINE INDEX IF NOT EXISTS idx_task_goals_impact ON TABLE task_goals COLUMNS out, impact_type;

-- =============================================================================
-- END MIGRATION
-- =============================================================================
