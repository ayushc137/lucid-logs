import { api, unwrap } from './client';
import type { Goal } from './goals';
import type { Task } from './tasks';

// =============================================================================
// TASK-GOAL LINK TYPES (matching Go backend)
// =============================================================================

export interface TaskGoalLink {
	id: string;
	task_id: string;
	goal_id: string;
	impact_type: 'positive' | 'negative' | 'neutral';
	impact_magnitude: number; // 1-5
	quantity_value?: number;
	quantity_unit?: string;
	notes?: string;
	source: 'manual' | 'auto';
	created_at: string;
}

export interface TaskGoalWithGoal extends TaskGoalLink {
	goal?: Goal;
}

export interface TaskGoalWithTask extends TaskGoalLink {
	task?: Task;
}

export interface LinkTaskToGoalRequest {
	goal_id: string;
	impact_type: 'positive' | 'negative' | 'neutral';
	impact_magnitude?: number;
	quantity_value?: number;
	quantity_unit?: string;
	notes?: string;
}

export interface BatchLinkRequest {
	links: LinkTaskToGoalRequest[];
}

export interface UpdateLinkRequest {
	impact_type?: 'positive' | 'negative' | 'neutral';
	impact_magnitude?: number;
	quantity_value?: number;
	quantity_unit?: string;
	notes?: string;
}

export interface GoalsForTaskResponse {
	task_id: string;
	goals: TaskGoalWithGoal[];
}

export interface TasksForGoalResponse {
	goal_id: string;
	tasks: TaskGoalWithTask[];
}

// =============================================================================
// TASK-GOAL LINK API FUNCTIONS
// =============================================================================

/**
 * Get all goals linked to a task
 */
export async function getGoalsForTask(
	taskId: string,
): Promise<GoalsForTaskResponse> {
	return unwrap(api.get(`tasks/${encodeURIComponent(taskId)}/goals`));
}

/**
 * Link a task to a goal
 */
export async function linkTaskToGoal(
	taskId: string,
	data: LinkTaskToGoalRequest,
): Promise<TaskGoalLink> {
	return unwrap(
		api.post(`tasks/${encodeURIComponent(taskId)}/goals`, { json: data }),
	);
}

/**
 * Link a task to multiple goals at once
 */
export async function batchLinkTaskToGoals(
	taskId: string,
	data: BatchLinkRequest,
): Promise<TaskGoalLink[]> {
	return unwrap(
		api.post(`tasks/${encodeURIComponent(taskId)}/goals/batch`, { json: data }),
	);
}

/**
 * Update a task-goal link
 */
export async function updateTaskGoalLink(
	taskId: string,
	linkId: string,
	data: UpdateLinkRequest,
): Promise<TaskGoalLink> {
	return unwrap(
		api.put(
			`tasks/${encodeURIComponent(taskId)}/goals/${encodeURIComponent(linkId)}`,
			{
				json: data,
			},
		),
	);
}

/**
 * Remove a task-goal link
 */
export async function unlinkTaskFromGoal(
	taskId: string,
	linkId: string,
): Promise<void> {
	await api.delete(
		`tasks/${encodeURIComponent(taskId)}/goals/${encodeURIComponent(linkId)}`,
	);
}

/**
 * Get all tasks linked to a goal
 */
export async function getTasksForGoal(
	goalId: string,
): Promise<TasksForGoalResponse> {
	return unwrap(api.get(`goals/${encodeURIComponent(goalId)}/tasks`));
}
