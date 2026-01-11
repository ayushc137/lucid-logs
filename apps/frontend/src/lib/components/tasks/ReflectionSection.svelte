<script lang="ts">
import type { TaskItem } from '$lib/api';
import type { Emotion } from '$lib/api/emotions';
import {
	QUADRANT_COLORS,
	type Quadrant,
} from '$lib/components/emotions/emotionData';
import { Card } from '$lib/components/ui';
import OpenMoji from '$lib/components/ui/OpenMoji.svelte';
import { emotionStore } from '$lib/stores/emotions.svelte';
import { Heart, Plus, Sparkles, ThumbsDown, ThumbsUp, X } from 'lucide-svelte';

interface Props {
	positives: TaskItem[];
	negatives: TaskItem[];
	pendingPositiveEmotion: Emotion | null;
	pendingNegativeEmotion: Emotion | null;
	onPositivesChange: (items: TaskItem[]) => void;
	onNegativesChange: (items: TaskItem[]) => void;
	onOpenEmotionForPositive: (index: number) => void;
	onOpenEmotionForNegative: (index: number) => void;
	onOpenEmotionForPendingPositive: () => void;
	onOpenEmotionForPendingNegative: () => void;
	onClearPendingPositiveEmotion: () => void;
	onClearPendingNegativeEmotion: () => void;
}

let {
	positives = [],
	negatives = [],
	pendingPositiveEmotion = null,
	pendingNegativeEmotion = null,
	onPositivesChange,
	onNegativesChange,
	onOpenEmotionForPositive,
	onOpenEmotionForNegative,
	onOpenEmotionForPendingPositive,
	onOpenEmotionForPendingNegative,
	onClearPendingPositiveEmotion,
	onClearPendingNegativeEmotion,
}: Props = $props();

let newPositive = $state('');
let newNegative = $state('');

function addPositive() {
	if (newPositive.trim()) {
		const newItem = {
			text: newPositive.trim(),
			emotion_id: pendingPositiveEmotion?.id,
		};
		onPositivesChange([...positives, newItem]);
		newPositive = '';
		onClearPendingPositiveEmotion();
	}
}

function addNegative() {
	if (newNegative.trim()) {
		const newItem = {
			text: newNegative.trim(),
			emotion_id: pendingNegativeEmotion?.id,
		};
		onNegativesChange([...negatives, newItem]);
		newNegative = '';
		onClearPendingNegativeEmotion();
	}
}

function removePositive(index: number) {
	onPositivesChange(positives.filter((_, i) => i !== index));
}

function removeNegative(index: number) {
	onNegativesChange(negatives.filter((_, i) => i !== index));
}

function clearPositiveEmotion(index: number) {
	onPositivesChange(
		positives.map((p, i) =>
			i === index ? { ...p, emotion_id: undefined } : p,
		),
	);
}

function clearNegativeEmotion(index: number) {
	onNegativesChange(
		negatives.map((n, i) =>
			i === index ? { ...n, emotion_id: undefined } : n,
		),
	);
}
</script>

<Card variant="bordered" class="transition-all duration-300 hover:shadow-lg">
    <div class="flex items-center gap-2 mb-4">
        <Sparkles class="w-4 h-4 opacity-50" />
        <span class="text-xs font-semibold uppercase opacity-50"
            >Reflection</span
        >
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- Positives Section -->
        <div class="flex flex-col h-full">
            <div class="flex items-center gap-2 mb-3">
                <div
                    class="w-6 h-6 rounded-lg bg-success/20 flex items-center justify-center"
                >
                    <ThumbsUp class="w-3.5 h-3.5 text-success" />
                </div>
                <span class="text-sm font-semibold text-success"
                    >What went well?</span
                >
            </div>

            <!-- New item input with emotion selector -->
            <div
                class="rounded-xl border border-success/30 bg-success/5 p-3 space-y-2"
            >
                <!-- Emotion selector -->
                <div class="flex items-center gap-2">
                    {#if pendingPositiveEmotion}
                        {@const colors =
                            QUADRANT_COLORS[
                                pendingPositiveEmotion.quadrant as Quadrant
                            ]}
                        <div
                            class="tooltip tooltip-bottom"
                            data-tip={pendingPositiveEmotion.description}
                        >
                            <button
                                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 hover:scale-105"
                                style="
                                    background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
                                    color: {colors.secondary};
                                    border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                "
                                onclick={onOpenEmotionForPendingPositive}
                            >
                                <span
                                    class="w-6 h-6 rounded-md flex items-center justify-center"
                                    style="background: {colors.gradient};"
                                >
                                    <OpenMoji
                                        emoji={pendingPositiveEmotion.emoji}
                                        alt={pendingPositiveEmotion.name}
                                        size="sm"
                                    />
                                </span>
                                <span class="truncate max-w-[80px]"
                                    >{pendingPositiveEmotion.name}</span
                                >
                                <!-- svelte-ignore a11y_click_events_have_key_events -->
                                <!-- svelte-ignore a11y_no_static_element_interactions -->
                                <span
                                    class="p-0.5 rounded-full hover:bg-base-content/10 transition-colors cursor-pointer"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        onClearPendingPositiveEmotion();
                                    }}
                                    role="button"
                                    tabindex="0"
                                >
                                    <X class="w-3 h-3" />
                                </span>
                            </button>
                        </div>
                    {:else}
                        <button
                            class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-medium bg-success/10 text-success/70 hover:bg-success/20 hover:text-success transition-all duration-200"
                            onclick={onOpenEmotionForPendingPositive}
                        >
                            <Heart class="w-3 h-3" />
                            <span>Add feeling</span>
                        </button>
                    {/if}
                </div>

                <!-- Text input -->
                <div class="flex items-start gap-2">
                    <textarea
                        bind:value={newPositive}
                        placeholder="What went well? What are you grateful for?"
                        class="textarea textarea-sm textarea-bordered flex-1 bg-transparent border-success/20 focus:border-success min-h-[60px] text-sm resize-none"
                        onkeydown={(e) => {
                            if (e.key === "Enter" && !e.shiftKey) {
                                e.preventDefault();
                                addPositive();
                            }
                        }}
                    ></textarea>
                    <button
                        class="btn btn-sm btn-success btn-outline shrink-0 gap-1"
                        onclick={addPositive}
                        disabled={!newPositive.trim()}
                    >
                        <Plus class="w-4 h-4" />
                    </button>
                </div>
            </div>

            <!-- Existing positives list -->
            {#if positives.length > 0}
                <div class="flex flex-col gap-2 mt-3">
                    {#each positives as p, i (i)}
                        {@const hasEmotion = !!p.emotion_id}
                        <div
                            class="rounded-xl bg-gradient-to-r from-success/10 to-success/5 border border-success/20 p-3 group hover:shadow-md transition-all duration-300 animate-fade-in"
                        >
                            <div class="flex items-start gap-3">
                                <ThumbsUp
                                    class="w-4 h-4 text-success shrink-0 mt-0.5"
                                />
                                <div class="flex-1 min-w-0">
                                    <p
                                        class="text-sm leading-relaxed break-words"
                                    >
                                        {p.text}
                                    </p>
                                    <!-- Emotion display -->
                                    <div class="mt-2">
                                        {#if hasEmotion}
                                            {@const emotion = emotionStore.get(
                                                p.emotion_id!,
                                            )}
                                            {#if emotion}
                                                {@const colors =
                                                    QUADRANT_COLORS[
                                                        emotion.quadrant as Quadrant
                                                    ]}
                                                <div
                                                    class="tooltip tooltip-bottom"
                                                    data-tip={emotion.description}
                                                >
                                                    <button
                                                        class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 hover:scale-105"
                                                        style="
                                                            background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
                                                            color: {colors.secondary};
                                                            border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                                        "
                                                        onclick={() =>
                                                            onOpenEmotionForPositive(
                                                                i,
                                                            )}
                                                    >
                                                        <span
                                                            class="w-6 h-6 rounded-md flex items-center justify-center"
                                                            style="background: {colors.gradient};"
                                                        >
                                                            <OpenMoji
                                                                emoji={emotion.emoji}
                                                                alt={emotion.name}
                                                                size="sm"
                                                            />
                                                        </span>
                                                        <span
                                                            class="truncate max-w-[80px]"
                                                            >{emotion.name}</span
                                                        >
                                                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                                                        <!-- svelte-ignore a11y_no_static_element_interactions -->
                                                        <span
                                                            class="p-0.5 rounded-full hover:bg-base-content/10 transition-colors cursor-pointer"
                                                            onclick={(e) => {
                                                                e.stopPropagation();
                                                                clearPositiveEmotion(
                                                                    i,
                                                                );
                                                            }}
                                                            role="button"
                                                            tabindex="0"
                                                        >
                                                            <X
                                                                class="w-3 h-3"
                                                            />
                                                        </span>
                                                    </button>
                                                </div>
                                            {:else}
                                                <span
                                                    class="text-xs text-success/50"
                                                    >Loading...</span
                                                >
                                            {/if}
                                        {:else}
                                            <button
                                                class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] text-base-content/40 hover:text-success hover:bg-success/10 transition-all duration-200"
                                                onclick={() =>
                                                    onOpenEmotionForPositive(i)}
                                            >
                                                <Plus class="w-3 h-3" />
                                                <span>Add feeling</span>
                                            </button>
                                        {/if}
                                    </div>
                                </div>
                                <button
                                    onclick={() => removePositive(i)}
                                    class="btn btn-ghost btn-xs text-base-content/30 hover:text-error hover:bg-error/10 transition-all duration-200 shrink-0 opacity-0 group-hover:opacity-100"
                                >
                                    <X class="w-3.5 h-3.5" />
                                </button>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>

        <!-- Negatives Section -->
        <div class="flex flex-col h-full">
            <div class="flex items-center gap-2 mb-3">
                <div
                    class="w-6 h-6 rounded-lg bg-error/20 flex items-center justify-center"
                >
                    <ThumbsDown class="w-3.5 h-3.5 text-error" />
                </div>
                <span class="text-sm font-semibold text-error"
                    >What could improve?</span
                >
            </div>

            <!-- New item input with emotion selector -->
            <div
                class="rounded-xl border border-error/30 bg-error/5 p-3 space-y-2"
            >
                <!-- Emotion selector -->
                <div class="flex items-center gap-2">
                    {#if pendingNegativeEmotion}
                        {@const colors =
                            QUADRANT_COLORS[
                                pendingNegativeEmotion.quadrant as Quadrant
                            ]}
                        <div
                            class="tooltip tooltip-bottom"
                            data-tip={pendingNegativeEmotion.description}
                        >
                            <button
                                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 hover:scale-105"
                                style="
                                    background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
                                    color: {colors.secondary};
                                    border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                "
                                onclick={onOpenEmotionForPendingNegative}
                            >
                                <span
                                    class="w-6 h-6 rounded-md flex items-center justify-center"
                                    style="background: {colors.gradient};"
                                >
                                    <OpenMoji
                                        emoji={pendingNegativeEmotion.emoji}
                                        alt={pendingNegativeEmotion.name}
                                        size="sm"
                                    />
                                </span>
                                <span class="truncate max-w-[80px]"
                                    >{pendingNegativeEmotion.name}</span
                                >
                                <!-- svelte-ignore a11y_click_events_have_key_events -->
                                <!-- svelte-ignore a11y_no_static_element_interactions -->
                                <span
                                    class="p-0.5 rounded-full hover:bg-base-content/10 transition-colors cursor-pointer"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        onClearPendingNegativeEmotion();
                                    }}
                                    role="button"
                                    tabindex="0"
                                >
                                    <X class="w-3 h-3" />
                                </span>
                            </button>
                        </div>
                    {:else}
                        <button
                            class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-medium bg-error/10 text-error/70 hover:bg-error/20 hover:text-error transition-all duration-200"
                            onclick={onOpenEmotionForPendingNegative}
                        >
                            <Heart class="w-3 h-3" />
                            <span>Add feeling</span>
                        </button>
                    {/if}
                </div>

                <!-- Text input -->
                <div class="flex items-start gap-2">
                    <textarea
                        bind:value={newNegative}
                        placeholder="What could be better? What challenged you?"
                        class="textarea textarea-sm textarea-bordered flex-1 bg-transparent border-error/20 focus:border-error min-h-[60px] text-sm resize-none"
                        onkeydown={(e) => {
                            if (e.key === "Enter" && !e.shiftKey) {
                                e.preventDefault();
                                addNegative();
                            }
                        }}
                    ></textarea>
                    <button
                        class="btn btn-sm btn-error btn-outline shrink-0 gap-1"
                        onclick={addNegative}
                        disabled={!newNegative.trim()}
                    >
                        <Plus class="w-4 h-4" />
                    </button>
                </div>
            </div>

            <!-- Existing negatives list -->
            {#if negatives.length > 0}
                <div class="flex flex-col gap-2 mt-3">
                    {#each negatives as n, i (i)}
                        {@const hasEmotion = !!n.emotion_id}
                        <div
                            class="rounded-xl bg-gradient-to-r from-error/10 to-error/5 border border-error/20 p-3 group hover:shadow-md transition-all duration-300 animate-fade-in"
                        >
                            <div class="flex items-start gap-3">
                                <ThumbsDown
                                    class="w-4 h-4 text-error shrink-0 mt-0.5"
                                />
                                <div class="flex-1 min-w-0">
                                    <p
                                        class="text-sm leading-relaxed break-words"
                                    >
                                        {n.text}
                                    </p>
                                    <!-- Emotion display -->
                                    <div class="mt-2">
                                        {#if hasEmotion}
                                            {@const emotion = emotionStore.get(
                                                n.emotion_id!,
                                            )}
                                            {#if emotion}
                                                {@const colors =
                                                    QUADRANT_COLORS[
                                                        emotion.quadrant as Quadrant
                                                    ]}
                                                <div
                                                    class="tooltip tooltip-bottom"
                                                    data-tip={emotion.description}
                                                >
                                                    <button
                                                        class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 hover:scale-105"
                                                        style="
                                                            background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
                                                            color: {colors.secondary};
                                                            border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                                        "
                                                        onclick={() =>
                                                            onOpenEmotionForNegative(
                                                                i,
                                                            )}
                                                    >
                                                        <span
                                                            class="w-6 h-6 rounded-md flex items-center justify-center"
                                                            style="background: {colors.gradient};"
                                                        >
                                                            <OpenMoji
                                                                emoji={emotion.emoji}
                                                                alt={emotion.name}
                                                                size="sm"
                                                            />
                                                        </span>
                                                        <span
                                                            class="truncate max-w-[80px]"
                                                            >{emotion.name}</span
                                                        >
                                                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                                                        <!-- svelte-ignore a11y_no_static_element_interactions -->
                                                        <span
                                                            class="p-0.5 rounded-full hover:bg-base-content/10 transition-colors cursor-pointer"
                                                            onclick={(e) => {
                                                                e.stopPropagation();
                                                                clearNegativeEmotion(
                                                                    i,
                                                                );
                                                            }}
                                                            role="button"
                                                            tabindex="0"
                                                        >
                                                            <X
                                                                class="w-3 h-3"
                                                            />
                                                        </span>
                                                    </button>
                                                </div>
                                            {:else}
                                                <span
                                                    class="text-xs text-error/50"
                                                    >Loading...</span
                                                >
                                            {/if}
                                        {:else}
                                            <button
                                                class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] text-base-content/40 hover:text-error hover:bg-error/10 transition-all duration-200"
                                                onclick={() =>
                                                    onOpenEmotionForNegative(i)}
                                            >
                                                <Plus class="w-3 h-3" />
                                                <span>Add feeling</span>
                                            </button>
                                        {/if}
                                    </div>
                                </div>
                                <button
                                    onclick={() => removeNegative(i)}
                                    class="btn btn-ghost btn-xs text-base-content/30 hover:text-error hover:bg-error/10 transition-all duration-200 shrink-0 opacity-0 group-hover:opacity-100"
                                >
                                    <X class="w-3.5 h-3.5" />
                                </button>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
    </div>
</Card>
