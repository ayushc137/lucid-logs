<script lang="ts">
import { cn } from '$lib/utils';
import type { Snippet } from 'svelte';
import type { HTMLButtonAttributes } from 'svelte/elements';

type Variant = 'primary' | 'neutral' | 'ghost' | 'outline' | 'error' | 'error-outline';
type Size = 'xs' | 'sm' | 'md' | 'lg';

interface Props extends Omit<HTMLButtonAttributes, 'class'> {
	/** Visual variant. One primary action per view; everything else ghost/outline. */
	variant?: Variant;
	/** Button size */
	size?: Size;
	/** Icon-only square button */
	iconOnly?: boolean;
	/** Full-width on mobile (sm and up: auto width) */
	blockMobile?: boolean;
	/** Additional classes */
	class?: string;
	children: Snippet;
}

let {
	variant = 'primary',
	size = 'md',
	iconOnly = false,
	blockMobile = false,
	class: className = '',
	children,
	...rest
}: Props = $props();

const variantClasses: Record<Variant, string> = {
	primary: 'btn-primary',
	neutral: 'btn-neutral',
	ghost: 'btn-ghost',
	outline: 'btn-outline',
	error: 'btn-error',
	'error-outline': 'btn-error btn-outline',
};

const sizeClasses: Record<Size, string> = {
	xs: 'btn-xs',
	sm: 'btn-sm',
	md: '',
	lg: 'btn-lg',
};
</script>

<button
	class={cn(
		'btn transition-ui active:scale-[0.98]',
		variantClasses[variant],
		sizeClasses[size],
		iconOnly && 'btn-square',
		blockMobile && 'w-full sm:w-auto',
		className,
	)}
	{...rest}
>
	{@render children()}
</button>
