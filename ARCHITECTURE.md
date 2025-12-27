# Architecture Guide

This document provides an overview of the Lucid Logs Daily Journaling App architecture for developers and AI agents.

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Backend | Go 1.21+ Gin | REST API server |
| Frontend | SvelteKit + TypeScript | Web UI |
| Database | SurrealDB | Document/graph database |
| Auth | JWT tokens | Bearer authentication |

---

## Directory Structure

```
lucid-logs/
├── apps/
│   ├── go_backend/
│   │   ├── cmd/                    # Entry points
│   │   │   └── api/main.go         # Main API server
│   │   ├── internal/
│   │   │   ├── features/           # Domain modules
│   │   │   │   ├── tasks/          # Task management
│   │   │   │   ├── categories/     # Category/tag system
│   │   │   │   ├── goals/          # Goal tracking
│   │   │   │   ├── emotions/       # Mood/emotion data
│   │   │   │   ├── auth/           # Authentication
│   │   │   │   ├── users/          # User management
│   │   │   │   ├── templates/      # Task templates
│   │   │   │   ├── analytics/      # Analytics/stats
│   │   │   │   ├── retrospectives/ # Retro entries
│   │   │   │   ├── goalactions/    # Goal action links
│   │   │   │   ├── goalentries/    # Goal progress entries
│   │   │   │   ├── taskgoals/      # Task-goal links
│   │   │   │   └── health/         # Health check endpoint
│   │   │   ├── shared/             # Cross-cutting utilities
│   │   │   │   ├── database/       # SurrealDB client
│   │   │   │   ├── errors/         # Error types
│   │   │   │   ├── middleware/     # Auth, logging middleware
│   │   │   │   ├── pagination/     # Pagination helpers
│   │   │   │   ├── response/       # HTTP response helpers
│   │   │   │   ├── timeutil/       # Time utilities
│   │   │   │   └── validator/      # Request validation
│   │   │   ├── bootstrap/          # Admin seeding
│   │   │   ├── config/             # Configuration
│   │   │   ├── scheduler/          # Background jobs
│   │   │   ├── server/             # Server setup
│   │   │   └── test/               # Test utilities
│   │   └── docs/                   # Swagger/OpenAPI
│   └── frontend/
│       └── src/
│           ├── lib/
│           │   ├── api/            # API client modules
│           │   ├── components/     # Svelte components
│           │   ├── stores/         # Svelte stores
│           │   └── assets/         # Static assets
│           └── routes/             # SvelteKit pages
├── db/
│   ├── schema.surql                # Full database schema reference
│   └── migrations/                 # Database migrations
├── deploy/                         # Docker compose files
└── docs/                           # Documentation
```

---

## Feature Module Pattern

Each feature in `internal/features/` follows a consistent structure:

```
features/{name}/
├── handler.go      # HTTP handlers & route registration
├── service.go      # Business logic layer
├── repository.go   # Database operations
└── models.go       # Types, DTOs, request/response structs
```

### Responsibilities

| File | Purpose |
|------|---------|
| `handler.go` | Parse requests, call service, format responses. Contains Swagger annotations. |
| `service.go` | Business logic, validation rules, orchestration between repos. |
| `repository.go` | Raw database queries using SurrealDB. |
| `models.go` | Struct definitions, JSON tags, validation tags. |

---

## Adding a New Feature

### Backend

1. Create directory: `internal/features/{name}/`
2. Create 4 files following the pattern above
3. Register routes in `cmd/api/main.go`:
   ```go
   featureGroup := r.Group("/api/v1/{name}")
   featureGroup.Use(authMiddleware)
   {name}.RegisterRoutes(featureGroup, {name}Service, validator)
   ```
4. Run `make swagger` to update API docs

### Frontend

1. Create API module: `src/lib/api/{name}.ts`
2. Export types and functions from `src/lib/api/index.ts`
3. Create route: `src/routes/{name}/+page.svelte`

---

## Database Patterns

### SurrealDB Record IDs

All IDs are in format `table:id`, e.g., `tasks:abc123`, `categories:work`.

### Record Links (Relations)

```sql
-- Task links to category
task.category = categories:work123

-- Query with fetch
SELECT *, category.* FROM tasks FETCH category
```

### Multi-tenancy

All tables use `created_by` field for user isolation. Queries always filter:
```sql
SELECT * FROM tasks WHERE created_by = $user_id AND deleted_at = NONE
```

---

## API Conventions

### Response Format

```json
{
  "data": { ... },  // For successful responses
  "error": {        // For error responses
    "code": "NOT_FOUND",
    "message": "Task not found"
  }
}
```

### Pagination

```json
{
  "items": [...],
  "total": 100,
  "limit": 20,
  "offset": 0,
  "has_more": true
}
```

### Common Query Parameters

| Param | Description |
|-------|-------------|
| `limit` | Items per page (default 20, max 100) |
| `offset` | Items to skip |
| `search` | Full-text search |
| `sort_field` | Field to sort by |
| `sort_order` | `asc` or `desc` |

---

## Authentication

1. User logs in via `POST /api/v1/auth/login`
2. Server returns JWT token
3. Client stores token in `sessionStorage`
4. All requests include `Authorization: Bearer {token}`
5. 401 response triggers logout and redirect to `/login`

---

## Key Files for Reference

| Purpose | File |
|---------|------|
| Feature example | `internal/features/tasks/` |
| Error definitions | `internal/shared/errors/errors.go` |
| Response helpers | `internal/shared/response/response.go` |
| Frontend API pattern | `src/lib/api/tasks.ts` |
| Database schema | `db/schema.surql` |
