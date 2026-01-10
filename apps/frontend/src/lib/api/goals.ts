import { api, unwrap, type PaginatedResponse } from './client';

// =============================================================================
// GOAL TYPES (matching Go backend)
// =============================================================================

export interface Goal {
    id: string;
    title: string;
    description?: string;
    icon?: string;

    // Target (optional - if absent, implies simple completion goal)
    target?: Target;

    // Recurrence (optional - if present, this is a habit)
    recurrence?: Recurrence;

    // Status: only 3 states
    status: 'active' | 'completed' | 'archived';

    // Computed statistics (populated on read)
    stats?: GoalStats;

    // Organization
    priority: number; // 1-3

    // Timeline
    start_date?: string;
    deadline?: string;
    completed_at?: string;

    // Metadata
    created_at: string;
    updated_at: string;
    deleted_at?: string;

    // Populated via graph queries (not stored on goal record)
    category?: { id: string; name: string; color: string; icon: string }; // Simplified from Category type for now
    linked_tasks?: GoalTaskLink[];
    children?: Goal[];
    parent?: Goal;
}

export interface Target {
    value: number;
    operator: 'gte' | 'lte' | 'eq';
    unit_id: string;
    per_period: boolean;
    track_completed_only?: boolean;
}

export interface Recurrence {
    frequency: number;
    period: 'day' | 'week' | 'month';
    active_days?: string[];
    before_time?: string;
    after_time?: string;
    grace_days?: number;
}

export interface GoalStats {
    current_value: number;
    progress_percent: number;
    current_streak: number;
    longest_streak: number;
    last_completed_date?: string;
    today_status?: 'pending' | 'met' | 'exceeded';
    children_total?: number;
    children_completed?: number;
    total_contributions: number;
}

export interface GoalTaskLink {
    task_id: string;
    task_title: string;
    impact_type: 'positive' | 'negative' | 'neutral';
    impact_magnitude: number;
    quantity_value?: number;
    unit_id?: string;
}

export interface CreateGoalRequest {
    title: string;
    description?: string;
    icon?: string;

    target?: {
        value: number;
        operator?: 'gte' | 'lte' | 'eq';
        unit_id: string;
        per_period?: boolean;
        track_completed_only?: boolean;
    };

    recurrence?: Recurrence;

    start_date?: string;
    deadline?: string;

    priority?: number;
    category_id?: string;

    parent_goal_id?: string;
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

// =============================================================================
// GOAL LOGS TYPES
// =============================================================================

export interface GoalLog {
    id: string;
    goal_id: string;
    event: GoalLogEvent;
    changes?: Record<string, any>;
    triggered_by_task_id?: string;
    snapshot_id?: string;
    created_at: string;
}

export type GoalLogEvent =
    | 'created'
    | 'updated'
    | 'completed'
    | 'archived'
    | 'reactivated'
    | 'deleted'
    | 'streak_updated'
    | 'streak_broken'
    | 'target_met'
    | 'target_exceeded'
    | 'period_end'
    | 'task_linked'
    | 'task_unlinked'
    | 'child_added'
    | 'child_removed';

export interface GoalLogsResponse {
    goal_id: string;
    logs: GoalLog[];
    total: number;
}

export interface GoalLogsSummary {
    days_met: number;
    days_missed: number;
    avg_daily?: number;
    best_day?: {
        date: string;
        value: number;
    };
}

// =============================================================================
// GOAL LOGS API FUNCTIONS
// =============================================================================

export async function getGoalLogs(
    goalId: string,
    params?: { limit?: number; offset?: number }
): Promise<GoalLogsResponse> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));

    const query = searchParams.toString() ? `?${searchParams}` : '';
    return unwrap(api.get(`goals/${goalId}/logs${query}`));
}

export async function getGoalLogsSummary(
    goalId: string,
    days?: number
): Promise<GoalLogsSummary> {
    const query = days ? `?days=${days}` : '';
    return unwrap(api.get(`goals/${goalId}/logs/summary${query}`));
}

