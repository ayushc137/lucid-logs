<script lang="ts">
import { cn } from '$lib/utils';
import { ArrowUpDown, ChevronDown, ChevronUp } from 'lucide-svelte';

interface Props {
	/** Column label */
	label: string;
	/** Field name for sorting */
	field: string;
	/** Current sort field */
	sortField?: string;
	/** Current sort direction */
	sortDirection?: 'asc' | 'desc';
	/** Sort handler */
	onSort?: (field: string) => void;
	/** Custom class */
	class?: string;
}

let {
	label,
	field,
	sortField = '',
	sortDirection = 'asc',
	onSort,
	class: className = '',
}: Props = $props();

const isActive = $derived(sortField === field);

function handleClick() {
	onSort?.(field);
}
</script>

<th class={className}>
    <button
        class={cn(
            "flex items-center gap-2 font-semibold transition-colors",
            isActive ? "text-primary" : "hover:text-primary",
        )}
        onclick={handleClick}
    >
        {label}
        {#if isActive}
            {#if sortDirection === "asc"}
                <ChevronUp class="w-4 h-4" />
            {:else}
                <ChevronDown class="w-4 h-4" />
            {/if}
        {:else}
            <ArrowUpDown class="w-3 h-3 opacity-40" />
        {/if}
    </button>
</th>
