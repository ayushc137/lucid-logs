//! Category HTTP handlers

use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    middleware,
    routing::{delete, get, post, put},
    Json, Router,
};
use validator::Validate;

use super::model::{Category, CreateCategoryRequest, UpdateCategoryRequest};
use crate::core::error::{ApiResponse, AppError, PaginatedResponse};
use crate::core::middleware::{auth_middleware, AuthenticatedUser};
use crate::shared::pagination::PaginationParams;
use crate::state::AppState;

/// Create category routes (protected by auth middleware)
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/categories", get(list_categories))
        .route("/categories", post(create_category))
        .route("/categories/{id}", get(get_category))
        .route("/categories/{id}", put(update_category))
        .route("/categories/{id}", delete(delete_category))
}

/// Create protected category routes with auth middleware applied
pub fn protected_routes(state: AppState) -> Router<AppState> {
    routes().layer(middleware::from_fn_with_state(state, auth_middleware))
}

/// List all categories with pagination
#[utoipa::path(
    get,
    path = "/api/v1/categories",
    params(PaginationParams),
    responses(
        (status = 200, description = "List of categories with pagination metadata"),
        (status = 400, description = "Invalid pagination parameters"),
        (status = 401, description = "Unauthorized"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "categories"
)]
pub async fn list_categories(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Query(params): Query<PaginationParams>,
) -> Result<(StatusCode, Json<ApiResponse<PaginatedResponse<Category>>>), AppError> {
    params.validate()?;

    tracing::debug!(
        user_id = %auth_user.user_id,
        limit = %params.limit,
        offset = %params.offset,
        "listing categories"
    );

    let (categories, total) = state
        .category_service
        .list_categories(&auth_user.user_id, params.limit, params.offset)
        .await?;

    tracing::info!(count = %categories.len(), total = %total, "categories listed successfully");

    let response = PaginatedResponse::new(categories, total, params.limit, params.offset);
    Ok((StatusCode::OK, Json(ApiResponse::success(response))))
}

/// Create a new category
#[utoipa::path(
    post,
    path = "/api/v1/categories",
    request_body = CreateCategoryRequest,
    responses(
        (status = 201, description = "Category created successfully", body = Category),
        (status = 400, description = "Invalid request or duplicate name"),
        (status = 401, description = "Unauthorized"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "categories"
)]
pub async fn create_category(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Json(req): Json<CreateCategoryRequest>,
) -> Result<(StatusCode, Json<ApiResponse<Category>>), AppError> {
    req.validate()?;

    tracing::debug!(
        user_id = %auth_user.user_id,
        name = %req.name,
        "creating category"
    );

    let category = state
        .category_service
        .create_category(req, &auth_user.user_id)
        .await?;

    tracing::info!(
        category_id = ?category.id,
        "category created successfully"
    );

    Ok((StatusCode::CREATED, Json(ApiResponse::success(category))))
}

/// Get a category by ID
#[utoipa::path(
    get,
    path = "/api/v1/categories/{id}",
    params(
        ("id" = String, Path, description = "Category ID")
    ),
    responses(
        (status = 200, description = "Category found", body = Category),
        (status = 401, description = "Unauthorized"),
        (status = 404, description = "Category not found"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "categories"
)]
pub async fn get_category(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
) -> Result<(StatusCode, Json<ApiResponse<Category>>), AppError> {
    tracing::debug!(
        category_id = %id,
        user_id = %auth_user.user_id,
        "getting category"
    );

    let category = state
        .category_service
        .get_category(&id, &auth_user.user_id)
        .await?;

    Ok((StatusCode::OK, Json(ApiResponse::success(category))))
}

/// Update a category
#[utoipa::path(
    put,
    path = "/api/v1/categories/{id}",
    params(
        ("id" = String, Path, description = "Category ID")
    ),
    request_body = UpdateCategoryRequest,
    responses(
        (status = 200, description = "Category updated successfully", body = Category),
        (status = 400, description = "Invalid request or duplicate name"),
        (status = 401, description = "Unauthorized"),
        (status = 404, description = "Category not found"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "categories"
)]
pub async fn update_category(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
    Json(req): Json<UpdateCategoryRequest>,
) -> Result<(StatusCode, Json<ApiResponse<Category>>), AppError> {
    req.validate()?;

    tracing::debug!(
        category_id = %id,
        user_id = %auth_user.user_id,
        "updating category"
    );

    let category = state
        .category_service
        .update_category(&id, req, &auth_user.user_id)
        .await?;

    tracing::info!(category_id = %id, "category updated successfully");

    Ok((StatusCode::OK, Json(ApiResponse::success(category))))
}

/// Delete a category (soft delete)
#[utoipa::path(
    delete,
    path = "/api/v1/categories/{id}",
    params(
        ("id" = String, Path, description = "Category ID")
    ),
    responses(
        (status = 204, description = "Category deleted successfully"),
        (status = 401, description = "Unauthorized"),
        (status = 404, description = "Category not found"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "categories"
)]
pub async fn delete_category(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
) -> Result<StatusCode, AppError> {
    tracing::debug!(
        category_id = %id,
        user_id = %auth_user.user_id,
        "deleting category"
    );

    state
        .category_service
        .delete_category(&id, &auth_user.user_id)
        .await?;

    tracing::info!(category_id = %id, "category deleted successfully");

    Ok(StatusCode::NO_CONTENT)
}
