-- Migration: 002_categories
-- Description: Categories table for task organization
-- Created: 2025-11-27

-- Categories table (user-owned lookup)
-- Each user can have their own set of categories
DEFINE TABLE categories SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

-- Category fields
DEFINE FIELD name ON categories TYPE string ASSERT string::len(string::trim($value)) > 0;
DEFINE FIELD color ON categories TYPE string ASSERT string::len(string::trim($value)) > 0;

-- Audit fields
DEFINE FIELD created_at ON categories TYPE option<datetime> VALUE $value OR time::now();
DEFINE FIELD updated_at ON categories TYPE option<datetime> VALUE time::now();
DEFINE FIELD deleted_at ON categories TYPE option<datetime> VALUE $value ?? NONE;
DEFINE FIELD created_by ON categories TYPE string ASSERT string::len($value) > 0;
DEFINE FIELD updated_by ON categories TYPE string ASSERT string::len($value) > 0;

-- Indexes
-- Unique constraint: name per user (allows same name for different users)
DEFINE INDEX idx_categories_user_name ON TABLE categories COLUMNS created_by, name UNIQUE;
DEFINE INDEX idx_categories_user ON TABLE categories COLUMNS created_by;

