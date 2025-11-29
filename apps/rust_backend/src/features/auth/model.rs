//! Authentication models and DTOs

use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use validator::Validate;

/// Authentication request payload
#[derive(Debug, Deserialize, Validate, ToSchema)]
#[schema(example = json!({
    "username": "admin@example.com",
    "password": "adminadmin"
}))]
pub struct AuthRequest {
    /// Username or email
    #[validate(length(min = 1, message = "Username is required"))]
    #[serde(alias = "email")]
    #[schema(example = "admin@example.com")]
    pub username: String,

    /// Password (minimum 6 characters)
    #[validate(length(min = 6, message = "Password must be at least 6 characters"))]
    #[serde(alias = "pass")]
    #[schema(example = "adminadmin")]
    pub password: String,
}

/// Authentication response with JWT token
#[derive(Debug, Serialize, ToSchema)]
#[schema(example = json!({
    "token": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9...",
    "user": "user:abc123"
}))]
pub struct AuthResponse {
    /// JWT authentication token
    #[schema(example = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9...")]
    pub token: String,

    /// User identifier (SurrealDB record ID)
    #[schema(example = "user:abc123")]
    pub user: String,
}

/// SurrealDB-specific JWT claims
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SurrealClaims {
    /// User ID (SurrealDB record ID)
    #[serde(rename = "ID")]
    pub id: String,
    /// Namespace
    #[serde(rename = "NS")]
    pub ns: String,
    /// Database
    #[serde(rename = "DB")]
    pub db: String,
    /// Access method
    #[serde(rename = "AC")]
    pub ac: String,
    /// Issued at
    #[serde(default)]
    pub iat: Option<i64>,
    /// Not before
    #[serde(default)]
    pub nbf: Option<i64>,
    /// Expiration
    #[serde(default)]
    pub exp: Option<i64>,
}

impl SurrealClaims {
    /// Parse and validate a SurrealDB-issued JWT token
    pub fn from_token(token: &str, secret: &str) -> Result<Self, AuthError> {
        if token.is_empty() {
            return Err(AuthError::MissingToken);
        }
        if secret.is_empty() {
            return Err(AuthError::SecretNotConfigured);
        }

        let mut validation = jsonwebtoken::Validation::new(jsonwebtoken::Algorithm::HS512);
        validation.validate_exp = true;
        validation.required_spec_claims.clear();

        let token_data = jsonwebtoken::decode::<SurrealClaims>(
            token,
            &jsonwebtoken::DecodingKey::from_secret(secret.as_bytes()),
            &validation,
        )
        .map_err(|e| AuthError::InvalidToken(e.to_string()))?;

        if token_data.claims.id.is_empty() {
            return Err(AuthError::InvalidToken("Missing user ID in token".into()));
        }

        Ok(token_data.claims)
    }
}

#[derive(Debug, thiserror::Error)]
pub enum AuthError {
    #[error("Missing token")]
    MissingToken,
    #[error("JWT secret not configured")]
    SecretNotConfigured,
    #[error("Invalid or expired token: {0}")]
    InvalidToken(String),
}
