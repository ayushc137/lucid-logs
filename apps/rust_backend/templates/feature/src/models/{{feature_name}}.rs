//! {{feature_name_pascal}} domain model and DTOs
//!
//! This module defines:
//! - The main {{feature_name_pascal}} entity (what's stored in the database)
//! - Request DTOs (Create{{feature_name_pascal}}Request, Update{{feature_name_pascal}}Request)
//! - Response DTOs if needed

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use validator::Validate;

// ============================================================
// ENTITY
// ============================================================

/// {{feature_name_pascal}} entity - represents a {{feature_name}} in the database
/// 
/// # Fields
/// - `id`: SurrealDB record ID (e.g., "{{table_name}}:abc123")
/// - Other fields specific to your domain
/// 
/// # Example
/// ```rust
/// let {{feature_name}} = {{feature_name_pascal}} {
///     id: Some("{{table_name}}:abc123".to_string()),
///     name: "My {{feature_name_pascal}}".to_string(),
///     // ...
/// };
/// ```
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct {{feature_name_pascal}} {
    /// Unique identifier (SurrealDB record ID)
    /// Format: "{{table_name}}:uuid"
    #[schema(example = "{{table_name}}:abc123")]
    pub id: Option<String>,

    /// Name/title of the {{feature_name}}
    #[schema(example = "My {{feature_name_pascal}}")]
    pub name: String,

    /// Optional description
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,

    // ============================================================
    // Add your custom fields here
    // ============================================================

    // ============================================================
    // Audit fields (standard across all entities)
    // ============================================================

    /// When the {{feature_name}} was created
    pub created_at: DateTime<Utc>,

    /// When the {{feature_name}} was last updated
    pub updated_at: DateTime<Utc>,

    /// Soft delete timestamp (None if not deleted)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub deleted_at: Option<DateTime<Utc>>,

    /// User who created this {{feature_name}}
    #[schema(example = "user:xyz789")]
    pub created_by: String,

    /// User who last updated this {{feature_name}}
    #[schema(example = "user:xyz789")]
    pub updated_by: String,
}

// ============================================================
// REQUEST DTOs
// ============================================================

/// Request body for creating a new {{feature_name}}
/// 
/// # Validation
/// - `name`: Required, 1-255 characters
/// - `description`: Optional, max 1000 characters
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct Create{{feature_name_pascal}}Request {
    /// Name of the {{feature_name}}
    #[validate(length(min = 1, max = 255, message = "name must be 1-255 characters"))]
    #[schema(example = "My New {{feature_name_pascal}}")]
    pub name: String,

    /// Optional description
    #[validate(length(max = 1000, message = "description must be at most 1000 characters"))]
    #[schema(example = "A description of my {{feature_name}}")]
    pub description: Option<String>,

    // Add more fields as needed
}

/// Request body for updating a {{feature_name}}
/// 
/// All fields are optional - only provided fields will be updated.
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct Update{{feature_name_pascal}}Request {
    /// New name (optional)
    #[validate(length(min = 1, max = 255, message = "name must be 1-255 characters"))]
    #[schema(example = "Updated Name")]
    pub name: Option<String>,

    /// New description (optional)
    #[validate(length(max = 1000, message = "description must be at most 1000 characters"))]
    pub description: Option<String>,

    // Add more fields as needed
}

// ============================================================
// TESTS
// ============================================================

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

