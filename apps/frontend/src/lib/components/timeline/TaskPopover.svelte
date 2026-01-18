<script lang="ts">
    import {
        QUADRANT_COLORS,
        type Quadrant,
    } from "$lib/components/emotions/emotionData";
    import { OpenMoji } from "$lib/components/ui";
    import { stripHtml } from "$lib/utils";
    import { Check, Clock, FileText, Heart, Sparkles } from "lucide-svelte";
    import { scale } from "svelte/transition";
    import type { TimelineTask } from "./types";

    interface Props {
        task: TimelineTask;
        position: { x: number; y: number };
    }

    let { task, position }: Props = $props();

    // Calculate smart positioning to avoid edge overflow
    const smartPosition = $derived.by(() => {
        if (typeof window === "undefined") {
            return {
                x: position.x,
                y: position.y,
            };
        }

        const POPOVER_WIDTH = 300;
        const MAX_POPOVER_HEIGHT = 400; // Maximum possible height
        const OFFSET = 12; // Distance from cursor
        const PADDING = 8;

        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;

        // Calculate available space
        const spaceRight = viewportWidth - position.x;
        const spaceLeft = position.x;
        const spaceBelow = viewportHeight - position.y;
        const spaceAbove = position.y;

        // Determine if we should flip horizontally
        const shouldFlipX = spaceRight < POPOVER_WIDTH + OFFSET + PADDING;

        // Determine if we should flip vertically
        // Only flip if there's more space above AND not enough space below
        const shouldFlipY = spaceBelow < MAX_POPOVER_HEIGHT + OFFSET + PADDING && spaceAbove > spaceBelow;

        // Calculate position
        let x = shouldFlipX
            ? Math.max(PADDING, position.x - POPOVER_WIDTH - OFFSET)
            : position.x + OFFSET;

        // For Y position, stay close to cursor
        let y = position.y + (shouldFlipY ? -OFFSET : OFFSET);

        return { x, y, shouldFlipY };
    });

    // Check for emotions
    const hasSelectedEmotion = $derived(
        !!task?.emotionId &&
            !!task?.emotionName &&
            !!task?.emotionEmoji &&
            !!task?.emotionQuadrant,
    );
    const hasInferredEmotion = $derived(
        !!task?.inferredEmotionId &&
            !!task?.inferredEmotionName &&
            !!task?.inferredEmotionEmoji &&
            !!task?.inferredEmotionQuadrant,
    );
    const hasAnyEmotion = $derived(hasSelectedEmotion || hasInferredEmotion);

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
        if (mins < 1)
            return `${Math.round((e.getTime() - s.getTime()) / 1000)}s`;
        if (mins < 60) return `${mins}m`;
        const h = Math.floor(mins / 60);
        const m = mins % 60;
        return m > 0 ? `${h}h ${m}m` : `${h}h`;
    }

    const descriptionPreview = $derived(
        task.description
            ? stripHtml(task.description, { newlineSeparator: " · " })
            : null,
    );
</script>

<div
    class="fixed z-[9999] pointer-events-none transition-all duration-150 ease-out {smartPosition.shouldFlipY ? 'origin-bottom' : 'origin-top'}"
    style="top: {smartPosition.y}px; left: {smartPosition.x}px; {smartPosition.shouldFlipY ? 'transform: translateY(-100%);' : ''}"
    in:scale={{ duration: 200, start: 0.9, opacity: 0, easing: (t) => t * (2 - t) }}
    out:scale={{ duration: 150, start: 1, opacity: 0, easing: (t) => t * t }}
>
    <div class="w-[300px] bg-base-100/95 backdrop-blur-xl rounded-2xl shadow-[0_20px_60px_rgba(0,0,0,0.15),0_0_30px_rgba(0,0,0,0.05)] border border-base-200/50 overflow-hidden">
        <!-- Header with category color accent -->
        <div
            class="h-1.5 w-full"
            style="background-color: {task.categoryColor || '#6b7280'};"
        ></div>

        <div class="p-4">
            <!-- Title Section -->
            <div class="flex gap-3 mb-3">
                <div
                    class="w-3 h-3 rounded-full shrink-0 mt-1 ring-2 ring-base-200 shadow-sm"
                    style="background-color: {task.categoryColor || '#6b7280'}"
                ></div>
                <div class="min-w-0 flex-1">
                    <p class="font-bold text-sm leading-snug line-clamp-2">
                        {task.title}
                    </p>
                    {#if task.categoryName}
                        <p
                            class="text-[10px] font-semibold uppercase tracking-wider text-base-content/40 mt-1"
                        >
                            {task.categoryName}
                        </p>
                    {/if}
                </div>
            </div>

            <!-- Description Preview -->
            {#if descriptionPreview}
                <div
                    class="flex items-start gap-2 mb-3 p-2.5 bg-base-200/50 rounded-lg"
                >
                    <FileText
                        class="w-3.5 h-3.5 text-base-content/40 mt-0.5 shrink-0"
                    />
                    <p
                        class="text-xs text-base-content/60 line-clamp-2 leading-relaxed"
                    >
                        {descriptionPreview}
                    </p>
                </div>
            {/if}

            <!-- Emotion Section -->
            {#if hasAnyEmotion}
                <div class="mb-3 p-2.5 bg-base-200/30 rounded-lg space-y-2">
                    <div
                        class="flex items-center gap-1.5 text-[10px] font-semibold uppercase text-base-content/40"
                    >
                        <Heart class="w-3 h-3" />
                        <span>Feeling</span>
                    </div>

                    <!-- Selected Emotion -->
                    {#if hasSelectedEmotion}
                        {@const colors =
                            QUADRANT_COLORS[task?.emotionQuadrant as Quadrant]}
                        <div class="flex items-center gap-2">
                            <div
                                class="flex items-center gap-1.5 px-2 py-1 rounded-lg"
                                style="
                                    background: linear-gradient(135deg, 
                                        color-mix(in srgb, {colors.primary} 15%, transparent) 0%, 
                                        color-mix(in srgb, {colors.secondary} 10%, transparent) 100%);
                                    border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                "
                            >
                                <OpenMoji
                                    emoji={task?.emotionEmoji ?? ""}
                                    size={18}
                                />
                                <div class="flex flex-col">
                                    <span
                                        class="text-xs font-semibold"
                                        style="color: {colors.primary};"
                                        >{task?.emotionName}</span
                                    >
                                    {#if task?.emotionDescription}
                                        <span
                                            class="text-[9px] text-base-content/50 line-clamp-1"
                                            >{task.emotionDescription}</span
                                        >
                                    {/if}
                                </div>
                            </div>
                        </div>
                    {/if}

                    <!-- Inferred Emotion -->
                    {#if hasInferredEmotion}
                        {@const colors =
                            QUADRANT_COLORS[
                                task?.inferredEmotionQuadrant as Quadrant
                            ]}
                        <div class="flex items-center gap-2">
                            <div
                                class="flex items-center gap-1.5 px-2 py-1 rounded-lg opacity-80"
                                style="
                                    background: linear-gradient(135deg, 
                                        color-mix(in srgb, {colors.primary} 8%, transparent) 0%, 
                                        color-mix(in srgb, {colors.secondary} 6%, transparent) 100%);
                                    border: 1.5px dashed color-mix(in srgb, {colors.primary} 40%, transparent);
                                "
                            >
                                <Sparkles
                                    class="w-3 h-3 opacity-60"
                                    style="color: {colors.primary};"
                                />
                                <OpenMoji
                                    emoji={task?.inferredEmotionEmoji ?? ""}
                                    size={16}
                                />
                                <div class="flex flex-col">
                                    <span
                                        class="text-[11px] font-medium"
                                        style="color: {colors.primary};"
                                        >{task?.inferredEmotionName}</span
                                    >
                                    <span
                                        class="text-[9px] text-base-content/40"
                                        >Inferred from reflections</span
                                    >
                                </div>
                            </div>
                        </div>
                    {/if}
                </div>
            {/if}

            <!-- Linked Goals Section -->
            {#if task.linkedGoals && task.linkedGoals.length > 0}
                <div class="mb-3">
                    <div
                        class="flex items-center gap-2 text-[10px] font-semibold uppercase text-base-content/40 mb-2"
                    >
                        <span class="text-sm">🎯</span>
                        <span>Linked Goals</span>
                    </div>
                    <div class="flex flex-col gap-1">
                        {#each task.linkedGoals as goal}
                            {@const unitDisplay =
                                goal.unitSymbol?.replace("units:", "") || ""}
                            {@const formattedQty = goal.quantityValue
                                ? Number.isInteger(goal.quantityValue)
                                    ? goal.quantityValue
                                    : goal.quantityValue.toFixed(1)
                                : null}
                            {@const impactSign =
                                goal.impactType === "negative" ? "−" : "+"}
                            <div
                                class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-xs bg-base-200/40 w-full"
                            >
                                <span class="text-sm">{goal.icon || "🎯"}</span>
                                <span
                                    class="font-medium flex-1 truncate"
                                    style="color: {goal.color || '#8b5cf6'};"
                                >
                                    {goal.title}
                                </span>
                                {#if formattedQty}
                                    <span
                                        class="text-[10px] font-semibold px-1.5 py-0.5 rounded {goal.impactType ===
                                        'positive'
                                            ? 'text-success bg-success/10'
                                            : goal.impactType === 'negative'
                                              ? 'text-error bg-error/10'
                                              : 'text-base-content/60'}"
                                    >
                                        {impactSign}{formattedQty}<span class="ml-0.5">{unitDisplay}</span>
                                    </span>
                                {/if}
                            </div>
                        {/each}
                    </div>
                </div>
            {/if}

            <!-- Time Info -->
            <div
                class="flex items-center justify-between pt-3 border-t border-base-200/50"
            >
                <div
                    class="flex items-center gap-1.5 text-xs text-base-content/50"
                >
                    <Clock class="w-3.5 h-3.5" />
                    <span class="font-medium">
                        {formatTime(task.startTime)} – {formatTime(
                            task.endTime,
                        )}
                    </span>
                </div>
                <span
                    class="font-bold text-xs text-primary bg-primary/10 px-2.5 py-1 rounded-lg"
                >
                    {formatDuration(task.startTime, task.endTime)}
                </span>
            </div>

            <!-- Completed Badge -->
            {#if task.completed}
                <div
                    class="flex items-center gap-1.5 mt-3 text-success bg-success/10 px-3 py-1.5 rounded-lg w-fit"
                >
                    <Check class="w-3.5 h-3.5" strokeWidth={3} />
                    <span class="text-[11px] font-bold uppercase tracking-wide"
                        >Completed</span
                    >
                </div>
            {/if}
        </div>
    </div>
</div>
