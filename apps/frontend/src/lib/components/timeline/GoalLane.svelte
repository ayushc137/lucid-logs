<script lang="ts">
    import { cn } from "$lib/utils";
    import {
        Check,
        ChevronDown,
        ChevronRight,
        Flame,
        Plus,
        Target,
    } from "lucide-svelte";
    import { slide } from "svelte/transition";
    import type { TimelineGoal } from "./types";

    interface Props {
        goals: TimelineGoal[];
        highlightedGoalIds?: string[];
        onGoalClick?: (goalId: string) => void;
        onGoalHover?: (goalId: string | null) => void;
        onCreateTaskFromGoal?: (goalId: string) => void;
    }

    let {
        goals = [],
        highlightedGoalIds = [],
        onGoalClick,
        onGoalHover,
        onCreateTaskFromGoal,
    }: Props = $props();

    // Expanded state
    let isExpanded = $state(true);

    // Scroll container ref for auto-scrolling
    let scrollContainer = $state<HTMLDivElement | null>(null);

    // Auto-scroll to first highlighted goal
    $effect(() => {
        if (highlightedGoalIds.length > 0 && scrollContainer && isExpanded) {
            const firstHighlightedId = highlightedGoalIds[0];
            const goalElement = scrollContainer.querySelector(
                `[data-goal-id="${firstHighlightedId}"]`,
            );
            if (goalElement) {
                goalElement.scrollIntoView({
                    behavior: "smooth",
                    block: "nearest",
                    inline: "center",
                });
            }
        }
    });

    // Format recurrence properly (fix "dayly" -> "daily")
    function formatRecurrence(
        rec: { frequency: number; period: string } | undefined,
    ): string {
        if (!rec) return "";
        const times = rec.frequency === 1 ? "" : `${rec.frequency}x/`;
        const periodMap: Record<string, string> = {
            day: "daily",
            week: "weekly",
            month: "monthly",
        };
        return times + (periodMap[rec.period] || `${rec.period}ly`);
    }

    // Format numbers nicely (cap large numbers)
    function formatNumber(value: number, max: number = 99): string {
        if (value > max) return `${max}+`;
        return String(value);
    }

    // Format progress (cap at 999%)
    function formatProgress(value: number): string {
        const rounded = Math.round(value);
        if (rounded > 999) return "999+%";
        return `${rounded}%`;
    }

    // Stats
    const doneCount = $derived(
        goals.filter(
            (g) => g.todayStatus === "met" || g.todayStatus === "exceeded",
        ).length,
    );

    // Local hover state for dimming other goals
    let localHoveredGoalId = $state<string | null>(null);

    function handleLocalHover(goalId: string | null) {
        localHoveredGoalId = goalId;
        onGoalHover?.(goalId);
    }
</script>

{#if goals.length > 0}
    <div
        class="goal-lane bg-base-100 border-b border-base-200 relative z-30"
        transition:slide={{ duration: 200 }}
    >
        <!-- Header (collapsible toggle) -->
        <button
            type="button"
            class="w-full flex items-center justify-between px-4 py-2 hover:bg-base-200/30 transition-colors"
            onclick={() => (isExpanded = !isExpanded)}
        >
            <div class="flex items-center gap-2">
                <div
                    class="w-6 h-6 rounded-md bg-gradient-to-br from-accent to-accent/70 flex items-center justify-center text-accent-content"
                >
                    <Target class="w-3 h-3" />
                </div>
                <span class="font-semibold text-sm">Today's Goals/Habits</span>
                <span class="text-xs text-base-content/50 font-mono">
                    {doneCount}/{goals.length}
                </span>
            </div>
            <div class="flex items-center">
                {#if isExpanded}
                    <ChevronDown class="w-4 h-4 opacity-50" />
                {:else}
                    <ChevronRight class="w-4 h-4 opacity-50" />
                {/if}
            </div>
        </button>

        <!-- Goal Cards (horizontal scroll) -->
        {#if isExpanded}
            <div
                class="px-4 pb-4 pt-1 overflow-x-auto overflow-y-visible scrollbar-thin"
                transition:slide={{ duration: 150 }}
                bind:this={scrollContainer}
            >
                <div class="flex gap-2.5 py-1">
                    {#each goals as goal (goal.id)}
                        {@const isMet =
                            goal.todayStatus === "met" ||
                            goal.todayStatus === "exceeded"}
                        {@const isPending = goal.todayStatus === "pending"}
                        {@const isHighlighted = highlightedGoalIds.includes(
                            goal.id,
                        )}
                        {@const isHovered = localHoveredGoalId === goal.id}
                        {@const isDimmed =
                            localHoveredGoalId &&
                            localHoveredGoalId !== goal.id &&
                            !isHighlighted}

                        <!-- Check if this goal should have the hover effect (direct hover OR linked to hovered task) -->
                        {@const shouldShowHoverEffect = isHovered || isHighlighted}

                        <button
                            type="button"
                            data-goal-id={goal.id}
                            class={cn(
                                "flex items-center gap-2.5 px-3 py-2.5 rounded-xl transition-all duration-300 group border-0 whitespace-nowrap relative",
                                isDimmed
                                    ? "opacity-40 scale-[0.98] grayscale-[50%] blur-[0.5px]"
                                    : "opacity-100",
                                shouldShowHoverEffect
                                    ? "shadow-[0_8px_30px_rgb(0,0,0,0.12)] scale-[1.02] -translate-y-1 z-20 ring-2 ring-white/60"
                                    : isMet
                                      ? "bg-success/10 hover:bg-success/15"
                                      : isPending
                                        ? "bg-warning/5 hover:bg-warning/10"
                                        : "bg-base-200/30 hover:bg-base-200/50",
                            )}
                            style={shouldShowHoverEffect
                                ? `box-shadow: 0 8px 30px rgba(0,0,0,0.12), 0 0 20px ${goal.color}40;`
                                : ""}
                            onclick={() => onGoalClick?.(goal.id)}
                            onmouseenter={() => handleLocalHover(goal.id)}
                            onmouseleave={() => handleLocalHover(null)}
                        >
                            <!-- Plus button - top right corner -->
                            <!-- svelte-ignore a11y_click_events_have_key_events -->
                            <!-- svelte-ignore a11y_no_static_element_interactions -->
                            <div
                                class="absolute -top-1 -right-1 w-5 h-5 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-all duration-200 hover:scale-125 shadow-md z-10"
                                style="background-color: {goal.color}; color: white;"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    onCreateTaskFromGoal?.(goal.id);
                                }}
                            >
                                <Plus class="w-3 h-3" strokeWidth={2.5} />
                            </div>

                            <!-- Icon -->
                            <div
                                class="w-9 h-9 rounded-lg flex items-center justify-center text-lg shrink-0 transition-transform group-hover:scale-110"
                                style="background-color: {goal.color}15; border: 1px solid {goal.color}30;"
                            >
                                {goal.icon}
                            </div>

                            <!-- Content -->
                            <div class="flex flex-col items-start min-w-0">
                                <span
                                    class="font-medium text-sm truncate max-w-28"
                                    class:text-success={isMet}
                                >
                                    {goal.title}
                                </span>
                                <div class="flex items-center gap-1.5 mt-0.5">
                                    {#if goal.recurrence}
                                        <span
                                            class="text-[10px] text-base-content/50"
                                        >
                                            {formatRecurrence(goal.recurrence)}
                                        </span>
                                    {/if}
                                    {#if goal.target}
                                        <span
                                            class="text-[10px] text-base-content/60 font-medium"
                                        >
                                            {formatNumber(
                                                goal.target.currentValue,
                                                999,
                                            )}/{formatNumber(
                                                goal.target.value,
                                                999,
                                            )}
                                        </span>
                                    {/if}
                                </div>
                            </div>

                            <!-- Status Indicator -->
                            <div class="shrink-0 ml-1 flex items-center gap-1">
                                {#if isMet}
                                    <div
                                        class="w-6 h-6 rounded-full bg-success text-success-content flex items-center justify-center"
                                    >
                                        <Check
                                            class="w-3.5 h-3.5"
                                            strokeWidth={3}
                                        />
                                    </div>
                                {:else if goal.currentStreak > 0}
                                    <span
                                        class="flex items-center gap-0.5 text-xs text-warning font-bold bg-warning/10 px-1.5 py-0.5 rounded"
                                    >
                                        <Flame class="w-3 h-3" />
                                        {formatNumber(goal.currentStreak)}
                                    </span>
                                {:else if goal.target && goal.progress > 0}
                                    <span
                                        class="text-[10px] font-bold px-1.5 py-0.5 rounded"
                                        style="color: {goal.color}; background-color: {goal.color}15;"
                                    >
                                        {formatProgress(goal.progress)}
                                    </span>
                                {/if}
                            </div>
                        </button>
                    {/each}
                </div>
            </div>
        {/if}
    </div>
{/if}

<style>
    .scrollbar-thin::-webkit-scrollbar {
        height: 4px;
    }
    .scrollbar-thin::-webkit-scrollbar-track {
        background: transparent;
    }
    .scrollbar-thin::-webkit-scrollbar-thumb {
        background: oklch(var(--bc) / 0.15);
        border-radius: 4px;
    }
    .scrollbar-thin::-webkit-scrollbar-thumb:hover {
        background: oklch(var(--bc) / 0.25);
    }

    @keyframes pulse-glow {
        0%,
        100% {
            opacity: 0.6;
        }
        50% {
            opacity: 1;
        }
    }

    :global(.animate-pulse-glow) {
        animation: pulse-glow 1.5s ease-in-out infinite;
    }
</style>
