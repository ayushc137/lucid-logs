import { api, unwrap, type PaginatedResponse } from './client';
import type { Category } from './tasks';
export type { Category };

// =============================================================================
// CATEGORY TYPES
// =============================================================================

export interface CreateCategoryRequest {
    name: string;
    color: string;
}

export interface UpdateCategoryRequest {
    name?: string;
    color?: string;
}

export interface CategoryPageResponse {
    items: Category[];
    total: number;
    limit: number;
    offset: number;
}

// =============================================================================
// CATEGORY API FUNCTIONS
// =============================================================================

export async function getCategories(params?: {
    limit?: number;
    offset?: number;
}): Promise<CategoryPageResponse> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));

    return unwrap(api.get('categories', { searchParams }));
}

export async function getCategory(id: string): Promise<Category> {
    return unwrap(api.get(`categories/${encodeURIComponent(id)}`));
}

export async function createCategory(data: CreateCategoryRequest): Promise<Category> {
    return unwrap(api.post('categories', { json: data }));
}

export async function updateCategory(id: string, data: UpdateCategoryRequest): Promise<Category> {
    return unwrap(api.put(`categories/${encodeURIComponent(id)}`, { json: data }));
}

export async function deleteCategory(id: string): Promise<void> {
    await api.delete(`categories/${encodeURIComponent(id)}`);
}
