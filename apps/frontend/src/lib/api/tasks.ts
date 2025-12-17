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

export async function getTasks(params?: {
    limit?: number;
    offset?: number;
    start_date?: string;
    end_date?: string;
}): Promise<PaginatedResponse<Task>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));
    if (params?.start_date) searchParams.set('start_date', params.start_date);
    if (params?.end_date) searchParams.set('end_date', params.end_date);

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
