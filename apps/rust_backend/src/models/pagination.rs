use serde::Deserialize;
use utoipa::{IntoParams, ToSchema};
use validator::Validate;

pub const DEFAULT_LIMIT: i64 = 25;
pub const MAX_LIMIT: i64 = 100;
pub const MAX_OFFSET: i64 = 100_000;

fn default_limit() -> i64 {
    DEFAULT_LIMIT
}

/// Standard pagination parameters shared across list endpoints.
#[derive(Debug, Deserialize, Validate, IntoParams, ToSchema, Clone, Copy)]
pub struct PaginationParams {
    /// Number of items to return (1-100)
    #[serde(default = "default_limit")]
    #[validate(range(min = 1, max = MAX_LIMIT, message = "limit must be between 1 and 100"))]
    #[param(example = 25, minimum = 1, maximum = 100)]
    pub limit: i64,

    /// Number of items to skip (0-100000)
    #[serde(default)]
    #[validate(range(
        min = 0,
        max = MAX_OFFSET,
        message = "offset must be between 0 and 100000"
    ))]
    #[param(example = 0, minimum = 0, maximum = 100000)]
    pub offset: i64,
}
