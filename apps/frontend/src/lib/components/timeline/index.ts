// Multi-view timeline (recommended)
export { default as TimelineGantt } from './TimelineGantt.svelte';
export { default as TimelineAgenda } from './TimelineAgenda.svelte';

// View switcher
export { default as TimelineViewSwitcher } from './TimelineViewSwitcher.svelte';

// Goal components
export { default as GoalLane } from './GoalLane.svelte';

// Shared components
export { default as DateNavigator } from './DateNavigator.svelte';
export { default as CategoryFilter } from './CategoryFilter.svelte';
export { default as LiveClock } from './LiveClock.svelte';
export { default as TaskPopover } from './TaskPopover.svelte';
export { default as AgendaTaskCard } from './AgendaTaskCard.svelte';

// Types
export type {
    TimelineTask,
    TimelineProps,
    TimelineGoal,
    LinkedGoalInfo,
    GoalViewMode,
} from './types';
