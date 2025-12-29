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
        if (mins < 60) return `${mins}min`;
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

    function getTimeBlockColor(block: TimeBlock): string {
        switch (block) {
            case "early-morning":
                return "from-orange-400/20 to-yellow-400/20 border-orange-400/30";
            case "morning":
                return "from-amber-400/20 to-orange-400/20 border-amber-400/30";
            case "afternoon":
                return "from-blue-400/20 to-cyan-400/20 border-blue-400/30";
            case "evening":
                return "from-purple-400/20 to-pink-400/20 border-purple-400/30";
            case "night":
                return "from-indigo-400/20 to-purple-400/20 border-indigo-400/30";
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
    class="timeline-agenda h-full flex flex-col bg-base-100 rounded-xl border border-base-300/50 overflow-hidden shadow-sm"
>
    <!-- Header -->
    <div
        class="flex items-center justify-between px-5 py-4 border-b border-base-200 bg-gradient-to-r from-base-100 via-base-100 to-base-200/50"
    >
        <div class="flex items-center gap-2">
            <button
                class="btn btn-ghost btn-sm btn-circle"
                onclick={goToPreviousDay}
            >
                <ChevronLeft class="w-4 h-4" />
            </button>
            <div class="text-center min-w-[140px]">
                <button
                    class="font-bold text-lg hover:text-primary transition-colors"
                    onclick={goToToday}
                >
                    {dateLabel()}
                </button>
                <p class="text-xs text-base-content/50">
                    {selectedDate.toLocaleDateString("en-US", {
                        year: "numeric",
                        month: "long",
                        day: "numeric",
                    })}
                </p>
            </div>
            <button
                class="btn btn-ghost btn-sm btn-circle"
                onclick={goToNextDay}
            >
                <ChevronRight class="w-4 h-4" />
            </button>
        </div>

        {#if isToday()}
            <div class="flex items-center gap-2">
                <div
                    class="flex items-center gap-2 px-4 py-2 rounded-full bg-gradient-to-r from-primary/10 to-secondary/10 border border-primary/20"
                >
                    <CalendarClock class="w-4 h-4 text-primary" />
                    <span class="text-sm font-semibold text-primary"
                        >{nowFormatted}</span
                    >
                </div>
            </div>
        {/if}
    </div>

    <!-- Agenda Content -->
    <div class="flex-1 overflow-y-auto">
        {#if activeBlocks.length === 0}
            <div class="flex items-center justify-center h-full">
                <div class="text-center py-12 px-6">
                    <div
                        class="w-20 h-20 rounded-2xl bg-gradient-to-br from-primary/10 to-secondary/10 flex items-center justify-center mx-auto mb-4"
                    >
                        <CalendarClock class="w-10 h-10 text-primary/40" />
                    </div>
                    <h3 class="text-lg font-semibold text-base-content/60">
                        Free Day
                    </h3>
                    <p
                        class="text-sm text-base-content/40 mt-1 max-w-xs mx-auto"
                    >
                        No tasks scheduled for {dateLabel().toLowerCase()}.
                        Enjoy your free time!
                    </p>
                </div>
            </div>
        {:else}
            <div class="divide-y divide-base-200">
                {#each activeBlocks as block}
                    {@const blockTasks = tasksByBlock.get(block) || []}
                    {@const BlockIcon = getTimeBlockIcon(block)}
                    {@const isCurrentBlock =
                        isToday() && block === currentBlock}

                    <div class="relative">
                        <!-- Block Header -->
                        <div
                            class="sticky top-0 z-10 px-5 py-3 bg-gradient-to-r {getTimeBlockColor(
                                block,
                            )} backdrop-blur-sm border-l-4"
                        >
                            <div class="flex items-center gap-3">
                                <div
                                    class="w-8 h-8 rounded-lg bg-base-100/50 flex items-center justify-center"
                                >
                                    <BlockIcon
                                        class="w-4 h-4 {isCurrentBlock
                                            ? 'text-primary'
                                            : 'text-base-content/60'}"
                                    />
                                </div>
                                <div>
                                    <h3
                                        class="font-bold text-sm {isCurrentBlock
                                            ? 'text-primary'
                                            : ''}"
                                    >
                                        {getTimeBlockLabel(block)}
                                    </h3>
                                    <p class="text-xs text-base-content/50">
                                        {blockTasks.length} task{blockTasks.length !==
                                        1
                                            ? "s"
                                            : ""}
                                    </p>
                                </div>
                                {#if isCurrentBlock}
                                    <span
                                        class="ml-auto text-xs font-semibold text-primary bg-primary/10 px-2 py-0.5 rounded-full"
                                    >
                                        Now
                                    </span>
                                {/if}
                            </div>
                        </div>

                        <!-- Tasks in block -->
                        <div class="px-5 py-3 space-y-2">
                            {#each blockTasks as task (task.id)}
                                {@const bg = task.categoryColor || "#6b7280"}
                                {@const status = getTaskStatus(task)}

                                <!-- svelte-ignore a11y_click_events_have_key_events -->
                                <!-- svelte-ignore a11y_interactive_supports_focus -->
                                <div
                                    class="group flex items-stretch gap-4 p-4 rounded-xl border cursor-pointer transition-all duration-200 shadow-sm
                                        {status === 'current'
                                        ? 'bg-base-100 border-primary/40 shadow-md shadow-primary/10 ring-1 ring-primary/20'
                                        : task.completed
                                          ? 'bg-base-100/60 border-success/30'
                                          : 'bg-base-100 border-base-300 hover:border-primary/30 hover:shadow-md'}
                                        {status === 'past' && !task.completed
                                        ? 'opacity-50'
                                        : ''}"
                                    role="button"
                                    onclick={() => onTaskClick?.(task.id)}
                                >
                                    <!-- Time column -->
                                    <div
                                        class="flex flex-col items-center shrink-0 min-w-[70px]"
                                    >
                                        <span
                                            class="text-sm font-bold {status ===
                                            'current'
                                                ? 'text-primary'
                                                : 'text-base-content/70'}"
                                        >
                                            {formatTime(task.startTime)}
                                        </span>
                                        <div
                                            class="flex-1 flex flex-col items-center py-1"
                                        >
                                            <div
                                                class="w-0.5 flex-1 bg-base-300/50"
                                            ></div>
                                        </div>
                                        <span
                                            class="text-xs text-base-content/40"
                                        >
                                            {formatTime(task.endTime)}
                                        </span>
                                    </div>

                                    <!-- Divider with category color -->
                                    <div
                                        class="w-1 rounded-full shrink-0 transition-all duration-300
                                            {status === 'current'
                                            ? 'animate-pulse'
                                            : ''}"
                                        style="background-color: {bg};"
                                    ></div>

                                    <!-- Content -->
                                    <div class="flex-1 min-w-0 py-0.5">
                                        <div
                                            class="flex items-start justify-between gap-3"
                                        >
                                            <div class="flex-1 min-w-0">
                                                <h4
                                                    class="font-semibold text-base truncate {task.completed
                                                        ? 'line-through opacity-60'
                                                        : ''}"
                                                >
                                                    {task.title}
                                                </h4>
                                                {#if task.categoryName}
                                                    <p
                                                        class="text-xs text-base-content/50 mt-0.5 flex items-center gap-1"
                                                    >
                                                        <span
                                                            class="w-2 h-2 rounded-full"
                                                            style="background-color: {bg};"
                                                        ></span>
                                                        {task.categoryName}
                                                    </p>
                                                {/if}
                                            </div>

                                            <div
                                                class="flex items-center gap-2"
                                            >
                                                <span
                                                    class="text-xs font-bold px-2 py-1 rounded-lg bg-base-200/50 text-base-content/70"
                                                >
                                                    {formatDuration(
                                                        task.startTime,
                                                        task.endTime,
                                                    )}
                                                </span>
                                                {#if task.completed}
                                                    <div
                                                        class="w-6 h-6 rounded-full bg-success flex items-center justify-center"
                                                    >
                                                        <Check
                                                            class="w-3.5 h-3.5 text-success-content"
                                                        />
                                                    </div>
                                                {:else if status === "current"}
                                                    <div
                                                        class="w-6 h-6 rounded-full bg-primary/20 flex items-center justify-center"
                                                    >
                                                        <span
                                                            class="w-2 h-2 rounded-full bg-primary animate-pulse"
                                                        ></span>
                                                    </div>
                                                {/if}
                                            </div>
                                        </div>
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
