import { browser } from '$app/environment';
import type { User } from '$lib/api';

/**
 * Auth store using Svelte 5 runes
 * Tracks current user and authentication state
 */
class AuthStore {
    user = $state<User | null>(null);
    token = $state<string | null>(null);
    isLoading = $state(true);

    get isAuthenticated() {
        return !!this.token && !!this.user;
    }

    constructor() {
        if (browser) {
            this.token = localStorage.getItem('token');
            this.isLoading = false;
        }
    }

    setUser(user: User | null) {
        this.user = user;
    }

    setToken(token: string | null) {
        this.token = token;
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
    }

    logout() {
        this.setUser(null);
        this.setToken(null);
    }
}

export const authStore = new AuthStore();
