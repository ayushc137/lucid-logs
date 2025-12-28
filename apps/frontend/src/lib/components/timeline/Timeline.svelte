<script lang="ts">
    import { Clock, Check, Circle, Plus, CheckCircle2 } from "lucide-svelte";
    import { onMount, onDestroy } from "svelte";
    import { fade } from "svelte/transition";

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
        onTaskClick?: (taskId: string) => void;
        onToggleComplete?: (taskId: string, completed: boolean) => void;
        onCategoryClick?: (categoryId: string) => void;
    }

    let {
        tasks = [],
        startHour = 0,
        endHour = 24,
        onTaskClick,
        onToggleComplete,
        onCategoryClick,
    }: Props = $props();

    const UNCATEGORIZED_COLOR = "#9ca3af"; // Gray-400
    const ROW_HEIGHT = 56;
    const ROW_GAP = 8;
    const GROUP_PADDING = 16; // Top/bottom padding for each category group

    // Current time state
    let currentTime = $state(new Date());
    let timeInterval: ReturnType<typeof setInterval>;

    // Hover state
    let hoveredTaskId = $state<string | null>(null);
    let tooltipPosition = $state<{ x: number; y: number } | null>(null);

    onMount(() => {
        timeInterval = setInterval(() => {
            currentTime = new Date();
        }, 1000);
    });

    onDestroy(() => {
        if (timeInterval) clearInterval(timeInterval);
    });

    // --- Helpers ---

    const allHours = $derived(
        Array.from(
            { length: endHour - startHour + 1 },
            (_, i) => startHour + i,
        ),
    );

    function formatHour(hour: number): string {
        if (hour === 0 || hour === 24) return "12 AM";
        if (hour === 12) return "12 PM";
        if (hour < 12) return `${hour} AM`;
        return `${hour - 12} PM`;
    }

    function formatExactTime(date: Date): string {
        if (!isValidDate(date)) return "--:--";
        return date.toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
            hour12: true,
        });
    }

    function isValidDate(date: any): boolean {
        return date instanceof Date && !isNaN(date.getTime());
    }

    function getDecimalHour(date: Date): number {
        if (!isValidDate(date)) return startHour;
        return date.getHours() + date.getMinutes() / 60;
    }

    // --- Layout Logic ---

    // Calculate position/width for a single task
    function getTaskStyle(task: Task): { left: number; width: number } {
        if (!isValidDate(task.startTime) || !isValidDate(task.endTime)) {
            return { left: 0, width: 0 };
        }

        const startDecimal = getDecimalHour(task.startTime);
        let endDecimal = getDecimalHour(task.endTime);
        const totalHours = endHour - startHour;

        if (endDecimal < startDecimal) endDecimal += 24; // Crosses midnight
        // Cap visual end for very long tasks to keep them sane in day view
        if (endDecimal - startDecimal > 24) endDecimal = startDecimal + 24;

        const leftPercent = ((startDecimal - startHour) / totalHours) * 100;
        const widthPercent = ((endDecimal - startDecimal) / totalHours) * 100;

        return {
            left: Math.max(0, leftPercent),
            width: Math.max(1, Math.min(widthPercent, 100 - leftPercent)), // Clamp to container
        };
    }

    // Packing algorithm: Assigns a 'row' index to each task in a list to prevent overlap
    function packTasks(taskList: Task[]) {
        if (taskList.length === 0)
            return { rows: [], height: ROW_HEIGHT + GROUP_PADDING * 2 };

        const validTasks = taskList.filter(
            (t) => isValidDate(t.startTime) && isValidDate(t.endTime),
        );
        const sorted = [...validTasks].sort(
            (a, b) => a.startTime.getTime() - b.startTime.getTime(),
        );

        const packedRows: Array<{ task: Task; row: number }> = [];
        const lanes: number[] = []; // stores 'endTime' timestamp of the last task in each lane

        const gapBufferMs = 15 * 60 * 1000; // 15 min buffer prevents tight visual squeezing

        for (const task of sorted) {
            const startMs = task.startTime.getTime();
            const endMs = task.endTime.getTime();
            let effectiveEndMs = endMs;

            // Enforce minimum visual width rules impact on stacking
            // If task is super short (e.g. 5 mins), visually it might be 30px wide
            // So we treat it as effectively longer for collision detection
            const minVisualDurationMs = 30 * 60 * 1000;
            if (effectiveEndMs - startMs < minVisualDurationMs) {
                effectiveEndMs = startMs + minVisualDurationMs;
            }

            // Find first lane where this task fits
            let laneIndex = lanes.findIndex(
                (laneEnd) => laneEnd + gapBufferMs <= startMs,
            );

            if (laneIndex === -1) {
                // New lane
                laneIndex = lanes.length;
                lanes.push(effectiveEndMs);
            } else {
                // Update existing lane
                lanes[laneIndex] = effectiveEndMs;
            }

            packedRows.push({ task, row: laneIndex });
        }

        const maxLane = lanes.length > 0 ? lanes.length : 1;
        const totalHeight =
            maxLane * ROW_HEIGHT + (maxLane - 1) * ROW_GAP + GROUP_PADDING * 2;

        return { rows: packedRows, height: totalHeight };
    }

    // Group tasks by category and calculate layout for each group
    const categoryGroups = $derived.by(() => {
        const groups = new Map<
            string,
            { id: string; name: string; color: string; tasks: Task[] }
        >();
        const uncategorizedTasks: Task[] = [];

        for (const task of tasks) {
            if (task.categoryId && task.categoryName) {
                if (!groups.has(task.categoryId)) {
                    groups.set(task.categoryId, {
                        id: task.categoryId,
                        name: task.categoryName,
                        color: task.categoryColor || UNCATEGORIZED_COLOR,
                        tasks: [],
                    });
                }
                groups.get(task.categoryId)!.tasks.push(task);
            } else {
                uncategorizedTasks.push(task);
            }
        }

        // Process categorized groups
        const result = Array.from(groups.values()).map((group) => {
            const { rows, height } = packTasks(group.tasks);
            return { ...group, rows, height };
        });

        // Add uncategorized if any
        if (uncategorizedTasks.length > 0) {
            const { rows, height } = packTasks(uncategorizedTasks);
            result.push({
                id: "uncategorized",
                name: "Uncategorized",
                color: UNCATEGORIZED_COLOR,
                tasks: uncategorizedTasks,
                rows,
                height,
            });
        }

        // Sort by name for consistency
        result.sort((a, b) => a.name.localeCompare(b.name));

        return result;
    });

    // --- Styling Helpers ---

    function getContrastTextColor(hexColor: string): string {
        if (!hexColor || !hexColor.startsWith("#")) return "#ffffff";
        const hex = hexColor.replace("#", "");
        if (hex.length !== 6) return "#ffffff";
        const r = parseInt(hex.substring(0, 2), 16);
        const g = parseInt(hex.substring(2, 4), 16);
        const b = parseInt(hex.substring(4, 6), 16);
        // Custom logic: darker threshold for better readability on pastels
        const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
        return luminance > 0.6 ? "#1f2937" : "#ffffff";
    }

    function getShadowColor(hexColor: string): string {
        if (!hexColor || !hexColor.startsWith("#"))
            return "rgba(107, 114, 128, 0.25)";
        return `${hexColor}40`;
    }

    // --- Current Time Indicator ---

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

    // --- Interaction ---

    function handleMouseEnter(e: MouseEvent, taskId: string) {
        hoveredTaskId = taskId;
        updateTooltipPosition(e);
    }
    function handleMouseMove(e: MouseEvent) {
        if (hoveredTaskId) updateTooltipPosition(e);
    }
    function handleMouseLeave() {
        hoveredTaskId = null;
    }
    function updateTooltipPosition(e: MouseEvent) {
        tooltipPosition = { x: e.clientX + 16, y: e.clientY + 16 };
    }
    const hoveredTask = $derived(tasks.find((t) => t.id === hoveredTaskId));
</script>

<div
    class="timeline-wrapper h-full flex flex-col relative bg-base-100/50 rounded-xl border border-base-200 shadow-inner"
>
    <!-- CHANGED: overflow-x-hidden to force hide horizontal scroll -->
    <div
        class="timeline-container flex-1 overflow-x-hidden overflow-y-auto rounded-xl"
    >
        <!-- CHANGED: w-full instead of min-w-full to avoid scrollbar conflict -->
        <div class="timeline-inner w-full h-full flex flex-col">
            <!-- Header: Time Labels -->
            <div
                class="sticky top-0 z-30 bg-base-100/95 backdrop-blur-sm border-b border-base-200"
            >
                <div class="h-10 relative">
                    {#each allHours as hour}
                        {@const leftPos =
                            ((hour - startHour) / (endHour - startHour)) * 100}
                        <div
                            class="absolute top-1/2 -translate-y-1/2 text-[10px] font-bold text-base-content/40 uppercase tracking-widest transform -translate-x-1/2 whitespace-nowrap"
                            style="left: {leftPos}%;"
                        >
                            {formatHour(hour)}
                        </div>
                    {/each}
                </div>
            </div>

            <!-- Body: Swimlanes -->
            <div class="relative flex-1 min-h-[300px]">
                <!-- Background Grid Lines (Absolute) -->
                <div class="absolute inset-0 pointer-events-none">
                    {#each allHours as hour}
                        {@const leftPos =
                            ((hour - startHour) / (endHour - startHour)) * 100}
                        <div
                            class="absolute top-0 bottom-0 w-px bg-base-content/5 border-r border-dashed border-base-content/5"
                            style="left: {leftPos}%;"
                        ></div>
                    {/each}
                </div>

                <!-- Current Time Indicator -->
                {#if showNowLine}
                    <div
                        class="absolute top-0 bottom-0 z-20 pointer-events-none"
                        style="left: {nowPosition}%;"
                    >
                        <div
                            class="absolute top-0 bottom-0 left-0 w-px bg-error shadow-[0_0_8px_rgba(255,0,0,0.6)]"
                        ></div>
                        <div
                            class="absolute -top-1 left-0 -translate-x-1/2 w-2 h-2 rounded-full bg-error animate-pulse"
                        ></div>
                    </div>
                {/if}

                <!-- Swimlanes (Groups) -->
                <div class="flex flex-col">
                    {#each categoryGroups as group, i}
                        <div
                            class="group-track relative border-b border-base-content/5 hover:bg-base-content/[0.02] transition-colors duration-300"
                            style="height: {group.height}px;"
                        >
                            <!-- Sticky Category Header (Interactive) -->
                            <div
                                class="sticky left-0 z-20 inline-flex items-start pt-4 pl-4 pr-6 w-48 pointer-events-none"
                            >
                                <button
                                    class="bg-base-100/90 backdrop-blur-md px-3 py-1.5 rounded-full border border-base-content/10 flex items-center gap-2 shadow-sm transition-all duration-200 pointer-events-auto group/label {group.id !==
                                    'uncategorized'
                                        ? 'cursor-pointer hover:scale-105 active:scale-95 hover:shadow-md hover:ring-1 ring-primary/20'
                                        : ''}"
                                    onclick={() =>
                                        group.id !== "uncategorized" &&
                                        onCategoryClick?.(group.id)}
                                    title={group.id !== "uncategorized"
                                        ? "Click to add task"
                                        : ""}
                                >
                                    <div
                                        class="w-2.5 h-2.5 rounded-full shrink-0 shadow-sm transition-transform group-hover/label:scale-125"
                                        style="background-color: {group.color};"
                                    ></div>
                                    <span
                                        class="text-xs font-bold truncate max-w-[100px]"
                                        >{group.name}</span
                                    >
                                    {#if group.id !== "uncategorized"}
                                        <Plus
                                            class="w-3 h-3 opacity-0 group-hover/label:opacity-100 -mr-1 transition-opacity text-base-content/50"
                                        />
                                    {/if}
                                </button>
                            </div>

                            <!-- Tasks Grid -->
                            <div class="absolute inset-0 top-4">
                                <!-- Offset matching top padding -->
                                {#each group.rows as { task, row }}
                                    {@const style = getTaskStyle(task)}
                                    {@const textColor = getContrastTextColor(
                                        group.color,
                                    )}
                                    {@const shadowColor = getShadowColor(
                                        group.color,
                                    )}
                                    {@const isFiltered =
                                        hoveredTaskId &&
                                        hoveredTaskId !== task.id}

                                    <!-- svelte-ignore a11y_interactive_supports_focus -->
                                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                                    <div
                                        class="absolute transition-all duration-300 ease-out"
                                        class:z-40={hoveredTaskId === task.id}
                                        class:z-10={hoveredTaskId !== task.id}
                                        class:opacity-30={isFiltered}
                                        class:scale-[1.02]={hoveredTaskId ===
                                            task.id}
                                        style="
                                            left: {style.left}%;
                                            width: {style.width}%;
                                            top: {row *
                                            (ROW_HEIGHT + ROW_GAP)}px;
                                            height: {ROW_HEIGHT}px;
                                        "
                                        role="button"
                                        onmouseenter={(e) =>
                                            handleMouseEnter(e, task.id)}
                                        onmousemove={handleMouseMove}
                                        onmouseleave={handleMouseLeave}
                                        onclick={(e) => onTaskClick?.(task.id)}
                                    >
                                        <div
                                            class="absolute inset-0 rounded-xl overflow-hidden transition-all duration-300 ring-1 ring-black/5 dark:ring-white/10"
                                            class:shadow-sm={!task.completed}
                                            class:hover:shadow-lg={!task.completed}
                                            class:shadow-inner={task.completed}
                                            class:opacity-90={task.completed}
                                            class:saturate-[0.75]={task.completed}
                                            style="background-color: {group.color}; {task.completed
                                                ? ''
                                                : `box-shadow: 0 4px 6px -2px ${shadowColor};`}"
                                        >
                                            <!-- Completed Texture Overlay -->
                                            {#if task.completed}
                                                <div
                                                    class="absolute inset-0 bg-striped opacity-30 pointer-events-none z-10"
                                                ></div>
                                                <div
                                                    class="absolute inset-0 bg-black/10 pointer-events-none z-0"
                                                ></div>
                                            {/if}

                                            <div
                                                class="relative z-20 h-full px-3 flex items-center justify-center gap-2"
                                                style="color: {textColor};"
                                            >
                                                <div
                                                    class="min-w-0 flex-1 flex flex-col justify-center"
                                                >
                                                    <span
                                                        class="font-bold text-xs truncate leading-snug"
                                                        class:opacity-60={task.completed}
                                                    >
                                                        {task.title}
                                                    </span>
                                                    <span
                                                        class="text-[10px] opacity-80 font-medium font-mono"
                                                    >
                                                        {formatExactTime(
                                                            task.startTime,
                                                        )}
                                                    </span>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/each}

                    <!-- Empty State -->
                    {#if categoryGroups.length === 0}
                        <div
                            class="h-64 flex flex-col items-center justify-center opacity-50"
                        >
                            <Clock class="w-10 h-10 mb-3" />
                            <p class="text-sm font-medium">
                                No tasks scheduled for today
                            </p>
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    </div>

    <!-- Tooltip (Portal) -->
    {#if hoveredTask && tooltipPosition}
        <div
            class="fixed z-[100] pointer-events-none animate-in fade-in zoom-in-95 duration-200"
            style="top: {tooltipPosition.y}px; left: {tooltipPosition.x}px;"
        >
            <div
                class="bg-base-100/95 backdrop-blur-md text-base-content rounded-xl shadow-2xl border border-base-200/50 p-4 min-w-[220px] max-w-xs"
            >
                <div class="flex items-start gap-3">
                    {#if hoveredTask.emoji}
                        <span class="text-2xl pt-1">{hoveredTask.emoji}</span>
                    {/if}
                    <div>
                        <h4 class="font-bold text-sm leading-snug">
                            {hoveredTask.title}
                        </h4>
                        {#if hoveredTask.categoryName}
                            <div class="flex items-center gap-1.5 mt-1.5">
                                <span
                                    class="w-2 h-2 rounded-full"
                                    style="background-color: {hoveredTask.categoryColor}"
                                ></span>
                                <span
                                    class="text-[10px] font-semibold opacity-60 uppercase"
                                    >{hoveredTask.categoryName}</span
                                >
                            </div>
                        {/if}
                    </div>
                </div>
                <div class="h-px bg-base-content/10 my-3"></div>
                <div
                    class="flex items-center gap-2 text-xs font-mono opacity-70"
                >
                    <Clock class="w-3.5 h-3.5" />
                    <span>
                        {formatExactTime(hoveredTask.startTime)} — {formatExactTime(
                            hoveredTask.endTime,
                        )}
                    </span>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    /* Sleek Scrollbar */
    .timeline-container {
        scrollbar-width: thin;
        scrollbar-color: oklch(var(--bc) / 0.1) transparent;
    }
    .timeline-container::-webkit-scrollbar {
        height: 6px;
    }
    .timeline-container::-webkit-scrollbar-thumb {
        background: oklch(var(--bc) / 0.1);
        border-radius: 10px;
    }
    .timeline-container::-webkit-scrollbar-track {
        background: transparent;
    }

    /* Striped Pattern for Completed Tasks */
    .bg-striped {
        background-image: linear-gradient(
            45deg,
            rgba(0, 0, 0, 0.1) 25%,
            transparent 25%,
            transparent 50%,
            rgba(0, 0, 0, 0.1) 50%,
            rgba(0, 0, 0, 0.1) 75%,
            transparent 75%,
            transparent
        );
        background-size: 8px 8px;
    }
</style>
