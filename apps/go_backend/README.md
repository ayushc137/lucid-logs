# Go Backend

A high-performance, feature-based backend built with Go and Chi, following clean architecture principles.

## Features

- **Feature-Based Architecture**: Vertical slices for each domain (tasks, categories, auth)
- **Chi Router**: Lightweight, idiomatic HTTP router
- **SurrealDB SDK**: Type-safe database operations using official Go SDK
- **JWT Authentication**: Secure token-based auth with SurrealDB integration
- **Standardized Responses**: Consistent API response format
- **Comprehensive Logging**: zerolog with structured logging
- **Request Validation**: go-playground/validator with custom rules
- **Code Quality**: golangci-lint, gofumpt, comprehensive linting
- **Developer Experience**: Hot reload, code snippets, boilr scaffolding

## Quick Start

```bash
# Complete setup (recommended for first time)
make setup

# Or step by step:
make deps          # Install Go dependencies
make tools         # Install dev tools (air, swag, lint, gofumpt)

# Start development
make dev           # Hot reload development server
```

## Project Structure

```
go_backend/
├── cmd/
│   ├── api/                    # Main API server
│   ├── migrate/                # Database migration tool
│   └── gen/                    # Feature generator
├── internal/
│   ├── config/                 # Configuration management
│   ├── features/               # Feature modules (vertical slices)
│   │   ├── auth/               # Authentication feature
│   │   ├── tasks/              # Tasks feature
│   │   ├── categories/         # Categories feature
│   │   └── health/             # Health checks
│   ├── server/                 # Router & server setup
│   └── shared/                 # Shared utilities
│       ├── database/           # SurrealDB SDK wrapper
│       ├── errors/             # Standardized error types
│       ├── response/           # HTTP response helpers
│       ├── pagination/         # Pagination utilities
│       ├── middleware/         # HTTP middleware
│       └── validator/          # Request validation
├── cmd/
│   ├── api/                    # Main API server
│   ├── migrate/                # Database migration tool
│   └── gen/                    # Feature generator tool
├── .vscode/
│   ├── settings.json           # Editor settings
│   └── go.code-snippets        # Code snippets
├── .air.toml                   # Hot reload config
├── .golangci.yml               # Linting config
├── Makefile                    # Development commands
└── Dockerfile                  # Production container
```

## Development Commands

### Quick Reference

```bash
make help          # Show all commands
make setup         # Complete project setup
make dev           # Hot reload development
make check         # Run all checks (fmt, vet, lint, test)
```

### Development

```bash
make dev           # Hot reload with Air
make run           # Build and run
make build         # Build binaries
make clean         # Remove build artifacts
```

### Code Quality

```bash
make fmt           # Format code (gofumpt)
make vet           # Run go vet
make lint          # Run golangci-lint
make lint-fix      # Auto-fix lint issues
make test          # Run tests with coverage
make check         # Run all checks
```

### Database

```bash
make db-start           # Start SurrealDB (memory)
make db-start-file      # Start SurrealDB (file storage)
make migrate-status     # Check migrations
make migrate-up         # Run migrations
make migrate-create name=add_tags  # Create migration
make migrate-reset      # Reset database (DESTRUCTIVE)
```

### Feature Scaffolding

```bash
make new name=comments     # Create new feature with boilr
make feature name=tags     # Alias for 'make new'
```

### Tools

```bash
make tools         # Install all dev tools
make tools-check   # Check tool installation
```

## Creating a New Feature

### Using the Generator

```bash
# Create a new feature
make new name=comments

# This creates:
# internal/features/comments/
#   ├── models.go      # Domain types
#   ├── repository.go  # Data access (SurrealDB SDK)
#   ├── service.go     # Business logic
#   └── handler.go     # HTTP handlers
```

### Register Routes

Add to `internal/server/server.go`:

```go
import "github.com/daily-journal/go-backend/internal/features/comments"

// In setupRoutes():
commentRepo := comments.NewRepository(cfg.DB)
commentService := comments.NewService(commentRepo)
r.Mount("/comments", comments.Routes(commentService, cfg.Validator))
```

### Add Migration

```bash
make migrate-create name=add_comments
```

## API Endpoints

### Public
- `GET /health` - Basic health check
- `GET /docs` - Swagger UI

### Auth
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration

### Tasks (Protected)
- `GET /api/v1/tasks` - List tasks (paginated)
- `GET /api/v1/tasks/{id}` - Get task
- `POST /api/v1/tasks` - Create task
- `PUT /api/v1/tasks/{id}` - Update task
- `DELETE /api/v1/tasks/{id}` - Delete task

### Categories (Protected)
- `GET /api/v1/categories` - List categories
- `GET /api/v1/categories/{id}` - Get category
- `POST /api/v1/categories` - Create category
- `PUT /api/v1/categories/{id}` - Update category
- `DELETE /api/v1/categories/{id}` - Delete category

## Architecture

### Feature-Based Structure

Each feature is a self-contained module with:
- **models.go** - Domain types and DTOs
- **repository.go** - Database operations using SurrealDB SDK
- **service.go** - Business logic
- **handler.go** - HTTP handlers and routes

### SurrealDB SDK Integration

Uses the official SurrealDB Go SDK for type-safe operations:

```go
// Type-safe queries
users, err := database.QueryAll[User](db, "SELECT * FROM user WHERE email = $email", vars)

// Type-safe CRUD
task, err := database.Create[Task](db, "tasks", data)
task, err := database.Select[Task](db, "tasks:abc123")
task, err := database.Merge[Task](db, "tasks:abc123", updates)
_, err := database.Delete[Task](db, "tasks:abc123")
```

SDK methods used:
- `surrealdb.Select[T]()` - Type-safe SELECT
- `surrealdb.Create[T]()` - Type-safe CREATE
- `surrealdb.Update[T]()` - Type-safe UPDATE (full replacement)
- `surrealdb.Merge[T]()` - Type-safe MERGE (partial update)
- `surrealdb.Delete[T]()` - Type-safe DELETE
- `surrealdb.Query[T]()` - Type-safe raw queries

### Standardized Error Handling

```go
// Predefined errors
errors.ErrNotFound
errors.ErrUnauthorized
errors.ErrValidationFailed.WithDetails(validationErrors)
errors.ErrDatabase.Wrap(dbErr)

// Error response format
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found",
    "details": null
  }
}
```

### Standardized Pagination

```go
// Request
GET /api/v1/tasks?limit=20&offset=0

// Response
{
  "data": {
    "items": [...],
    "total": 150,
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}
```

## Code Snippets (VS Code)

Use these snippets for rapid development:

| Prefix | Description |
|--------|-------------|
| `feature` | Full feature scaffold |
| `handler` | HTTP handler |
| `handlerbody` | Handler with JSON body |
| `service` | Service interface + impl |
| `repo` | Repository interface + impl |
| `dbquery` | SurrealDB query |
| `dbcreate` | SurrealDB create |
| `errcheck` | Error check pattern |
| `errlog` | Error with logging |
| `logd/logi/loge` | Log statements |

## Environment Variables

```bash
# Application
APP_ENV=development          # development, production
HTTP_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=8000
DB_USER=root
DB_PASS=root
DB_NAMESPACE=daily_journal
DB_DATABASE=core

# JWT (required in production)
JWT_SECRET=your-secret-here
JWT_EXPIRATION_HOURS=24

# Admin seeding
ADMIN_USERNAME=admin@example.com
ADMIN_PASSWORD=adminadmin
```

## Development Tools

| Tool | Purpose | Install |
|------|---------|---------|
| [Air](https://github.com/air-verse/air) | Hot reload | `make tools` |
| [golangci-lint](https://golangci-lint.run/) | Linting | `make tools` |
| [gofumpt](https://github.com/mvdan/gofumpt) | Formatting | `make tools` |
| [swag](https://github.com/swaggo/swag) | API docs | `make tools` |
| cmd/gen | Feature scaffolding | Built-in |

## License

MIT
