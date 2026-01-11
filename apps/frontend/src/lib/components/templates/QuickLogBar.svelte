<script lang="ts">
import { type TaskTemplate, getQuickLogTemplates } from '$lib/api';
import { createQuery } from '@tanstack/svelte-query';
import { ChevronRight, Plus, Zap } from 'lucide-svelte';
import { flip } from 'svelte/animate';
import { fade, fly } from 'svelte/transition';

interface Props {
	onUseTemplate?: (template: TaskTemplate) => void;
	onCreateTask?: () => void;
}

let { onUseTemplate, onCreateTask }: Props = $props();

const templatesQuery = createQuery({
	queryKey: ['templates', 'quick-log'],
	queryFn: getQuickLogTemplates,
});

const templates = $derived($templatesQuery.data || []);
const sortedTemplates = $derived(
	[...templates].sort(
		(a, b) => (a.quick_log_order || 0) - (b.quick_log_order || 0),
	),
);

let showAll = $state(false);
const displayedTemplates = $derived(
	showAll ? sortedTemplates : sortedTemplates.slice(0, 8),
);
</script>

<div class="space-y-2">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
            <div
                class="w-8 h-8 rounded-lg bg-gradient-to-br from-secondary to-secondary/70 flex items-center justify-center text-secondary-content shadow-sm"
            >
                <Zap class="w-4 h-4" />
            </div>
            <span class="font-semibold text-sm">Quick Log</span>
        </div>
        {#if sortedTemplates.length > 8}
            <button
                type="button"
                class="btn btn-ghost btn-xs gap-1"
                onclick={() => (showAll = !showAll)}
            >
                {showAll ? "Show less" : `+${sortedTemplates.length - 8} more`}
                <ChevronRight
                    class="w-3 h-3 {showAll
                        ? 'rotate-90'
                        : ''} transition-transform"
                />
            </button>
        {/if}
    </div>

    <!-- Templates Grid -->
    <div class="flex items-center gap-1.5 flex-wrap">
        {#if $templatesQuery.isLoading}
            {#each Array(4) as _, i}
                <div class="skeleton w-20 h-9 rounded-lg"></div>
            {/each}
        {:else if templates.length === 0}
            <p class="text-xs opacity-50">
                No quick log templates. Create one to get started!
            </p>
        {:else}
            {#each displayedTemplates as template (template.id)}
                <button
                    type="button"
                    class="btn btn-ghost btn-sm gap-1.5 px-2.5 hover:bg-base-200"
                    onclick={() => onUseTemplate?.(template)}
                    title={template.title}
                    in:fly={{ y: -5, duration: 200 }}
                    animate:flip={{ duration: 200 }}
                >
                    <span class="text-base">{template.icon || "⚡"}</span>
                    <span class="text-xs font-medium hidden sm:inline"
                        >{template.title}</span
                    >
                </button>
            {/each}
        {/if}

        <!-- Add new log button -->
        <button
            type="button"
            class="btn btn-ghost btn-sm btn-circle"
            onclick={onCreateTask}
            title="Create new task"
        >
            <Plus class="w-4 h-4" />
        </button>
    </div>
</div>
