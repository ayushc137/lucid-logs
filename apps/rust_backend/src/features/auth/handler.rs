//! Authentication HTTP handlers

use axum::{extract::State, routing::post, Json, Router};
use validator::Validate;

use super::model::{AuthRequest, AuthResponse};
use crate::core::error::AppError;
use crate::state::AppState;

/// Create auth routes
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/auth/login", post(login))
        .route("/auth/register", post(register))
}

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

    let response = state.auth_service.login(req).await?;

    Ok(Json(response))
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

    let response = state.auth_service.register(req).await?;

    Ok(Json(response))
}

