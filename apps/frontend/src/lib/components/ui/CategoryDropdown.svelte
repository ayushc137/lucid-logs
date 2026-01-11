<script lang="ts">
import type { Category } from '$lib/api/categories';
import { cn } from '$lib/utils';
import { Check, ChevronDown, Plus, Search, X } from 'lucide-svelte';

interface Props {
	categories: Category[];
	value: string | undefined;
	onchange?: (value: string | undefined) => void;
	label?: string;
	size?: 'sm' | 'md';
	showSearch?: boolean;
	showAllOption?: boolean;
	allLabel?: string;
	showNoCategoryOption?: boolean;
	noCategoryLabel?: string;
	placeholder?: string;
	showCreateButton?: boolean;
	onCreate?: () => void;
	class?: string;
}

let {
	categories,
	value = $bindable(),
	onchange,
	label,
	size = 'sm',
	showSearch = true,
	showAllOption = false,
	allLabel = 'All Categories',
	showNoCategoryOption = false,
	noCategoryLabel = 'Uncategorized',
	placeholder = 'Select category...',
	showCreateButton = false,
	onCreate,
	class: className,
}: Props = $props();

let isOpen = $state(false);
let search = $state('');
let dropdownRef = $state<HTMLDivElement | null>(null);
let searchInputRef = $state<HTMLInputElement | null>(null);

const selectedCategory = $derived(categories.find((c) => c.id === value));

const filteredCategories = $derived(
	search.trim()
		? categories.filter((c) =>
				c.name.toLowerCase().includes(search.toLowerCase()),
			)
		: categories,
);

const displayLabel = $derived(() => {
	if (showAllOption && value === 'all') return allLabel;
	if (showNoCategoryOption && value === 'none') return noCategoryLabel;
	if (selectedCategory) return selectedCategory.name;
	if (!value) return placeholder;
	return placeholder;
});

function handleSelect(categoryId: string | undefined) {
	value = categoryId;
	onchange?.(categoryId);
	isOpen = false;
	search = '';
}

function handleClickOutside(e: MouseEvent) {
	if (dropdownRef && !dropdownRef.contains(e.target as Node)) {
		isOpen = false;
		search = '';
	}
}

function handleOpen() {
	isOpen = !isOpen;
	if (isOpen && showSearch) {
		setTimeout(() => searchInputRef?.focus(), 50);
	}
}

$effect(() => {
	if (isOpen) {
		document.addEventListener('click', handleClickOutside);
		return () => document.removeEventListener('click', handleClickOutside);
	}
});
</script>

<div class={cn("form-control", className)}>
    {#if label}
        <div class="label py-1">
            <span class="label-text text-xs font-medium opacity-70"
                >{label}</span
            >
        </div>
    {/if}

    <div class="relative" bind:this={dropdownRef}>
        <!-- Trigger Button -->
        <button
            type="button"
            class={cn(
                "flex items-center justify-between w-full gap-2 px-3 rounded-lg border border-base-300 bg-base-100 transition-all hover:border-base-content/30",
                size === "sm" ? "h-8 text-sm" : "h-10",
                isOpen && "border-primary ring-1 ring-primary/20",
            )}
            onclick={handleOpen}
        >
            <div class="flex items-center gap-2 truncate">
                {#if selectedCategory}
                    <span
                        class="w-3 h-3 rounded-full flex-shrink-0"
                        style="background-color: {selectedCategory.color};"
                    ></span>
                    <span class="truncate">{selectedCategory.name}</span>
                {:else if showAllOption && value === "all"}
                    <span class="w-3 h-3 rounded-full flex-shrink-0 bg-base-300"
                    ></span>
                    <span class="truncate">{allLabel}</span>
                {:else if showNoCategoryOption && value === "none"}
                    <span
                        class="w-3 h-3 rounded-full flex-shrink-0 bg-base-content/20 border border-dashed border-base-content/40"
                    ></span>
                    <span class="truncate">{noCategoryLabel}</span>
                {:else}
                    <span class="opacity-50 truncate">{placeholder}</span>
                {/if}
            </div>
            <ChevronDown
                class={cn(
                    "w-4 h-4 opacity-50 flex-shrink-0 transition-transform",
                    isOpen && "rotate-180",
                )}
            />
        </button>

        <!-- Dropdown Menu -->
        {#if isOpen}
            <div
                class="absolute z-50 top-full left-0 right-0 mt-1 bg-base-100 border border-base-300 rounded-lg shadow-lg overflow-hidden animate-dropdown"
            >
                <!-- Search -->
                {#if showSearch && categories.length > 5}
                    <div class="p-2 border-b border-base-200">
                        <div class="relative">
                            <Search
                                class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 opacity-40"
                            />
                            <input
                                bind:this={searchInputRef}
                                type="text"
                                placeholder="Search categories..."
                                class="input input-sm input-bordered w-full pl-8 pr-8"
                                bind:value={search}
                                onclick={(e) => e.stopPropagation()}
                            />
                            {#if search}
                                <button
                                    class="absolute right-2 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        search = "";
                                    }}
                                >
                                    <X class="w-3 h-3" />
                                </button>
                            {/if}
                        </div>
                    </div>
                {/if}

                <!-- Options -->
                <div class="max-h-52 overflow-y-auto py-1">
                    <!-- All Option -->
                    {#if showAllOption}
                        <button
                            type="button"
                            class={cn(
                                "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                                value === "all"
                                    ? "bg-primary/10 text-primary"
                                    : "hover:bg-base-200",
                            )}
                            onclick={() => handleSelect("all" as any)}
                        >
                            <span
                                class="w-3 h-3 rounded-full flex-shrink-0 bg-base-300"
                            ></span>
                            <span class="truncate flex-1">{allLabel}</span>
                            {#if value === "all"}
                                <Check class="w-3.5 h-3.5 text-primary" />
                            {/if}
                        </button>
                    {/if}

                    <!-- Uncategorized Option (when showNoCategoryOption is true) -->
                    {#if showNoCategoryOption}
                        <button
                            type="button"
                            class={cn(
                                "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                                value === "none"
                                    ? "bg-primary/10 text-primary"
                                    : "hover:bg-base-200",
                            )}
                            onclick={() => handleSelect("none" as any)}
                        >
                            <span
                                class="w-3 h-3 rounded-full flex-shrink-0 bg-base-content/20 border border-dashed border-base-content/40"
                            ></span>
                            <span class="truncate flex-1 opacity-70"
                                >{noCategoryLabel}</span
                            >
                            {#if value === "none"}
                                <Check class="w-3.5 h-3.5 text-primary" />
                            {/if}
                        </button>
                    {/if}

                    <!-- No Category Option (when not showing all option) -->
                    {#if !showAllOption}
                        <button
                            type="button"
                            class={cn(
                                "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                                value === undefined
                                    ? "bg-primary/10 text-primary"
                                    : "hover:bg-base-200",
                            )}
                            onclick={() => handleSelect(undefined)}
                        >
                            <span
                                class="w-3 h-3 rounded-full flex-shrink-0 bg-base-300"
                            ></span>
                            <span class="truncate flex-1 opacity-60"
                                >No category</span
                            >
                            {#if value === undefined}
                                <Check class="w-3.5 h-3.5 text-primary" />
                            {/if}
                        </button>
                    {/if}

                    <!-- Category List -->
                    {#each filteredCategories as category (category.id)}
                        <button
                            type="button"
                            class={cn(
                                "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                                value === category.id
                                    ? "bg-primary/10 text-primary"
                                    : "hover:bg-base-200",
                            )}
                            onclick={() => handleSelect(category.id)}
                        >
                            <span
                                class="w-3 h-3 rounded-full flex-shrink-0"
                                style="background-color: {category.color};"
                            ></span>
                            <span class="truncate flex-1">{category.name}</span>
                            {#if value === category.id}
                                <Check class="w-3.5 h-3.5 text-primary" />
                            {/if}
                        </button>
                    {:else}
                        {#if search && categories.length > 0}
                            <div
                                class="px-3 py-4 text-center text-sm opacity-50"
                            >
                                No matching categories
                            </div>
                        {/if}
                    {/each}
                </div>

                <!-- Create Button -->
                {#if showCreateButton && onCreate}
                    <div class="border-t border-base-200 p-1">
                        <button
                            type="button"
                            class="w-full px-3 py-2 text-left flex items-center gap-2 text-sm text-primary hover:bg-base-200 rounded transition-colors"
                            onclick={() => {
                                isOpen = false;
                                onCreate();
                            }}
                        >
                            <Plus class="w-4 h-4" />
                            Create new category
                        </button>
                    </div>
                {/if}
            </div>
        {/if}
    </div>
</div>

<style>
    @keyframes dropdown-in {
        from {
            opacity: 0;
            transform: translateY(-4px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .animate-dropdown {
        animation: dropdown-in 0.15s ease-out;
    }
</style>
