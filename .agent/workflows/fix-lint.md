---
description: Fix linting and type errors in the codebase
---

# Fix Lint and Type Errors

Workflow to identify and fix code quality issues.

## Step 1: Identify Frontend Issues

// turbo
```bash
cd apps/frontend && pnpm check 2>&1 | head -100
```

## Step 2: Identify Backend Issues

// turbo
```bash
cd apps/go_backend && golangci-lint run ./... 2>&1 | head -100
```

## Step 3: Fix Issues

Based on the output from steps 1 and 2:
- For TypeScript errors: Check imports, types, and Svelte 5 rune usage
- For Go errors: Check error handling, unused variables, and type conversions
- For both: Run formatters first

### Auto-format Code

// turbo
```bash
cd apps/frontend && pnpm format
```

// turbo
```bash
cd apps/go_backend && go fmt ./...
```

## Step 4: Verify Fixes

// turbo
```bash
cd apps/frontend && pnpm check && pnpm build
```

// turbo
```bash
cd apps/go_backend && golangci-lint run ./...
```
