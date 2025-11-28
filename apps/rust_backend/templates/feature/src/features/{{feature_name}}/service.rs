//! {{feature_name_pascal}} service

use async_trait::async_trait;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use super::model::{Create{{feature_name_pascal}}Request, {{feature_name_pascal}}, Update{{feature_name_pascal}}Request};
use super::repository::{{feature_name_pascal}}Repository;
use crate::core::error::AppError;

/// {{feature_name_pascal}} service trait for dependency injection
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

/// Production implementation of {{feature_name_pascal}}Service
#[derive(Clone)]
pub struct {{feature_name_pascal}}ServiceImpl {
    repo: {{feature_name_pascal}}Repository,
}

impl {{feature_name_pascal}}ServiceImpl {
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
        self.repo.find_by_user_paginated(user_id, limit, offset).await
    }

    async fn get_{{feature_name}}(&self, id: &str, user_id: &str) -> Result<{{feature_name_pascal}}, AppError> {
        self.repo.find_by_id(id, user_id).await
    }

    async fn create_{{feature_name}}(
        &self,
        req: Create{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError> {
        // Add business validation here if needed
        self.repo.create(req, user_id).await
    }

    async fn update_{{feature_name}}(
        &self,
        id: &str,
        req: Update{{feature_name_pascal}}Request,
        user_id: &str,
    ) -> Result<{{feature_name_pascal}}, AppError> {
        // Add business validation here if needed
        self.repo.update(id, req, user_id).await
    }

    async fn delete_{{feature_name}}(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        // Add business logic here if needed (e.g., check dependencies)
        self.repo.delete(id, user_id).await
    }
}

#[cfg(test)]
mod tests {
    // Add your tests here
}

