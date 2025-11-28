//! Feature modules (vertical slices)
//!
//! Each feature contains all related code:
//! - handler.rs - HTTP handlers
//! - model.rs - Domain models and DTOs
//! - service.rs - Business logic
//! - repository.rs - Data access (if needed)
//!
//! # Adding a New Feature
//!
//! 1. Create a new directory: `features/my_feature/`
//! 2. Add the files: `mod.rs`, `handler.rs`, `model.rs`, `service.rs`
//! 3. Register it here and in `app.rs`

pub mod auth;
pub mod categories;
pub mod health;
pub mod tasks;

// Re-export feature routes for app.rs
pub use auth::routes as auth_routes;
pub use categories::protected_routes as category_protected_routes;
pub use health::routes as health_routes;
pub use tasks::protected_routes as task_protected_routes;

// Re-export service traits for dependency injection
pub use auth::{AuthService, AuthServiceImpl};
pub use categories::{CategoryService, CategoryServiceImpl};
pub use tasks::{TaskService, TaskServiceImpl};

