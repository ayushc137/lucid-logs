-- Migration: 004_relax_permissions
-- Description: Allow the Rust API (service account) to read/write task data
-- Created: 2025-11-27
--
-- The Axum service connects to SurrealDB using the root service account and
-- enforces multi-tenancy at the application layer (via created_by filters).
-- The previous `$auth`-based permissions blocked all writes because `$auth.id`
-- never matched the user IDs stored in `created_by`. We relax the permissions
-- to FULL so the service can manage data while business rules remain in code.

DEFINE TABLE categories
  PERMISSIONS FULL;

DEFINE TABLE tasks
  PERMISSIONS FULL;

