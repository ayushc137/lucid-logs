<script lang="ts">
/**
 * EmotionBadge Component
 * - Displays emotion with emoji, name, and optional "inferred" indicator
 * - Compact badge format for task cards and listings
 * - Supports both actual and inferred emotions
 */
import OpenMoji from '$lib/components/ui/OpenMoji.svelte';
import { Sparkles, X } from 'lucide-svelte';
import { QUADRANT_COLORS, type Quadrant } from './emotionData';

interface Props {
	/** Emotion name to display */
	name: string;
	/** Emoji hex code or character */
	emoji: string;
	/** Quadrant for color theming */
	quadrant: Quadrant;
	/** Whether this is an inferred emotion (AI suggested) */
	isInferred?: boolean;
	/** Size variant */
	size?: 'xs' | 'sm' | 'md';
	/** Whether to show a remove button */
	removable?: boolean;
	/** Callback when badge is clicked */
	onclick?: () => void;
	/** Callback when remove is clicked */
	onRemove?: () => void;
	/** Additional classes */
	class?: string;
}

let {
	name,
	emoji,
	quadrant,
	isInferred = false,
	size = 'sm',
	removable = false,
	onclick,
	onRemove,
	class: className = '',
}: Props = $props();

const colors = $derived(QUADRANT_COLORS[quadrant]);

// Size mappings
const sizeConfig = {
	xs: {
		badge: 'px-1.5 py-0.5 text-[10px] gap-1',
		emoji: 'xs' as const,
		icon: 'w-2.5 h-2.5',
	},
	sm: {
		badge: 'px-2 py-1 text-xs gap-1.5',
		emoji: 'sm' as const,
		icon: 'w-3 h-3',
	},
	md: {
		badge: 'px-2.5 py-1.5 text-sm gap-2',
		emoji: 'md' as const,
		icon: 'w-3.5 h-3.5',
	},
};

const config = $derived(sizeConfig[size]);
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_interactive_supports_focus -->
<div
    class="inline-flex items-center rounded-full font-medium transition-all duration-200 {config.badge} {className}"
    class:cursor-pointer={onclick}
    class:hover:scale-105={onclick}
    style="
        background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
        color: {colors.secondary};
        border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
    "
    role={onclick ? "button" : "presentation"}
    {onclick}
>
    <!-- Emoji -->
    <span
        class="flex items-center justify-center rounded-full"
        style="background: {colors.gradient}; padding: 2px;"
    >
        <OpenMoji {emoji} alt={name} size={config.emoji} />
    </span>

    <!-- Name -->
    <span class="font-semibold truncate max-w-[100px]">{name}</span>

    <!-- Inferred indicator -->
    {#if isInferred}
        <span
            class="flex items-center gap-0.5 px-1 py-0.5 rounded-full text-[0.65rem] font-bold uppercase tracking-wider"
            style="background: color-mix(in srgb, {colors.accent} 40%, transparent);"
            title="AI inferred from your reflection"
        >
            <Sparkles class={config.icon} />
            <span class="hidden sm:inline">Inferred</span>
        </span>
    {/if}

    <!-- Remove button -->
    {#if removable && onRemove}
        <button
            class="ml-0.5 p-0.5 rounded-full hover:bg-base-content/10 transition-colors"
            onclick={(e) => {
                e.stopPropagation();
                onRemove?.();
            }}
            title="Remove emotion"
        >
            <X class={config.icon} />
        </button>
    {/if}
</div>
