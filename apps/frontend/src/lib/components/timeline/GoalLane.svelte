<script lang="ts">
    import { cn } from "$lib/utils";
    import {
        Check,
        ChevronDown,
        ChevronRight,
        Flame,
        ListTodo,
        Plus,
        Target,
        X,
    } from "lucide-svelte";
    import { slide } from "svelte/transition";
    import type { TimelineGoal } from "./types";
    import GoalPopover from "./GoalPopover.svelte";

    interface Props {
        goals: TimelineGoal[];
        taskSearchQuery?: string;
        onTaskSearchChange?: (query: string) => void;
        taskCount?: number;
        onGoalClick?: (goalId: string) => void;
        onGoalHover?: (goalId: string | null) => void;
        onCreateTaskFromGoal?: (goalId: string) => void;
    }

    let {
        goals = [],
        taskSearchQuery = "",
        onTaskSearchChange,
        taskCount = 0,
        onGoalClick,
        onGoalHover,
        onCreateTaskFromGoal,
    }: Props = $props();

    // Expanded state
    let isExpanded = $state(true);

    // Search mode: unified (single search for both) or split (separate searches)
    let isSplitMode = $state(false);

    // Internal search state for goals (used in split mode)
    let goalSearchQuery = $state("");

    // Unified search query (filters both tasks and goals)
    let unifiedSearchQuery = $state("");

    // Get the effective goal search query based on mode
    const effectiveGoalQuery = $derived(
        isSplitMode ? goalSearchQuery : unifiedSearchQuery,
    );

    // Auto-expand when searching
    $effect(() => {
        if (
            (effectiveGoalQuery || (!isSplitMode && unifiedSearchQuery)) &&
            !isExpanded
        ) {
            isExpanded = true;
        }
    });

    // Sync unified search to task search callback when in unified mode
    $effect(() => {
        if (!isSplitMode) {
            onTaskSearchChange?.(unifiedSearchQuery);
        }
    });

    // Clear searches when switching modes
    function toggleSplitMode() {
        if (isSplitMode) {
            // Switching to unified - clear individual searches
            goalSearchQuery = "";
            onTaskSearchChange?.("");
            unifiedSearchQuery = "";
        } else {
            // Switching to split - clear unified search
            unifiedSearchQuery = "";
            onTaskSearchChange?.("");
        }
        isSplitMode = !isSplitMode;
    }

    // Filtered goals based on search
    const filteredGoals = $derived.by(() => {
        if (!effectiveGoalQuery.trim()) return goals;
        const query = effectiveGoalQuery.toLowerCase().trim();
        return goals.filter((g) => g.title.toLowerCase().includes(query));
    });

    // Hover state for popover
    let hoveredGoalId = $state<string | null>(null);
    let popoverPosition = $state<{ x: number; y: number } | null>(null);
    let hoverTimeout: ReturnType<typeof setTimeout> | null = null;

    // Scroll container and scroll state for affordance indicators
    let scrollContainer = $state<HTMLDivElement | null>(null);
    let canScrollLeft = $state(false);
    let canScrollRight = $state(false);

    function updateScrollIndicators() {
        if (!scrollContainer) return;
        const { scrollLeft, scrollWidth, clientWidth } = scrollContainer;
        canScrollLeft = scrollLeft > 5;
        canScrollRight = scrollLeft < scrollWidth - clientWidth - 5;
    }

    $effect(() => {
        if (scrollContainer && isExpanded) {
            updateScrollIndicators();
            // Check again after a brief delay for layout settling
            setTimeout(updateScrollIndicators, 100);
        }
    });

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

    const filteredDoneCount = $derived(
        filteredGoals.filter(
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
        <div class="flex items-center justify-between px-4 py-2.5">
            <button
                type="button"
                class="flex items-center gap-2.5 hover:bg-base-200/30 rounded-lg px-2 py-1 -ml-2 transition-colors"
                onclick={() => (isExpanded = !isExpanded)}
            >
                <div
                    class="w-7 h-7 rounded-lg bg-gradient-to-br from-primary to-primary/70 flex items-center justify-center text-primary-content shadow-sm"
                >
                    <Target class="w-3.5 h-3.5" />
                </div>
                <span class="font-bold text-sm">Today's Goals & Habits</span>
                <span
                    class="text-xs text-base-content/40 font-mono bg-base-200/50 px-2 py-0.5 rounded-full"
                >
                    {#if effectiveGoalQuery}
                        {filteredDoneCount}/{filteredGoals.length}
                    {:else}
                        {doneCount}/{goals.length}
                    {/if}
                </span>
                {#if isExpanded}
                    <ChevronDown class="w-4 h-4 opacity-50" />
                {:else}
                    <ChevronRight class="w-4 h-4 opacity-50" />
                {/if}
            </button>

            <!-- Search Section -->
            <div class="flex items-center gap-2">
                {#if isSplitMode}
                    <!-- Split Mode: Separate Task + Goal Searches -->
                    <!-- Task Search -->
                    <div
                        class={cn(
                            "flex items-center gap-2 px-3 py-1.5 rounded-xl border-2 transition-all duration-200",
                            taskSearchQuery
                                ? "w-40 border-secondary/40 bg-secondary/5 shadow-sm shadow-secondary/10"
                                : "w-32 border-base-200 bg-base-50 hover:border-base-300",
                        )}
                    >
                        <div
                            class={cn(
                                "flex items-center justify-center w-5 h-5 rounded-md shrink-0 transition-all duration-200",
                                taskSearchQuery
                                    ? "bg-gradient-to-br from-secondary/30 to-secondary/10"
                                    : "bg-gradient-to-br from-base-300/50 to-base-200/30",
                            )}
                        >
                            <ListTodo
                                class={cn(
                                    "w-3 h-3 transition-colors duration-200",
                                    taskSearchQuery
                                        ? "text-secondary"
                                        : "text-base-content/40",
                                )}
                            />
                        </div>
                        <input
                            type="text"
                            placeholder="Tasks..."
                            class="flex-1 min-w-0 bg-transparent text-sm outline-none placeholder:text-base-content/40 truncate"
                            value={taskSearchQuery}
                            oninput={(e) =>
                                onTaskSearchChange?.(e.currentTarget.value)}
                        />
                        {#if taskSearchQuery}
                            <button
                                type="button"
                                class="shrink-0 w-4 h-4 rounded flex items-center justify-center hover:bg-base-200 transition-colors"
                                onclick={() => onTaskSearchChange?.("")}
                            >
                                <X class="w-2.5 h-2.5 text-base-content/50" />
                            </button>
                        {/if}
                    </div>

                    <!-- Goal Search -->
                    <div
                        class={cn(
                            "flex items-center gap-2 px-3 py-1.5 rounded-xl border-2 transition-all duration-200",
                            goalSearchQuery
                                ? "w-40 border-primary/40 bg-primary/5 shadow-sm shadow-primary/10"
                                : "w-32 border-base-200 bg-base-50 hover:border-base-300",
                        )}
                    >
                        <div
                            class={cn(
                                "flex items-center justify-center w-5 h-5 rounded-md shrink-0 transition-all duration-200",
                                goalSearchQuery
                                    ? "bg-gradient-to-br from-primary/30 to-primary/10"
                                    : "bg-gradient-to-br from-base-300/50 to-base-200/30",
                            )}
                        >
                            <Target
                                class={cn(
                                    "w-3 h-3 transition-colors duration-200",
                                    goalSearchQuery
                                        ? "text-primary"
                                        : "text-base-content/40",
                                )}
                            />
                        </div>
                        <input
                            type="text"
                            placeholder="Goals..."
                            class="flex-1 min-w-0 bg-transparent text-sm outline-none placeholder:text-base-content/40 truncate"
                            bind:value={goalSearchQuery}
                        />
                        {#if goalSearchQuery}
                            <button
                                type="button"
                                class="shrink-0 w-4 h-4 rounded flex items-center justify-center hover:bg-base-200 transition-colors"
                                onclick={() => (goalSearchQuery = "")}
                            >
                                <X class="w-2.5 h-2.5 text-base-content/50" />
                            </button>
                        {/if}
                    </div>
                {:else}
                    <!-- Unified Mode: Single Search for Both -->
                    <div
                        class={cn(
                            "flex items-center gap-2 px-3 py-1.5 rounded-xl border-2 transition-all duration-200",
                            unifiedSearchQuery
                                ? "w-52 border-accent/40 bg-accent/5 shadow-sm shadow-accent/10"
                                : "w-44 border-base-200 bg-base-50 hover:border-base-300",
                        )}
                    >
                        <div
                            class={cn(
                                "flex items-center justify-center w-6 h-6 rounded-lg shrink-0 transition-all duration-200",
                                unifiedSearchQuery
                                    ? "bg-gradient-to-br from-accent/30 to-accent/10"
                                    : "bg-gradient-to-br from-base-300/50 to-base-200/30",
                            )}
                        >
                            <svg
                                class={cn(
                                    "w-3.5 h-3.5 transition-colors duration-200",
                                    unifiedSearchQuery
                                        ? "text-accent"
                                        : "text-base-content/40",
                                )}
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                            >
                                <circle cx="11" cy="11" r="8" />
                                <path d="m21 21-4.35-4.35" />
                            </svg>
                        </div>
                        <input
                            type="text"
                            placeholder="Search all..."
                            class="flex-1 min-w-0 bg-transparent text-sm outline-none placeholder:text-base-content/40 truncate"
                            bind:value={unifiedSearchQuery}
                        />
                        {#if unifiedSearchQuery}
                            <button
                                type="button"
                                class="shrink-0 w-5 h-5 rounded-md flex items-center justify-center hover:bg-base-200 transition-colors"
                                onclick={() => (unifiedSearchQuery = "")}
                            >
                                <X class="w-3 h-3 text-base-content/50" />
                            </button>
                        {/if}
                    </div>
                {/if}

                <!-- Split/Merge Toggle Button -->
                <button
                    type="button"
                    class={cn(
                        "flex items-center justify-center w-8 h-8 rounded-lg border-2 transition-all duration-200",
                        isSplitMode
                            ? "border-info/40 bg-info/10 text-info shadow-sm"
                            : "border-base-200 bg-base-50 text-base-content/40 hover:border-base-300 hover:text-base-content/60",
                    )}
                    onclick={toggleSplitMode}
                    title={isSplitMode ? "Merge searches" : "Split searches"}
                >
                    {#if isSplitMode}
                        <!-- Merge icon -->
                        <svg
                            class="w-4 h-4"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <path d="M17 8l4 4-4 4" />
                            <path d="M3 12h18" />
                            <path d="M7 8l-4 4 4 4" />
                        </svg>
                    {:else}
                        <!-- Split icon -->
                        <svg
                            class="w-4 h-4"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <path d="M16 3h5v5" />
                            <path d="M8 3H3v5" />
                            <path d="M12 22v-8.3a4 4 0 0 0-1.172-2.872L3 3" />
                            <path d="m15 9 6-6" />
                        </svg>
                    {/if}
                </button>
            </div>
        </div>

        <!-- Goal Cards -->
        {#if isExpanded}
            <div class="relative" transition:slide={{ duration: 150 }}>
                <!-- Left scroll indicator -->
                {#if canScrollLeft}
                    <div class="absolute left-0 top-0 bottom-0 w-12 bg-gradient-to-r from-base-100 to-transparent z-10 pointer-events-none"></div>
                {/if}

                <!-- Right scroll indicator -->
                {#if canScrollRight}
                    <div class="absolute right-0 top-0 bottom-0 w-12 bg-gradient-to-l from-base-100 to-transparent z-10 pointer-events-none"></div>
                {/if}

                <div
                    class="px-4 pb-3 overflow-x-auto goal-scroll"
                    bind:this={scrollContainer}
                    onscroll={updateScrollIndicators}
                >
                    <div class="flex gap-3 py-2">
                        {#each filteredGoals as goal (goal.id)}
                            {@const isMet =
                                goal.todayStatus === "met" ||
                                goal.todayStatus === "exceeded"}
                            {@const isHovered = localHoveredGoalId === goal.id}
                            {@const isDimmed =
                                localHoveredGoalId &&
                                localHoveredGoalId !== goal.id}
                            {@const isHabit = !!goal.recurrence}

                            <button
                                type="button"
                                data-goal-id={goal.id}
                                class={cn(
                                    "goal-card flex flex-col w-48 shrink-0 rounded-xl transition-all duration-200 group overflow-hidden relative",
                                    "bg-base-100 border-2 shadow-md",
                                    isDimmed && "opacity-40",
                                    isHovered && "shadow-xl scale-[1.02] -translate-y-0.5",
                                    isMet
                                        ? "border-success/30 bg-success/5"
                                        : "border-base-200 hover:border-base-300",
                                )}
                                onclick={() => onGoalClick?.(goal.id)}
                                onmouseenter={(e) => handleLocalHover(goal.id, e)}
                                onmousemove={onMouseMove}
                                onmouseleave={() => handleLocalHover(null)}
                            >
                                <!-- Color accent bar at top -->
                                <div
                                    class="h-1.5 w-full"
                                    style="background-color: {goal.color};"
                                ></div>

                                <!-- Completed badge -->
                                {#if isMet}
                                    <div class="absolute top-3 right-2 w-5 h-5 rounded-full bg-success flex items-center justify-center shadow-sm">
                                        <Check class="w-3 h-3 text-success-content" strokeWidth={3} />
                                    </div>
                                {/if}

                                <!-- Card content -->
                                <div class="p-3.5 flex flex-col gap-3 flex-1">
                                    <!-- Header: Icon + Title -->
                                    <div class="flex items-start gap-3">
                                        <!-- Icon - shows goal icon normally, plus icon on hover -->
                                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                                        <!-- svelte-ignore a11y_no_static_element_interactions -->
                                        <div
                                            class="w-10 h-10 rounded-xl flex items-center justify-center text-lg shrink-0 transition-all duration-200 relative cursor-pointer shadow-sm"
                                            style="background-color: {goal.color}20; border: 1px solid {goal.color}30;"
                                            onclick={(e) => {
                                                e.stopPropagation();
                                                onCreateTaskFromGoal?.(goal.id);
                                            }}
                                        >
                                            <span
                                                class="absolute inset-0 flex items-center justify-center transition-all duration-150 group-hover:opacity-0 group-hover:scale-75"
                                            >
                                                {goal.icon}
                                            </span>
                                            <span
                                                class="absolute inset-0 flex items-center justify-center opacity-0 scale-75 transition-all duration-150 group-hover:opacity-100 group-hover:scale-100"
                                                style="color: {goal.color};"
                                            >
                                                <Plus class="w-5 h-5" strokeWidth={2.5} />
                                            </span>
                                        </div>

                                        <div class="flex flex-col min-w-0 flex-1 pr-4">
                                            <h3 class="font-bold text-sm leading-tight truncate text-left">
                                                {goal.title}
                                            </h3>
                                            <span class="text-[10px] text-base-content/40 uppercase tracking-wider font-medium mt-0.5">
                                                {isHabit ? "Habit" : "Goal"}
                                            </span>
                                        </div>
                                    </div>

                                    <!-- Divider -->
                                    <div class="h-px bg-base-200/80"></div>

                                    <!-- Progress or Status -->
                                    <div class="mt-auto">
                                        {#if goal.target}
                                            <div class="flex flex-col gap-2">
                                                <div class="h-2 bg-base-200 rounded-full overflow-hidden">
                                                    <div
                                                        class="h-full rounded-full transition-all duration-300"
                                                        style="width: {Math.min(goal.progress, 100)}%; background-color: {goal.color};"
                                                    ></div>
                                                </div>
                                                <div class="flex items-center justify-between">
                                                    <span class="text-xs text-base-content/50 tabular-nums">
                                                        {Math.round(goal.target.currentValue * 100) / 100} / {formatNumber(goal.target.value)}
                                                    </span>
                                                    <span
                                                        class="text-xs font-bold tabular-nums"
                                                        style="color: {goal.color};"
                                                    >
                                                        {Math.round(goal.progress)}%
                                                    </span>
                                                </div>
                                            </div>
                                        {:else if isMet}
                                            <div class="flex items-center gap-2 text-success bg-success/10 px-2.5 py-1.5 rounded-lg">
                                                <Check class="w-4 h-4" strokeWidth={2.5} />
                                                <span class="text-xs font-bold">Completed today</span>
                                            </div>
                                        {:else if goal.currentStreak > 0}
                                            <div class="flex items-center gap-2 bg-warning/10 px-2.5 py-1.5 rounded-lg">
                                                <Flame class="w-4 h-4 text-warning" />
                                                <span class="text-xs font-bold text-warning tabular-nums">
                                                    {formatNumber(goal.currentStreak)} day streak
                                                </span>
                                            </div>
                                        {:else}
                                            <div class="flex items-center gap-2 text-base-content/40 bg-base-200/50 px-2.5 py-1.5 rounded-lg">
                                                <Target class="w-4 h-4" />
                                                <span class="text-xs font-medium">Not started today</span>
                                            </div>
                                        {/if}
                                    </div>
                                </div>
                            </button>
                        {/each}
                    </div>
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
    .goal-scroll {
        scrollbar-width: none; /* Firefox */
        -ms-overflow-style: none; /* IE/Edge */
    }
    .goal-scroll::-webkit-scrollbar {
        display: none; /* Chrome/Safari */
    }
</style>
