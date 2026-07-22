import { type PaginatedResponse, api, unwrap } from './client';
import type { Category } from './tasks';
export type { Category };

// =============================================================================
// CATEGORY TYPES
// =============================================================================

export interface CreateCategoryRequest {
	name: string;
	color: string;
	icon?: string;
	parent_id?: string;
	description?: string;
}

export interface UpdateCategoryRequest {
	name?: string;
	color?: string;
	icon?: string;
	parent_id?: string;
	description?: string;
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
	search?: string;
}): Promise<CategoryPageResponse> {
	const searchParams = new URLSearchParams();
	if (params?.limit) searchParams.set('limit', String(params.limit));
	if (params?.offset) searchParams.set('offset', String(params.offset));
	if (params?.search) searchParams.set('search', params.search);

	return unwrap(api.get('categories', { searchParams }));
}

export async function getCategory(id: string): Promise<Category> {
	return unwrap(api.get(`categories/${encodeURIComponent(id)}`));
}

export async function createCategory(
	data: CreateCategoryRequest,
): Promise<Category> {
	return unwrap(api.post('categories', { json: data }));
}

export async function updateCategory(
	id: string,
	data: UpdateCategoryRequest,
): Promise<Category> {
	return unwrap(
		api.put(`categories/${encodeURIComponent(id)}`, { json: data }),
	);
}

export async function deleteCategory(id: string): Promise<void> {
	await api.delete(`categories/${encodeURIComponent(id)}`);
}
