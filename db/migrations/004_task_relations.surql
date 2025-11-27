-- Migration: 004_task_relations
-- Description: Graph relations for tasks (category, future: tags, mood, etc.)
-- Created: 2025-11-27
--
-- Using SurrealDB graph relations instead of foreign keys for:
-- - Future many-to-many support
-- - Complex graph traversals and analytics
-- - Relationship metadata (when assigned, by whom, etc.)
--
-- Relation Pattern:
--   task -[in_category]-> category
--   
-- Query Examples:
--   Get task's category:     SELECT ->in_category->categories AS category FROM tasks:id
--   Get category's tasks:    SELECT <-in_category<-tasks AS tasks FROM categories:id
--   Find tasks by category:  SELECT <-in_category<-tasks FROM categories WHERE name = "Work"

-- Edge table: task -> category relationship
-- SCHEMAFULL ensures data integrity on the edge
DEFINE TABLE in_category SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR in.created_by = $auth.id
    FOR create WHERE $auth = NONE OR in.created_by = $auth.id
    FOR update WHERE $auth = NONE OR in.created_by = $auth.id
    FOR delete WHERE $auth = NONE OR in.created_by = $auth.id;

-- Edge metadata (optional but useful for analytics)
DEFINE FIELD in ON in_category TYPE record<tasks>;
DEFINE FIELD out ON in_category TYPE record<categories>;
DEFINE FIELD assigned_at ON in_category TYPE datetime VALUE time::now();
DEFINE FIELD assigned_by ON in_category TYPE option<string>;

-- Ensure a task can only be in one category (for now - remove for many-to-many)
-- This enforces one-to-many: one task -> one category
DEFINE INDEX idx_in_category_unique_task ON TABLE in_category COLUMNS in UNIQUE;

-- Index for reverse lookups (find all tasks in a category)
DEFINE INDEX idx_in_category_out ON TABLE in_category COLUMNS out;

