//! {{feature_name_pascal}} HTTP handlers

use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    middleware,
    routing::{delete, get, post, put},
    Json, Router,
};
use validator::Validate;

use super::model::{Create{{feature_name_pascal}}Request, {{feature_name_pascal}}, Update{{feature_name_pascal}}Request};
use crate::core::error::{ApiResponse, AppError, PaginatedResponse};
use crate::core::middleware::{auth_middleware, AuthenticatedUser};
use crate::shared::pagination::PaginationParams;
use crate::state::AppState;

/// Create {{feature_name}} routes
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
    // let response = PaginatedResponse::new(items, total, params.limit, params.offset);
    // Ok((StatusCode::OK, Json(ApiResponse::success(response))))

    todo!("Implement list_{{feature_name}}s - wire up {{feature_name}}_service in AppState first")
}

/// Get a {{feature_name}} by ID
#[utoipa::path(
    get,
    path = "/api/v1/{{feature_name}}s/{id}",
    params(
        ("id" = String, Path, description = "{{feature_name_pascal}} ID")
    ),
    responses(
        (status = 200, description = "{{feature_name_pascal}} found", body = {{feature_name_pascal}}),
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
) -> Result<(StatusCode, Json<ApiResponse<{{feature_name_pascal}}>>), AppError> {
    tracing::debug!(
        id = %id,
        user_id = %auth_user.user_id,
        "getting {{feature_name}}"
    );

    // TODO: Implement using service
    // let item = state.{{feature_name}}_service.get_{{feature_name}}(&id, &auth_user.user_id).await?;
    // Ok((StatusCode::OK, Json(ApiResponse::success(item))))

    todo!("Implement get_{{feature_name}}")
}

/// Create a new {{feature_name}}
#[utoipa::path(
    post,
    path = "/api/v1/{{feature_name}}s",
    request_body = Create{{feature_name_pascal}}Request,
    responses(
        (status = 201, description = "{{feature_name_pascal}} created successfully", body = {{feature_name_pascal}}),
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
    // let item = state.{{feature_name}}_service.create_{{feature_name}}(req, &auth_user.user_id).await?;
    // Ok((StatusCode::CREATED, Json(ApiResponse::success(item))))

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
        (status = 200, description = "{{feature_name_pascal}} updated successfully", body = {{feature_name_pascal}}),
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
) -> Result<(StatusCode, Json<ApiResponse<{{feature_name_pascal}}>>), AppError> {
    req.validate()?;

    tracing::debug!(
        id = %id,
        user_id = %auth_user.user_id,
        "updating {{feature_name}}"
    );

    // TODO: Implement using service
    // let item = state.{{feature_name}}_service.update_{{feature_name}}(&id, req, &auth_user.user_id).await?;
    // Ok((StatusCode::OK, Json(ApiResponse::success(item))))

    todo!("Implement update_{{feature_name}}")
}

/// Delete a {{feature_name}} (soft delete)
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
    // state.{{feature_name}}_service.delete_{{feature_name}}(&id, &auth_user.user_id).await?;
    // Ok(StatusCode::NO_CONTENT)

    todo!("Implement delete_{{feature_name}}")
}

