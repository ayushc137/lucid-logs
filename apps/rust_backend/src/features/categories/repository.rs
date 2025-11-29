//! Category repository for database operations
//!
//! Uses SurrealDB's fluent builders and typed helpers instead of
//! raw SQL whenever possible.
//!
//! # SurrealDB 2.x Patterns Used
//!
//! - **Fluent builders**: `db.create()`, `db.select()`, `db.update().merge()`
//! - **Server-side functions**: `fn::category::count_for_user()` for counts
//! - **Typed payloads**: Serde structs for insert/update content
//! - **Tuple keys**: `(table, id)` for select/update operations

use chrono::{DateTime, Utc};
use rand::Rng;
use serde::{Deserialize, Serialize};
use surrealdb::{engine::remote::ws::Client, Surreal};

use super::model::{Category, CreateCategoryRequest, UpdateCategoryRequest};
use crate::core::error::AppError;
use crate::shared::db::DbResultExt;
use crate::shared::CategoryId;

/// Server-side function to count categories for a user
const COUNT_CATEGORIES_FN: &str = "RETURN fn::category::count_for_user($user)";

/// Raw query for paginated listing (fluent builders don't support LIMIT/START)
const LIST_BY_USER_QUERY: &str = r"
    SELECT id, name, color, created_at, updated_at, deleted_at, created_by, updated_by
    FROM categories
    WHERE created_by = $user AND deleted_at = NONE
    ORDER BY name ASC
    LIMIT $limit START $offset
";


/// Wrapper for extracting scalar count from server-side function
#[derive(Debug, Deserialize)]
struct CountWrapper(i64);

#[derive(Serialize)]
struct CategoryInsertPayload {
    name: String,
    color: String,
    created_by: String,
    updated_by: String,
    created_at: DateTime<Utc>,
    updated_at: DateTime<Utc>,
    deleted_at: Option<DateTime<Utc>>,
}

impl CategoryInsertPayload {
    fn new(name: String, color: String, user_id: &str) -> Self {
        let now = Utc::now();
        Self {
            name,
            color,
            created_by: user_id.to_string(),
            updated_by: user_id.to_string(),
            created_at: now,
            updated_at: now,
            deleted_at: None,
        }
    }
}

#[derive(Serialize)]
struct CategoryUpdatePayload {
    name: String,
    color: String,
    updated_by: String,
    updated_at: DateTime<Utc>,
}

impl CategoryUpdatePayload {
    fn new(category: Category, user_id: &str) -> Self {
        Self {
            name: category.name,
            color: category.color,
            updated_by: user_id.to_string(),
            updated_at: Utc::now(),
        }
    }
}

#[derive(Serialize)]
struct CategorySoftDeletePayload {
    deleted_at: Option<DateTime<Utc>>,
    updated_by: String,
    updated_at: DateTime<Utc>,
}

/// Category repository for database operations
///
/// Handles all category CRUD operations with ownership enforcement.
/// All queries filter by `created_by` to ensure data isolation.
#[derive(Clone)]
pub struct CategoryRepository {
    db: Surreal<Client>,
}

impl CategoryRepository {
    /// Create a new category repository instance
    pub fn new(db: Surreal<Client>) -> Self {
        Self { db }
    }

    /// Create a new category
    ///
    /// Uses SurrealDB's fluent `create().content()` builder for type-safe insertion.
    ///
    /// # Arguments
    /// - `req`: Category creation request
    /// - `user_id`: ID of the user creating the category
    ///
    /// # Returns
    /// The newly created category
    ///
    /// # Errors
    /// - `BadRequest` if a category with this name already exists for the user
    pub async fn create(
        &self,
        req: CreateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError> {
        let CreateCategoryRequest { name, color } = req;

        tracing::debug!(name = %name, "creating category");

        let payload = CategoryInsertPayload::new(name.clone(), color, user_id);
        let category_id = generate_category_id();

        self.db
            .create::<Option<Category>>(category_id.as_key())
            .content(payload)
            .await
            .log_db_err(|err| tracing::error!(?err, %category_id, user_id, "category create failed"))
            .map_err(map_category_error)?;

        let category = self.find_by_id(&category_id.full_id(), user_id).await?;
        tracing::debug!(category_id = ?category.id, "category created");
        Ok(category)
    }

    /// List categories for a user with pagination
    ///
    /// Uses server-side function `fn::category::count_for_user()` for efficient counting
    /// and raw query for pagination (fluent builders don't support LIMIT/START).
    ///
    /// # Arguments
    /// - `user_id`: User ID for ownership filter
    /// - `limit`: Maximum number of categories to return
    /// - `offset`: Number of categories to skip
    ///
    /// # Returns
    /// Tuple of (categories, total_count)
    pub async fn find_by_user_paginated(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Category>, i64), AppError> {
        // Batch both queries in a single round-trip
        let mut result = self
            .db
            .query(COUNT_CATEGORIES_FN)
            .query(LIST_BY_USER_QUERY)
            .bind(("user", user_id.to_string()))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        // Server-side function returns scalar directly
        let total: Option<CountWrapper> = result.take(0)?;
        let total = total.map(|w| w.0).unwrap_or(0);

        let categories: Vec<Category> = result.take(1)?;

        Ok((categories, total))
    }

    /// Get a category by ID
    ///
    /// Uses SurrealDB's fluent `select()` builder with tuple key.
    ///
    /// # Arguments
    /// - `id`: Category ID (with or without table prefix)
    /// - `user_id`: User ID for ownership verification
    ///
    /// # Returns
    /// The category if found and owned by user, NotFound otherwise
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<Category, AppError> {
        let category_id = CategoryId::new(id);
        // SurrealDB 2.x: use tuple (table, id) for select
        let record: Option<Category> = self.db.select(category_id.as_key()).await?;

        record
            .filter(|category| category._created_by == user_id && category.deleted_at.is_none())
            .ok_or(AppError::NotFound)
    }

    /// Update a category (enforces ownership)
    ///
    /// Uses SurrealDB's fluent `update().merge()` builder for partial updates.
    /// Only provided fields are updated; others remain unchanged.
    /// Ownership is verified before the update.
    ///
    /// # Arguments
    /// - `id`: Category ID to update
    /// - `req`: Update request with optional field values
    /// - `user_id`: User ID for ownership verification
    ///
    /// # Errors
    /// - `NotFound` if category doesn't exist or isn't owned by user
    /// - `BadRequest` if new name conflicts with existing category
    pub async fn update(
        &self,
        id: &str,
        req: UpdateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError> {
        let mut existing = self.find_by_id(id, user_id).await?;

        if let Some(name) = req.name {
            existing.name = name;
        }
        if let Some(color) = req.color {
            existing.color = color;
        }

        let category_id = CategoryId::new(id);

        let payload = CategoryUpdatePayload::new(existing, user_id);

        // SurrealDB 2.x: use tuple (table, id) for update
        let updated: Option<Category> = self
            .db
            .update(category_id.as_key())
            .merge(payload)
            .await
            .log_db_err(|err| tracing::error!(?err, %category_id, user_id, "category update failed"))
            .map_err(map_category_error)?;

        updated.ok_or(AppError::NotFound)
    }

    /// Soft-delete a category
    ///
    /// Uses SurrealDB's fluent `update().merge()` to set `deleted_at` timestamp
    /// instead of actually removing the record.
    /// Ownership is verified as part of the update.
    ///
    /// # Arguments
    /// - `id`: Category ID to delete
    /// - `user_id`: User ID for ownership verification
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        // Verify ownership
        let _ = self.find_by_id(id, user_id).await?;

        let category_id = CategoryId::new(id);
        let now = Utc::now();
        let payload = CategorySoftDeletePayload {
            deleted_at: Some(now),
            updated_by: user_id.to_string(),
            updated_at: now,
        };

        // SurrealDB 2.x: use tuple (table, id) for update
        let deleted: Option<Category> = self
            .db
            .update(category_id.as_key())
            .merge(payload)
            .await
            .log_db_err(|err| tracing::error!(?err, %category_id, user_id, "category delete failed"))?;

        if deleted.is_none() {
            return Err(AppError::NotFound);
        }

        tracing::info!(category_id = %category_id, "category soft-deleted");
        Ok(())
    }

}

fn map_category_error(err: surrealdb::Error) -> AppError {
    let message = err.to_string();
    if message.contains("idx_categories_user_name") {
        AppError::BadRequest("A category with this name already exists".into())
    } else {
        tracing::error!(error = %message, "category repository DB error");
        AppError::Internal
    }
}

fn generate_category_id() -> CategoryId {
    let mut rng = rand::rng();
    let value: u128 = rng.random();
    CategoryId::new(format!("{value:032x}"))
}
