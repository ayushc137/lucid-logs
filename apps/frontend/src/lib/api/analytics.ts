import { api, unwrap } from './client';

// ===== TYPES =====

export interface CategoryBreakdown {
	category_id: string;
	category_name: string;
	task_count: number;
	hours: number;
	percentage: number;
}

export interface TaskMetrics {
	completion_rate: number;
	total_tasks: number;
	completed_tasks: number;
	postponed_tasks: number;
	abandoned_tasks: number;
	total_duration_hours: number;
	avg_task_duration_minutes: number;
	velocity: number;
	peak_hours: number[];
	focus_score: number;
	by_category?: CategoryBreakdown[];
}

export interface EmotionCount {
	emotion_id: string;
	emotion_name: string;
	quadrant: string;
	count: number;
}

export interface DailyMood {
	date: string;
	valence: number;
	arousal: number;
	quadrant: string;
}

export interface QuadrantDist {
	yellow: number;
	green: number;
	red: number;
	blue: number;
}

export interface EmotionMetrics {
	average_valence: number;
	average_arousal: number;
	mood_stability: number;
	dominant_quadrant: string;
	quadrant_distribution: QuadrantDist;
	emotion_counts: Record<string, number>;
	top_emotions: EmotionCount[];
	trend?: DailyMood[];
}

export interface GoalProgressItem {
	goal_id: string;
	goal_title: string;
	goal_type: string;
	progress: number;
	current_value?: number;
	target_value?: number;
	status: string;
}

export interface StreakInfo {
	goal_id: string;
	goal_title: string;
	current_streak: number;
	longest_streak: number;
}

export interface GoalMetrics {
	active_goals: number;
	completed_goals: number;
	avg_progress: number;
	total_streak_days: number;
	goal_progress: GoalProgressItem[];
	streak_leaders: StreakInfo[];
}

export interface NeglectedCategory {
	category_id: string;
	category_name: string;
	days_since_active: number;
}

export interface LifeDomainBreakdown {
	domain: string;
	hours: number;
	percentage: number;
}

export interface CategoryMetrics {
	total_hours: number;
	distribution: CategoryBreakdown[];
	neglected_areas?: NeglectedCategory[];
	life_domain_distribution?: LifeDomainBreakdown[];
}

export interface DashboardResponse {
	period: string;
	tasks: TaskMetrics;
	emotions: EmotionMetrics;
	goals: GoalMetrics;
	categories: CategoryMetrics;
}

export interface StreaksResponse {
	total_current_streak_days: number;
	longest_streak_ever: number;
	active_streaks: StreakInfo[];
}

export interface ActivityHeatmapDay {
	date: string;
	count: number;
	minutes: number;
	has_entry: boolean;
	intensity: number;
}

export interface ActivityHeatmapResponse {
	days: ActivityHeatmapDay[];
	total_days: number;
	days_logged: number;
	current_streak: number;
	longest_streak: number;
}

// ===== API FUNCTIONS =====

export async function getAnalyticsDashboard(period?: string): Promise<DashboardResponse> {
	const query = period ? `?period=${period}` : '';
	return unwrap(api.get(`analytics/dashboard${query}`));
}

export async function getAnalyticsStreaks(): Promise<StreaksResponse> {
	return unwrap(api.get('analytics/streaks'));
}

export async function getActivityHeatmap(params?: { start_date?: string; end_date?: string }): Promise<ActivityHeatmapResponse> {
	const searchParams = new URLSearchParams();
	if (params?.start_date) searchParams.set('start_date', params.start_date);
	if (params?.end_date) searchParams.set('end_date', params.end_date);
	const query = searchParams.toString();
	return unwrap(api.get(`analytics/activity-heatmap${query ? `?${query}` : ''}`));
}
