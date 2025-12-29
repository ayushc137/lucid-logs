<script lang="ts">
    import {
        Clock,
        ChevronLeft,
        ChevronRight,
        Check,
        CalendarClock,
        Coffee,
        Sunrise,
        Sun,
        Sunset,
        Moon,
    } from "lucide-svelte";
    import { onMount, onDestroy } from "svelte";
    import type { TimelineTask, TimelineProps } from "./types";

    let {
        tasks = [],
        selectedDate = new Date(),
        onTaskClick,
        onToggleComplete,
        onCategoryClick,
        onDateChange,
    }: TimelineProps = $props();

    let currentTime = $state(new Date());
    let timeInterval: ReturnType<typeof setInterval>;

    onMount(() => {
        timeInterval = setInterval(() => (currentTime = new Date()), 1000);
    });

    onDestroy(() => {
        if (timeInterval) clearInterval(timeInterval);
    });

    function goToPreviousDay() {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() - 1);
        onDateChange?.(d);
    }

    function goToNextDay() {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() + 1);
        onDateChange?.(d);
    }

    function goToToday() {
        onDateChange?.(new Date());
    }

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
            weekday: "long",
            month: "short",
            day: "numeric",
        });
    });

    function formatTime(d: Date): string {
        if (!(d instanceof Date) || isNaN(d.getTime())) return "--:--";
        return d.toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
            hour12: true,
        });
    }

    function formatDuration(s: Date, e: Date): string {
        if (!(s instanceof Date) || !(e instanceof Date)) return "";
        const mins = Math.round((e.getTime() - s.getTime()) / 60000);
        if (mins < 1)
            return `${Math.round((e.getTime() - s.getTime()) / 1000)}s`;
        if (mins < 60) return `${mins}m`;
        const h = Math.floor(mins / 60),
            m = mins % 60;
        return m > 0 ? `${h}h ${m}m` : `${h}h`;
    }

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

    // Group tasks by time block
    const tasksByBlock = $derived.by(() => {
        const blocks = new Map<TimeBlock, TimelineTask[]>();
        const valid = tasks.filter(
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

    const nowFormatted = $derived(
        currentTime.toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
            hour12: true,
        }),
    );

    function getTaskStatus(task: TimelineTask): "past" | "current" | "future" {
        if (!isToday()) return "future";
        const now = currentTime.getTime();
        const start = task.startTime.getTime();
        const end = task.endTime.getTime();

        if (now >= start && now <= end) return "current";
        if (now > end) return "past";
        return "future";
    }

    const currentBlock = $derived(getTimeBlock(currentTime.getHours()));
</script>

<div
    class="timeline-agenda h-full flex flex-col bg-base-100 rounded-xl border border-base-300/40 overflow-hidden"
>
    <!-- Header (Matching Gantt) -->
    <div
        class="flex items-center justify-between px-3 py-2.5 border-b border-base-200/80 bg-base-100"
    >
        <div class="flex items-center gap-1">
            <button
                class="btn btn-ghost btn-sm btn-square"
                onclick={goToPreviousDay}
            >
                <ChevronLeft class="w-4 h-4" />
            </button>
            <button
                class="btn btn-sm px-4 min-w-[100px] font-semibold {isToday()
                    ? 'btn-primary'
                    : 'btn-ghost'}"
                onclick={goToToday}
            >
                {dateLabel()}
            </button>
            <button
                class="btn btn-ghost btn-sm btn-square"
                onclick={goToNextDay}
            >
                <ChevronRight class="w-4 h-4" />
            </button>
        </div>

        {#if isToday()}
            <div class="flex items-center gap-2">
                <div
                    class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-primary/5 text-primary border border-primary/10"
                >
                    <Clock class="w-3.5 h-3.5" />
                    <span class="text-xs font-semibold font-mono"
                        >{nowFormatted}</span
                    >
                </div>
            </div>
        {/if}
    </div>

    <!-- Agenda Content -->
    <div class="flex-1 overflow-y-auto bg-base-100/50">
        {#if activeBlocks.length === 0}
            <div class="flex items-center justify-center h-full">
                <div class="text-center py-12 px-6 opacity-60">
                    <CalendarClock
                        class="w-12 h-12 mx-auto text-base-content/20 mb-3"
                    />
                    <h3 class="text-sm font-medium text-base-content/40">
                        No tasks for {dateLabel()}
                    </h3>
                    <p class="text-xs text-base-content/30 mt-1">
                        Click "New Task" to add one
                    </p>
                </div>
            </div>
        {:else}
            <div class="pb-6">
                {#each activeBlocks as block}
                    {@const blockTasks = tasksByBlock.get(block) || []}
                    {@const BlockIcon = getTimeBlockIcon(block)}
                    {@const isCurrentBlock =
                        isToday() && block === currentBlock}

                    <div class="relative">
                        <!-- Block Header -->
                        <div
                            class="sticky top-0 z-10 px-4 py-2 bg-base-100/95 backdrop-blur-sm border-b border-base-200/50 flex items-center justify-between"
                        >
                            <div class="flex items-center gap-2">
                                <BlockIcon
                                    class="w-4 h-4 {isCurrentBlock
                                        ? 'text-primary'
                                        : 'text-base-content/40'}"
                                />
                                <h3
                                    class="text-xs font-bold uppercase tracking-wider {isCurrentBlock
                                        ? 'text-primary'
                                        : 'text-base-content/50'}"
                                >
                                    {getTimeBlockLabel(block)}
                                </h3>
                                <span
                                    class="text-[10px] text-base-content/30 font-medium"
                                >
                                    • {blockTasks.length}
                                </span>
                            </div>
                        </div>

                        <!-- Tasks in block -->
                        <div class="px-4 py-2 space-y-2">
                            {#each blockTasks as task (task.id)}
                                {@const bg = task.categoryColor || "#6b7280"}
                                {@const status = getTaskStatus(task)}

                                <!-- svelte-ignore a11y_click_events_have_key_events -->
                                <!-- svelte-ignore a11y_interactive_supports_focus -->
                                <div
                                    class="group flex items-center gap-3 p-3 rounded-lg border bg-base-100 transition-all duration-200
                                        {status === 'current'
                                        ? 'border-primary/30 shadow-sm'
                                        : task.completed
                                          ? 'border-base-200 opacity-60'
                                          : 'border-base-200 hover:border-base-300 hover:shadow-sm'}"
                                    role="button"
                                    onclick={() => onTaskClick?.(task.id)}
                                >
                                    <!-- Time column -->
                                    <div
                                        class="flex flex-col items-end shrink-0 min-w-[50px]"
                                    >
                                        <span
                                            class="text-xs font-bold {status ===
                                            'current'
                                                ? 'text-primary'
                                                : 'text-base-content/70'}"
                                        >
                                            {formatTime(task.startTime)}
                                        </span>
                                        <span
                                            class="text-[10px] text-base-content/40"
                                        >
                                            {formatDuration(
                                                task.startTime,
                                                task.endTime,
                                            )}
                                        </span>
                                    </div>

                                    <!-- Indicator -->
                                    <div
                                        class="w-1 h-8 rounded-full shrink-0"
                                        style="background-color: {bg};"
                                    ></div>

                                    <!-- Content -->
                                    <div class="flex-1 min-w-0">
                                        <h4
                                            class="text-sm font-semibold truncate {task.completed
                                                ? 'line-through opacity-70'
                                                : ''}"
                                        >
                                            {task.title}
                                        </h4>
                                        {#if task.categoryName}
                                            <p
                                                class="text-xs text-base-content/50 truncate"
                                            >
                                                {task.categoryName}
                                            </p>
                                        {/if}
                                    </div>

                                    <!-- Actions/Status -->
                                    <div class="flex items-center gap-2">
                                        {#if task.completed}
                                            <Check
                                                class="w-4 h-4 text-success"
                                            />
                                        {:else if status === "current"}
                                            <div
                                                class="w-2 h-2 rounded-full bg-primary animate-pulse"
                                            ></div>
                                        {/if}
                                    </div>
                                </div>
                            {/each}
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
    .timeline-agenda ::-webkit-scrollbar {
        width: 6px;
    }
    .timeline-agenda ::-webkit-scrollbar-thumb {
        background: oklch(var(--bc) / 0.15);
        border-radius: 4px;
    }
    .timeline-agenda ::-webkit-scrollbar-thumb:hover {
        background: oklch(var(--bc) / 0.25);
    }
    .timeline-agenda ::-webkit-scrollbar-track {
        background: transparent;
    }
</style>
