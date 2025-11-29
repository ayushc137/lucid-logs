//! Core infrastructure module
//!
//! Contains shared infrastructure used across all features:
//! - Configuration loading
//! - Error types and handling
//! - Database utilities
//! - Middleware (authentication, etc.)

pub mod config;
pub mod db;
pub mod error;
pub mod middleware;

// Re-export commonly used types
pub use config::Settings;
pub use db::{init_schema, resolve_migrations_dir, MigrationRunner, SchemaInitOptions};
