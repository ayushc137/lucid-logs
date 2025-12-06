-- Migration: 002_task_emotions
-- Description: Edge table for emotion analytics
-- 
-- Run: surreal sql -e http://localhost:8000 -u root -p root -ns daily_journal -db core < db/migrations/002_task_emotions.surql

-- -----------------------------------------------------------------------------
-- TASK_EMOTIONS (Edge table)
-- -----------------------------------------------------------------------------
-- Links tasks to emotions for graph queries:
--   SELECT ->task_emotions->emotions.* FROM tasks:abc
--   SELECT <-task_emotions<-tasks.* FROM emotions:E16
-- Types: primary (main emotion), positive (from positive items), negative (from negative items)

DEFINE TABLE task_emotions TYPE RELATION IN tasks OUT emotions PERMISSIONS FULL;
DEFINE FIELD type ON task_emotions TYPE string ASSERT $value IN ["primary", "positive", "negative"];
DEFINE FIELD text ON task_emotions TYPE option<string>;
DEFINE FIELD created_at ON task_emotions TYPE datetime DEFAULT time::now();
DEFINE INDEX idx_task_emotions_out ON TABLE task_emotions COLUMNS out;
DEFINE INDEX idx_task_emotions_type ON TABLE task_emotions COLUMNS type;

