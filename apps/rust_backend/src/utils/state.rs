//! Application state management
//!
//! This module defines `AppState`, which holds all shared resources that handlers
//! need access to. In Axum, state is cloned for each request, so we use `Arc`
//! (Atomic Reference Counting) for expensive resources.
//!
//! # For Rust Beginners
//!
//! ## Why Arc?
//! - Web servers handle many requests concurrently (at the same time)
//! - Each request handler needs access to services and the database
//! - Cloning the actual data for each request would be expensive
//! - `Arc` lets us share data by just incrementing a counter (cheap!)
//!
//! ## Why `dyn Trait`?
//! - `dyn TaskService` means "anything that implements TaskService"
//! - This is called "dynamic dispatch" or "trait objects"
//! - It lets us swap implementations at runtime (great for testing!)
//! - In tests, we use `MockTaskService`; in production, `TaskServiceImpl`
//!
//! ## The #[derive(Clone)] attribute
//! - Axum clones AppState for each request
//! - Because all fields are Arc (which is Clone), AppState can be Clone
//! - Cloning just increments reference counts - very fast!

use crate::config::Settings;
use crate::services::{AuthService, CategoryService, TaskService};
use std::sync::Arc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

/// Application state shared across all HTTP handlers.
///
/// This struct holds everything handlers need:
/// - Database connection for direct queries
/// - Configuration settings
/// - Services for business logic
///
/// # Architecture Pattern
///
/// ```text
/// ┌─────────────────────────────────────────────────┐
/// │                   AppState                       │
/// │  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
/// │  │    db    │  │ settings │  │   services   │  │
/// │  │ (Surreal)│  │  (Arc)   │  │ (Arc<dyn T>) │  │
/// │  └──────────┘  └──────────┘  └──────────────┘  │
/// └─────────────────────────────────────────────────┘
///                       │
///         ┌─────────────┼─────────────┐
///         ▼             ▼             ▼
///    ┌─────────┐  ┌─────────┐  ┌─────────┐
///    │Handler 1│  │Handler 2│  │Handler 3│
///    └─────────┘  └─────────┘  └─────────┘
/// ```
///
/// # Example Usage (in handlers)
///
/// ```rust,ignore
/// pub async fn my_handler(
///     State(state): State<AppState>,  // Axum extracts state here
/// ) -> Result<Json<Response>, AppError> {
///     // Access services through state
///     let result = state.task_service.list_tasks(...).await?;
///     Ok(Json(result))
/// }
/// ```
///
/// # Why Services Use `Arc<dyn Trait>`
///
/// | Approach | Pros | Cons |
/// |----------|------|------|
/// | `Arc<dyn Trait>` | Mockable, flexible | Slight runtime cost |
/// | `Arc<ConcreteType>` | Faster | Hard to mock |
/// | Generic `<T: Trait>` | Fastest | Complex, viral generics |
///
/// We choose `Arc<dyn Trait>` for testability - the performance difference
/// is negligible for web applications.
#[derive(Clone)]
pub struct AppState {
    /// Database connection
    ///
    /// `Surreal<Client>` is already internally reference-counted,
    /// so cloning is cheap. We keep this for direct DB access when
    /// services don't cover a use case.
    pub db: Surreal<Client>,

    /// Application configuration
    ///
    /// Wrapped in `Arc` because Settings is a larger struct and
    /// we don't want to clone it for every request.
    pub settings: Arc<Settings>,

    /// Task management service
    ///
    /// `Arc<dyn TaskService>` means:
    /// - `Arc`: Thread-safe reference counting (cheap to clone)
    /// - `dyn`: Dynamic dispatch (concrete type decided at runtime)
    /// - `TaskService`: The trait that defines task operations
    ///
    /// In production: `Arc<TaskServiceImpl>`
    /// In tests: `Arc<MockTaskService>`
    pub task_service: Arc<dyn TaskService>,

    /// Category management service
    ///
    /// Handles CRUD operations for user-owned categories.
    /// Categories can be attached to tasks for organization.
    pub category_service: Arc<dyn CategoryService>,

    /// Authentication service
    ///
    /// Handles login, registration, and token management.
    /// Same pattern as task_service for testability.
    pub auth_service: Arc<dyn AuthService>,
}

impl AppState {
    /// Create new AppState with provided services (for production)
    pub fn new(
        db: Surreal<Client>,
        settings: Arc<Settings>,
        task_service: Arc<dyn TaskService>,
        category_service: Arc<dyn CategoryService>,
        auth_service: Arc<dyn AuthService>,
    ) -> Self {
        Self {
            db,
            settings,
            task_service,
            category_service,
            auth_service,
        }
    }
}
