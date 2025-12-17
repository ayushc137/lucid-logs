/**
 * Theme Store - DaisyUI Themes for Lucid Logs
 */

export type ThemeId =
  | 'light' | 'dark' | 'cupcake' | 'emerald' | 'corporate'
  | 'synthwave' | 'retro' | 'cyberpunk' | 'forest' | 'dracula'
  | 'night' | 'coffee' | 'nord' | 'sunset';

export interface Theme {
  id: ThemeId;
  label: string;
  emoji: string;
  description: string;
  isDark: boolean;
}

export const THEMES: Theme[] = [
  { id: 'light', label: 'Light', emoji: '☀️', description: 'Clean & Bright', isDark: false },
  { id: 'dark', label: 'Dark', emoji: '🌙', description: 'Easy on Eyes', isDark: true },
  { id: 'cupcake', label: 'Cupcake', emoji: '🧁', description: 'Sweet Pastel', isDark: false },
  { id: 'emerald', label: 'Emerald', emoji: '💚', description: 'Green Focus', isDark: false },
  { id: 'corporate', label: 'Corporate', emoji: '🏢', description: 'Professional', isDark: false },
  { id: 'synthwave', label: 'Synthwave', emoji: '🌆', description: 'Retro Neon', isDark: true },
  { id: 'retro', label: 'Retro', emoji: '📺', description: 'Vintage Vibes', isDark: false },
  { id: 'cyberpunk', label: 'Cyberpunk', emoji: '🤖', description: 'Futuristic', isDark: true },
  { id: 'forest', label: 'Forest', emoji: '🌲', description: 'Nature', isDark: true },
  { id: 'dracula', label: 'Dracula', emoji: '🧛', description: 'Dark Purple', isDark: true },
  { id: 'night', label: 'Night', emoji: '🌃', description: 'Deep Dark', isDark: true },
  { id: 'coffee', label: 'Coffee', emoji: '☕', description: 'Warm Brown', isDark: true },
  { id: 'nord', label: 'Nord', emoji: '❄️', description: 'Arctic Cool', isDark: true },
  { id: 'sunset', label: 'Sunset', emoji: '🌅', description: 'Warm Glow', isDark: true },
];

const STORAGE_KEY = 'lucid-logs-theme';
const DEFAULT_THEME: ThemeId = 'dark';

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
