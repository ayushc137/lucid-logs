//! Generic repository base trait for CRUD operations.
//!
//! Provides a consistent interface for database operations across entities,
//! reducing boilerplate for new entities.

use async_trait::async_trait;
use serde::{de::DeserializeOwned, Serialize};

use crate::error::AppError;

/// Generic repository trait for CRUD operations.
///
/// This trait is designed to be implemented by new repository types.
/// See `TaskRepository` for an example of how to implement this pattern.
///
/// Implement this trait to get a consistent interface for basic database operations.
/// The trait is designed to work with any entity type that can be serialized/deserialized.
///
/// # Type Parameters
/// - `T`: The entity type (must implement Serialize, DeserializeOwned, Send, Sync)
/// - `CreateReq`: The request type for creating new entities
/// - `UpdateReq`: The request type for updating entities
/// - `Id`: The ID type (usually String)
///
/// # Example
/// ```ignore
/// use async_trait::async_trait;
/// use crate::repositories::base::Repository;
///
/// struct TaskRepository { /* ... */ }
///
/// #[async_trait]
/// impl Repository<Task, CreateTaskRequest, UpdateTaskRequest, String> for TaskRepository {
///     // implement methods...
/// }
/// ```
#[allow(dead_code)] // Available for implementing new repositories
#[async_trait]
pub trait Repository<T, CreateReq, UpdateReq, Id>: Send + Sync
where
    T: DeserializeOwned + Serialize + Send + Sync,
    CreateReq: Send + Sync,
    UpdateReq: Send + Sync,
    Id: Send + Sync,
{
    /// Create a new entity
    async fn create(&self, req: CreateReq, owner_id: &str) -> Result<T, AppError>;

    /// Find entity by ID (with ownership check)
    async fn find_by_id(&self, id: &Id, owner_id: &str) -> Result<T, AppError>;

    /// Find all entities for an owner with pagination
    async fn find_by_owner_paginated(
        &self,
        owner_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<T>, i64), AppError>;

    /// Update an entity (with ownership check)
    async fn update(&self, id: &Id, req: UpdateReq, owner_id: &str) -> Result<T, AppError>;

    /// Soft delete an entity (with ownership check)
    async fn delete(&self, id: &Id, owner_id: &str) -> Result<(), AppError>;
}

/// Helper to ensure SurrealDB record IDs have the table prefix
pub fn ensure_record_id(id: &str, table: &str) -> String {
    if id.contains(':') {
        id.to_string()
    } else {
        format!("{}:{}", table, id)
    }
}

/// Count result helper struct for aggregation queries
#[derive(Debug, serde::Deserialize)]
pub struct CountResult {
    pub count: i64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ensure_record_id_with_prefix() {
        assert_eq!(ensure_record_id("tasks:123", "tasks"), "tasks:123");
    }

    #[test]
    fn test_ensure_record_id_without_prefix() {
        assert_eq!(ensure_record_id("123", "tasks"), "tasks:123");
    }
}

