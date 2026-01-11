import { api, unwrap } from './client';

// =============================================================================
// GOAL ACTION TYPES (matching Go backend)
// =============================================================================

export interface GoalAction {
	id: string;
	goal_id: string;
	title: string;
	description?: string;
	order: number;
	quantity_value?: number;
	quantity_unit?: string;
	completed: boolean;
	completed_at?: string;
	created_at: string;
	updated_at: string;
	deleted_at?: string;
}

export interface CreateGoalActionRequest {
	title: string;
	description?: string;
	order?: number;
	quantity_value?: number;
	quantity_unit?: string;
}

export interface UpdateGoalActionRequest {
	title?: string;
	description?: string;
	order?: number;
	quantity_value?: number;
	quantity_unit?: string;
	completed?: boolean;
}

export interface ReorderActionsRequest {
	action_ids: string[];
}

export interface ActionListResponse {
	goal_id: string;
	actions: GoalAction[];
	count: number;
}

// =============================================================================
// GOAL ACTION API FUNCTIONS
// =============================================================================

/**
 * Get all actions for a goal
 */
export async function getGoalActions(
	goalId: string,
): Promise<ActionListResponse> {
	return unwrap(api.get(`goals/${encodeURIComponent(goalId)}/actions`));
}

/**
 * Create a new action for a goal
 */
export async function createGoalAction(
	goalId: string,
	data: CreateGoalActionRequest,
): Promise<GoalAction> {
	return unwrap(
		api.post(`goals/${encodeURIComponent(goalId)}/actions`, { json: data }),
	);
}

/**
 * Update a goal action
 */
export async function updateGoalAction(
	goalId: string,
	actionId: string,
	data: UpdateGoalActionRequest,
): Promise<GoalAction> {
	return unwrap(
		api.put(
			`goals/${encodeURIComponent(goalId)}/actions/${encodeURIComponent(actionId)}`,
			{
				json: data,
			},
		),
	);
}

/**
 * Delete a goal action
 */
export async function deleteGoalAction(
	goalId: string,
	actionId: string,
): Promise<void> {
	await api.delete(
		`goals/${encodeURIComponent(goalId)}/actions/${encodeURIComponent(actionId)}`,
	);
}

/**
 * Reorder actions within a goal
 */
export async function reorderGoalActions(
	goalId: string,
	data: ReorderActionsRequest,
): Promise<void> {
	await api.put(`goals/${encodeURIComponent(goalId)}/actions/reorder`, {
		json: data,
	});
}

/**
 * Mark an action as complete/incomplete
 */
export async function toggleActionComplete(
	goalId: string,
	actionId: string,
	completed: boolean,
): Promise<GoalAction> {
	return updateGoalAction(goalId, actionId, { completed });
}
