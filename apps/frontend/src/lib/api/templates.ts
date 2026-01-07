import { api, unwrap, type PaginatedResponse } from './client';
import type { Category } from './categories';

// =============================================================================
// TEMPLATE TYPES (matching Go backend)
// =============================================================================

import type { Goal } from './goals';
import type { Category } from './categories';

export interface TaskTemplate {
    id: string;
    title: string;
    description?: string;
    icon?: string;

    // Defaults for tasks
    default_duration?: number; // seconds
    default_emotion_id?: string;
    expected_quadrant?: 'green' | 'yellow' | 'red' | 'blue';

    // Quick log settings
    is_quick_log: boolean;
    quick_log_order?: number;

    // Quantity settings
    quantity_enabled: boolean;
    quantity_default?: number;
    quantity_step?: number;
    // Note: quantity unit is inherited from linked goal's target

    // Usage stats
    use_count: number;
    last_used_at?: string;

    // Metadata
    created_at: string;
    updated_at: string;
    deleted_at?: string;

    // Populated via graph
    goals?: Goal[];
    category?: Category;
}

export interface GoalLinkInput {
    goal_id: string;
    auto_link_tasks?: boolean;
    quantity_multiplier?: number;
}

export interface CreateTemplateRequest {
    title: string;
    description?: string;
    icon?: string;

    default_duration?: number;

    is_quick_log?: boolean;
    quick_log_order?: number;

    quantity_enabled?: boolean;
    quantity_default?: number;
    quantity_step?: number;

    expected_quadrant?: 'green' | 'yellow' | 'red' | 'blue';
    default_emotion_id?: string;

    category_id?: string;
    goal_links?: GoalLinkInput[];
}

export interface UpdateTemplateRequest {
    title?: string;
    description?: string;
    icon?: string;

    default_duration?: number;

    is_quick_log?: boolean;
    quick_log_order?: number;

    quantity_enabled?: boolean;
    quantity_default?: number;
    quantity_step?: number;

    expected_quadrant?: 'green' | 'yellow' | 'red' | 'blue';
    default_emotion_id?: string;
}

export interface InstantiateTemplateRequest {
    start_date: string;
    end_date?: string;
    quantity?: number;
    notes?: string;
}

// =============================================================================
// TEMPLATE API FUNCTIONS
// =============================================================================

export async function getTemplates(params?: {
    limit?: number;
    offset?: number;
}): Promise<PaginatedResponse<TaskTemplate>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));

    return unwrap(api.get('templates', { searchParams }));
}

export async function getTemplate(id: string): Promise<TaskTemplate> {
    return unwrap(api.get(`templates/${encodeURIComponent(id)}`));
}

export async function createTemplate(data: CreateTemplateRequest): Promise<TaskTemplate> {
    return unwrap(api.post('templates', { json: data }));
}

export async function updateTemplate(id: string, data: UpdateTemplateRequest): Promise<TaskTemplate> {
    return unwrap(api.put(`templates/${encodeURIComponent(id)}`, { json: data }));
}

export async function deleteTemplate(id: string): Promise<void> {
    await api.delete(`templates/${encodeURIComponent(id)}`);
}

export async function getQuickLogTemplates(): Promise<TaskTemplate[]> {
    return unwrap(api.get('templates/quick-log'));
}
