import { api, unwrap } from './client';

// ===== TYPES =====

export interface MoodEvent {
	date: string;
	emotion: string;
	context?: string;
}

export interface QuadrantDist {
	yellow: number;
	green: number;
	red: number;
	blue: number;
}

export interface MoodSummary {
	average_valence: number;
	average_arousal: number;
	dominant_quadrant: string;
	quadrant_distribution: QuadrantDist;
	notable_spikes?: MoodEvent[];
	notable_dips?: MoodEvent[];
}

export interface HabitStatus {
	habit_id: string;
	name: string;
	success_rate?: number;
}

export interface StreakUpdate {
	habit_id: string;
	name: string;
	streak?: number;
	was?: number;
	now?: number;
}

export interface StreaksSummary {
	continued?: StreakUpdate[];
	broken?: StreakUpdate[];
	started?: StreakUpdate[];
}

export interface HabitsSummary {
	met: HabitStatus[];
	partially_met: HabitStatus[];
	missed: HabitStatus[];
	streaks: StreaksSummary;
}

export interface CategoryCount {
	category: string;
	count: number;
	duration_hours: number;
}

export interface TasksSummary {
	completed: number;
	postponed: number;
	canceled: number;
	not_started: number;
	total_duration_hours: number;
	by_category?: CategoryCount[];
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

export interface GoalsSummary {
	net_impact?: GoalImpact[];
	significantly_advanced?: GoalHighlight[];
	negatively_impacted?: GoalHighlight[];
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

export interface CategoriesSummary {
	time_distribution: CategoryTime[];
	neglected?: NeglectedArea[];
}

export interface RetroAutoSummary {
	mood: MoodSummary;
	habits: HabitsSummary;
	tasks: TasksSummary;
	goals: GoalsSummary;
	categories: CategoriesSummary;
	insights?: string[];
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

export interface Retrospective {
	id?: string;
	retro_type: 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'yearly' | 'custom';
	start_date: string;
	end_date: string;
	auto_summary: RetroAutoSummary;
	user_content: UserReflection;
	status: 'draft' | 'completed';
	generated_at: string;
	created_at: string;
	updated_at: string;
}

export interface RetrospectiveListResponse {
	retrospectives: Retrospective[];
	total: number;
	limit: number;
	offset: number;
}

export interface GenerateRetrospectiveRequest {
	retro_type: 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'yearly' | 'custom';
	date?: string;
	start_date?: string;
	end_date?: string;
}

// ===== API FUNCTIONS =====

export async function getRetrospectives(params?: {
	limit?: number;
	offset?: number;
}): Promise<RetrospectiveListResponse> {
	const searchParams = new URLSearchParams();
	if (params?.limit) searchParams.set('limit', String(params.limit));
	if (params?.offset) searchParams.set('offset', String(params.offset));
	const query = searchParams.toString() ? `?${searchParams}` : '';
	return unwrap(api.get(`retrospectives${query}`));
}

export async function getRetrospective(id: string): Promise<Retrospective> {
	return unwrap(api.get(`retrospectives/${id}`));
}

export async function generateRetrospective(
	data: GenerateRetrospectiveRequest,
): Promise<Retrospective> {
	return unwrap(api.post('retrospectives/generate', { json: data }));
}

export async function updateRetrospective(
	id: string,
	data: { user_content?: UserReflection; status?: 'draft' | 'completed' },
): Promise<Retrospective> {
	return unwrap(api.put(`retrospectives/${id}`, { json: data }));
}

export async function deleteRetrospective(id: string): Promise<void> {
	return unwrap(api.delete(`retrospectives/${id}`));
}
