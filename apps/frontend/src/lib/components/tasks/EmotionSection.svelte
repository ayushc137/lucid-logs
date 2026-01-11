<script lang="ts">
    import type { Emotion, InferredEmotion } from "$lib/api/emotions";
    import {
        QUADRANT_COLORS,
        QUADRANT_META,
        type Quadrant,
    } from "$lib/components/emotions/emotionData";
    import OpenMoji from "$lib/components/ui/OpenMoji.svelte";
    import { Card } from "$lib/components/ui";
    import {
        Heart,
        Plus,
        X,
        Sparkles,
        CircleAlert,
        ChevronRight,
        Info,
    } from "lucide-svelte";

    interface Props {
        selectedEmotion: Emotion | null;
        inferredEmotion: InferredEmotion | null;
        inferredEmotionFull: Emotion | null;
        inferringEmotion: boolean;
        inferenceError: string | null;
        onOpenEmotionForTask: () => void;
        onOpenEmotionForSuggested: () => void;
        onClearTaskEmotion: () => void;
    }

    let {
        selectedEmotion = null,
        inferredEmotion = null,
        inferredEmotionFull = null,
        inferringEmotion = false,
        inferenceError = null,
        onOpenEmotionForTask,
        onOpenEmotionForSuggested,
        onClearTaskEmotion,
    }: Props = $props();
</script>

<Card
    variant="bordered"
    class="transition-all duration-300 hover:shadow-lg overflow-visible"
>
    <div class="flex items-center gap-2 mb-4">
        <Heart class="w-4 h-4 opacity-50" />
        <span class="text-xs font-semibold uppercase opacity-50">Feeling</span>
    </div>

    <!-- Two-column display: Selected | Inferred -->
    <div class="grid grid-cols-1 gap-3">
        <!-- User Selected Emotion -->
        <div class="relative">
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

                <div
                    class="tooltip tooltip-bottom w-full"
                    data-tip={selectedEmotion.description}
                >
                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                    <div
                        class="w-full rounded-xl p-3 flex items-center gap-3 transition-all duration-300 hover:shadow-lg cursor-pointer group/card relative overflow-hidden border"
                        style="
                            background: linear-gradient(135deg, 
                                color-mix(in srgb, {colors.primary} 12%, var(--b1)) 0%, 
                                color-mix(in srgb, {colors.secondary} 8%, var(--b1)) 100%);
                            border-color: color-mix(in srgb, {colors.primary} 25%, transparent);
                        "
                        onclick={onOpenEmotionForTask}
                        onkeydown={(e) =>
                            e.key === "Enter" && onOpenEmotionForTask()}
                        role="button"
                        tabindex="0"
                    >
                        <!-- Background glow on hover -->
                        <div
                            class="absolute inset-0 opacity-0 group-hover/card:opacity-100 transition-opacity duration-500 pointer-events-none"
                            style="background: radial-gradient(circle at 30% 50%, {colors.glow}, transparent 60%);"
                        ></div>

                        <!-- Emoji -->
                        <div
                            class="w-11 h-11 rounded-xl flex items-center justify-center shrink-0 shadow-md group-hover/card:scale-110 transition-transform duration-300 relative z-10"
                            style="background: {colors.gradient};"
                        >
                            <OpenMoji
                                emoji={selectedEmotion.emoji}
                                alt={selectedEmotion.name}
                                size="md"
                            />
                        </div>

                        <!-- Info -->
                        <div class="flex-1 min-w-0 text-left relative z-10">
                            <div class="flex items-center gap-2">
                                <span
                                    class="text-sm font-bold truncate"
                                    style="color: {colors.secondary};"
                                >
                                    {selectedEmotion.name}
                                </span>
                                <span
                                    class="badge badge-xs font-bold uppercase tracking-wider px-1.5 shrink-0"
                                    style="background: {colors.primary}; color: white; border: none;"
                                >
                                    ✓
                                </span>
                            </div>
                            <div class="text-[10px] opacity-60 mt-0.5">
                                {meta?.energyLabel} Energy • {meta?.pleasantnessLabel}
                            </div>
                        </div>

                        <!-- Clear button -->
                        <button
                            class="p-1.5 rounded-full hover:bg-base-content/10 transition-colors relative z-10 opacity-0 group-hover/card:opacity-100"
                            onclick={(e) => {
                                e.stopPropagation();
                                onClearTaskEmotion();
                            }}
                            title="Remove"
                        >
                            <X class="w-4 h-4 opacity-60 hover:opacity-100" />
                        </button>
                    </div>
                </div>
            {:else}
                <!-- Empty State -->
                <button
                    class="w-full rounded-xl border-2 border-dashed border-base-300 p-4 flex items-center justify-center gap-3 hover:border-primary hover:bg-primary/5 transition-all duration-300 text-base-content/50 hover:text-primary group/empty"
                    onclick={onOpenEmotionForTask}
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

        <!-- Inferred Emotion -->
        {#if inferringEmotion || inferredEmotion || inferenceError}
            <div class="relative">
                <div
                    class="text-[10px] uppercase font-semibold text-primary/70 mb-2 flex items-center gap-1.5"
                >
                    <Sparkles
                        class="w-3 h-3 text-primary {inferringEmotion
                            ? 'animate-pulse'
                            : ''}"
                    />
                    <span>Suggested</span>
                    <div
                        class="tooltip tooltip-right cursor-help normal-case"
                        data-tip="Based on your reflections"
                    >
                        <Info class="w-3 h-3 text-primary/50" />
                    </div>
                    {#if inferringEmotion}
                        <span
                            class="loading loading-spinner loading-xs text-primary"
                        ></span>
                    {/if}
                </div>

                {#if inferenceError}
                    <!-- Error State -->
                    <div
                        class="w-full rounded-xl p-2.5 flex items-center gap-2 bg-error/10 border border-error/30"
                    >
                        <CircleAlert class="w-4 h-4 text-error shrink-0" />
                        <span class="text-xs text-error">{inferenceError}</span>
                    </div>
                {:else if inferringEmotion}
                    <!-- Loading State -->
                    <div
                        class="w-full rounded-xl p-2.5 flex items-center gap-3 bg-base-200 border border-base-300 animate-pulse"
                    >
                        <div class="w-9 h-9 rounded-lg bg-base-300"></div>
                        <div class="flex-1">
                            <div
                                class="h-3 bg-base-300 rounded w-24 mb-1"
                            ></div>
                            <div class="h-2 bg-base-300 rounded w-32"></div>
                        </div>
                    </div>
                {:else if inferredEmotion}
                    {@const inferredQuadrant = (inferredEmotionFull?.quadrant ||
                        inferredEmotion.quadrant ||
                        "green") as Quadrant}
                    {@const colors = QUADRANT_COLORS[inferredQuadrant]}
                    {@const meta = QUADRANT_META[inferredQuadrant]}

                    <div
                        class="tooltip tooltip-bottom w-full"
                        data-tip={inferredEmotionFull?.description ||
                            "Calculated from your reflections"}
                    >
                        <button
                            class="w-full rounded-xl p-2.5 flex items-center gap-3 transition-all duration-300 hover:shadow-md cursor-pointer group/inferred relative overflow-hidden"
                            style="
                                background: linear-gradient(135deg,
                                    color-mix(in srgb, {colors.primary} 6%, var(--b2)) 0%,
                                    color-mix(in srgb, {colors.secondary} 4%, var(--b2)) 100%);
                                border: 1px dashed color-mix(in srgb, {colors.primary} 35%, transparent);
                            "
                            onclick={onOpenEmotionForSuggested}
                        >
                            <!-- Emoji -->
                            <div
                                class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 shadow-sm group-hover/inferred:scale-105 transition-transform duration-300"
                                style="background: {colors.gradient}; opacity: 0.9;"
                            >
                                {#if inferredEmotionFull}
                                    <OpenMoji
                                        emoji={inferredEmotionFull.emoji}
                                        alt={inferredEmotionFull?.name ||
                                            "Inferred Emotion"}
                                        size="md"
                                    />
                                {:else}
                                    <Sparkles class="w-4 h-4 text-white/80" />
                                {/if}
                            </div>

                            <!-- Info -->
                            <div class="flex-1 min-w-0 text-left">
                                <div class="flex items-center gap-2">
                                    <span
                                        class="text-sm font-semibold opacity-80"
                                    >
                                        {inferredEmotionFull?.name ||
                                            "Suggesting..."}
                                    </span>
                                </div>
                                <div class="text-[10px] opacity-50 mt-0.5">
                                    {meta?.energyLabel} Energy • {meta?.pleasantnessLabel}
                                </div>
                            </div>

                            <ChevronRight
                                class="w-4 h-4 text-primary/40 group-hover/inferred:text-primary transition-colors"
                            />
                        </button>
                    </div>
                {/if}
            </div>
        {/if}
    </div>
</Card>
