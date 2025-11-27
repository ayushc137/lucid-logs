use axum::{
    body::Body,
    extract::State,
    http::{header, Request, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};

use crate::error::ApiResponse;
use crate::models::auth::SurrealClaims;
use crate::utils::state::AppState;

/// Authenticated user data extracted from JWT
#[derive(Clone, Debug)]
pub struct AuthenticatedUser {
    pub user_id: String,
}

/// Authentication middleware that validates SurrealDB-issued JWTs
/// and injects user claims into the request extensions
pub async fn auth_middleware(
    State(state): State<AppState>,
    mut req: Request<Body>,
    next: Next,
) -> Response {
    let auth_header = req
        .headers()
        .get(header::AUTHORIZATION)
        .and_then(|h| h.to_str().ok())
        .map(|s| s.trim().to_string());

    let auth_header = match auth_header {
        Some(h) if !h.is_empty() => h,
        _ => {
            return unauthorized_response("Authorization header required");
        },
    };

    const BEARER_PREFIX: &str = "Bearer ";
    if !auth_header.starts_with(BEARER_PREFIX) {
        return unauthorized_response("Authorization header must start with 'Bearer '");
    }

    let token = auth_header[BEARER_PREFIX.len()..].trim();
    if token.is_empty() {
        return unauthorized_response("Token is empty");
    }

    // Parse and validate the token
    let claims = match SurrealClaims::from_token(token, &state.settings.jwt.secret) {
        Ok(claims) => claims,
        Err(e) => {
            tracing::warn!(error = %e, "token validation failed");
            return unauthorized_response("Invalid or expired token");
        },
    };

    if claims.id.is_empty() {
        return unauthorized_response("Invalid token: missing user ID");
    }

    // Insert authenticated user into request extensions
    let auth_user = AuthenticatedUser { user_id: claims.id };

    req.extensions_mut().insert(auth_user);

    next.run(req).await
}

fn unauthorized_response(message: &str) -> Response {
    let body = ApiResponse::<()>::error("UNAUTHORIZED", message);
    (StatusCode::UNAUTHORIZED, Json(body)).into_response()
}

/// Extractor for getting the authenticated user from request extensions
pub mod extractors {
    use axum::{extract::FromRequestParts, http::request::Parts};

    use super::AuthenticatedUser;
    use crate::error::AppError;

    impl<S> FromRequestParts<S> for AuthenticatedUser
    where
        S: Send + Sync,
    {
        type Rejection = AppError;

        async fn from_request_parts(
            parts: &mut Parts,
            _state: &S,
        ) -> Result<Self, Self::Rejection> {
            parts
                .extensions
                .get::<AuthenticatedUser>()
                .cloned()
                .ok_or_else(|| AppError::Unauthorized("Authentication required".into()))
        }
    }
}
