<script lang="ts">
    import { cn } from "$lib/utils";
    import {
        Check,
        ChevronDown,
        ChevronRight,
        Flame,
        Plus,
        Repeat,
        Target,
    } from "lucide-svelte";
    import { slide } from "svelte/transition";
    import type { TimelineGoal } from "./types";
    import GoalPopover from "./GoalPopover.svelte";

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

    // Hover state for popover
    let hoveredGoalId = $state<string | null>(null);
    let popoverPosition = $state<{ x: number; y: number } | null>(null);
    let hoverTimeout: ReturnType<typeof setTimeout> | null = null;

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

    // Format recurrence properly
    function formatRecurrence(
        rec: { frequency: number; period: string } | undefined,
    ): string {
        if (!rec) return "";
        const times = rec.frequency === 1 ? "" : `${rec.frequency}x/`;
        const periodMap: Record<string, string> = {
            day: "day",
            week: "week",
            month: "month",
        };
        return times + (periodMap[rec.period] || rec.period);
    }

    // Format numbers nicely
    function formatNumber(value: number, max: number = 99): string {
        if (value > max) return `${max}+`;
        return String(value);
    }

    // Stats
    const doneCount = $derived(
        goals.filter(
            (g) => g.todayStatus === "met" || g.todayStatus === "exceeded",
        ).length,
    );

    // Local hover state for dimming other goals
    let localHoveredGoalId = $state<string | null>(null);

    function handleLocalHover(goalId: string | null, e?: MouseEvent) {
        // Clear any pending hover timeout
        if (hoverTimeout) {
            clearTimeout(hoverTimeout);
            hoverTimeout = null;
        }

        localHoveredGoalId = goalId;
        onGoalHover?.(goalId);

        if (goalId && e) {
            hoveredGoalId = goalId;
            updatePopoverPosition(e);
        } else {
            hoveredGoalId = null;
            popoverPosition = null;
        }
    }

    function updatePopoverPosition(e: MouseEvent) {
        popoverPosition = { x: e.clientX, y: e.clientY };
    }

    function onMouseMove(e: MouseEvent) {
        if (hoveredGoalId) {
            updatePopoverPosition(e);
        }
    }

    const hoveredGoal = $derived(goals.find((g) => g.id === hoveredGoalId));
</script>

{#if goals.length > 0}
    <div
        class="goal-lane bg-base-100 border-b border-base-200 relative z-30"
        transition:slide={{ duration: 200 }}
    >
        <!-- Header -->
        <button
            type="button"
            class="w-full flex items-center justify-between px-4 py-2.5 hover:bg-base-200/30 transition-colors"
            onclick={() => (isExpanded = !isExpanded)}
        >
            <div class="flex items-center gap-2.5">
                <div
                    class="w-7 h-7 rounded-lg bg-gradient-to-br from-primary to-primary/70 flex items-center justify-center text-primary-content shadow-sm"
                >
                    <Target class="w-3.5 h-3.5" />
                </div>
                <span class="font-bold text-sm">Today's Goals & Habits</span>
                <span
                    class="text-xs text-base-content/40 font-mono bg-base-200/50 px-2 py-0.5 rounded-full"
                >
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

        <!-- Goal Cards -->
        {#if isExpanded}
            <div
                class="px-4 overflow-x-auto overflow-visible scrollbar-thin"
                transition:slide={{ duration: 150 }}
                bind:this={scrollContainer}
            >
                <div class="flex gap-2.5 py-3">
                    {#each goals as goal (goal.id)}
                        {@const isMet =
                            goal.todayStatus === "met" ||
                            goal.todayStatus === "exceeded"}
                        {@const isHighlighted = highlightedGoalIds.includes(
                            goal.id,
                        )}
                        {@const isHovered = localHoveredGoalId === goal.id}
                        {@const isDimmed =
                            (localHoveredGoalId &&
                                localHoveredGoalId !== goal.id &&
                                !isHighlighted) ||
                            (highlightedGoalIds.length > 0 && !isHighlighted)}
                        {@const shouldShowHoverEffect =
                            isHovered || isHighlighted}
                        {@const isHabit = !!goal.recurrence}

                        <button
                            type="button"
                            data-goal-id={goal.id}
                            class={cn(
                                "flex items-center gap-2.5 px-3 py-2.5 rounded-xl transition-all duration-200 group border bg-base-100 relative",
                                isDimmed
                                    ? "opacity-30 grayscale border-base-300"
                                    : "opacity-100 border-base-300",
                                shouldShowHoverEffect
                                    ? "shadow-xl z-20 border-2"
                                    : "hover:shadow-md",
                            )}
                            style={shouldShowHoverEffect
                                ? `border-color: ${goal.color};`
                                : ""}
                            onclick={() => onGoalClick?.(goal.id)}
                            onmouseenter={(e) => handleLocalHover(goal.id, e)}
                            onmousemove={onMouseMove}
                            onmouseleave={() => handleLocalHover(null)}
                        >
                            <!-- Icon - shows goal icon normally, plus icon on hover -->
                            <!-- svelte-ignore a11y_click_events_have_key_events -->
                            <!-- svelte-ignore a11y_no_static_element_interactions -->
                            <div
                                class="w-9 h-9 rounded-xl flex items-center justify-center text-lg shrink-0 transition-all duration-200 relative cursor-pointer"
                                style="background: linear-gradient(135deg, {goal.color}15 0%, {goal.color}08 100%); border: 1.5px solid {goal.color}30;"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    onCreateTaskFromGoal?.(goal.id);
                                }}
                            >
                                <span
                                    class="absolute inset-0 flex items-center justify-center transition-all duration-200 group-hover:opacity-0 group-hover:scale-75"
                                >
                                    {goal.icon}
                                </span>
                                <span
                                    class="absolute inset-0 flex items-center justify-center opacity-0 scale-75 transition-all duration-200 group-hover:opacity-100 group-hover:scale-100"
                                    style="color: {goal.color};"
                                >
                                    <Plus
                                        class="w-4.5 h-4.5"
                                        strokeWidth={2.5}
                                    />
                                </span>
                            </div>

                            <!-- Content -->
                            <div class="flex flex-col gap-1.5 min-w-0 flex-1">
                                <!-- Title and type -->
                                <div class="flex items-center gap-2">
                                    <h3
                                        class="font-semibold text-sm leading-tight truncate max-w-[140px]"
                                    >
                                        {goal.title}
                                    </h3>
                                    {#if isHabit}
                                        <div
                                            class="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-base-200/50"
                                        >
                                            <Repeat
                                                class="w-2.5 h-2.5 text-base-content/50 shrink-0"
                                            />
                                            <span
                                                class="text-[9px] font-semibold text-base-content/60 uppercase"
                                                >Habit</span
                                            >
                                        </div>
                                    {:else}
                                        <div
                                            class="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-base-200/50"
                                        >
                                            <Target
                                                class="w-2.5 h-2.5 text-base-content/50 shrink-0"
                                            />
                                            <span
                                                class="text-[9px] font-semibold text-base-content/60 uppercase"
                                                >Goal</span
                                            >
                                        </div>
                                    {/if}
                                </div>

                                <!-- Progress or Status -->
                                {#if goal.target}
                                    <div class="flex items-center gap-2.5">
                                        <div
                                            class="flex-1 h-1.5 bg-base-200/60 rounded-full overflow-hidden min-w-[70px]"
                                        >
                                            <div
                                                class="h-full rounded-full transition-all duration-300"
                                                style="width: {Math.min(
                                                    goal.progress,
                                                    100,
                                                )}%; background-color: {goal.color};"
                                            ></div>
                                        </div>
                                        <span
                                            class="text-[10px] font-bold shrink-0 tabular-nums"
                                            style="color: {goal.color};"
                                        >
                                            {Math.round(goal.target.currentValue * 100) / 100}/{formatNumber(goal.target.value)}
                                        </span>
                                    </div>
                                {:else if isMet}
                                    <div
                                        class="flex items-center gap-1.5 text-success"
                                    >
                                        <Check
                                            class="w-3 h-3"
                                            strokeWidth={3}
                                        />
                                        <span
                                            class="text-[10px] font-bold uppercase tracking-wide"
                                            >Completed</span
                                        >
                                    </div>
                                {:else if goal.currentStreak > 0}
                                    <div class="flex items-center gap-1.5">
                                        <Flame class="w-3 h-3 text-warning" />
                                        <span
                                            class="text-[10px] font-bold text-warning tabular-nums"
                                        >
                                            {formatNumber(goal.currentStreak)} day{goal.currentStreak !==
                                            1
                                                ? "s"
                                                : ""}
                                        </span>
                                    </div>
                                {/if}
                            </div>
                        </button>
                    {/each}
                </div>
            </div>
        {/if}
    </div>
{/if}

<!-- Goal Popover -->
{#if hoveredGoal && popoverPosition}
    <GoalPopover goal={hoveredGoal} position={popoverPosition} />
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
</style>
