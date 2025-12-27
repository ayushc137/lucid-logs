# AI Agent Context for Lucid Logs

Quick reference for AI coding assistants working on this codebase.

## Quick Facts

- **Backend**: Go 1.21+ with Gin framework
- **Frontend**: SvelteKit + TypeScript
- **Database**: SurrealDB (schemaless with record links)
- **Auth**: JWT Bearer tokens in `sessionStorage`
- **API Base**: `/api/v1`

---

## Common Operations

### Backend: Add New CRUD Endpoint

```bash
# 1. Create feature directory
mkdir -p apps/go_backend/internal/features/{name}

# 2. Copy pattern from tasks feature
# 3. Modify handler.go, service.go, repository.go, models.go
# 4. Register routes in cmd/api/main.go
# 5. Regenerate swagger
cd apps/go_backend && make swagger
```

### Frontend: API Call Pattern

```typescript
import { api, unwrap } from '$lib/api';

// GET request
const items = await unwrap<Item[]>(api.get('items'));

// POST with body
const created = await unwrap<Item>(
  api.post('items', { json: { title: 'New' }})
);

// DELETE
await unwrap<void>(api.delete(`items/${id}`));
```

### Frontend: Add New API Module

```typescript
// src/lib/api/{name}.ts
import { api, unwrap, type PaginatedResponse } from './client';

export interface MyType {
  id: string;
  title: string;
  created_at: string;
}

export async function getItems(): Promise<PaginatedResponse<MyType>> {
  return unwrap(api.get('items'));
}

// Then export from src/lib/api/index.ts
```

---

## Database Query Patterns

### SurrealDB Record IDs

IDs include table name: `tasks:abc123`, `categories:work`

### Basic CRUD in Go

```go
// Create
result, err := db.Query("CREATE tasks CONTENT $data RETURN *", map[string]any{
    "data": map[string]any{
        "title":      req.Title,
        "created_by": userID,
    },
})

// Read with relation
result, err := db.Query(`
    SELECT *, category.* FROM tasks 
    WHERE id = $id AND created_by = $user 
    FETCH category
`, map[string]any{"id": id, "user": userID})

// Update
result, err := db.Query(`
    UPDATE $id SET title = $title, updated_at = time::now()
`, map[string]any{"id": id, "title": newTitle})

// Soft Delete
result, err := db.Query(`
    UPDATE $id SET deleted_at = time::now()
`, map[string]any{"id": id})
```

---

## Error Handling

### Backend Errors

Use predefined errors from `internal/shared/errors`:

```go
import "github.com/lucid-logs/go-backend/internal/shared/errors"

// Return not found
return errors.ErrNotFound.WithMessage("Task not found")

// Return validation error
return errors.ErrValidationFailed.WithDetails(validationErrors)

// Return conflict (duplicate)
return errors.ErrConflict.WithMessage("Category already exists")
```

### Frontend Error Display

```typescript
try {
  await createTask(data);
} catch (error) {
  // error.message contains the backend error message
  showError(error instanceof Error ? error.message : 'Unknown error');
}
```

---

## Files to Read First

| When You Need To... | Read This |
|---------------------|-----------|
| Add a new feature | `internal/features/tasks/` (handler, service, repo, models) |
| Understand errors | `internal/shared/errors/errors.go` |
| Add frontend API | `src/lib/api/tasks.ts` |
| Understand DB schema | `db/schema.surql` |
| See all routes | `apps/go_backend/docs/swagger.json` |

---

## Code Style Notes

### Go

- Use section separators: `// ===== SECTION =====`
- Add Swagger annotations to all handlers
- Always check `created_by` for multi-tenancy
- Soft delete with `deleted_at`, never hard delete

### TypeScript/Svelte

- Use `$lib/api` for all API calls
- Type all API responses
- Store auth token in sessionStorage
- Handle 401 with redirect to /login

---

## Running Locally

```bash
# Start everything (from project root)
task dev

# Backend only
cd apps/go_backend && air

# Frontend only  
cd apps/frontend && pnpm dev
```
