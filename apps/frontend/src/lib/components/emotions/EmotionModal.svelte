<script lang="ts">
    /**
     * Emotion Selector Modal
     * - Clean header with description
     * - OpenMojiColor font
     * - Tailwind CSS styling
     */
    import EmotionGrid from "./EmotionGrid.svelte";
    import type { Emotion } from "./emotionData";
    import { QUADRANT_COLORS } from "./emotionData";
    import { X, Heart } from "lucide-svelte";

    interface Props {
        open?: boolean;
        selectedEmotion?: Emotion | null;
        onSelect?: (emotion: Emotion) => void;
        onClose?: () => void;
    }

    let {
        open = $bindable(false),
        selectedEmotion = $bindable(null),
        onSelect,
        onClose,
    }: Props = $props();

    let showIndicators = $state(false);
    let hoveredEmotion = $state<Emotion | null>(null);

    const displayEmotion = $derived(hoveredEmotion || selectedEmotion);

    function handleSelect(emotion: Emotion) {
        selectedEmotion = emotion;
    }

    function handleHoveredChange(emotion: Emotion | null) {
        hoveredEmotion = emotion;
    }

    function handleClose() {
        open = false;
        onClose?.();
    }

    function handleConfirm() {
        if (selectedEmotion) {
            onSelect?.(selectedEmotion);
        }
        open = false;
        onClose?.();
    }

    function clearSelection() {
        selectedEmotion = null;
    }

    function getIndicatorDots(
        emotion: Emotion,
    ): { color: string; label: string }[] {
        const dots: { color: string; label: string }[] = [];
        if (emotion.dominance > 0.7)
            dots.push({ color: "#fbbf24", label: "Empowered" });
        else if (emotion.dominance < -0.5)
            dots.push({ color: "#a78bfa", label: "Vulnerable" });
        if (emotion.social > 0.6)
            dots.push({ color: "#60a5fa", label: "Social" });
        if (emotion.certainty < -0.3)
            dots.push({ color: "#9ca3af", label: "Uncertain" });
        if (emotion.intensity > 0.85)
            dots.push({ color: "#f87171", label: "Intense" });
        return dots;
    }
</script>

{#if open}
    <dialog class="modal modal-open">
        <div
            class="modal-box max-w-5xl h-[85vh] max-h-[750px] p-0 flex flex-col overflow-hidden"
        >
            <!-- Header -->
            <div
                class="flex items-start justify-between gap-4 px-5 py-4 border-b border-base-300 shrink-0 min-h-20 bg-gradient-to-r from-base-200/50 to-transparent"
            >
                {#if displayEmotion}
                    {@const colors = QUADRANT_COLORS[displayEmotion.quadrant]}
                    {@const dots = getIndicatorDots(displayEmotion)}
                    {@const isSelected =
                        selectedEmotion?.id === displayEmotion.id}

                    <div class="flex items-start gap-4 flex-1">
                        <div
                            class="w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 transition-all duration-300 shadow-lg"
                            style="background: {colors.gradient};"
                        >
                            <span
                                class="font-['OpenMojiColor'] text-2xl leading-none"
                                >{displayEmotion.emoji}</span
                            >
                        </div>

                        <div class="flex-1 min-w-0">
                            <div
                                class="flex items-center gap-2 mb-1.5 flex-wrap"
                            >
                                <h3
                                    class="text-xl font-bold m-0 tracking-tight"
                                >
                                    {displayEmotion.name}
                                </h3>
                                {#if isSelected}
                                    <span
                                        class="text-[0.6rem] px-2.5 py-1 bg-primary text-primary-content rounded-full font-semibold uppercase tracking-wide shadow-sm"
                                        >✓ Selected</span
                                    >
                                {/if}
                                {#if showIndicators}
                                    {#each dots as d}
                                        <span
                                            class="text-[0.55rem] px-1.5 py-0.5 rounded-full font-medium"
                                            style="background: color-mix(in srgb, {d.color} 15%, transparent); color: color-mix(in srgb, {d.color} 80%, var(--bc, #333));"
                                            >{d.label}</span
                                        >
                                    {/each}
                                {/if}
                            </div>
                            <p
                                class="m-0 text-sm text-base-content/70 leading-relaxed"
                            >
                                {displayEmotion.description}
                            </p>
                        </div>
                    </div>
                {:else}
                    <div class="flex items-start gap-4 flex-1">
                        <div
                            class="w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-primary/20 via-secondary/20 to-accent/20 border border-base-300/50"
                        >
                            <Heart
                                size={24}
                                strokeWidth={1.5}
                                class="text-primary/60"
                            />
                        </div>
                        <div class="flex-1 min-w-0">
                            <h3 class="text-xl font-bold m-0 tracking-tight">
                                How are you feeling?
                            </h3>
                            <p
                                class="m-0 text-sm text-base-content/60 leading-relaxed mt-1"
                            >
                                Explore the grid below. Hover or tap an emotion
                                to preview, then click to select.
                            </p>
                        </div>
                    </div>
                {/if}

                <button
                    class="btn btn-ghost btn-sm btn-circle hover:bg-base-300/50"
                    onclick={handleClose}
                >
                    <X class="w-5 h-5" />
                </button>
            </div>

            <!-- Grid -->
            <div class="flex-1 overflow-hidden p-2">
                <EmotionGrid
                    bind:selectedEmotion
                    onSelect={handleSelect}
                    onHoveredChange={handleHoveredChange}
                    bind:showIndicators
                />
            </div>

            <!-- Footer -->
            <div
                class="flex items-center justify-between gap-3 px-5 py-3.5 border-t border-base-300 shrink-0 bg-base-200/30"
            >
                <div class="flex items-center gap-2">
                    {#if selectedEmotion}
                        {@const colors =
                            QUADRANT_COLORS[selectedEmotion.quadrant]}
                        <div
                            class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-base-100 border border-base-300"
                        >
                            <div
                                class="w-5 h-5 rounded-full flex items-center justify-center text-xs"
                                style="background: {colors.gradient};"
                            >
                                <span
                                    class="font-['OpenMojiColor'] text-[0.7rem]"
                                    >{selectedEmotion.emoji}</span
                                >
                            </div>
                            <span class="text-sm font-medium"
                                >{selectedEmotion.name}</span
                            >
                            <button
                                class="ml-1 p-0.5 rounded-full hover:bg-base-300 transition-colors"
                                onclick={clearSelection}
                                title="Reset selection"
                            >
                                <X
                                    class="w-3.5 h-3.5 opacity-60 hover:opacity-100"
                                />
                            </button>
                        </div>
                    {:else}
                        <span class="text-sm text-base-content/50 italic"
                            >No emotion selected</span
                        >
                    {/if}
                </div>

                <div class="flex items-center gap-2">
                    <button class="btn btn-ghost btn-sm" onclick={handleClose}>
                        Cancel
                    </button>
                    <button
                        class="btn btn-primary btn-sm min-w-[100px]"
                        onclick={handleConfirm}
                        disabled={!selectedEmotion}
                    >
                        {#if selectedEmotion}
                            Confirm
                        {:else}
                            Select One
                        {/if}
                    </button>
                </div>
            </div>
        </div>
        <form method="dialog" class="modal-backdrop bg-base-content/20">
            <button onclick={handleClose}>close</button>
        </form>
    </dialog>
{/if}
