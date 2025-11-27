//! Daily Journal Backend - Main Entry Point
//!
//! This is where the application starts. It's responsible for:
//! 1. Loading configuration from environment variables
//! 2. Setting up logging/tracing
//! 3. Connecting to the database
//! 4. Creating services with dependency injection
//! 5. Building the HTTP router with all routes
//! 6. Starting the server
//!
//! # For Rust Beginners
//!
//! ## Module Declarations (`mod xyz;`)
//!
//! The `mod` keyword declares a module. Rust will look for:
//! - `xyz.rs` file, OR
//! - `xyz/mod.rs` file
//!
//! This is how Rust organizes code into separate files.
//!
//! ## The `use` Keyword
//!
//! `use` brings items into scope so you don't have to write full paths:
//! ```rust,ignore
//! // Without use:
//! let router = axum::Router::new();
//!
//! // With use:
//! use axum::Router;
//! let router = Router::new();
//! ```
//!
//! ## `#[tokio::main]` Macro
//!
//! Rust's `main()` can't be async by default. This macro:
//! 1. Creates a Tokio async runtime
//! 2. Runs our async code inside it
//!
//! Without it, we couldn't use `.await` in main!

// ============================================================================
// IMPORTS
// ============================================================================
// Organized by: std lib → external crates → internal modules

use axum::{http::HeaderValue, response::Html, routing::get, Json, Router};
use std::{net::SocketAddr, sync::Arc};
use surrealdb::engine::remote::ws::{Client, Ws};
use surrealdb::Surreal;
use tower_http::cors::{AllowOrigin, Any, CorsLayer};
use tower_http::trace::TraceLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};
use utoipa::OpenApi;
use utoipa_scalar::{Scalar, Servable};

// ============================================================================
// MODULE DECLARATIONS
// ============================================================================
// Each `mod xyz;` tells Rust to include code from xyz.rs or xyz/mod.rs

mod config; // Configuration loading (from env vars)
mod error; // Error types and API response wrappers
mod handlers; // HTTP request handlers (the API endpoints)
mod models; // Data structures (entities, DTOs)
mod repositories; // Database access layer
mod services; // Business logic layer
mod utils; // Utilities (middleware, state)

// ============================================================================
// INTERNAL IMPORTS
// ============================================================================
// `crate::` means "from this project's root"

use crate::config::Settings;
use crate::repositories::{
    init_schema, resolve_migrations_dir, MigrationRunner, SchemaInitOptions,
};
use crate::services::{AuthServiceImpl, CategoryServiceImpl, TaskServiceImpl};
use crate::utils::state::AppState;

// ============================================================================
// OPENAPI DOCUMENTATION
// ============================================================================
// The `#[derive(OpenApi)]` macro generates OpenAPI/Swagger documentation.
// This powers the /docs endpoint with interactive API documentation.
//
// # How It Works
// - `paths(...)` - Lists all handler functions to document
// - `components(schemas(...))` - Lists all request/response types
// - `tags(...)` - Groups endpoints in the UI
//
// The Scalar UI (at /docs) reads this and renders interactive docs.

#[derive(OpenApi)]
#[openapi(
    paths(
        // Health endpoints - always good to have for monitoring
        handlers::health::health_check,
        handlers::health::health_check_v1,
        // Auth endpoints - login and registration
        handlers::auth::login,
        handlers::auth::register,
        // Task endpoints - the main CRUD operations
        handlers::task::list_tasks,
        handlers::task::get_task,
        handlers::task::create_task,
        handlers::task::update_task,
        handlers::task::delete_task,
        // Category endpoints - user-owned categories for task organization
        handlers::category::list_categories,
        handlers::category::create_category,
        handlers::category::get_category,
        handlers::category::update_category,
        handlers::category::delete_category,
    ),
    components(
        schemas(
            // Auth types
            models::auth::AuthRequest,
            models::auth::AuthResponse,
            // Task types
            models::task::Task,
            models::task::CreateTaskRequest,
            models::task::UpdateTaskRequest,
            models::pagination::PaginationParams,
            error::PaginatedTaskResponse,
            // Category types
            models::category::Category,
            models::category::CreateCategoryRequest,
            models::category::UpdateCategoryRequest,
            error::PaginatedCategoryResponse,
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
///
/// This adds the "Authorize" button in Scalar UI that lets you input a JWT token.
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
// The entry point of our application.
//
// # The `#[tokio::main]` Macro
//
// This transforms:
// ```rust,ignore
// #[tokio::main]
// async fn main() { ... }
// ```
//
// Into:
// ```rust,ignore
// fn main() {
//     tokio::runtime::Runtime::new()
//         .unwrap()
//         .block_on(async { ... })
// }
// ```
//
// This is necessary because Rust's built-in main() doesn't support async.

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // ========================================================================
    // STEP 1: LOAD CONFIGURATION
    // ========================================================================
    // Load .env file (if exists) and parse settings from environment variables.
    // The `let _ =` pattern ignores the Result - we don't care if .env is missing
    // because env vars might be set directly (common in production/Docker).

    let _ = dotenvy::dotenv(); // Loads .env file into environment
    let settings = Settings::new()?; // Parse env vars into typed Settings struct

    // ========================================================================
    // STEP 2: INITIALIZE LOGGING (TRACING)
    // ========================================================================
    // We use the `tracing` crate for structured logging. It's more powerful
    // than println! because:
    // - Log levels (debug, info, warn, error)
    // - Structured fields (key=value pairs)
    // - Can send to multiple outputs (console, files, services)
    //
    // The RUST_LOG env var controls what gets logged:
    // - "debug" = everything
    // - "info" = info and above (no debug)
    // - "rust_backend=debug,tower_http=info" = different levels per module

    let log_level = if settings.is_development() {
        "rust_backend=debug,tower_http=debug"
    } else {
        "rust_backend=info,tower_http=info"
    };

    // Build and initialize the tracing subscriber
    // .with() adds "layers" that process log events
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| log_level.into()),
        ))
        .with(tracing_subscriber::fmt::layer().with_target(true))
        .init();

    // Now we can use tracing macros: info!, debug!, warn!, error!
    tracing::info!("Starting Daily Journal API (Rust)");
    tracing::info!("  Environment: {}", settings.app.env);
    tracing::info!("  Server port: {}", settings.server.port);
    tracing::info!("  Database URL: {}:{}", settings.db.host, settings.db.port);
    tracing::info!(
        "  Database NS/DB: {}/{}",
        settings.db.namespace,
        settings.db.database
    );

    // ========================================================================
    // STEP 3: CONNECT TO DATABASE
    // ========================================================================
    // SurrealDB uses WebSocket for persistent connections. This keeps auth
    // state and is more efficient than HTTP for many queries.
    //
    // # The `match` Expression
    // Rust doesn't have try/catch. Instead, functions return `Result<T, E>`.
    // The `match` expression handles both success (Ok) and error (Err) cases.
    //
    // # `tokio::time::timeout`
    // Wraps an async operation with a timeout. Returns:
    // - Ok(Ok(value)) = completed successfully
    // - Ok(Err(e)) = completed with error
    // - Err(_) = timed out

    tracing::info!("Connecting to SurrealDB...");
    let db_url = settings.db.ws_url();
    tracing::debug!("Using WebSocket URL: {}", db_url);

    // Try to connect with a 10-second timeout
    let db: Surreal<Client> = match tokio::time::timeout(
        std::time::Duration::from_secs(10),
        Surreal::new::<Ws>(&db_url),
    )
    .await
    {
        // Connection succeeded
        Ok(Ok(db)) => {
            tracing::info!("Connected to SurrealDB successfully");
            db
        },
        // Connection failed (but didn't timeout)
        Ok(Err(e)) => {
            tracing::error!("Failed to connect to SurrealDB: {}", e);
            tracing::error!("Make sure SurrealDB is running on {}", db_url);
            tracing::error!("You can start it with: task db:up");
            return Err(e.into()); // Early return with error
        },
        // Connection timed out
        Err(_) => {
            tracing::error!("Connection to SurrealDB timed out after 10 seconds");
            tracing::error!("Make sure SurrealDB is accessible at {}", db_url);
            anyhow::bail!("Database connection timeout"); // Macro for returning error
        },
    };

    // ========================================================================
    // STEP 4: AUTHENTICATE TO DATABASE
    // ========================================================================
    // The `?` operator is Rust's way of propagating errors. If signin() returns
    // Err, the function immediately returns that error. If Ok, unwrap the value.

    db.signin(surrealdb::opt::auth::Root {
        username: &settings.db.user,
        password: &settings.db.pass,
    })
    .await?; // <-- If this fails, main() returns the error

    // Select which namespace and database to use
    db.use_ns(&settings.db.namespace)
        .use_db(&settings.db.database)
        .await?;
    tracing::info!(
        "Using namespace: {}, database: {}",
        settings.db.namespace,
        settings.db.database
    );

    // ========================================================================
    // STEP 4B: RUN DATABASE MIGRATIONS
    // ========================================================================
    let migrations_dir = resolve_migrations_dir(settings.db.migrations_path.as_deref());
    tracing::info!(
        path = %migrations_dir.display(),
        "checking for pending database migrations"
    );
    let migration_runner =
        MigrationRunner::new(db.clone(), &migrations_dir, settings.jwt.secret.clone());
    match migration_runner.run_pending().await {
        Ok(applied) if applied.is_empty() => {
            tracing::info!("No pending migrations to apply");
        },
        Ok(applied) => {
            for migration in &applied {
                tracing::info!(
                    version = migration.version,
                    name = %migration.name,
                    "migration applied"
                );
            }
            tracing::info!(count = applied.len(), "finished applying migrations");
        },
        Err(e) => {
            tracing::error!(error = %e, "failed to run migrations");
            return Err(e.into());
        },
    }

    // Initialize schema (creates tables, indexes, etc.)
    let schema_opts = SchemaInitOptions::from(&settings);
    if let Err(e) = init_schema(&db, schema_opts).await {
        // This warning is fine - schema might already exist from previous runs
        tracing::warn!(
            error = %e,
            "Failed to init schema (this is normal if scope already exists)"
        );
    }

    // ========================================================================
    // STEP 5: CREATE APPLICATION STATE (DEPENDENCY INJECTION)
    // ========================================================================
    // This is where we wire up our services. The pattern is:
    // 1. Create concrete service implementations
    // 2. Wrap them in Arc for shared ownership
    // 3. Pass them to AppState
    //
    // # Why Arc::new()?
    // `Arc` = Atomic Reference Counting
    // - Multiple handlers need access to the same service
    // - Arc lets us share without copying the entire service
    // - When the last reference is dropped, the service is cleaned up

    let settings = Arc::new(settings); // Wrap in Arc for sharing

    // Create services - these are the concrete implementations
    // In tests, we'd use MockTaskService, MockCategoryService, and MockAuthService instead
    let task_service = Arc::new(TaskServiceImpl::new(db.clone()));
    let category_service = Arc::new(CategoryServiceImpl::new(db.clone()));
    let auth_service = Arc::new(AuthServiceImpl::new(db.clone(), settings.clone()));

    // Bundle everything into AppState
    let app_state = AppState::new(
        db,
        settings.clone(),
        task_service,
        category_service,
        auth_service,
    );

    // ========================================================================
    // STEP 6: BUILD THE ROUTER
    // ========================================================================
    // Axum's Router is like Express.js routing. We:
    // 1. Define routes and their handlers
    // 2. Add middleware layers
    // 3. Attach shared state
    //
    // # Key Concepts:
    // - `.route("/path", get(handler))` - Maps HTTP method + path to handler
    // - `.merge(other_router)` - Combines routers together
    // - `.nest("/prefix", router)` - Mounts router under a path prefix
    // - `.layer(middleware)` - Adds middleware (runs for every request)
    // - `.with_state(state)` - Makes state available to handlers

    // Create Scalar UI for interactive API documentation
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

    // CORS (Cross-Origin Resource Sharing) configuration
    // This controls which websites can call our API from browsers
    let cors = build_cors_layer(&settings);

    // Build API v1 routes
    // Each module exports a routes() function that returns its Router
    let api_v1 = Router::new()
        .merge(handlers::health::routes())     // /api/v1/health
        .merge(handlers::auth::routes())       // /api/v1/auth/*
        .merge(handlers::task::protected_routes(app_state.clone())) // /api/v1/tasks/*
        .merge(handlers::category::protected_routes(app_state.clone())); // /api/v1/categories/*

    // Combine everything into the main application router
    let app = Router::new()
        // Root health check (for load balancers/k8s probes)
        .route("/health", get(handlers::health::health_check))
        // API documentation
        .merge(scalar_route)
        .route(
            "/api-docs/openapi.json",
            get(|| async { Json(ApiDoc::openapi()) }),
        )
        // All /api/v1/* routes
        .nest("/api/v1", api_v1)
        // Attach our AppState (makes it available to handlers)
        .with_state(app_state)
        // Middleware: request/response tracing (logging)
        .layer(TraceLayer::new_for_http())
        // Middleware: CORS headers
        .layer(cors);

    // ========================================================================
    // STEP 7: START THE SERVER
    // ========================================================================
    // Finally, we bind to a port and start accepting connections.
    //
    // `[0, 0, 0, 0]` means "listen on all network interfaces" - necessary
    // for Docker/containers where the IP isn't known in advance.
    //
    // `.with_graceful_shutdown(shutdown_signal())` means:
    // - When we receive Ctrl+C or SIGTERM, finish current requests
    // - Then shut down cleanly (important for not dropping requests)

    let addr = SocketAddr::from(([0, 0, 0, 0], settings.server.port));
    tracing::info!("Server started on http://{}", addr);
    tracing::info!("Scalar UI available at http://{}/docs", addr);

    // Bind to the address and start accepting connections
    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    tracing::info!("Server shut down gracefully");
    Ok(()) // Everything worked!
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

/// Build CORS layer based on environment configuration
///
/// CORS (Cross-Origin Resource Sharing) controls which websites can call our API.
/// - In development: Allow any origin (easier for testing)
/// - In production: Only allow configured origins (security)
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

        tracing::info!(
            "CORS: restricting to {} configured origin(s)",
            origins.len()
        );
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
///
/// This function returns a Future that completes when we should shut down.
/// It listens for:
/// - Ctrl+C (SIGINT) - When you press Ctrl+C in terminal
/// - SIGTERM - When Docker/Kubernetes wants to stop the container
///
/// # The `tokio::select!` Macro
///
/// `select!` waits for multiple async operations and returns when ANY of them
/// completes. It's like Promise.race() in JavaScript.
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
