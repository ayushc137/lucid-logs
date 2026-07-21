import { api, unwrap } from './client';

// =============================================================================
// RETROSPECTIVE TYPES (matching Go backend)
// =============================================================================

export interface Retrospective {
	id: string;
	retro_type: 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'yearly' | 'custom';
	start_date: string;
	end_date: string;
	auto_summary: RetroAutoSummary;
	user_content: UserReflection;
	status: 'draft' | 'completed';
	generated_at: string;
	created_at: string;
	updated_at: string;
	deleted_at?: string;
}

export interface RetroAutoSummary {
	mood?: MoodSummary;
	habits?: HabitsSummary;
	tasks?: TasksSummary;
	goals?: GoalsSummary;
	categories?: CategoriesSummary;
	insights?: string[];
	ai_narrative?: string;
}

export interface MoodSummary {
	average_valence: number;
	average_arousal: number;
	dominant_quadrant: string;
	quadrant_distribution: QuadrantDist;
	notable_spikes?: MoodEvent[];
	notable_dips?: MoodEvent[];
}

export interface QuadrantDist {
	yellow: number;
	green: number;
	red: number;
	blue: number;
}

export interface MoodEvent {
	date: string;
	emotion: string;
	context?: string;
}

export interface HabitsSummary {
	met?: HabitStatus[];
	partially_met?: HabitStatus[];
	missed?: HabitStatus[];
	streaks?: StreaksSummary;
}

export interface HabitStatus {
	habit_id: string;
	name: string;
	success_rate?: number;
}

export interface StreaksSummary {
	continued?: StreakUpdate[];
	broken?: StreakUpdate[];
	started?: StreakUpdate[];
}

export interface StreakUpdate {
	habit_id: string;
	name: string;
	streak?: number;
	was?: number;
	now?: number;
}

export interface TasksSummary {
	completed: number;
	postponed: number;
	canceled: number;
	not_started?: number;
	total_duration_hours: number;
	by_category?: CategoryCount[];
}

export interface CategoryCount {
	category: string;
	count: number;
	duration_hours: number;
}

export interface GoalsSummary {
	net_impact?: GoalImpact[];
	significantly_advanced?: GoalHighlight[];
	negatively_impacted?: GoalHighlight[];
}

export interface GoalImpact {
	goal_id: string;
	name: string;
	positive: number;
	negative: number;
}

export interface GoalHighlight {
	goal_id: string;
	name: string;
	progress_delta?: number;
	reason?: string;
}

export interface CategoriesSummary {
	time_distribution?: CategoryTime[];
	neglected?: NeglectedArea[];
}

export interface CategoryTime {
	category: string;
	hours: number;
	percentage: number;
}

export interface NeglectedArea {
	category: string;
	days_since_last_task: number;
}

export interface UserReflection {
	what_went_well?: string;
	what_didnt_go_well?: string;
	what_learned?: string;
	gratitude?: string[];
	proud_of?: string;
	change_tomorrow?: string;
	additional_notes?: string;
}

export interface GenerateRetroRequest {
	retro_type: 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'yearly' | 'custom';
	date?: string;
	start_date?: string;
	end_date?: string;
}

export interface UpdateRetroRequest {
	user_content?: Partial<UserReflection>;
	status?: 'draft' | 'completed';
}

export interface RetroListResponse {
	retrospectives: Retrospective[];
	total: number;
	limit: number;
	offset: number;
}

// =============================================================================
// RETROSPECTIVE API FUNCTIONS
// =============================================================================

export async function getRetrospectives(params?: {
	limit?: number;
	offset?: number;
}): Promise<RetroListResponse> {
	const searchParams = new URLSearchParams();
	if (params?.limit) searchParams.set('limit', String(params.limit));
	if (params?.offset) searchParams.set('offset', String(params.offset));

	return unwrap(api.get('retrospectives', { searchParams }));
}

export async function getRetrospective(id: string): Promise<Retrospective> {
	return unwrap(api.get(`retrospectives/${encodeURIComponent(id)}`));
}

export async function generateRetrospective(
	data: GenerateRetroRequest,
): Promise<Retrospective> {
	return unwrap(api.post('retrospectives/generate', { json: data }));
}

export async function updateRetrospective(
	id: string,
	data: UpdateRetroRequest,
): Promise<Retrospective> {
	return unwrap(
		api.put(`retrospectives/${encodeURIComponent(id)}`, { json: data }),
	);
}

export async function deleteRetrospective(id: string): Promise<void> {
	return unwrap(api.delete(`retrospectives/${encodeURIComponent(id)}`));
}

export async function regenerateInsights(id: string): Promise<Retrospective> {
	return unwrap(
		api.post(`retrospectives/${encodeURIComponent(id)}/regenerate-insights`),
	);
}
