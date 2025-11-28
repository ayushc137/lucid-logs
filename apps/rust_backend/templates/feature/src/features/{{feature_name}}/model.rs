//! {{feature_name_pascal}} models and DTOs

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

/// {{feature_name_pascal}} entity
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
#[schema(example = json!({
    "id": "{{table_name}}:abc123",
    "name": "My {{feature_name_pascal}}",
    "description": "A description",
    "created_at": "2025-11-24T10:00:00Z",
    "updated_at": "2025-11-24T10:00:00Z"
}))]
pub struct {{feature_name_pascal}} {
    /// Unique identifier (SurrealDB record ID like "{{table_name}}:abc123")
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        deserialize_with = "deserialize_record_id",
        serialize_with = "serialize_record_id"
    )]
    #[schema(example = "{{table_name}}:abc123")]
    pub id: Option<String>,

    /// Name of the {{feature_name}}
    #[schema(example = "My {{feature_name_pascal}}")]
    pub name: String,

    /// Optional description
    #[serde(default, skip_serializing_if = "Option::is_none")]
    #[schema(example = "A description")]
    pub description: Option<String>,

    // ============================================================
    // Add your custom fields here
    // ============================================================

    // ============================================================
    // System-managed fields (read-only, set by DB)
    // ============================================================

    /// Creation timestamp
    #[serde(default = "Utc::now")]
    #[schema(value_type = String, format = DateTime, read_only = true)]
    pub created_at: DateTime<Utc>,

    /// Last update timestamp
    #[serde(default = "Utc::now")]
    #[schema(value_type = String, format = DateTime, read_only = true)]
    pub updated_at: DateTime<Utc>,

    /// Soft delete timestamp
    #[serde(default, skip_serializing_if = "Option::is_none")]
    #[schema(value_type = Option<String>, format = DateTime, read_only = true)]
    pub deleted_at: Option<DateTime<Utc>>,

    /// User ID who created (hidden from API)
    #[serde(default, skip_serializing, rename = "created_by")]
    #[schema(example = "user:xyz789", read_only = true)]
    pub _created_by: String,

    /// User ID who last updated (hidden from API)
    #[serde(default, skip_serializing, rename = "updated_by")]
    #[schema(example = "user:xyz789", read_only = true)]
    pub _updated_by: String,
}

/// Request payload for creating a new {{feature_name}}
#[derive(Debug, Deserialize, Validate, ToSchema)]
#[schema(example = json!({
    "name": "My New {{feature_name_pascal}}",
    "description": "A description"
}))]
pub struct Create{{feature_name_pascal}}Request {
    /// Name of the {{feature_name}}
    #[validate(length(min = 1, max = 255, message = "name must be 1-255 characters"))]
    #[schema(example = "My New {{feature_name_pascal}}")]
    pub name: String,

    /// Optional description
    #[validate(length(max = 1000, message = "description must be at most 1000 characters"))]
    #[schema(example = "A description")]
    pub description: Option<String>,

    // Add more fields as needed
}

/// Request payload for updating a {{feature_name}}
#[derive(Debug, Deserialize, Validate, ToSchema)]
#[schema(example = json!({
    "name": "Updated Name",
    "description": "Updated description"
}))]
pub struct Update{{feature_name_pascal}}Request {
    /// New name (optional)
    #[validate(custom(function = "validate_optional_name"))]
    #[schema(example = "Updated Name")]
    pub name: Option<String>,

    /// New description (optional)
    #[validate(length(max = 1000, message = "description must be at most 1000 characters"))]
    pub description: Option<String>,

    // Add more fields as needed
}

/// Validate that optional name is non-empty if provided
fn validate_optional_name(name: &str) -> Result<(), validator::ValidationError> {
    if name.trim().is_empty() {
        return Err(validator::ValidationError::new("name_empty")
            .with_message("Name cannot be empty".into()));
    }
    if name.len() > 255 {
        return Err(validator::ValidationError::new("name_too_long")
            .with_message("Name must be at most 255 characters".into()));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_create_request_validation() {
        // Valid request
        let req = Create{{feature_name_pascal}}Request {
            name: "Test".to_string(),
            description: None,
        };
        assert!(req.validate().is_ok());

        // Empty name should fail
        let req = Create{{feature_name_pascal}}Request {
            name: "".to_string(),
            description: None,
        };
        assert!(req.validate().is_err());
    }
}

