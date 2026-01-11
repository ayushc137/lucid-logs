<script lang="ts">
import type { GoalLog, GoalLogEvent } from '$lib/api/goals';
import { cn } from '$lib/utils';
import {
	AlertCircle,
	Archive,
	ArrowDown,
	ArrowUp,
	CalendarDays,
	Clock,
	Flame,
	Link2,
	RotateCcw,
	Sparkles,
	Target,
	Trophy,
	Unlink,
} from 'lucide-svelte';

interface Props {
	logs: GoalLog[];
	isLoading?: boolean;
	filter?: 'all' | 'with-tasks' | 'goal-only';
	onFilterChange?: (filter: 'all' | 'with-tasks' | 'goal-only') => void;
}

let {
	logs = [],
	isLoading = false,
	filter = 'all',
	onFilterChange,
}: Props = $props();

// Event display config
const eventConfig: Record<
	GoalLogEvent,
	{ icon: typeof Sparkles; color: string; label: string }
> = {
	created: {
		icon: Sparkles,
		color: 'text-success',
		label: 'Goal created',
	},
	updated: { icon: RotateCcw, color: 'text-info', label: 'Goal updated' },
	completed: {
		icon: Trophy,
		color: 'text-success',
		label: 'Goal completed',
	},
	archived: {
		icon: Archive,
		color: 'text-warning',
		label: 'Goal archived',
	},
	reactivated: {
		icon: Sparkles,
		color: 'text-success',
		label: 'Goal reactivated',
	},
	deleted: {
		icon: AlertCircle,
		color: 'text-error',
		label: 'Goal deleted',
	},
	streak_updated: {
		icon: Flame,
		color: 'text-warning',
		label: 'Streak updated',
	},
	streak_broken: {
		icon: AlertCircle,
		color: 'text-error',
		label: 'Streak broken',
	},
	target_met: {
		icon: Target,
		color: 'text-success',
		label: 'Target reached!',
	},
	target_exceeded: {
		icon: AlertCircle,
		color: 'text-error',
		label: 'Target exceeded',
	},
	period_end: {
		icon: CalendarDays,
		color: 'text-info',
		label: 'Period ended',
	},
	task_linked: {
		icon: Link2,
		color: 'text-primary',
		label: 'Task linked',
	},
	task_unlinked: {
		icon: Unlink,
		color: 'text-warning',
		label: 'Task unlinked',
	},
	child_added: {
		icon: ArrowDown,
		color: 'text-success',
		label: 'Child added',
	},
	child_removed: {
		icon: ArrowUp,
		color: 'text-error',
		label: 'Child removed',
	},
};

// Filter logs based on filter prop
const filteredLogs = $derived.by(() => {
	if (filter === 'all') return logs;
	if (filter === 'with-tasks') {
		return logs.filter(
			(l) =>
				l.event === 'task_linked' ||
				l.event === 'task_unlinked' ||
				l.triggered_by_task_id,
		);
	}
	return logs.filter(
		(l) =>
			l.event !== 'task_linked' &&
			l.event !== 'task_unlinked' &&
			!l.triggered_by_task_id,
	);
});

function formatEventDate(dateStr: string): string {
	const date = new Date(dateStr);
	const now = new Date();
	const diff = now.getTime() - date.getTime();
	const days = Math.floor(diff / (1000 * 60 * 60 * 24));

	if (days === 0) {
		return date.toLocaleTimeString([], {
			hour: '2-digit',
			minute: '2-digit',
		});
	}
	if (days === 1) {
		return 'Yesterday';
	}
	if (days < 7) {
		return `${days} days ago`;
	}
	return date.toLocaleDateString([], {
		month: 'short',
		day: 'numeric',
	});
}

function formatChanges(changes: Record<string, unknown>): Array<{
	key: string;
	value: string;
}> {
	return Object.entries(changes).map(([key, value]) => {
		const formattedKey = key
			.replace(/_/g, ' ')
			.replace(/\b\w/g, (l) => l.toUpperCase());

		let formattedValue: string;
		if (typeof value === 'boolean') {
			formattedValue = value ? '✓ Yes' : '✗ No';
		} else if (typeof value === 'object' && value !== null) {
			// Handle objects more gracefully
			if ('from' in value && 'to' in value) {
				formattedValue = `${value.from} → ${value.to}`;
			} else {
				formattedValue = JSON.stringify(value, null, 2);
			}
		} else if (value === null || value === undefined) {
			formattedValue = '—';
		} else {
			formattedValue = String(value);
		}

		return { key: formattedKey, value: formattedValue };
	});
}
</script>

<div class="space-y-4">
    <!-- Filter Buttons -->
    <div class="flex items-center gap-2 justify-center">
        <button
            class={cn(
                "btn btn-xs",
                filter === "all" ? "btn-primary" : "btn-ghost",
            )}
            onclick={() => onFilterChange?.("all")}
        >
            All
        </button>
        <button
            class={cn(
                "btn btn-xs gap-1",
                filter === "with-tasks" ? "btn-primary" : "btn-ghost",
            )}
            onclick={() => onFilterChange?.("with-tasks")}
        >
            <Link2 class="w-3 h-3" />
            With Tasks
        </button>
        <button
            class={cn(
                "btn btn-xs gap-1",
                filter === "goal-only" ? "btn-primary" : "btn-ghost",
            )}
            onclick={() => onFilterChange?.("goal-only")}
        >
            <Target class="w-3 h-3" />
            Goal Only
        </button>
    </div>

    <!-- Timeline -->
    {#if isLoading}
        <div class="flex items-center justify-center py-8">
            <span class="loading loading-spinner loading-md"></span>
        </div>
    {:else if filteredLogs.length === 0}
        <div class="text-center py-8 text-base-content/40">
            <Clock class="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p class="text-sm">No activity yet</p>
        </div>
    {:else}
        <ul class="timeline timeline-vertical timeline-compact">
            {#each filteredLogs as log, i (log.id)}
                {@const config =
                    eventConfig[log.event as GoalLogEvent] ||
                    eventConfig.updated}
                {@const IconComponent = config.icon}
                {@const hasChanges =
                    log.changes && Object.keys(log.changes).length > 0}

                <li>
                    {#if i > 0}
                        <hr class="bg-base-300" />
                    {/if}

                    <div
                        class="timeline-start text-xs text-base-content/50 pr-2 whitespace-nowrap"
                    >
                        {formatEventDate(log.created_at)}
                    </div>

                    <div class="timeline-middle">
                        <div
                            class={cn(
                                "w-7 h-7 rounded-full flex items-center justify-center",
                                config.color,
                                "bg-base-200",
                            )}
                        >
                            <IconComponent class="w-3.5 h-3.5" />
                        </div>
                    </div>

                    <div
                        class="timeline-end timeline-box py-2 px-3 ml-2 bg-base-100 border-base-300"
                    >
                        <div class="flex items-center gap-2 mb-1">
                            <span class="text-sm font-medium"
                                >{config.label}</span
                            >
                            {#if log.triggered_by_task_id}
                                <span class="badge badge-xs badge-ghost"
                                    >via task</span
                                >
                            {/if}
                        </div>

                        {#if hasChanges}
                            {@const formattedChanges = formatChanges(
                                log.changes!,
                            )}
                            <div class="space-y-0.5 mt-1.5">
                                {#each formattedChanges as change}
                                    <div
                                        class="text-xs text-base-content/60 flex items-start gap-1.5"
                                    >
                                        <span class="opacity-50">•</span>
                                        <span class="font-medium opacity-70"
                                            >{change.key}:</span
                                        >
                                        <span class="opacity-90"
                                            >{change.value}</span
                                        >
                                    </div>
                                {/each}
                            </div>
                        {/if}
                    </div>

                    {#if i < filteredLogs.length - 1}
                        <hr class="bg-base-300" />
                    {/if}
                </li>
            {/each}
        </ul>
    {/if}
</div>
