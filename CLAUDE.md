# Lucid Logs - Claude Code Instructions

A daily journaling and task management app with Go backend, SvelteKit frontend, and SurrealDB database.

---

## Quick Start

```bash
# Start everything (backend + frontend + database)
task dev

# Or individually:
task go          # Go backend with hot reload on :8080
task fe          # Frontend on :5173
task db:up       # Start SurrealDB
```

**Application URLs:**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080/api/v1
- Swagger Docs: http://localhost:8080/swagger/index.html

---

## Tech Stack

| Component | Technology | Details |
|-----------|------------|---------|
| **Backend** | Go 1.21+ / Gin | REST API, Swagger docs, Air hot reload |
| **Frontend** | SvelteKit + Svelte 5 | TypeScript, TailwindCSS 4, DaisyUI 5 |
| **Database** | SurrealDB | Document/graph database with record links |
| **API Client** | ky | HTTP client with auto token handling |
| **State** | TanStack Query | Server state management |
| **Rich Text** | TipTap | Rich text editor for journals |
| **Auth** | JWT tokens | Stored in sessionStorage |

---

## Project Structure

```
lucid-logs/
├── apps/
│   ├── go_backend/
│   │   ├── cmd/
│   │   │   ├── api/main.go         # Main server entry
│   │   │   ├── gen/                # Feature generator
│   │   │   └── seed/               # Test data seeder
│   │   ├── internal/
│   │   │   ├── features/           # Feature modules (vertical slices)
│   │   │   │   ├── tasks/          # Task management
│   │   │   │   ├── goals/          # Goal/habit tracking
│   │   │   │   ├── categories/     # Organization
│   │   │   │   ├── emotions/       # Mood tracking
│   │   │   │   ├── templates/      # Task templates
│   │   │   │   ├── retrospectives/ # Daily/weekly reflections
│   │   │   │   ├── taskgoals/      # Task-goal linking
│   │   │   │   ├── goallogs/       # Goal history/changelog
│   │   │   │   ├── units/          # Measurement units
│   │   │   │   ├── analytics/      # Stats/insights
│   │   │   │   ├── auth/           # Authentication
│   │   │   │   └── users/          # User management
│   │   │   └── shared/             # Cross-cutting utilities
│   │   │       ├── database/       # SurrealDB client wrappers
│   │   │       ├── errors/         # Predefined AppError types
│   │   │       ├── middleware/     # Auth, logging
│   │   │       ├── pagination/     # Pagination helpers
│   │   │       ├── response/       # HTTP response helpers
│   │   │       └── validator/      # Request validation
│   │   └── docs/                   # Generated Swagger docs
│   └── frontend/
│       └── src/
│           ├── lib/
│           │   ├── api/            # API client modules (ky-based)
│           │   ├── components/
│           │   │   ├── ui/         # Reusable UI primitives
│           │   │   ├── tasks/      # Task-specific components
│           │   │   ├── goals/      # Goal-specific components
│           │   │   ├── timeline/   # Timeline/Gantt views
│           │   │   ├── emotions/   # Emotion picker/display
│           │   │   └── layout/     # Layout components
│           │   ├── stores/         # Svelte stores (auth, theme, ui)
│           │   ├── utils/          # Utility functions
│           │   └── DESIGN_LANGUAGE.md  # UI component patterns
│           └── routes/             # SvelteKit pages
│               ├── +page.svelte    # Dashboard (main timeline view)
│               ├── tasks/          # Task list/detail
│               ├── goals/          # Goals management
│               ├── categories/     # Category management
│               ├── templates/      # Task templates
│               ├── retrospectives/ # Reflections
│               ├── analytics/      # Insights
│               └── settings/       # User settings
├── db/
│   ├── schema.surql               # Complete database schema reference
│   └── migrations/                # SurrealDB migrations
├── deploy/
│   └── docker-compose.yml         # SurrealDB container
├── Taskfile.yml                   # All development commands
└── docs/                          # Project documentation
```

---

## Backend Feature Pattern

Each feature in `internal/features/{name}/` follows this structure:

```
features/{name}/
├── handler.go    # HTTP handlers, routes, Swagger annotations
├── service.go    # Business logic layer
├── repository.go # SurrealDB operations
└── models.go     # Types, DTOs, request/response structs
```

### Creating a New Feature

```bash
# Generate scaffold
task new:go -- featurename

# Then register in cmd/api/main.go and regenerate docs
cd apps/go_backend && make docs
```

### Backend Error Handling

Use predefined errors from `internal/shared/errors`:

```go
import "github.com/lucid-logs/go-backend/internal/shared/errors"

// Return errors
return errors.ErrNotFound.WithMessage("Task not found")
return errors.ErrValidationFailed.WithDetails(validationErrors)
return errors.ErrConflict.WithMessage("Category already exists")
return errors.ErrDatabase.Wrap(err)

// Check errors
if errors.Is(err, errors.ErrNotFound) { ... }
```

### SurrealDB Queries (Go)

```go
import "github.com/lucid-logs/go-backend/internal/shared/database"

// Query multiple records
tasks, err := database.QueryAll[Task](db, 
    "SELECT * FROM tasks WHERE created_by = $user AND deleted_at = NONE", 
    map[string]any{"user": userID})

// Query single record
task, err := database.QueryFirst[Task](db, 
    "SELECT * FROM tasks WHERE id = $id LIMIT 1", 
    map[string]any{"id": taskID})

// Create with typed wrapper
task, err := database.Create[Task](db, "tasks", createData)

// Partial update (MERGE)
task, err := database.Merge[Task](db, "tasks:123", updateData)
```

---

## Frontend Patterns

### API Calls

```typescript
import { api, unwrap, type PaginatedResponse } from '$lib/api';

// GET with pagination
const result = await unwrap<PaginatedResponse<Task>>(
    api.get('tasks', { searchParams: { limit: 20, offset: 0 }})
);

// POST with body
const created = await unwrap<Task>(
    api.post('tasks', { json: { title: 'New Task', start_date: '2024-01-15' }})
);

// PUT update
const updated = await unwrap<Task>(
    api.put(`tasks/${id}`, { json: updateData })
);

// DELETE
await unwrap<void>(api.delete(`tasks/${id}`));
```

### Svelte 5 Runes

Always use Svelte 5 runes for state management:

```typescript
// Reactive state
let count = $state(0);
let items = $state<Task[]>([]);

// Derived values (auto-updates)
let filteredItems = $derived(items.filter(i => i.completed));
let itemCount = $derived(items.length);

// Side effects
$effect(() => {
    console.log('Count changed:', count);
});

// Props in components
interface Props {
    task: Task;
    onSave?: (task: Task) => void;
}
let { task, onSave }: Props = $props();
```

### TanStack Query Pattern

```typescript
import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';

// Query
const tasksQuery = createQuery({
    queryKey: ['tasks', { date, status }],
    queryFn: () => getTasks({ start_date: date, status })
});

// Mutation with cache invalidation
const queryClient = useQueryClient();
const updateMutation = createMutation({
    mutationFn: (data) => updateTask(id, data),
    onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['tasks'] });
    }
});
```

### UI Components

Import from `$lib/components/ui`:

```typescript
import { 
    Modal, ConfirmDialog, Card, IconBox, StatCard,
    PageHeader, SectionHeader, EmptyState, ErrorAlert, 
    LoadingCard, DataTable, SortableHeader, ColorPicker
} from '$lib/components/ui';
```

See `apps/frontend/src/lib/DESIGN_LANGUAGE.md` for complete component patterns.

---

## Database Schema

### Key Tables

| Table | Purpose |
|-------|---------|
| `users` | User accounts (schemafull) |
| `tasks` | Task/journal entries |
| `goals` | Goals, habits, projects |
| `categories` | Organization/tagging |
| `emotions` | Mood meter emotions (seeded) |
| `templates` | Reusable task templates |
| `retrospectives` | Daily/weekly reflections |
| `units` | Measurement units |

### Relation Tables (Graph Edges)

| Relation | Description |
|----------|-------------|
| `task_emotions` | tasks → emotions (primary, positive, negative) |
| `task_goals` | tasks → goals (with impact_type, quantity) |
| `in_category` | tasks/goals/templates → categories |
| `goal_children` | goals → goals (parent-child hierarchy) |
| `goal_logs` | Goal event history |
| `created_from` | tasks → templates (origin tracking) |

### Record ID Format

All IDs include table name: `tasks:abc123`, `categories:work`, `goals:fitness`

### Multi-tenancy

All queries filter by `created_by` and `deleted_at`:
```sql
SELECT * FROM tasks 
WHERE created_by = $user_id 
  AND deleted_at = NONE
```

---

## Common Commands

```bash
# Development
task dev              # Full stack (Go + SvelteKit + DB)
task go               # Backend only (with hot reload)
task fe               # Frontend only
task db:up            # Start database
task db:reset         # Reset database (DESTRUCTIVE!)

# Code Quality
task lint             # Lint all (Go + Frontend + Rust)
task lint:go          # Go: gofumpt + golangci-lint
task lint:fe          # Frontend: Biome
task check:fe         # Frontend: TypeScript check
task test:go          # Go tests with coverage

# Within apps/go_backend/
make docs             # Regenerate Swagger
make lint             # Lint Go code
make test             # Run tests
make new name=foo     # Generate new feature scaffold

# Within apps/frontend/
pnpm check            # TypeScript/Svelte check
pnpm build            # Production build
pnpm format           # Format with Biome
pnpm lint:fix         # Fix lint issues

# Database
task seed             # Populate test data
task seed:reset       # Reset and repopulate test data
```

---

## Code Style

### Go

- Use section separators: `// ===== SECTION =====`
- Add Swagger annotations to all handlers
- Always filter by `created_by` for multi-tenancy
- Soft delete with `deleted_at`, never hard delete
- Use `gofumpt` for formatting

### TypeScript/Svelte

- Use `$lib/api` for all API calls
- Type all API responses
- Use Svelte 5 runes: `$state()`, `$derived()`, `$effect()`, `$props()`
- Handle 401 with redirect to /login
- Follow patterns in `DESIGN_LANGUAGE.md`
- Use TailwindCSS 4 + DaisyUI 5 for styling

### Component Patterns & DaisyUI

- **DaisyUI First**: Use DaisyUI components (`btn`, `card`, `badge`, `toggle`, `file-input`) for consistent styling. Avoid rebuilding standard UI elements with raw Tailwind.
- **Tailwind Preference**: Use standard Tailwind utility classes for all custom styling. Only resort to custom CSS (e.g., in `<style>` blocks) if standard Tailwind utilities are not optimal or sufficient.
- **Color Backgrounds**: `bg-{color}/10` (10% opacity) for container backgrounds.
- **Icon Containers**: Use `<IconBox>` component.
- **Spacing**: Use the 4px grid system (`gap-1`, `p-4`, `m-2`).
- **States**: handle loading (`<LoadingCard>`), error (`<ErrorAlert>`), and empty (`<EmptyState>`) states.

## Coding Conventions & Best Practices

### Modularity & Reusability
- **Small Components**: Keep components focused and under ~200 lines. Extract sub-components for complex UIs (e.g., `TaskItem.svelte` inside `TaskList.svelte`).
- **Reusable Primitives**: Place generic UI components in `src/lib/components/ui`.
- **Composition**: Use slots/snippets for content injection instead of rigid prop drilling.

### Clean Code
- **Helper Functions**: Extract logic (formatting, calculations, transformations) into `src/lib/utils/`. Do not keep complex logic inside Svelte components.
- **Constants**: Move magic strings, configuration values, and repeated literals to `src/lib/constants.ts` or feature-specific constant files.
- **Type Safety**: Strictly type all props, events, and API responses. Avoid `any`.
- **DRY Principle**: If you copy-paste code twice, refactor it into a utility or component.

### File Organization
- **Feature grouping**: Group related files by feature (e.g., `src/lib/components/goals/`).
- **Barrelling**: Use `index.ts` files to export public members from a directory (cleaner imports).

---

## Files to Check First

| When You Need To... | Read This |
|---------------------|-----------|
| Add backend feature | `internal/features/tasks/` (complete example) |
| Understand errors | `internal/shared/errors/errors.go` |
| Add frontend API | `src/lib/api/tasks.ts` |
| Understand DB schema | `db/schema.surql` |
| See API endpoints | `apps/go_backend/docs/swagger.json` |
| UI component patterns | `src/lib/DESIGN_LANGUAGE.md` |
| Frontend API index | `src/lib/api/index.ts` |
| All task commands | `Taskfile.yml` |

---

## Goal Types (Inferred from Structure)

Goals don't have an explicit type enum - nature is inferred:

| Structure | Type | Example |
|-----------|------|---------|
| Has `recurrence` | Habit | "Exercise 3x/week" |
| Has `target` | Measurable | "Run 100km this month" |
| Has `target.operator = "lte"/"eq"` | Avoidance | "Max 2 coffees/day" |
| Has children via `goal_children` | Grouped | "Q1 Objectives" |
| None of above | Simple | "Learn Spanish" |

---

## Task-Goal Linking

Tasks can link to goals with impact metadata:

```typescript
// Link structure
interface TaskGoalLink {
    goal_id: string;           // "goals:hydration"
    impact_type: "positive" | "negative" | "neutral";
    impact_magnitude: 1-5;
    quantity_value?: number;   // For measurable goals
    unit_id?: string;          // "units:glasses"
    is_milestone?: boolean;
    notes?: string;
}
```

---

## Environment Variables

Copy `env.example` to `.env`:

```bash
# Database
DB_HOST=localhost
DB_PORT=8000
DB_USER=root
DB_PASS=root
DB_NAMESPACE=lucid
DB_DATABASE=logs

# JWT
JWT_SECRET=your-secret-here

# Admin (for seeding)
ADMIN_USERNAME=admin@example.com
ADMIN_PASSWORD=adminadmin

# Frontend
VITE_API_URL=http://localhost:8080
```

---

## Preferences

- Prefer TypeScript strict mode
- Use descriptive variable names
- Keep components small and focused (<200 lines ideal)
- Always handle loading and error states in UI
- Use SvelteKit's file-based routing
- Prefer composition over inheritance
- Use graph edges for relationships, not embedded arrays
- Log errors on server, show user-friendly messages on frontend
