// =============================================================================
// GOAL COMPONENTS
// Components for goal management and display
// =============================================================================

export { default as GoalModal } from './GoalModal.svelte';
export { default as GoalCard } from './GoalCard.svelte';
export { default as GoalSelector } from './GoalSelector.svelte';
export { default as GoalSearchModal, type GoalLinkConfig } from './GoalSearchModal.svelte';
export { default as TodayGoals } from './TodayGoals.svelte';
export { default as GoalPrioritySlider } from './GoalPrioritySlider.svelte';
export { default as GoalTypeSelector } from './GoalTypeSelector.svelte';
export { default as GoalTimeline } from './GoalTimeline.svelte';

// Sub-components
export { default as GoalRecurrenceSettings } from './GoalRecurrenceSettings.svelte';
export { default as GoalTargetSettings } from './GoalTargetSettings.svelte';
export { default as GoalTasksTab } from './GoalTasksTab.svelte';
export { default as GoalHistoryTab } from './GoalHistoryTab.svelte';

// Analytics components (pre-aggregated data visualization)
export { default as GoalProgressChart } from './GoalProgressChart.svelte';
export { default as GoalStreakHistory } from './GoalStreakHistory.svelte';

