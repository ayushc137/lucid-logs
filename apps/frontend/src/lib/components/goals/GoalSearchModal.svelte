<script lang="ts">
import { FALLBACK_CATEGORY_COLOR } from '$lib/utils';

    import { type Goal, getGoals, getUnits } from "$lib/api";
    import { cn } from "$lib/utils";
    import { createQuery } from "@tanstack/svelte-query";
    import {
        Check,
        ChevronRight,
        Flame,
        Search,
        Target,
        TrendingDown,
        TrendingUp,
        X,
    } from "lucide-svelte";
    import { fade, scale } from "svelte/transition";

    export interface GoalLinkConfig {
        goal_id: string;
        impact_type: "positive" | "negative";
        quantity_value?: number;
        quantity_unit?: string;
    }

    interface Props {
        open?: boolean;
        onAdd?: (links: GoalLinkConfig[]) => void;
        onClose?: () => void;
        excludeGoalIds?: string[];
    }

    let {
        open = $bindable(false),
        onAdd,
        onClose,
        excludeGoalIds = [],
    }: Props = $props();

    let searchQuery = $state("");
    let searchInputRef = $state<HTMLInputElement | null>(null);
    let pendingLinks = $state<GoalLinkConfig[]>([]);

    const goalsQuery = createQuery({
        queryKey: ["goals", "active"],
        queryFn: () => getGoals({ status: "active", limit: 100 }),
    });

    const unitsQuery = createQuery({
        queryKey: ["units"],
        queryFn: () => getUnits(),
        staleTime: 1000 * 60 * 5,
    });

    const goals = $derived($goalsQuery.data?.items || []);
    const units = $derived($unitsQuery.data?.items || []);

    const allExcludedIds = $derived([
        ...excludeGoalIds,
        ...pendingLinks.map((l) => l.goal_id),
    ]);

    const filteredGoals = $derived(
        goals.filter(
            (g) =>
                g.title.toLowerCase().includes(searchQuery.toLowerCase()) &&
                !allExcludedIds.includes(g.id),
        ),
    );

    const pendingGoals = $derived(
        pendingLinks
            .map((link) => ({
                ...link,
                goal: goals.find((g) => g.id === link.goal_id),
            }))
            .filter((l) => l.goal),
    );

    function getUnitSymbol(unitId?: string): string {
        if (!unitId) return "";
        const unit = units.find((u) => u.id === unitId);
        return unit?.symbol || unitId.replace("units:", "");
    }

    function handleAddGoal(goal: Goal) {
        pendingLinks = [
            ...pendingLinks,
            {
                goal_id: goal.id,
                impact_type: "positive",
                quantity_value: undefined,
                quantity_unit: goal.target?.unit_id,
            },
        ];
    }

    function handleRemovePending(goalId: string) {
        pendingLinks = pendingLinks.filter((l) => l.goal_id !== goalId);
    }

    function handleToggleImpact(goalId: string) {
        pendingLinks = pendingLinks.map((l) =>
            l.goal_id === goalId
                ? {
                      ...l,
                      impact_type:
                          l.impact_type === "positive"
                              ? "negative"
                              : "positive",
                  }
                : l,
        );
    }

    function handleUpdateQuantity(goalId: string, value: number | undefined) {
        pendingLinks = pendingLinks.map((l) =>
            l.goal_id === goalId ? { ...l, quantity_value: value } : l,
        );
    }

    function handleConfirmAll() {
        if (pendingLinks.length === 0) return;
        onAdd?.(pendingLinks);
        handleClose();
    }

    function handleClose() {
        open = false;
        pendingLinks = [];
        searchQuery = "";
        onClose?.();
    }

    function handleBackdropClick(e: MouseEvent) {
        if (e.target === e.currentTarget) {
            handleClose();
        }
    }

    $effect(() => {
        if (open && searchInputRef) {
            setTimeout(() => searchInputRef?.focus(), 100);
        }
    });

    function handleKeydown(e: KeyboardEvent) {
        if (!open) return;
        if (e.key === "Escape") handleClose();
    }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions a11y_interactive_supports_focus -->
    <div
        class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-0 sm:p-6"
        role="presentation"
        onclick={handleBackdropClick}
        transition:fade={{ duration: 150 }}
    >
        <div class="absolute inset-0 bg-base-content/20 backdrop-blur-sm"></div>

        <div
            class="relative bg-base-100 rounded-t-box sm:rounded-box shadow-xl w-full max-w-4xl overflow-hidden border border-base-content/10"
            transition:scale={{ duration: 200, start: 0.95 }}
        >
            <!-- Mobile drag handle -->
            <div class="sm:hidden flex justify-center pt-2.5 pb-1 shrink-0">
                <div class="w-10 h-1 rounded-full bg-base-content/15"></div>
            </div>
            <!-- Header -->
            <div
                class="flex items-center justify-between px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-content/5"
            >
                <div class="flex items-center gap-4">
                    <div
                        class="w-10 h-10 rounded-box bg-primary/10 flex items-center justify-center shrink-0"
                    >
                        <Target class="w-5 h-5 text-primary" />
                    </div>
                    <div>
                        <h2 class="text-lg font-semibold">Link Goals</h2>
                        <p class="text-sm text-base-content/50">
                            Select goals to track with this task
                        </p>
                    </div>
                </div>
                <button class="btn btn-ghost btn-sm btn-square" onclick={handleClose} aria-label="Close">
                    <X class="w-5 h-5" />
                </button>
            </div>

            <!-- Search -->
            <div class="px-4 sm:px-6 py-4">
                <div class="relative">
                    <Search
                        class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-base-content/40"
                    />
                    <input
                        type="text"
                        class="input input-bordered w-full pl-12 h-12 text-base bg-base-100"
                        placeholder="Search goals..."
                        bind:value={searchQuery}
                        bind:this={searchInputRef}
                    />
                </div>
            </div>

            <!-- Content -->
            <div class="flex h-[550px]">
                <!-- Goals List -->
                <div
                    class="flex-1 overflow-y-auto border-r border-base-content/5 p-4"
                >
                    {#if $goalsQuery.isLoading}
                        <div class="flex items-center justify-center h-full">
                            <span
                                class="loading loading-spinner loading-lg text-primary"
                            ></span>
                        </div>
                    {:else if filteredGoals.length === 0}
                        <div
                            class="flex flex-col items-center justify-center h-full text-center"
                        >
                            <div
                                class="w-16 h-16 rounded-2xl bg-base-200 flex items-center justify-center mb-4"
                            >
                                <Target class="w-8 h-8 text-base-content/30" />
                            </div>
                            <p class="font-medium text-base-content/60">
                                {searchQuery
                                    ? "No goals match your search"
                                    : "All goals have been added"}
                            </p>
                        </div>
                    {:else}
                        <div class="space-y-2">
                            {#each filteredGoals as goal (goal.id)}
                                <button
                                    type="button"
                                    class="w-full flex items-center gap-4 p-4 rounded-2xl bg-base-50 hover:bg-base-200/60 border border-transparent hover:border-base-300 transition-all group"
                                    onclick={() => handleAddGoal(goal)}
                                >
                                    <div
                                        class="w-12 h-12 rounded-xl flex items-center justify-center text-2xl shrink-0 transition-transform group-hover:scale-110"
                                        style="background-color: {goal.category
                                            ?.color || FALLBACK_CATEGORY_COLOR}20;"
                                    >
                                        {goal.icon || "🎯"}
                                    </div>
                                    <div class="flex-1 text-left min-w-0">
                                        <div class="font-semibold truncate">
                                            {goal.title}
                                        </div>
                                        <div
                                            class="flex items-center gap-3 text-sm text-base-content/50"
                                        >
                                            {#if goal.category}
                                                <span
                                                    class="flex items-center gap-1.5"
                                                >
                                                    <span
                                                        class="w-2 h-2 rounded-full"
                                                        style="background-color: {goal
                                                            .category.color}"
                                                    ></span>
                                                    {goal.category.name}
                                                </span>
                                            {/if}
                                            {#if goal.target}
                                                <span>
                                                    {goal.stats
                                                        ?.current_value ||
                                                        0}/{goal.target.value}
                                                    {getUnitSymbol(
                                                        goal.target.unit_id,
                                                    )}
                                                </span>
                                            {/if}
                                            {#if (goal.stats?.current_streak || 0) > 0}
                                                <span
                                                    class="flex items-center gap-1 text-warning font-semibold"
                                                >
                                                    <Flame class="w-4 h-4" />
                                                    {goal.stats?.current_streak}
                                                </span>
                                            {/if}
                                        </div>
                                    </div>
                                    <ChevronRight
                                        class="w-5 h-5 text-base-content/30 group-hover:text-primary transition-colors"
                                    />
                                </button>
                            {/each}
                        </div>
                    {/if}
                </div>

                <!-- Selected Panel -->
                <div class="w-96 shrink-0 flex flex-col bg-base-200/30">
                    <div class="px-5 py-4 border-b border-base-200">
                        <h3 class="font-semibold">Selected Goals</h3>
                        <p class="text-sm text-base-content/50">
                            {pendingGoals.length} goal{pendingGoals.length !== 1
                                ? "s"
                                : ""} ready to link
                        </p>
                    </div>

                    <div class="flex-1 overflow-y-auto p-4">
                        {#if pendingGoals.length === 0}
                            <div
                                class="flex flex-col items-center justify-center h-full text-center px-4"
                            >
                                <p class="text-sm text-base-content/40">
                                    Click on goals from the list to add them
                                    here
                                </p>
                            </div>
                        {:else}
                            <div class="space-y-3">
                                {#each pendingGoals as pending (pending.goal_id)}
                                    {@const goal = pending.goal}
                                    {@const isPositive =
                                        pending.impact_type === "positive"}
                                    {#if goal}
                                        <div
                                            class="p-4 rounded-xl bg-base-100 border border-base-200 shadow-sm"
                                            transition:scale={{ duration: 150 }}
                                        >
                                            <div
                                                class="flex items-start gap-3 mb-3"
                                            >
                                                <span class="text-xl"
                                                    >{goal.icon || "🎯"}</span
                                                >
                                                <div class="flex-1 min-w-0">
                                                    <div
                                                        class="font-medium text-sm truncate"
                                                    >
                                                        {goal.title}
                                                    </div>
                                                    {#if goal.target}
                                                        <div
                                                            class="text-xs text-base-content/50"
                                                        >
                                                            {goal.stats
                                                                ?.current_value ||
                                                                0}/{goal.target
                                                                .value}
                                                            {getUnitSymbol(
                                                                goal.target
                                                                    .unit_id,
                                                            )}
                                                        </div>
                                                    {/if}
                                                </div>
                                                <button
                                                    class="p-1 rounded-lg hover:bg-base-200 text-base-content/40 hover:text-base-content"
                                                    onclick={() =>
                                                        handleRemovePending(
                                                            pending.goal_id,
                                                        )}
                                                >
                                                    <X class="w-4 h-4" />
                                                </button>
                                            </div>

                                            <!-- Impact Toggle -->
                                            <div class="flex gap-2 mb-3">
                                                <button
                                                    class={cn(
                                                        "flex-1 flex items-center justify-center gap-2 py-2 rounded-lg text-sm font-medium transition-all",
                                                        isPositive
                                                            ? "bg-success/20 text-success border border-success/30"
                                                            : "bg-base-200 text-base-content/50 hover:bg-base-300",
                                                    )}
                                                    onclick={() =>
                                                        isPositive ||
                                                        handleToggleImpact(
                                                            pending.goal_id,
                                                        )}
                                                >
                                                    <TrendingUp
                                                        class="w-4 h-4"
                                                    />
                                                    Positive
                                                </button>
                                                <button
                                                    class={cn(
                                                        "flex-1 flex items-center justify-center gap-2 py-2 rounded-lg text-sm font-medium transition-all",
                                                        !isPositive
                                                            ? "bg-error/20 text-error border border-error/30"
                                                            : "bg-base-200 text-base-content/50 hover:bg-base-300",
                                                    )}
                                                    onclick={() =>
                                                        !isPositive ||
                                                        handleToggleImpact(
                                                            pending.goal_id,
                                                        )}
                                                >
                                                    <TrendingDown
                                                        class="w-4 h-4"
                                                    />
                                                    Negative
                                                </button>
                                            </div>

                                            <!-- Quantity -->
                                            {#if goal.target}
                                                <div
                                                    class="flex items-center gap-2"
                                                >
                                                    <input
                                                        type="number"
                                                        step="0.1"
                                                        min="0"
                                                        class="input input-bordered input-sm flex-1"
                                                        placeholder="Amount"
                                                        value={pending.quantity_value ??
                                                            ""}
                                                        onchange={(e) => {
                                                            const val =
                                                                parseFloat(
                                                                    e
                                                                        .currentTarget
                                                                        .value,
                                                                );
                                                            handleUpdateQuantity(
                                                                pending.goal_id,
                                                                isNaN(val)
                                                                    ? undefined
                                                                    : val,
                                                            );
                                                        }}
                                                    />
                                                    <span
                                                        class="text-sm font-medium text-base-content/60 min-w-[40px]"
                                                    >
                                                        {getUnitSymbol(
                                                            goal.target.unit_id,
                                                        )}
                                                    </span>
                                                </div>
                                            {/if}
                                        </div>
                                    {/if}
                                {/each}
                            </div>
                        {/if}
                    </div>
                </div>
            </div>

            <!-- Footer -->
            <div
                class="flex items-center justify-end gap-2 px-4 sm:px-6 py-4 border-t border-base-content/5"
            >
                <button class="btn btn-ghost" onclick={handleClose}>
                    Cancel
                </button>
                <button
                    class="btn btn-primary gap-2"
                    onclick={handleConfirmAll}
                    disabled={pendingLinks.length === 0}
                >
                    <Check class="w-5 h-5" />
                    Link {pendingLinks.length} Goal{pendingLinks.length !== 1
                        ? "s"
                        : ""}
                </button>
            </div>
        </div>
    </div>
{/if}
