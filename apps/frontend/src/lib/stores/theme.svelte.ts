/**
 * Theme Store - DaisyUI Themes for Lucid Logs
 * Single source of truth for theme configuration across the app.
 *
 * Design language (lib/DESIGN_LANGUAGE.md §3.1): exactly two themes ship.
 * Anything else was never designed for and must not be selectable.
 */

export type ThemeId = 'lucid-light' | 'lucid-dark';

export interface Theme {
	id: ThemeId;
	label: string;
	emoji: string;
	description: string;
	isDark: boolean;
}

export const THEMES: Theme[] = [
	{
		id: 'lucid-light',
		label: 'Light',
		emoji: '✨',
		description: 'Clean & modern',
		isDark: false,
	},
	{
		id: 'lucid-dark',
		label: 'Dark',
		emoji: '🌌',
		description: 'Calm & focused',
		isDark: true,
	},
];

// Use same key as sidebar was using for backward compatibility
const STORAGE_KEY = 'theme';
const DEFAULT_THEME: ThemeId = 'lucid-light';

/** Map a stored theme id (possibly from a retired theme) to a shipped one. */
function resolveTheme(stored: string | null): ThemeId {
	if (stored === 'lucid-light' || stored === 'lucid-dark') return stored;
	// Retired dark themes fall back to lucid-dark, light ones to lucid-light.
	const retiredDark = ['dark', 'nord', 'night', 'dim', 'dracula'];
	if (stored && retiredDark.includes(stored)) return 'lucid-dark';
	return DEFAULT_THEME;
}

function createThemeStore() {
	// Determine initial theme before creating state to avoid state_referenced_locally warning
	let initialTheme: ThemeId = DEFAULT_THEME;
	if (typeof window !== 'undefined') {
		initialTheme = resolveTheme(localStorage.getItem(STORAGE_KEY));
		// Apply theme immediately
		document.documentElement.setAttribute('data-theme', initialTheme);
	}

	let current = $state<ThemeId>(initialTheme);

	function set(theme: ThemeId) {
		current = theme;
		if (typeof window !== 'undefined') {
			localStorage.setItem(STORAGE_KEY, theme);
			document.documentElement.setAttribute('data-theme', theme);
		}
	}

	return {
		get current() {
			return current;
		},
		get currentTheme() {
			return THEMES.find((t) => t.id === current) || THEMES[0];
		},
		set,
	};
}

export const themeStore = createThemeStore();
