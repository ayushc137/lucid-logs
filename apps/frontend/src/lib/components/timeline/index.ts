// Multi-view timeline (recommended)
export { default as TimelineMultiView, type TimelineView } from './TimelineMultiView.svelte';

// Individual view components
export { default as TimelineGantt } from './TimelineGantt.svelte';
export { default as TimelineAgenda } from './TimelineAgenda.svelte';

// View switcher (optional, can be used standalone)
export { default as TimelineViewSwitcher } from './TimelineViewSwitcher.svelte';

// Types
export type { TimelineTask, TimelineProps } from './types';
