---
description: Start the development environment
---

# Start Development Environment

The recommended way to start development is using the `task dev` command which starts everything together.

## Option 1: Full Stack (Recommended)

// turbo
```bash
task dev
```

This single command:
- Starts SurrealDB via Docker
- Starts Go backend with Air hot reload on `:8080`
- Waits for backend health check
- Starts SvelteKit frontend on `:5173`

**Application URLs:**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080/api/v1
- Swagger Docs: http://localhost:8080/swagger/index.html

## Option 2: Individual Services

If you need to run services separately (e.g., for debugging):

1. Start database:
// turbo
```bash
task db:up
```

2. Start Go backend (in a terminal):
```bash
task go
```

3. Start frontend (in another terminal):
```bash
task fe
```

## Seeding Test Data

To populate the database with test data:

```bash
task seed           # Add test data
task seed:reset     # Reset and repopulate
```

## Stopping Services

```bash
# Stop database
task db:down

# Kill processes on ports (if needed)
lsof -ti :8080 | xargs kill -9  # Backend
lsof -ti :5173 | xargs kill -9  # Frontend
```
