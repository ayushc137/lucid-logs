import { api, unwrap, type PaginatedResponse } from './client';

// =============================================================================
// TASK TYPES (matching Go backend)
// =============================================================================

export interface Task {
    id: string;
    title: string;
    journal: string;
    start_date: string;
    end_date: string;
    completed: boolean;
    priority: number;
    source: string;
    note: string;
    positives: TaskItem[];
    negatives: TaskItem[];
    category?: Category;
    emotion_id?: string;
    inferred_emotion?: InferredEmotion;
    activity_key?: string;
    template_id?: string;
    quantity?: Quantity;
    linked_goals?: TaskGoalLink[];
    created_at: string;
    updated_at: string;
    deleted_at?: string;
}

export interface TaskItem {
    text: string;
    emotion_id?: string;
}

export interface Quantity {
    value: number;
    unit: string;
}

export interface TaskGoalLink {
    goal_id: string;
    goal_title: string;
    impact_type: string;
    impact_magnitude: number;
    quantity_value?: number;
    quantity_unit?: string;
}

export interface Category {
    id: string;
    name: string;
    color: string;
    created_at: string;
    updated_at: string;
}

export interface InferredEmotion {
    valence: number;
    arousal: number;
    dominance: number;
    quadrant: string;
    closest_emotion_id: string;
    closest_emotion_name: string;
    positive_count: number;
    negative_count: number;
    dissonance: number;
}

export interface CreateTaskRequest {
    title: string;
    journal?: string;
    start_date: string;
    end_date: string;
    priority?: number;
    source?: string;
    note?: string;
    positives?: TaskItem[];
    negatives?: TaskItem[];
    category_id?: string;
    emotion_id?: string;
}

export interface UpdateTaskRequest {
    title?: string;
    journal?: string;
    start_date?: string;
    end_date?: string;
    completed?: boolean;
    priority?: number;
    note?: string;
    positives?: TaskItem[];
    negatives?: TaskItem[];
    category_id?: string;
    emotion_id?: string;
}

// =============================================================================
// TASK API FUNCTIONS
// =============================================================================

/**
 * Filter parameters for listing tasks.
 * All filters are optional and can be combined (AND logic).
 */
export interface TaskFilterParams {
    /** Items per page (default 20, max 100) */
    limit?: number;
    /** Items to skip (default 0) */
    offset?: number;
    /** Full-text search across title, journal, note (uses SurrealDB FTS) */
    search?: string;
    /** Filter by category ID */
    category_id?: string;
    /** Filter by status: "all", "completed", "pending" */
    status?: 'all' | 'completed' | 'pending';
    /** Filter by minimum priority (1-10) */
    priority_min?: number;
    /** Filter by maximum priority (1-10) */
    priority_max?: number;
    /** Filter tasks starting on or after this date (RFC3339) */
    start_date_from?: string;
    /** Filter tasks starting on or before this date (RFC3339) */
    start_date_to?: string;
    /** Sort by field: "start_date", "priority", "title", "created_at" */
    sort_field?: 'start_date' | 'priority' | 'title' | 'created_at';
    /** Sort direction: "asc" or "desc" */
    sort_order?: 'asc' | 'desc';
}

/**
 * Get paginated list of tasks with optional filters and full-text search.
 * 
 * @param params - Filter and pagination parameters
 * @returns Paginated response with tasks
 */
export async function getTasks(params?: TaskFilterParams): Promise<PaginatedResponse<Task>> {
    const searchParams = new URLSearchParams();

    // Pagination
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));

    // Full-text search
    if (params?.search) searchParams.set('search', params.search);

    // Filters
    if (params?.category_id) searchParams.set('category_id', params.category_id);
    if (params?.status && params.status !== 'all') searchParams.set('status', params.status);
    if (params?.priority_min !== undefined) searchParams.set('priority_min', String(params.priority_min));
    if (params?.priority_max !== undefined) searchParams.set('priority_max', String(params.priority_max));
    if (params?.start_date_from) searchParams.set('start_date_from', params.start_date_from);
    if (params?.start_date_to) searchParams.set('start_date_to', params.start_date_to);

    // Sorting
    if (params?.sort_field) searchParams.set('sort_field', params.sort_field);
    if (params?.sort_order) searchParams.set('sort_order', params.sort_order);

    return unwrap(api.get('tasks', { searchParams }));
}

export async function getTask(id: string): Promise<Task> {
    return unwrap(api.get(`tasks/${encodeURIComponent(id)}`));
}

export async function createTask(data: CreateTaskRequest): Promise<Task> {
    return unwrap(api.post('tasks', { json: data }));
}

export async function updateTask(id: string, data: UpdateTaskRequest): Promise<Task> {
    return unwrap(api.put(`tasks/${encodeURIComponent(id)}`, { json: data }));
}

export async function deleteTask(id: string): Promise<void> {
    await api.delete(`tasks/${encodeURIComponent(id)}`);
}
