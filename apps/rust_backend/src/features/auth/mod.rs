//! Authentication feature
//!
//! Handles user login and registration via SurrealDB's access methods.

pub mod handler;
pub mod model;
pub mod service;

pub use handler::routes;
pub use model::SurrealClaims;
pub use service::{AuthService, AuthServiceImpl};

