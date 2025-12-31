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
        icon: Icon,
        showAddButton = false,
        addButtonLabel = "Add",
        onAdd,
        class: className = "",
    }: Props = $props();
</script>

<div
    class="flex flex-col gap-6 md:flex-row md:items-center md:justify-between {className}"
>
    <div class="flex items-start gap-4">
        {#if Icon}
            <div class="p-3 rounded-2xl bg-base-200/50 text-primary">
                <Icon class="w-8 h-8" />
            </div>
        {/if}
        <div class="flex flex-col gap-1">
            <h1 class="text-3xl font-bold tracking-tight text-base-content">
                {title}
            </h1>
            {#if subtitle}
                <p
                    class="text-base font-medium text-base-content/60 max-w-2xl leading-relaxed"
                >
                    {subtitle}
                </p>
            {/if}
        </div>
    </div>

    {#if showAddButton}
        <button
            class="group relative inline-flex items-center justify-center gap-2 px-6 py-2.5 font-medium text-primary-content bg-primary rounded-xl shadow-lg shadow-primary/20 hover:shadow-primary/30 hover:scale-[1.02] active:scale-[0.98] transition-all duration-200"
            onclick={onAdd}
        >
            <Plus
                class="w-5 h-5 transition-transform duration-200 group-hover:rotate-90"
            />
            <span>{addButtonLabel}</span>
        </button>
    {/if}
</div>
