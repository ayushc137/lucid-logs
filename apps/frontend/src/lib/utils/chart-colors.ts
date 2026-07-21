/**
 * Chart & category color constants.
 *
 * Design language (lib/DESIGN_LANGUAGE.md §3.4): these are the ONLY hardcoded
 * hex values allowed in the frontend. Everything else uses DaisyUI theme tokens.
 * User-defined category/goal colors come from the API; these are the fallbacks
 * when a color is unset.
 */

/** Fallback for a category with no color chosen (neutral gray). */
export const FALLBACK_CATEGORY_COLOR = '#6b7280';

/** Default color when creating a new category (indigo, matches primary family). */
export const DEFAULT_CATEGORY_COLOR = '#6366f1';

/** Fallback accent for goals with no category color (violet). */
export const FALLBACK_GOAL_COLOR = '#8b5cf6';

/** Convenience: category color with graceful fallback. */
export function categoryColor(color: string | null | undefined): string {
	return color || FALLBACK_CATEGORY_COLOR;
}

/** Convenience: goal color with graceful fallback. */
export function goalColor(color: string | null | undefined): string {
	return color || FALLBACK_GOAL_COLOR;
}
