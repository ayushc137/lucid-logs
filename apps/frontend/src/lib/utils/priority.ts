/**
 * Priority mapping utilities for Category & Priority inheritance
 *
 * Goal priorities (1-3) are mapped to Task priorities (0-5):
 *   Goal 1 (High)   → Task 4 (Critical)
 *   Goal 2 (Medium) → Task 3 (High)
 *   Goal 3 (Low)    → Task 2 (Medium)
 */

const GOAL_TO_TASK_PRIORITY: Record<number, number> = {
	1: 4, // High → Critical
	2: 3, // Medium → High
	3: 2, // Low → Medium
};

/**
 * Maps a goal priority (1-3) to task priority (0-5)
 */
export function mapGoalPriorityToTask(goalPriority: number): number {
	return GOAL_TO_TASK_PRIORITY[goalPriority] ?? 3;
}

/**
 * Finds the highest priority value from an array of priorities
 * Lower number = higher priority, so we use Math.min
 */
export function getHighestPriority(priorities: number[]): number {
	if (priorities.length === 0) return 3; // Default to medium
	return Math.min(...priorities);
}
