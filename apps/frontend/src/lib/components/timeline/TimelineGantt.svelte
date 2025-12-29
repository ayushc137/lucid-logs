<script lang="ts">
    import {
        Clock,
        ChevronLeft,
        ChevronRight,
        Minus,
        Plus,
        Check,
        Pencil,
        GripVertical,
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
        onTaskTimeUpdate,
        editMode = false,
        onEditModeChange,
    }: TimelineProps = $props();

    // Internal edit mode state (can be controlled externally or internally)
    let internalEditMode = $state(false);
    const isEditMode = $derived(editMode || internalEditMode);

    function toggleEditMode() {
        internalEditMode = !internalEditMode;
        onEditModeChange?.(internalEditMode);
    }

    // Zoom levels (hours visible)
    const ZOOM_LEVELS = [24, 18, 12, 8, 6, 4, 2];
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
    let gridRef: HTMLDivElement;
    let containerWidth = $state(800);

    // Much bigger row dimensions for better UX
    const ROW_HEIGHT = 52;
    const ROW_GAP = 8;
    const MIN_TASK_WIDTH_PX = 8;

    // Edit mode state
    type DragMode = "move" | "resize-start" | "resize-end" | null;
    let dragMode = $state<DragMode>(null);
    let draggedTaskId = $state<string | null>(null);
    let dragStartX = $state(0);
    let originalStartTime = $state<Date | null>(null);
    let originalEndTime = $state<Date | null>(null);

    // Preview times during drag (used for live UI updates)
    let previewStartTime = $state<Date | null>(null);
    let previewEndTime = $state<Date | null>(null);

    // Track if we're currently in a drag operation (to prevent click)
    let hasDragged = $state(false);

    // Snap to minutes (1 = every minute, 5 = every 5 minutes, etc.)
    const SNAP_MINUTES = 1;

    function snapToMinutes(date: Date): Date {
        const snapped = new Date(date);
        const minutes = snapped.getMinutes();
        const snappedMinutes =
            Math.round(minutes / SNAP_MINUTES) * SNAP_MINUTES;
        snapped.setMinutes(snappedMinutes);
        snapped.setSeconds(0);
        snapped.setMilliseconds(0);
        return snapped;
    }

    function getHourFromMouseX(clientX: number): number {
        if (!gridRef) return 0;
        const rect = gridRef.getBoundingClientRect();
        const x = clientX - rect.left + timelineRef.scrollLeft;
        const totalWidth = gridRef.scrollWidth;
        return Math.max(0, Math.min(24, (x / totalWidth) * 24));
    }

    function hourToDate(hour: number): Date {
        const date = new Date(selectedDate);
        date.setHours(0, 0, 0, 0);
        const hours = Math.floor(hour);
        const minutes = Math.round((hour - hours) * 60);
        date.setHours(hours, minutes, 0, 0);
        return date;
    }

    function handleDragStart(
        e: MouseEvent,
        task: TimelineTask,
        mode: DragMode,
    ) {
        if (!isEditMode || !mode) return;
        e.preventDefault();
        e.stopPropagation();

        dragMode = mode;
        draggedTaskId = task.id;
        dragStartX = e.clientX;
        originalStartTime = new Date(task.startTime);
        originalEndTime = new Date(task.endTime);
        previewStartTime = new Date(task.startTime);
        previewEndTime = new Date(task.endTime);
        hasDragged = false;

        // Add listeners
        document.addEventListener("mousemove", handleDragMove);
        document.addEventListener("mouseup", handleDragEnd);
    }

    function handleDragMove(e: MouseEvent) {
        if (
            !dragMode ||
            !draggedTaskId ||
            !originalStartTime ||
            !originalEndTime
        )
            return;

        // Mark that we've started dragging (to prevent click)
        const deltaX = Math.abs(e.clientX - dragStartX);
        if (deltaX > 3) {
            hasDragged = true;
        }

        const currentHour = getHourFromMouseX(e.clientX);
        const startHour = getHourFromMouseX(dragStartX);
        const deltaHours = currentHour - startHour;

        if (dragMode === "move") {
            // Move the entire task
            const originalStartHour =
                originalStartTime.getHours() +
                originalStartTime.getMinutes() / 60;
            const originalEndHour =
                originalEndTime.getHours() + originalEndTime.getMinutes() / 60;
            const duration = originalEndHour - originalStartHour;

            let newStartHour = originalStartHour + deltaHours;
            let newEndHour = newStartHour + duration;

            // Clamp to day boundaries
            if (newStartHour < 0) {
                newStartHour = 0;
                newEndHour = duration;
            }
            if (newEndHour > 24) {
                newEndHour = 24;
                newStartHour = 24 - duration;
            }

            previewStartTime = snapToMinutes(hourToDate(newStartHour));
            previewEndTime = snapToMinutes(hourToDate(newEndHour));
        } else if (dragMode === "resize-start") {
            // Resize from start
            const originalStartHour =
                originalStartTime.getHours() +
                originalStartTime.getMinutes() / 60;
            let newStartHour = originalStartHour + deltaHours;
            const originalEndHour =
                originalEndTime.getHours() + originalEndTime.getMinutes() / 60;

            // Ensure minimum duration of 5 minutes
            const minEndHour = originalEndHour - 5 / 60;
            newStartHour = Math.max(0, Math.min(minEndHour, newStartHour));

            previewStartTime = snapToMinutes(hourToDate(newStartHour));
            previewEndTime = new Date(originalEndTime);
        } else if (dragMode === "resize-end") {
            // Resize from end
            const originalEndHour =
                originalEndTime.getHours() + originalEndTime.getMinutes() / 60;
            let newEndHour = originalEndHour + deltaHours;
            const originalStartHour =
                originalStartTime.getHours() +
                originalStartTime.getMinutes() / 60;

            // Ensure minimum duration of 5 minutes
            const minStartHour = originalStartHour + 5 / 60;
            newEndHour = Math.max(minStartHour, Math.min(24, newEndHour));

            previewStartTime = new Date(originalStartTime);
            previewEndTime = snapToMinutes(hourToDate(newEndHour));
        }
    }

    function handleDragEnd() {
        if (
            draggedTaskId &&
            previewStartTime &&
            previewEndTime &&
            onTaskTimeUpdate &&
            hasDragged
        ) {
            // Only update if times actually changed
            const startChanged =
                originalStartTime?.getTime() !== previewStartTime.getTime();
            const endChanged =
                originalEndTime?.getTime() !== previewEndTime.getTime();

            if (startChanged || endChanged) {
                onTaskTimeUpdate(
                    draggedTaskId,
                    previewStartTime,
                    previewEndTime,
                );
            }
        }

        // Reset state
        dragMode = null;
        draggedTaskId = null;
        dragStartX = 0;
        originalStartTime = null;
        originalEndTime = null;
        previewStartTime = null;
        previewEndTime = null;

        // Delay resetting hasDragged to prevent immediate click
        setTimeout(() => {
            hasDragged = false;
        }, 100);

        // Remove listeners
        document.removeEventListener("mousemove", handleDragMove);
        document.removeEventListener("mouseup", handleDragEnd);
    }

    // Get preview position for dragged task (used for live UI updates)
    function getPreviewPosition(task: TimelineTask): {
        left: number;
        width: number;
        minWidth: boolean;
    } | null {
        if (draggedTaskId !== task.id || !previewStartTime || !previewEndTime) {
            return null;
        }

        const startH =
            previewStartTime.getHours() + previewStartTime.getMinutes() / 60;
        const endH =
            previewEndTime.getHours() + previewEndTime.getMinutes() / 60;
        const widthPercent = Math.max(0.15, ((endH - startH) / 24) * 100);

        const totalWidth = containerWidth * (isZoomed ? 24 / hoursInView : 1);
        const pixelWidth = (widthPercent / 100) * totalWidth;
        const needsMinWidth = pixelWidth < MIN_TASK_WIDTH_PX;

        return {
            left: (startH / 24) * 100,
            width: widthPercent,
            minWidth: needsMinWidth,
        };
    }

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
        // Clean up any lingering listeners
        document.removeEventListener("mousemove", handleDragMove);
        document.removeEventListener("mouseup", handleDragEnd);
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

    // Better time formatting based on zoom level
    function formatHour(h: number): string {
        const hour = h % 24;
        if (hoursInView <= 4) {
            // Very zoomed in - show full format
            const period = hour < 12 ? "AM" : "PM";
            const displayHour = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
            return `${displayHour}:00 ${period}`;
        } else if (hoursInView <= 8) {
            // Medium zoom - show hour with AM/PM
            const period = hour < 12 ? "AM" : "PM";
            const displayHour = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
            return `${displayHour} ${period}`;
        } else {
            // Full day view - compact
            const period = hour < 12 ? "AM" : "PM";
            const displayHour = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
            return `${displayHour}${period}`;
        }
    }

    // Format for sub-hour markers
    function formatSubHour(h: number, minutes: number): string {
        const hour = Math.floor(h) % 24;
        const period = hour < 12 ? "AM" : "PM";
        const displayHour = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
        const mins = String(minutes).padStart(2, "0");
        return `${displayHour}:${mins}`;
    }

    function formatTime(d: Date): string {
        if (!(d instanceof Date) || isNaN(d.getTime())) return "--:--";
        return d.toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
            hour12: true,
        });
    }

    function formatTimeCompact(d: Date): string {
        if (!(d instanceof Date) || isNaN(d.getTime())) return "--:--";
        const hours = d.getHours();
        const mins = d.getMinutes();
        const period = hours < 12 ? "AM" : "PM";
        const displayHour = hours === 0 ? 12 : hours > 12 ? hours - 12 : hours;
        return `${displayHour}:${String(mins).padStart(2, "0")} ${period}`;
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

        const totalWidth = containerWidth * (isZoomed ? 24 / hoursInView : 1);
        const pixelWidth = (widthPercent / 100) * totalWidth;
        const needsMinWidth = pixelWidth < MIN_TASK_WIDTH_PX;

        return {
            left: (startH / 24) * 100,
            width: widthPercent,
            minWidth: needsMinWidth,
        };
    }

    // Pack tasks with live preview support
    function packTasks(list: TimelineTask[]) {
        const valid = list.filter(
            (t) => t.startTime instanceof Date && t.endTime instanceof Date,
        );

        // Create modified list with preview positions
        const modifiedList = valid.map((task) => {
            if (
                draggedTaskId === task.id &&
                previewStartTime &&
                previewEndTime
            ) {
                return {
                    ...task,
                    startTime: previewStartTime,
                    endTime: previewEndTime,
                };
            }
            return task;
        });

        const sorted = [...modifiedList].sort(
            (a, b) => a.startTime.getTime() - b.startTime.getTime(),
        );
        const rows: Array<{ task: TimelineTask; row: number }> = [];
        const lanes: number[] = [];

        for (const task of sorted) {
            const pos = getTaskPosition(task);
            const left = pos.left;
            const effectiveWidth = pos.minWidth ? 0.5 : pos.width;
            const right = pos.left + effectiveWidth;
            let lane = lanes.findIndex((end) => end <= left);
            if (lane === -1) {
                lane = lanes.length;
                lanes.push(right);
            } else {
                lanes[lane] = right;
            }
            // Find original task to preserve ID
            const originalTask = valid.find((t) => t.id === task.id) || task;
            rows.push({ task: originalTask, row: lane });
        }

        return {
            rows,
            height:
                Math.max(1, lanes.length) * ROW_HEIGHT +
                (lanes.length - 1) * ROW_GAP +
                24,
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

    function getSecondaryTextColor(bg: string): string {
        if (!bg?.startsWith("#") || bg.length !== 7)
            return "rgba(255,255,255,0.7)";
        const r = parseInt(bg.slice(1, 3), 16);
        const g = parseInt(bg.slice(3, 5), 16);
        const b = parseInt(bg.slice(5, 7), 16);
        return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.55
            ? "rgba(31,41,55,0.6)"
            : "rgba(255,255,255,0.7)";
    }

    // Simpler content modes: show title+duration if space, otherwise nothing
    function canShowContent(
        widthPercent: number,
    ): "full" | "title-only" | "none" {
        const totalWidth = containerWidth * (isZoomed ? 24 / hoursInView : 1);
        const taskPixelWidth = (widthPercent / 100) * totalWidth;
        if (taskPixelWidth >= 100) return "full";
        if (taskPixelWidth >= 50) return "title-only";
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

    // Generate grid line data based on zoom level
    const gridLines = $derived.by(() => {
        const lines: Array<{
            hour: number;
            type: "major" | "half" | "quarter" | "minor";
            label?: string;
        }> = [];

        for (let h = 0; h <= 24; h++) {
            // Major hour lines
            lines.push({ hour: h, type: "major", label: formatHour(h) });

            if (h < 24) {
                // Half-hour lines
                lines.push({
                    hour: h + 0.5,
                    type: "half",
                    label: hoursInView <= 6 ? formatSubHour(h, 30) : undefined,
                });

                // Quarter-hour lines (only when zoomed in)
                if (hoursInView <= 8) {
                    lines.push({ hour: h + 0.25, type: "quarter" });
                    lines.push({ hour: h + 0.75, type: "quarter" });
                }

                // 5-minute lines (only when very zoomed in)
                if (hoursInView <= 4) {
                    for (let m = 5; m < 60; m += 5) {
                        if (m !== 15 && m !== 30 && m !== 45) {
                            lines.push({ hour: h + m / 60, type: "minor" });
                        }
                    }
                }
            }
        }

        return lines;
    });

    function onMouseEnter(e: MouseEvent, id: string) {
        if (dragMode) return; // Don't show tooltip while dragging
        hoveredTaskId = id;
        updateTooltipPosition(e);
    }

    function updateTooltipPosition(e: MouseEvent) {
        // Position tooltip above and to the right, ensuring it stays visible
        const x = e.clientX + 12;
        const y = Math.max(80, e.clientY - 80); // Keep above the mouse, min 80px from top
        tooltipPosition = { x, y };
    }

    function onMouseMove(e: MouseEvent) {
        if (hoveredTaskId && !dragMode) {
            updateTooltipPosition(e);
        }
    }
    function onMouseLeave() {
        hoveredTaskId = null;
    }
    const hovered = $derived(tasks.find((t) => t.id === hoveredTaskId));

    function handleTaskClick(e: MouseEvent, taskId: string) {
        // Don't open modal in edit mode or if we just finished dragging
        if (isEditMode || hasDragged || dragMode) {
            e.preventDefault();
            e.stopPropagation();
            return;
        }
        onTaskClick?.(taskId);
    }
</script>

<div
    class="timeline-gantt h-full flex flex-col bg-base-100 rounded-xl border border-base-300/40 overflow-hidden"
    class:edit-active={isEditMode}
    onwheel={handleWheel}
>
    <!-- Header -->
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

        <div class="flex items-center gap-3">
            <!-- Edit Mode Toggle -->
            <div
                class="tooltip tooltip-left"
                data-tip={isEditMode
                    ? "Exit Edit Mode"
                    : "Edit Mode - Drag & Resize"}
            >
                <button
                    class="btn btn-sm gap-2 {isEditMode
                        ? 'btn-warning'
                        : 'btn-ghost border-base-300'}"
                    onclick={toggleEditMode}
                >
                    <Pencil class="w-4 h-4" />
                    <span class="text-xs font-medium"
                        >{isEditMode ? "Done" : "Edit"}</span
                    >
                </button>
            </div>

            <div class="h-5 w-px bg-base-300/60"></div>

            <div class="flex items-center gap-2">
                <span class="text-[10px] text-base-content/40 hidden lg:inline"
                    >Ctrl+Scroll to zoom</span
                >
                <div
                    class="join border border-base-300/60 rounded-lg overflow-hidden"
                >
                    <button
                        class="join-item btn btn-ghost btn-xs px-2"
                        onclick={() => zoomOut()}
                        disabled={zoomIndex === 0}
                    >
                        <Minus class="w-3.5 h-3.5" />
                    </button>
                    <span
                        class="join-item flex items-center px-3 text-xs font-semibold bg-base-200/50 min-w-[40px] justify-center"
                    >
                        {hoursInView}h
                    </span>
                    <button
                        class="join-item btn btn-ghost btn-xs px-2"
                        onclick={() => zoomIn()}
                        disabled={zoomIndex === ZOOM_LEVELS.length - 1}
                    >
                        <Plus class="w-3.5 h-3.5" />
                    </button>
                </div>
                {#if isZoomed}
                    <button
                        class="btn btn-ghost btn-xs text-xs px-2"
                        onclick={resetZoom}>Reset</button
                    >
                {/if}
            </div>
        </div>

        {#if showNow}
            <div class="flex items-center gap-2">
                <span class="w-2 h-2 rounded-full bg-error animate-pulse"
                ></span>
                <span class="text-xs font-mono font-semibold text-error"
                    >{nowFormatted}</span
                >
            </div>
        {:else}
            <div class="w-20"></div>
        {/if}
    </div>

    <!-- Edit Mode Info Bar -->
    {#if isEditMode}
        <div
            class="flex items-center gap-4 px-4 py-2 bg-warning/10 border-b border-warning/20"
        >
            <div class="flex items-center gap-2">
                <GripVertical class="w-4 h-4 text-warning" />
                <span class="text-xs font-medium text-warning"
                    >Drag tasks to move</span
                >
            </div>
            <div class="h-4 w-px bg-warning/30"></div>
            <div class="flex items-center gap-2">
                <div class="flex items-center">
                    <div class="w-1 h-4 bg-warning/60 rounded-full"></div>
                    <div class="w-3 h-0.5 bg-warning/40"></div>
                    <div class="w-1 h-4 bg-warning/60 rounded-full"></div>
                </div>
                <span class="text-xs font-medium text-warning"
                    >Drag edges to resize</span
                >
            </div>
            <div class="flex-1"></div>
            <span class="text-xs text-base-content/50"
                >Changes save automatically</span
            >
        </div>
    {/if}

    <!-- Timeline Grid -->
    <div
        class="flex-1 overflow-x-auto overflow-y-hidden"
        bind:this={timelineRef}
        onscroll={handleScroll}
    >
        <div
            class="relative min-h-full"
            style="width: {isZoomed ? (24 / hoursInView) * 100 : 100}%;"
            bind:this={gridRef}
        >
            <!-- Hour labels -->
            <div
                class="sticky top-0 z-20 h-8 bg-base-100 border-b border-base-200"
            >
                <div class="relative h-full">
                    {#each gridLines.filter((l) => l.type === "major" || (l.type === "half" && l.label)) as line}
                        <div
                            class="absolute top-0 bottom-0 flex items-end pb-1"
                            style="left: {(line.hour / 24) * 100}%;"
                        >
                            <span
                                class="text-[10px] font-medium whitespace-nowrap pl-1 {line.type ===
                                'major'
                                    ? 'text-base-content/70'
                                    : 'text-base-content/40'}"
                            >
                                {line.label || ""}
                            </span>
                        </div>
                    {/each}
                </div>
            </div>

            <!-- Grid and tasks -->
            <div
                class="relative"
                style="min-height: max({packed.height}px, calc(100vh - 240px));"
            >
                <!-- Grid lines -->
                <div class="absolute inset-0 pointer-events-none">
                    {#each gridLines as line}
                        <div
                            class="absolute top-0 bottom-0 {line.type ===
                            'major'
                                ? 'w-px bg-base-300/80'
                                : line.type === 'half'
                                  ? 'w-px bg-base-200/60'
                                  : line.type === 'quarter'
                                    ? 'w-px bg-base-200/40'
                                    : 'w-px bg-base-200/25'}"
                            style="left: {(line.hour / 24) * 100}%;"
                        ></div>
                    {/each}

                    <!-- Now line -->
                    {#if showNow}
                        <div
                            class="absolute top-0 bottom-0 w-0.5 bg-error z-30"
                            style="left: {nowPercent}%;"
                        >
                            <div
                                class="absolute -top-1 left-1/2 -translate-x-1/2 w-2.5 h-2.5 rounded-full bg-error shadow-lg"
                            ></div>
                        </div>
                    {/if}
                </div>

                <!-- Tasks -->
                <div class="relative py-3 px-1">
                    {#each packed.rows as { task, row } (task.id)}
                        {@const originalPos = getTaskPosition(task)}
                        {@const previewPos = getPreviewPosition(task)}
                        {@const pos = previewPos || originalPos}
                        {@const bg = task.categoryColor || "#6b7280"}
                        {@const txt = getTextColor(bg)}
                        {@const txtSecondary = getSecondaryTextColor(bg)}
                        {@const isHov = hoveredTaskId === task.id}
                        {@const isFaded = hoveredTaskId && !isHov && !dragMode}
                        {@const contentMode = canShowContent(pos.width)}
                        {@const isDragging = draggedTaskId === task.id}
                        {@const showPreviewTimes =
                            isDragging && previewStartTime && previewEndTime}
                        {@const taskStartTime =
                            isDragging && previewStartTime
                                ? previewStartTime
                                : task.startTime}
                        {@const taskEndTime =
                            isDragging && previewEndTime
                                ? previewEndTime
                                : task.endTime}

                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                        <!-- svelte-ignore a11y_interactive_supports_focus -->
                        <div
                            class="task-wrapper absolute {isFaded
                                ? 'opacity-40'
                                : ''} {isHov && !isDragging
                                ? 'z-50'
                                : 'z-10'} {isDragging ? 'z-[100]' : ''}"
                            style="left: {pos.left}%; width: {pos.width}%; top: {row *
                                (ROW_HEIGHT +
                                    ROW_GAP)}px; height: {ROW_HEIGHT}px; {pos.minWidth
                                ? `min-width: ${MIN_TASK_WIDTH_PX}px;`
                                : ''} transition: {isDragging
                                ? 'none'
                                : 'opacity 0.15s ease, transform 0.15s ease'};"
                            role="button"
                            onmouseenter={(e) => onMouseEnter(e, task.id)}
                            onmousemove={onMouseMove}
                            onmouseleave={onMouseLeave}
                            onclick={(e) => handleTaskClick(e, task.id)}
                        >
                            <!-- Time preview bubble during drag - positioned ABOVE the task bar -->
                            {#if showPreviewTimes && previewStartTime && previewEndTime}
                                <div
                                    class="absolute -top-9 left-1/2 -translate-x-1/2 z-[300] pointer-events-none"
                                >
                                    <div
                                        class="time-preview-badge bg-warning text-warning-content px-3 py-1 rounded-full text-xs font-bold shadow-xl whitespace-nowrap flex items-center gap-2"
                                    >
                                        <span
                                            >{formatTimeCompact(
                                                previewStartTime,
                                            )}</span
                                        >
                                        <span class="opacity-60">→</span>
                                        <span
                                            >{formatTimeCompact(
                                                previewEndTime,
                                            )}</span
                                        >
                                        <span
                                            class="ml-1 px-1.5 py-0.5 bg-warning-content/20 rounded text-[10px]"
                                        >
                                            {formatDuration(
                                                previewStartTime,
                                                previewEndTime,
                                            )}
                                        </span>
                                    </div>
                                </div>
                            {/if}

                            <!-- svelte-ignore a11y_no_static_element_interactions -->
                            <div
                                class="task-bar relative h-full rounded-lg overflow-visible shadow-md
                                    {task.completed ? 'task-completed' : ''} 
                                    {isHov && !isDragging
                                    ? 'ring-2 ring-base-content/25 shadow-xl scale-[1.02]'
                                    : ''}
                                    {isDragging
                                    ? 'ring-2 ring-warning shadow-2xl scale-[1.03] cursor-grabbing'
                                    : ''}
                                    {isEditMode && !isDragging
                                    ? 'edit-mode-task cursor-grab'
                                    : 'cursor-pointer'}"
                                style="background: linear-gradient(135deg, {bg} 0%, {bg}dd 100%); border: 1px solid rgba(0,0,0,0.1);"
                                onmousedown={(e) =>
                                    isEditMode &&
                                    handleDragStart(e, task, "move")}
                            >
                                <!-- Edit mode resize handles -->
                                {#if isEditMode}
                                    <!-- Left resize handle -->
                                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                                    <div
                                        class="resize-handle resize-left absolute left-0 top-0 bottom-0 w-3 cursor-ew-resize z-20 flex items-center justify-center rounded-l-lg"
                                        role="presentation"
                                        onmousedown={(e) =>
                                            handleDragStart(
                                                e,
                                                task,
                                                "resize-start",
                                            )}
                                    >
                                        <div
                                            class="w-1 h-6 rounded-full bg-white/40 group-hover:bg-white/60"
                                        ></div>
                                    </div>

                                    <!-- Right resize handle -->
                                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                                    <div
                                        class="resize-handle resize-right absolute right-0 top-0 bottom-0 w-3 cursor-ew-resize z-20 flex items-center justify-center rounded-r-lg"
                                        role="presentation"
                                        onmousedown={(e) =>
                                            handleDragStart(
                                                e,
                                                task,
                                                "resize-end",
                                            )}
                                    >
                                        <div
                                            class="w-1 h-6 rounded-full bg-white/40 group-hover:bg-white/60"
                                        ></div>
                                    </div>
                                {/if}

                                <!-- Content - Clean and minimal -->
                                <div
                                    class="relative h-full flex items-center px-2.5 overflow-hidden"
                                    style="color: {txt};"
                                >
                                    {#if contentMode === "full"}
                                        <!-- Full: Title + Duration -->
                                        <div
                                            class="flex-1 min-w-0 flex items-center gap-3"
                                        >
                                            <p
                                                class="text-sm font-semibold truncate flex-1 {task.completed
                                                    ? 'opacity-60'
                                                    : ''}"
                                            >
                                                {task.title}
                                            </p>
                                            <span
                                                class="text-xs font-medium px-2 py-0.5 rounded bg-black/10 shrink-0"
                                            >
                                                {formatDuration(
                                                    taskStartTime,
                                                    taskEndTime,
                                                )}
                                            </span>
                                        </div>
                                    {:else if contentMode === "title-only"}
                                        <!-- Title only -->
                                        <p
                                            class="text-xs font-semibold truncate {task.completed
                                                ? 'opacity-60'
                                                : ''}"
                                        >
                                            {task.title}
                                        </p>
                                    {:else}
                                        <!-- Too small - empty, just shows the colored bar -->
                                    {/if}
                                </div>

                                <!-- Completed overlay -->
                                {#if task.completed}
                                    <div
                                        class="completed-overlay absolute inset-0 pointer-events-none rounded-lg overflow-hidden"
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
                        <div class="text-center py-12 opacity-60">
                            <Clock
                                class="w-12 h-12 mx-auto text-base-content/20 mb-3"
                            />
                            <p class="text-sm font-medium text-base-content/40">
                                No tasks for {dateLabel()}
                            </p>
                            <p class="text-xs text-base-content/30 mt-1">
                                Click "New Task" to add one
                            </p>
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </div>
</div>

<!-- Tooltip - High z-index to ensure visibility -->
{#if hovered && tooltipPosition && !dragMode && !isEditMode}
    <div
        class="fixed z-[9999] pointer-events-none"
        style="top: {tooltipPosition.y}px; left: {tooltipPosition.x}px;"
    >
        <div
            class="bg-base-100 rounded-xl shadow-2xl border border-base-300 p-3 min-w-[160px] max-w-[240px]"
        >
            <div class="flex gap-2.5">
                <div
                    class="w-1.5 rounded-full shrink-0"
                    style="background-color: {hovered.categoryColor ||
                        '#6b7280'}"
                ></div>
                <div class="min-w-0 flex-1">
                    <p class="font-semibold text-sm leading-tight">
                        {hovered.title}
                    </p>
                    {#if hovered.categoryName}
                        <p class="text-xs text-base-content/50 mt-0.5">
                            {hovered.categoryName}
                        </p>
                    {/if}
                </div>
            </div>
            <div
                class="flex justify-between items-center mt-2.5 pt-2 border-t border-base-200 text-xs"
            >
                <span class="text-base-content/60">
                    {formatTime(hovered.startTime)} – {formatTime(
                        hovered.endTime,
                    )}
                </span>
                <span class="font-bold text-primary">
                    {formatDuration(hovered.startTime, hovered.endTime)}
                </span>
            </div>
            {#if hovered.completed}
                <div class="flex items-center gap-1.5 mt-2 text-success">
                    <Check class="w-3.5 h-3.5" />
                    <span class="text-xs font-medium">Completed</span>
                </div>
            {/if}
        </div>
    </div>
{/if}

<style>
    .task-completed {
        opacity: 0.8;
    }

    .completed-overlay {
        background: repeating-linear-gradient(
            -45deg,
            transparent,
            transparent 4px,
            rgba(255, 255, 255, 0.08) 4px,
            rgba(255, 255, 255, 0.08) 6px
        );
    }

    .timeline-gantt ::-webkit-scrollbar {
        height: 6px;
        width: 6px;
    }
    .timeline-gantt ::-webkit-scrollbar-thumb {
        background: oklch(var(--bc) / 0.15);
        border-radius: 3px;
    }
    .timeline-gantt ::-webkit-scrollbar-thumb:hover {
        background: oklch(var(--bc) / 0.25);
    }
    .timeline-gantt ::-webkit-scrollbar-track {
        background: transparent;
    }

    /* Edit mode styles */
    .edit-active {
        --edit-border-color: oklch(var(--wa));
    }

    .edit-mode-task:hover .resize-handle {
        opacity: 1;
        background: rgba(255, 255, 255, 0.15);
    }

    .resize-handle {
        opacity: 0;
        transition:
            opacity 0.15s ease,
            background 0.15s ease;
    }

    .resize-handle:hover {
        opacity: 1;
        background: rgba(255, 255, 255, 0.25) !important;
    }

    .resize-left:hover {
        border-radius: 8px 0 0 8px;
    }

    .resize-right:hover {
        border-radius: 0 8px 8px 0;
    }

    .time-preview-badge {
        animation: pop-in 0.12s ease-out;
    }

    @keyframes pop-in {
        0% {
            transform: translateX(-50%) scale(0.9);
            opacity: 0;
        }
        100% {
            transform: translateX(-50%) scale(1);
            opacity: 1;
        }
    }

    /* Task hover effects */
    .task-bar {
        transition:
            transform 0.15s ease,
            box-shadow 0.15s ease,
            ring 0.15s ease;
    }

    .task-wrapper {
        transition: opacity 0.15s ease;
    }
</style>
