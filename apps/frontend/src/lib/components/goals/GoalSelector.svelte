<script lang="ts">
import { type Goal, getGoals } from '$lib/api';
import { cn } from '$lib/utils';
import { createQuery } from '@tanstack/svelte-query';
import {
	Check,
	ChevronDown,
	Flame,
	Minus,
	Plus,
	Search,
	Target,
	TrendingDown,
	TrendingUp,
	X,
} from 'lucide-svelte';
import { fade, fly } from 'svelte/transition';

interface GoalLink {
	goal_id: string;
	impact_type: 'positive' | 'negative' | 'neutral';
	impact_magnitude: number;
	quantity_value?: number;
	quantity_unit?: string;
}

interface Props {
	value: GoalLink[];
	onChange: (links: GoalLink[]) => void;
	disabled?: boolean;
}

let { value = [], onChange, disabled = false }: Props = $props();

let isOpen = $state(false);
let searchQuery = $state('');
let editingLink = $state<GoalLink | null>(null);

// Fetch goals
const goalsQuery = createQuery({
	queryKey: ['goals', 'active'],
	queryFn: () => getGoals({ status: 'active', limit: 100 }),
});

const goals = $derived($goalsQuery.data?.items || []);
const filteredGoals = $derived(
	goals.filter(
		(g) =>
			g.title.toLowerCase().includes(searchQuery.toLowerCase()) &&
			!value.some((v) => v.goal_id === g.id),
	),
);

// Get selected goals with full data
const selectedGoals = $derived(
	value
		.map((link) => ({
			...link,
			goal: goals.find((g) => g.id === link.goal_id),
		}))
		.filter((l) => l.goal),
);

function addGoal(goal: Goal) {
	const newLink: GoalLink = {
		goal_id: goal.id,
		impact_type: 'positive',
		impact_magnitude: 3,
		quantity_value: goal.target ? undefined : undefined,
		quantity_unit: goal.target?.unit_id,
	};
	editingLink = newLink;
}

function confirmAddGoal() {
	if (!editingLink) return;
	onChange([...value, editingLink]);
	editingLink = null;
	searchQuery = '';
}

function removeGoal(goalId: string) {
	onChange(value.filter((v) => v.goal_id !== goalId));
}

function updateLink(goalId: string, updates: Partial<GoalLink>) {
	onChange(value.map((v) => (v.goal_id === goalId ? { ...v, ...updates } : v)));
}

const impactTypes = [
	{
		value: 'positive',
		label: 'Positive',
		icon: TrendingUp,
		color: 'text-success',
	},
	{
		value: 'negative',
		label: 'Negative',
		icon: TrendingDown,
		color: 'text-error',
	},
	{
		value: 'neutral',
		label: 'Neutral',
		icon: Minus,
		color: 'text-base-content/60',
	},
] as const;
</script>

<div class="space-y-3">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
            <Target class="w-4 h-4 text-accent" />
            <span class="text-sm font-semibold">Linked Goals</span>
            {#if value.length > 0}
                <span class="badge badge-sm badge-ghost">{value.length}</span>
            {/if}
        </div>
        <button
            type="button"
            class="btn btn-ghost btn-xs gap-1"
            onclick={() => (isOpen = !isOpen)}
            {disabled}
        >
            <Plus class="w-3.5 h-3.5" />
            Add Goal
        </button>
    </div>

    <!-- Selected Goals List -->
    {#if selectedGoals.length > 0}
        <div class="space-y-2">
            {#each selectedGoals as link (link.goal_id)}
                {@const goal = link.goal}
                {#if goal}
                    <div
                        class="flex items-center gap-3 p-3 rounded-xl bg-base-200/50 border border-base-200 group"
                        in:fly={{ y: -5, duration: 200 }}
                    >
                        <!-- Goal Icon -->
                        <div
                            class="w-8 h-8 rounded-lg flex items-center justify-center text-base shrink-0"
                            style="background-color: {goal.category?.color ||
                                '#6b7280'}20;"
                        >
                            {goal.icon || "🎯"}
                        </div>

                        <!-- Goal Info -->
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

                            <!-- Impact Controls -->
                            <div class="flex items-center gap-2 mt-1">
                                <!-- Impact Type -->
                                <div class="join join-horizontal">
                                    {#each impactTypes as type}
                                        {@const Icon = type.icon}
                                        <button
                                            type="button"
                                            class={cn(
                                                "join-item btn btn-xs px-2",
                                                link.impact_type === type.value
                                                    ? type.value === "positive"
                                                        ? "btn-success"
                                                        : type.value ===
                                                            "negative"
                                                          ? "btn-error"
                                                          : "btn-ghost btn-active"
                                                    : "btn-ghost",
                                            )}
                                            onclick={() =>
                                                updateLink(link.goal_id, {
                                                    impact_type: type.value,
                                                })}
                                            title={type.label}
                                        >
                                            <Icon class="w-3 h-3" />
                                        </button>
                                    {/each}
                                </div>

                                <!-- Impact Magnitude -->
                                <div class="flex items-center gap-1">
                                    {#each [1, 2, 3, 4, 5] as mag}
                                        <button
                                            type="button"
                                            class={cn(
                                                "w-4 h-4 rounded-full transition-all",
                                                link.impact_magnitude >= mag
                                                    ? link.impact_type ===
                                                      "positive"
                                                        ? "bg-success"
                                                        : link.impact_type ===
                                                            "negative"
                                                          ? "bg-error"
                                                          : "bg-base-content/40"
                                                    : "bg-base-200 hover:bg-base-300",
                                            )}
                                            onclick={() =>
                                                updateLink(link.goal_id, {
                                                    impact_magnitude: mag,
                                                })}
                                            title={`Impact: ${mag}/5`}
                                        ></button>
                                    {/each}
                                </div>

                                <!-- Quantity (for measurable goals) -->
                                {#if goal.target}
                                    <div class="flex items-center gap-1 ml-2">
                                        <input
                                            type="number"
                                            step="0.1"
                                            min="0"
                                            class="input input-bordered input-xs w-16"
                                            placeholder="Qty"
                                            value={link.quantity_value ?? ""}
                                            onchange={(e) => {
                                                const val = parseFloat(
                                                    e.currentTarget.value,
                                                );
                                                updateLink(link.goal_id, {
                                                    quantity_value: isNaN(val)
                                                        ? undefined
                                                        : val,
                                                });
                                            }}
                                        />
                                        <span class="text-[10px] opacity-60"
                                            >{goal.target.unit_id}</span
                                        >
                                    </div>
                                {/if}
                            </div>
                        </div>

                        <!-- Remove Button -->
                        <button
                            type="button"
                            class="btn btn-ghost btn-xs btn-square opacity-0 group-hover:opacity-100 transition-opacity"
                            onclick={() => removeGoal(link.goal_id)}
                        >
                            <X class="w-3.5 h-3.5" />
                        </button>
                    </div>
                {/if}
            {/each}
        </div>
    {/if}

    <!-- Add Goal Dropdown -->
    {#if isOpen}
        <div
            class="p-3 rounded-xl border border-base-200 bg-base-100 shadow-lg space-y-3"
            in:fly={{ y: -5, duration: 200 }}
        >
            <!-- Search -->
            <div class="relative">
                <Search
                    class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-40"
                />
                <input
                    type="text"
                    class="input input-bordered input-sm w-full pl-9"
                    placeholder="Search goals..."
                    bind:value={searchQuery}
                />
            </div>

            <!-- Editing Link (setting impact before adding) -->
            {#if editingLink}
                {@const goal = goals.find((g) => g.id === editingLink?.goal_id)}
                {#if goal}
                    <div
                        class="p-3 rounded-lg bg-accent/10 border border-accent/30 space-y-2"
                    >
                        <div class="flex items-center gap-2">
                            <span class="text-xl">{goal.icon || "🎯"}</span>
                            <span class="font-medium text-sm">{goal.title}</span
                            >
                        </div>

                        <div class="flex items-center gap-3 flex-wrap">
                            <span class="text-xs font-semibold opacity-60"
                                >Impact:</span
                            >
                            <!-- Impact Type -->
                            <div class="join join-horizontal">
                                {#each impactTypes as type}
                                    {@const Icon = type.icon}
                                    <button
                                        type="button"
                                        class={cn(
                                            "join-item btn btn-xs px-2",
                                            editingLink.impact_type ===
                                                type.value
                                                ? type.value === "positive"
                                                    ? "btn-success"
                                                    : type.value === "negative"
                                                      ? "btn-error"
                                                      : "btn-ghost btn-active"
                                                : "btn-ghost",
                                        )}
                                        onclick={() => {
                                            if (editingLink) {
                                                editingLink = {
                                                    ...editingLink,
                                                    impact_type: type.value,
                                                };
                                            }
                                        }}
                                    >
                                        <Icon class="w-3 h-3" />
                                        <span class="text-[10px]"
                                            >{type.label}</span
                                        >
                                    </button>
                                {/each}
                            </div>

                            <!-- Magnitude -->
                            <div class="flex items-center gap-1">
                                {#each [1, 2, 3, 4, 5] as mag}
                                    <button
                                        type="button"
                                        class={cn(
                                            "w-5 h-5 rounded-full transition-all text-[10px] font-bold",
                                            editingLink.impact_magnitude >= mag
                                                ? editingLink.impact_type ===
                                                  "positive"
                                                    ? "bg-success text-success-content"
                                                    : editingLink.impact_type ===
                                                        "negative"
                                                      ? "bg-error text-error-content"
                                                      : "bg-base-content/40 text-base-100"
                                                : "bg-base-200 hover:bg-base-300",
                                        )}
                                        onclick={() => {
                                            if (editingLink) {
                                                editingLink = {
                                                    ...editingLink,
                                                    impact_magnitude: mag,
                                                };
                                            }
                                        }}
                                    >
                                        {mag}
                                    </button>
                                {/each}
                            </div>

                            <!-- Quantity for measurable -->
                            {#if goal.target}
                                <input
                                    type="number"
                                    step="0.1"
                                    min="0"
                                    class="input input-bordered input-xs w-20"
                                    placeholder={goal.target.unit_id}
                                    bind:value={editingLink.quantity_value}
                                />
                            {/if}
                        </div>

                        <div class="flex justify-end gap-2 pt-1">
                            <button
                                type="button"
                                class="btn btn-ghost btn-xs"
                                onclick={() => (editingLink = null)}
                            >
                                Cancel
                            </button>
                            <button
                                type="button"
                                class="btn btn-primary btn-xs gap-1"
                                onclick={confirmAddGoal}
                            >
                                <Check class="w-3 h-3" />
                                Add Link
                            </button>
                        </div>
                    </div>
                {/if}
            {:else}
                <!-- Goals List -->
                <div class="max-h-48 overflow-y-auto space-y-1">
                    {#if $goalsQuery.isLoading}
                        <div class="flex items-center justify-center py-4">
                            <span class="loading loading-spinner loading-sm"
                            ></span>
                        </div>
                    {:else if filteredGoals.length === 0}
                        <p class="text-center text-sm opacity-50 py-4">
                            {searchQuery
                                ? "No goals match your search"
                                : "No goals available"}
                        </p>
                    {:else}
                        {#each filteredGoals as goal (goal.id)}
                            <button
                                type="button"
                                class="flex items-center gap-3 w-full p-2 rounded-lg hover:bg-base-200/60 transition-colors text-left"
                                onclick={() => addGoal(goal)}
                            >
                                <div
                                    class="w-8 h-8 rounded-lg flex items-center justify-center text-base shrink-0"
                                    style="background-color: {goal.category
                                        ?.color || '#6b7280'}20;"
                                >
                                    {goal.icon || "🎯"}
                                </div>
                                <div class="flex-1 min-w-0">
                                    <span
                                        class="font-medium text-sm truncate block"
                                        >{goal.title}</span
                                    >
                                    <span class="text-[10px] opacity-50">
                                        {#if goal.target}
                                            · {goal.stats?.current_value ||
                                                0}/{goal.target.value}
                                            {goal.target.unit_id}
                                        {/if}
                                    </span>
                                </div>
                                {#if (goal.stats?.current_streak || 0) > 0}
                                    <div
                                        class="flex items-center gap-0.5 text-[10px] text-warning font-bold"
                                    >
                                        <Flame class="w-3 h-3" />
                                        {goal.stats?.current_streak || 0}
                                    </div>
                                {/if}
                                <Plus class="w-4 h-4 opacity-40" />
                            </button>
                        {/each}
                    {/if}
                </div>
            {/if}

            <!-- Close Button -->
            {#if !editingLink}
                <button
                    type="button"
                    class="btn btn-ghost btn-sm btn-block"
                    onclick={() => (isOpen = false)}
                >
                    Done
                </button>
            {/if}
        </div>
    {/if}
</div>
