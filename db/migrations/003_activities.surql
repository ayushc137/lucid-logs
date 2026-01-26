-- =============================================================================
-- ACTIVITIES MIGRATION (Reference Only)
-- =============================================================================
--
-- NOTE: Activity tables are now consolidated in 001_schema.surql.
-- This file is kept for reference and data migration examples.
--
-- Tables included in 001_schema.surql:
--   - activities
--   - activity_goals (relation: activities -> goals)
--   - created_from_activity (relation: tasks -> activities)
--   - timer_sessions
--
-- =============================================================================


-- =============================================================================
-- DATA MIGRATION: TEMPLATES -> ACTIVITIES (if needed)
-- =============================================================================
-- Run these manually if migrating from the old template system.

-- Step 1: Insert activities from templates
/*
INSERT INTO activities (
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
    pinned,
    sort_order,
    use_count,
    last_used_at,
    created_by,
    created_at,
    updated_at
) SELECT
    title,
    icon,
    description,
    default_duration,
    default_emotion_id,
    3,
    is_quick_log,
    quantity_enabled,
    quantity_default,
    quantity_step,
    is_quick_log,
    quick_log_order,
    use_count,
    last_used_at,
    created_by,
    created_at,
    updated_at
FROM templates
WHERE deleted_at IS NONE;
*/

-- Step 2: Migrate template_goals to activity_goals
/*
FOR $link IN (SELECT * FROM template_goals) {
    LET $template_id = record::id($link.in);
    LET $activity = (SELECT id FROM activities WHERE title = (SELECT title FROM templates WHERE id = $link.in)[0].title LIMIT 1)[0];
    IF $activity {
        RELATE $activity.id -> activity_goals -> $link.out
        SET
            auto_link_tasks = $link.auto_link_tasks,
            quantity_multiplier = 1.0,
            default_impact = "positive",
            created_at = $link.created_at;
    }
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
     WHERE out.deleted_at IS NONE
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
        auto_link_tasks,
        quantity_multiplier,
        default_impact
     FROM ->activity_goals
     WHERE out.deleted_at IS NONE AND out.status = "active"
    ) AS goals
FROM activities:$activity_id
WHERE created_by = $user_id AND deleted_at IS NONE;
*/


-- =============================================================================
-- END MIGRATION
-- =============================================================================
