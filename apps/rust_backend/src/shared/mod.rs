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
//! use crate::shared::db::{TaskId, CategoryId, task_queries};
//!
//! // Type-safe IDs
//! let task_id = TaskId::new("abc123");
//!
//! // Centralized queries
//! db.query(task_queries::SELECT_BY_ID)
//!     .bind(("id", task_id.full_id()))
//!     .await?;
//! ```

pub mod db;
pub mod pagination;
pub mod repository;

// Re-export commonly used types
pub use db::{category_queries, task_queries, CategoryId, TaskId};
