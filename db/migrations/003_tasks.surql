-- Migration: 003_tasks
-- Description: Tasks table (schemaless) with category record link
-- Created: 2025-11-27
--
-- Tasks store a direct record link to category:
--   task.category = categories:work123
--
-- Query with FETCH to hydrate:
--   SELECT *, category.* FROM tasks FETCH category

-- Schemaless table with permissions
DEFINE TABLE tasks
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

-- Indexes for performance
DEFINE INDEX idx_tasks_user ON TABLE tasks COLUMNS created_by;
DEFINE INDEX idx_tasks_time ON TABLE tasks COLUMNS start_date;
DEFINE INDEX idx_tasks_status ON TABLE tasks COLUMNS completed;
DEFINE INDEX idx_tasks_category ON TABLE tasks COLUMNS category;
