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

// =============================================================================
// MOOD QUADRANT COLORS
// These map to the circumplex model quadrants used in emotion tracking.
// Yellow = high-valence high-arousal (excited, elated)
// Green  = high-valence low-arousal (calm, content)
// Red    = low-valence high-arousal (angry, tense)
// Blue   = low-valence low-arousal (sad, fatigued)
// =============================================================================

export const QUADRANT_COLORS: Record<string, string> = {
	yellow: '#eab308',
	green: '#22c55e',
	red: '#ef4444',
	blue: '#3b82f6',
};

/** Returns the hex color for a mood quadrant name. */
export function quadrantColor(quadrant: string): string {
	return QUADRANT_COLORS[quadrant] ?? FALLBACK_CATEGORY_COLOR;
}

// =============================================================================
// CHART CATEGORICAL PALETTE
// Cycle-through palette for charts that don't have semantic color mapping.
// =============================================================================

export const CHART_COLORS = [
	'#6366f1', // indigo (primary family)
	'#14b8a6', // teal (secondary family)
	'#f59e0b', // amber
	'#ef4444', // red
	'#8b5cf6', // violet
	'#ec4899', // pink
	'#10b981', // emerald
	'#6b7280', // gray
];
