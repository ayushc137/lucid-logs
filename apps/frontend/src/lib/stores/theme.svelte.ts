/**
 * Theme Store - DaisyUI Themes for Lucid Logs
 * Single source of truth for theme configuration across the app.
 */

export type ThemeId =
  | 'light' | 'dark' | 'nord' | 'night' | 'dim'
  | 'lofi' | 'winter' | 'corporate' | 'retro' | 'dracula';

export interface Theme {
  id: ThemeId;
  label: string;
  emoji: string;
  description: string;
  isDark: boolean;
}

// 10 curated themes - used by both sidebar and settings
export const THEMES: Theme[] = [
  { id: 'light', label: 'Light', emoji: '☀️', description: 'Clean & Bright', isDark: false },
  { id: 'dark', label: 'Dark', emoji: '🌙', description: 'Easy on Eyes', isDark: true },
  { id: 'nord', label: 'Nord', emoji: '❄️', description: 'Arctic Cool', isDark: true },
  { id: 'night', label: 'Night', emoji: '🌃', description: 'Deep Blue', isDark: true },
  { id: 'dim', label: 'Dim', emoji: '🌑', description: 'Soft Dark', isDark: true },
  { id: 'lofi', label: 'Lofi', emoji: '📻', description: 'Muted Tones', isDark: false },
  { id: 'winter', label: 'Winter', emoji: '⛄', description: 'Frosty Light', isDark: false },
  { id: 'corporate', label: 'Corporate', emoji: '💼', description: 'Professional', isDark: false },
  { id: 'retro', label: 'Retro', emoji: '🕹️', description: 'Vintage Vibes', isDark: false },
  { id: 'dracula', label: 'Dracula', emoji: '🧛', description: 'Dark Purple', isDark: true },
];

// Use same key as sidebar was using for backward compatibility
const STORAGE_KEY = 'theme';
const DEFAULT_THEME: ThemeId = 'dark';

function createThemeStore() {
  // Determine initial theme before creating state to avoid state_referenced_locally warning
  let initialTheme: ThemeId = DEFAULT_THEME;
  if (typeof window !== 'undefined') {
    const stored = localStorage.getItem(STORAGE_KEY) as ThemeId | null;
    if (stored && THEMES.some((t) => t.id === stored)) {
      initialTheme = stored;
    }
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
