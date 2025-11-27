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
    models::task::{CreateTaskRequest, Task, UpdateTaskRequest},
    utils::middleware::{auth_middleware, AuthenticatedUser},
    utils::state::AppState,
};

/// Create task routes (protected by auth middleware)
/// Note: Auth middleware is applied via `protected_routes()` which requires AppState
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/tasks", get(list_tasks))
        .route("/tasks", post(create_task))
        .route("/tasks/{id}", get(get_task))
        .route("/tasks/{id}", put(update_task))
        .route("/tasks/{id}", delete(delete_task))
}

/// Create protected task routes with auth middleware applied
pub fn protected_routes(state: AppState) -> Router<AppState> {
    routes().layer(middleware::from_fn_with_state(state, auth_middleware))
}

#[derive(Debug, Deserialize, Validate, IntoParams, ToSchema)]
pub struct PaginationParams {
    /// Number of items to return (1-100)
    #[serde(default = "default_limit")]
    #[validate(range(min = 1, max = 100, message = "limit must be between 1 and 100"))]
    #[param(example = 25, minimum = 1, maximum = 100)]
    pub limit: i64,

    /// Number of items to skip (0-100000)
    #[serde(default)]
    #[validate(range(min = 0, max = 100000, message = "offset must be between 0 and 100000"))]
    #[param(example = 0, minimum = 0, maximum = 100000)]
    pub offset: i64,
}

fn default_limit() -> i64 {
    25
}

/// List all tasks with pagination
#[utoipa::path(
    get,
    path = "/api/v1/tasks",
    params(PaginationParams),
    responses(
        (status = 200, description = "List of tasks with pagination metadata", body = crate::error::PaginatedTaskResponse),
        (status = 400, description = "Invalid pagination parameters"),
        (status = 401, description = "Unauthorized"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "tasks"
)]
pub async fn list_tasks(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Query(params): Query<PaginationParams>,
) -> Result<(StatusCode, Json<ApiResponse<PaginatedResponse<Task>>>), AppError> {
    // Validate pagination params
    params.validate()?;

    tracing::debug!(
        user_id = %auth_user.user_id,
        limit = %params.limit,
        offset = %params.offset,
        "listing tasks"
    );

    let (tasks, total) = state
        .task_service
        .list_tasks(&auth_user.user_id, params.limit, params.offset)
        .await?;

    tracing::info!(count = %tasks.len(), total = %total, "tasks listed successfully");

    let response = PaginatedResponse::new(tasks, total, params.limit, params.offset);
    Ok((StatusCode::OK, Json(ApiResponse::success(response))))
}

/// Get a task by ID
#[utoipa::path(
    get,
    path = "/api/v1/tasks/{id}",
    params(
        ("id" = String, Path, description = "Task ID")
    ),
    responses(
        (status = 200, description = "Task found", body = Task),
        (status = 401, description = "Unauthorized"),
        (status = 404, description = "Task not found"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "tasks"
)]
pub async fn get_task(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
) -> Result<(StatusCode, Json<ApiResponse<Task>>), AppError> {
    tracing::debug!(
        task_id = %id,
        user_id = %auth_user.user_id,
        "getting task"
    );

    let task = state
        .task_service
        .get_task(&id, &auth_user.user_id)
        .await?;

    Ok((StatusCode::OK, Json(ApiResponse::success(task))))
}

/// Create a new task
#[utoipa::path(
    post,
    path = "/api/v1/tasks",
    request_body = CreateTaskRequest,
    responses(
        (status = 201, description = "Task created successfully", body = Task),
        (status = 400, description = "Invalid request"),
        (status = 401, description = "Unauthorized"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "tasks"
)]
pub async fn create_task(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Json(req): Json<CreateTaskRequest>,
) -> Result<(StatusCode, Json<ApiResponse<Task>>), AppError> {
    // Validate request
    req.validate()?;

    tracing::debug!(
        user_id = %auth_user.user_id,
        title = %req.title,
        "creating task"
    );

    // Date validation is handled by service layer
    let task = state.task_service.create_task(req, &auth_user.user_id).await?;

    tracing::info!(
        task_id = ?task.id,
        "task created successfully"
    );

    Ok((StatusCode::CREATED, Json(ApiResponse::success(task))))
}

/// Update a task
#[utoipa::path(
    put,
    path = "/api/v1/tasks/{id}",
    params(
        ("id" = String, Path, description = "Task ID")
    ),
    request_body = UpdateTaskRequest,
    responses(
        (status = 200, description = "Task updated successfully", body = Task),
        (status = 400, description = "Invalid request"),
        (status = 401, description = "Unauthorized"),
        (status = 404, description = "Task not found"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "tasks"
)]
pub async fn update_task(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
    Json(req): Json<UpdateTaskRequest>,
) -> Result<(StatusCode, Json<ApiResponse<Task>>), AppError> {
    // Validate request fields
    req.validate()?;

    tracing::debug!(
        task_id = %id,
        user_id = %auth_user.user_id,
        "updating task"
    );

    // Date validation is handled by service layer
    let task = state
        .task_service
        .update_task(&id, req, &auth_user.user_id)
        .await?;

    tracing::info!(task_id = %id, "task updated successfully");

    Ok((StatusCode::OK, Json(ApiResponse::success(task))))
}

/// Delete a task (soft delete)
#[utoipa::path(
    delete,
    path = "/api/v1/tasks/{id}",
    params(
        ("id" = String, Path, description = "Task ID")
    ),
    responses(
        (status = 204, description = "Task deleted successfully"),
        (status = 401, description = "Unauthorized"),
        (status = 404, description = "Task not found"),
        (status = 500, description = "Internal server error")
    ),
    security(("bearer_auth" = [])),
    tag = "tasks"
)]
pub async fn delete_task(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Path(id): Path<String>,
) -> Result<StatusCode, AppError> {
    tracing::debug!(
        task_id = %id,
        user_id = %auth_user.user_id,
        "deleting task"
    );

    state
        .task_service
        .delete_task(&id, &auth_user.user_id)
        .await?;

    tracing::info!(task_id = %id, "task deleted successfully");

    Ok(StatusCode::NO_CONTENT)
}
