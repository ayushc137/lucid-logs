<script lang="ts">
import { cn } from '$lib/utils';
import { Check, ChevronDown } from 'lucide-svelte';

type PriorityValue = 'all' | 'high' | 'medium' | 'low';

interface PriorityOption {
	value: PriorityValue;
	label: string;
	badgeClass: string;
	range: string;
}

interface Props {
	value: PriorityValue;
	onchange?: (value: PriorityValue) => void;
	label?: string;
	size?: 'sm' | 'md';
	showAllOption?: boolean;
	class?: string;
}

let {
	value = $bindable('all'),
	onchange,
	label,
	size = 'sm',
	showAllOption = true,
	class: className,
}: Props = $props();

let isOpen = $state(false);
let dropdownRef = $state<HTMLDivElement | null>(null);

const allOptions: PriorityOption[] = [
	{
		value: 'all',
		label: 'All Priorities',
		badgeClass: 'badge-ghost',
		range: '',
	},
	{
		value: 'high',
		label: 'High',
		badgeClass: 'badge-error',
		range: '7-10',
	},
	{
		value: 'medium',
		label: 'Medium',
		badgeClass: 'badge-warning',
		range: '4-6',
	},
	{ value: 'low', label: 'Low', badgeClass: 'badge-info', range: '1-3' },
];

const priorityOptions = $derived(
	showAllOption ? allOptions : allOptions.filter((o) => o.value !== 'all'),
);

const selectedOption = $derived(
	priorityOptions.find((o) => o.value === value) ?? priorityOptions[0],
);

function handleSelect(optionValue: PriorityValue) {
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
            <div class="flex items-center gap-2">
                <span class={cn("badge badge-sm", selectedOption.badgeClass)}>
                    {selectedOption.label}
                </span>
                {#if selectedOption.range}
                    <span class="text-xs opacity-40"
                        >({selectedOption.range})</span
                    >
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
                class="absolute z-50 top-full left-0 right-0 mt-1 py-1 bg-base-100 border border-base-300 rounded-lg shadow-lg overflow-hidden animate-dropdown"
            >
                {#each priorityOptions as option (option.value)}
                    <button
                        type="button"
                        class={cn(
                            "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                            value === option.value
                                ? "bg-primary/10"
                                : "hover:bg-base-200",
                        )}
                        onclick={() => handleSelect(option.value)}
                    >
                        <span class={cn("badge badge-sm", option.badgeClass)}>
                            {option.label}
                        </span>
                        {#if option.range}
                            <span class="text-xs opacity-40"
                                >({option.range})</span
                            >
                        {/if}
                        <span class="flex-1"></span>
                        {#if value === option.value}
                            <Check class="w-3.5 h-3.5 text-primary" />
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
