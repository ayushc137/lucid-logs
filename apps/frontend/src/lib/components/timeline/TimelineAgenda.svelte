<script lang="ts">
    import {
        ChevronLeft,
        ChevronRight,
        Calendar,
        Sunrise,
        Sun,
        Sunset,
        Moon,
        Coffee,
        Check,
    } from "lucide-svelte";
    import { onMount, onDestroy } from "svelte";
    import { fade, slide } from "svelte/transition";
    import type { TimelineTask, TimelineProps } from "./types";

    let {
        tasks = [],
        selectedDate = new Date(),
        onTaskClick,
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
            weekday: "short",
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
        if (mins < 1) return `Less than 1m`;
        if (mins < 60) return `${mins}m`;
        const h = Math.floor(mins / 60),
            m = mins % 60;
        return m > 0 ? `${h}h ${m}m` : `${h}h`;
    }

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
    class="h-full flex flex-col bg-base-100 rounded-2xl border border-base-200 shadow-sm overflow-hidden select-none"
>
    <!-- Header -->
    <div
        class="flex items-center justify-between px-5 py-3 bg-base-100/90 backdrop-blur-md border-b border-base-200 sticky top-0 z-40 transition-all duration-300"
    >
        <div class="flex items-center gap-3">
            <div class="join bg-base-200/50 p-1 rounded-xl shadow-inner">
                <button
                    class="join-item btn btn-sm btn-ghost btn-square rounded-lg hover:bg-base-100 transition-colors"
                    onclick={goToPreviousDay}
                    aria-label="Previous Day"
                >
                    <ChevronLeft class="w-4 h-4" />
                </button>
                <!-- Richer Date Display matching Gantt -->
                <div
                    class="flex flex-col items-center justify-center px-4 min-w-[140px] cursor-pointer hover:opacity-70 transition-opacity"
                    onclick={goToToday}
                    role="button"
                    tabindex="0"
                    onkeydown={(e) => e.key === "Enter" && goToToday()}
                >
                    <span
                        class="text-[10px] font-bold uppercase tracking-wider text-base-content/40 mb-[-2px]"
                        >{selectedDate.getFullYear()}</span
                    >
                    <span
                        class="font-bold text-sm text-base-content leading-tight"
                        class:text-primary={isToday()}>{dateLabel()}</span
                    >
                </div>
                <button
                    class="join-item btn btn-sm btn-ghost btn-square rounded-lg hover:bg-base-100 transition-colors"
                    onclick={goToNextDay}
                    aria-label="Next Day"
                >
                    <ChevronRight class="w-4 h-4" />
                </button>
            </div>
            {#if !isToday()}
                <button
                    class="btn btn-xs btn-ghost text-primary hover:bg-primary/10 rounded-full px-3 transition-all"
                    onclick={goToToday}
                    in:fade={{ duration: 200 }}
                >
                    Return to Today
                </button>
            {/if}
        </div>

        {#if isToday()}
            <div
                class="flex items-center gap-2 px-3 py-1.5 bg-base-200/50 rounded-full border border-base-200/60"
                in:fade
            >
                <div
                    class="w-2 h-2 rounded-full bg-primary animate-pulse"
                ></div>
                <span class="text-xs font-mono font-bold text-base-content/70"
                    >{nowFormatted}</span
                >
            </div>
        {/if}
    </div>

    <!-- Agenda Content -->
    <div
        class="flex-1 overflow-y-auto custom-scrollbar p-6 relative bg-base-50/30"
    >
        <!-- Spine Line -->
        <div
            class="absolute left-[39px] top-6 bottom-0 w-px bg-gradient-to-b from-transparent via-base-300/60 to-transparent z-0"
        ></div>

        {#if activeBlocks.length === 0}
            <div
                class="flex items-center justify-center h-full p-8"
                in:fade={{ duration: 400 }}
            >
                <div
                    class="flex flex-col items-center justify-center text-center p-8 border border-dashed border-base-300 rounded-3xl bg-base-100/40 backdrop-blur-sm max-w-sm"
                >
                    <div class="bg-base-200/50 p-4 rounded-full mb-4">
                        <Calendar class="w-8 h-8 text-base-content/20" />
                    </div>
                    <h3 class="font-bold text-base-content/60">
                        No tasks for {dateLabel()}
                    </h3>
                    <p class="text-xs text-base-content/40 mt-1">
                        Enjoy your free time or add new tasks to get started.
                    </p>
                </div>
            </div>
        {:else}
            {#each activeBlocks as block, i}
                {@const blockTasks = tasksByBlock.get(block) || []}
                {@const BlockIcon = getTimeBlockIcon(block)}
                {@const isCurrentBlock = isToday() && block === currentBlock}

                <div
                    class="relative z-10 mb-8 last:mb-2"
                    in:slide={{ duration: 400, delay: i * 50 }}
                >
                    <!-- Block Header -->
                    <div class="flex items-center gap-4 mb-4">
                        <div
                            class="w-8 h-8 rounded-full flex items-center justify-center shadow-sm z-20 border-[3px] border-base-100 transition-colors
                             {isCurrentBlock
                                ? 'bg-primary text-primary-content ring-2 ring-primary/20'
                                : 'bg-base-200 text-base-content/60'}"
                        >
                            <BlockIcon class="w-4 h-4" />
                        </div>
                        <div class="flex items-baseline gap-3">
                            <h3
                                class="text-sm font-bold tracking-wide uppercase {isCurrentBlock
                                    ? 'text-primary'
                                    : 'text-base-content/70'}"
                            >
                                {getTimeBlockLabel(block)}
                            </h3>
                            <span
                                class="text-xs text-base-content/40 font-medium"
                            >
                                {blockTasks.length}
                                {blockTasks.length === 1 ? "Task" : "Tasks"}
                            </span>
                        </div>
                    </div>

                    <!-- Tasks List -->
                    <div class="pl-[48px] space-y-3">
                        {#each blockTasks as task, j (task.id)}
                            {@const status = getTaskStatus(task)}
                            {@const bg = task.categoryColor || "#6b7280"}
                            {@const isCompleted = task.completed}

                            <!-- svelte-ignore a11y_click_events_have_key_events -->
                            <!-- svelte-ignore a11y_interactive_supports_focus -->
                            <div
                                class="group relative flex flex-col sm:flex-row items-stretch gap-0 rounded-xl border transition-all duration-300 w-full overflow-hidden bg-base-100
                                    {isCompleted
                                    ? 'opacity-80 border-base-200 hover:opacity-100'
                                    : status === 'current'
                                      ? 'shadow-lg ring-1 ring-primary/20 border-primary/30 -translate-x-1'
                                      : 'shadow-sm border-base-200/80 hover:shadow-md hover:border-base-300 hover:scale-[1.01]'}"
                                role="button"
                                onclick={() => onTaskClick?.(task.id)}
                                in:slide={{
                                    duration: 300,
                                    delay: i * 30 + j * 50,
                                }}
                            >
                                <!-- Left Color Indicator -->
                                <div
                                    class="w-1.5 shrink-0 transition-colors"
                                    class:opacity-50={isCompleted}
                                    style="background-color: {bg};"
                                ></div>

                                <!-- Main Card Content -->
                                <div
                                    class="flex-1 flex items-center p-3 sm:px-4 sm:py-3.5 relative"
                                >
                                    <!-- Background Pattern for Completed -->
                                    {#if isCompleted}
                                        <div
                                            class="absolute inset-0 opacity-[0.03] pointer-events-none"
                                            style="background-image: repeating-linear-gradient(45deg, #000 0, #000 1px, transparent 0, transparent 50%); background-size: 8px 8px;"
                                        ></div>
                                    {/if}

                                    <!-- Time column -->
                                    <div
                                        class="flex flex-col w-[70px] shrink-0 mr-4 text-right justify-center border-r border-base-200/50 pr-4"
                                    >
                                        <span
                                            class="text-sm font-bold font-mono tracking-tight text-base-content/90"
                                        >
                                            {formatTime(task.startTime)}
                                        </span>
                                        <span
                                            class="text-[10px] text-base-content/40 font-medium"
                                        >
                                            {formatDuration(
                                                task.startTime,
                                                task.endTime,
                                            )}
                                        </span>
                                    </div>

                                    <!-- Task Info -->
                                    <div
                                        class="flex-1 min-w-0 flex flex-col justify-center py-0.5"
                                    >
                                        <h4
                                            class="text-sm font-semibold truncate leading-tight transition-colors
                                                {isCompleted
                                                ? 'text-base-content/60 line-through decoration-base-content/20'
                                                : 'text-base-content'}"
                                        >
                                            {task.title}
                                        </h4>
                                        {#if task.categoryName}
                                            <div
                                                class="flex items-center gap-2 mt-1.5"
                                            >
                                                <span
                                                    class="text-[10px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-md bg-base-200/50"
                                                    style="color: {bg}; background-color: {bg}15;"
                                                >
                                                    {task.categoryName}
                                                </span>
                                            </div>
                                        {/if}
                                    </div>

                                    <!-- Status / Action -->
                                    <div class="flex items-center pl-3">
                                        {#if isCompleted}
                                            <div
                                                class="badge badge-sm badge-ghost gap-1 opacity-70 font-medium bg-base-200/80"
                                            >
                                                <Check class="w-3 h-3" /> Done
                                            </div>
                                        {:else if status === "current"}
                                            <div
                                                class="flex items-center gap-1.5 text-primary text-xs font-bold animate-pulse"
                                            >
                                                <div
                                                    class="w-1.5 h-1.5 rounded-full bg-primary"
                                                ></div>
                                                Now
                                            </div>
                                        {:else}
                                            <div
                                                class="opacity-0 group-hover:opacity-100 transition-opacity"
                                            >
                                                <ChevronRight
                                                    class="w-4 h-4 text-base-content/30"
                                                />
                                            </div>
                                        {/if}
                                    </div>
                                </div>
                            </div>
                        {/each}
                    </div>
                </div>
            {/each}
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
        background: oklch(var(--bc) / 0.1);
        border-radius: 10px;
    }
    .custom-scrollbar::-webkit-scrollbar-thumb:hover {
        background: oklch(var(--bc) / 0.2);
    }
    .custom-scrollbar::-webkit-scrollbar-track {
        background: transparent;
    }
</style>
