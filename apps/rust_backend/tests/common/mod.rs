//! Shared test utilities and helpers
//!
//! This module provides common functionality for integration tests:
//! - Mock service implementations
//! - Test fixtures and factories
//! - Database setup/teardown helpers

#![allow(clippy::unwrap_used)]

use std::sync::Arc;

use async_trait::async_trait;
use chrono::{Duration, Utc};
use parking_lot::Mutex;

use rust_backend::features::auth::model::{AuthRequest, AuthResponse};
use rust_backend::features::tasks::model::{
    CreateTaskRequest, DateTimeInput, Task, UpdateTaskRequest,
};
use rust_backend::{AuthService, TaskService};

// Re-export AppError from the crate
pub use rust_backend::core::error::AppError;

/// Mock implementation of TaskService for testing
#[derive(Default)]
pub struct MockTaskService {
    pub tasks: Arc<Mutex<Vec<Task>>>,
    pub should_fail: Arc<Mutex<bool>>,
}

impl MockTaskService {
    pub fn new() -> Self {
        Self::default()
    }

    #[allow(dead_code)]
    pub fn with_tasks(tasks: Vec<Task>) -> Self {
        Self {
            tasks: Arc::new(Mutex::new(tasks)),
            should_fail: Arc::new(Mutex::new(false)),
        }
    }

    #[allow(dead_code)]
    pub fn set_should_fail(&self, fail: bool) {
        *self.should_fail.lock() = fail;
    }
}

#[async_trait]
impl TaskService for MockTaskService {
    async fn list_tasks(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Task>, i64), AppError> {
        if *self.should_fail.lock() {
            return Err(AppError::Internal);
        }

        let tasks = self.tasks.lock();
        let filtered: Vec<_> = tasks
            .iter()
            .filter(|t| t._created_by == user_id)
            .cloned()
            .collect();

        let total = filtered.len() as i64;
        let offset = offset as usize;
        let limit = limit as usize;

        let paginated = filtered.into_iter().skip(offset).take(limit).collect();

        Ok((paginated, total))
    }

    async fn create_task(&self, req: CreateTaskRequest, user_id: &str) -> Result<Task, AppError> {
        if *self.should_fail.lock() {
            return Err(AppError::Internal);
        }

        let task = Task {
            id: Some(format!("tasks:{}", uuid_v4())),
            title: req.title,
            journal: req.journal,
            start_date: req.start_date.time_value(),
            end_date: req.end_date.time_value(),
            completed: false,
            priority: req.priority,
            source: req.source.unwrap_or_else(|| "manual".to_string()),
            note: req.note.unwrap_or_default(),
            positives: req.positives.unwrap_or_default(),
            negatives: req.negatives.unwrap_or_default(),
            category: None,
            created_at: Utc::now(),
            updated_at: Utc::now(),
            deleted_at: None,
            _created_by: user_id.to_string(),
            _updated_by: user_id.to_string(),
        };

        self.tasks.lock().push(task.clone());
        Ok(task)
    }

    async fn update_task(
        &self,
        id: &str,
        req: UpdateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        if *self.should_fail.lock() {
            return Err(AppError::Internal);
        }

        let mut tasks = self.tasks.lock();
        let task = tasks
            .iter_mut()
            .find(|t| t.id.as_deref() == Some(id) && t._created_by == user_id)
            .ok_or(AppError::NotFound)?;

        if let Some(title) = req.title {
            task.title = title;
        }
        if let Some(journal) = req.journal {
            task.journal = journal;
        }
        if let Some(completed) = req.completed {
            task.completed = completed;
        }
        if let Some(priority) = req.priority {
            task.priority = priority;
        }

        task.updated_at = Utc::now();
        task._updated_by = user_id.to_string();

        Ok(task.clone())
    }

    async fn delete_task(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        if *self.should_fail.lock() {
            return Err(AppError::Internal);
        }

        let mut tasks = self.tasks.lock();
        let pos = tasks
            .iter()
            .position(|t| t.id.as_deref() == Some(id) && t._created_by == user_id)
            .ok_or(AppError::NotFound)?;

        tasks.remove(pos);
        Ok(())
    }

    async fn get_task(&self, id: &str, user_id: &str) -> Result<Task, AppError> {
        if *self.should_fail.lock() {
            return Err(AppError::Internal);
        }

        let tasks = self.tasks.lock();
        tasks
            .iter()
            .find(|t| t.id.as_deref() == Some(id) && t._created_by == user_id)
            .cloned()
            .ok_or(AppError::NotFound)
    }
}

/// Mock implementation of AuthService for testing
#[derive(Default)]
pub struct MockAuthService {
    pub should_fail: Arc<Mutex<bool>>,
    pub users: Arc<Mutex<Vec<(String, String)>>>, // (username, password)
}

impl MockAuthService {
    pub fn new() -> Self {
        Self::default()
    }

    #[allow(dead_code)]
    pub fn with_user(username: &str, password: &str) -> Self {
        let service = Self::new();
        service
            .users
            .lock()
            .push((username.to_string(), password.to_string()));
        service
    }
}

#[async_trait]
impl AuthService for MockAuthService {
    async fn login(&self, req: AuthRequest) -> Result<AuthResponse, AppError> {
        if *self.should_fail.lock() {
            return Err(AppError::Internal);
        }

        let users = self.users.lock();
        let valid = users
            .iter()
            .any(|(u, p)| u == &req.username && p == &req.password);

        if valid {
            Ok(AuthResponse {
                token: format!("mock_token_{}", req.username),
                user: format!("user:{}", uuid_v4()),
            })
        } else {
            Err(AppError::Unauthorized("Invalid credentials".into()))
        }
    }

    async fn register(&self, req: AuthRequest) -> Result<AuthResponse, AppError> {
        if *self.should_fail.lock() {
            return Err(AppError::Internal);
        }

        let mut users = self.users.lock();
        if users.iter().any(|(u, _)| u == &req.username) {
            return Err(AppError::BadRequest("User already exists".into()));
        }

        users.push((req.username.clone(), req.password));

        Ok(AuthResponse {
            token: format!("mock_token_{}", req.username),
            user: format!("user:{}", uuid_v4()),
        })
    }
}

/// Generate a simple UUID v4 for testing
fn uuid_v4() -> String {
    use std::time::{SystemTime, UNIX_EPOCH};
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_nanos();
    format!("{:x}", now)
}

/// Test fixture builder for creating test data
pub mod fixtures {
    use super::*;

    /// Create a sample CreateTaskRequest for testing
    #[allow(dead_code)]
    pub fn create_task_request(title: &str) -> CreateTaskRequest {
        let now = Utc::now();
        CreateTaskRequest {
            title: title.to_string(),
            journal: "Test journal".to_string(),
            start_date: DateTimeInput(now),
            end_date: DateTimeInput(now + Duration::hours(1)),
            priority: 1,
            source: None,
            note: None,
            positives: None,
            negatives: None,
            category_id: None,
        }
    }

    /// Create a sample Task for testing
    #[allow(dead_code)]
    pub fn task(id: &str, title: &str, user_id: &str) -> Task {
        let now = Utc::now();
        Task {
            id: Some(id.to_string()),
            title: title.to_string(),
            journal: "Test journal".to_string(),
            start_date: now,
            end_date: now + Duration::hours(1),
            completed: false,
            priority: 1,
            source: "manual".to_string(),
            note: String::new(),
            positives: vec![],
            negatives: vec![],
            category: None,
            created_at: now,
            updated_at: now,
            deleted_at: None,
            _created_by: user_id.to_string(),
            _updated_by: user_id.to_string(),
        }
    }
}
