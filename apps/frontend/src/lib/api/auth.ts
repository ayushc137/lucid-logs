import { api, unwrap } from './client';

// =============================================================================
// AUTH TYPES
// =============================================================================

export interface User {
	id: string;
	email: string;
	is_admin: boolean;
	created_at: string;
	updated_at: string;
}

export interface LoginRequest {
	username: string;
	password: string;
}

export interface RegisterRequest {
	username: string;
	password: string;
}

export interface AuthResponse {
	token: string;
	user: string; // User ID string like "users:xyz"
	is_admin: boolean;
}

// =============================================================================
// AUTH API FUNCTIONS
// =============================================================================

export async function login(data: LoginRequest): Promise<AuthResponse> {
	return unwrap(api.post('auth/login', { json: data }));
}

export async function register(data: RegisterRequest): Promise<AuthResponse> {
	return unwrap(api.post('auth/register', { json: data }));
}

export async function getMe(): Promise<User> {
	return unwrap(api.get('users/me'));
}

export async function logout(): Promise<void> {
	if (typeof window !== 'undefined') {
		sessionStorage.removeItem('token');
	}
}

/**
 * Store auth token after successful login/register
 */
export function storeToken(token: string): void {
	if (typeof window !== 'undefined') {
		sessionStorage.setItem('token', token);
	}
}

/**
 * Check if user is authenticated
 */
export function isAuthenticated(): boolean {
	if (typeof window === 'undefined') return false;
	return !!sessionStorage.getItem('token');
}
