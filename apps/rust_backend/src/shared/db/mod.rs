//! Database utilities and abstractions
//!
//! This module provides type-safe wrappers and utilities for working with SurrealDB:
//!
//! - **`types`**: Type-safe record ID wrappers for each table
//! - **`queries`**: Centralized query registry for maintainability
//! - **`transaction`**: Transaction support utilities
//! - **`result`**: Common result types for database operations
//!
//! # Usage
//!
//! ```rust,ignore
//! use crate::shared::db::{RecordId, TaskId, CategoryId, queries};
//!
//! // Type-safe record IDs
//! let task_id = TaskId::new("abc123");
//! let category_id = CategoryId::new("work");
//!
//! // Centralized queries
//! db.query(queries::tasks::LIST_BY_USER)
//!     .bind(("user", user_id))
//!     .await?;
//! ```

mod queries;
mod result;
pub mod types;

pub use queries::{categories as category_queries, tasks as task_queries};
pub use result::{CountResult, IdResult};
pub use types::{CategoryId, TaskId};

