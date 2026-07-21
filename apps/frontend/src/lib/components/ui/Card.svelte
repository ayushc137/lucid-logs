<script lang="ts">
import { cn } from '$lib/utils';
import type { Snippet } from 'svelte';

type Variant = 'default' | 'bordered' | 'compact';

interface Props {
	/** Card variant style */
	variant?: Variant;
	/** Add shadow */
	shadow?: boolean;
	/** Custom class */
	class?: string;
	/** Card content */
	children: Snippet;
}

let {
	variant = 'default',
	shadow = true,
	class: className = '',
	children,
}: Props = $props();

const shadowClass = $derived(
	shadow ? 'shadow-sm hover:shadow-md transition-shadow' : '',
);
const variantClasses: Record<Variant, string> = {
	default: 'border border-base-300/60',
	bordered: 'border border-base-300',
	compact: 'border border-base-300/60',
};
const bodyClasses: Record<Variant, string> = {
	default: 'p-4 sm:p-5',
	bordered: 'p-4 sm:p-5',
	compact: 'p-3',
};
</script>

<div
    class={cn(
        "card bg-base-100 rounded-2xl",
        shadowClass,
        variantClasses[variant],
        className
    )}
>
    <div class="card-body {bodyClasses[variant]} gap-0">
        {@render children()}
    </div>
</div>
