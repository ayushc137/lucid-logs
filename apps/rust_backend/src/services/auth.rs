//! Auth service implementation

use async_trait::async_trait;
use serde_json::json;
use std::sync::Arc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use crate::config::Settings;
use crate::error::AppError;
use crate::models::auth::{AuthRequest, AuthResponse, SurrealClaims};
use crate::services::traits::AuthService;

/// Production implementation of AuthService
#[derive(Clone)]
pub struct AuthServiceImpl {
    db: Surreal<Client>,
    settings: Arc<Settings>,
}

impl AuthServiceImpl {
    pub fn new(db: Surreal<Client>, settings: Arc<Settings>) -> Self {
        Self { db, settings }
    }
}

#[async_trait]
impl AuthService for AuthServiceImpl {
    async fn login(&self, req: AuthRequest) -> Result<AuthResponse, AppError> {
        tracing::debug!(username = %req.username, "login attempt via service");

        // Sign in against SurrealDB's "account" access method
        let token = self
            .db
            .signin(surrealdb::opt::auth::Record {
                namespace: &self.settings.db.namespace,
                database: &self.settings.db.database,
                access: "account",
                params: json!({
                    "user": req.username,
                    "pass": req.password,
                }),
            })
            .await
            .map_err(|e| {
                tracing::warn!(username = %req.username, error = %e, "login failed");
                AppError::Unauthorized("Invalid credentials".into())
            })?;

        let token_str = token.into_insecure_token();

        // Parse the token to get the user ID
        let claims =
            SurrealClaims::from_token(&token_str, &self.settings.jwt.secret).map_err(|e| {
                tracing::error!(error = %e, "failed to verify auth token");
                AppError::Internal
            })?;

        tracing::info!(user_id = %claims.id, "login successful");

        Ok(AuthResponse {
            token: token_str,
            user: claims.id,
        })
    }

    async fn register(&self, req: AuthRequest) -> Result<AuthResponse, AppError> {
        tracing::debug!(username = %req.username, "registration attempt via service");

        // Sign up using SurrealDB's "account" access method
        let token = self
            .db
            .signup(surrealdb::opt::auth::Record {
                namespace: &self.settings.db.namespace,
                database: &self.settings.db.database,
                access: "account",
                params: json!({
                    "user": req.username,
                    "pass": req.password,
                }),
            })
            .await
            .map_err(|e| {
                tracing::error!(username = %req.username, error = %e, "registration failed");
                AppError::BadRequest(format!("Registration failed: {}", e))
            })?;

        let token_str = token.into_insecure_token();

        // Parse the token to get the user ID
        let claims =
            SurrealClaims::from_token(&token_str, &self.settings.jwt.secret).map_err(|e| {
                tracing::error!(error = %e, "failed to verify auth token");
                AppError::Internal
            })?;

        tracing::info!(user_id = %claims.id, "user registered successfully");

        Ok(AuthResponse {
            token: token_str,
            user: claims.id,
        })
    }
}
