import type { Category } from './categories';
import { type PaginatedResponse, api, unwrap } from './client';
import type { Goal } from './goals';

// =============================================================================
// TYPES
// =============================================================================

export interface Activity {
	id: string;
	title: string;
	description?: string;
	icon?: string;

	// Task Defaults
	default_duration?: number; // seconds
	default_emotion_id?: string;
	default_priority: number;
	default_completed: boolean;

	// Quantity Settings
	quantity_enabled: boolean;
	quantity_default?: number;
	quantity_step?: number;
	quantity_unit_id?: string;

	// Goal Link Defaults
	default_impact: 'positive' | 'negative' | 'neutral';

	// Display
	pinned: boolean;
	sort_order: number;

	// Usage Stats
	use_count: number;
	last_used_at?: string;

	// Active Timer Session
	active_session?: TimerSession;

	// Metadata
	created_at: string;
	updated_at: string;
	deleted_at?: string;

	// Populated via graph
	goals?: ActivityGoalLink[];
	category?: Category;
}

export interface ActivityGoalLink {
	goal_id: string;
	auto_link_tasks: boolean;
	quantity_multiplier: number;
	default_impact: string;
	goal?: Goal;
}

export interface TimerSession {
	task_id: string;
	started_at: string;
	breaks?: TimerBreak[];
}

export interface TimerBreak {
	started_at: string;
	ended_at?: string;
	duration?: number; // seconds
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

export interface CreateActivityRequest {
	title: string;
	icon?: string;
	description?: string;

	default_duration?: number;
	default_emotion_id?: string;
	default_priority?: number;
	default_completed?: boolean;

	quantity_enabled?: boolean;
	quantity_default?: number;
	quantity_step?: number;
	quantity_unit_id?: string;

	default_impact?: 'positive' | 'negative' | 'neutral';

	pinned?: boolean;
	sort_order?: number;

	category_id?: string;
	goal_links?: GoalLinkInput[];
}

export interface UpdateActivityRequest {
	title?: string;
	icon?: string;
	description?: string;

	default_duration?: number;
	default_emotion_id?: string;
	default_priority?: number;
	default_completed?: boolean;

	quantity_enabled?: boolean;
	quantity_default?: number;
	quantity_step?: number;
	quantity_unit_id?: string;

	default_impact?: 'positive' | 'negative' | 'neutral';

	pinned?: boolean;
	sort_order?: number;
}

export interface GoalLinkInput {
	goal_id: string;
	auto_link_tasks?: boolean;
	quantity_multiplier?: number;
	default_impact?: 'positive' | 'negative' | 'neutral';
}

export interface InstantLogRequest {
	quantity?: number;
	notes?: string;
	timestamp?: string;
}

export interface ScheduleRequest {
	start_date?: string;
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

export interface InstantLogResponse {
	task_id: string;
	task_title: string;
	goals_updated?: GoalUpdateSummary[];
}

export interface GoalUpdateSummary {
	goal_id: string;
	goal_title: string;
	goal_icon?: string;
	value_added: number;
	new_total: number;
	target_value?: number;
	is_completed: boolean;
}

export interface ScheduleResponse {
	activity: Activity;
	task_defaults: TaskDefaults;
	goal_links: GoalLinkDefault[];
}

export interface TaskDefaults {
	title: string;
	journal?: string;
	duration?: number;
	priority: number;
	category_id?: string;
	emotion_id?: string;
	quantity?: number;
	quantity_unit?: string;
}

export interface GoalLinkDefault {
	goal_id: string;
	goal_title: string;
	goal_icon?: string;
	impact_type: string;
	quantity?: number;
	quantity_unit?: string;
}

export interface ActivityGoalLinkDetail {
	goal_id: string;
	goal_title: string;
	goal_icon?: string;
	goal_color?: string;
	auto_link_tasks: boolean;
	quantity_multiplier: number;
	default_impact: string;
	target_unit_id?: string;
	target_unit_symbol?: string;
}

// =============================================================================
// API FUNCTIONS
// =============================================================================

/**
 * Get paginated list of activities
 */
export async function getActivities(params?: {
	limit?: number;
	offset?: number;
}): Promise<PaginatedResponse<Activity>> {
	const searchParams = new URLSearchParams();
	if (params?.limit) searchParams.set('limit', String(params.limit));
	if (params?.offset) searchParams.set('offset', String(params.offset));

	return unwrap(api.get('activities', { searchParams }));
}

/**
 * Get pinned activities for quick bar
 */
export async function getPinnedActivities(): Promise<Activity[]> {
	return unwrap(api.get('activities/pinned'));
}

/**
 * Get a single activity by ID
 */
export async function getActivity(id: string): Promise<Activity> {
	return unwrap(api.get(`activities/${encodeURIComponent(id)}`));
}

/**
 * Create a new activity
 */
export async function createActivity(
	data: CreateActivityRequest
): Promise<Activity> {
	return unwrap(api.post('activities', { json: data }));
}

/**
 * Update an existing activity
 */
export async function updateActivity(
	id: string,
	data: UpdateActivityRequest
): Promise<Activity> {
	return unwrap(api.put(`activities/${encodeURIComponent(id)}`, { json: data }));
}

/**
 * Delete an activity (soft delete)
 */
export async function deleteActivity(id: string): Promise<void> {
	await api.delete(`activities/${encodeURIComponent(id)}`);
}

/**
 * Instant log - Create a completed task immediately
 */
export async function instantLog(
	id: string,
	data?: InstantLogRequest
): Promise<InstantLogResponse> {
	return unwrap(
		api.post(`activities/${encodeURIComponent(id)}/instant`, {
			json: data || {}
		})
	);
}

/**
 * Schedule - Get pre-filled task data for the task form
 */
export async function scheduleActivity(
	id: string,
	data?: ScheduleRequest
): Promise<ScheduleResponse> {
	return unwrap(
		api.post(`activities/${encodeURIComponent(id)}/schedule`, {
			json: data || {}
		})
	);
}

/**
 * Get goals linked to an activity
 */
export async function getActivityLinkedGoals(
	id: string
): Promise<ActivityGoalLinkDetail[]> {
	return unwrap(api.get(`activities/${encodeURIComponent(id)}/goals`));
}

/**
 * Link a goal to an activity
 */
export async function linkGoalToActivity(
	activityId: string,
	data: GoalLinkInput
): Promise<void> {
	await api.post(`activities/${encodeURIComponent(activityId)}/goals`, {
		json: data
	});
}

/**
 * Unlink a goal from an activity
 */
export async function unlinkGoalFromActivity(
	activityId: string,
	goalId: string
): Promise<void> {
	await api.delete(
		`activities/${encodeURIComponent(activityId)}/goals/${encodeURIComponent(goalId)}`
	);
}
