<script lang="ts">
import { cn } from '$lib/utils';
import { BarChart3 } from 'lucide-svelte';

interface Props {
	value: number;
	class?: string;
}

let { value = $bindable(5), class: className }: Props = $props();

const priorityLabels = [
	{ value: 1, label: 'Lowest' },
	{ value: 2, label: 'Low' },
	{ value: 3, label: 'Medium' },
	{ value: 4, label: 'High' },
	{ value: 5, label: 'Highest' },
];

const currentLabel = $derived(
	priorityLabels.find((p) => p.value === value)?.label || 'Medium',
);
</script>

<div class={cn("space-y-2", className)}>
    <!-- Header matching Category style -->
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
            <BarChart3 class="w-4 h-4 opacity-50" />
            <span class="text-xs font-semibold uppercase opacity-50"
                >Priority</span
            >
        </div>
        <span class="text-xs font-medium opacity-70">{currentLabel}</span>
    </div>

    <!-- Slider -->
    <input
        type="range"
        min="1"
        max="5"
        bind:value
        class="range range-sm range-primary w-full"
        step="1"
    />

    <!-- Labels -->
    <div class="flex justify-between text-[10px] opacity-40 px-1">
        {#each priorityLabels as p}
            <span>{p.label}</span>
        {/each}
    </div>
</div>
