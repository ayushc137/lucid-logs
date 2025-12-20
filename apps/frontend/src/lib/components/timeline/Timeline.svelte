<script lang="ts">
    import { Clock } from "lucide-svelte";
    import { onMount, onDestroy } from "svelte";

    interface Task {
        id: string;
        title: string;
        startTime: Date;
        endTime: Date;
        categoryColor?: string;
        categoryName?: string;
        completed?: boolean;
        emoji?: string;
        categoryId?: string;
    }

    interface Props {
        tasks?: Task[];
        startHour?: number;
        endHour?: number;
    }

    let { tasks = [], startHour = 0, endHour = 24 }: Props = $props();

    // Default color for uncategorized tasks
    const DEFAULT_TASK_COLOR = "#525252";

    // Current time state - updates every second
    let currentTime = $state(new Date());
    let timeInterval: ReturnType<typeof setInterval>;

    onMount(() => {
        timeInterval = setInterval(() => {
            currentTime = new Date();
        }, 1000);
    });

    onDestroy(() => {
        if (timeInterval) clearInterval(timeInterval);
    });

    // Generate all hours for labels
    const allHours = $derived(
        Array.from(
            { length: endHour - startHour + 1 },
            (_, i) => startHour + i,
        ),
    );

    // Format hour display - full format
    function formatHour(hour: number): string {
        if (hour === 0 || hour === 24) return "12 AM";
        if (hour === 12) return "12 PM";
        if (hour < 12) return `${hour} AM`;
        return `${hour - 12} PM`;
    }

    // Format exact time
    function formatExactTime(date: Date): string {
        return date.toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
            second: "2-digit",
            hour12: true,
        });
    }

    // Get time in hours (decimal) from Date
    function getDecimalHour(date: Date): number {
        return date.getHours() + date.getMinutes() / 60;
    }

    // Get task position and width based on time (horizontal layout)
    function getTaskStyle(task: Task): { left: number; width: number } {
        const startDecimal = getDecimalHour(task.startTime);
        const endDecimal = getDecimalHour(task.endTime);
        const totalHours = endHour - startHour;

        const leftPercent = ((startDecimal - startHour) / totalHours) * 100;
        const widthPercent = ((endDecimal - startDecimal) / totalHours) * 100;

        return {
            left: Math.max(0, leftPercent),
            width: Math.max(3, widthPercent),
        };
    }

    // Get task color
    function getTaskColor(task: Task): string {
        return task.categoryColor || DEFAULT_TASK_COLOR;
    }

    // Calculate current time position
    const currentHour = $derived(
        currentTime.getHours() +
            currentTime.getMinutes() / 60 +
            currentTime.getSeconds() / 3600,
    );
    const nowPosition = $derived(
        ((currentHour - startHour) / (endHour - startHour)) * 100,
    );
    const showNowLine = $derived(
        currentHour >= startHour && currentHour <= endHour,
    );

    // Group overlapping tasks into rows
    function getTaskRows(tasks: Task[]): Array<{ task: Task; row: number }> {
        if (tasks.length === 0) return [];

        const sorted = [...tasks].sort(
            (a, b) => a.startTime.getTime() - b.startTime.getTime(),
        );
        const result: Array<{ task: Task; row: number }> = [];
        const rows: Array<{ endTime: Date }> = [];

        for (const task of sorted) {
            let rowIndex = rows.findIndex(
                (row) => row.endTime <= task.startTime,
            );

            if (rowIndex === -1) {
                rowIndex = rows.length;
                rows.push({ endTime: task.endTime });
            } else {
                rows[rowIndex].endTime = task.endTime;
            }

            result.push({ task, row: rowIndex });
        }

        return result;
    }

    const taskRows = $derived(getTaskRows(tasks));
    const maxRows = $derived(
        taskRows.length > 0 ? Math.max(...taskRows.map((t) => t.row)) + 1 : 1,
    );
</script>

<div class="timeline-wrapper h-full flex flex-col">
    <div class="timeline-container flex-1 overflow-x-auto overflow-y-hidden">
        <div
            class="timeline-inner min-w-[1800px] h-full flex flex-col pl-6 pr-6"
        >
            <!-- Hour labels row -->
            <div
                class="hour-labels flex-shrink-0 h-6 relative border-b border-base-300/50"
            >
                {#each allHours as hour}
                    {@const leftPos =
                        ((hour - startHour) / (endHour - startHour)) * 100}
                    <div
                        class="hour-label absolute text-[10px] font-medium opacity-50 transform -translate-x-1/2 whitespace-nowrap"
                        style="left: {leftPos}%;"
                    >
                        {formatHour(hour)}
                    </div>
                {/each}
            </div>

            <!-- Timeline grid area -->
            <div
                class="timeline-grid flex-1 relative bg-gradient-to-b from-base-200/20 to-base-200/40 rounded-xl overflow-visible"
            >
                <!-- Hour lines (vertical) -->
                {#each allHours as hour}
                    {@const leftPos =
                        ((hour - startHour) / (endHour - startHour)) * 100}
                    <div
                        class="absolute top-0 bottom-0 w-px bg-base-300/40"
                        style="left: {leftPos}%;"
                    ></div>
                {/each}

                <!-- Current time indicator - Improved design -->
                {#if showNowLine}
                    <div
                        class="now-indicator absolute z-30"
                        style="left: {nowPosition}%;"
                    >
                        <!-- Gradient glow behind line -->
                        <div
                            class="absolute -inset-x-4 top-0 bottom-0 bg-gradient-to-r from-transparent via-error/10 to-transparent pointer-events-none"
                        ></div>

                        <!-- Main vertical line -->
                        <div
                            class="absolute top-0 bottom-0 left-1/2 w-0.5 -translate-x-1/2 bg-gradient-to-b from-error via-error to-error/50"
                        ></div>

                        <!-- Top badge with time -->
                        <div
                            class="absolute -top-1 left-1/2 -translate-x-1/2 flex flex-col items-center pointer-events-none"
                        >
                            <!-- Pulsing dot -->
                            <div class="relative">
                                <div
                                    class="w-3 h-3 rounded-full bg-error"
                                ></div>
                                <div
                                    class="absolute inset-0 w-3 h-3 rounded-full bg-error animate-ping opacity-75"
                                ></div>
                            </div>
                            <!-- Time badge -->
                            <div
                                class="mt-1.5 px-2.5 py-1 rounded-lg bg-error text-error-content text-[11px] font-bold tracking-wide whitespace-nowrap shadow-lg shadow-error/30"
                            >
                                {formatExactTime(currentTime)}
                            </div>
                        </div>
                    </div>
                {/if}

                <!-- Tasks area -->
                <div class="tasks-area absolute inset-x-0 top-14 bottom-3">
                    {#each taskRows as { task, row }}
                        {@const style = getTaskStyle(task)}
                        {@const color = getTaskColor(task)}
                        {@const rowHeight = 100 / Math.max(maxRows, 2)}
                        <div
                            class="task-card absolute cursor-pointer transition-all duration-200 hover:z-20 group"
                            style="
                                left: {style.left}%;
                                width: {style.width}%;
                                top: calc({row * rowHeight}% + 4px);
                                height: calc({rowHeight}% - 8px);
                                min-height: 52px;
                            "
                            role="button"
                            tabindex="0"
                        >
                            <!-- Card with modern design -->
                            <div
                                class="absolute inset-0 rounded-2xl transition-all duration-200 group-hover:scale-[1.03] group-hover:shadow-xl overflow-hidden"
                                style="background: {color};"
                            >
                                <!-- Glass overlay -->
                                <div
                                    class="absolute inset-0 bg-gradient-to-br from-white/20 via-white/5 to-black/10"
                                ></div>
                                <!-- Left accent bar -->
                                <div
                                    class="absolute left-0 top-2 bottom-2 w-1 rounded-r-full bg-white/40"
                                ></div>
                            </div>

                            <!-- Card content -->
                            <div
                                class="relative h-full px-4 py-2 flex items-center gap-3 text-white overflow-hidden"
                            >
                                {#if task.emoji}
                                    <span
                                        class="text-lg flex-shrink-0 drop-shadow"
                                        >{task.emoji}</span
                                    >
                                {/if}
                                <div class="min-w-0 flex-1">
                                    <h4
                                        class="font-semibold text-sm truncate drop-shadow-sm"
                                    >
                                        {task.title}
                                    </h4>
                                    <div
                                        class="flex items-center gap-2 text-[11px] opacity-80 mt-0.5"
                                    >
                                        {#if task.categoryName}
                                            <span class="truncate"
                                                >{task.categoryName}</span
                                            >
                                            <span class="opacity-50">•</span>
                                        {/if}
                                        <span
                                            class="whitespace-nowrap font-medium"
                                        >
                                            {task.startTime.toLocaleTimeString(
                                                [],
                                                {
                                                    hour: "numeric",
                                                    minute: "2-digit",
                                                },
                                            )}
                                        </span>
                                    </div>
                                </div>
                                {#if task.completed}
                                    <div
                                        class="w-6 h-6 rounded-full bg-white/25 flex items-center justify-center flex-shrink-0 backdrop-blur-sm"
                                    >
                                        <span class="text-xs font-bold">✓</span>
                                    </div>
                                {/if}
                            </div>

                            <!-- Hover tooltip - position based on row -->
                            <div
                                class={`tooltip-popup opacity-0 group-hover:opacity-100 absolute z-50 pointer-events-none transition-all duration-200 scale-95 group-hover:scale-100 ${row === 0 ? "tooltip-bottom" : "tooltip-top"}`}
                            >
                                <div
                                    class="bg-base-100 text-base-content rounded-2xl shadow-2xl border border-base-300 p-4 min-w-[220px]"
                                >
                                    <div class="flex items-start gap-3">
                                        {#if task.emoji}
                                            <span class="text-2xl"
                                                >{task.emoji}</span
                                            >
                                        {/if}
                                        <div class="flex-1 min-w-0">
                                            <p class="font-bold text-sm">
                                                {task.title}
                                            </p>
                                            {#if task.categoryName}
                                                <div
                                                    class="flex items-center gap-2 mt-1.5"
                                                >
                                                    <span
                                                        class="w-3 h-3 rounded-full shadow-sm"
                                                        style="background-color: {color};"
                                                    ></span>
                                                    <span
                                                        class="text-xs opacity-70"
                                                        >{task.categoryName}</span
                                                    >
                                                </div>
                                            {/if}
                                        </div>
                                    </div>
                                    <div class="divider my-2"></div>
                                    <div
                                        class="flex items-center gap-2 text-xs opacity-60"
                                    >
                                        <Clock class="w-4 h-4" />
                                        <span>
                                            {task.startTime.toLocaleTimeString(
                                                [],
                                                {
                                                    hour: "2-digit",
                                                    minute: "2-digit",
                                                },
                                            )} -
                                            {task.endTime.toLocaleTimeString(
                                                [],
                                                {
                                                    hour: "2-digit",
                                                    minute: "2-digit",
                                                },
                                            )}
                                        </span>
                                    </div>
                                    {#if task.completed}
                                        <div
                                            class="mt-2 badge badge-success badge-sm gap-1"
                                        >
                                            <span>✓</span> Completed
                                        </div>
                                    {/if}
                                </div>
                            </div>
                        </div>
                    {/each}
                </div>
            </div>
        </div>
    </div>

    <!-- Empty state -->
    {#if tasks.length === 0}
        <div
            class="absolute inset-0 flex flex-col items-center justify-center text-center pointer-events-none bg-base-100/80 backdrop-blur-sm rounded-xl"
        >
            <div
                class="w-16 h-16 rounded-2xl bg-gradient-to-br from-info/20 to-primary/20 flex items-center justify-center mb-4"
            >
                <Clock class="w-8 h-8 text-info" />
            </div>
            <h4 class="text-lg font-bold">No tasks today</h4>
            <p class="text-sm opacity-50 mt-1 max-w-xs">
                Add your first task to see it on the timeline
            </p>
        </div>
    {/if}
</div>

<style>
    .timeline-wrapper {
        position: relative;
        min-height: 180px;
    }

    .timeline-container {
        scrollbar-width: thin;
        scrollbar-color: oklch(var(--bc) / 0.2) transparent;
    }

    .timeline-container::-webkit-scrollbar {
        height: 6px;
    }

    .timeline-container::-webkit-scrollbar-track {
        background: transparent;
    }

    .timeline-container::-webkit-scrollbar-thumb {
        background: oklch(var(--bc) / 0.2);
        border-radius: 3px;
    }

    .timeline-grid {
        min-height: 140px;
    }

    .now-indicator {
        top: 0;
        bottom: 0;
    }

    .task-card {
        min-width: 120px;
    }

    /* Tooltip positioning */
    .tooltip-popup {
        filter: drop-shadow(0 8px 24px rgba(0, 0, 0, 0.2));
    }

    .tooltip-top {
        bottom: 100%;
        left: 50%;
        transform: translateX(-50%);
        margin-bottom: 12px;
    }

    .tooltip-bottom {
        top: 100%;
        left: 50%;
        transform: translateX(-50%);
        margin-top: 12px;
    }

    .group:hover .tooltip-top {
        transform: translateX(-50%) scale(1);
    }

    .group:hover .tooltip-bottom {
        transform: translateX(-50%) scale(1);
    }
</style>
