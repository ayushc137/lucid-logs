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

```
src/
├── bin/
│   └── schema.rs       # Database schema CLI
├── config/             # Configuration management
├── error/              # Error types and handling
├── handlers/           # HTTP request handlers (each exports routes())
├── models/             # Data models and DTOs
├── repositories/       # Database access layer
├── services/           # Business logic (async traits for DI)
├── utils/              # Utilities and middleware
├── lib.rs              # Library exports for tests
└── main.rs             # Application entry point

tests/
├── common/             # Shared test utilities, mocks
├── integration/        # API integration tests
└── fixtures/           # Test data
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
- `GET /api/v1/tasks` - List tasks (with pagination)
- `POST /api/v1/tasks` - Create task
- `PUT /api/v1/tasks/{id}` - Update task
- `DELETE /api/v1/tasks/{id}` - Delete task

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

### Modular Routes

Each handler module exports a `routes()` function:

```rust
// handlers/my_feature.rs
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/my-feature", get(list_items))
}
```

### Service Layer with DI

Business logic is encapsulated in services with trait-based DI:

```rust
// Define trait
#[async_trait]
pub trait MyService: Send + Sync {
    async fn do_thing(&self) -> Result<(), AppError>;
}

// Implement
pub struct MyServiceImpl { /* ... */ }

#[async_trait]
impl MyService for MyServiceImpl {
    async fn do_thing(&self) -> Result<(), AppError> { /* ... */ }
}

// Wire up via AppState
let service = Arc::new(MyServiceImpl::new());
let state = AppState::new(/* ... */, service);
```

### Generic Repository

Use the base repository trait for consistent CRUD operations:

```rust
use crate::repositories::base::{Repository, ensure_record_id};
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
