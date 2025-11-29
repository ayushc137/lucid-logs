# Rust Backend

A high-performance backend built with Axum, following industry best practices.

## Features

- **Axum Framework**: Fast, ergonomic web framework
- **SurrealDB**: Modern multi-model database
- **JWT Authentication**: Secure token-based auth
- **Service Layer**: Business logic with DI support
- **Auto-reload**: Development server with cargo-watch
- **Structured Logging**: tracing and tracing-subscriber
- **Validation**: Request validation with validator
- **Error Handling**: Comprehensive error types with thiserror
- **OpenAPI/Scalar**: Interactive API documentation

## Project Structure

The codebase follows a **feature-based (vertical slice) architecture**:

```
src/
├── bin/
│   └── schema.rs           # Database schema CLI
├── core/                   # Shared infrastructure
│   ├── config.rs           # Configuration management
│   ├── error.rs            # Error types and API response wrappers
│   ├── middleware.rs       # Auth middleware
│   ├── db/                 # Database utilities
│   │   ├── migrations.rs   # Migration runner
│   │   └── schema.rs       # Schema initialization
│   └── mod.rs
├── features/               # Feature modules (vertical slices)
│   ├── auth/               # Authentication feature
│   │   ├── handler.rs      # HTTP handlers
│   │   ├── model.rs        # DTOs and domain models
│   │   └── service.rs      # Business logic
│   ├── categories/         # Category management
│   │   ├── handler.rs
│   │   ├── model.rs
│   │   ├── repository.rs   # Database operations
│   │   └── service.rs
│   ├── tasks/              # Task management
│   │   ├── handler.rs
│   │   ├── model.rs
│   │   ├── repository.rs
│   │   └── service.rs
│   └── health/             # Health checks
│       └── handler.rs
├── shared/                 # Cross-feature utilities
│   ├── db/                 # Database types and queries
│   │   ├── types.rs        # Type-safe record IDs (TaskId, CategoryId)
│   │   ├── queries.rs      # Centralized SQL query registry
│   │   └── result.rs       # Common result types
│   ├── pagination.rs       # Pagination parameters
│   └── repository.rs       # Repository utilities
├── state.rs                # Application state (AppState)
├── lib.rs                  # Library exports for tests
└── main.rs                 # Application entry point

tests/
├── common/                 # Shared test utilities, mocks
├── integration/            # API integration tests
└── unit/                   # Unit tests
```

## Setup

1. Copy the environment file:
```bash
cp .env.example .env
```

2. Install development tools:
```bash
task rust:tools
```

This installs:
- `cargo-watch` - Hot reload during development
- `cargo-nextest` - Faster test runner
- `cargo-audit` - Security vulnerability scanner
- `cargo-deny` - License/dependency checks
- `cargo-machete` - Unused dependency detection
- `cargo-expand` - Macro expansion debugging
- `bacon` - Background code checker

3. Start the database (from project root):
```bash
task db:up
```

4. Run the development server:
```bash
task rust:dev
```

## Development Commands

| Command | Description |
|---------|-------------|
| `task rust:dev` | Start with hot reload |
| `task rust:build` | Build the project |
| `task rust:test` | Run tests with nextest |
| `task rust:test:watch` | Run tests in watch mode |
| `task rust:lint` | Run clippy + format check |
| `task rust:fmt` | Format code |
| `task rust:audit` | Security audit |
| `task rust:machete` | Find unused deps |
| `task rust:doc` | Generate documentation |
| `task rust:bacon` | Background checker |

## Schema Management

The schema CLI provides database management commands:

```bash
# Apply migrations
task rust:schema:migrate

# Check database status
task rust:schema:status

# Reset database (DESTRUCTIVE!)
task rust:schema -- reset --force
```

## API Endpoints

### Public
- `GET /health` - Health check
- `GET /api/v1/health` - Health check with DB status
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration

### Protected (requires JWT)

**Tasks**
- `GET /api/v1/tasks` - List tasks (with pagination)
- `GET /api/v1/tasks/{id}` - Get task by ID
- `POST /api/v1/tasks` - Create task
- `PUT /api/v1/tasks/{id}` - Update task
- `DELETE /api/v1/tasks/{id}` - Delete task (soft delete)

**Categories**
- `GET /api/v1/categories` - List categories (with pagination)
- `GET /api/v1/categories/{id}` - Get category by ID
- `POST /api/v1/categories` - Create category
- `PUT /api/v1/categories/{id}` - Update category
- `DELETE /api/v1/categories/{id}` - Delete category (soft delete)

### Documentation
- `GET /docs` - Scalar API documentation UI
- `GET /api-docs/openapi.json` - OpenAPI spec

## Environment Variables

See `.env.example` for all available configuration options.

Key variables:
- `APP_ENV` - Environment (development/production)
- `HTTP_PORT` - Server port (default: 8080)
- `DB_HOST` - SurrealDB host
- `DB_PORT` - SurrealDB port (default: 8000)
- `JWT_SECRET` - JWT signing secret (required in production)

## Architecture Patterns

### Feature-Based Structure

Each feature is a self-contained module with handler, model, service, and repository:

```rust
// features/my_feature/handler.rs
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/my-feature", get(list_items))
        .route("/my-feature", post(create_item))
}

pub fn protected_routes(state: AppState) -> Router<AppState> {
    routes().layer(middleware::from_fn_with_state(state, auth_middleware))
}
```

### Service Layer with DI

Business logic is encapsulated in services with trait-based dependency injection:

```rust
// features/my_feature/service.rs

// Define trait for testability
#[async_trait]
pub trait MyService: Send + Sync {
    async fn do_thing(&self) -> Result<MyItem, AppError>;
}

// Production implementation
pub struct MyServiceImpl {
    repo: MyRepository,
}

#[async_trait]
impl MyService for MyServiceImpl {
    async fn do_thing(&self) -> Result<MyItem, AppError> {
        // Business logic + repository calls
        self.repo.create(/* ... */).await
    }
}

// Wire up via AppState in main.rs
let service = Arc::new(MyServiceImpl::new(db.clone()));
let state = AppState::new(db, settings, service, /* ... */);
```

### Type-Safe Database IDs

Use the type-safe ID wrappers from `shared/db/types.rs`:

```rust
use crate::shared::{TaskId, CategoryId};

// Type-safe ID creation
let task_id = TaskId::new("abc123");
assert_eq!(task_id.full_id(), "tasks:abc123");

// Works seamlessly with the SurrealDB fluent builders
let task: Option<Task> = db.select(task_id.as_thing()).await?;
```

### Fluent Builders & SurrealDB Functions

Repositories use SurrealDB's builder API plus schema-defined functions for complex logic:

```rust
// Mutation via builder (type-safe payloads)
let inserted: Vec<Task> = db
    .create("tasks")
    .content(task_payload)
    .await?;

// Server-side function for reusable logic
let mut result = db
    .query("RETURN fn::task::with_category(type::thing($id))")
    .bind(("id", task_id.full_id()))
    .await?;

let task_with_category: Vec<Task> = result.take(0)?;
```

## Testing

```bash
# Run all tests
task rust:test

# Run specific test
cargo nextest run test_name

# Run with output
cargo nextest run -- --nocapture
```

See `tests/common/mod.rs` for mock implementations.

## Code Quality

The project enforces strict linting via:
- `.cargo/config.toml` - Clippy configuration
- `rustfmt.toml` - Formatting rules
- `deny.toml` - Dependency checks
- `Cargo.toml` - Lint settings

Run checks with:
```bash
task rust:lint
```

## Building for Production

```bash
cargo build --release
```

The optimized binary will be in `target/release/rust_backend`.

Production optimizations (see `Cargo.toml`):
- LTO (Link-Time Optimization)
- Single codegen unit
- Panic abort
- Symbol stripping

## For Rust Beginners

New to Rust? We've got you covered! Check out these resources in the `docs/` folder:

| Document | Description |
|----------|-------------|
| [RUST_FOR_BEGINNERS.md](./docs/RUST_FOR_BEGINNERS.md) | Learn Rust concepts through our actual code |
| [CODE_WALKTHROUGH.md](./docs/CODE_WALKTHROUGH.md) | Follow a request through the codebase |
| [RUST_CHEATSHEET.md](./docs/RUST_CHEATSHEET.md) | Quick reference for common patterns |
| [TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md) | Common errors and how to fix them |

### Quick Help Commands

```bash
task rust:help           # Show all available commands
task rust:check          # Quick type check (faster than build)
task rust:explain -- E0382  # Explain a compiler error
task rust:new:handler    # Guide for creating a new handler
task rust:new:service    # Guide for creating a new service
```

### VSCode/Cursor Setup

The `.vscode/` folder includes recommended settings and extensions. Install them for:
- Inline error display
- Auto-formatting on save
- Code snippets for handlers, services, etc.
- Debugging configuration

Type `handler` in a `.rs` file and press Tab to generate a handler template!

### API Testing

Interactive API documentation is available at `/docs` (Scalar UI):
```bash
task rust:dev                    # Start the server
open http://localhost:8080/docs  # Open API docs
```

The docs are auto-generated from code annotations - no manual maintenance needed!

### Pre-commit Hooks

Install git hooks to catch issues before commit:
```bash
task rust:hooks   # Install lefthook hooks
# Now commits automatically run: fmt check, clippy, type check
```

### Dev Container (One-Click Setup)

Open in VS Code/Cursor and click "Reopen in Container" for a fully configured environment:
- Rust toolchain pre-installed
- All cargo tools ready
- SurrealDB running
- Git hooks configured

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for detailed guidelines on:
- Adding new features
- Service layer patterns
- Testing conventions
- Code style
