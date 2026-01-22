// =============================================================================
// TASK COMPONENTS
// Components for task management and display
// =============================================================================

export { default as TaskForm } from './TaskForm.svelte';

// Sub-components
export { default as ReflectionSection } from './ReflectionSection.svelte';
export { default as EmotionSection } from './EmotionSection.svelte';
export { default as ScheduleSection } from './ScheduleSection.svelte';
export { default as TaskPrioritySlider } from './TaskPrioritySlider.svelte';

// NOTE: TaskModal has been deprecated and moved to $lib/components/graveyard/
// The app now uses dedicated pages (/tasks/create and /tasks/[id]) instead of modals
