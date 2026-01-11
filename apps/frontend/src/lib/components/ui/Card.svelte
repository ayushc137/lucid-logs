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

const baseClass = 'card bg-base-100';
const shadowClass = $derived(shadow ? 'shadow-lg' : 'shadow-sm');
const variantClasses: Record<Variant, string> = {
	default: '',
	bordered: 'border border-base-200',
	compact: '',
};
const bodyClasses: Record<Variant, string> = {
	default: 'p-4 lg:p-5',
	bordered: 'p-4 lg:p-5',
	compact: 'p-3',
};
</script>

<div class={cn(baseClass, shadowClass, variantClasses[variant], className)}>
    <div class="card-body {bodyClasses[variant]}">
        {@render children()}
    </div>
</div>
