-- Migration: 005_composite_indexes
-- Description: Add optimized composite indexes for common query patterns
-- 
-- This migration adds composite indexes that cover the most frequent queries:
-- 1. User's tasks by date (primary list query)
-- 2. User's active records (excludes soft-deleted)
-- 3. User's tasks by completion status
--
-- These indexes improve query performance by allowing the database to
-- satisfy the full WHERE clause from a single index scan.

-- -----------------------------------------------------------------------------
-- CATEGORIES: Composite indexes
-- -----------------------------------------------------------------------------

-- Active categories for a user (excludes soft-deleted)
-- Optimizes: WHERE created_by = $user AND deleted_at = NONE
DEFINE INDEX IF NOT EXISTS idx_categories_user_active 
  ON TABLE categories 
  COLUMNS created_by, deleted_at;

-- -----------------------------------------------------------------------------
-- TASKS: Composite indexes
-- -----------------------------------------------------------------------------

-- User's tasks ordered by date (most common query pattern)
-- Optimizes: WHERE created_by = $user ORDER BY start_date
DEFINE INDEX IF NOT EXISTS idx_tasks_user_date 
  ON TABLE tasks 
  COLUMNS created_by, start_date;

-- Active tasks for a user (excludes soft-deleted)
-- Optimizes: WHERE created_by = $user AND deleted_at = NONE
DEFINE INDEX IF NOT EXISTS idx_tasks_user_active 
  ON TABLE tasks 
  COLUMNS created_by, deleted_at;

-- User's tasks by completion status
-- Optimizes: WHERE created_by = $user AND completed = true/false
DEFINE INDEX IF NOT EXISTS idx_tasks_user_completed 
  ON TABLE tasks 
  COLUMNS created_by, completed;

-- -----------------------------------------------------------------------------
-- DATABASE FUNCTIONS
-- -----------------------------------------------------------------------------

-- Get a task with its category populated
DEFINE FUNCTION IF NOT EXISTS fn::task::with_category($id: record<tasks>) {
    RETURN SELECT *, category.* FROM $id FETCH category
};

-- Count tasks for a user
DEFINE FUNCTION IF NOT EXISTS fn::task::count_for_user($user_id: string) {
    RETURN (SELECT count() FROM tasks WHERE created_by = $user_id AND deleted_at = NONE GROUP ALL)[0].count OR 0
};

-- Count categories for a user
DEFINE FUNCTION IF NOT EXISTS fn::category::count_for_user($user_id: string) {
    RETURN (SELECT count() FROM categories WHERE created_by = $user_id AND deleted_at = NONE GROUP ALL)[0].count OR 0
};

