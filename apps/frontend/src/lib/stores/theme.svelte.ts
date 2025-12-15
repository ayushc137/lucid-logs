/**
 * Theme Store - 6 Custom Themes for Lucid Logs
 */

export type ThemeId = 'midnight' | 'modern' | 'cozy' | 'ocean' | 'sunset' | 'forest';

export interface Theme {
  id: ThemeId;
  label: string;
  emoji: string;
  description: string;
  isDark: boolean;
}

export const THEMES: Theme[] = [
  { id: 'midnight', label: 'Midnight', emoji: '🌙', description: 'Dark & Premium', isDark: true },
  { id: 'modern', label: 'Modern', emoji: '☀️', description: 'Light & Clean', isDark: false },
  { id: 'cozy', label: 'Cozy', emoji: '📜', description: 'Warm Journal', isDark: false },
  { id: 'ocean', label: 'Ocean', emoji: '🌊', description: 'Cool Calm', isDark: true },
  { id: 'sunset', label: 'Sunset', emoji: '🌅', description: 'Warm Vibrant', isDark: false },
  { id: 'forest', label: 'Forest', emoji: '🌲', description: 'Nature', isDark: false },
];

const STORAGE_KEY = 'lucid-logs-theme';
const DEFAULT_THEME: ThemeId = 'modern';

function createThemeStore() {
  let current = $state<ThemeId>(DEFAULT_THEME);

  // Initialize from localStorage
  if (typeof window !== 'undefined') {
    const stored = localStorage.getItem(STORAGE_KEY) as ThemeId | null;
    if (stored && THEMES.some((t) => t.id === stored)) {
      current = stored;
    }
    // Apply theme immediately
    document.documentElement.setAttribute('data-theme', current);
  }

  return {
    get current() {
      return current;
    },
    get currentTheme() {
      return THEMES.find((t) => t.id === current) || THEMES[0];
    },
    set(theme: ThemeId) {
      current = theme;
      if (typeof window !== 'undefined') {
        localStorage.setItem(STORAGE_KEY, theme);
        document.documentElement.setAttribute('data-theme', theme);
      }
    },
  };
}

export const themeStore = createThemeStore();
