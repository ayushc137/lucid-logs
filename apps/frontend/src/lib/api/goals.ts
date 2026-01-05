import { api, unwrap, type PaginatedResponse } from './client';

// =============================================================================
// GOAL TYPES (matching Go backend)
// =============================================================================

export interface Goal {
    id: string;
    title: string;
    description?: string;
    why?: string;
    icon?: string;
    color?: string;
    goal_type: 'discrete' | 'measurable' | 'epic' | 'avoidance';
    recurrence?: Recurrence;
    target?: Target;
    start_date?: string;
    deadline?: string;
    status: 'active' | 'completed' | 'paused' | 'abandoned';
    completion_date?: string;
    current_streak: number;
    longest_streak: number;
    last_completed_date?: string;
    grace_days_used: number;
    priority: 1 | 2 | 3;
    value_score: 1 | 2 | 3 | 4 | 5;
    category_id?: string;
    parent_goal_id?: string;
    life_domain?: string;
    success_signal?: string;
    linked_template_id?: string;
    activity_key?: string;
    is_private: boolean;
    linked_tasks?: GoalTaskLink[];
    child_goals?: Goal[];
    created_at: string;
    updated_at: string;
    deleted_at?: string;
}

export interface Recurrence {
    frequency: number;
    period: 'day' | 'week' | 'month';
    active_days?: string[];
    before_time?: string;
    after_time?: string;
    grace_days?: number;
}

export interface Target {
    value: number;
    unit: string;
    current_value: number;
    per_period?: boolean;
    track_completed_only?: boolean;
}

export interface GoalTaskLink {
    task_id: string;
    task_title: string;
    impact_type: string;
    impact_magnitude: number;
    quantity_value?: number;
    quantity_unit?: string;
}

export interface CreateGoalRequest {
    title: string;
    description?: string;
    why?: string;
    icon?: string;
    color?: string;
    goal_type: 'discrete' | 'measurable' | 'epic' | 'avoidance';
    recurrence?: Recurrence;
    target?: { value: number; unit: string; per_period?: boolean; track_completed_only?: boolean };
    start_date?: string;
    deadline?: string;
    priority?: 1 | 2 | 3;
    value_score?: 1 | 2 | 3 | 4 | 5;
    category_id?: string;
    parent_goal_id?: string;
    life_domain?: string;
    activity_key?: string;
    is_private?: boolean;
}

// =============================================================================
// GOAL API FUNCTIONS
// =============================================================================

export async function getGoals(params?: {
    limit?: number;
    offset?: number;
    status?: string;
    goal_type?: string;
    recurring?: boolean;
    search?: string;
    sort_by?: string;
}): Promise<PaginatedResponse<Goal>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));
    if (params?.status) searchParams.set('status', params.status);
    if (params?.goal_type) searchParams.set('goal_type', params.goal_type);
    if (params?.recurring !== undefined) searchParams.set('recurring', String(params.recurring));
    if (params?.search) searchParams.set('search', params.search);
    if (params?.sort_by) searchParams.set('sort_by', params.sort_by);

    return unwrap(api.get('goals', { searchParams }));
}

export async function getGoal(id: string): Promise<Goal> {
    return unwrap(api.get(`goals/${id}`));
}

export async function createGoal(data: CreateGoalRequest): Promise<Goal> {
    return unwrap(api.post('goals', { json: data }));
}

export async function updateGoal(id: string, data: Partial<CreateGoalRequest>): Promise<Goal> {
    return unwrap(api.put(`goals/${id}`, { json: data }));
}

export async function deleteGoal(id: string): Promise<void> {
    await api.delete(`goals/${id}`);
}

export async function getTodayGoals(): Promise<Goal[]> {
    return unwrap(api.get('goals/today'));
}
