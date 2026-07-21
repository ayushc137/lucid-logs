<script lang="ts">
import type { Snippet } from 'svelte';
import IconBox from './IconBox.svelte';

interface Props {
	/** Main value to display */
	value: string | number;
	/** Label text */
	label: string;
	/** Optional unit/suffix */
	unit?: string;
	/** Semantic color */
	color?:
		| 'primary'
		| 'secondary'
		| 'accent'
		| 'info'
		| 'success'
		| 'warning'
		| 'error';
	/** Icon slot */
	icon?: Snippet;
	/** Custom class */
	class?: string;
}

let {
	value,
	label,
	unit = '',
	color = 'primary',
	icon,
	class: className = '',
}: Props = $props();
</script>

<div class="flex items-center gap-3.5 {className}">
    {#if icon}
        <IconBox size="md" variant="subtle" {color}>
            {@render icon()}
        </IconBox>
    {/if}
    <div class="min-w-0">
        <p class="text-xs font-medium text-base-content/50 leading-tight">{label}</p>
        <p class="text-2xl font-bold tracking-tight leading-tight stat-value mt-0.5">
            {value}{#if unit}<span class="text-sm font-medium text-base-content/50 ml-0.5"
                    >{unit}</span
                >{/if}
        </p>
    </div>
</div>
