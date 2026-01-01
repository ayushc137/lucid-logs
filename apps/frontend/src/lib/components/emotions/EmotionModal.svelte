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
                class="flex items-start justify-between gap-3 px-5 py-4 border-b border-base-300 shrink-0 min-h-20"
            >
                {#if displayEmotion}
                    {@const colors = QUADRANT_COLORS[displayEmotion.quadrant]}
                    {@const dots = getIndicatorDots(displayEmotion)}
                    {@const isSelected =
                        selectedEmotion?.id === displayEmotion.id}

                    <div class="flex items-start gap-3.5 flex-1">
                        <div
                            class="w-13 h-13 rounded-xl flex items-center justify-center shrink-0 transition-all duration-300"
                            style="background: {colors.gradient};"
                        >
                            <span
                                class="font-['OpenMojiColor'] text-2xl leading-none"
                                >{displayEmotion.emoji}</span
                            >
                        </div>

                        <div class="flex-1 min-w-0">
                            <div class="flex items-center gap-2 mb-1">
                                <h3 class="text-lg font-semibold m-0">
                                    {displayEmotion.name}
                                </h3>
                                {#if isSelected}
                                    <span
                                        class="text-[0.6rem] px-2 py-0.5 bg-primary text-white rounded-lg font-medium uppercase tracking-wide"
                                        >Selected</span
                                    >
                                {/if}
                            </div>
                            <p class="m-0 text-sm opacity-70 leading-relaxed">
                                {displayEmotion.description}
                            </p>
                            {#if showIndicators && dots.length > 0}
                                <div class="flex gap-1.5 mt-2 flex-wrap">
                                    {#each dots as d}
                                        <span
                                            class="text-[0.6rem] px-2 py-0.5 rounded-lg font-medium"
                                            style="background: color-mix(in srgb, {d.color} 15%, transparent); color: color-mix(in srgb, {d.color} 80%, var(--bc, #333));"
                                            >{d.label}</span
                                        >
                                    {/each}
                                </div>
                            {/if}
                        </div>
                    </div>
                {:else}
                    <div class="flex items-start gap-3.5 flex-1">
                        <div
                            class="w-13 h-13 rounded-xl flex items-center justify-center shrink-0 bg-gradient-to-br from-indigo-200 to-indigo-300 text-indigo-500"
                        >
                            <Heart size={22} strokeWidth={2} />
                        </div>
                        <div class="flex-1 min-w-0">
                            <h3 class="text-lg font-semibold m-0">
                                How are you feeling?
                            </h3>
                            <p class="m-0 text-sm opacity-70 leading-relaxed">
                                Pan around the grid or hover over an emotion to
                                see details
                            </p>
                        </div>
                    </div>
                {/if}

                <button
                    class="btn btn-ghost btn-sm btn-circle"
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
                    {showIndicators}
                />
            </div>

            <!-- Footer -->
            <div
                class="flex items-center gap-2 px-4 py-3 border-t border-base-300 shrink-0"
            >
                {#if selectedEmotion}
                    <button
                        class="btn btn-ghost btn-sm"
                        onclick={clearSelection}
                    >
                        Clear
                    </button>
                {/if}
                <div class="flex-1"></div>
                <button class="btn btn-ghost btn-sm" onclick={handleClose}>
                    Cancel
                </button>
                <button
                    class="btn btn-primary btn-sm"
                    onclick={handleConfirm}
                    disabled={!selectedEmotion}
                >
                    Confirm
                </button>
            </div>
        </div>
        <form method="dialog" class="modal-backdrop bg-base-content/20">
            <button onclick={handleClose}>close</button>
        </form>
    </dialog>
{/if}
