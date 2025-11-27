-- Migration: 003_tasks
-- Description: Tasks table - the core entity
-- Created: 2025-11-27

-- Tasks table (core entity)
-- Permissions: $auth is NONE for root sessions, otherwise check ownership
DEFINE TABLE tasks SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

-- Core task fields
DEFINE FIELD title ON tasks TYPE string ASSERT string::len(string::trim($value)) > 0;
DEFINE FIELD journal ON tasks TYPE option<string>;
DEFINE FIELD start_date ON tasks TYPE datetime ASSERT $value != NONE;
DEFINE FIELD end_date ON tasks TYPE datetime ASSERT $value >= start_date;
DEFINE FIELD completed ON tasks TYPE bool DEFAULT false;
DEFINE FIELD priority ON tasks TYPE number DEFAULT 0;
DEFINE FIELD source ON tasks TYPE string VALUE $value ?? "manual" ASSERT $value = "manual";
DEFINE FIELD note ON tasks TYPE option<string>;
DEFINE FIELD positives ON tasks TYPE array<string> DEFAULT [];
DEFINE FIELD negatives ON tasks TYPE array<string> DEFAULT [];

-- NOTE: category_id removed - using graph relations instead (see 004_task_relations.surql)

-- Audit fields
DEFINE FIELD created_at ON tasks TYPE option<datetime> VALUE $value OR time::now();
DEFINE FIELD updated_at ON tasks TYPE option<datetime> VALUE time::now();
DEFINE FIELD deleted_at ON tasks TYPE option<datetime> VALUE $value ?? NONE;
DEFINE FIELD created_by ON tasks TYPE string ASSERT string::len($value) > 0;
DEFINE FIELD updated_by ON tasks TYPE string ASSERT string::len($value) > 0;

-- Indexes for common queries
DEFINE INDEX idx_tasks_user ON TABLE tasks COLUMNS created_by;
DEFINE INDEX idx_tasks_time ON TABLE tasks COLUMNS start_date;
DEFINE INDEX idx_tasks_status ON TABLE tasks COLUMNS completed;
