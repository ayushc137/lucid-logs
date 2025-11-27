//! Task service implementation

use async_trait::async_trait;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use crate::error::AppError;
use crate::models::task::{CreateTaskRequest, Task, UpdateTaskRequest};
use crate::repositories::TaskRepository;
use crate::services::traits::TaskService;

/// Production implementation of TaskService
#[derive(Clone)]
pub struct TaskServiceImpl {
    repo: TaskRepository,
}

impl TaskServiceImpl {
    pub fn new(db: Surreal<Client>) -> Self {
        Self {
            repo: TaskRepository::new(db),
        }
    }
}

#[async_trait]
impl TaskService for TaskServiceImpl {
    async fn list_tasks(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Task>, i64), AppError> {
        self.repo.find_by_user_paginated(user_id, limit, offset).await
    }

    async fn create_task(&self, req: CreateTaskRequest, user_id: &str) -> Result<Task, AppError> {
        // Validate dates
        if req.end_date.time_value() < req.start_date.time_value() {
            return Err(AppError::BadRequest(
                "end_date must be on or after start_date".to_string(),
            ));
        }

        self.repo.create(req, user_id).await
    }

    async fn update_task(
        &self,
        id: &str,
        req: UpdateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        // Validate dates if both are provided
        if let (Some(start), Some(end)) = (&req.start_date, &req.end_date) {
            if end.time_value() < start.time_value() {
                return Err(AppError::BadRequest(
                    "end_date must be on or after start_date".to_string(),
                ));
            }
        }

        self.repo.update(id, req, user_id).await
    }

    async fn delete_task(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        self.repo.delete(id, user_id).await
    }

    async fn get_task(&self, id: &str, user_id: &str) -> Result<Task, AppError> {
        self.repo.find_by_id(id, user_id).await
    }
}

