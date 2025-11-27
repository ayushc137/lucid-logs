//! {{feature_name_pascal}} repository
//!
//! Handles all database operations for {{feature_name}}s.
//! This layer is responsible for:
//! - CRUD operations
//! - Query building
//! - Data mapping (DB format ↔ Domain models)
//!
//! # Note
//! This layer should NOT contain business logic - that belongs in the service layer.

use chrono::Utc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use crate::error::AppError;
use crate::models::{{feature_name}}::{Create{{feature_name_pascal}}Request, {{feature_name_pascal}}, Update{{feature_name_pascal}}Request};
use crate::repositories::base::{ensure_record_id, CountResult};

/// Database table name for {{feature_name}}s
const TABLE: &str = "{{table_name}}";

/// Repository for {{feature_name}} database operations
pub struct {{feature_name_pascal}}Repository {
    /// SurrealDB connection
    /// 
    /// Note: Surreal<Client> is Clone and internally uses Arc,
    /// so cloning is cheap (just increments reference count).
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
    /// * `req` - The creation request containing {{feature_name}} data
    /// * `user_id` - ID of the user creating the {{feature_name}}
    /// 
    /// # Returns
    /// The created {{feature_name}} with generated ID and timestamps
    pub async fn create(
        &self,
        req: Create{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError> {
        let now = Utc::now();

        // Build the record data
        // SurrealDB will generate the ID automatically
        let data = serde_json::json!({
            "name": req.name,
            "description": req.description,
            "created_at": now,
            "updated_at": now,
            "deleted_at": null,
            "created_by": user_id,
            "updated_by": user_id,
        });

        // Insert into database
        // The ::<Option<{{feature_name_pascal}}>> is "turbofish" syntax - tells Rust what type to deserialize into
        let result: Option<{{feature_name_pascal}}> = self.db
            .create(TABLE)
            .content(data)
            .await?;

        result.ok_or(AppError::Internal)
    }

    /// Find a {{feature_name}} by ID
    /// 
    /// # Arguments
    /// * `id` - The {{feature_name}} ID (with or without table prefix)
    /// * `user_id` - ID of the requesting user (for ownership check)
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<{{feature_name_pascal}}, AppError> {
        // Ensure ID has table prefix (e.g., "{{table_name}}:abc123")
        let record_id = ensure_record_id(id, TABLE);

        // Query with ownership check
        let result: Option<{{feature_name_pascal}}> = self.db
            .query(
                "SELECT * FROM type::thing($table, $id) 
                 WHERE created_by = $user_id 
                 AND deleted_at IS NULL"
            )
            .bind(("table", TABLE))
            .bind(("id", &record_id))
            .bind(("user_id", user_id))
            .await?
            .take(0)?;

        result.ok_or(AppError::NotFound)
    }

    /// Find {{feature_name}}s by owner with pagination
    /// 
    /// # Returns
    /// A tuple of (items, total_count)
    pub async fn find_by_owner_paginated(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<{{feature_name_pascal}}>, i64), AppError> {
        // Get paginated results
        let items: Vec<{{feature_name_pascal}}> = self.db
            .query(
                "SELECT * FROM {{table_name}} 
                 WHERE created_by = $user_id 
                 AND deleted_at IS NULL 
                 ORDER BY created_at DESC 
                 LIMIT $limit START $offset"
            )
            .bind(("user_id", user_id))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?
            .take(0)?;

        // Get total count
        let count: Option<CountResult> = self.db
            .query(
                "SELECT count() as count FROM {{table_name}} 
                 WHERE created_by = $user_id 
                 AND deleted_at IS NULL 
                 GROUP ALL"
            )
            .bind(("user_id", user_id))
            .await?
            .take(0)?;

        let total = count.map(|c| c.count).unwrap_or(0);

        Ok((items, total))
    }

    /// Update a {{feature_name}}
    /// 
    /// Only updates fields that are provided (Some values).
    pub async fn update(
        &self,
        id: &str,
        req: Update{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError> {
        let record_id = ensure_record_id(id, TABLE);

        // First verify ownership
        let existing = self.find_by_id(&record_id, user_id).await?;

        // Build update data - only include provided fields
        let mut update_data = serde_json::Map::new();
        
        if let Some(name) = req.name {
            update_data.insert("name".to_string(), serde_json::json!(name));
        }
        if let Some(description) = req.description {
            update_data.insert("description".to_string(), serde_json::json!(description));
        }
        
        // Always update audit fields
        update_data.insert("updated_at".to_string(), serde_json::json!(Utc::now()));
        update_data.insert("updated_by".to_string(), serde_json::json!(user_id));

        // Perform update
        let result: Option<{{feature_name_pascal}}> = self.db
            .update((&record_id).into())
            .merge(serde_json::Value::Object(update_data))
            .await?;

        result.ok_or(AppError::Internal)
    }

    /// Soft delete a {{feature_name}}
    /// 
    /// Sets `deleted_at` timestamp instead of actually deleting.
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        let record_id = ensure_record_id(id, TABLE);

        // Verify ownership first
        let _ = self.find_by_id(&record_id, user_id).await?;

        // Soft delete by setting deleted_at
        let _: Option<{{feature_name_pascal}}> = self.db
            .update((&record_id).into())
            .merge(serde_json::json!({
                "deleted_at": Utc::now(),
                "updated_at": Utc::now(),
                "updated_by": user_id,
            }))
            .await?;

        Ok(())
    }
}

