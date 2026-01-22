// Timeline view
export { default as TimelineGantt } from './TimelineGantt.svelte';

// Goal components
export { default as GoalLane } from './GoalLane.svelte';

// Shared components
export { default as DateNavigator } from './DateNavigator.svelte';
export { default as CategoryFilter } from './CategoryFilter.svelte';
export { default as LiveClock } from './LiveClock.svelte';
export { default as TaskPopover } from './TaskPopover.svelte';

// Types
export type {
	TimelineTask,
	TimelineProps,
	TimelineGoal,
	LinkedGoalInfo,
	GoalViewMode,
} from './types';

// NOTE: TimelineAgenda, AgendaTaskCard, and TimelineViewSwitcher have been
// deprecated and moved to $lib/components/graveyard/
