//! Category repository for database operations
//!
//! Uses the centralized query registry and type-safe ID wrappers
//! for maintainable and safe database operations.
//!
//! # Architecture
//!
//! - All queries are defined in `shared::db::queries::categories`
//! - Record IDs use type-safe `CategoryId` wrapper
//! - Unique constraint on (created_by, name) prevents duplicates per user

use chrono::Utc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use super::model::{Category, CreateCategoryRequest, UpdateCategoryRequest};
use crate::core::error::AppError;
use crate::shared::db::CountResult;
use crate::shared::{category_queries as queries, CategoryId};

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

        let mut result = self
            .db
            .query(queries::CREATE)
            .bind(("name", name.clone()))
            .bind(("color", color))
            .bind(("user", user_id.to_string()))
            .await?;

        // Check for constraint violations
        let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
        if !errors.is_empty() {
            let error_str = format!("{:?}", errors);
            tracing::error!(?errors, "category create failed");

            if error_str.contains("idx_categories_user_name") {
                return Err(AppError::BadRequest(
                    "A category with this name already exists".to_string(),
                ));
            }
            return Err(AppError::Internal);
        }

        // Try to get the created category
        let created: Option<Category> = result.take(0)?;

        match created {
            Some(category) => {
                tracing::debug!(category_id = ?category.id, "category created");
                Ok(category)
            },
            None => {
                // Fallback: fetch by name (handles race conditions)
                tracing::debug!("fallback: fetching created category by name");
                if let Ok(Some(cat)) = self.find_by_name(&name, user_id).await {
                    return Ok(cat);
                }
                Err(AppError::Internal)
            },
        }
    }

    /// List categories for a user with pagination
    ///
    /// Returns categories ordered by name ascending.
    /// Soft-deleted categories are excluded.
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

        // Extract categories from second query
        let categories: Vec<Category> = result.take(1)?;

        Ok((categories, total))
    }

    /// Get a category by ID
    ///
    /// # Arguments
    /// - `id`: Category ID (with or without table prefix)
    /// - `user_id`: User ID for ownership verification
    ///
    /// # Returns
    /// The category if found and owned by user, NotFound otherwise
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<Category, AppError> {
        let category_id = CategoryId::new(id);

        let mut result = self
            .db
            .query(queries::SELECT_BY_ID)
            .bind(("id", category_id.full_id()))
            .bind(("user", user_id.to_string()))
            .await?;

        let categories: Vec<Category> = result.take(0)?;
        categories.into_iter().next().ok_or(AppError::NotFound)
    }

    /// Update a category (enforces ownership)
    ///
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
        // Fetch existing category (verifies ownership)
        let mut existing = self.find_by_id(id, user_id).await?;

        // Apply updates
        if let Some(name) = req.name {
            existing.name = name;
        }
        if let Some(color) = req.color {
            existing.color = color;
        }

        let category_id = CategoryId::new(id);

        let mut result = self
            .db
            .query(queries::UPDATE)
            .bind(("id", category_id.full_id()))
            .bind(("name", existing.name))
            .bind(("color", existing.color))
            .bind(("user", user_id.to_string()))
            .await?;

        // Check for unique constraint violations
        let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
        if !errors.is_empty() {
            let error_str = format!("{:?}", errors);
            if error_str.contains("idx_categories_user_name") {
                return Err(AppError::BadRequest(
                    "A category with this name already exists".to_string(),
                ));
            }
        }

        let updated: Option<Category> = result.take(0)?;
        updated.ok_or(AppError::NotFound)
    }

    /// Soft-delete a category
    ///
    /// Sets `deleted_at` timestamp instead of actually removing the record.
    /// Ownership is verified as part of the update.
    ///
    /// # Arguments
    /// - `id`: Category ID to delete
    /// - `user_id`: User ID for ownership verification
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        let category_id = CategoryId::new(id);
        let now = Utc::now().to_rfc3339();

        let mut result = self
            .db
            .query(queries::SOFT_DELETE)
            .bind(("id", category_id.full_id()))
            .bind(("now", now))
            .bind(("user", user_id.to_string()))
            .await?;

        let deleted: Option<Category> = result.take(0)?;
        if deleted.is_none() {
            return Err(AppError::NotFound);
        }

        tracing::info!(category_id = %category_id, "category soft-deleted");
        Ok(())
    }

    /// Find a category by name for a specific user
    ///
    /// # Arguments
    /// - `name`: Category name to search for
    /// - `user_id`: User ID for ownership filter
    ///
    /// # Returns
    /// The category if found, None otherwise
    pub async fn find_by_name(
        &self,
        name: &str,
        user_id: &str,
    ) -> Result<Option<Category>, AppError> {
        let mut result = self
            .db
            .query(queries::SELECT_BY_NAME)
            .bind(("name", name.to_string()))
            .bind(("user", user_id.to_string()))
            .await?;

        let categories: Vec<Category> = result.take(0)?;
        Ok(categories.into_iter().next())
    }
}

