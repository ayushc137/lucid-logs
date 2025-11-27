//! Service Traits for Dependency Injection
//!
//! This module defines **traits** (similar to interfaces in other languages) that
//! describe what our services can do, without specifying HOW they do it.
//!
//! # For Rust Beginners
//!
//! ## What is a Trait?
//!
//! A trait is like a contract that says "any type implementing this trait MUST
//! have these methods". It's similar to:
//! - **Interfaces** in Java/TypeScript/Go
//! - **Protocols** in Swift
//! - **Abstract base classes** in Python
//!
//! ## Why Use Traits for Services?
//!
//! ```text
//! Without traits (hard to test):
//! ┌──────────┐         ┌──────────────────┐
//! │ Handler  │ ──────► │ TaskServiceImpl  │  ← Handler depends on concrete type
//! └──────────┘         └──────────────────┘    Can't swap for tests!
//!
//! With traits (easy to test):
//! ┌──────────┐         ┌──────────────┐
//! │ Handler  │ ──────► │ TaskService  │  ← Handler depends on trait (contract)
//! └──────────┘         └──────────────┘
//!                            ▲
//!              ┌─────────────┴─────────────┐
//!              │                           │
//!    ┌──────────────────┐       ┌─────────────────┐
//!    │ TaskServiceImpl  │       │ MockTaskService │
//!    │   (production)   │       │    (testing)    │
//!    └──────────────────┘       └─────────────────┘
//! ```
//!
//! ## Key Syntax Explained
//!
//! ```rust,ignore
//! #[async_trait]  // Macro: enables async methods in traits (Rust doesn't support this natively yet)
//! pub trait TaskService: Send + Sync {  // `: Send + Sync` = safe to use across threads
//!     async fn list_tasks(&self, ...) -> Result<..., AppError>;
//!     //                  ^^^^^ `&self` = this method borrows the service (doesn't consume it)
//! }
//! ```
//!
//! ## Common Questions
//!
//! **Q: What does `Send + Sync` mean?**
//! - `Send`: This type can be transferred to another thread
//! - `Sync`: This type can be referenced from multiple threads simultaneously
//! - Required because web servers handle requests on multiple threads
//!
//! **Q: Why `#[async_trait]`?**
//! - Rust traits don't natively support async methods (yet!)
//! - This macro transforms async methods into something traits can handle
//! - It's a common pattern - you'll see it in many Rust web projects
//!
//! **Q: What's the `&self` parameter?**
//! - Like `this` in JavaScript or `self` in Python
//! - `&self` = immutable borrow (can read, not modify)
//! - `&mut self` = mutable borrow (can modify)
//! - `self` = takes ownership (consumes the value)

use async_trait::async_trait;

use crate::error::AppError;
use crate::models::auth::{AuthRequest, AuthResponse};
use crate::models::category::{Category, CreateCategoryRequest, UpdateCategoryRequest};
use crate::models::task::{CreateTaskRequest, Task, UpdateTaskRequest};

// ============================================================================
// AUTHENTICATION SERVICE
// ============================================================================

/// Authentication service trait
///
/// Defines the contract for user authentication operations.
/// Any type implementing this trait can be used for auth in our app.
///
/// # Implementations
///
/// - [`AuthServiceImpl`](super::AuthServiceImpl) - Production implementation using SurrealDB
/// - `MockAuthService` - Test implementation (see `tests/common/mod.rs`)
///
/// # Example
///
/// ```rust,ignore
/// // In handlers, we use the trait, not the concrete type:
/// async fn login_handler(
///     State(state): State<AppState>,
///     Json(req): Json<AuthRequest>,
/// ) -> Result<Json<AuthResponse>, AppError> {
///     // state.auth_service is Arc<dyn AuthService>
///     // We don't know (or care) if it's real or mock!
///     let response = state.auth_service.login(req).await?;
///     Ok(Json(response))
/// }
/// ```
#[async_trait]
pub trait AuthService: Send + Sync {
    /// Authenticate a user with username/password
    ///
    /// # Arguments
    /// * `req` - Login credentials (username and password)
    ///
    /// # Returns
    /// * `Ok(AuthResponse)` - Contains JWT token and user ID
    /// * `Err(AppError::Unauthorized)` - Invalid credentials
    ///
    /// # Example
    /// ```rust,ignore
    /// let response = auth_service.login(AuthRequest {
    ///     username: "alice".to_string(),
    ///     password: "secret123".to_string(),
    /// }).await?;
    /// println!("Token: {}", response.token);
    /// ```
    async fn login(&self, req: AuthRequest) -> Result<AuthResponse, AppError>;

    /// Register a new user
    ///
    /// Creates a new user account and returns a JWT token for immediate login.
    ///
    /// # Arguments
    /// * `req` - Registration details (username and password)
    ///
    /// # Returns
    /// * `Ok(AuthResponse)` - Contains JWT token for the new user
    /// * `Err(AppError::BadRequest)` - Username already exists
    async fn register(&self, req: AuthRequest) -> Result<AuthResponse, AppError>;
}

// ============================================================================
// TASK SERVICE
// ============================================================================

/// Task management service trait
///
/// Defines the contract for all task-related business logic.
/// This is where validation, authorization, and business rules live.
///
/// # Architecture Note
///
/// The service layer sits between handlers and repositories:
///
/// ```text
/// Handler (HTTP) → Service (Business Logic) → Repository (Database)
/// ```
///
/// Services should:
/// - Validate business rules (e.g., "end_date must be after start_date")
/// - Orchestrate multiple repository calls if needed
/// - NOT know about HTTP (no StatusCode, no Json)
/// - NOT contain raw SQL/database queries (that's the repository's job)
///
/// # Implementations
///
/// - [`TaskServiceImpl`](super::TaskServiceImpl) - Production implementation
/// - `MockTaskService` - Test implementation (see `tests/common/mod.rs`)
#[async_trait]
pub trait TaskService: Send + Sync {
    /// List tasks for a user with pagination
    ///
    /// # Arguments
    /// * `user_id` - The owner's user ID (for authorization)
    /// * `limit` - Maximum number of tasks to return
    /// * `offset` - Number of tasks to skip (for pagination)
    ///
    /// # Returns
    /// A tuple of `(tasks, total_count)` where:
    /// - `tasks` - The paginated list of tasks
    /// - `total_count` - Total number of tasks (for pagination UI)
    ///
    /// # Example
    /// ```rust,ignore
    /// let (tasks, total) = service.list_tasks("user:123", 10, 0).await?;
    /// println!("Showing {} of {} tasks", tasks.len(), total);
    /// ```
    async fn list_tasks(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Task>, i64), AppError>;

    /// Create a new task
    ///
    /// # Arguments
    /// * `req` - Task creation request with title, dates, etc.
    /// * `user_id` - The creator's user ID
    ///
    /// # Business Rules
    /// - `start_date` must be before `end_date`
    /// - Title must not be empty
    ///
    /// # Returns
    /// The created task with generated ID and timestamps
    async fn create_task(&self, req: CreateTaskRequest, user_id: &str) -> Result<Task, AppError>;

    /// Update an existing task
    ///
    /// # Arguments
    /// * `id` - The task ID to update
    /// * `req` - Fields to update (only provided fields are changed)
    /// * `user_id` - The requesting user's ID (must be the owner)
    ///
    /// # Returns
    /// * `Ok(Task)` - The updated task
    /// * `Err(AppError::NotFound)` - Task doesn't exist or not owned by user
    async fn update_task(
        &self,
        id: &str,
        req: UpdateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError>;

    /// Delete a task (soft delete)
    ///
    /// Soft delete means we set a `deleted_at` timestamp instead of
    /// actually removing the record. This allows for:
    /// - Recovery if needed
    /// - Audit trails
    /// - Data retention compliance
    ///
    /// # Arguments
    /// * `id` - The task ID to delete
    /// * `user_id` - The requesting user's ID (must be the owner)
    async fn delete_task(&self, id: &str, user_id: &str) -> Result<(), AppError>;

    /// Get a task by ID
    ///
    /// # Arguments
    /// * `id` - The task ID
    /// * `user_id` - The requesting user's ID (must be the owner)
    ///
    /// # Returns
    /// * `Ok(Task)` - The requested task
    /// * `Err(AppError::NotFound)` - Task doesn't exist or not owned by user
    async fn get_task(&self, id: &str, user_id: &str) -> Result<Task, AppError>;
}

// ============================================================================
// CATEGORY SERVICE
// ============================================================================

/// Category management service trait
///
/// Defines the contract for category-related business logic.
/// Categories are user-specific and can be attached to tasks.
///
/// # Implementations
///
/// - [`CategoryServiceImpl`](super::CategoryServiceImpl) - Production implementation
/// - `MockCategoryService` - Test implementation
#[async_trait]
pub trait CategoryService: Send + Sync {
    /// List categories for a user with pagination
    ///
    /// # Arguments
    /// * `user_id` - The owner's user ID (for authorization)
    /// * `limit` - Maximum number of categories to return
    /// * `offset` - Number of categories to skip (for pagination)
    ///
    /// # Returns
    /// A tuple of `(categories, total_count)` where:
    /// - `categories` - The paginated list of categories
    /// - `total_count` - Total number of categories (for pagination UI)
    async fn list_categories(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Category>, i64), AppError>;

    /// Create a new category
    ///
    /// # Arguments
    /// * `req` - Category creation request with name and color
    /// * `user_id` - The creator's user ID
    ///
    /// # Business Rules
    /// - Category name must be unique per user
    /// - Name and color are required
    ///
    /// # Returns
    /// The created category with generated ID and timestamps
    async fn create_category(
        &self,
        req: CreateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError>;

    /// Update an existing category
    ///
    /// # Arguments
    /// * `id` - The category ID to update
    /// * `req` - Fields to update (only provided fields are changed)
    /// * `user_id` - The requesting user's ID (must be the owner)
    ///
    /// # Returns
    /// * `Ok(Category)` - The updated category
    /// * `Err(AppError::NotFound)` - Category doesn't exist or not owned by user
    async fn update_category(
        &self,
        id: &str,
        req: UpdateCategoryRequest,
        user_id: &str,
    ) -> Result<Category, AppError>;

    /// Delete a category (soft delete)
    ///
    /// # Arguments
    /// * `id` - The category ID to delete
    /// * `user_id` - The requesting user's ID (must be the owner)
    async fn delete_category(&self, id: &str, user_id: &str) -> Result<(), AppError>;

    /// Get a category by ID
    ///
    /// # Arguments
    /// * `id` - The category ID
    /// * `user_id` - The requesting user's ID (must be the owner)
    ///
    /// # Returns
    /// * `Ok(Category)` - The requested category
    /// * `Err(AppError::NotFound)` - Category doesn't exist or not owned by user
    async fn get_category(&self, id: &str, user_id: &str) -> Result<Category, AppError>;
}
