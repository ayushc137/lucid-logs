import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { page } from '$app/stores';
import { get } from 'svelte/store';

export type ParamValue = string | number | boolean | Date | null | undefined;

interface UpdateUrlOptions {
	replace?: boolean;
	keepFocus?: boolean;
	noScroll?: boolean;
}

/**
 * Updates URL search parameters without reloading the page.
 * Uses SvelteKit's goto function.
 */
export function updateUrlParams(
	params: Record<string, ParamValue>,
	options: UpdateUrlOptions = {},
) {
	if (!browser) return;

	const url = new URL(window.location.href);
	let hasChanges = false;

	Object.entries(params).forEach(([key, value]) => {
		const currentValue = url.searchParams.get(key);
		let newValue: string | null = null;

		if (value instanceof Date) {
			// Use full ISO string for persistence to preserve time info if needed,
			// but consumer can formatting string before passing if they want YYYY-MM-DD
			newValue = value.toISOString();
		} else if (value !== null && value !== undefined && value !== '') {
			newValue = String(value);
		}

		if (newValue !== currentValue) {
			if (newValue === null) {
				url.searchParams.delete(key);
			} else {
				url.searchParams.set(key, newValue);
			}
			hasChanges = true;
		}
	});

	if (hasChanges) {
		goto(url, {
			replaceState: options.replace ?? true,
			keepFocus: options.keepFocus ?? true,
			noScroll: options.noScroll ?? true,
		});
	}
}

/**
 * Helper to get current URL params as a typed object.
 * Useful for initializing state from URL.
 */
export function getUrlParams<T>(
	parsers: Record<keyof T, (val: string | null) => any>,
): T {
	if (!browser) {
		// Return defaults if running on server (can't access window.location)
		// Here we rely on the parser to handle null (missing param) and return default
		const result: any = {};
		for (const key in parsers) {
			result[key] = parsers[key](null);
		}
		return result as T;
	}

	const searchParams = new URLSearchParams(window.location.search);
	const result: any = {};

	for (const key in parsers) {
		const value = searchParams.get(key);
		result[key] = parsers[key](value);
	}

	return result as T;
}

// Common parsers
export const parsers = {
	string:
		(defaultVal = '') =>
		(val: string | null) =>
			val ?? defaultVal,
	number:
		(defaultVal = 0) =>
		(val: string | null) =>
			val ? Number(val) : defaultVal,
	boolean:
		(defaultVal = false) =>
		(val: string | null) =>
			val === 'true',
	date:
		(defaultVal: Date = new Date()) =>
		(val: string | null) =>
			val ? new Date(val) : defaultVal,
	/**
	 * Parses YYYY-MM-DD string to a Date object in LOCAL time (00:00:00).
	 * Note: new Date('YYYY-MM-DD') parses as UTC, which can result in previous day in Western hemispheres.
	 * This parser ensures the date matches the string in the local timezone.
	 */
	dateOnly:
		(defaultVal: Date = new Date()) =>
		(val: string | null) => {
			if (!val) return defaultVal;
			const [y, m, d] = val.split('-').map(Number);
			if (!y || !m || !d) return defaultVal;
			return new Date(y, m - 1, d);
		},
	array:
		(defaultVal: string[] = []) =>
		(val: string | null) =>
			val ? val.split(',') : defaultVal,
};
