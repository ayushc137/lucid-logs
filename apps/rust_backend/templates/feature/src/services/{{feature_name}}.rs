//! {{feature_name_pascal}} service implementation
//!
//! This module contains the business logic for {{feature_name}}s.
//! The service sits between handlers (HTTP layer) and repositories (data layer).
//!
//! # Responsibilities
//! - Business rule validation
//! - Orchestrating multiple repository calls
//! - Transforming data between layers

use async_trait::async_trait;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use crate::error::AppError;
use crate::models::{{feature_name}}::{Create{{feature_name_pascal}}Request, {{feature_name_pascal}}, Update{{feature_name_pascal}}Request};
use crate::repositories::{{feature_name}}::{{feature_name_pascal}}Repository;

// ============================================================
// TRAIT DEFINITION
// ============================================================
// Define this in services/traits.rs and import here

/// {{feature_name_pascal}} service trait
/// 
/// Defines the contract for {{feature_name}} business logic.
/// Having a trait allows us to:
/// - Mock the service in tests
/// - Swap implementations at runtime
/// - Decouple handlers from concrete implementations
#[async_trait]
pub trait {{feature_name_pascal}}Service: Send + Sync {
    /// List {{feature_name}}s for a user with pagination
    async fn list_{{feature_name}}s(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<{{feature_name_pascal}}>, i64), AppError>;

    /// Get a {{feature_name}} by ID
    async fn get_{{feature_name}}(&self, id: &str, user_id: &str) -> Result<{{feature_name_pascal}}, AppError>;

    /// Create a new {{feature_name}}
    async fn create_{{feature_name}}(
        &self,
        req: Create{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError>;

    /// Update an existing {{feature_name}}
    async fn update_{{feature_name}}(
        &self,
        id: &str,
        req: Update{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError>;

    /// Delete a {{feature_name}} (soft delete)
    async fn delete_{{feature_name}}(&self, id: &str, user_id: &str) -> Result<(), AppError>;
}

// ============================================================
// IMPLEMENTATION
// ============================================================

/// Production implementation of {{feature_name_pascal}}Service
/// 
/// This struct holds the repository instance needed for database operations.
pub struct {{feature_name_pascal}}ServiceImpl {
    repo: {{feature_name_pascal}}Repository,
}

impl {{feature_name_pascal}}ServiceImpl {
    /// Create a new service instance
    /// 
    /// # Arguments
    /// * `db` - SurrealDB connection (cloned for the repository)
    pub fn new(db: Surreal<Client>) -> Self {
        Self {
            repo: {{feature_name_pascal}}Repository::new(db),
        }
    }
}

#[async_trait]
impl {{feature_name_pascal}}Service for {{feature_name_pascal}}ServiceImpl {
    async fn list_{{feature_name}}s(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<{{feature_name_pascal}}>, i64), AppError> {
        // Business logic can go here
        // For example: filtering, sorting, access control checks
        
        self.repo.find_by_owner_paginated(user_id, limit, offset).await
    }

    async fn get_{{feature_name}}(&self, id: &str, user_id: &str) -> Result<{{feature_name_pascal}}, AppError> {
        self.repo.find_by_id(id, user_id).await
    }

    async fn create_{{feature_name}}(
        &self,
        req: Create{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError> {
        // Add business validation here
        // Example: Check for duplicates, enforce limits, etc.

        self.repo.create(req, user_id).await
    }

    async fn update_{{feature_name}}(
        &self,
        id: &str,
        req: Update{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError> {
        // Add business validation here
        // Example: Check permissions, validate state transitions

        self.repo.update(id, req, user_id).await
    }

    async fn delete_{{feature_name}}(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        // Add business logic here
        // Example: Check for dependencies before deleting

        self.repo.delete(id, user_id).await
    }
}

// ============================================================
// TESTS
// ============================================================

#[cfg(test)]
mod tests {
    use super::*;
    // Add your tests here
    // See tests/common/mod.rs for mock examples
}

