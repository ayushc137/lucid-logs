//! Category service implementation

use async_trait::async_trait;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use crate::error::AppError;
use crate::models::category::{Category, CreateCategoryRequest, UpdateCategoryRequest};
use crate::repositories::CategoryRepository;
use crate::services::traits::CategoryService;

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
        self.repo.find_by_user_paginated(user_id, limit, offset).await
    }

    async fn create_category(
        &self,
        req: CreateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError> {
        // Validation is handled by the model's Validate derive
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

