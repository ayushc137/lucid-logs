//! Daily Journal Backend API - Rust/Axum implementation
//!
//! This library exposes the core modules for use in integration tests
//! and other binaries (like the schema CLI).
//!
//! # Architecture (Feature-Based / Vertical Slices)
//!
//! ```text
//! src/
//! ├── core/           - Shared infrastructure (config, error, db, middleware)
//! ├── features/       - Feature modules (vertical slices)
//! │   ├── auth/       - Authentication (handler, model, service)
//! │   ├── tasks/      - Task management (handler, model, repository, service)
//! │   ├── categories/ - Category management (handler, model, repository, service)
//! │   └── health/     - Health checks (handler)
//! ├── shared/         - Shared types (pagination, repository utilities)
//! └── state.rs        - Application state (AppState)
//! ```
//!
//! # Adding a New Feature
//!
//! 1. Create a new directory: `features/my_feature/`
//! 2. Add files: `mod.rs`, `handler.rs`, `model.rs`, `service.rs`, `repository.rs` (if needed)
//! 3. Export from `features/mod.rs`
//! 4. Register routes in `main.rs`

#![deny(clippy::all)]
#![warn(clippy::pedantic)]
#![allow(clippy::module_name_repetitions)]
#![allow(clippy::must_use_candidate)]

pub mod core;
pub mod features;
pub mod shared;
pub mod state;

// Re-export commonly used types for convenience (from core)
pub use core::{init_schema, resolve_migrations_dir, MigrationRunner, SchemaInitOptions, Settings};

// Re-export shared utilities
pub use shared::{category_queries, task_queries, CategoryId, TaskId};
pub use state::AppState;

// Re-export feature types
pub use features::{
    auth::{AuthService, AuthServiceImpl},
    categories::{Category, CategoryService, CategoryServiceImpl},
    tasks::{TaskService, TaskServiceImpl},
};
