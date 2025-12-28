<script lang="ts">
    import type { Component } from "svelte";
    import { Plus } from "lucide-svelte";

    interface Props {
        /** Page title */
        title: string;
        /** Subtitle/description */
        subtitle?: string;
        /** Icon component from lucide-svelte */
        icon?: Component<{ class?: string }>;
        /** Show add button */
        showAddButton?: boolean;
        /** Add button label */
        addButtonLabel?: string;
        /** Add button click handler */
        onAdd?: () => void;
        /** Custom class for container */
        class?: string;
    }

    let {
        title,
        subtitle = "",
        showAddButton = false,
        addButtonLabel = "Add",
        onAdd,
        class: className = "",
    }: Props = $props();
</script>

<div
    class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between {className}"
>
    <div>
        <h1 class="text-3xl font-bold">{title}</h1>
        {#if subtitle}
            <p class="text-sm opacity-60 mt-1">{subtitle}</p>
        {/if}
    </div>
    {#if showAddButton}
        <button
            class="btn btn-primary gap-2 shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-all"
            onclick={onAdd}
        >
            <Plus class="w-4 h-4" />
            {addButtonLabel}
        </button>
    {/if}
</div>
