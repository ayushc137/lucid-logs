-- Migration: 000_init
-- Description: Initial schema setup - migration tracking table
-- Created: 2025-11-27
-- 
-- This migration creates the infrastructure for tracking migrations.
-- It MUST be the first migration applied.

-- Migration tracking table
-- Records which migrations have been applied to this database
DEFINE TABLE _migrations SCHEMAFULL;

DEFINE FIELD version ON _migrations TYPE int;
DEFINE FIELD name ON _migrations TYPE string;
DEFINE FIELD applied_at ON _migrations TYPE datetime VALUE time::now();
DEFINE FIELD checksum ON _migrations TYPE option<string>;

-- Unique constraint on version to prevent duplicate migrations
DEFINE INDEX idx_migrations_version ON TABLE _migrations COLUMNS version UNIQUE;

