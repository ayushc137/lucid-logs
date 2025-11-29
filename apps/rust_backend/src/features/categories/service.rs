//! Category service

use async_trait::async_trait;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use super::model::{Category, CreateCategoryRequest, UpdateCategoryRequest};
use super::repository::CategoryRepository;
use crate::core::error::AppError;

/// Category service trait for dependency injection
#[async_trait]
pub trait CategoryService: Send + Sync {
    /// List categories for a user with pagination
    async fn list_categories(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Category>, i64), AppError>;

    /// Create a new category
    async fn create_category(
        &self,
        req: CreateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError>;

    /// Update an existing category
    async fn update_category(
        &self,
        id: &str,
        req: UpdateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError>;

    /// Delete a category (soft delete)
    async fn delete_category(&self, id: &str, user_id: &str) -> Result<(), AppError>;

    /// Get a category by ID
    async fn get_category(&self, id: &str, user_id: &str) -> Result<Category, AppError>;
}

/// Production implementation of CategoryService
#[derive(Clone)]
pub struct CategoryServiceImpl {
    repo: CategoryRepository,
}

impl CategoryServiceImpl {
    pub fn new(db: Surreal<Client>) -> Self {
        Self {
            repo: CategoryRepository::new(db),
        }
    }
}

#[async_trait]
impl CategoryService for CategoryServiceImpl {
    async fn list_categories(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Category>, i64), AppError> {
        self.repo
            .find_by_user_paginated(user_id, limit, offset)
            .await
    }

    async fn create_category(
        &self,
        req: CreateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError> {
        self.repo.create(req, user_id).await
    }

    async fn update_category(
        &self,
        id: &str,
        req: UpdateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError> {
        self.repo.update(id, req, user_id).await
    }

    async fn delete_category(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        self.repo.delete(id, user_id).await
    }

    async fn get_category(&self, id: &str, user_id: &str) -> Result<Category, AppError> {
        self.repo.find_by_id(id, user_id).await
    }
}
