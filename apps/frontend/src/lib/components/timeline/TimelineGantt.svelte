<script lang="ts">
    import {
        Clock,
        ChevronLeft,
        ChevronRight,
        Minus,
        Plus,
        Check,
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

    // Zoom levels (hours visible)
    const ZOOM_LEVELS = [24, 18, 12, 8, 6, 4];
    let zoomIndex = $state(0);
    const hoursInView = $derived(ZOOM_LEVELS[zoomIndex]);
    const isZoomed = $derived(zoomIndex > 0);

    // Scroll offset for panning
    let scrollOffsetHours = $state(0);
    const maxScrollOffset = $derived(Math.max(0, 24 - hoursInView));
    const clampedOffset = $derived(
        Math.min(Math.max(0, scrollOffsetHours), maxScrollOffset),
    );
    const viewStartHour = $derived(clampedOffset);
    const viewEndHour = $derived(clampedOffset + hoursInView);

    let timelineRef: HTMLDivElement;
    let containerWidth = $state(800);

    function zoomIn(mouseXRatio?: number, hourAtMouse?: number) {
        if (zoomIndex < ZOOM_LEVELS.length - 1) {
            const oldHours = hoursInView;
            zoomIndex++;
            const newHours = ZOOM_LEVELS[zoomIndex];

            if (mouseXRatio !== undefined && hourAtMouse !== undefined) {
                scrollOffsetHours = Math.max(
                    0,
                    Math.min(
                        24 - newHours,
                        hourAtMouse - mouseXRatio * newHours,
                    ),
                );
            } else {
                const center = viewStartHour + oldHours / 2;
                scrollOffsetHours = Math.max(
                    0,
                    Math.min(24 - newHours, center - newHours / 2),
                );
            }
        }
    }

    function zoomOut(mouseXRatio?: number, hourAtMouse?: number) {
        if (zoomIndex > 0) {
            const oldHours = hoursInView;
            zoomIndex--;
            const newHours = ZOOM_LEVELS[zoomIndex];

            if (mouseXRatio !== undefined && hourAtMouse !== undefined) {
                scrollOffsetHours = Math.max(
                    0,
                    Math.min(
                        24 - newHours,
                        hourAtMouse - mouseXRatio * newHours,
                    ),
                );
            } else {
                const center = viewStartHour + oldHours / 2;
                scrollOffsetHours = Math.max(
                    0,
                    Math.min(24 - newHours, center - newHours / 2),
                );
            }
        }
    }

    function resetZoom() {
        zoomIndex = 0;
        scrollOffsetHours = 0;
    }

    function handleWheel(e: WheelEvent) {
        if (!e.ctrlKey && !e.metaKey) return;
        e.preventDefault();

        const rect = timelineRef.getBoundingClientRect();
        const mouseX = e.clientX - rect.left;
        const contentWidth = timelineRef.scrollWidth;
        const mouseXInContent = mouseX + timelineRef.scrollLeft;
        const mouseXRatio = mouseX / rect.width;
        const hourAtMouse = (mouseXInContent / contentWidth) * 24;

        if (e.deltaY < 0) {
            zoomIn(mouseXRatio, hourAtMouse);
        } else {
            zoomOut(mouseXRatio, hourAtMouse);
        }
    }

    function handleScroll(e: Event) {
        if (!isZoomed) return;
        const target = e.target as HTMLDivElement;
        const maxScroll = target.scrollWidth - target.clientWidth;
        if (maxScroll > 0) {
            scrollOffsetHours =
                (target.scrollLeft / maxScroll) * maxScrollOffset;
        }
    }

    $effect(() => {
        if (timelineRef && isZoomed) {
            const maxScroll = timelineRef.scrollWidth - timelineRef.clientWidth;
            if (maxScroll > 0) {
                const targetScroll =
                    (clampedOffset / maxScrollOffset) * maxScroll;
                if (Math.abs(timelineRef.scrollLeft - targetScroll) > 10) {
                    timelineRef.scrollLeft = targetScroll;
                }
            }
        }
    });

    const ROW_HEIGHT = 32;
    const ROW_GAP = 4;
    const MIN_TASK_WIDTH_PX = 6; // Minimum pixel width for very short tasks

    let currentTime = $state(new Date());
    let timeInterval: ReturnType<typeof setInterval>;

    let hoveredTaskId = $state<string | null>(null);
    let tooltipPosition = $state<{ x: number; y: number } | null>(null);

    onMount(() => {
        timeInterval = setInterval(() => (currentTime = new Date()), 1000);
        if (timelineRef) {
            containerWidth = timelineRef.clientWidth;
            const observer = new ResizeObserver(() => {
                containerWidth = timelineRef?.clientWidth ?? 800;
            });
            observer.observe(timelineRef);
            return () => observer.disconnect();
        }
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

    function formatHour(h: number): string {
        const hour = h % 24;
        if (hour === 0) return "12a";
        if (hour === 12) return "12p";
        return hour < 12 ? `${hour}a` : `${hour - 12}p`;
    }

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

    function getDecimalHour(d: Date): number {
        return d instanceof Date && !isNaN(d.getTime())
            ? d.getHours() + d.getMinutes() / 60
            : 0;
    }

    function getTaskPosition(task: TimelineTask): {
        left: number;
        width: number;
        minWidth: boolean;
    } {
        const s = task.startTime,
            e = task.endTime;
        if (!(s instanceof Date) || !(e instanceof Date))
            return { left: 0, width: 0, minWidth: false };

        let startH = getDecimalHour(s);
        let endH = getDecimalHour(e);

        const sameDay = (a: Date, b: Date) =>
            a.toDateString() === b.toDateString();
        if (!sameDay(s, selectedDate) && s < selectedDate) startH = 0;
        if (!sameDay(e, selectedDate) && e > selectedDate) endH = 24;
        if (endH < startH) endH = 24;

        const widthPercent = Math.max(0.15, ((endH - startH) / 24) * 100);

        // Calculate actual pixel width to determine if we need minimum width
        const totalWidth = containerWidth * (isZoomed ? 24 / hoursInView : 1);
        const pixelWidth = (widthPercent / 100) * totalWidth;
        const needsMinWidth = pixelWidth < MIN_TASK_WIDTH_PX;

        return {
            left: (startH / 24) * 100,
            width: widthPercent,
            minWidth: needsMinWidth,
        };
    }

    function packTasks(list: TimelineTask[]) {
        const valid = list.filter(
            (t) => t.startTime instanceof Date && t.endTime instanceof Date,
        );
        const sorted = [...valid].sort(
            (a, b) => a.startTime.getTime() - b.startTime.getTime(),
        );
        const rows: Array<{ task: TimelineTask; row: number }> = [];
        const lanes: number[] = [];

        for (const task of sorted) {
            const pos = getTaskPosition(task);
            const left = pos.left;
            // Use a slightly larger right boundary for small tasks to prevent overlap
            const effectiveWidth = pos.minWidth ? 0.5 : pos.width;
            const right = pos.left + effectiveWidth;
            let lane = lanes.findIndex((end) => end <= left);
            if (lane === -1) {
                lane = lanes.length;
                lanes.push(right);
            } else {
                lanes[lane] = right;
            }
            rows.push({ task, row: lane });
        }

        return {
            rows,
            height:
                Math.max(1, lanes.length) * ROW_HEIGHT +
                (lanes.length - 1) * ROW_GAP +
                16,
        };
    }

    const packed = $derived.by(() => packTasks(tasks));

    function getTextColor(bg: string): string {
        if (!bg?.startsWith("#") || bg.length !== 7) return "#fff";
        const r = parseInt(bg.slice(1, 3), 16);
        const g = parseInt(bg.slice(3, 5), 16);
        const b = parseInt(bg.slice(5, 7), 16);
        return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.55
            ? "#1f2937"
            : "#fff";
    }

    function canShowText(
        task: TimelineTask,
        widthPercent: number,
    ): "full" | "icon" | "none" {
        const totalWidth = containerWidth * (isZoomed ? 24 / hoursInView : 1);
        const taskPixelWidth = (widthPercent / 100) * totalWidth;
        if (taskPixelWidth >= 60) return "full";
        if (taskPixelWidth >= 24) return "icon";
        return "none";
    }

    const nowHour = $derived(
        currentTime.getHours() + currentTime.getMinutes() / 60,
    );
    const nowPercent = $derived((nowHour / 24) * 100);
    const showNow = $derived(
        isToday() && nowHour >= viewStartHour && nowHour <= viewEndHour,
    );
    const nowFormatted = $derived(
        currentTime.toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
            hour12: true,
        }),
    );

    function onMouseEnter(e: MouseEvent, id: string) {
        hoveredTaskId = id;
        tooltipPosition = { x: e.clientX + 12, y: e.clientY + 12 };
    }
    function onMouseMove(e: MouseEvent) {
        if (hoveredTaskId)
            tooltipPosition = { x: e.clientX + 12, y: e.clientY + 12 };
    }
    function onMouseLeave() {
        hoveredTaskId = null;
    }
    const hovered = $derived(tasks.find((t) => t.id === hoveredTaskId));
</script>

<div
    class="timeline-gantt h-full flex flex-col bg-base-100 rounded-xl border border-base-300/40 overflow-hidden"
    onwheel={handleWheel}
>
    <!-- Header -->
    <div
        class="flex items-center justify-between px-3 py-2 border-b border-base-200/80 bg-base-100"
    >
        <div class="flex items-center gap-0.5">
            <button
                class="btn btn-ghost btn-sm btn-square"
                onclick={goToPreviousDay}
            >
                <ChevronLeft class="w-4 h-4" />
            </button>
            <button
                class="btn btn-sm px-3 min-w-[90px] font-medium {isToday()
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

        <div class="flex items-center gap-2">
            <span class="text-[9px] text-base-content/50 hidden lg:inline"
                >Ctrl+Scroll</span
            >
            <div
                class="join border border-base-300/60 rounded-md overflow-hidden"
            >
                <button
                    class="join-item btn btn-ghost btn-xs px-1.5"
                    onclick={() => zoomOut()}
                    disabled={zoomIndex === 0}
                >
                    <Minus class="w-3 h-3" />
                </button>
                <span
                    class="join-item flex items-center px-2 text-[10px] font-semibold bg-base-200/40 min-w-[32px] justify-center"
                >
                    {hoursInView}h
                </span>
                <button
                    class="join-item btn btn-ghost btn-xs px-1.5"
                    onclick={() => zoomIn()}
                    disabled={zoomIndex === ZOOM_LEVELS.length - 1}
                >
                    <Plus class="w-3 h-3" />
                </button>
            </div>
            {#if isZoomed}
                <button
                    class="btn btn-ghost btn-xs text-[9px] px-2"
                    onclick={resetZoom}>Reset</button
                >
            {/if}
        </div>

        {#if showNow}
            <div class="flex items-center gap-1.5">
                <span class="w-1.5 h-1.5 rounded-full bg-error animate-pulse"
                ></span>
                <span class="text-[10px] font-mono font-medium text-error"
                    >{nowFormatted}</span
                >
            </div>
        {:else}
            <div class="w-16"></div>
        {/if}
    </div>

    <!-- Timeline Grid -->
    <div
        class="flex-1 overflow-x-auto overflow-y-hidden"
        bind:this={timelineRef}
        onscroll={handleScroll}
    >
        <div
            class="relative min-h-full"
            style="width: {isZoomed ? (24 / hoursInView) * 100 : 100}%;"
        >
            <!-- Hour labels -->
            <div
                class="sticky top-0 z-20 h-6 bg-base-100 border-b border-base-200/50"
            >
                <div class="relative h-full">
                    {#each Array.from({ length: 25 }, (_, i) => i) as hour}
                        {@const showLabel =
                            hoursInView <= 8 || hour % 2 === 0 || hour === 24}
                        <div
                            class="absolute top-0 bottom-0 flex items-center"
                            style="left: {(hour / 24) * 100}%; width: {(1 /
                                24) *
                                100}%;"
                        >
                            {#if showLabel}
                                <span
                                    class="text-[9px] font-medium text-base-content/60 pl-1"
                                >
                                    {formatHour(hour)}
                                </span>
                            {/if}
                        </div>
                    {/each}
                </div>
            </div>

            <!-- Grid and tasks -->
            <div
                class="relative"
                style="min-height: max({packed.height}px, calc(100vh - 220px));"
            >
                <!-- Grid lines -->
                <div class="absolute inset-0 pointer-events-none">
                    {#each Array.from({ length: 25 }, (_, i) => i) as hour}
                        <div
                            class="absolute top-0 bottom-0 w-px {hour === 0 ||
                            hour === 24
                                ? 'bg-base-300/60'
                                : 'bg-base-200/50'}"
                            style="left: {(hour / 24) * 100}%;"
                        ></div>
                    {/each}

                    <!-- Quarter-hour lines (subtle) -->
                    {#each Array.from({ length: 24 }, (_, i) => i) as hour}
                        <div
                            class="absolute top-0 bottom-0 w-px bg-base-200/30"
                            style="left: {((hour + 0.5) / 24) * 100}%;"
                        ></div>
                    {/each}

                    <!-- Now line -->
                    {#if showNow}
                        <div
                            class="absolute top-0 bottom-0 w-0.5 bg-error z-30"
                            style="left: {nowPercent}%;"
                        >
                            <div
                                class="absolute -top-0.5 left-1/2 -translate-x-1/2 w-2 h-2 rounded-full bg-error"
                            ></div>
                        </div>
                    {/if}
                </div>

                <!-- Tasks -->
                <div class="relative py-2 px-0.5">
                    {#each packed.rows as { task, row } (task.id)}
                        {@const pos = getTaskPosition(task)}
                        {@const bg = task.categoryColor || "#6b7280"}
                        {@const txt = getTextColor(bg)}
                        {@const isHov = hoveredTaskId === task.id}
                        {@const isFaded = hoveredTaskId && !isHov}
                        {@const textMode = canShowText(task, pos.width)}

                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                        <!-- svelte-ignore a11y_interactive_supports_focus -->
                        <div
                            class="task-wrapper absolute transition-all duration-100 {isFaded
                                ? 'opacity-50'
                                : ''} {isHov ? 'z-50' : 'z-10'}"
                            style="left: {pos.left}%; width: {pos.width}%; top: {row *
                                (ROW_HEIGHT +
                                    ROW_GAP)}px; height: {ROW_HEIGHT}px; {pos.minWidth
                                ? `min-width: ${MIN_TASK_WIDTH_PX}px;`
                                : ''}"
                            role="button"
                            onmouseenter={(e) => onMouseEnter(e, task.id)}
                            onmousemove={onMouseMove}
                            onmouseleave={onMouseLeave}
                            onclick={() => onTaskClick?.(task.id)}
                        >
                            <div
                                class="task-bar relative h-full rounded cursor-pointer overflow-hidden
                                    {task.completed ? 'task-completed' : ''} 
                                    {isHov
                                    ? 'ring-2 ring-base-content/20 shadow-lg scale-y-110'
                                    : 'shadow-sm'}"
                                style="background-color: {bg};"
                                title={task.title}
                            >
                                <!-- Content -->
                                <div
                                    class="relative h-full flex items-center px-1.5 overflow-hidden"
                                    style="color: {txt};"
                                >
                                    {#if textMode === "full"}
                                        <span
                                            class="text-[10px] font-semibold truncate leading-tight"
                                        >
                                            {task.title}
                                        </span>
                                    {:else if textMode === "none" && pos.minWidth}
                                        <!-- Dot indicator for very small tasks -->
                                    {/if}
                                </div>

                                <!-- Completed overlay (contained within bounds) -->
                                {#if task.completed}
                                    <div
                                        class="completed-overlay absolute inset-0 pointer-events-none rounded overflow-hidden"
                                    ></div>
                                {/if}
                            </div>
                        </div>
                    {/each}
                </div>

                {#if packed.rows.length === 0}
                    <div
                        class="absolute inset-0 flex items-center justify-center"
                    >
                        <div class="text-center py-8 opacity-50">
                            <Clock
                                class="w-8 h-8 mx-auto text-base-content/20 mb-2"
                            />
                            <p class="text-xs text-base-content/40">
                                No tasks for {dateLabel()}
                            </p>
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </div>
</div>

<!-- Tooltip -->
{#if hovered && tooltipPosition}
    <div
        class="fixed z-[200] pointer-events-none"
        style="top: {tooltipPosition.y}px; left: {tooltipPosition.x}px;"
    >
        <div
            class="bg-base-100 rounded-lg shadow-xl border border-base-300 p-2.5 min-w-[140px] max-w-[200px]"
        >
            <div class="flex gap-2">
                <div
                    class="w-1 rounded-full shrink-0"
                    style="background-color: {hovered.categoryColor ||
                        '#6b7280'}"
                ></div>
                <div class="min-w-0 flex-1">
                    <p class="font-semibold text-xs leading-tight">
                        {hovered.title}
                    </p>
                    {#if hovered.categoryName}
                        <p class="text-[9px] text-base-content/50 mt-0.5">
                            {hovered.categoryName}
                        </p>
                    {/if}
                </div>
            </div>
            <div
                class="flex justify-between items-center mt-2 pt-1.5 border-t border-base-200 text-[9px]"
            >
                <span class="text-base-content/50">
                    {formatTime(hovered.startTime)} – {formatTime(
                        hovered.endTime,
                    )}
                </span>
                <span class="font-bold text-primary">
                    {formatDuration(hovered.startTime, hovered.endTime)}
                </span>
            </div>
            {#if hovered.completed}
                <p class="text-[9px] text-success font-medium mt-1">
                    ✓ Completed
                </p>
            {/if}
        </div>
    </div>
{/if}

<style>
    .task-completed {
        opacity: 0.75;
    }

    .completed-overlay {
        background: repeating-linear-gradient(
            -45deg,
            transparent,
            transparent 3px,
            rgba(255, 255, 255, 0.15) 3px,
            rgba(255, 255, 255, 0.15) 5px
        );
    }

    .timeline-gantt ::-webkit-scrollbar {
        height: 4px;
        width: 4px;
    }
    .timeline-gantt ::-webkit-scrollbar-thumb {
        background: oklch(var(--bc) / 0.1);
        border-radius: 2px;
    }
    .timeline-gantt ::-webkit-scrollbar-thumb:hover {
        background: oklch(var(--bc) / 0.2);
    }
    .timeline-gantt ::-webkit-scrollbar-track {
        background: transparent;
    }
</style>
