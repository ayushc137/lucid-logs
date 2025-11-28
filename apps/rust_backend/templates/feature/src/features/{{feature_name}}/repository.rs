//! {{feature_name_pascal}} repository for database operations
//!
//! Uses the centralized query patterns and type-safe ID wrappers
//! for maintainable and safe database operations.
//!
//! # Extending This Repository
//!
//! 1. Add queries to `shared/db/queries.rs` under a new `{{feature_name}}` module
//! 2. Add a type-safe ID wrapper in `shared/db/types.rs` if needed
//! 3. Update this repository to use those queries

use chrono::Utc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use super::model::{Create{{feature_name_pascal}}Request, {{feature_name_pascal}}, Update{{feature_name_pascal}}Request};
use crate::core::error::AppError;
use crate::shared::db::{CountResult, IdResult};

/// Database table name
const TABLE: &str = "{{table_name}}";

// =============================================================================
// QUERIES
// =============================================================================
// 
// For a production feature, move these to `shared/db/queries.rs`
// under a `pub mod {{feature_name}}` module.

mod queries {
    pub const CREATE: &str = r#"
        INSERT INTO {{table_name}} (name, description, created_by, updated_by) 
        VALUES ($name, $description, $user, $user)
    "#;

    pub const SELECT_BY_ID: &str = r#"
        SELECT * FROM type::thing($id)
        WHERE created_by = $user AND deleted_at = NONE
    "#;

    pub const COUNT_BY_USER: &str = r#"
        SELECT count() FROM {{table_name}}
        WHERE created_by = $user AND deleted_at = NONE
        GROUP ALL
    "#;

    pub const LIST_BY_USER: &str = r#"
        SELECT * FROM {{table_name}}
        WHERE created_by = $user AND deleted_at = NONE
        ORDER BY created_at DESC
        LIMIT $limit START $offset
    "#;

    pub const UPDATE: &str = r#"
        UPDATE type::thing($id) SET
            name = $name,
            description = $description,
            updated_by = $user,
            updated_at = time::now()
        WHERE created_by = $user AND deleted_at = NONE
        RETURN *
    "#;

    pub const SOFT_DELETE: &str = r#"
        UPDATE type::thing($id) SET
            deleted_at = type::datetime($now),
            updated_by = $user,
            updated_at = time::now()
        WHERE created_by = $user AND deleted_at = NONE
        RETURN id
    "#;
}

// =============================================================================
// HELPER: Type-safe record ID
// =============================================================================
//
// For a production feature, add this to `shared/db/types.rs` using the
// `define_record_id!` macro.

/// Type-safe {{feature_name}} record ID
#[derive(Debug, Clone)]
struct {{feature_name_pascal}}Id(String);

impl {{feature_name_pascal}}Id {
    /// Create from ID (strips table prefix if present)
    fn new(id: impl Into<String>) -> Self {
        let id_str = id.into();
        let clean = id_str
            .strip_prefix(concat!("{{table_name}}", ":"))
            .map(String::from)
            .unwrap_or(id_str);
        Self(clean)
    }

    /// Get full record ID (table:id)
    fn full_id(&self) -> String {
        format!("{}:{}", TABLE, self.0)
    }
}

// =============================================================================
// REPOSITORY
// =============================================================================

/// Repository for {{feature_name}} database operations
///
/// Handles all {{feature_name}} CRUD operations with ownership enforcement.
/// All queries filter by `created_by` to ensure data isolation.
#[derive(Clone)]
pub struct {{feature_name_pascal}}Repository {
    db: Surreal<Client>,
}

impl {{feature_name_pascal}}Repository {
    /// Create a new repository instance
    pub fn new(db: Surreal<Client>) -> Self {
        Self { db }
    }

    /// Create a new {{feature_name}}
    ///
    /// # Arguments
    /// - `req`: Creation request with required fields
    /// - `user_id`: ID of the user creating the record
    ///
    /// # Returns
    /// The newly created {{feature_name}}
    pub async fn create(
        &self,
        req: Create{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError> {
        let Create{{feature_name_pascal}}Request { name, description } = req;

        tracing::debug!(name = %name, "creating {{feature_name}}");

        let mut result = self
            .db
            .query(queries::CREATE)
            .bind(("name", name.clone()))
            .bind(("description", description))
            .bind(("user", user_id.to_string()))
            .await?;

        // Check for query errors
        let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
        if !errors.is_empty() {
            tracing::error!(?errors, "{{feature_name}} create failed");
            return Err(AppError::Internal);
        }

        let created: Option<{{feature_name_pascal}}> = result.take(0)?;

        match created {
            Some(item) => {
                tracing::debug!(id = ?item.id, "{{feature_name}} created");
                Ok(item)
            },
            None => {
                tracing::error!("{{feature_name}} create returned no result");
                Err(AppError::Internal)
            },
        }
    }

    /// List {{feature_name}}s for a user with pagination
    ///
    /// Returns records ordered by created_at descending.
    /// Soft-deleted records are excluded.
    ///
    /// # Arguments
    /// - `user_id`: User ID for ownership filter
    /// - `limit`: Maximum number of records to return
    /// - `offset`: Number of records to skip
    ///
    /// # Returns
    /// Tuple of (items, total_count)
    pub async fn find_by_user_paginated(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<{{feature_name_pascal}}>, i64), AppError> {
        // Execute count and list queries in a single batch
        let mut result = self
            .db
            .query(queries::COUNT_BY_USER)
            .query(queries::LIST_BY_USER)
            .bind(("user", user_id.to_string()))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        // Extract count from first query
        let count_result: Option<CountResult> = result.take(0)?;
        let total = CountResult::unwrap_or_zero(count_result);

        // Extract items from second query
        let items: Vec<{{feature_name_pascal}}> = result.take(1)?;

        Ok((items, total))
    }

    /// Get a {{feature_name}} by ID
    ///
    /// # Arguments
    /// - `id`: Record ID (with or without table prefix)
    /// - `user_id`: User ID for ownership verification
    ///
    /// # Returns
    /// The record if found and owned by user, NotFound otherwise
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<{{feature_name_pascal}}, AppError> {
        let record_id = {{feature_name_pascal}}Id::new(id);

        let mut result = self
            .db
            .query(queries::SELECT_BY_ID)
            .bind(("id", record_id.full_id()))
            .bind(("user", user_id.to_string()))
            .await?;

        let items: Vec<{{feature_name_pascal}}> = result.take(0)?;
        items.into_iter().next().ok_or(AppError::NotFound)
    }

    /// Update a {{feature_name}} (enforces ownership)
    ///
    /// Only provided fields are updated; others remain unchanged.
    /// Ownership is verified before the update.
    ///
    /// # Arguments
    /// - `id`: Record ID to update
    /// - `req`: Update request with optional field values
    /// - `user_id`: User ID for ownership verification
    pub async fn update(
        &self,
        id: &str,
        req: Update{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError> {
        // Fetch existing (verifies ownership)
        let mut existing = self.find_by_id(id, user_id).await?;

        // Apply updates
        if let Some(name) = req.name {
            existing.name = name;
        }
        if let Some(description) = req.description {
            existing.description = Some(description);
        }

        let record_id = {{feature_name_pascal}}Id::new(id);

        let mut result = self
            .db
            .query(queries::UPDATE)
            .bind(("id", record_id.full_id()))
            .bind(("name", existing.name))
            .bind(("description", existing.description))
            .bind(("user", user_id.to_string()))
            .await?;

        let updated: Option<{{feature_name_pascal}}> = result.take(0)?;
        updated.ok_or(AppError::NotFound)
    }

    /// Soft-delete a {{feature_name}}
    ///
    /// Sets `deleted_at` timestamp instead of actually removing the record.
    /// Ownership is verified as part of the update.
    ///
    /// # Arguments
    /// - `id`: Record ID to delete
    /// - `user_id`: User ID for ownership verification
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        let record_id = {{feature_name_pascal}}Id::new(id);
        let now = Utc::now().to_rfc3339();

        let mut result = self
            .db
            .query(queries::SOFT_DELETE)
            .bind(("id", record_id.full_id()))
            .bind(("now", now))
            .bind(("user", user_id.to_string()))
            .await?;

        let deleted: Option<IdResult> = result.take(0)?;
        if deleted.is_none() {
            return Err(AppError::NotFound);
        }

        tracing::info!(id = %record_id.full_id(), "{{feature_name}} soft-deleted");
        Ok(())
    }
}

// =============================================================================
// TESTS
// =============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_record_id_handling() {
        let id = {{feature_name_pascal}}Id::new("abc123");
        assert_eq!(id.full_id(), "{{table_name}}:abc123");

        // With prefix already
        let id2 = {{feature_name_pascal}}Id::new("{{table_name}}:xyz789");
        assert_eq!(id2.full_id(), "{{table_name}}:xyz789");
    }
}
