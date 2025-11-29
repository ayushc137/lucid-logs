//! Error types and API response wrappers
//!
//! Provides standardized error handling across all features.

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::Serialize;
use utoipa::ToSchema;

/// Standardized API error response
#[derive(Debug, Serialize, ToSchema)]
pub struct ApiError {
    pub code: String,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<serde_json::Value>,
}

/// Standardized API response wrapper
#[derive(Debug, Serialize, ToSchema)]
pub struct ApiResponse<T>
where
    T: Serialize + ToSchema,
{
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<T>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<ApiError>,
}

/// Paginated response wrapper for list endpoints
#[derive(Debug, Serialize, ToSchema)]
pub struct PaginatedResponse<T>
where
    T: Serialize + ToSchema,
{
    pub items: Vec<T>,
    pub total: i64,
    pub limit: i64,
    pub offset: i64,
    pub has_more: bool,
}

impl<T> PaginatedResponse<T>
where
    T: Serialize + ToSchema,
{
    pub fn new(items: Vec<T>, total: i64, limit: i64, offset: i64) -> Self {
        let has_more = offset + (items.len() as i64) < total;
        Self {
            items,
            total,
            limit,
            offset,
            has_more,
        }
    }
}

impl<T> ApiResponse<T>
where
    T: Serialize + ToSchema,
{
    pub fn success(data: T) -> Self {
        Self {
            data: Some(data),
            error: None,
        }
    }

    pub fn error(code: impl Into<String>, message: impl Into<String>) -> ApiResponse<()> {
        ApiResponse {
            data: None,
            error: Some(ApiError {
                code: code.into(),
                message: message.into(),
                details: None,
            }),
        }
    }

    pub fn error_with_details(
        code: impl Into<String>,
        message: impl Into<String>,
        details: serde_json::Value,
    ) -> ApiResponse<()> {
        ApiResponse {
            data: None,
            error: Some(ApiError {
                code: code.into(),
                message: message.into(),
                details: Some(details),
            }),
        }
    }
}

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("Database error: {0}")]
    Database(#[from] surrealdb::Error),

    #[error("Validation error: {0}")]
    Validation(#[from] validator::ValidationErrors),

    #[error("Not found")]
    NotFound,

    #[error("Unauthorized: {0}")]
    Unauthorized(String),

    #[error("Internal server error")]
    Internal,

    #[error("Bad request: {0}")]
    BadRequest(String),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, code, message, details) = match &self {
            AppError::Database(e) => {
                tracing::error!(error = %e, "database error");
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "INTERNAL_ERROR",
                    "An internal error occurred".to_string(),
                    None,
                )
            },
            AppError::Validation(e) => {
                let details: Vec<ValidationErrorDetail> = e
                    .field_errors()
                    .iter()
                    .flat_map(|(field, errors)| {
                        errors.iter().map(move |err| ValidationErrorDetail {
                            field: to_json_field(field),
                            message: err
                                .message
                                .as_ref()
                                .map(|m| m.to_string())
                                .unwrap_or_else(|| format!("validation failed: {}", err.code)),
                        })
                    })
                    .collect();

                tracing::warn!(errors = ?details, "validation failed");
                (
                    StatusCode::BAD_REQUEST,
                    "VALIDATION_FAILED",
                    "Request validation failed".to_string(),
                    Some(serde_json::to_value(details).unwrap_or_default()),
                )
            },
            AppError::NotFound => (
                StatusCode::NOT_FOUND,
                "NOT_FOUND",
                "Resource not found".to_string(),
                None,
            ),
            AppError::Unauthorized(msg) => {
                tracing::warn!(message = %msg, "unauthorized");
                (
                    StatusCode::UNAUTHORIZED,
                    "UNAUTHORIZED",
                    if msg.is_empty() {
                        "Authentication required".to_string()
                    } else {
                        msg.clone()
                    },
                    None,
                )
            },
            AppError::Internal => {
                tracing::error!("internal server error");
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "INTERNAL_ERROR",
                    "An internal error occurred".to_string(),
                    None,
                )
            },
            AppError::BadRequest(msg) => {
                tracing::warn!(message = %msg, "bad request");
                (StatusCode::BAD_REQUEST, "BAD_REQUEST", msg.clone(), None)
            },
        };

        let body = if let Some(details) = details {
            ApiResponse::<()>::error_with_details(code, message, details)
        } else {
            ApiResponse::<()>::error(code, message)
        };

        (status, Json(body)).into_response()
    }
}

#[derive(Debug, Serialize)]
struct ValidationErrorDetail {
    field: String,
    message: String,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct OperationMessage {
    #[schema(example = "Task deleted")]
    pub message: String,
}

/// Convert PascalCase field names to camelCase
fn to_json_field(field: &str) -> String {
    if field.is_empty() {
        return field.to_string();
    }
    let mut chars = field.chars();
    match chars.next() {
        Some(first) => first.to_lowercase().chain(chars).collect(),
        None => String::new(),
    }
}
