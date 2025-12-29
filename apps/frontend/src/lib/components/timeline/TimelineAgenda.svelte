<script lang="ts">
    import {
        Clock,
        ChevronLeft,
        ChevronRight,
        Calendar,
        Sunrise,
        Sun,
        Sunset,
        Moon,
        Coffee,
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
    class="h-full flex flex-col bg-base-100/50 rounded-xl border border-base-300/40 overflow-hidden"
>
    <!-- Header -->
    <div
        class="flex items-center justify-between px-4 py-3 bg-base-100 border-b border-base-200/50 backdrop-blur-sm sticky top-0 z-40"
    >
        <div class="flex items-center gap-2">
            <div class="join shadow-sm border border-base-200/60 rounded-lg">
                <button
                    class="join-item btn btn-sm btn-ghost hover:bg-base-200 px-2"
                    onclick={goToPreviousDay}
                    aria-label="Previous Day"
                >
                    <ChevronLeft class="w-4 h-4" />
                </button>
                <button
                    class="join-item btn btn-sm btn-ghost hover:bg-base-200 min-w-[140px] font-semibold text-base-content"
                    class:text-primary={isToday()}
                    onclick={goToToday}
                >
                    {dateLabel()}
                </button>
                <button
                    class="join-item btn btn-sm btn-ghost hover:bg-base-200 px-2"
                    onclick={goToNextDay}
                    aria-label="Next Day"
                >
                    <ChevronRight class="w-4 h-4" />
                </button>
            </div>
            {#if !isToday()}
                <button
                    class="btn btn-xs btn-ghost text-xs font-medium text-primary ml-1"
                    onclick={goToToday}
                    in:fade={{ duration: 200 }}
                >
                    Return to Today
                </button>
            {/if}
        </div>

        {#if isToday()}
            <div
                class="flex items-center gap-2 px-3 py-1 bg-primary/5 border border-primary/10 rounded-full"
                in:fade
            >
                <Clock class="w-3.5 h-3.5 text-primary" />
                <span class="text-xs font-mono font-bold text-primary"
                    >{nowFormatted}</span
                >
            </div>
        {/if}
    </div>

    <!-- Agenda Content -->
    <div class="flex-1 overflow-y-auto custom-scrollbar p-4 relative">
        <!-- Spine Line -->
        <div
            class="absolute left-[35px] top-6 bottom-0 w-px bg-base-200/60 z-0"
        ></div>

        {#if activeBlocks.length === 0}
            <div
                class="flex items-center justify-center h-full"
                in:fade={{ duration: 400 }}
            >
                <div class="text-center py-12 px-6 opacity-60">
                    <div
                        class="bg-base-200/50 p-4 rounded-full mb-4 inline-block"
                    >
                        <Calendar class="w-8 h-8 text-base-content/30" />
                    </div>
                    <h3 class="text-sm font-semibold text-base-content/60">
                        No tasks for {dateLabel()}
                    </h3>
                    <p class="text-xs text-base-content/40 mt-1">
                        Use the "New Task" button to plan your day.
                    </p>
                </div>
            </div>
        {:else}
            {#each activeBlocks as block, i}
                {@const blockTasks = tasksByBlock.get(block) || []}
                {@const BlockIcon = getTimeBlockIcon(block)}
                {@const isCurrentBlock = isToday() && block === currentBlock}

                <div
                    class="relative z-10 mb-5 last:mb-0"
                    in:slide={{ duration: 400, delay: i * 50 }}
                >
                    <!-- Block Header -->
                    <div class="flex items-center gap-4 mb-2">
                        <div
                            class="w-8 h-8 rounded-full flex items-center justify-center shadow-sm z-20 border-4 border-base-100
                             {isCurrentBlock
                                ? 'bg-primary text-primary-content ring-2 ring-primary/20'
                                : 'bg-base-200 text-base-content/50'}"
                        >
                            <BlockIcon class="w-4 h-4" />
                        </div>
                        <div>
                            <h3
                                class="text-sm font-bold tracking-wide uppercase {isCurrentBlock
                                    ? 'text-primary'
                                    : 'text-base-content/60'}"
                            >
                                {getTimeBlockLabel(block)}
                            </h3>
                            <p
                                class="text-[10px] text-base-content/40 font-medium"
                            >
                                {blockTasks.length}
                                {blockTasks.length === 1 ? "Task" : "Tasks"}
                            </p>
                        </div>
                    </div>

                    <!-- Tasks List -->
                    <div class="pl-[44px] space-y-1.5">
                        {#each blockTasks as task, j (task.id)}
                            {@const status = getTaskStatus(task)}
                            {@const bg = task.categoryColor || "#6b7280"}

                            <!-- svelte-ignore a11y_click_events_have_key_events -->
                            <!-- svelte-ignore a11y_interactive_supports_focus -->
                            <div
                                class="group relative flex items-stretch gap-0 rounded-xl border transition-all duration-200 hover:scale-[1.01] hover:shadow-lg overflow-hidden w-full
                                    bg-base-100 shadow-sm
                                    {status === 'current'
                                    ? 'border-primary/40 ring-1 ring-primary/10 shadow-md'
                                    : 'border-base-200/60 hover:border-base-300'}"
                                role="button"
                                onclick={() => onTaskClick?.(task.id)}
                                in:slide={{
                                    duration: 300,
                                    delay: i * 30 + j * 50,
                                }}
                            >
                                <!-- Left Color Strip -->
                                <div
                                    class="w-1.5 shrink-0"
                                    style="background-color: {bg};"
                                ></div>

                                <!-- Main Content -->
                                <div
                                    class="flex-1 flex items-center p-3 relative"
                                >
                                    <!-- Background Pattern for Completed -->
                                    {#if task.completed}
                                        <div
                                            class="absolute inset-0 opacity-[0.03] pointer-events-none"
                                            style="background-image: repeating-linear-gradient(45deg, #000 0, #000 1px, transparent 0, transparent 50%); background-size: 10px 10px;"
                                        ></div>
                                    {/if}

                                    <!-- Time & Status -->
                                    <div
                                        class="flex flex-col min-w-[70px] mr-3 text-right justify-center"
                                    >
                                        <span
                                            class="text-sm font-bold font-mono tracking-tight {status ===
                                            'current'
                                                ? 'text-primary'
                                                : 'text-base-content/80'}"
                                        >
                                            {formatTime(task.startTime)}
                                        </span>
                                        <span
                                            class="text-[10px] text-base-content/40 font-medium mt-0.5"
                                        >
                                            {formatDuration(
                                                task.startTime,
                                                task.endTime,
                                            )}
                                        </span>
                                    </div>

                                    <!-- Title & Category -->
                                    <div
                                        class="flex-1 min-w-0 flex flex-col justify-center border-l border-base-100 pl-3 py-0.5"
                                    >
                                        <h4
                                            class="text-sm font-bold truncate leading-snug {task.completed
                                                ? 'opacity-50'
                                                : 'text-base-content'}"
                                        >
                                            {task.title}
                                        </h4>
                                        {#if task.categoryName}
                                            <div
                                                class="flex items-center gap-2 mt-1"
                                            >
                                                <span
                                                    class="text-[10px] font-medium uppercase tracking-wider opacity-60"
                                                    style="color: {bg}"
                                                >
                                                    {task.categoryName}
                                                </span>
                                            </div>
                                        {/if}
                                    </div>

                                    <!-- Right Status Indicator -->
                                    <div class="flex items-center pl-3">
                                        {#if task.completed}
                                            <div
                                                class="badge badge-sm badge-ghost gap-1 opacity-70"
                                            >
                                                Done
                                            </div>
                                        {:else if status === "current"}
                                            <div
                                                class="badge badge-sm badge-primary badge-outline gap-1 animate-pulse"
                                            >
                                                Now
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
        width: 6px;
    }
    .custom-scrollbar::-webkit-scrollbar-track {
        background: transparent;
    }
    .custom-scrollbar::-webkit-scrollbar-thumb {
        background-color: rgb(0 0 0 / 0.1);
        border-radius: 20px;
    }
    .custom-scrollbar::-webkit-scrollbar-thumb:hover {
        background-color: rgb(0 0 0 / 0.2);
    }
</style>
