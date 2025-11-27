-- Migration: 002_categories
-- Description: Categories table (schemaless)
-- Created: 2025-11-27

-- Schemaless table with permissions
DEFINE TABLE categories
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

-- Indexes for performance
DEFINE INDEX idx_categories_user_name ON TABLE categories COLUMNS created_by, name UNIQUE;
DEFINE INDEX idx_categories_user ON TABLE categories COLUMNS created_by;
