import { api, unwrap, type PaginatedResponse } from './client';
import type { Category } from './categories';

// =============================================================================
// TEMPLATE TYPES (matching Go backend)
// =============================================================================

export interface ShowFields {
    journal?: boolean;
    duration?: boolean;
    quantity?: boolean;
    emotion?: boolean;
    positives_negatives?: boolean;
    notes?: boolean;
}

export interface TaskTemplate {
    id: string;
    title: string;
    description?: string;
    icon?: string;
    color?: string;

    // Defaults for tasks
    default_duration?: number; // seconds
    default_priority?: number;
    default_category?: Category;

    // Quick log settings
    is_quick_log: boolean;
    quick_log_order?: number;

    // Quantity settings
    quantity_enabled: boolean;
    quantity_default?: number;
    quantity_unit?: string;
    quantity_step?: number;

    // Emotion defaults
    expected_quadrant?: 'green' | 'yellow' | 'red' | 'blue';
    default_emotion_id?: string;

    // Goal/Activity linking
    activity_key?: string;
    goal_id?: string;

    // Fields to show
    show_fields?: ShowFields;

    // Source
    is_default: boolean;
    source_task_id?: string;

    // Usage stats
    use_count: number;
    last_used_at?: string;

    // Metadata
    created_at: string;
    updated_at: string;
    deleted_at?: string;
}

export interface CreateTemplateRequest {
    title: string;
    description?: string;
    icon?: string;
    color?: string;

    default_duration?: number;
    default_priority?: number;
    default_category_id?: string;

    is_quick_log?: boolean;
    quick_log_order?: number;

    quantity_enabled?: boolean;
    quantity_default?: number;
    quantity_unit?: string;
    quantity_step?: number;

    expected_quadrant?: 'green' | 'yellow' | 'red' | 'blue';
    default_emotion_id?: string;

    activity_key?: string;
    goal_id?: string;

    show_fields?: ShowFields;
}

export interface UpdateTemplateRequest {
    title?: string;
    description?: string;
    icon?: string;
    color?: string;

    default_duration?: number;
    default_priority?: number;
    default_category_id?: string;

    is_quick_log?: boolean;
    quick_log_order?: number;

    quantity_enabled?: boolean;
    quantity_default?: number;
    quantity_unit?: string;
    quantity_step?: number;

    expected_quadrant?: 'green' | 'yellow' | 'red' | 'blue';
    default_emotion_id?: string;

    show_fields?: ShowFields;
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
