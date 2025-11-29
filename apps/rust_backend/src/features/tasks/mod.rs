//! Task management feature
//!
//! Handles CRUD operations for user-owned tasks with category support.

pub mod handler;
pub mod model;
pub mod repository;
pub mod service;

pub use handler::protected_routes;
pub use service::{TaskService, TaskServiceImpl};
