//! Daily Journal Backend API - Rust/Axum implementation
//!
//! This library exposes the core modules for use in integration tests
//! and other binaries (like the schema CLI).
//!
//! # Architecture
//!
//! ```text
//! handlers/   - HTTP route handlers (each exports routes())
//! services/   - Business logic (async traits for DI)
//! repositories/ - Data access layer
//! models/     - Domain models and DTOs
//! config/     - Application configuration
//! error/      - Error types and handling
//! utils/      - Shared utilities (state, middleware)
//! ```
//!
//! # Example: Using services in tests
//!
//! ```ignore
//! use rust_backend::services::{TaskService, TaskServiceImpl};
//! use rust_backend::config::Settings;
//!
//! #[tokio::test]
//! async fn test_task_service() {
//!     let settings = Settings::new().unwrap();
//!     // ... setup mock or real services
//! }
//! ```

#![deny(clippy::all)]
#![warn(clippy::pedantic)]
#![allow(clippy::module_name_repetitions)]
#![allow(clippy::must_use_candidate)]

pub mod config;
pub mod error;
pub mod handlers;
pub mod models;
pub mod repositories;
pub mod services;
pub mod utils;

// Re-export commonly used types for convenience
pub use config::Settings;
pub use error::{ApiError, ApiResponse, AppError};
pub use utils::state::{AppState, AppStateBuilder};

