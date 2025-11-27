//! Category repository for database operations

use chrono::Utc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use super::base::{ensure_record_id, CountResult};
use crate::error::AppError;
use crate::models::category::{Category, CreateCategoryRequest, UpdateCategoryRequest};

/// Category projection fields for queries
const CATEGORY_PROJECTION: &str =
    "id, name, color, created_at, updated_at, deleted_at, created_by, updated_by";

/// Table name for categories (must match schema.surql)
const CATEGORIES_TABLE: &str = "categories";

#[derive(Clone)]
pub struct CategoryRepository {
    db: Surreal<Client>,
}

impl CategoryRepository {
    pub fn new(db: Surreal<Client>) -> Self {
        Self { db }
    }

    /// Create a new category
    pub async fn create(
        &self,
        req: CreateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError> {
        let CreateCategoryRequest { name, color } = req;
        let user_owned = user_id.to_string();

        // Use INSERT instead of CREATE for better error handling
        // INSERT returns the created record or throws constraint errors
        let sql = format!(
            "INSERT INTO {} (name, color, created_by, updated_by) VALUES ($name, $color, $user, $user)",
            CATEGORIES_TABLE
        );

        tracing::debug!(name = %name, "creating category in database");

        let mut result = self
            .db
            .query(&sql)
            .bind(("name", name.clone()))
            .bind(("color", color))
            .bind(("user", user_owned.clone()))
            .await?;

        // Check for errors first
        let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
        if !errors.is_empty() {
            let error_str = format!("{:?}", errors);
            tracing::error!(?errors, "database create failed with errors");
            if error_str.contains("idx_categories_user_name") {
                return Err(AppError::BadRequest(
                    "A category with this name already exists".to_string(),
                ));
            }
            return Err(AppError::Internal);
        }

        let created: Option<Category> = result.take(0)?;

        match created {
            Some(category) => {
                tracing::debug!(category_id = ?category.id, "category created successfully");
                Ok(category)
            }
            None => {
                tracing::error!(
                    "database create failed - no category returned and no errors reported"
                );
                // Try to fetch by name as fallback (in case INSERT worked but didn't return)
                tracing::debug!("attempting fallback: fetching recently created category by name");
                if let Ok(Some(cat)) = self.find_by_name(&name, &user_owned).await {
                    return Ok(cat);
                }
                Err(AppError::Internal)
            }
        }
    }

    /// List categories for a user with pagination (excludes soft-deleted)
    /// Returns (categories, total_count)
    pub async fn find_by_user_paginated(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Category>, i64), AppError> {
        let user_id_owned = user_id.to_string();

        // Get count and items in a single query batch
        let count_sql = format!(
            "SELECT count() FROM {} WHERE created_by = $user AND deleted_at = NONE GROUP ALL",
            CATEGORIES_TABLE
        );
        let items_sql = format!(
            "SELECT {} FROM {} WHERE created_by = $user AND deleted_at = NONE ORDER BY name ASC LIMIT $limit START $offset",
            CATEGORY_PROJECTION, CATEGORIES_TABLE
        );

        let mut result = self
            .db
            .query(count_sql)
            .query(items_sql)
            .bind(("user", user_id_owned))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        // Extract count from first query
        let count_result: Option<CountResult> = result.take(0)?;
        let total = count_result.map(|c| c.count).unwrap_or(0);

        // Extract categories from second query
        let categories: Vec<Category> = result.take(1)?;

        Ok((categories, total))
    }

    /// Get a category by ID (excludes soft-deleted, enforces ownership)
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<Category, AppError> {
        let record_id = ensure_record_id(id, CATEGORIES_TABLE);
        let sql = format!(
            "SELECT {} FROM {} WHERE id = type::thing($record) AND created_by = $user AND deleted_at = NONE",
            CATEGORY_PROJECTION, CATEGORIES_TABLE
        );

        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
            .bind(("user", user_id.to_string()))
            .await?;

        let categories: Vec<Category> = result.take(0)?;
        // Return NotFound for both non-existent and unauthorized (prevents ID enumeration)
        categories.into_iter().next().ok_or(AppError::NotFound)
    }

    /// Update a category (enforces ownership)
    pub async fn update(
        &self,
        id: &str,
        req: UpdateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError> {
        // Fetch existing category (ownership verified by find_by_id)
        let mut existing = self.find_by_id(id, user_id).await?;

        // Update fields if provided
        if let Some(name) = req.name {
            existing.name = name;
        }
        if let Some(color) = req.color {
            existing.color = color;
        }

        let record_id = ensure_record_id(id, CATEGORIES_TABLE);
        // Note: created_by is NOT updated - preserves original owner
        let sql = format!(
            "UPDATE type::thing($record) SET
                name = $name,
                color = $color,
                updated_by = $updated_by
             WHERE created_by = $user
             RETURN {}",
            CATEGORY_PROJECTION
        );

        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
            .bind(("name", existing.name))
            .bind(("color", existing.color))
            .bind(("updated_by", user_id.to_string()))
            .bind(("user", user_id.to_string()))
            .await?;

        let updated: Option<Category> = result.take(0)?;

        match updated {
            Some(category) => Ok(category),
            None => {
                // Check for unique constraint violation
                let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
                if !errors.is_empty() {
                    let error_str = format!("{:?}", errors);
                    if error_str.contains("idx_categories_user_name") {
                        return Err(AppError::BadRequest(
                            "A category with this name already exists".to_string(),
                        ));
                    }
                }
                Err(AppError::NotFound)
            }
        }
    }

    /// Soft-delete a category (sets deleted_at timestamp, enforces ownership)
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        let record_id = ensure_record_id(id, CATEGORIES_TABLE);
        let user_id_owned = user_id.to_string();
        let now = Utc::now().to_rfc3339();

        // Add ownership check to prevent deleting others' categories
        let sql = format!(
            "UPDATE type::thing($record) SET deleted_at = type::datetime($now), updated_by = $user WHERE created_by = $user AND deleted_at = NONE RETURN {}",
            CATEGORY_PROJECTION
        );
        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
            .bind(("now", now))
            .bind(("user", user_id_owned))
            .await?;

        let deleted: Option<Category> = result.take(0)?;
        if deleted.is_none() {
            // Return NotFound for both non-existent and unauthorized (prevents ID enumeration)
            return Err(AppError::NotFound);
        }

        tracing::info!(category_id = %id, "category soft-deleted successfully");
        Ok(())
    }

    /// Find a category by name for a specific user (excludes soft-deleted)
    #[allow(dead_code)]
    pub async fn find_by_name(&self, name: &str, user_id: &str) -> Result<Option<Category>, AppError> {
        let sql = format!(
            "SELECT {} FROM {} WHERE name = $name AND created_by = $user AND deleted_at = NONE",
            CATEGORY_PROJECTION, CATEGORIES_TABLE
        );

        let mut result = self
            .db
            .query(sql)
            .bind(("name", name.to_string()))
            .bind(("user", user_id.to_string()))
            .await?;

        let categories: Vec<Category> = result.take(0)?;
        Ok(categories.into_iter().next())
    }
}

