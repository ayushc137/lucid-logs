<script lang="ts">
    /**
     * Emotion Selection - Modal Only
     */
    import { PageHeader } from "$lib/components/ui";
    import { Sparkles, Heart } from "lucide-svelte";
    import { EmotionModal } from "$lib/components/emotions";
    import type { Emotion } from "$lib/components/emotions/emotionData";
    import { QUADRANT_COLORS } from "$lib/components/emotions/emotionData";

    let selectedEmotion = $state<Emotion | null>(null);
    let showModal = $state(false);

    function openPicker() {
        showModal = true;
    }

    function handleSelect(emotion: Emotion) {
        selectedEmotion = emotion;
        console.log("Selected:", emotion.name);
    }
</script>

<svelte:head>
    <title>Emotion Selection | Lucid Logs</title>
    <meta name="description" content="Track your emotions throughout the day" />
</svelte:head>

<div class="emotions-page">
    <PageHeader
        title="Emotion Tracking"
        subtitle="How are you feeling right now?"
        icon={Sparkles}
    />

    <!-- Emotion Picker Trigger -->
    <div class="picker-section">
        <button class="emotion-picker-btn" onclick={openPicker}>
            {#if selectedEmotion}
                {@const colors = QUADRANT_COLORS[selectedEmotion.quadrant]}
                <div class="picker-blob" style="background: {colors.gradient};">
                    <span class="picker-emoji">{selectedEmotion.emoji}</span>
                </div>
                <div class="picker-text">
                    <span class="picker-label">Currently feeling</span>
                    <span class="picker-name">{selectedEmotion.name}</span>
                    <span class="picker-desc"
                        >{selectedEmotion.description}</span
                    >
                </div>
            {:else}
                <div class="picker-blob empty">
                    <Heart class="w-7 h-7" />
                </div>
                <div class="picker-text">
                    <span class="picker-label">No emotion selected</span>
                    <span class="picker-name">Tap to select how you feel</span>
                </div>
            {/if}
            <span class="picker-arrow">→</span>
        </button>
    </div>

    <!-- Modal -->
    <EmotionModal
        bind:open={showModal}
        bind:selectedEmotion
        onSelect={handleSelect}
    />
</div>

<style>
    .emotions-page {
        padding: 24px;
        max-width: 600px;
        margin: 0 auto;
    }

    .picker-section {
        margin-top: 32px;
    }

    .emotion-picker-btn {
        width: 100%;
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 20px 24px;
        background: white;
        border: 1px solid rgba(0, 0, 0, 0.08);
        border-radius: 20px;
        cursor: pointer;
        text-align: left;
        transition: all 0.25s ease;
    }

    .emotion-picker-btn:hover {
        border-color: rgba(0, 0, 0, 0.12);
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
        transform: translateY(-2px);
    }

    .picker-blob {
        width: 56px;
        height: 56px;
        border-radius: 16px;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }

    .picker-blob.empty {
        background: linear-gradient(135deg, #e0e7ff, #c7d2fe);
        color: #6366f1;
    }

    .picker-emoji {
        font-family: "OpenMoji-Color", "OpenMoji", sans-serif;
        font-size: 1.8rem;
        line-height: 1;
    }

    .picker-text {
        flex: 1;
        min-width: 0;
    }

    .picker-label {
        display: block;
        font-size: 0.72rem;
        color: #9ca3af;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        margin-bottom: 2px;
    }

    .picker-name {
        display: block;
        font-size: 1.1rem;
        font-weight: 600;
        color: #1f2937;
    }

    .picker-desc {
        display: block;
        font-size: 0.82rem;
        color: #6b7280;
        margin-top: 4px;
        line-height: 1.4;
    }

    .picker-arrow {
        font-size: 1.4rem;
        color: #d1d5db;
        transition:
            transform 0.2s,
            color 0.2s;
    }

    .emotion-picker-btn:hover .picker-arrow {
        transform: translateX(4px);
        color: #9ca3af;
    }

    /* Dark mode */
    :global([data-theme="dark"]) .emotion-picker-btn {
        background: #1e293b;
        border-color: rgba(255, 255, 255, 0.08);
    }

    :global([data-theme="dark"]) .picker-name {
        color: #f3f4f6;
    }

    :global([data-theme="dark"]) .picker-desc {
        color: #9ca3af;
    }

    :global([data-theme="dark"]) .picker-arrow {
        color: #4b5563;
    }

    @media (max-width: 480px) {
        .emotions-page {
            padding: 16px;
        }

        .emotion-picker-btn {
            padding: 16px 20px;
        }

        .picker-blob {
            width: 48px;
            height: 48px;
        }

        .picker-emoji {
            font-size: 1.5rem;
        }
    }
</style>
