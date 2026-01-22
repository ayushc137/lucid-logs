<script lang="ts">
    import {
        Calendar,
        Check,
        Coffee,
        Moon,
        Sun,
        Sunrise,
        Sunset,
    } from "lucide-svelte";
    import { cubicOut } from "svelte/easing";
    import { fade, fly } from "svelte/transition";
    import type { TimelineProps, TimelineTask } from "./types";

    import AgendaTaskCard from "./AgendaTaskCard.svelte";
    import CategoryFilter from "./CategoryFilter.svelte";
    // Shared components
    import DateNavigator from "./DateNavigator.svelte";
    import GoalLane from "./GoalLane.svelte";

    let {
        tasks = [],
        goals = [],
        selectedDate = new Date(),
        onTaskClick,
        onDateChange,
        // Goal-related props
        showGoals = true,
        onShowGoalsChange,
        onGoalClick,
        onCreateTaskFromGoal,
    }: TimelineProps = $props();

    let currentTime = $state(new Date());

    // Category filter state
    let selectedCategoryFilter = $state<string | null>(null);

    // Task search state (synced with GoalLane)
    let taskSearchQuery = $state("");

    // Hover states for goal-task highlighting
    let hoveredGoalId = $state<string | null>(null);

    function handleGoalHover(goalId: string | null) {
        hoveredGoalId = goalId;
    }

    // Build a map from goalId -> taskIds using goals' linkedTaskIds
    const goalToTasksMap = $derived.by(() => {
        const map = new Map<string, string[]>();
        for (const goal of goals) {
            if (goal.linkedTaskIds && goal.linkedTaskIds.length > 0) {
                map.set(goal.id, goal.linkedTaskIds);
            }
        }
        return map;
    });

    // Get highlighted task IDs when hovering a goal
    const highlightedTaskIds = $derived.by(() => {
        if (!hoveredGoalId) return new Set<string>();
        const taskIds = goalToTasksMap.get(hoveredGoalId) || [];
        return new Set(taskIds);
    });

    // Build a map from taskId -> goalIds
    const taskToGoalsMap = $derived.by(() => {
        const map = new Map<string, string[]>();

        // Use tasks' linkedGoals if available
        for (const task of tasks) {
            if (task.linkedGoals && task.linkedGoals.length > 0) {
                map.set(
                    task.id,
                    task.linkedGoals.map((g) => g.id),
                );
            }
        }

        // Also build reverse mapping from goals' linkedTaskIds
        for (const goal of goals) {
            if (goal.linkedTaskIds) {
                for (const taskId of goal.linkedTaskIds) {
                    const existing = map.get(taskId) || [];
                    if (!existing.includes(goal.id)) {
                        map.set(taskId, [...existing, goal.id]);
                    }
                }
            }
        }

        return map;
    });

    // Check if a task is linked to the currently hovered goal
    function isTaskLinkedToHoveredGoal(taskId: string): boolean {
        return highlightedTaskIds.has(taskId);
    }

    // Hovered task state for highlighting related goals
    let hoveredTaskId = $state<string | null>(null);

    // Get highlighted goal IDs when hovering a task
    const highlightedGoalIds = $derived.by(() => {
        if (!hoveredTaskId) return [];
        return taskToGoalsMap.get(hoveredTaskId) || [];
    });

    // Update current time for task status
    $effect(() => {
        const interval = setInterval(() => (currentTime = new Date()), 1000);
        return () => clearInterval(interval);
    });

    const isToday = $derived(() => {
        const t = new Date();
        return (
            selectedDate.getFullYear() === t.getFullYear() &&
            selectedDate.getMonth() === t.getMonth() &&
            selectedDate.getDate() === t.getDate()
        );
    });

    const dateLabel = $derived(() => {
        const t = new Date();
        const y = new Date(t);
        y.setDate(y.getDate() - 1);
        const tm = new Date(t);
        tm.setDate(tm.getDate() + 1);
        if (isToday()) return "Today";
        if (selectedDate.toDateString() === y.toDateString())
            return "Yesterday";
        if (selectedDate.toDateString() === tm.toDateString())
            return "Tomorrow";
        return selectedDate.toLocaleDateString("en-US", {
            weekday: "short",
            month: "short",
            day: "numeric",
        });
    });

    function formatTime(d: Date): string {
        if (!(d instanceof Date) || Number.isNaN(d.getTime())) return "--:--";
        return d.toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
            hour12: true,
        });
    }

    function formatDuration(s: Date, e: Date): string {
        if (!(s instanceof Date) || !(e instanceof Date)) return "";
        const mins = Math.round((e.getTime() - s.getTime()) / 60000);
        if (mins < 1) return "<1m";
        if (mins < 60) return `${mins}m`;
        const h = Math.floor(mins / 60);
        const m = mins % 60;
        return m > 0 ? `${h}h ${m}m` : `${h}h`;
    }

    // --- Categories ---
    const categories = $derived.by(() => {
        const catMap = new Map<string, { color: string; count: number }>();
        tasks.forEach((t) => {
            if (t.categoryName && t.categoryColor) {
                const existing = catMap.get(t.categoryName);
                if (existing) {
                    existing.count++;
                } else {
                    catMap.set(t.categoryName, {
                        color: t.categoryColor,
                        count: 1,
                    });
                }
            }
        });
        return Array.from(catMap.entries())
            .map(([name, data]) => ({ name, ...data }))
            .sort((a, b) => b.count - a.count);
    });

    // Count uncategorized tasks
    const uncategorizedCount = $derived(
        tasks.filter((t) => !t.categoryName).length,
    );

    // Filtered tasks - handle category filter AND search query
    const filteredTasks = $derived.by(() => {
        let result = tasks;

        // Apply category filter
        if (selectedCategoryFilter === "__uncategorized__") {
            result = result.filter((t) => !t.categoryName);
        } else if (selectedCategoryFilter) {
            result = result.filter(
                (t) => t.categoryName === selectedCategoryFilter,
            );
        }

        // Apply task search filter
        if (taskSearchQuery.trim()) {
            const query = taskSearchQuery.toLowerCase().trim();
            result = result.filter(
                (t) =>
                    t.title.toLowerCase().includes(query) ||
                    (t.categoryName &&
                        t.categoryName.toLowerCase().includes(query)),
            );
        }

        return result;
    });

    // --- Time Block Logic ---
    function getHourFromDate(d: Date): number {
        return d.getHours();
    }

    type TimeBlock =
        | "early-morning"
        | "morning"
        | "afternoon"
        | "evening"
        | "night";

    function getTimeBlock(hour: number): TimeBlock {
        if (hour >= 5 && hour < 9) return "early-morning";
        if (hour >= 9 && hour < 12) return "morning";
        if (hour >= 12 && hour < 17) return "afternoon";
        if (hour >= 17 && hour < 21) return "evening";
        return "night";
    }

    function getTimeBlockLabel(block: TimeBlock): string {
        switch (block) {
            case "early-morning":
                return "Early Morning";
            case "morning":
                return "Morning";
            case "afternoon":
                return "Afternoon";
            case "evening":
                return "Evening";
            case "night":
                return "Night";
        }
    }

    function getTimeBlockSubLabel(block: TimeBlock): string {
        switch (block) {
            case "early-morning":
                return "5 AM – 9 AM";
            case "morning":
                return "9 AM – 12 PM";
            case "afternoon":
                return "12 PM – 5 PM";
            case "evening":
                return "5 PM – 9 PM";
            case "night":
                return "9 PM – 5 AM";
        }
    }

    function getTimeBlockIcon(block: TimeBlock) {
        switch (block) {
            case "early-morning":
                return Sunrise;
            case "morning":
                return Coffee;
            case "afternoon":
                return Sun;
            case "evening":
                return Sunset;
            case "night":
                return Moon;
        }
    }

    function getTimeBlockGradient(block: TimeBlock): string {
        switch (block) {
            case "early-morning":
                return "from-amber-500/15 to-orange-500/5";
            case "morning":
                return "from-sky-500/15 to-blue-500/5";
            case "afternoon":
                return "from-yellow-500/15 to-amber-500/5";
            case "evening":
                return "from-orange-500/15 to-rose-500/5";
            case "night":
                return "from-indigo-500/15 to-purple-500/5";
        }
    }

    function getTimeBlockIconBg(block: TimeBlock): string {
        switch (block) {
            case "early-morning":
                return "bg-gradient-to-br from-amber-400 to-orange-500";
            case "morning":
                return "bg-gradient-to-br from-sky-400 to-blue-500";
            case "afternoon":
                return "bg-gradient-to-br from-yellow-400 to-amber-500";
            case "evening":
                return "bg-gradient-to-br from-orange-400 to-rose-500";
            case "night":
                return "bg-gradient-to-br from-indigo-400 to-purple-500";
        }
    }

    // Group tasks by time block (using filtered tasks)
    const tasksByBlock = $derived.by(() => {
        const blocks = new Map<TimeBlock, TimelineTask[]>();
        const valid = filteredTasks.filter(
            (t) => t.startTime instanceof Date && t.endTime instanceof Date,
        );
        const sorted = [...valid].sort(
            (a, b) => a.startTime.getTime() - b.startTime.getTime(),
        );

        for (const task of sorted) {
            const hour = getHourFromDate(task.startTime);
            const block = getTimeBlock(hour);
            if (!blocks.has(block)) {
                blocks.set(block, []);
            }
            blocks.get(block)!.push(task);
        }

        return blocks;
    });

    // Get blocks in order
    const blocksInOrder: TimeBlock[] = [
        "early-morning",
        "morning",
        "afternoon",
        "evening",
        "night",
    ];
    const activeBlocks = $derived(
        blocksInOrder.filter((b) => tasksByBlock.has(b)),
    );

    // Stats (use filtered tasks)
    const totalTasks = $derived(filteredTasks.length);
    const completedTasks = $derived(
        filteredTasks.filter((t) => t.completed).length,
    );

    const currentBlock = $derived(getTimeBlock(currentTime.getHours()));

    function getTaskStatus(task: TimelineTask): "past" | "current" | "future" {
        if (!isToday()) return "future";
        const now = currentTime.getTime();
        const start = task.startTime.getTime();
        const end = task.endTime.getTime();

        if (now >= start && now <= end) return "current";
        if (now > end) return "past";
        return "future";
    }
</script>

<div
    class="h-full flex flex-col bg-base-100 rounded-2xl border border-base-200 shadow-xl overflow-hidden select-none"
>
    <!-- Header -->
    <div
        class="flex flex-col md:flex-row items-center justify-between gap-4 px-4 py-3 bg-base-100/95 backdrop-blur-xl border-b border-base-200 sticky top-0 z-40"
    >
        <div class="flex items-center justify-between w-full md:w-auto gap-4">
            <!-- Date Navigation -->
            <DateNavigator {selectedDate} {onDateChange} />

            <!-- Stats (Mobile Compact) -->
            <div class="flex items-center gap-3 text-xs md:hidden">
                <span class="text-base-content/60">
                    <span class="font-bold text-base-content">{totalTasks}</span
                    >
                    tasks
                </span>
                {#if completedTasks > 0}
                    <div
                        class="flex items-center gap-1 text-success font-medium"
                    >
                        <Check class="w-3.5 h-3.5" />
                        {completedTasks}
                    </div>
                {/if}
            </div>
        </div>

        <!-- Center: Category Filter -->
        <div
            class="w-full md:flex-1 flex justify-center order-last md:order-none"
        >
            <div class="w-full md:w-auto flex justify-center">
                <CategoryFilter
                    {categories}
                    totalTaskCount={tasks.length}
                    filteredTaskCount={filteredTasks.length}
                    {uncategorizedCount}
                    selectedCategory={selectedCategoryFilter}
                    onCategoryChange={(cat) => (selectedCategoryFilter = cat)}
                />
            </div>
        </div>

        <!-- Right: Stats (Desktop) -->
        <div class="hidden md:flex items-center gap-3">
            <div class="flex items-center gap-3 text-sm">
                <span class="text-base-content/60">
                    <span class="font-bold text-base-content">{totalTasks}</span
                    >
                    tasks
                </span>
                {#if completedTasks > 0}
                    <div
                        class="flex items-center gap-1 text-success font-medium"
                    >
                        <Check class="w-3.5 h-3.5" />
                        {completedTasks}
                    </div>
                {/if}
            </div>
        </div>
    </div>

    <!-- Goal Lane -->
    {#if showGoals && goals.length > 0}
        <GoalLane
            {goals}
            {highlightedGoalIds}
            {taskSearchQuery}
            onTaskSearchChange={(q) => (taskSearchQuery = q)}
            taskCount={tasks.length}
            {onGoalClick}
            onGoalHover={handleGoalHover}
            {onCreateTaskFromGoal}
        />
    {/if}

    <!-- Agenda Content -->
    <div class="flex-1 overflow-y-auto custom-scrollbar bg-base-200/30">
        {#if activeBlocks.length === 0}
            <div
                class="flex items-center justify-center h-full p-8"
                in:fade={{ duration: 400 }}
            >
                <div
                    class="flex flex-col items-center justify-center text-center p-10 border-2 border-dashed border-base-300/60 rounded-3xl bg-base-100/80 max-w-md shadow-inner"
                >
                    <div class="bg-base-200 p-5 rounded-2xl mb-5">
                        <Calendar class="w-10 h-10 text-base-content/30" />
                    </div>
                    <h3 class="text-lg font-bold text-base-content/70">
                        {#if taskSearchQuery}
                            No tasks matching "{taskSearchQuery}"
                        {:else if selectedCategoryFilter === "__uncategorized__"}
                            No uncategorized tasks
                        {:else if selectedCategoryFilter}
                            No tasks in "{selectedCategoryFilter}"
                        {:else}
                            No tasks scheduled
                        {/if}
                    </h3>
                    <p
                        class="text-sm text-base-content/40 mt-2 leading-relaxed"
                    >
                        {#if taskSearchQuery}
                            Try a different search term or clear the search.
                        {:else if selectedCategoryFilter}
                            Try clearing the filter or add tasks to this
                            category.
                        {:else if isToday()}
                            Your day is wide open! Add some tasks to get
                            started.
                        {:else}
                            Nothing planned for {dateLabel()}. Enjoy the free
                            time!
                        {/if}
                    </p>
                </div>
            </div>
        {:else}
            <div class="p-5 space-y-6">
                {#each activeBlocks as block, i}
                    {@const blockTasks = tasksByBlock.get(block) || []}
                    {@const visibleTasksInBlock = hoveredGoalId
                        ? blockTasks.filter((t) =>
                              isTaskLinkedToHoveredGoal(t.id),
                          )
                        : blockTasks}
                    {@const hasVisibleTasks = visibleTasksInBlock.length > 0}
                    {@const BlockIcon = getTimeBlockIcon(block)}
                    {@const isCurrentBlock =
                        isToday() && block === currentBlock}
                    {@const gradient = getTimeBlockGradient(block)}
                    {@const iconBg = getTimeBlockIconBg(block)}

                    {#if hasVisibleTasks}
                        <div
                            class="relative"
                            in:fly={{
                                y: 15,
                                duration: 350,
                                delay: i * 60,
                                easing: cubicOut,
                            }}
                        >
                            <!-- Block Header -->
                            <div
                                class="flex items-center gap-3 mb-3 px-4 py-3 rounded-xl bg-gradient-to-r {gradient} border border-base-200/40"
                            >
                                <div
                                    class="w-10 h-10 rounded-lg flex items-center justify-center shadow-lg text-white transition-all duration-300
                                     {isCurrentBlock
                                        ? 'ring-4 ring-white/30 scale-110'
                                        : ''} {iconBg}"
                                >
                                    <BlockIcon class="w-5 h-5" />
                                </div>
                                <div class="flex-1">
                                    <div class="flex items-center gap-2">
                                        <h3
                                            class="text-sm font-bold {isCurrentBlock
                                                ? 'text-primary'
                                                : 'text-base-content'}"
                                        >
                                            {getTimeBlockLabel(block)}
                                        </h3>
                                        {#if isCurrentBlock}
                                            <span
                                                class="badge badge-primary badge-xs font-bold"
                                                >NOW</span
                                            >
                                        {/if}
                                    </div>
                                    <p class="text-[11px] text-base-content/50">
                                        {getTimeBlockSubLabel(block)} •
                                        <span class="font-semibold"
                                            >{visibleTasksInBlock.length}</span
                                        >
                                        {visibleTasksInBlock.length === 1
                                            ? "task"
                                            : "tasks"}
                                    </p>
                                </div>
                            </div>

                            <!-- Tasks List -->
                            <div class="space-y-2 pl-1">
                                {#each blockTasks as task, j (task.id)}
                                    {@const isLinkedToHoveredGoal =
                                        isTaskLinkedToHoveredGoal(task.id)}
                                    {@const isDimmed =
                                        hoveredGoalId !== null &&
                                        !isLinkedToHoveredGoal}
                                    <AgendaTaskCard
                                        {task}
                                        status={getTaskStatus(task)}
                                        transitionDelay={i * 30 + j * 40}
                                        {onTaskClick}
                                        isHighlighted={isLinkedToHoveredGoal}
                                        {isDimmed}
                                        isGoalHoverMode={hoveredGoalId !== null}
                                        onMouseEnter={() =>
                                            (hoveredTaskId = task.id)}
                                        onMouseLeave={() =>
                                            (hoveredTaskId = null)}
                                    />
                                {/each}
                            </div>
                        </div>
                    {/if}
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
    .custom-scrollbar::-webkit-scrollbar {
        height: 6px;
        width: 6px;
    }
    .custom-scrollbar::-webkit-scrollbar-corner {
        background: transparent;
    }

    .custom-scrollbar::-webkit-scrollbar-thumb {
        background: oklch(var(--bc) / 0.12);
        border-radius: 10px;
    }
    .custom-scrollbar::-webkit-scrollbar-thumb:hover {
        background: oklch(var(--bc) / 0.2);
    }
    .custom-scrollbar::-webkit-scrollbar-track {
        background: transparent;
    }
</style>
