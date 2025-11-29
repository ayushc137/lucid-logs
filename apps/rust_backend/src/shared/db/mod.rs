//! Database utilities and abstractions for SurrealDB 2.x
//!
//! This module provides type-safe wrappers and utilities for working with SurrealDB:
//!
//! - **`types`**: Type-safe record ID wrappers for each table
//!
//! # SurrealDB Integration Patterns
//!
//! We use a layered approach to database access:
//!
//! 1. **Fluent Builders** (preferred for simple CRUD):
//!    - `db.create("table").content(payload)` - Insert with typed payload
//!    - `db.select(id.as_key())` - Select by ID using tuple key
//!    - `db.update(id.as_key()).merge(payload)` - Partial update
//!
//! 2. **Server-Side Functions** (for complex/reusable logic):
//!    - `fn::task::with_category($id)` - Get task with category populated
//!    - `fn::task::count_for_user($user)` - Count user's tasks
//!    - `fn::analytics::task_stats($user)` - Get completion statistics
//!
//! 3. **Raw Queries** (when builders don't support the operation):
//!    - Pagination with `LIMIT`/`START`
//!    - Complex `WHERE` clauses
//!    - `FETCH` for record links
//!
//! # Usage
//!
//! ```rust,ignore
//! use crate::shared::db::{TaskId, CategoryId};
//!
//! // Type-safe record IDs
//! let task_id = TaskId::new("abc123");
//! let category_id = CategoryId::new("work");
//!
//! // Use with fluent builders (tuple key for SurrealDB 2.x)
//! let task: Option<Task> = db.select(task_id.as_key()).await?;
//! let updated: Option<Task> = db.update(task_id.as_key()).merge(payload).await?;
//!
//! // Use with raw queries (full ID string)
//! let task: Vec<Task> = db
//!     .query("RETURN fn::task::with_category(type::thing($id))")
//!     .bind(("id", task_id.full_id()))
//!     .await?.take(0)?;
//! ```

mod logging;
pub mod types;

pub use logging::DbResultExt;
pub use types::{CategoryId, TaskId};
