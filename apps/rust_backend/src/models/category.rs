//! Category model and request/response types

use chrono::{DateTime, Utc};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use surrealdb::RecordId;
use utoipa::ToSchema;
use validator::Validate;

/// Custom deserializer for SurrealDB RecordId -> Option<String>
fn deserialize_record_id<'de, D>(deserializer: D) -> Result<Option<String>, D::Error>
where
    D: Deserializer<'de>,
{
    Option::<RecordId>::deserialize(deserializer).map(|opt| opt.map(|id| id.to_string()))
}

/// Custom serializer that passes through Option<String>
fn serialize_record_id<S>(value: &Option<String>, serializer: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    match value {
        Some(s) => serializer.serialize_some(s),
        None => serializer.serialize_none(),
    }
}

/// Category model representing a task category
#[derive(Debug, Serialize, Deserialize, Clone, ToSchema)]
#[schema(example = json!({
    "id": "categories:abc123",
    "name": "Work",
    "color": "#3B82F6",
    "created_at": "2025-11-24T10:00:00Z",
    "updated_at": "2025-11-24T10:00:00Z"
}))]
pub struct Category {
    /// Category ID (SurrealDB record ID like "categories:abc123")
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        deserialize_with = "deserialize_record_id",
        serialize_with = "serialize_record_id"
    )]
    #[schema(example = "categories:abc123")]
    pub id: Option<String>,

    /// Category name (unique per user)
    #[schema(example = "Work")]
    pub name: String,

    /// Color associated with this category (hex format)
    #[schema(example = "#3B82F6")]
    pub color: String,

    // === System-managed fields (read-only, set by DB) ===
    /// Creation timestamp (system-managed, read-only)
    #[serde(default = "Utc::now")]
    #[schema(value_type = String, format = DateTime, read_only = true)]
    pub created_at: DateTime<Utc>,

    /// Last update timestamp (system-managed, read-only)
    #[serde(default = "Utc::now")]
    #[schema(value_type = String, format = DateTime, read_only = true)]
    pub updated_at: DateTime<Utc>,

    /// Soft delete timestamp (system-managed, read-only)
    #[serde(default, skip_serializing_if = "Option::is_none")]
    #[schema(value_type = Option<String>, format = DateTime, read_only = true)]
    pub deleted_at: Option<DateTime<Utc>>,

    /// User ID who created the category (system-managed, read-only, hidden from API)
    #[serde(default, skip_serializing, rename = "created_by")]
    #[schema(example = "user:xyz789", read_only = true)]
    pub _created_by: String,

    /// User ID who last updated the category (system-managed, read-only, hidden from API)
    #[serde(default, skip_serializing, rename = "updated_by")]
    #[schema(example = "user:xyz789", read_only = true)]
    pub _updated_by: String,
}

/// Request payload for creating a new category
#[derive(Debug, Deserialize, Validate, ToSchema)]
#[schema(example = json!({
    "name": "Work",
    "color": "#3B82F6"
}))]
pub struct CreateCategoryRequest {
    /// Category name (required, unique per user)
    #[validate(length(
        min = 1,
        max = 100,
        message = "Name must be between 1 and 100 characters"
    ))]
    #[schema(example = "Work")]
    pub name: String,

    /// Color for the category (required, hex format recommended)
    #[validate(length(
        min = 1,
        max = 50,
        message = "Color must be between 1 and 50 characters"
    ))]
    #[schema(example = "#3B82F6")]
    pub color: String,
}

/// Request payload for updating an existing category
#[derive(Debug, Deserialize, Validate, ToSchema)]
#[schema(example = json!({
    "name": "Personal",
    "color": "#10B981"
}))]
pub struct UpdateCategoryRequest {
    /// New name (optional, must be unique per user if provided)
    #[validate(custom(function = "validate_optional_name"))]
    #[schema(example = "Personal")]
    pub name: Option<String>,

    /// New color (optional)
    #[validate(custom(function = "validate_optional_color"))]
    #[schema(example = "#10B981")]
    pub color: Option<String>,
}

/// Validate that optional name is non-empty if provided
fn validate_optional_name(name: &str) -> Result<(), validator::ValidationError> {
    if name.trim().is_empty() {
        return Err(validator::ValidationError::new("name_empty")
            .with_message("Name cannot be empty".into()));
    }
    if name.len() > 100 {
        return Err(validator::ValidationError::new("name_too_long")
            .with_message("Name must be at most 100 characters".into()));
    }
    Ok(())
}

/// Validate that optional color is non-empty if provided
fn validate_optional_color(color: &str) -> Result<(), validator::ValidationError> {
    if color.trim().is_empty() {
        return Err(validator::ValidationError::new("color_empty")
            .with_message("Color cannot be empty".into()));
    }
    if color.len() > 50 {
        return Err(validator::ValidationError::new("color_too_long")
            .with_message("Color must be at most 50 characters".into()));
    }
    Ok(())
}
