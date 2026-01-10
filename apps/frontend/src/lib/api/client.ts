import ky from 'ky';
import { browser } from '$app/environment';

/**
 * API base URL - use relative path for dev proxy, full URL for production
 */
const API_BASE = import.meta.env.VITE_API_URL || '';

/**
 * Flag to prevent 401 redirect during initial auth check
 */
let suppressAuthRedirect = true;

/**
 * Call this after auth is confirmed to enable 401 redirects
 */
export function enableAuthRedirect() {
    suppressAuthRedirect = false;
}

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
                    const token = sessionStorage.getItem('token');
                    if (token) {
                        request.headers.set('Authorization', `Bearer ${token}`);
                    }
                }
            }
        ],
        afterResponse: [
            async (_request, _options, response) => {
                if (!response.ok) {
                    console.error('API Error:', response.status, response.statusText, await response.clone().text());
                }
                // Handle 401 Unauthorized - only redirect if auth redirect is enabled
                // and we're not already on the login page
                if (response.status === 401 && browser && !suppressAuthRedirect) {
                    const isOnLoginPage = window.location.pathname.startsWith('/login');
                    if (!isOnLoginPage) {
                        sessionStorage.removeItem('token');
                        window.location.href = '/login';
                    }
                }
            }
        ]
    }
});

/**
 * Helper to extract data from API response
 * Properly handles error responses and extracts error messages
 */
export async function unwrap<T>(response: Promise<Response>): Promise<T> {
    try {
        const res = await response;
        const json = (await res.json()) as { data: T; error?: { code: string; message: string } };

        // If there's an error in the response, throw it
        if (json.error) {
            throw new Error(json.error.message);
        }

        return json.data;
    } catch (error) {
        // Handle ky HTTPError - extract the actual error message from the response
        if (error && typeof error === 'object' && 'response' in error) {
            const httpError = error as { response: Response };
            try {
                // Clone the response to avoid consuming the body
                const clonedResponse = httpError.response.clone();
                const errorJson = await clonedResponse.json() as { error?: { message: string; code: string } };
                if (errorJson.error?.message) {
                    throw new Error(errorJson.error.message);
                }
            } catch (parseError) {
                // If parseError is already the error we created above, re-throw it
                if (parseError instanceof Error && !('response' in parseError)) {
                    throw parseError;
                }
                // Otherwise, it's a parse error - re-throw the original with a better message
            }
        }
        throw error;
    }
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
