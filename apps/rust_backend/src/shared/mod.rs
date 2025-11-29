//! Shared types and utilities
//!
//! Contains types that are used across multiple features:
//!
//! - **`db`**: Database utilities (types, queries, transactions)
//! - **`pagination`**: Pagination parameters
//! - **`repository`**: Repository base traits and helpers
//!
//! # Database Utilities
//!
//! The `db` module provides type-safe abstractions for SurrealDB:
//!
//! ```rust,ignore
//! use crate::shared::db::{TaskId, CategoryId};
//!
//! // Type-safe IDs
//! let task_id = TaskId::new("abc123");
//!
//! // Works seamlessly with SurrealDB builders
//! let record = db.select(task_id.as_thing()).await?;
//! ```

pub mod db;
pub mod pagination;
pub mod repository;

// Re-export commonly used types
pub use db::{CategoryId, TaskId};
