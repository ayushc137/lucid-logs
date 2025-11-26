use axum::{
    http::HeaderValue,
    middleware,
    response::Html,
    routing::{delete, get, post, put},
    Json, Router,
};
use std::{net::SocketAddr, sync::Arc};
use surrealdb::engine::remote::ws::{Client, Ws};
use surrealdb::Surreal;
use tower_http::cors::{AllowOrigin, Any, CorsLayer};
use tower_http::trace::TraceLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};
use utoipa::OpenApi;
use utoipa_scalar::{Scalar, Servable};

mod config;
mod error;
mod handlers;
mod models;
mod repositories;
mod utils;

use crate::config::Settings;
use crate::repositories::{init_schema, SchemaInitOptions};
use crate::utils::middleware::auth_middleware;
use crate::utils::state::AppState;

#[derive(OpenApi)]
#[openapi(
    paths(
        handlers::health::health_check,
        handlers::health::health_check_v1,
        handlers::auth::login,
        handlers::auth::register,
        handlers::task::list_tasks,
        handlers::task::create_task,
        handlers::task::update_task,
        handlers::task::delete_task,
    ),
    components(
        schemas(
            models::auth::AuthRequest,
            models::auth::AuthResponse,
            models::task::Task,
            models::task::CreateTaskRequest,
            models::task::UpdateTaskRequest,
            handlers::task::PaginationParams,
            error::PaginatedTaskResponse,
        )
    ),
    tags(
        (name = "health", description = "Health check endpoints"),
        (name = "auth", description = "Authentication endpoints"),
        (name = "tasks", description = "Task management endpoints")
    ),
    info(
        title = "Daily Journal API",
        version = "1.0.0",
        description = "Backend API for the Daily Journal Application built with Rust and Axum"
    ),
    security(
        ("bearer_auth" = [])
    ),
    modifiers(&SecurityAddon)
)]
struct ApiDoc;

struct SecurityAddon;

impl utoipa::Modify for SecurityAddon {
    fn modify(&self, openapi: &mut utoipa::openapi::OpenApi) {
        if let Some(components) = openapi.components.as_mut() {
            components.add_security_scheme(
                "bearer_auth",
                utoipa::openapi::security::SecurityScheme::Http(
                    utoipa::openapi::security::Http::new(
                        utoipa::openapi::security::HttpAuthScheme::Bearer,
                    ),
                ),
            );
        }
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Load environment variables from .env file
    let _ = dotenvy::dotenv();

    // Load configuration
    let settings = Settings::new()?;

    // Initialize tracing based on environment
    let log_level = if settings.is_development() {
        "rust_backend=debug,tower_http=debug"
    } else {
        "rust_backend=info,tower_http=info"
    };

    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::new(
                std::env::var("RUST_LOG").unwrap_or_else(|_| log_level.into()),
            ),
        )
        .with(tracing_subscriber::fmt::layer().with_target(true))
        .init();

    tracing::info!("Starting Daily Journal API (Rust)");
    tracing::info!("  Environment: {}", settings.app.env);
    tracing::info!("  Server port: {}", settings.server.port);
    tracing::info!("  Database URL: {}:{}", settings.db.host, settings.db.port);
    tracing::info!("  Database NS/DB: {}/{}", settings.db.namespace, settings.db.database);

    // Connect to SurrealDB via WebSocket (maintains persistent connection & auth state)
    tracing::info!("Connecting to SurrealDB...");
    let db_url = settings.db.ws_url();
    tracing::debug!("Using WebSocket URL: {}", db_url);

    // Connect with a reasonable timeout for initial connection
    let db: Surreal<Client> = match tokio::time::timeout(
        std::time::Duration::from_secs(10),
        Surreal::new::<Ws>(&db_url),
    )
    .await
    {
        Ok(Ok(db)) => {
            tracing::info!("Connected to SurrealDB successfully");
            db
        }
        Ok(Err(e)) => {
            tracing::error!("Failed to connect to SurrealDB: {}", e);
            tracing::error!("Make sure SurrealDB is running on {}", db_url);
            tracing::error!("You can start it with: task db:up");
            return Err(e.into());
        }
        Err(_) => {
            tracing::error!("Connection to SurrealDB timed out after 10 seconds");
            tracing::error!("Make sure SurrealDB is accessible at {}", db_url);
            anyhow::bail!("Database connection timeout");
        }
    };

    // Sign in as root user for initial setup
    db.signin(surrealdb::opt::auth::Root {
        username: &settings.db.user,
        password: &settings.db.pass,
    })
    .await?;

    // Select namespace and database
    db.use_ns(&settings.db.namespace)
        .use_db(&settings.db.database)
        .await?;
    tracing::info!(
        "Using namespace: {}, database: {}",
        settings.db.namespace,
        settings.db.database
    );

    // Initialize schema (applies schema.surql and seeds admin user)
    let schema_opts = SchemaInitOptions::from(&settings);
    if let Err(e) = init_schema(&db, schema_opts).await {
        tracing::warn!(
            error = %e,
            "Failed to init schema (this is normal if scope already exists)"
        );
    }

    // Create application state with Arc<Settings> for efficient sharing
    let settings = Arc::new(settings);
    let app_state = AppState::new(db, settings.clone());

    // Create Scalar UI for API docs
    let scalar_html = Arc::new(Scalar::with_url("/docs", ApiDoc::openapi()).to_html());
    let scalar_route = Router::new().route(
        "/docs",
        get({
            let html = scalar_html.clone();
            move || {
                let html = html.clone();
                async move { Html(html.as_ref().clone()) }
            }
        }),
    );

    // Build environment-aware CORS layer
    let cors = build_cors_layer(&settings);

    // Build protected routes (require authentication)
    let protected_routes = Router::new()
        .route("/tasks", get(handlers::task::list_tasks))
        .route("/tasks", post(handlers::task::create_task))
        .route("/tasks/{id}", put(handlers::task::update_task))
        .route("/tasks/{id}", delete(handlers::task::delete_task))
        .layer(middleware::from_fn_with_state(
            app_state.clone(),
            auth_middleware,
        ));

    // Build API v1 routes
    let api_v1 = Router::new()
        .route("/health", get(handlers::health::health_check_v1))
        .route("/auth/login", post(handlers::auth::login))
        .route("/auth/register", post(handlers::auth::register))
        .merge(protected_routes);

    // Build application
    let app = Router::new()
        // Root health check
        .route("/health", get(handlers::health::health_check))
        // Scalar UI & OpenAPI JSON
        .merge(scalar_route)
        .route(
            "/api-docs/openapi.json",
            get(|| async { Json(ApiDoc::openapi()) }),
        )
        // API v1 routes
        .nest("/api/v1", api_v1)
        // Shared state
        .with_state(app_state)
        // Middleware layers
        .layer(TraceLayer::new_for_http())
        .layer(cors);

    // Start server with graceful shutdown
    let addr = SocketAddr::from(([0, 0, 0, 0], settings.server.port));
    tracing::info!("Server started on http://{}", addr);
    tracing::info!("Scalar UI available at http://{}/docs", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    tracing::info!("Server shut down gracefully");
    Ok(())
}

/// Build CORS layer based on environment configuration
fn build_cors_layer(settings: &Settings) -> CorsLayer {
    let cors = CorsLayer::new();

    if settings.is_development() || settings.cors.allowed_origins.is_empty() {
        // Development: allow any origin (no credentials)
        tracing::debug!("CORS: allowing any origin (development mode)");
        cors.allow_origin(Any)
            .allow_methods(Any)
            .allow_headers(Any)
            .allow_credentials(false)
    } else {
        // Production: restrict to configured origins
        let origins: Vec<HeaderValue> = settings
            .cors
            .allowed_origins
            .iter()
            .filter_map(|o| o.parse().ok())
            .collect();

        tracing::info!("CORS: restricting to {} configured origin(s)", origins.len());
        cors.allow_origin(AllowOrigin::list(origins))
            .allow_methods([
                axum::http::Method::GET,
                axum::http::Method::POST,
                axum::http::Method::PUT,
                axum::http::Method::DELETE,
                axum::http::Method::OPTIONS,
            ])
            .allow_headers([
                axum::http::header::AUTHORIZATION,
                axum::http::header::CONTENT_TYPE,
                axum::http::header::ACCEPT,
            ])
            .allow_credentials(true)
    }
}

/// Shutdown signal handler for graceful shutdown
async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            tracing::info!("Received Ctrl+C, initiating graceful shutdown...");
        },
        _ = terminate => {
            tracing::info!("Received SIGTERM, initiating graceful shutdown...");
        },
    }
}
