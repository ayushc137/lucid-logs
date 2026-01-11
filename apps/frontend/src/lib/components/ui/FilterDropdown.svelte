<script lang="ts">
import { cn } from '$lib/utils';
import { ChevronDown, X } from 'lucide-svelte';

interface Option {
	value: string;
	label: string;
	color?: string;
	badge?: string;
	badgeClass?: string;
	icon?: typeof ChevronDown;
}

interface Props {
	options: Option[];
	value: string;
	onchange?: (value: string) => void;
	placeholder?: string;
	label?: string;
	size?: 'sm' | 'md';
	showColorDot?: boolean;
	allLabel?: string;
	class?: string;
}

let {
	options,
	value = $bindable(),
	onchange,
	placeholder = 'Select...',
	label,
	size = 'sm',
	showColorDot = false,
	allLabel = 'All',
	class: className,
}: Props = $props();

let isOpen = $state(false);
let dropdownRef = $state<HTMLDivElement | null>(null);

const selectedOption = $derived(options.find((o) => o.value === value));

const displayLabel = $derived(
	value === 'all' ? allLabel : (selectedOption?.label ?? placeholder),
);

function handleSelect(optionValue: string) {
	value = optionValue;
	onchange?.(optionValue);
	isOpen = false;
}

function handleClickOutside(e: MouseEvent) {
	if (dropdownRef && !dropdownRef.contains(e.target as Node)) {
		isOpen = false;
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
            onclick={() => (isOpen = !isOpen)}
        >
            <div class="flex items-center gap-2 truncate">
                {#if showColorDot && selectedOption?.color}
                    <span
                        class="w-3 h-3 rounded-full flex-shrink-0"
                        style="background-color: {selectedOption.color};"
                    ></span>
                {/if}
                {#if selectedOption?.badge}
                    <span
                        class={cn("badge badge-sm", selectedOption.badgeClass)}
                    >
                        {selectedOption.badge}
                    </span>
                {/if}
                <span class={cn("truncate", !selectedOption && "opacity-50")}>
                    {displayLabel}
                </span>
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
                class="absolute z-50 top-full left-0 right-0 mt-1 py-1 bg-base-100 border border-base-300 rounded-lg shadow-lg max-h-60 overflow-y-auto animate-dropdown"
            >
                {#each options as option (option.value)}
                    <button
                        type="button"
                        class={cn(
                            "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                            value === option.value
                                ? "bg-primary/10 text-primary"
                                : "hover:bg-base-200",
                        )}
                        onclick={() => handleSelect(option.value)}
                    >
                        {#if showColorDot && option.color}
                            <span
                                class="w-3 h-3 rounded-full flex-shrink-0"
                                style="background-color: {option.color};"
                            ></span>
                        {:else if showColorDot && option.value === "all"}
                            <span
                                class="w-3 h-3 rounded-full flex-shrink-0 bg-base-300"
                            ></span>
                        {/if}

                        {#if option.badge}
                            <span
                                class={cn("badge badge-sm", option.badgeClass)}
                            >
                                {option.badge}
                            </span>
                        {/if}

                        <span class="truncate flex-1">{option.label}</span>

                        {#if value === option.value}
                            <span class="text-primary">✓</span>
                        {/if}
                    </button>
                {/each}
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
