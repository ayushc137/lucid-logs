<script lang="ts">
import type { Snippet } from 'svelte';

interface Props {
	/** Maximum width class for the inner action row */
	maxWidth?: string;
	/** Actions rendered on the left side */
	left?: Snippet;
	/** Actions rendered on the right side */
	right?: Snippet;
}

let { maxWidth = 'max-w-6xl', left, right }: Props = $props();
</script>

<div
	class="fixed bottom-0 right-0 z-30 border-t border-base-300 bg-base-100/95 px-4 py-3 shadow-lg backdrop-blur-md sm:px-6 pb-[calc(0.75rem+env(safe-area-inset-bottom,0px))] left-0 lg:left-64"
>
	<div class="{maxWidth} mx-auto flex items-center justify-between gap-2 sm:gap-4">
		<div class="flex items-center gap-3">
			{#if left}
				{@render left()}
			{/if}
		</div>
		<div class="flex items-center gap-2">
			{#if right}
				{@render right()}
			{/if}
		</div>
	</div>
</div>
