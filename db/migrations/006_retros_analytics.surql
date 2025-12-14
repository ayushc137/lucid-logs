-- =============================================================================
-- RETROS & ANALYTICS MIGRATION
-- =============================================================================
--
-- This migration adds:
--   1. Retrospectives table for daily/weekly/monthly reflections
--   2. User preferences fields for scheduling (added to users)
--   3. Daily aggregation table for pre-computed analytics
--
-- Run: surreal import --conn http://localhost:8000 --ns lucid --db logs db/migrations/006_retros_analytics.surql -u root -p root
--
-- =============================================================================

-- =============================================================================
-- RETROSPECTIVES TABLE
-- =============================================================================
-- Retrospectives store auto-generated analytics summaries and user reflections

DEFINE TABLE IF NOT EXISTS retrospectives PERMISSIONS FULL;

-- Indexes for query performance
DEFINE INDEX IF NOT EXISTS idx_retros_user_date ON TABLE retrospectives COLUMNS created_by, start_date;
DEFINE INDEX IF NOT EXISTS idx_retros_type ON TABLE retrospectives COLUMNS created_by, retro_type;
DEFINE INDEX IF NOT EXISTS idx_retros_deleted ON TABLE retrospectives COLUMNS created_by, deleted_at;

-- Retrospective fields (schemaless, documented for reference):
-- {
--   id: "retrospectives:abc123",
--   created_by: "users:xyz",
--
--   // Type & Scope
--   retro_type: "daily" | "weekly" | "monthly" | "quarterly" | "yearly" | "custom",
--   start_date: datetime,
--   end_date: datetime,
--
--   // Auto-Generated Content
--   auto_summary: {
--     mood: { average_valence, average_arousal, dominant_quadrant, quadrant_distribution, notable_spikes, notable_dips },
--     habits: { met, partially_met, missed, streaks },
--     tasks: { completed, postponed, cancelled, not_started, total_duration_hours, by_category },
--     goals: { net_impact, significantly_advanced, negatively_impacted },
--     categories: { time_distribution, neglected }
--   },
--
--   // User-Editable Content
--   user_content: {
--     what_went_well: string,
--     what_didnt_go_well: string,
--     what_learned: string,
--     gratitude: string[],
--     proud_of: string,
--     change_tomorrow: string,
--     additional_notes: string
--   },
--
--   // Status
--   status: "draft" | "completed",
--
--   // Metadata
--   generated_at: datetime,
--   created_at: datetime,
--   updated_at: datetime,
--   deleted_at: datetime  // Soft delete
-- }

-- =============================================================================
-- DAILY AGGREGATION TABLE
-- =============================================================================
-- Pre-computed daily metrics for fast dashboard queries

DEFINE TABLE IF NOT EXISTS agg_daily PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_agg_daily ON TABLE agg_daily COLUMNS user_id, date;

-- Aggregation fields (schemaless, documented for reference):
-- {
--   id: "agg_daily:abc123",
--   user_id: "users:xyz",
--   date: datetime,  // Date this aggregation represents
--
--   // Task metrics
--   tasks_completed: int,
--   tasks_postponed: int,
--   tasks_abandoned: int,
--   total_task_hours: float,
--
--   // Emotion metrics
--   avg_valence: float,
--   avg_arousal: float,
--   dominant_quadrant: string,
--
--   // Goal metrics
--   habits_met: int,
--   habits_missed: int,
--   streak_days_added: int,
--
--   // Computed at
--   computed_at: datetime
-- }

-- =============================================================================
-- USER PREFERENCES UPDATE
-- =============================================================================
-- Note: SurrealDB is schemaless, so we just document the expected fields.
-- The preferences field will be added to user records on first update.

-- Expected preferences structure on users table:
-- {
--   preferences: {
--     daily_retro: {
--       enabled: bool,          // true = auto-generate daily retros
--       time: string,           // "21:00" (24h format)
--       timezone: string,       // "Asia/Kolkata"
--       notification_enabled: bool,
--       auto_generate: bool     // Generate even if user doesn't open app
--     },
--     weekly_retro_day: string,  // "sunday"
--     monthly_retro_day: int,    // 1-31
--     timezone: string           // Default timezone for user
--   }
-- }

-- =============================================================================
-- END MIGRATION
-- =============================================================================
