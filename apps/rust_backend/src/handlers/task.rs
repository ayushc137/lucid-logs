use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    Json,
};
use serde::Deserialize;
use utoipa::{IntoParams, ToSchema};
use validator::Validate;

use crate::{
    error::{ApiResponse, AppError, PaginatedResponse},
    models::task::{CreateTaskRequest, Task, UpdateTaskRequest},
    repositories::TaskRepository,
    utils::middleware::AuthenticatedUser,
    utils::state::AppState,
};

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

    let repo = TaskRepository::new(state.db.clone());
    let (tasks, total) = repo
        .find_by_user_paginated(&auth_user.user_id, params.limit, params.offset)
        .await?;

    tracing::info!(count = %tasks.len(), total = %total, "tasks listed successfully");

    let response = PaginatedResponse::new(tasks, total, params.limit, params.offset);
    Ok((StatusCode::OK, Json(ApiResponse::success(response))))
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

    // Validate dates
    if req.end_date.time_value() < req.start_date.time_value() {
        return Err(AppError::BadRequest(
            "end_date must be on or after start_date".to_string(),
        ));
    }

    tracing::debug!(
        user_id = %auth_user.user_id,
        title = %req.title,
        "creating task"
    );

    let repo = TaskRepository::new(state.db.clone());
    let task = repo.create(req, &auth_user.user_id).await?;

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

    // Validate dates if both are provided
    if let (Some(start), Some(end)) = (&req.start_date, &req.end_date) {
        if end.time_value() < start.time_value() {
            return Err(AppError::BadRequest(
                "end_date must be on or after start_date".to_string(),
            ));
        }
    }

    tracing::debug!(
        task_id = %id,
        user_id = %auth_user.user_id,
        "updating task"
    );

    let repo = TaskRepository::new(state.db.clone());
    let task = repo.update(&id, req, &auth_user.user_id).await?;

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

    let repo = TaskRepository::new(state.db.clone());
    repo.delete(&id, &auth_user.user_id).await?;

    tracing::info!(task_id = %id, "task deleted successfully");

    Ok(StatusCode::NO_CONTENT)
}
