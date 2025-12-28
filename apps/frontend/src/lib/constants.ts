/**
 * Shared constants for the application.
 * These are reused across multiple components.
 */

/**
 * 16 curated color presets for categories, tags, etc.
 * Colors are arranged in a visually pleasing order for grid display.
 */
export const COLOR_PRESETS = [
    "#ef4444", // Red
    "#f97316", // Orange
    "#eab308", // Yellow
    "#22c55e", // Green
    "#10b981", // Emerald
    "#06b6d4", // Cyan
    "#3b82f6", // Blue
    "#6366f1", // Indigo
    "#8b5cf6", // Violet
    "#a855f7", // Purple
    "#ec4899", // Pink
    "#f43f5e", // Rose
    "#64748b", // Slate
    "#78716c", // Stone
    "#0ea5e9", // Sky
    "#14b8a6", // Teal
] as const;

export type ColorPreset = (typeof COLOR_PRESETS)[number];

/**
 * Default color for new categories, labels, etc.
 */
export const DEFAULT_COLOR = "#6366f1";

/**
 * Calculate contrasting text color (white or dark) based on background luminance.
 * @param hexColor - Hex color string (e.g., "#ef4444")
 * @returns "#ffffff" for dark backgrounds, "#1f2937" for light backgrounds
 */
export function getContrastColor(hexColor: string): string {
    const hex = hexColor.replace("#", "");
    const r = parseInt(hex.substring(0, 2), 16);
    const g = parseInt(hex.substring(2, 4), 16);
    const b = parseInt(hex.substring(4, 6), 16);
    const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
    return luminance > 0.5 ? "#1f2937" : "#ffffff";
}
