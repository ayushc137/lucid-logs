//! {{feature_name_pascal}} repository for database operations
//!
//! Uses SurrealDB's fluent builder patterns and type-safe ID wrappers
//! for maintainable and safe database operations.
//!
//! # Extending This Repository
//!
//! 1. Add any reusable SurrealDB functions you need (e.g. `fn::{{feature_name}}::count_for_user`)
//! 2. Add a type-safe ID wrapper in `shared/db/types.rs` if the table will be reused elsewhere
//! 3. Prefer the builder API for mutations; reserve raw SQL for complex reads

use chrono::{DateTime, Utc};
use serde::Serialize;
use surrealdb::{engine::remote::ws::Client, sql::Thing, Surreal};

use super::model::{Create{{feature_name_pascal}}Request, {{feature_name_pascal}}, Update{{feature_name_pascal}}Request};
use crate::core::error::AppError;

/// Database table name
const TABLE: &str = "{{table_name}}";
const COUNT_BY_USER_FN: &str = "RETURN fn::{{feature_name}}::count_for_user($user)";
const LIST_BY_USER_QUERY: &str = r#"
    SELECT * FROM {{table_name}}
    WHERE created_by = $user AND deleted_at = NONE
    ORDER BY created_at DESC
    LIMIT $limit START $offset
"#;

#[derive(Serialize)]
struct {{feature_name_pascal}}InsertPayload {
    name: String,
    description: Option<String>,
    created_by: String,
    updated_by: String,
    created_at: DateTime<Utc>,
    updated_at: DateTime<Utc>,
    deleted_at: Option<DateTime<Utc>>,
}

impl {{feature_name_pascal}}InsertPayload {
    fn new(name: String, description: Option<String>, user_id: &str) -> Self {
        let now = Utc::now();
        Self {
            name,
            description,
            created_by: user_id.to_string(),
            updated_by: user_id.to_string(),
            created_at: now,
            updated_at: now,
            deleted_at: None,
        }
    }
}

#[derive(Serialize)]
struct {{feature_name_pascal}}UpdatePayload {
    name: String,
    description: Option<String>,
    updated_by: String,
    updated_at: DateTime<Utc>,
}

impl {{feature_name_pascal}}UpdatePayload {
    fn new(item: {{feature_name_pascal}}, user_id: &str) -> Self {
        Self {
            name: item.name,
            description: item.description,
            updated_by: user_id.to_string(),
            updated_at: Utc::now(),
        }
    }
}

#[derive(Serialize)]
struct {{feature_name_pascal}}SoftDeletePayload {
    deleted_at: Option<DateTime<Utc>>,
    updated_by: String,
    updated_at: DateTime<Utc>,
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

    /// Convert to a SurrealDB Thing for builder APIs
    fn as_thing(&self) -> Thing {
        Thing::from((TABLE, self.0.as_str()))
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

        let payload = {{feature_name_pascal}}InsertPayload::new(name.clone(), description, user_id);

        let created: Vec<{{feature_name_pascal}}> = self
            .db
            .create(TABLE)
            .content(payload)
            .await?;

        created
            .into_iter()
            .next()
            .map(|item| {
                tracing::debug!(id = ?item.id, "{{feature_name}} created");
                item
            })
            .ok_or(AppError::Internal)
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
        let mut result = self
            .db
            .query(COUNT_BY_USER_FN)
            .query(LIST_BY_USER_QUERY)
            .bind(("user", user_id.to_string()))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        let total: i64 = result.take(0)?;
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

        let record: Option<{{feature_name_pascal}}> = self.db.select(record_id.as_thing()).await?;

        record
            .filter(|item| item._created_by == user_id && item.deleted_at.is_none())
            .ok_or(AppError::NotFound)
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
        let mut existing = self.find_by_id(id, user_id).await?;

        if let Some(name) = req.name {
            existing.name = name;
        }
        if let Some(description) = req.description {
            existing.description = Some(description);
        }

        let record_id = {{feature_name_pascal}}Id::new(id);

        let payload = {{feature_name_pascal}}UpdatePayload::new(existing, user_id);

        let updated: Option<{{feature_name_pascal}}> = self
            .db
            .update(record_id.as_thing())
            .merge(payload)
            .await?;

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
        let _ = self.find_by_id(id, user_id).await?;

        let record_id = {{feature_name_pascal}}Id::new(id);
        let now = Utc::now();
        let payload = {{feature_name_pascal}}SoftDeletePayload {
            deleted_at: Some(now),
            updated_by: user_id.to_string(),
            updated_at: now,
        };

        let deleted: Option<{{feature_name_pascal}}> = self
            .db
            .update(record_id.as_thing())
            .merge(payload)
            .await?;

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
