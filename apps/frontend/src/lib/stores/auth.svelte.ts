import { browser } from '$app/environment';
import type { User, AuthResponse } from '$lib/api';

/**
 * Auth store using Svelte 5 runes
 * Tracks current user and authentication state
 */
class AuthStore {
    user = $state<User | null>(null);
    userIdString = $state<string | null>(null);
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
            this.token = localStorage.getItem('token');
            this.userIdString = localStorage.getItem('userId');
            this.isAdmin = localStorage.getItem('isAdmin') === 'true';
            this.isLoading = false;
            // If we have a token in localStorage, we're authenticated
            if (this.token) {
                this.isInitialized = true;
            }
        }
    }

    /**
     * Initialize auth state - just checks if token exists
     * Since backend getMe has issues, we rely on localStorage
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
    loginWithResponse(response: AuthResponse) {
        this.token = response.token;
        this.userIdString = response.user;
        this.isAdmin = response.is_admin;
        this.isInitialized = true;
        this.isLoading = false;
        
        if (browser) {
            localStorage.setItem('token', response.token);
            localStorage.setItem('userId', response.user);
            localStorage.setItem('isAdmin', String(response.is_admin));
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
                localStorage.setItem('token', token);
            } else {
                localStorage.removeItem('token');
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
        this.isAdmin = false;
        this.isInitialized = false;
        if (browser) {
            localStorage.removeItem('token');
            localStorage.removeItem('userId');
            localStorage.removeItem('isAdmin');
        }
    }
}

export const authStore = new AuthStore();
