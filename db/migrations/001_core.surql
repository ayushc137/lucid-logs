-- Migration: 001_core
-- Description: Core tables - users, categories, tasks
-- 
-- Run: surreal sql -e http://localhost:8000 -u root -p root -ns lucid -db lucid < db/migrations/001_core.surql

-- -----------------------------------------------------------------------------
-- USERS (SCHEMAFULL - fields must be defined)
-- -----------------------------------------------------------------------------
-- Use IF NOT EXISTS to preserve existing data - migrations may run multiple times
DEFINE TABLE IF NOT EXISTS users SCHEMAFULL PERMISSIONS FULL;
DEFINE FIELD IF NOT EXISTS email ON users TYPE string ASSERT string::is::email($value);
DEFINE FIELD IF NOT EXISTS pass ON users TYPE string;
DEFINE FIELD IF NOT EXISTS is_admin ON users TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS created_at ON users TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON users TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_users_email ON TABLE users COLUMNS email UNIQUE;

-- -----------------------------------------------------------------------------
-- CATEGORIES (Schemaless)
-- -----------------------------------------------------------------------------
DEFINE TABLE IF NOT EXISTS categories PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_categories_user_name ON TABLE categories COLUMNS created_by, name UNIQUE;
DEFINE INDEX IF NOT EXISTS idx_categories_user_active ON TABLE categories COLUMNS created_by, deleted_at;

-- -----------------------------------------------------------------------------
-- TASKS (Schemaless with category link)
-- -----------------------------------------------------------------------------
DEFINE TABLE IF NOT EXISTS tasks PERMISSIONS FULL;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_date ON TABLE tasks COLUMNS created_by, start_date;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_active ON TABLE tasks COLUMNS created_by, deleted_at;
DEFINE INDEX IF NOT EXISTS idx_tasks_user_completed ON TABLE tasks COLUMNS created_by, completed;
DEFINE INDEX IF NOT EXISTS idx_tasks_category ON TABLE tasks COLUMNS category;


