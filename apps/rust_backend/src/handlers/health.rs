use axum::{extract::State, http::StatusCode, response::IntoResponse, routing::get, Json, Router};
use serde_json::json;

use crate::error::ApiResponse;
use crate::utils::state::AppState;

/// Create health routes
pub fn routes() -> Router<AppState> {
    Router::new().route("/health", get(health_check_v1))
}

/// Health check endpoint (basic)
#[utoipa::path(
    get,
    path = "/health",
    responses(
        (status = 200, description = "Service is healthy")
    ),
    tag = "health"
)]
pub async fn health_check() -> impl IntoResponse {
    let data = json!({
        "status": "ok",
        "service": "daily-journal-backend"
    });

    (StatusCode::OK, Json(ApiResponse::success(data)))
}

/// API v1 health check with database connectivity
#[utoipa::path(
    get,
    path = "/api/v1/health",
    responses(
        (status = 200, description = "Service is healthy"),
        (status = 503, description = "Service degraded - database unreachable")
    ),
    tag = "health"
)]
pub async fn health_check_v1(State(state): State<AppState>) -> impl IntoResponse {
    // Ping database with a simple query
    let db_status = match state.db.query("RETURN true").await {
        Ok(_) => "connected",
        Err(e) => {
            tracing::warn!(error = %e, "database health check failed");
            return (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(ApiResponse::success(json!({
                    "status": "degraded",
                    "service": "daily-journal-backend",
                    "database": "disconnected"
                }))),
            );
        }
    };

    let data = json!({
        "status": "ok",
        "service": "daily-journal-backend",
        "database": db_status
    });

    (StatusCode::OK, Json(ApiResponse::success(data)))
}
