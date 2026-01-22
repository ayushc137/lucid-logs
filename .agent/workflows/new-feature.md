---
description: Create a new feature (backend + frontend)
---

# Create a New Feature

This workflow helps you scaffold a complete new feature.

## Step 1: Gather Requirements
Ask for:
- Feature name (e.g., "comments", "tags", "notifications")
- What data it manages (fields and types)
- Related features/tables

## Step 2: Backend Setup

1. Create the feature directory:
```bash
mkdir -p apps/go_backend/internal/features/{name}
```

2. Create the 4 required files following patterns:
   - `models.go` - Domain types, DTOs, validation structs
   - `repository.go` - SurrealDB operations
   - `service.go` - Business logic
   - `handler.go` - HTTP handlers with Swagger annotations

   Reference templates at: `.agent/templates/feature-handler.md`

3. Register routes in `apps/go_backend/cmd/api/main.go`:
```go
{name}Repo := {name}.NewRepository(dbClient)
{name}Service := {name}.NewService({name}Repo)
{name}Group := api.Group("/{name}")
{name}Group.Use(authMiddleware.Auth())
{name}.RegisterRoutes({name}Group, {name}Service, validatorInstance)
```

4. Regenerate Swagger docs:
```bash
cd apps/go_backend && make swagger
```

## Step 3: Database Migration (if new table)

1. Create migration file in `db/migrations/`:
```bash
touch db/migrations/0XX_{name}.surql
```

2. Add to `db/schema.surql` for reference

## Step 4: Frontend API Module

1. Create API module following `.agent/templates/frontend-api.md`:
```bash
touch apps/frontend/src/lib/api/{name}.ts
```

2. Export from `apps/frontend/src/lib/api/index.ts`

## Step 5: Frontend Route (Optional)

1. Create page:
```bash
mkdir -p apps/frontend/src/routes/{name}
touch apps/frontend/src/routes/{name}/+page.svelte
```

## Step 6: Verify

// turbo
```bash
cd apps/frontend && pnpm check
```

// turbo
```bash
cd apps/go_backend && golangci-lint run ./...
```
