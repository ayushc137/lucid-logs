<script lang="ts">
    import {
        Minus,
        Plus,
        Check,
        Pencil,
        GripVertical,
        Clock,
        MoveHorizontal,
    } from "lucide-svelte";
    import { onMount, onDestroy } from "svelte";
    import { fade, scale, slide } from "svelte/transition";
    import type { TimelineTask, TimelineProps } from "./types";
    import { stripHtml } from "$lib/utils";
    import { OpenMoji } from "$lib/components/ui";

    // Shared components
    import DateNavigator from "./DateNavigator.svelte";
    import CategoryFilter from "./CategoryFilter.svelte";
    import TaskPopover from "./TaskPopover.svelte";

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

    // Category filter state
    let selectedCategoryFilter = $state<string | null>(null);

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

    // Track mouse position and task position during drag for the fixed popover
    let dragMouseX = $state(0);
    let dragTaskTop = $state(0);

    // Optimistic updates to prevent snap-back while saving
    let optimisticOverrides = $state<
        Record<string, { startTime: Date; endTime: Date }>
    >({});

    // Clear overrides when tasks prop updates (reconciliation)
    $effect(() => {
        let changed = false;
        const nextOverrides = { ...optimisticOverrides };
        const currentIds = new Set(tasks.map((t) => t.id));

        // 1. Remove overrides for deleted tasks
        for (const id in nextOverrides) {
            if (!currentIds.has(id)) {
                delete nextOverrides[id];
                changed = true;
            }
        }

        // 2. Remove overrides that match the current server state
        for (const task of tasks) {
            const override = nextOverrides[task.id];
            if (override) {
                const startDiff = Math.abs(
                    task.startTime.getTime() - override.startTime.getTime(),
                );
                const endDiff = Math.abs(
                    task.endTime.getTime() - override.endTime.getTime(),
                );
                // If reasonably close (within 5s to account for precision loss), assume synced
                if (startDiff < 5000 && endDiff < 5000) {
                    delete nextOverrides[task.id];
                    changed = true;
                }
            }
        }

        if (changed) {
            optimisticOverrides = nextOverrides;
        }
    });

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
        // getBoundingClientRect already accounts for scroll position
        // so we just need clientX - rect.left to get position within grid
        const x = clientX - rect.left;
        const totalWidth = gridRef.scrollWidth;
        // Convert to hour and clamp to 0-24 range
        const hour = (x / totalWidth) * 24;
        return Math.max(0, Math.min(24, hour));
    }

    function hourToDate(hour: number): Date {
        const date = new Date(selectedDate);
        date.setHours(0, 0, 0, 0);
        const hours = Math.floor(hour);
        const minutes = Math.round((hour - hours) * 60);
        date.setHours(hours, minutes, 0, 0);
        return date;
    }

    function getRelativeHour(date: Date): number {
        const startOfDay = new Date(selectedDate);
        startOfDay.setHours(0, 0, 0, 0);
        // Calculate difference in milliseconds and convert to hours
        return (date.getTime() - startOfDay.getTime()) / (1000 * 60 * 60);
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
        dragMouseX = e.clientX;

        // Get the task element's position for the popover
        const taskElement = (e.target as HTMLElement).closest(".task-wrapper");
        if (taskElement) {
            const rect = taskElement.getBoundingClientRect();
            dragTaskTop = rect.top;
        }

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

        // Update mouse X position for the fixed popover
        dragMouseX = e.clientX;

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
            const originalStartHour = getRelativeHour(originalStartTime);
            const originalEndHour = getRelativeHour(originalEndTime);
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
            const originalStartHour = getRelativeHour(originalStartTime);
            let newStartHour = originalStartHour + deltaHours;
            const originalEndHour = getRelativeHour(originalEndTime);

            // Ensure minimum duration of 5 minutes
            const minEndHour = originalEndHour - 5 / 60;
            newStartHour = Math.max(0, Math.min(minEndHour, newStartHour));

            previewStartTime = snapToMinutes(hourToDate(newStartHour));
            previewEndTime = new Date(originalEndTime);
        } else if (dragMode === "resize-end") {
            // Resize from end
            const originalEndHour = getRelativeHour(originalEndTime);
            let newEndHour = originalEndHour + deltaHours;
            const originalStartHour = getRelativeHour(originalStartTime);

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
                // 1. Optimistic Update
                optimisticOverrides = {
                    ...optimisticOverrides,
                    [draggedTaskId]: {
                        startTime: previewStartTime,
                        endTime: previewEndTime,
                    },
                };

                // 2. Trigger Callback
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
        dragMouseX = 0;
        dragTaskTop = 0;

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

    // Extract unique categories with task counts
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

    // Filtered tasks - handle "__uncategorized__" special filter
    const filteredTasks = $derived(
        selectedCategoryFilter === "__uncategorized__"
            ? tasks.filter((t) => !t.categoryName)
            : selectedCategoryFilter
              ? tasks.filter((t) => t.categoryName === selectedCategoryFilter)
              : tasks,
    );

    // Check if viewing today
    const isToday = $derived(() => {
        const t = new Date();
        return (
            selectedDate.getFullYear() === t.getFullYear() &&
            selectedDate.getMonth() === t.getMonth() &&
            selectedDate.getDate() === t.getDate()
        );
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

    // Pack tasks - modified to prevent vertical jitter during drag
    // 1. Apply optimistic overrides FIRST
    // 2. Ignore drag previews for sorting/lanes (keeps rows stable during drag)
    function packTasks(list: TimelineTask[]) {
        const viewStart = new Date(selectedDate);
        viewStart.setHours(0, 0, 0, 0);
        const viewEnd = new Date(selectedDate);
        viewEnd.setHours(23, 59, 59, 999);

        // Filter valid AND visible tasks
        const valid = list.filter((t) => {
            if (!(t.startTime instanceof Date) || !(t.endTime instanceof Date))
                return false;
            // Must overlap with today
            // Task ends after view starts AND starts before view ends
            return t.endTime > viewStart && t.startTime < viewEnd;
        });

        // Apply optimistic overrides
        const effectiveTasks = valid.map((task) => {
            if (optimisticOverrides[task.id]) {
                return {
                    ...task,
                    ...optimisticOverrides[task.id],
                };
            }
            return task;
        });

        // Sort by effective time
        const sorted = [...effectiveTasks].sort(
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
            // Find original task to preserve ID (we want the original object ref if possible, but updated times logic handled nicely inside loop)
            // Actually map back to the original task object but we need the calculated row
            // The template will handle the display position (preview vs optimistic vs original)
            const originalTask = list.find((t) => t.id === task.id) || task;
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

    const packed = $derived.by(() => packTasks(filteredTasks));

    function getTextColor(bg: string): string {
        // Always white for clean, consistent look on vibrant/dark colors
        return "#fff";
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
    const showNow = $derived(isToday());
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

        // Dynamic Label Density: Calculate pixels per hour to prevent overlapping text
        const pixelsPerHour = containerWidth / hoursInView;
        let labelStep = 1;

        if (pixelsPerHour < 25)
            labelStep = 6; // Very tight (e.g., mobile 24h view)
        else if (pixelsPerHour < 45)
            labelStep = 3; // Tight
        else if (pixelsPerHour < 60) labelStep = 2; // Medium

        for (let h = 0; h <= 24; h++) {
            // Major hour lines
            const showLabel = h % labelStep === 0;
            // Don't show label for 24h (end of day) if it's 12 AM
            const label = h === 24 || !showLabel ? undefined : formatHour(h);
            lines.push({ hour: h, type: "major", label });

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
        // Position tooltip below and to the right of the mouse
        const x = e.clientX + 16;
        const y = e.clientY + 16;
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

    function getEffectiveTime(
        task: TimelineTask,
        overrides: Record<string, { startTime: Date; endTime: Date }>,
        isDragging: boolean,
        previewStart: Date | null,
        previewEnd: Date | null,
        type: "start" | "end",
    ): Date {
        // use preview times if dragging THIS task
        if (isDragging && previewStart && previewEnd) {
            return type === "start" ? previewStart : previewEnd;
        }
        // use optimistic override if strictly present
        const override = overrides[task.id];
        if (override) {
            return type === "start" ? override.startTime : override.endTime;
        }
        // fallback to task time
        return type === "start" ? task.startTime : task.endTime;
    }
</script>

<div
    class="timeline-gantt h-full flex flex-col bg-base-100 rounded-2xl border border-base-200 shadow-sm overflow-hidden select-none"
    class:edit-active={isEditMode}
    onwheel={handleWheel}
>
    {#snippet controls()}
        <div class="flex items-center gap-2 sm:gap-3">
            <!-- Edit Mode Toggle -->
            <button
                class="btn btn-sm gap-2 transition-all duration-300 rounded-lg {isEditMode
                    ? 'btn-warning text-warning-content shadow-lg shadow-warning/20'
                    : 'btn-ghost hover:bg-base-200 text-base-content/70'}"
                onclick={toggleEditMode}
            >
                <Pencil class="w-3.5 h-3.5" />
                <span class="font-medium text-xs hidden sm:inline"
                    >{isEditMode ? "Done" : "Edit"}</span
                >
            </button>

            <!-- Zoom Controls -->
            <div
                class="flex items-center gap-1 bg-base-200/50 p-1 rounded-lg border border-base-200/60"
            >
                <button
                    class="btn btn-xs btn-square btn-ghost hover:bg-base-100 rounded-md"
                    onclick={() => zoomOut()}
                    disabled={zoomIndex === 0}
                    aria-label="Zoom Out"
                >
                    <Minus class="w-3 h-3" />
                </button>
                <span
                    class="text-[10px] font-bold w-6 text-center tabular-nums opacity-60 select-none px-1"
                >
                    {hoursInView}h
                </span>
                <button
                    class="btn btn-xs btn-square btn-ghost hover:bg-base-100 rounded-md"
                    onclick={() => zoomIn()}
                    disabled={zoomIndex === ZOOM_LEVELS.length - 1}
                    aria-label="Zoom In"
                >
                    <Plus class="w-3 h-3" />
                </button>
            </div>
        </div>
    {/snippet}

    <!-- Header -->
    <div
        class="flex flex-col md:flex-row items-center justify-between gap-4 px-4 py-3 bg-base-100/95 backdrop-blur-xl border-b border-base-200 z-40 sticky top-0"
    >
        <div class="flex items-center justify-between w-full md:w-auto gap-4">
            <!-- Left: Date Navigation -->
            <DateNavigator {selectedDate} {onDateChange} />

            <!-- Controls (Mobile) -->
            <div class="md:hidden">
                {@render controls()}
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

        <!-- Right: Controls (Desktop) -->
        <div class="hidden md:block">
            {@render controls()}
        </div>
    </div>

    <!-- Edit Mode Info Bar -->
    {#if isEditMode}
        <div
            class="flex items-center justify-between px-5 py-2 bg-warning/10 border-b border-warning/10 text-warning-content"
            transition:slide={{ axis: "y", duration: 200 }}
        >
            <div class="flex items-center gap-6">
                <div class="flex items-center gap-2">
                    <GripVertical class="w-4 h-4 opacity-70" />
                    <span class="text-xs font-semibold">Drag to move</span>
                </div>
                <div class="flex items-center gap-2">
                    <MoveHorizontal class="w-4 h-4 opacity-70" />
                    <span class="text-xs font-semibold"
                        >Drag edges to resize</span
                    >
                </div>
            </div>
            <span
                class="text-[10px] font-bold tracking-wider uppercase opacity-60"
                >Auto-saving</span
            >
        </div>
    {/if}

    <!-- Timeline Grid -->
    <div
        class="flex-1 overflow-x-auto overflow-y-hidden custom-scrollbar bg-base-50/30 relative"
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
                class="sticky top-0 z-30 h-9 bg-base-100/90 backdrop-blur-sm border-b border-base-200 shadow-sm"
            >
                <div class="relative h-full">
                    {#each gridLines.filter((l) => l.type === "major" || (l.type === "half" && l.label)) as line}
                        <div
                            class="absolute top-0 bottom-0 flex items-end pb-1.5 transition-all duration-300"
                            style="left: {(line.hour / 24) * 100}%;"
                        >
                            <span
                                class="text-[10px] font-bold whitespace-nowrap pl-1.5 select-none {line.type ===
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
                class="relative pt-4 pb-12"
                style="min-height: max({packed.height}px, calc(100vh - 280px));"
            >
                <!-- Grid lines -->
                <div class="absolute inset-0 pointer-events-none">
                    {#each gridLines as line}
                        <div
                            class="absolute top-0 bottom-0 {line.type ===
                            'major'
                                ? 'w-px bg-base-300/50'
                                : line.type === 'half'
                                  ? 'w-px bg-base-200/50'
                                  : 'w-px bg-base-200/20'}"
                            style="left: {(line.hour / 24) * 100}%;"
                        ></div>
                    {/each}

                    <!-- Now line -->
                    {#if showNow}
                        <div
                            class="absolute top-0 bottom-0 w-px bg-error z-20 shadow-[0_0_12px_rgba(239,68,68,0.5)] opacity-80"
                            style="left: {nowPercent}%;"
                        >
                            <div
                                class="absolute -top-1.5 left-1/2 -translate-x-1/2 w-3 h-3 rounded-full bg-error shadow-sm grid place-items-center"
                            >
                                <div
                                    class="w-1.5 h-1.5 rounded-full bg-white/80"
                                ></div>
                            </div>
                        </div>
                    {/if}
                </div>

                <!-- Tasks -->
                <div class="relative px-2">
                    {#each packed.rows as { task, row } (task.id)}
                        {@const bg = task.categoryColor || "#6b7280"}
                        {@const txt = getTextColor(bg)}
                        {@const isHov = hoveredTaskId === task.id}
                        {@const isDragging = draggedTaskId === task.id}

                        <!-- Dim other tasks when focusing on one -->
                        {@const isDimmed =
                            (hoveredTaskId || draggedTaskId) &&
                            !isHov &&
                            !isDragging}

                        <!-- Calculate times & Dragging Logic -->
                        {@const previewTimes =
                            isDragging && previewStartTime && previewEndTime}
                        {@const taskStartTime = getEffectiveTime(
                            task,
                            optimisticOverrides,
                            isDragging,
                            previewStartTime,
                            previewEndTime,
                            "start",
                        )}
                        {@const taskEndTime = getEffectiveTime(
                            task,
                            optimisticOverrides,
                            isDragging,
                            previewStartTime,
                            previewEndTime,
                            "end",
                        )}
                        {@const pos = getTaskPosition({
                            ...task,
                            startTime: taskStartTime,
                            endTime: taskEndTime,
                        })}

                        {@const contentMode = canShowContent(pos.width)}
                        {@const isLeftEdge = pos.left <= 0.05}
                        {@const isRightEdge = pos.left + pos.width >= 99.95}

                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                        <!-- svelte-ignore a11y_interactive_supports_focus -->
                        <div
                            class="task-wrapper absolute will-change-transform
                                {isDimmed
                                ? 'opacity-30 grayscale-[50%] blur-[1px]'
                                : 'opacity-100'} 
                                {isHov && !isDragging ? 'z-50' : 'z-10'} 
                                {isDragging ? 'z-[100]' : ''}"
                            style="
                                left: {pos.left}%; 
                                width: {pos.width}%; 
                                top: {row * (ROW_HEIGHT + ROW_GAP)}px; 
                                height: {ROW_HEIGHT}px; 
                                {pos.minWidth
                                ? `min-width: ${MIN_TASK_WIDTH_PX}px;`
                                : ''} 
                                transition: {isDragging
                                ? 'none'
                                : 'all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1)'};
                            "
                            role="button"
                            onmouseenter={(e) => onMouseEnter(e, task.id)}
                            onmousemove={onMouseMove}
                            onmouseleave={onMouseLeave}
                            onclick={(e) => handleTaskClick(e, task.id)}
                        >
                            <!-- svelte-ignore a11y_no_static_element_interactions -->
                            <div
                                class="task-bar relative h-full rounded-lg overflow-hidden transition-all duration-300
                                    {isHov && !isDragging
                                    ? 'shadow-xl ring-2 ring-white/60 scale-[1.01] -translate-y-0.5'
                                    : 'shadow-md'}
                                    {isDragging
                                    ? 'shadow-2xl ring-2 ring-warning scale-[1.02] cursor-grabbing'
                                    : ''}
                                    {isEditMode && !isDragging
                                    ? 'cursor-grab hover:ring-2 hover:ring-warning/50'
                                    : 'cursor-pointer'}
                                    {task.completed
                                    ? 'ring-1 ring-base-content/5 opacity-85 saturate-[0.8]'
                                    : ''}
                                    {isLeftEdge ? '!rounded-l-none' : ''} 
                                    {isRightEdge ? '!rounded-r-none' : ''}"
                                style="background-color: {bg} !important;"
                                onmousedown={(e) =>
                                    isEditMode &&
                                    handleDragStart(e, task, "move")}
                            >
                                <!-- Interactive Gradient & Sheen -->
                                <div
                                    class="absolute inset-0 bg-gradient-to-br from-white/20 via-transparent to-black/10 pointer-events-none"
                                ></div>
                                {#if isHov && !isDragging}
                                    <div
                                        class="absolute inset-0 bg-white/10 pointer-events-none transition-opacity duration-200"
                                    ></div>
                                {/if}

                                <!-- Edit Handles -->
                                {#if isEditMode}
                                    {#if !isLeftEdge}
                                        <div
                                            class="absolute left-0 top-0 bottom-0 w-3 cursor-ew-resize z-20 flex items-center justify-start pl-0.5 opacity-0 hover:opacity-100 transition-opacity bg-black/10 hover:bg-black/20"
                                            onmousedown={(e) =>
                                                handleDragStart(
                                                    e,
                                                    task,
                                                    "resize-start",
                                                )}
                                        >
                                            <div
                                                class="w-1 h-3 rounded-full bg-white/80 shadow-sm"
                                            ></div>
                                        </div>
                                    {/if}
                                    {#if !isRightEdge}
                                        <div
                                            class="absolute right-0 top-0 bottom-0 w-3 cursor-ew-resize z-20 flex items-center justify-end pr-0.5 opacity-0 hover:opacity-100 transition-opacity bg-black/10 hover:bg-black/20"
                                            onmousedown={(e) =>
                                                handleDragStart(
                                                    e,
                                                    task,
                                                    "resize-end",
                                                )}
                                        >
                                            <div
                                                class="w-1 h-3 rounded-full bg-white/80 shadow-sm"
                                            ></div>
                                        </div>
                                    {/if}
                                {/if}

                                <!-- Content -->
                                <div
                                    class="relative h-full flex items-center px-3 z-10"
                                    style="color: {txt};"
                                >
                                    {#if contentMode === "full"}
                                        <div
                                            class="flex flex-col justify-center min-w-0 flex-1"
                                        >
                                            <div
                                                class="flex items-center gap-2"
                                            >
                                                <span
                                                    class="font-extrabold text-sm truncate drop-shadow-md {task.completed
                                                        ? 'opacity-80'
                                                        : ''}"
                                                    >{task.title}</span
                                                >
                                            </div>
                                            {#if task.description && ROW_HEIGHT > 40}
                                                <span
                                                    class="text-[10px] opacity-80 truncate"
                                                    >{stripHtml(
                                                        task.description,
                                                        { compact: true },
                                                    )}</span
                                                >
                                            {/if}
                                        </div>
                                        <div
                                            class="ml-auto pl-2 flex items-center gap-1"
                                        >
                                            <!-- Emotion Emoji near duration -->
                                            {#if task.emotionEmoji}
                                                <div
                                                    class="shrink-0 drop-shadow-md"
                                                >
                                                    <OpenMoji
                                                        emoji={task.emotionEmoji}
                                                        size={24}
                                                    />
                                                </div>
                                            {:else if task.inferredEmotionEmoji}
                                                <div
                                                    class="shrink-0 drop-shadow-md opacity-70"
                                                >
                                                    <OpenMoji
                                                        emoji={task.inferredEmotionEmoji}
                                                        size={22}
                                                    />
                                                </div>
                                            {/if}
                                            <span
                                                class="text-[10px] font-bold bg-black/20 px-1.5 py-0.5 rounded-md backdrop-blur-sm whitespace-nowrap border border-white/10"
                                            >
                                                {formatDuration(
                                                    taskStartTime,
                                                    taskEndTime,
                                                )}
                                            </span>
                                        </div>
                                    {:else if contentMode === "title-only"}
                                        <span
                                            class="font-bold text-xs truncate drop-shadow-md {task.completed
                                                ? 'opacity-80'
                                                : ''}">{task.title}</span
                                        >
                                    {:else}
                                        <!-- Minimal View - no content shown -->
                                    {/if}
                                </div>

                                <!-- Completed Overlay with Icon -->
                                {#if task.completed}
                                    <!-- Subtle striped pattern for completed tasks -->
                                    <div
                                        class="absolute inset-0 pointer-events-none mix-blend-overlay opacity-30"
                                        style="background-image: repeating-linear-gradient(45deg, transparent, transparent 5px, rgba(0,0,0,0.2) 5px, rgba(0,0,0,0.2) 10px);"
                                    ></div>
                                {/if}
                            </div>
                        </div>
                    {/each}
                </div>

                {#if packed.rows.length === 0}
                    <div
                        class="absolute inset-0 flex items-center justify-center p-8 pointer-events-none"
                    >
                        <div
                            class="flex flex-col items-center justify-center text-center p-8 border border-dashed border-base-300 rounded-3xl bg-base-100/40 backdrop-blur-sm max-w-sm"
                        >
                            <div class="bg-base-200/50 p-4 rounded-full mb-4">
                                <Clock class="w-8 h-8 text-base-content/20" />
                            </div>
                            <h3 class="font-bold text-base-content/60">
                                No tasks for today
                            </h3>
                            <p class="text-xs text-base-content/40 mt-1">
                                Click 'New Task' to get started.
                            </p>
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </div>
</div>

<!-- Task Popover -->
{#if hovered && tooltipPosition && !dragMode}
    <TaskPopover task={hovered} position={tooltipPosition} />
{/if}

<!-- Fixed position drag time preview - positioned above task at mouse X -->
{#if dragMode && previewStartTime && previewEndTime}
    <div
        class="fixed pointer-events-none z-[9999]"
        style="left: {dragMouseX}px; top: {dragTaskTop -
            36}px; transform: translateX(-50%);"
    >
        <div
            class="bg-warning text-warning-content px-3 py-1.5 rounded-lg text-xs font-semibold shadow-lg flex items-center gap-2 whitespace-nowrap"
        >
            <span class="font-mono">{formatTimeCompact(previewStartTime)}</span>
            <span class="opacity-50">→</span>
            <span class="font-mono">{formatTimeCompact(previewEndTime)}</span>
            <span class="opacity-60 text-[10px] font-bold ml-1"
                >{formatDuration(previewStartTime, previewEndTime)}</span
            >
        </div>
    </div>
{/if}

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

    /* Edit mode styles */
    .edit-active {
        --edit-border-color: oklch(var(--wa));
    }
</style>
