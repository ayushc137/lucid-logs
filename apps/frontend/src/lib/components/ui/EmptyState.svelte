<script lang="ts">
import { Plus } from 'lucide-svelte';
import type { Snippet } from 'svelte';

interface Props {
	/** Main title */
	title: string;
	/** Description text */
	description: string;
	/** Show action button */
	showButton?: boolean;
	/** Button label */
	buttonLabel?: string;
	/** Button click handler */
	onButtonClick?: () => void;
	/** Custom class for container */
	class?: string;
	/** Icon slot */
	icon?: Snippet;
}

let {
	title,
	description,
	showButton = true,
	buttonLabel = 'Get Started',
	onButtonClick,
	class: className = '',
	icon,
}: Props = $props();
</script>

<div
    class="rounded-2xl border-2 border-dashed border-base-300 bg-base-100/50 {className}"
>
    <div class="py-14 px-6 text-center max-w-sm mx-auto">
        {#if icon}
            <div
                class="w-14 h-14 rounded-2xl bg-base-200 text-base-content/40 flex items-center justify-center mx-auto mb-4"
            >
                {@render icon()}
            </div>
        {/if}
        <h2 class="text-lg font-semibold tracking-tight">{title}</h2>
        <p class="text-sm text-base-content/55 mt-1.5 leading-relaxed">
            {description}
        </p>
        {#if showButton}
            <button
                class="btn btn-primary btn-sm mt-5 gap-1.5 rounded-lg"
                onclick={onButtonClick}
            >
                <Plus class="w-4 h-4" strokeWidth={2.5} />
                {buttonLabel}
            </button>
        {/if}
    </div>
</div>
