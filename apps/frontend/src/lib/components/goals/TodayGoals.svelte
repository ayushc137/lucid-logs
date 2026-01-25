<script lang="ts">
import { goto } from '$app/navigation';
import { type Goal, getGoals } from '$lib/api';
import { cn } from '$lib/utils';
import { createQuery } from '@tanstack/svelte-query';
import {
	Check,
	ChevronRight,
	Flame,
	Play,
	Plus,
	Repeat,
	Target,
} from 'lucide-svelte';
import { flip } from 'svelte/animate';
import { fade, fly } from 'svelte/transition';

interface Props {
	selectedDate: Date;
	onCreateTaskFromGoal?: (goal: Goal) => void;
	onGoalClick?: (goal: Goal) => void;
}

let { selectedDate, onCreateTaskFromGoal, onGoalClick }: Props = $props();

const goalsQuery = createQuery({
	queryKey: ['goals', 'active-recurring'],
	queryFn: () => getGoals({ status: 'active', limit: 50 }),
});

// Filter to only recurring goals (habits)
const recurringGoals = $derived(
	($goalsQuery.data?.items || []).filter((g) => g.recurrence),
);

// Sort by streak (descending) and then by priority
const sortedGoals = $derived(
	[...recurringGoals].sort((a, b) => {
		if ((b.stats?.current_streak || 0) !== (a.stats?.current_streak || 0)) {
			return (b.stats?.current_streak || 0) - (a.stats?.current_streak || 0);
		}
		return (b.priority || 2) - (a.priority || 2);
	}),
);

// Show first 5 by default
let expanded = $state(false);
const displayedGoals = $derived(
	expanded ? sortedGoals : sortedGoals.slice(0, 5),
);

function handleGoalClick(goal: Goal) {
	if (onGoalClick) {
		onGoalClick(goal);
	} else if (onCreateTaskFromGoal) {
		onCreateTaskFromGoal(goal);
	}
}

function formatRecurrence(rec: Goal['recurrence']): string {
	if (!rec) return '';
	const times = rec.frequency === 1 ? '' : `${rec.frequency}x `;
	return `${times}${rec.period}ly`;
}
</script>

<div class="space-y-3">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
            <div
                class="w-8 h-8 rounded-lg bg-gradient-to-br from-accent to-accent/70 flex items-center justify-center text-accent-content shadow-sm"
            >
                <Target class="w-4 h-4" />
            </div>
            <span class="font-semibold text-sm">Today's Goals</span>
            {#if recurringGoals.length > 0}
                <span class="badge badge-sm badge-ghost"
                    >{recurringGoals.length}</span
                >
            {/if}
        </div>
        <button
            type="button"
            class="btn btn-ghost btn-xs gap-1"
            onclick={() => goto("/goals")}
        >
            View All
            <ChevronRight class="w-3 h-3" />
        </button>
    </div>

    <!-- Goals List -->
    {#if $goalsQuery.isLoading}
        <div class="flex items-center justify-center py-4">
            <span class="loading loading-spinner loading-sm text-accent"></span>
        </div>
    {:else if sortedGoals.length === 0}
        <div class="text-center py-4">
            <p class="text-sm opacity-50">No active habits</p>
            <button
                type="button"
                class="btn btn-ghost btn-sm mt-2"
                onclick={() => goto("/goals")}
            >
                <Plus class="w-4 h-4" />
                Create a Goal
            </button>
        </div>
    {:else}
        <div class="space-y-2">
            {#each displayedGoals as goal (goal.id)}
                <button
                    type="button"
                    class="flex items-center gap-3 w-full p-2.5 rounded-xl
                        border border-base-200/60 hover:border-accent/40
                        hover:bg-accent/5 transition-all text-left group"
                    onclick={() => handleGoalClick(goal)}
                    in:fly={{ x: -8, duration: 200 }}
                    animate:flip={{ duration: 200 }}
                >
                    <!-- Icon -->
                    <div
                        class="w-9 h-9 rounded-lg flex items-center justify-center text-lg shrink-0 transition-transform group-hover:scale-110"
                        style="background-color: {goal.category?.color ||
                            '#8b5cf6'}20;"
                    >
                        {goal.icon || "🎯"}
                    </div>

                    <!-- Content -->
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2">
                            <span class="font-medium text-sm truncate"
                                >{goal.title}</span
                            >
                            {#if (goal.stats?.current_streak || 0) > 0}
                                <div
                                    class="flex items-center gap-0.5 text-[10px] text-warning font-bold"
                                >
                                    <Flame class="w-3 h-3" />
                                    {goal.stats?.current_streak || 0}
                                </div>
                            {/if}
                        </div>
                        <div class="flex items-center gap-2 mt-0.5">
                            <span
                                class="text-[10px] opacity-50 flex items-center gap-0.5"
                            >
                                <Repeat class="w-2.5 h-2.5" />
                                {formatRecurrence(goal.recurrence)}
                            </span>
                            {#if goal.target}
                                <span class="text-[10px] opacity-50">
                                    {goal.stats?.current_value || 0}/{goal
                                        .target.value}
                                    {goal.target.unit_id}
                                </span>
                            {/if}
                        </div>
                    </div>

                    <!-- Progress or Log Button -->
                    {#if goal.target}
                        {@const progress = Math.min(
                            100,
                            Math.round(
                                ((goal.stats?.current_value || 0) /
                                    goal.target.value) *
                                    100,
                            ),
                        )}
                        <div
                            class="radial-progress text-xs font-bold"
                            style="--value:{progress}; --size:2rem; --thickness:3px; color: {goal
                                .category?.color || '#8b5cf6'};"
                            role="progressbar"
                        >
                            {progress}%
                        </div>
                    {:else}
                        <div
                            class="w-8 h-8 rounded-full flex items-center justify-center
                                bg-accent/10 text-accent opacity-0 group-hover:opacity-100
                                transition-all group-hover:scale-110"
                        >
                            <Plus class="w-4 h-4" />
                        </div>
                    {/if}
                </button>
            {/each}

            {#if sortedGoals.length > 5}
                <button
                    type="button"
                    class="btn btn-ghost btn-sm btn-block gap-1"
                    onclick={() => (expanded = !expanded)}
                >
                    {expanded
                        ? "Show Less"
                        : `Show ${sortedGoals.length - 5} More`}
                    <ChevronRight
                        class="w-3 h-3 transition-transform {expanded
                            ? 'rotate-90'
                            : ''}"
                    />
                </button>
            {/if}
        </div>
    {/if}
</div>
