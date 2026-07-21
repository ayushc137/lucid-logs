<script lang="ts">
import { Filter } from 'lucide-svelte';
import { scale } from 'svelte/transition';

type Category = {
	name: string;
	color: string;
	count: number;
};

let {
	categories = [],
	totalTaskCount = 0,
	filteredTaskCount = 0,
	uncategorizedCount = 0,
	selectedCategory = null,
	onCategoryChange,
}: {
	categories: Category[];
	totalTaskCount: number;
	filteredTaskCount: number;
	uncategorizedCount: number;
	selectedCategory: string | null;
	onCategoryChange?: (category: string | null) => void;
} = $props();

let showDropdown = $state(false);
let containerRef: HTMLDivElement = null!;

$effect(() => {
	const handleClickOutside = (e: MouseEvent) => {
		if (containerRef && !containerRef.contains(e.target as Node)) {
			showDropdown = false;
		}
	};
	document.addEventListener('click', handleClickOutside);
	return () => document.removeEventListener('click', handleClickOutside);
});

function selectCategory(category: string | null) {
	onCategoryChange?.(category);
	showDropdown = false;
}

const selectedCat = $derived(
	categories.find((c) => c.name === selectedCategory),
);
</script>

<div class="relative" bind:this={containerRef}>
    <button
        class="btn btn-sm gap-2 transition-all {selectedCategory
            ? 'btn-soft-primary'
            : 'btn-ghost hover:bg-base-200'}"
        onclick={(e) => {
            e.stopPropagation();
            showDropdown = !showDropdown;
        }}
    >
        <Filter class="w-4 h-4" />
        {#if selectedCategory}
            {#if selectedCat}
                <div
                    class="w-2.5 h-2.5 rounded-full"
                    style="background-color: {selectedCat.color}"
                ></div>
                <span class="max-w-[120px] truncate">{selectedCat.name}</span>
            {:else if selectedCategory === "__uncategorized__"}
                <span>Uncategorized</span>
            {/if}
            <span class="text-xs opacity-60">({filteredTaskCount})</span>
        {:else}
            <span>All Categories</span>
            <span class="text-xs opacity-60">({totalTaskCount})</span>
        {/if}
    </button>

    {#if showDropdown}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
            class="absolute top-full left-1/2 -translate-x-1/2 mt-2 bg-base-100 rounded-xl shadow-2xl border border-base-200 z-50 min-w-[220px] max-w-[300px]"
            in:scale={{ duration: 150, start: 0.95 }}
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.key === "Escape" && (showDropdown = false)}
            role="dialog"
            tabindex="-1"
        >
            <div class="p-2">
                <div
                    class="text-[10px] font-bold text-base-content/50 uppercase tracking-wider px-2 py-1 flex justify-between"
                >
                    <span>Filter by Category</span>
                    {#if selectedCategory}
                        <button
                            class="text-error hover:underline text-[10px] font-medium normal-case"
                            onclick={() => selectCategory(null)}
                        >
                            Clear
                        </button>
                    {/if}
                </div>
                <div
                    class="max-h-[300px] overflow-y-auto custom-scrollbar space-y-0.5 mt-1"
                >
                    <button
                        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg transition-colors text-left {selectedCategory ===
                        null
                            ? 'bg-primary/10 text-primary font-semibold'
                            : 'hover:bg-base-200'}"
                        onclick={() => selectCategory(null)}
                    >
                        <div
                            class="w-3 h-3 rounded-full bg-gradient-to-br from-primary to-secondary"
                        ></div>
                        <span class="flex-1 text-sm">All Categories</span>
                        <span class="text-xs opacity-60">{totalTaskCount}</span>
                    </button>
                    {#each categories as cat}
                        <button
                            class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg transition-colors text-left {selectedCategory ===
                            cat.name
                                ? 'bg-primary/10 font-semibold'
                                : 'hover:bg-base-200'}"
                            style={selectedCategory === cat.name
                                ? `color: ${cat.color}`
                                : ""}
                            onclick={() => selectCategory(cat.name)}
                        >
                            <div
                                class="w-3 h-3 rounded-full shrink-0"
                                style="background-color: {cat.color}"
                            ></div>
                            <span class="flex-1 text-sm truncate"
                                >{cat.name}</span
                            >
                            <span class="text-xs opacity-60">{cat.count}</span>
                        </button>
                    {/each}
                    {#if uncategorizedCount > 0}
                        <button
                            class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg transition-colors text-left {selectedCategory ===
                            '__uncategorized__'
                                ? 'bg-base-200 font-semibold'
                                : 'hover:bg-base-200'}"
                            onclick={() => selectCategory("__uncategorized__")}
                        >
                            <div
                                class="w-3 h-3 rounded-full bg-base-300 shrink-0"
                            ></div>
                            <span class="flex-1 text-sm text-base-content/60"
                                >Uncategorized</span
                            >
                            <span class="text-xs opacity-60"
                                >{uncategorizedCount}</span
                            >
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .custom-scrollbar::-webkit-scrollbar {
        width: 6px;
    }
    .custom-scrollbar::-webkit-scrollbar-thumb {
        background: oklch(var(--bc) / 0.12);
        border-radius: 10px;
    }
    .custom-scrollbar::-webkit-scrollbar-thumb:hover {
        background: oklch(var(--bc) / 0.2);
    }
    .custom-scrollbar::-webkit-scrollbar-track {
        background: transparent;
    }
</style>
