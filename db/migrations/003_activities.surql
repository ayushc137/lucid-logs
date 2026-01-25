-- =============================================================================
-- ACTIVITIES MIGRATION
-- =============================================================================
-- 
-- Creates the activities system to replace templates with enhanced functionality.
-- Supports: instant logging, scheduling, and timer/Flowmodoro modes.
--
-- Run: surreal sql -e http://localhost:8000 -u root -p root -ns lucid -db logs < db/migrations/003_activities.surql
--
-- =============================================================================


-- =============================================================================
-- ACTIVITIES TABLE
-- =============================================================================

DEFINE TABLE IF NOT EXISTS activities SCHEMAFULL PERMISSIONS FULL;

-- Identity
DEFINE FIELD IF NOT EXISTS title ON activities TYPE string ASSERT $value != NONE AND string::len($value) >= 1;
DEFINE FIELD IF NOT EXISTS icon ON activities TYPE option<string>;
DEFINE FIELD IF NOT EXISTS description ON activities TYPE option<string>;

-- Task Defaults (pre-fill when creating task)
DEFINE FIELD IF NOT EXISTS default_duration ON activities TYPE option<int>;  -- seconds
DEFINE FIELD IF NOT EXISTS default_emotion_id ON activities TYPE option<string>;  -- record ID as string
DEFINE FIELD IF NOT EXISTS default_priority ON activities TYPE int DEFAULT 3;
DEFINE FIELD IF NOT EXISTS default_completed ON activities TYPE bool DEFAULT true;  -- true for instant log

-- Quantity Settings
DEFINE FIELD IF NOT EXISTS quantity_enabled ON activities TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS quantity_default ON activities TYPE option<float>;
DEFINE FIELD IF NOT EXISTS quantity_step ON activities TYPE option<float>;
DEFINE FIELD IF NOT EXISTS quantity_unit_id ON activities TYPE option<string>;  -- fallback unit if no goal

-- Goal Link Defaults
DEFINE FIELD IF NOT EXISTS default_impact ON activities TYPE string DEFAULT "positive" 
    ASSERT $value IN ["positive", "negative", "neutral"];

-- Display & Organization
DEFINE FIELD IF NOT EXISTS category_id ON activities TYPE option<string>;  -- for quick access, also use in_category relation
DEFINE FIELD IF NOT EXISTS pinned ON activities TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS sort_order ON activities TYPE int DEFAULT 0;

-- Usage Statistics
DEFINE FIELD IF NOT EXISTS use_count ON activities TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS last_used_at ON activities TYPE option<datetime>;

-- Active Timer Session (Flowmodoro)
-- Stored as flexible object: { task_id: string, started_at: datetime, breaks: [...] }
DEFINE FIELD IF NOT EXISTS active_session ON activities TYPE option<object>;

-- Multi-tenancy
DEFINE FIELD IF NOT EXISTS created_by ON activities TYPE record<users>;

-- Metadata
DEFINE FIELD IF NOT EXISTS created_at ON activities TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON activities TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS deleted_at ON activities TYPE option<datetime>;


-- =============================================================================
-- ACTIVITIES INDEXES
-- =============================================================================

-- Primary user index (all queries filter by user)
DEFINE INDEX IF NOT EXISTS idx_activities_user ON activities FIELDS created_by;

-- Soft delete filter (most queries exclude deleted)
DEFINE INDEX IF NOT EXISTS idx_activities_user_active ON activities FIELDS created_by, deleted_at;

-- Pinned activities for quick bar (hot path)
DEFINE INDEX IF NOT EXISTS idx_activities_pinned ON activities FIELDS created_by, pinned, sort_order;

-- Active timer sessions (find user's active timers)
DEFINE INDEX IF NOT EXISTS idx_activities_active_session ON activities FIELDS created_by, active_session;

-- Usage stats for suggestions
DEFINE INDEX IF NOT EXISTS idx_activities_last_used ON activities FIELDS created_by, last_used_at;
DEFINE INDEX IF NOT EXISTS idx_activities_use_count ON activities FIELDS created_by, use_count;


-- =============================================================================
-- ACTIVITY_GOALS RELATION TABLE
-- =============================================================================
-- Many-to-many: One activity can link to multiple goals
-- When activity is used, tasks are auto-linked to these goals

DEFINE TABLE IF NOT EXISTS activity_goals TYPE RELATION IN activities OUT goals PERMISSIONS FULL;

-- Auto-link tasks created from this activity to the goal
DEFINE FIELD IF NOT EXISTS auto_link_tasks ON activity_goals TYPE bool DEFAULT true;

-- Quantity multiplier (e.g., 1 workout = 30 minutes toward cardio goal)
DEFINE FIELD IF NOT EXISTS quantity_multiplier ON activity_goals TYPE float DEFAULT 1.0;

-- Override impact type per goal (e.g., same activity positive for one goal, negative for another)
DEFINE FIELD IF NOT EXISTS default_impact ON activity_goals TYPE string DEFAULT "positive"
    ASSERT $value IN ["positive", "negative", "neutral"];

-- Metadata
DEFINE FIELD IF NOT EXISTS created_at ON activity_goals TYPE datetime DEFAULT time::now();

-- Indexes for activity_goals
DEFINE INDEX IF NOT EXISTS idx_activity_goals_out ON activity_goals FIELDS out;  -- Find activities for a goal
DEFINE INDEX IF NOT EXISTS idx_activity_goals_in ON activity_goals FIELDS in;    -- Find goals for an activity


-- =============================================================================
-- UPDATE IN_CATEGORY TO INCLUDE ACTIVITIES
-- =============================================================================

-- Redefine to include activities as valid input (templates removed)
DEFINE TABLE IF NOT EXISTS in_category TYPE RELATION IN tasks | goals | activities OUT categories PERMISSIONS FULL;


-- =============================================================================
-- CREATED_FROM RELATION - UPDATE FOR ACTIVITIES
-- =============================================================================

-- Track which activity a task was created from
DEFINE TABLE IF NOT EXISTS created_from_activity TYPE RELATION IN tasks OUT activities PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS created_at ON created_from_activity TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS mode ON created_from_activity TYPE string DEFAULT "instant"
    ASSERT $value IN ["instant", "schedule", "timer"];

-- Index for activity usage stats
DEFINE INDEX IF NOT EXISTS idx_created_from_activity_out ON created_from_activity FIELDS out;
DEFINE INDEX IF NOT EXISTS idx_created_from_activity_mode ON created_from_activity FIELDS out, mode;


-- =============================================================================
-- TIMER SESSIONS TABLE (Optional - for persistent timer state)
-- =============================================================================
-- Stores active Flowmodoro sessions separately for reliability
-- (Alternative: store directly on activity.active_session)

DEFINE TABLE IF NOT EXISTS timer_sessions SCHEMAFULL PERMISSIONS FULL;

DEFINE FIELD IF NOT EXISTS activity_id ON timer_sessions TYPE record<activities>;
DEFINE FIELD IF NOT EXISTS task_id ON timer_sessions TYPE record<tasks>;
DEFINE FIELD IF NOT EXISTS created_by ON timer_sessions TYPE record<users>;

-- Session state
DEFINE FIELD IF NOT EXISTS status ON timer_sessions TYPE string DEFAULT "active"
    ASSERT $value IN ["active", "paused", "completed", "abandoned"];

-- Timing
DEFINE FIELD IF NOT EXISTS started_at ON timer_sessions TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS ended_at ON timer_sessions TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS total_work_seconds ON timer_sessions TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS total_break_seconds ON timer_sessions TYPE int DEFAULT 0;

-- Flowmodoro breaks: [{ started_at, ended_at, duration_seconds, type: "short"|"long" }]
DEFINE FIELD IF NOT EXISTS breaks ON timer_sessions TYPE option<array>;

-- Current segment tracking
DEFINE FIELD IF NOT EXISTS current_segment_start ON timer_sessions TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS is_on_break ON timer_sessions TYPE bool DEFAULT false;

-- Metadata
DEFINE FIELD IF NOT EXISTS created_at ON timer_sessions TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON timer_sessions TYPE datetime DEFAULT time::now();

-- Indexes
DEFINE INDEX IF NOT EXISTS idx_timer_sessions_user_active ON timer_sessions FIELDS created_by, status;
DEFINE INDEX IF NOT EXISTS idx_timer_sessions_activity ON timer_sessions FIELDS activity_id;


-- =============================================================================
-- DATA MIGRATION: TEMPLATES -> ACTIVITIES
-- =============================================================================
-- This migrates existing templates to activities

-- Step 1: Insert activities from templates
-- (Run manually or via Go migration code for safety)
/*
INSERT INTO activities (
    id,
    title,
    icon,
    description,
    default_duration,
    default_emotion_id,
    default_priority,
    default_completed,
    quantity_enabled,
    quantity_default,
    quantity_step,
    quantity_unit_id,
    default_impact,
    category_id,
    pinned,
    sort_order,
    use_count,
    last_used_at,
    active_session,
    created_by,
    created_at,
    updated_at,
    deleted_at
) SELECT
    string::concat("activities:", record::id(id)),  -- New ID
    title,
    icon,
    description,
    default_duration,
    default_emotion_id,
    3,  -- default_priority
    is_quick_log,  -- default_completed = is_quick_log
    quantity_enabled,
    quantity_default,
    quantity_step,
    NONE,  -- quantity_unit_id (will be populated from goals)
    "positive",  -- default_impact
    NONE,  -- category_id (use in_category relation)
    is_quick_log,  -- pinned = is_quick_log
    quick_log_order,  -- sort_order
    use_count,
    last_used_at,
    NONE,  -- active_session
    created_by,
    created_at,
    updated_at,
    deleted_at
FROM templates;
*/

-- Step 2: Migrate template_goals to activity_goals
-- (Run manually or via Go migration code)
/*
FOR $link IN (SELECT * FROM template_goals) {
    LET $new_in = string::concat("activities:", record::id($link.in));
    RELATE type::thing("activities", record::id($link.in)) -> activity_goals -> $link.out
    SET
        auto_link_tasks = $link.auto_link_tasks,
        quantity_multiplier = IF $link.quantity_multiplier THEN $link.quantity_multiplier ELSE 1.0 END,
        default_impact = "positive",
        created_at = $link.created_at;
};
*/


-- =============================================================================
-- USEFUL QUERIES
-- =============================================================================

-- Get pinned activities with their linked goals
/*
SELECT 
    *,
    (SELECT 
        out.id AS goal_id,
        out.title AS goal_title,
        out.icon AS goal_icon,
        out.target AS goal_target,
        auto_link_tasks,
        quantity_multiplier,
        default_impact
     FROM ->activity_goals
    ) AS goals,
    (SELECT out.* FROM ->in_category LIMIT 1)[0] AS category
FROM activities
WHERE created_by = $user_id 
  AND deleted_at IS NONE 
  AND pinned = true
ORDER BY sort_order ASC;
*/

-- Get activity with full goal details for instant log
/*
SELECT 
    *,
    (SELECT 
        out.id AS goal_id,
        out.title AS goal_title,
        out.icon AS goal_icon,
        out.target.value AS target_value,
        out.target.unit_id AS target_unit_id,
        out.target.operator AS target_operator,
        (SELECT symbol FROM units WHERE id = out.target.unit_id)[0].symbol AS unit_symbol,
        auto_link_tasks,
        quantity_multiplier,
        default_impact
     FROM ->activity_goals
     WHERE out.deleted_at IS NONE AND out.status = "active"
    ) AS goals
FROM activities:$activity_id
WHERE created_by = $user_id AND deleted_at IS NONE;
*/

-- Find user's active timer session
/*
SELECT * FROM timer_sessions 
WHERE created_by = $user_id AND status = "active"
LIMIT 1;
*/


-- =============================================================================
-- END MIGRATION
-- =============================================================================
