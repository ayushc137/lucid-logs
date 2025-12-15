import { defineConfig, presetUno, presetIcons } from 'unocss';
import extractorSvelte from '@unocss/extractor-svelte';

export default defineConfig({
  extractors: [extractorSvelte()],
  presets: [
    presetUno(),
    presetIcons({
      scale: 1.2,
      extraProperties: {
        display: 'inline-block',
        'vertical-align': 'middle',
      },
    }),
  ],
  // DaisyUI is loaded via CSS, not as a UnoCSS preset
  // Custom shortcuts for common patterns
  shortcuts: {
    'glass': 'bg-base-100/80 backdrop-blur-xl',
    'card-comfy': 'card bg-base-100 shadow-lg rounded-3xl',
    'btn-comfy': 'btn rounded-2xl',
  },
  theme: {
    // Extend with comfy border radius
    borderRadius: {
      'comfy': '1.5rem',
      'comfy-lg': '2rem',
    },
  },
  safelist: [
    // Icon classes used dynamically
    'i-lucide-home',
    'i-lucide-list-todo',
    'i-lucide-target',
    'i-lucide-zap',
    'i-lucide-calendar',
    'i-lucide-bar-chart-3',
    'i-lucide-settings',
    'i-lucide-chevron-left',
    'i-lucide-chevron-right',
    'i-lucide-sun',
    'i-lucide-moon',
    'i-lucide-bell',
    'i-lucide-search',
    'i-lucide-menu',
    'i-lucide-plus',
    'i-lucide-palette',
    'i-lucide-check',
    'i-lucide-user',
    'i-lucide-shield',
    'i-lucide-laptop',
    'i-lucide-clock',
    'i-lucide-smile',
    'i-lucide-trending-up',
    'i-lucide-x',
    'i-lucide-log-out',
  ],
});
