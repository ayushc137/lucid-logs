//! Application state management
//!
//! This module defines `AppState`, which holds all shared resources that handlers
//! need access to. In Axum, state is cloned for each request, so we use `Arc`
//! (Atomic Reference Counting) for expensive resources.

use std::sync::Arc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use crate::core::config::Settings;
use crate::features::{AuthService, CategoryService, TaskService};

/// Application state shared across all HTTP handlers.
///
/// This struct holds everything handlers need:
/// - Database connection for direct queries
/// - Configuration settings
/// - Services for business logic
#[derive(Clone)]
pub struct AppState {
    /// Database connection
    pub db: Surreal<Client>,

    /// Application configuration
    pub settings: Arc<Settings>,

    /// Task management service
    pub task_service: Arc<dyn TaskService>,

    /// Category management service
    pub category_service: Arc<dyn CategoryService>,

    /// Authentication service
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

