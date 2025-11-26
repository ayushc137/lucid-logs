use axum::{extract::State, Json};
use serde_json::json;
use validator::Validate;

use crate::error::AppError;
use crate::models::auth::{AuthRequest, AuthResponse, SurrealClaims};
use crate::utils::state::AppState;

/// User login
///
/// Authenticates user against SurrealDB's account access method and returns a JWT token.
#[utoipa::path(
    post,
    path = "/api/v1/auth/login",
    request_body = AuthRequest,
    responses(
        (status = 200, description = "Login successful", body = AuthResponse),
        (status = 401, description = "Invalid credentials"),
        (status = 400, description = "Invalid request")
    ),
    tag = "auth"
)]
pub async fn login(
    State(state): State<AppState>,
    Json(req): Json<AuthRequest>,
) -> Result<Json<AuthResponse>, AppError> {
    req.validate()?;

    tracing::debug!(username = %req.username, "login attempt");

    // Sign in against SurrealDB's "account" access method
    let token = state
        .db
        .signin(surrealdb::opt::auth::Record {
            namespace: &state.settings.db.namespace,
            database: &state.settings.db.database,
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
    let claims = SurrealClaims::from_token(&token_str, &state.settings.jwt.secret).map_err(|e| {
        tracing::error!(error = %e, "failed to verify auth token");
        AppError::Internal
    })?;

    tracing::info!(user_id = %claims.id, "login successful");

    Ok(Json(AuthResponse {
        token: token_str,
        user: claims.id,
    }))
}

/// User registration
///
/// Creates a new user via SurrealDB's account access method and returns a JWT token.
#[utoipa::path(
    post,
    path = "/api/v1/auth/register",
    request_body = AuthRequest,
    responses(
        (status = 200, description = "Registration successful", body = AuthResponse),
        (status = 400, description = "Registration failed")
    ),
    tag = "auth"
)]
pub async fn register(
    State(state): State<AppState>,
    Json(req): Json<AuthRequest>,
) -> Result<Json<AuthResponse>, AppError> {
    req.validate()?;

    tracing::debug!(username = %req.username, "registration attempt");

    // Sign up using SurrealDB's "account" access method
    let token = state
        .db
        .signup(surrealdb::opt::auth::Record {
            namespace: &state.settings.db.namespace,
            database: &state.settings.db.database,
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
    let claims = SurrealClaims::from_token(&token_str, &state.settings.jwt.secret).map_err(|e| {
        tracing::error!(error = %e, "failed to verify auth token");
        AppError::Internal
    })?;

    tracing::info!(user_id = %claims.id, "user registered successfully");

    Ok(Json(AuthResponse {
        token: token_str,
        user: claims.id,
    }))
}
