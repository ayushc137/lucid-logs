import ky from 'ky';
import { browser } from '$app/environment';

/**
 * API base URL - defaults to localhost for development
 */
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

/**
 * Type-safe HTTP client for API calls
 * Includes automatic token handling and error management
 */
export const api = ky.create({
    prefixUrl: `${API_BASE}/api/v1`,
    timeout: 30000,
    hooks: {
        beforeRequest: [
            (request) => {
                // Add auth token to all requests
                if (browser) {
                    const token = localStorage.getItem('token');
                    if (token) {
                        request.headers.set('Authorization', `Bearer ${token}`);
                    }
                }
            }
        ],
        afterResponse: [
            async (_request, _options, response) => {
                // Handle 401 Unauthorized - redirect to login
                if (response.status === 401 && browser) {
                    localStorage.removeItem('token');
                    window.location.href = '/login';
                }
            }
        ]
    }
});

/**
 * Helper to extract data from API response
 */
export async function unwrap<T>(response: Promise<Response>): Promise<T> {
    const res = await response;
    const json = (await res.json()) as { data: T };
    return json.data;
}

/**
 * API Response wrapper type
 */
export interface ApiResponse<T> {
    data: T;
    error?: {
        code: string;
        message: string;
        details?: unknown;
    };
}

/**
 * Paginated response type
 */
export interface PaginatedResponse<T> {
    items: T[];
    total: number;
    limit: number;
    offset: number;
    has_more: boolean;
}
