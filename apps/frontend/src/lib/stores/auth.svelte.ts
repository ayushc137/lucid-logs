import { browser } from '$app/environment';
import type { AuthResponse, User } from '$lib/api';
import { login as apiLogin } from '$lib/api';

// Dev auth bypass config from environment
const DEV_AUTH_BYPASS = import.meta.env.VITE_DEV_AUTH_BYPASS === 'true';
const DEV_ADMIN_EMAIL =
	import.meta.env.VITE_DEV_ADMIN_EMAIL || 'admin@example.com';
const DEV_ADMIN_PASSWORD =
	import.meta.env.VITE_DEV_ADMIN_PASSWORD || 'adminadmin';

/**
 * Check if dev auth bypass is enabled
 */
export function isDevAuthBypassEnabled(): boolean {
	return DEV_AUTH_BYPASS;
}

/**
 * Auth store using Svelte 5 runes
 * Tracks current user and authentication state
 */
class AuthStore {
	user = $state<User | null>(null);
	userIdString = $state<string | null>(null);
	userEmail = $state<string | null>(null);
	isAdmin = $state(false);
	token = $state<string | null>(null);
	isLoading = $state(true);
	isInitialized = $state(false);

	get isAuthenticated() {
		return !!this.token;
	}

	get hasToken() {
		return !!this.token;
	}

	constructor() {
		if (browser) {
			this.token = sessionStorage.getItem('token');
			this.userIdString = sessionStorage.getItem('userId');
			this.userEmail = sessionStorage.getItem('userEmail');
			this.isAdmin = sessionStorage.getItem('isAdmin') === 'true';
			this.isLoading = false;
			// If we have a token in sessionStorage, we're authenticated
			if (this.token) {
				this.isInitialized = true;
			}
		}
	}

	/**
	 * Initialize auth state - just checks if token exists
	 * Since backend getMe has issues, we rely on sessionStorage
	 */
	async initialize(): Promise<boolean> {
		if (this.isInitialized) {
			return this.isAuthenticated;
		}

		this.isLoading = true;

		try {
			if (!this.token) {
				return false;
			}
			// Token exists, consider authenticated
			return true;
		} finally {
			this.isLoading = false;
			this.isInitialized = true;
		}
	}

	/**
	 * Login with auth response from API
	 */
	loginWithResponse(response: AuthResponse, email?: string) {
		this.token = response.token;
		this.userIdString = response.user;
		this.userEmail = email || null;
		this.isAdmin = response.is_admin;
		this.isInitialized = true;
		this.isLoading = false;

		if (browser) {
			sessionStorage.setItem('token', response.token);
			sessionStorage.setItem('userId', response.user);
			sessionStorage.setItem('isAdmin', String(response.is_admin));
			if (email) {
				sessionStorage.setItem('userEmail', email);
			}
		}
	}

	setUser(user: User | null) {
		this.user = user;
	}

	setToken(token: string | null) {
		this.token = token;
		// Reset initialization state when token changes so initialize() will re-run
		this.isInitialized = false;
		this.user = null;
		if (browser) {
			if (token) {
				sessionStorage.setItem('token', token);
			} else {
				sessionStorage.removeItem('token');
			}
		}
	}

	login(user: User, token: string) {
		this.setUser(user);
		this.setToken(token);
		this.isInitialized = true;
	}

	logout() {
		this.user = null;
		this.token = null;
		this.userIdString = null;
		this.userEmail = null;
		this.isAdmin = false;
		this.isInitialized = false;
		if (browser) {
			sessionStorage.removeItem('token');
			sessionStorage.removeItem('userId');
			sessionStorage.removeItem('userEmail');
			sessionStorage.removeItem('isAdmin');
		}
	}

	/**
	 * Perform auto-login using dev credentials from environment variables.
	 * This is used when VITE_DEV_AUTH_BYPASS is enabled.
	 * @returns true if auto-login succeeded, false otherwise
	 */
	async devAutoLogin(): Promise<boolean> {
		if (!DEV_AUTH_BYPASS) {
			return false;
		}

		this.isLoading = true;
		try {
			console.log('[Dev Auth] Auto-login with dev credentials...');
			const response = await apiLogin({
				username: DEV_ADMIN_EMAIL,
				password: DEV_ADMIN_PASSWORD,
			});
			this.loginWithResponse(response, DEV_ADMIN_EMAIL);
			console.log('[Dev Auth] Auto-login successful');
			return true;
		} catch (error) {
			console.error('[Dev Auth] Auto-login failed:', error);
			return false;
		} finally {
			this.isLoading = false;
		}
	}
}

export const authStore = new AuthStore();
