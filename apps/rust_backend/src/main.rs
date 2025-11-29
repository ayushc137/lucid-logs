//! Daily Journal Backend - Main Entry Point
//!
//! This is where the application starts. It's responsible for:
//! 1. Loading configuration from environment variables
//! 2. Setting up logging/tracing
//! 3. Connecting to the database
//! 4. Creating services with dependency injection
//! 5. Building the HTTP router with all routes
//! 6. Starting the server

use axum::{http::HeaderValue, response::Html, routing::get, Json, Router};
use std::{net::SocketAddr, sync::Arc};
use surrealdb::engine::remote::ws::{Client, Ws};
use surrealdb::Surreal;
use tower_http::cors::{AllowOrigin, Any, CorsLayer};
use tower_http::trace::TraceLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};
use utoipa::OpenApi;
use utoipa_scalar::{Scalar, Servable};

// Module imports from the new structure
mod core;
mod features;
mod shared;
mod state;

use crate::core::{
    init_schema, resolve_migrations_dir, MigrationRunner, SchemaInitOptions, Settings,
};
use crate::features::{
    auth_routes, category_protected_routes, health_routes, task_protected_routes, AuthServiceImpl,
    CategoryServiceImpl, TaskServiceImpl,
};
use crate::state::AppState;

// ============================================================================
// OPENAPI DOCUMENTATION
// ============================================================================

#[derive(OpenApi)]
#[openapi(
    paths(
        // Health endpoints
        features::health::handler::health_check,
        features::health::handler::health_check_v1,
        // Auth endpoints
        features::auth::handler::login,
        features::auth::handler::register,
        // Task endpoints
        features::tasks::handler::list_tasks,
        features::tasks::handler::get_task,
        features::tasks::handler::create_task,
        features::tasks::handler::update_task,
        features::tasks::handler::delete_task,
        // Category endpoints
        features::categories::handler::list_categories,
        features::categories::handler::create_category,
        features::categories::handler::get_category,
        features::categories::handler::update_category,
        features::categories::handler::delete_category,
    ),
    components(
        schemas(
            // Auth types
            features::auth::model::AuthRequest,
            features::auth::model::AuthResponse,
            // Task types
            features::tasks::model::Task,
            features::tasks::model::CreateTaskRequest,
            features::tasks::model::UpdateTaskRequest,
            shared::pagination::PaginationParams,
            // Category types
            features::categories::model::Category,
            features::categories::model::CreateCategoryRequest,
            features::categories::model::UpdateCategoryRequest,
        )
    ),
    tags(
        (name = "health", description = "Health check endpoints"),
        (name = "auth", description = "Authentication endpoints"),
        (name = "tasks", description = "Task management endpoints"),
        (name = "categories", description = "Category management endpoints")
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

/// Security configuration for OpenAPI docs
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

// ============================================================================
// MAIN FUNCTION
// ============================================================================

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Load configuration
    let _ = dotenvy::dotenv();
    let settings = Settings::new()?;

    // Initialize logging
    let log_level = if settings.is_development() {
        "rust_backend=debug,tower_http=debug"
    } else {
        "rust_backend=info,tower_http=info"
    };

    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| log_level.into()),
        ))
        .with(tracing_subscriber::fmt::layer().with_target(true))
        .init();

    tracing::info!("Starting Daily Journal API (Rust)");
    tracing::info!("  Environment: {}", settings.app.env);
    tracing::info!("  Server port: {}", settings.server.port);
    tracing::info!("  Database URL: {}:{}", settings.db.host, settings.db.port);

    // Connect to database
    tracing::info!("Connecting to SurrealDB...");
    let db_url = settings.db.ws_url();

    let db: Surreal<Client> = match tokio::time::timeout(
        std::time::Duration::from_secs(10),
        Surreal::new::<Ws>(&db_url),
    )
    .await
    {
        Ok(Ok(db)) => {
            tracing::info!("Connected to SurrealDB successfully");
            db
        },
        Ok(Err(e)) => {
            tracing::error!("Failed to connect to SurrealDB: {}", e);
            return Err(e.into());
        },
        Err(_) => {
            tracing::error!("Connection to SurrealDB timed out");
            anyhow::bail!("Database connection timeout");
        },
    };

    // Authenticate
    db.signin(surrealdb::opt::auth::Root {
        username: &settings.db.user,
        password: &settings.db.pass,
    })
    .await?;

    db.use_ns(&settings.db.namespace)
        .use_db(&settings.db.database)
        .await?;

    // Run migrations
    let migrations_dir = resolve_migrations_dir(settings.db.migrations_path.as_deref());
    tracing::info!(path = %migrations_dir.display(), "checking for pending migrations");

    let migration_runner =
        MigrationRunner::new(db.clone(), &migrations_dir, settings.jwt.secret.clone());

    match migration_runner.run_pending().await {
        Ok(applied) if applied.is_empty() => {
            tracing::info!("No pending migrations to apply");
        },
        Ok(applied) => {
            tracing::info!(count = applied.len(), "migrations applied");
        },
        Err(e) => {
            tracing::error!(error = %e, "failed to run migrations");
            return Err(e.into());
        },
    }

    // Initialize schema
    let schema_opts = SchemaInitOptions::from(&settings);
    if let Err(e) = init_schema(&db, schema_opts).await {
        tracing::warn!(error = %e, "Failed to init schema (this is normal if scope already exists)");
    }

    // Create application state
    let settings = Arc::new(settings);
    let task_service = Arc::new(TaskServiceImpl::new(db.clone()));
    let category_service = Arc::new(CategoryServiceImpl::new(db.clone()));
    let auth_service = Arc::new(AuthServiceImpl::new(db.clone(), settings.clone()));

    let app_state = AppState::new(
        db,
        settings.clone(),
        task_service,
        category_service,
        auth_service,
    );

    // Build router
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

    let cors = build_cors_layer(&settings);

    // Build API v1 routes using the new feature-based structure
    let api_v1 = Router::new()
        .merge(health_routes())
        .merge(auth_routes())
        .merge(task_protected_routes(app_state.clone()))
        .merge(category_protected_routes(app_state.clone()));

    let app = Router::new()
        .route("/health", get(features::health::health_check))
        .merge(scalar_route)
        .route(
            "/api-docs/openapi.json",
            get(|| async { Json(ApiDoc::openapi()) }),
        )
        .nest("/api/v1", api_v1)
        .with_state(app_state)
        .layer(TraceLayer::new_for_http())
        .layer(cors);

    // Start server
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

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

fn build_cors_layer(settings: &Settings) -> CorsLayer {
    let cors = CorsLayer::new();

    if settings.is_development() || settings.cors.allowed_origins.is_empty() {
        cors.allow_origin(Any)
            .allow_methods(Any)
            .allow_headers(Any)
            .allow_credentials(false)
    } else {
        let origins: Vec<HeaderValue> = settings
            .cors
            .allowed_origins
            .iter()
            .filter_map(|o| o.parse().ok())
            .collect();

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
