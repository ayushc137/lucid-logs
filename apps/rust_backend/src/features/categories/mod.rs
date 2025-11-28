//! Category management feature
//!
//! Handles CRUD operations for user-owned categories.

pub mod handler;
pub mod model;
pub mod repository;
pub mod service;

pub use handler::protected_routes;
pub use model::Category;
pub use service::{CategoryService, CategoryServiceImpl};

