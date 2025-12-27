---
description: Template for creating a new frontend API module following the established pattern
---

# Frontend API Module Template

Use this template when creating new API client modules in `src/lib/api/`.

## {name}.ts

```typescript
/**
 * API module for {Name} operations.
 * @module api/{name}
 */
import { api, unwrap, type PaginatedResponse } from './client';

// =============================================================================
// TYPES
// =============================================================================

/**
 * {Singular} entity as returned from the API.
 */
export interface {Singular} {
    /** Unique ID in format "{name}:xyz" */
    id: string;
    /** Human-readable title */
    title: string;
    /** User who created this record */
    created_by: string;
    /** ISO 8601 timestamp of creation */
    created_at: string;
    /** ISO 8601 timestamp of last update */
    updated_at: string;
}

/**
 * Request payload for creating a new {singular}.
 */
export interface Create{Singular}Request {
    /** Required. Title for the {singular} */
    title: string;
    // Add other required/optional fields
}

/**
 * Request payload for updating an existing {singular}.
 * All fields are optional - only provided fields will be updated.
 */
export interface Update{Singular}Request {
    /** Optional. New title */
    title?: string;
    // Add other optional fields
}

/**
 * Query parameters for filtering {name} list.
 */
export interface {Singular}FilterParams {
    /** Full-text search query */
    search?: string;
    /** Sort field: 'created_at' | 'title' | 'updated_at' */
    sort_field?: string;
    /** Sort direction: 'asc' | 'desc' */
    sort_order?: 'asc' | 'desc';
}

// =============================================================================
// API FUNCTIONS
// =============================================================================

/**
 * Fetches a paginated list of {name}.
 *
 * @param params - Pagination and filter parameters
 * @returns Paginated response containing {name}
 *
 * @example
 * const response = await get{Name}({ limit: 20, offset: 0 });
 * console.log(response.items); // Array of {Singular}
 */
export async function get{Name}(
    params: {Singular}FilterParams & { limit?: number; offset?: number } = {}
): Promise<PaginatedResponse<{Singular}>> {
    const searchParams = new URLSearchParams();

    if (params.limit) searchParams.set('limit', String(params.limit));
    if (params.offset) searchParams.set('offset', String(params.offset));
    if (params.search) searchParams.set('search', params.search);
    if (params.sort_field) searchParams.set('sort_field', params.sort_field);
    if (params.sort_order) searchParams.set('sort_order', params.sort_order);

    const query = searchParams.toString();
    const url = query ? `{name}?${query}` : '{name}';

    return unwrap(api.get(url));
}

/**
 * Fetches a single {singular} by ID.
 *
 * @param id - The {singular} ID (e.g., "{name}:abc123")
 * @returns The {singular} entity
 * @throws {Error} If {singular} not found (404)
 *
 * @example
 * const item = await get{Singular}('{name}:abc123');
 */
export async function get{Singular}(id: string): Promise<{Singular}> {
    return unwrap(api.get(`{name}/${id}`));
}

/**
 * Creates a new {singular}.
 *
 * @param data - The creation payload
 * @returns The created {singular} with generated ID
 *
 * @example
 * const created = await create{Singular}({ title: 'New Item' });
 * console.log(created.id); // "{name}:xyz123"
 */
export async function create{Singular}(data: Create{Singular}Request): Promise<{Singular}> {
    return unwrap(api.post('{name}', { json: data }));
}

/**
 * Updates an existing {singular}.
 *
 * @param id - The {singular} ID to update
 * @param data - Fields to update (partial update supported)
 * @returns The updated {singular}
 * @throws {Error} If {singular} not found (404)
 *
 * @example
 * const updated = await update{Singular}('{name}:abc123', { title: 'New Title' });
 */
export async function update{Singular}(
    id: string,
    data: Update{Singular}Request
): Promise<{Singular}> {
    return unwrap(api.put(`{name}/${id}`, { json: data }));
}

/**
 * Deletes a {singular} (soft delete).
 *
 * @param id - The {singular} ID to delete
 * @throws {Error} If {singular} not found (404)
 *
 * @example
 * await delete{Singular}('{name}:abc123');
 */
export async function delete{Singular}(id: string): Promise<void> {
    await unwrap(api.delete(`{name}/${id}`));
}
```

## Usage

Replace placeholders:
- `{name}` → feature name in lowercase plural (e.g., `widgets`)
- `{singular}` → singular lowercase (e.g., `widget`)
- `{Singular}` → singular PascalCase (e.g., `Widget`)
- `{Name}` → plural PascalCase (e.g., `Widgets`)

Then export from `src/lib/api/index.ts`:

```typescript
// {Name}
export {
    get{Name},
    get{Singular},
    create{Singular},
    update{Singular},
    delete{Singular},
    type {Singular},
    type Create{Singular}Request,
    type Update{Singular}Request
} from './{name}';
```
