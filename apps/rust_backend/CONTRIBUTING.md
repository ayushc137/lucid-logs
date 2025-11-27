# Contributing to Daily Journal Rust Backend

This document describes the architecture patterns, conventions, and development workflow for the Rust/Axum backend.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Module Structure](#module-structure)
- [Adding New Features](#adding-new-features)
- [Service Layer & Dependency Injection](#service-layer--dependency-injection)
- [Repository Pattern](#repository-pattern)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Development Workflow](#development-workflow)

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         Handlers                             │
│  (HTTP layer - routing, request parsing, response formatting)│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                         Services                             │
│        (Business logic - validation, orchestration)          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Repositories                           │
│              (Data access - database operations)             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        Database                              │
│                       (SurrealDB)                            │
└─────────────────────────────────────────────────────────────┘
```

## Module Structure

```
src/
├── bin/
│   └── schema.rs           # Schema CLI binary
├── config/
│   └── mod.rs              # Application configuration
├── error/
│   └── mod.rs              # Error types and API response wrappers
├── handlers/
│   ├── mod.rs
│   ├── auth.rs             # Auth endpoints (exports routes())
│   ├── health.rs           # Health endpoints (exports routes())
│   └── task.rs             # Task endpoints (exports routes())
├── models/
│   ├── mod.rs
│   ├── auth.rs             # Auth DTOs
│   └── task.rs             # Task domain model
├── repositories/
│   ├── mod.rs
│   ├── base.rs             # Generic CRUD trait
│   ├── schema.rs           # Schema initialization
│   └── task.rs             # Task repository
├── services/
│   ├── mod.rs
│   ├── traits.rs           # Service trait definitions
│   ├── auth.rs             # Auth service implementation
│   └── task.rs             # Task service implementation
├── utils/
│   ├── mod.rs
│   ├── middleware.rs       # Auth middleware
│   └── state.rs            # AppState definition
├── lib.rs                  # Library exports for tests
└── main.rs                 # Application entry point
```

## Adding New Features

### 1. Adding a New Handler Module

Each handler module should export a `routes()` function:

```rust
// src/handlers/my_feature.rs
use axum::{routing::get, Router};
use crate::utils::state::AppState;

/// Create routes for my feature
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/my-feature", get(list_items))
        .route("/my-feature/{id}", get(get_item))
}

async fn list_items(/* ... */) { /* ... */ }
async fn get_item(/* ... */) { /* ... */ }
```

Then merge in `main.rs`:

```rust
let api_v1 = Router::new()
    .merge(handlers::health::routes())
    .merge(handlers::auth::routes())
    .merge(handlers::my_feature::routes());  // Add here
```

### 2. Adding a New Service

1. Define the trait in `services/traits.rs`:

```rust
#[async_trait]
pub trait MyFeatureService: Send + Sync {
    async fn do_something(&self, input: Input) -> Result<Output, AppError>;
}
```

2. Implement in `services/my_feature.rs`:

```rust
pub struct MyFeatureServiceImpl {
    repo: MyFeatureRepository,
}

#[async_trait]
impl MyFeatureService for MyFeatureServiceImpl {
    async fn do_something(&self, input: Input) -> Result<Output, AppError> {
        // Business logic here
    }
}
```

3. Add to `AppState` in `utils/state.rs`:

```rust
pub struct AppState {
    // ... existing fields
    pub my_feature_service: Arc<dyn MyFeatureService>,
}
```

4. Wire up in `main.rs`:

```rust
let my_feature_service = Arc::new(MyFeatureServiceImpl::new(db.clone()));
let app_state = AppState::new(/* ... */, my_feature_service);
```

### 3. Adding a New Repository

1. Create `repositories/my_entity.rs`:

```rust
use crate::repositories::base::{ensure_record_id, CountResult};

const TABLE: &str = "my_entities";

pub struct MyEntityRepository {
    db: Surreal<Client>,
}

impl MyEntityRepository {
    pub fn new(db: Surreal<Client>) -> Self {
        Self { db }
    }

    pub async fn create(&self, req: CreateRequest, user_id: &str) -> Result<Entity, AppError> {
        // Implementation
    }

    // ... other methods
}
```

2. Export in `repositories/mod.rs`:

```rust
pub mod my_entity;
pub use my_entity::MyEntityRepository;
```

## Service Layer & Dependency Injection

Services are defined as traits, enabling:
- **Testability**: Mock implementations for unit tests
- **Flexibility**: Swap implementations at runtime
- **Decoupling**: Handlers don't depend on concrete implementations

### Production Setup

```rust
let task_service = Arc::new(TaskServiceImpl::new(db.clone()));
let auth_service = Arc::new(AuthServiceImpl::new(db.clone(), settings.clone()));

let app_state = AppState::new(db, settings, task_service, auth_service);
```

### Test Setup

```rust
use crate::tests::common::{MockTaskService, MockAuthService};

let mock_task_service = Arc::new(MockTaskService::with_tasks(vec![/* ... */]));
let mock_auth_service = Arc::new(MockAuthService::with_user("test", "pass"));

let app_state = AppState::new(/* ... */, mock_task_service, mock_auth_service);
```

## Repository Pattern

### Generic Repository Trait

The `Repository` trait in `repositories/base.rs` defines a standard CRUD interface:

```rust
#[async_trait]
pub trait Repository<T, CreateReq, UpdateReq, Id>: Send + Sync
where
    T: DeserializeOwned + Serialize + Send + Sync,
{
    async fn create(&self, req: CreateReq, owner_id: &str) -> Result<T, AppError>;
    async fn find_by_id(&self, id: &Id, owner_id: &str) -> Result<T, AppError>;
    async fn find_by_owner_paginated(&self, owner_id: &str, limit: i64, offset: i64) -> Result<(Vec<T>, i64), AppError>;
    async fn update(&self, id: &Id, req: UpdateReq, owner_id: &str) -> Result<T, AppError>;
    async fn delete(&self, id: &Id, owner_id: &str) -> Result<(), AppError>;
}
```

### Helper Functions

- `ensure_record_id(id, table)`: Ensures SurrealDB record IDs have the table prefix
- `CountResult`: Helper struct for count aggregation queries

## Error Handling

### AppError Enum

All errors should be converted to `AppError`:

```rust
pub enum AppError {
    Database(surrealdb::Error),
    Validation(validator::ValidationErrors),
    NotFound,
    Unauthorized(String),
    Internal,
    BadRequest(String),
}
```

### In Handlers

```rust
pub async fn my_handler(/* ... */) -> Result<Json<Response>, AppError> {
    let result = service.do_something().await?;  // ? converts to AppError
    Ok(Json(ApiResponse::success(result)))
}
```

### API Response Format

All responses use the `ApiResponse` wrapper:

```json
{
    "data": { /* success data */ },
    "error": null
}
```

Or on error:

```json
{
    "data": null,
    "error": {
        "code": "VALIDATION_FAILED",
        "message": "Request validation failed",
        "details": [/* ... */]
    }
}
```

## Testing

### Test Structure

```
tests/
├── common/
│   └── mod.rs          # Mock services, test utilities
├── integration/
│   └── *.rs            # API integration tests
└── fixtures/
    └── mod.rs          # Static test data
```

### Running Tests

```bash
# Run all tests with nextest (faster)
task rust:test

# Run tests in watch mode
task rust:test:watch

# Run specific test
cargo nextest run test_name

# Run with output
cargo nextest run -- --nocapture
```

### Writing Tests

Use the provided mocks:

```rust
use crate::tests::common::{MockTaskService, fixtures};

#[tokio::test]
async fn test_create_task() {
    let mock_service = MockTaskService::new();
    let request = fixtures::create_task_request("Test Task");
    
    let result = mock_service.create_task(request, "user:123").await;
    assert!(result.is_ok());
}
```

### Snapshot Testing

Use `insta` for snapshot tests:

```rust
use insta::assert_json_snapshot;

#[test]
fn test_response_format() {
    let response = ApiResponse::success(data);
    assert_json_snapshot!(response);
}
```

## Code Quality

### Linting

```bash
# Run clippy and format check
task rust:lint

# Auto-fix formatting
task rust:fmt
```

### Security

```bash
# Run security audit
task rust:audit

# Find unused dependencies
task rust:machete
```

### Pre-commit Checklist

1. `task rust:fmt` - Format code
2. `task rust:lint` - Check lints
3. `task rust:test` - Run tests
4. `task rust:audit` - Security check

## Development Workflow

### Quick Start

```bash
# Install tools (one time)
task rust:tools

# Start database
task db:up

# Run in development mode with hot reload
task rust:dev
```

### Schema Management

```bash
# Apply migrations
task rust:schema:migrate

# Check status
task rust:schema:status

# Reset database (DESTRUCTIVE!)
task rust:schema -- reset --force
```

### Useful Commands

| Command | Description |
|---------|-------------|
| `task rust:dev` | Start with hot reload |
| `task rust:test` | Run tests |
| `task rust:lint` | Check code quality |
| `task rust:doc` | Generate docs |
| `task rust:bacon` | Background checker |
| `task rust:audit` | Security scan |

## Conventions

### Naming

- **Files**: `snake_case.rs`
- **Types**: `PascalCase`
- **Functions**: `snake_case`
- **Constants**: `SCREAMING_SNAKE_CASE`

### Imports

Organize imports in this order:
1. Standard library
2. External crates
3. Internal crates (`crate::`)

### Documentation

- All public items should have doc comments
- Use `///` for items, `//!` for modules
- Include examples where helpful

### Async

- Prefer `async fn` over `impl Future`
- Use `tokio::spawn` for background tasks
- Handle cancellation gracefully

---

Questions? Open an issue or reach out to the maintainers.

