//! {{feature_name_pascal}} HTTP handlers
//!
//! This module handles all HTTP requests related to {{feature_name}}s.
//! Each handler function corresponds to an API endpoint.

use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    middleware,
    routing::{delete, get, post, put},
    Json, Router,
};
use serde::Deserialize;
use utoipa::{IntoParams, ToSchema};
use validator::Validate;

use crate::{
    error::{ApiResponse, AppError, PaginatedResponse},
    models::{{feature_name}}::{Create{{feature_name_pascal}}Request, {{feature_name_pascal}}, Update{{feature_name_pascal}}Request},
    utils::middleware::{auth_middleware, AuthenticatedUser},
    utils::state::AppState,
};

// ============================================================
// ROUTES
// ============================================================

/// Create {{feature_name}} routes
/// 
/// These routes are merged into the main router in `main.rs`.
/// Auth middleware is applied via `protected_routes()`.
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/{{feature_name}}s", get(list_{{feature_name}}s))
        .route("/{{feature_name}}s", post(create_{{feature_name}}))
        .route("/{{feature_name}}s/{id}", get(get_{{feature_name}}))
        .route("/{{feature_name}}s/{id}", put(update_{{feature_name}}))
        .route("/{{feature_name}}s/{id}", delete(delete_{{feature_name}}))
}

/// Create protected routes with auth middleware applied
pub fn protected_routes(state: AppState) -> Router<AppState> {
    routes().layer(middleware::from_fn_with_state(state, auth_middleware))
}

// ============================================================
// QUERY PARAMETERS
// ============================================================

#[derive(Debug, Deserialize, Validate, IntoParams, ToSchema)]
pub struct PaginationParams {
    /// Number of items to return (1-100)
    #[serde(default = "default_limit")]
    #[validate(range(min = 1, max = 100))]
    #[param(example = 25, minimum = 1, maximum = 100)]
    pub limit: i64,

    /// Number of items to skip
    #[serde(default)]
    #[validate(range(min = 0, max = 100000))]
    #[param(example = 0, minimum = 0, maximum = 100000)]
    pub offset: i64,
}

fn default_limit() -> i64 {
    25
}

// ============================================================
// HANDLERS
// ============================================================

/// List all {{feature_name}}s with pagination
#[utoipa::path(
    get,
    path = "/api/v1/{{feature_name}}s",
    params(PaginationParams),
    responses(
        (status = 200, description = "List of {{feature_name}}s"),
        (status = 401, description = "Unauthorized"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "{{feature_name}}s"
)]
pub async fn list_{{feature_name}}s(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Query(params): Query<PaginationParams>,
) -> Result<(StatusCode, Json<ApiResponse<PaginatedResponse<{{feature_name_pascal}}>>>), AppError> {
    params.validate()?;

    tracing::debug!(
        user_id = %auth_user.user_id,
        limit = %params.limit,
        offset = %params.offset,
        "listing {{feature_name}}s"
    );

    // TODO: Implement using service
    // let (items, total) = state
    //     .{{feature_name}}_service
    //     .list_{{feature_name}}s(&auth_user.user_id, params.limit, params.offset)
    //     .await?;

    todo!("Implement list_{{feature_name}}s")
}

/// Get a {{feature_name}} by ID
#[utoipa::path(
    get,
    path = "/api/v1/{{feature_name}}s/{id}",
    params(
        ("id" = String, Path, description = "{{feature_name_pascal}} ID")
    ),
    responses(
        (status = 200, description = "{{feature_name_pascal}} found"),
        (status = 404, description = "{{feature_name_pascal}} not found"),
        (status = 401, description = "Unauthorized")
    ),
    security(("bearer_auth" = [])),
    tag = "{{feature_name}}s"
)]
pub async fn get_{{feature_name}}(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
) -> Result<Json<ApiResponse<{{feature_name_pascal}}>>, AppError> {
    tracing::debug!(
        id = %id,
        user_id = %auth_user.user_id,
        "getting {{feature_name}}"
    );

    // TODO: Implement using service
    todo!("Implement get_{{feature_name}}")
}

/// Create a new {{feature_name}}
#[utoipa::path(
    post,
    path = "/api/v1/{{feature_name}}s",
    request_body = Create{{feature_name_pascal}}Request,
    responses(
        (status = 201, description = "{{feature_name_pascal}} created successfully"),
        (status = 400, description = "Invalid request"),
        (status = 401, description = "Unauthorized"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "{{feature_name}}s"
)]
pub async fn create_{{feature_name}}(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Json(req): Json<Create{{feature_name_pascal}}Request>,
) -> Result<(StatusCode, Json<ApiResponse<{{feature_name_pascal}}>>), AppError> {
    req.validate()?;

    tracing::debug!(
        user_id = %auth_user.user_id,
        "creating {{feature_name}}"
    );

    // TODO: Implement using service
    todo!("Implement create_{{feature_name}}")
}

/// Update a {{feature_name}}
#[utoipa::path(
    put,
    path = "/api/v1/{{feature_name}}s/{id}",
    params(
        ("id" = String, Path, description = "{{feature_name_pascal}} ID")
    ),
    request_body = Update{{feature_name_pascal}}Request,
    responses(
        (status = 200, description = "{{feature_name_pascal}} updated successfully"),
        (status = 400, description = "Invalid request"),
        (status = 401, description = "Unauthorized"),
        (status = 404, description = "{{feature_name_pascal}} not found"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "{{feature_name}}s"
)]
pub async fn update_{{feature_name}}(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
    Json(req): Json<Update{{feature_name_pascal}}Request>,
) -> Result<Json<ApiResponse<{{feature_name_pascal}}>>, AppError> {
    req.validate()?;

    tracing::debug!(
        id = %id,
        user_id = %auth_user.user_id,
        "updating {{feature_name}}"
    );

    // TODO: Implement using service
    todo!("Implement update_{{feature_name}}")
}

/// Delete a {{feature_name}}
#[utoipa::path(
    delete,
    path = "/api/v1/{{feature_name}}s/{id}",
    params(
        ("id" = String, Path, description = "{{feature_name_pascal}} ID")
    ),
    responses(
        (status = 204, description = "{{feature_name_pascal}} deleted successfully"),
        (status = 401, description = "Unauthorized"),
        (status = 404, description = "{{feature_name_pascal}} not found"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "{{feature_name}}s"
)]
pub async fn delete_{{feature_name}}(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
) -> Result<StatusCode, AppError> {
    tracing::debug!(
        id = %id,
        user_id = %auth_user.user_id,
        "deleting {{feature_name}}"
    );

    // TODO: Implement using service
    todo!("Implement delete_{{feature_name}}")
}

