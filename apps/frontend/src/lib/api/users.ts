import { api, unwrap } from './client';

// =============================================================================
// USER PREFERENCES TYPES
// =============================================================================

export interface AISettings {
	enabled: boolean;
	provider: string;
	base_url?: string;
	model: string;
	has_key?: boolean;
	api_key?: string; // write-only — only sent on PUT, never returned on GET
}

export interface DailyRetroSettings {
	enabled: boolean;
	time: string;
	timezone: string;
	notification_enabled: boolean;
	auto_generate: boolean;
}

export interface UserPreferences {
	daily_retro?: DailyRetroSettings;
	weekly_retro_day?: string;
	monthly_retro_day?: number;
	timezone?: string;
	ai?: AISettings;
}

export interface UserWithPreferences {
	id: string;
	email: string;
	is_admin: boolean;
	preferences?: UserPreferences;
	created_at: string;
	updated_at: string;
}

export interface UpdatePreferencesRequest {
	daily_retro?: DailyRetroSettings;
	weekly_retro_day?: string;
	monthly_retro_day?: number;
	timezone?: string;
	ai?: AISettings;
}

// =============================================================================
// USER PREFERENCES API FUNCTIONS
// =============================================================================

export async function getUserPreferences(): Promise<UserWithPreferences> {
	return unwrap(api.get('users/me'));
}

export async function updateUserPreferences(
	data: UpdatePreferencesRequest,
): Promise<UserWithPreferences> {
	return unwrap(api.put('users/me/preferences', { json: data }));
}

// =============================================================================
// AI MODELS & DEFAULTS
// =============================================================================

export interface AIModelsResponse {
	models: string[];
}

export interface AIDefaultsResponse {
	provider: string;
	base_url: string;
	model: string;
	has_key: boolean;
}

export async function getAIModels(): Promise<AIModelsResponse> {
	return unwrap(api.get('users/me/ai/models'));
}

export async function getAIDefaults(): Promise<AIDefaultsResponse> {
	return unwrap(api.get('users/me/ai/defaults'));
}
