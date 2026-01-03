<script lang="ts">
    /**
     * EmotionSelector Component
     * - Displays selected and inferred emotions side by side
     * - Shows emotion with emoji, quadrant color, and clear labels
     * - Hover tooltips with quadrant details
     * - Clean, professional design with fluid animations
     */
    import OpenMoji from "$lib/components/ui/OpenMoji.svelte";
    import {
        QUADRANT_COLORS,
        QUADRANT_META,
        type Quadrant,
    } from "./emotionData";
    import { getIndicatorDots } from "./emotionUtils";
    import type { Emotion, InferredEmotion } from "$lib/api/emotions";
    import {
        Heart,
        Plus,
        Sparkles,
        X,
        Info,
        ArrowUp,
        ArrowDown,
        ArrowLeft,
        ArrowRight,
    } from "lucide-svelte";
    import { cn } from "$lib/utils";

    interface Props {
        /** User-selected emotion */
        selectedEmotion: Emotion | null;
        /** AI-inferred emotion from reflections */
        inferredEmotion: InferredEmotion | null;
        /** Callback when user wants to select/change emotion */
        onSelect: () => void;
        /** Callback when user clears the selected emotion */
        onClear: () => void;
        /** Additional classes */
        class?: string;
    }

    let {
        selectedEmotion,
        inferredEmotion,
        onSelect,
        onClear,
        class: className = "",
    }: Props = $props();

    // Hover state for tooltips
    let hoveredQuadrant = $state<Quadrant | null>(null);
    let tooltipTimeoutId: ReturnType<typeof setTimeout>;

    function getQuadrantIcon(quadrant: Quadrant | string) {
        const meta = QUADRANT_META[quadrant as Quadrant];
        const isHigh = meta?.energyLabel === "High";
        const isPleasant = meta?.pleasantnessLabel === "Pleasant";

        if (isHigh && isPleasant)
            return { energy: ArrowUp, pleasantness: ArrowRight };
        if (isHigh && !isPleasant)
            return { energy: ArrowUp, pleasantness: ArrowLeft };
        if (!isHigh && isPleasant)
            return { energy: ArrowDown, pleasantness: ArrowRight };
        return { energy: ArrowDown, pleasantness: ArrowLeft };
    }

    function handleMouseEnter(quadrant: Quadrant) {
        clearTimeout(tooltipTimeoutId);
        hoveredQuadrant = quadrant;
    }

    function handleMouseLeave() {
        tooltipTimeoutId = setTimeout(() => {
            hoveredQuadrant = null;
        }, 150);
    }
</script>

<div class={cn("space-y-4", className)}>
    <!-- Header -->
    <div class="flex items-center gap-2">
        <Heart class="w-4 h-4 opacity-50" />
        <span class="text-xs font-semibold uppercase opacity-50">Emotion</span>
    </div>

    <!-- Main Emotion Display Grid -->
    <div class="grid grid-cols-1 gap-3">
        <!-- Selected Emotion -->
        <div class="relative group">
            <div
                class="text-[10px] uppercase font-semibold text-base-content/40 mb-2 flex items-center gap-1.5"
            >
                <span>Your Selection</span>
            </div>

            {#if selectedEmotion}
                {@const colors =
                    QUADRANT_COLORS[selectedEmotion.quadrant as Quadrant]}
                {@const meta =
                    QUADRANT_META[selectedEmotion.quadrant as Quadrant]}
                {@const dots = getIndicatorDots(selectedEmotion)}
                {@const icons = getQuadrantIcon(selectedEmotion.quadrant)}

                <button
                    class="w-full rounded-xl p-3 flex items-center gap-3 transition-all duration-300 hover:shadow-lg cursor-pointer group/card relative overflow-hidden"
                    style="
                        background: linear-gradient(135deg, 
                            color-mix(in srgb, {colors.primary} 15%, var(--b1)) 0%, 
                            color-mix(in srgb, {colors.secondary} 10%, var(--b1)) 100%);
                        border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                    "
                    onclick={onSelect}
                    onmouseenter={() =>
                        handleMouseEnter(selectedEmotion.quadrant as Quadrant)}
                    onmouseleave={handleMouseLeave}
                >
                    <!-- Background glow effect -->
                    <div
                        class="absolute inset-0 opacity-0 group-hover/card:opacity-100 transition-opacity duration-500"
                        style="background: radial-gradient(circle at 50% 50%, {colors.glow}, transparent 70%);"
                    ></div>

                    <!-- Emoji Container -->
                    <div
                        class="w-12 h-12 rounded-xl flex items-center justify-center shrink-0 shadow-md group-hover/card:scale-110 transition-transform duration-300 relative z-10"
                        style="background: {colors.gradient};"
                    >
                        <OpenMoji
                            emoji={selectedEmotion.emoji}
                            alt={selectedEmotion.name}
                            size="md"
                            class="drop-shadow-sm"
                        />
                    </div>

                    <!-- Info -->
                    <div class="flex-1 min-w-0 text-left relative z-10">
                        <div class="flex items-center gap-2 flex-wrap">
                            <span
                                class="text-base font-bold truncate"
                                style="color: {colors.secondary};"
                            >
                                {selectedEmotion.name}
                            </span>
                            <span
                                class="badge badge-xs font-bold uppercase tracking-wider px-1.5"
                                style="background: {colors.primary}; color: white; border: none;"
                            >
                                Selected
                            </span>
                        </div>
                        <!-- Quadrant info -->
                        <div
                            class="flex items-center gap-2 mt-1 text-xs opacity-70"
                        >
                            <svelte:component
                                this={icons.energy}
                                class="w-3 h-3"
                            />
                            <span>{meta?.energyLabel} Energy</span>
                            <span class="opacity-30">•</span>
                            <svelte:component
                                this={icons.pleasantness}
                                class="w-3 h-3"
                            />
                            <span>{meta?.pleasantnessLabel}</span>
                        </div>
                        <!-- Indicator dots -->
                        {#if dots.length > 0}
                            <div class="flex gap-1 mt-1.5">
                                {#each dots as d}
                                    <span
                                        class="text-[10px] px-1.5 py-0.5 rounded-full font-medium"
                                        style="background: color-mix(in srgb, {d.color} 20%, transparent); color: {d.color};"
                                        >{d.label}</span
                                    >
                                {/each}
                            </div>
                        {/if}
                    </div>

                    <!-- Clear button -->
                    <button
                        class="p-1.5 rounded-full hover:bg-base-content/10 transition-colors relative z-10 opacity-0 group-hover/card:opacity-100"
                        onclick={(e) => {
                            e.stopPropagation();
                            onClear();
                        }}
                        title="Remove emotion"
                    >
                        <X class="w-4 h-4 opacity-60 hover:opacity-100" />
                    </button>
                </button>

                <!-- Quadrant Detail Tooltip on Hover -->
                {#if hoveredQuadrant === selectedEmotion.quadrant}
                    <div
                        class="absolute left-0 right-0 top-full mt-2 z-50 p-3 rounded-xl bg-base-100 border border-base-300 shadow-xl animate-fade-in"
                    >
                        <div class="flex items-center gap-2 mb-2">
                            <div
                                class="w-4 h-4 rounded-full"
                                style="background: {colors.gradient};"
                            ></div>
                            <span class="text-sm font-bold">{meta?.label}</span>
                        </div>
                        <p class="text-xs opacity-70 mb-2">
                            {#if selectedEmotion.quadrant === "yellow"}
                                High energy, pleasant feelings like excitement,
                                enthusiasm, and joy.
                            {:else if selectedEmotion.quadrant === "green"}
                                Calm, pleasant feelings like contentment, peace,
                                and relaxation.
                            {:else if selectedEmotion.quadrant === "red"}
                                High energy, challenging feelings like
                                frustration, anxiety, or anger.
                            {:else}
                                Low energy, challenging feelings like sadness,
                                exhaustion, or apathy.
                            {/if}
                        </p>
                        <div
                            class="flex gap-3 text-[10px] uppercase opacity-50"
                        >
                            <span class="flex items-center gap-1">
                                <svelte:component
                                    this={icons.energy}
                                    class="w-3 h-3"
                                />
                                {meta?.energyLabel} Energy
                            </span>
                            <span class="flex items-center gap-1">
                                <svelte:component
                                    this={icons.pleasantness}
                                    class="w-3 h-3"
                                />
                                {meta?.pleasantnessLabel}
                            </span>
                        </div>
                    </div>
                {/if}
            {:else}
                <!-- Empty State - Select Emotion Button -->
                <button
                    class="w-full rounded-xl border-2 border-dashed border-base-300 p-4 flex items-center justify-center gap-3 hover:border-primary hover:bg-primary/5 transition-all duration-300 text-base-content/50 hover:text-primary group/empty"
                    onclick={onSelect}
                >
                    <div
                        class="w-10 h-10 rounded-xl bg-base-200 flex items-center justify-center group-hover/empty:bg-primary/20 transition-colors duration-300"
                    >
                        <Heart
                            class="w-5 h-5 group-hover/empty:scale-110 transition-transform duration-300"
                        />
                    </div>
                    <div class="text-left">
                        <span class="text-sm font-semibold">Select Emotion</span
                        >
                        <p class="text-xs opacity-60">How are you feeling?</p>
                    </div>
                    <Plus
                        class="w-5 h-5 ml-auto opacity-40 group-hover/empty:opacity-100 transition-opacity"
                    />
                </button>
            {/if}
        </div>

        <!-- Inferred Emotion (shown when available) -->
        {#if inferredEmotion}
            {@const inferredQuadrant =
                (inferredEmotion.closest_emotion_quadrant as Quadrant) ||
                "green"}
            {@const colors = QUADRANT_COLORS[inferredQuadrant]}
            {@const meta = QUADRANT_META[inferredQuadrant]}
            {@const icons = getQuadrantIcon(inferredQuadrant)}

            <div class="relative">
                <div
                    class="text-[10px] uppercase font-semibold text-primary/60 mb-2 flex items-center gap-1.5"
                >
                    <Sparkles class="w-3 h-3 text-primary" />
                    <span>AI Inferred</span>
                    <div
                        class="tooltip tooltip-right cursor-help"
                        data-tip="Calculated from emotions in your reflections"
                    >
                        <Info class="w-3 h-3 text-primary/50" />
                    </div>
                </div>

                <button
                    class="w-full rounded-xl p-3 flex items-center gap-3 transition-all duration-300 hover:shadow-md cursor-pointer group/inferred relative overflow-hidden"
                    style="
                        background: linear-gradient(135deg, 
                            color-mix(in srgb, {colors.primary} 8%, var(--b2)) 0%, 
                            color-mix(in srgb, {colors.secondary} 5%, var(--b2)) 100%);
                        border: 1px dashed color-mix(in srgb, {colors.primary} 40%, transparent);
                    "
                    onclick={onSelect}
                    onmouseenter={() => handleMouseEnter(inferredQuadrant)}
                    onmouseleave={handleMouseLeave}
                >
                    <!-- Sparkle decoration -->
                    <div class="absolute top-2 right-2 opacity-30">
                        <Sparkles class="w-4 h-4 text-primary animate-pulse" />
                    </div>

                    <!-- Emoji Container -->
                    <div
                        class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 shadow-sm group-hover/inferred:scale-105 transition-transform duration-300"
                        style="background: {colors.gradient}; opacity: 0.85;"
                    >
                        <OpenMoji
                            emoji={inferredEmotion.closest_emotion_emoji ||
                                "1F914"}
                            alt={inferredEmotion.closest_emotion_name}
                            size="sm"
                            class="drop-shadow-sm"
                        />
                    </div>

                    <!-- Info -->
                    <div class="flex-1 min-w-0 text-left">
                        <div class="flex items-center gap-2">
                            <span class="text-sm font-semibold opacity-80">
                                {inferredEmotion.closest_emotion_name}
                            </span>
                            <span
                                class="badge badge-xs font-medium uppercase tracking-wider px-1.5 bg-primary/20 text-primary border-none"
                            >
                                Inferred
                            </span>
                        </div>
                        <!-- Quadrant info -->
                        <div
                            class="flex items-center gap-2 mt-0.5 text-[10px] opacity-50"
                        >
                            <svelte:component
                                this={icons.energy}
                                class="w-2.5 h-2.5"
                            />
                            <span>{meta?.energyLabel}</span>
                            <span class="opacity-30">•</span>
                            <svelte:component
                                this={icons.pleasantness}
                                class="w-2.5 h-2.5"
                            />
                            <span>{meta?.pleasantnessLabel}</span>
                        </div>
                    </div>
                </button>

                <!-- Quadrant Detail Tooltip on Hover -->
                {#if hoveredQuadrant === inferredQuadrant && !selectedEmotion}
                    <div
                        class="absolute left-0 right-0 top-full mt-2 z-50 p-3 rounded-xl bg-base-100 border border-base-300 shadow-xl animate-fade-in"
                    >
                        <div class="flex items-center gap-2 mb-2">
                            <div
                                class="w-4 h-4 rounded-full"
                                style="background: {colors.gradient};"
                            ></div>
                            <span class="text-sm font-bold">{meta?.label}</span>
                        </div>
                        <p class="text-xs opacity-70">
                            {#if inferredQuadrant === "yellow"}
                                High energy, pleasant feelings like excitement,
                                enthusiasm, and joy.
                            {:else if inferredQuadrant === "green"}
                                Calm, pleasant feelings like contentment, peace,
                                and relaxation.
                            {:else if inferredQuadrant === "red"}
                                High energy, challenging feelings like
                                frustration, anxiety, or anger.
                            {:else}
                                Low energy, challenging feelings like sadness,
                                exhaustion, or apathy.
                            {/if}
                        </p>
                    </div>
                {/if}
            </div>
        {/if}
    </div>
</div>

<style>
    @keyframes fade-in {
        from {
            opacity: 0;
            transform: translateY(-4px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .animate-fade-in {
        animation: fade-in 0.2s ease-out;
    }
</style>
