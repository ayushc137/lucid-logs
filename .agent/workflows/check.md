---
description: Run linting and type checking for both frontend and backend
---

# Run Project Checks

Verify code quality across the entire stack:

// turbo-all

1. Run frontend TypeScript/Svelte checks:
```bash
cd apps/frontend && pnpm check
```

2. Run Go linting:
```bash
cd apps/go_backend && golangci-lint run ./...
```

3. Optionally build frontend to catch additional errors:
```bash
cd apps/frontend && pnpm build
```
