# Contributing to Daily Journal

A multi-backend task management application with Go and Rust backends sharing a common SurrealDB database.

## Table of Contents

- [Quick Start](#quick-start)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Go Backend](#go-backend)
- [Rust Backend](#rust-backend)
- [Database](#database)
- [Architecture](#architecture)
- [Adding New Features](#adding-new-features)

---

## Quick Start

```bash
# 1. One-time setup (installs all tools)
task setup

# 2. Start development (choose one)
task dev:go      # Go backend with hot reload
task dev:rust    # Rust backend with hot reload

# Shortcuts
task go          # Same as dev:go
task rust        # Same as dev:rust
```

## Project Structure

```
task/
├── apps/
│   ├── go_backend/         # Go/Chi backend
│   │   ├── cmd/
│   │   │   ├── api/        # Main server entry point
│   │   │   └── gen/        # Feature generator
│   │   └── internal/
│   │       ├── config/     # Configuration
│   │       ├── features/   # Feature modules (vertical slices)
│   │       │   ├── auth/
│   │       │   ├── tasks/
│   │       │   ├── categories/
│   │       │   └── health/
│   │       ├── server/     # HTTP server setup
│   │       └── shared/     # Shared utilities
│   │           ├── database/   # SurrealDB SDK wrapper
│   │           ├── errors/     # Error types
│   │           ├── middleware/ # Auth, logging
│   │           ├── pagination/
│   │           ├── response/   # HTTP responses
│   │           └── validator/
│   │
│   └── rust_backend/       # Rust/Axum backend
│       └── src/
│           ├── bin/        # CLI tools (schema)
│           ├── core/       # Config, DB, errors
│           ├── features/   # Feature modules
│           │   ├── auth/
│           │   ├── tasks/
│           │   ├── categories/
│           │   └── health/
│           ├── shared/     # Shared utilities
│           └── state.rs    # Application state
│
├── db/
│   ├── schema.surql        # Consolidated schema
│   └── migrations/         # Migration files
│
├── deploy/
│   └── docker-compose.yml  # Database container
│
├── Taskfile.yml            # Development commands
└── CONTRIBUTING.md         # This file
```

## Development Workflow

### Essential Commands

| Command | Description |
|---------|-------------|
| `task setup` | One-time setup (install tools) |
| `task dev:go` | Start Go backend with hot reload |
| `task dev:rust` | Start Rust backend with hot reload |
| `task test` | Run all tests |
| `task lint` | Lint all code |
| `task db:up` | Start database |
| `task db:down` | Stop database |
| `task db:reset` | Reset database (DESTRUCTIVE) |

### Go-Specific

| Command | Description |
|---------|-------------|
| `task new:go -- <name>` | Generate new feature |
| `task test:go` | Run Go tests |
| `task lint:go` | Lint & format Go code |
| `task build:go` | Build Go binary |

### Rust-Specific

| Command | Description |
|---------|-------------|
| `task test:rust` | Run Rust tests |
| `task lint:rust` | Lint & format Rust code |
| `task build:rust` | Build Rust binary |
| `task migrate:rust` | Run DB migrations |

---

## Go Backend

### Tech Stack

- **Router**: [Chi](https://github.com/go-chi/chi) - Lightweight HTTP router
- **Database**: [SurrealDB Go SDK](https://surrealdb.com/docs/sdk/golang) - Type-safe operations
- **Validation**: [go-playground/validator](https://github.com/go-playground/validator)
- **Logging**: [zerolog](https://github.com/rs/zerolog) - Structured logging
- **Config**: [Viper](https://github.com/spf13/viper)
- **Hot Reload**: [Air](https://github.com/air-verse/air)

### Feature Structure

Each feature is a self-contained module:

```
internal/features/<feature>/
├── models.go      # Domain types, DTOs, validation
├── repository.go  # Database operations (SurrealDB SDK)
├── service.go     # Business logic
└── handler.go     # HTTP handlers, routes
```

### Creating a New Feature (Go)

```bash
# Generate feature scaffold
task new:go -- comments

# This creates:
# internal/features/comments/
#   ├── models.go      - Domain types
#   ├── repository.go  - SurrealDB SDK operations
#   ├── service.go     - Business logic
#   └── handler.go     - HTTP handlers
```

Then register routes in `internal/server/server.go`:

```go
import "github.com/lucid-logs/go-backend/internal/features/comments"

// In setupRoutes():
commentRepo := comments.NewRepository(cfg.DB)
commentService := comments.NewService(commentRepo)
r.Mount("/comments", comments.Routes(commentService, cfg.Validator))
```

### SurrealDB SDK Usage (Go)

The Go backend uses type-safe SDK wrappers:

```go
// Query all records
users, err := database.QueryAll[User](db, "SELECT * FROM user WHERE ...", vars)

// Query single record
user, err := database.QueryFirst[User](db, "SELECT * FROM user:123", nil)

// Create record
task, err := database.Create[Task](db, "tasks", createData)

// Partial update (MERGE)
task, err := database.Merge[Task](db, "tasks:123", updateData)

// Select by ID
task, err := database.Select[Task](db, "tasks:123")
```

### Hot Reload (Go)

Air watches for file changes and automatically:
- Rebuilds the binary
- Restarts the server
- Shows build errors

Configuration: `apps/go_backend/.air.toml`

---

## Rust Backend

### Tech Stack

- **Framework**: [Axum](https://github.com/tokio-rs/axum) - Async web framework
- **Database**: [SurrealDB](https://surrealdb.com) with native SDK
- **Validation**: [validator](https://crates.io/crates/validator)
- **Logging**: [tracing](https://crates.io/crates/tracing)
- **Hot Reload**: [cargo-watch](https://crates.io/crates/cargo-watch)

### Feature Structure

```
src/features/<feature>/
├── mod.rs         # Module exports, routes
├── model.rs       # Domain types, DTOs
├── repository.rs  # Database operations
├── service.rs     # Business logic
└── handler.rs     # HTTP handlers
```

### Creating a New Feature (Rust)

1. Create the feature directory structure
2. Add module to `src/features/mod.rs`
3. Implement model → repository → service → handler
4. Register routes in `main.rs`

See `apps/rust_backend/CONTRIBUTING.md` for detailed patterns.

### Hot Reload (Rust)

```bash
task dev:rust
# Uses cargo-watch to rebuild on changes
```

---

## Database

### SurrealDB

Both backends share the same SurrealDB instance:

```bash
# Start database
task db:up

# Stop database
task db:down

# Reset (delete all data)
task db:reset
```

### Schema

The consolidated schema is in `db/schema.surql`:

- **Tables**: `user`, `tasks`, `categories`, `_migrations`
- **Functions**: Server-side functions for counting, fetching with relations
- **Permissions**: Row-level security based on `created_by`

### Migrations

Migration files in `db/migrations/`:

```
000_init.surql       # Migration tracking table
001_auth.surql       # User/auth setup
002_categories.surql # Categories table
003_tasks.surql      # Tasks table
...
```

Run migrations:
```bash
task migrate:rust    # Rust migration runner
```

---

## Architecture

### Layered Architecture

Both backends follow the same pattern:

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
│                      SurrealDB                               │
│                  (Shared database)                           │
└─────────────────────────────────────────────────────────────┘
```

### Feature-Based Organization (Vertical Slices)

Instead of grouping by technical layer, code is grouped by feature:

```
features/
├── auth/           # Everything auth-related
├── tasks/          # Everything task-related
├── categories/     # Everything category-related
└── health/         # Health checks
```

Benefits:
- **Cohesion**: Related code lives together
- **Independence**: Features can be developed in isolation
- **Scalability**: Easy to add/remove features

### Standardized Patterns

Both backends use:

1. **Consistent Error Handling**
   ```json
   {
     "error": {
       "code": "NOT_FOUND",
       "message": "Resource not found",
       "details": null
     }
   }
   ```

2. **Pagination**
   ```json
   {
     "items": [...],
     "total": 150,
     "limit": 20,
     "offset": 0,
     "has_more": true
   }
   ```

3. **Soft Deletes**: Records have `deleted_at` field
4. **Ownership**: Records track `created_by`, `updated_by`
5. **Timestamps**: `created_at`, `updated_at` auto-managed

---

## Adding New Features

### Step-by-Step Guide

1. **Define the domain model**
   - What data does this feature manage?
   - What are the create/update DTOs?
   - What validation rules apply?

2. **Create the repository**
   - CRUD operations using SurrealDB SDK
   - Use typed queries for safety
   - Handle soft deletes

3. **Implement the service**
   - Business logic and validation
   - Orchestrate repository calls
   - Transform data as needed

4. **Create HTTP handlers**
   - Parse requests, validate input
   - Call service methods
   - Format responses

5. **Register routes**
   - Add to router configuration
   - Apply authentication middleware

6. **Add database migration** (if new table)
   - Create migration file
   - Define table, indexes, permissions

### Example: Adding "Tags" Feature

```bash
# Go
task new:go -- tag

# Creates internal/features/tag/ with all files
# Then register in server.go
```

For Rust, see the detailed guide in `apps/rust_backend/CONTRIBUTING.md`.

---

## Environment Variables

Copy `env.example` to `.env` and configure:

```bash
# Database
DB_HOST=localhost
DB_PORT=8000
DB_USER=root
DB_PASS=root
DB_NAMESPACE=daily_journal
DB_DATABASE=core

# JWT
JWT_SECRET=your-secret-here

# Admin (for seeding)
ADMIN_USERNAME=admin@example.com
ADMIN_PASSWORD=adminadmin
```

---

## API Endpoints

Both backends expose the same API:

### Public
- `GET /health` - Health check

### Auth
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/register` - Register

### Tasks (Protected)
- `GET /api/v1/tasks` - List (paginated)
- `GET /api/v1/tasks/{id}` - Get
- `POST /api/v1/tasks` - Create
- `PUT /api/v1/tasks/{id}` - Update
- `DELETE /api/v1/tasks/{id}` - Delete

### Categories (Protected)
- `GET /api/v1/categories` - List
- `GET /api/v1/categories/{id}` - Get
- `POST /api/v1/categories` - Create
- `PUT /api/v1/categories/{id}` - Update
- `DELETE /api/v1/categories/{id}` - Delete

---

## Code Quality

### Go
```bash
task lint:go       # Format + lint
task test:go       # Run tests
```

### Rust
```bash
task lint:rust     # Format + clippy
task test:rust     # Run tests
```

### Pre-commit Checklist
1. Format code (`task lint`)
2. Run tests (`task test`)
3. Check for new linter warnings

---

## Troubleshooting

### Database won't start
```bash
task db:reset     # Reset containers
docker logs lucid-logs-surrealdb  # Check logs
```

### Port already in use
```bash
# Find and kill process on port 8080
fuser -k 8080/tcp
```

### Go build fails
```bash
cd apps/go_backend
go mod tidy       # Fix dependencies
```

### Rust build fails
```bash
cd apps/rust_backend
cargo clean       # Clear cache
cargo build       # Rebuild
```

---

## Questions?

Open an issue or check the backend-specific docs:
- `apps/go_backend/README.md`
- `apps/rust_backend/CONTRIBUTING.md`
- `apps/rust_backend/docs/` - Rust learning resources

