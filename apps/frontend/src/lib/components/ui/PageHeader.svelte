<script lang="ts">
import { Plus } from 'lucide-svelte';

interface Props {
	/** Page title */
	title: string;
	/** Subtitle/description */
	subtitle?: string;
	/** Icon component from lucide-svelte.
	 *  Loose structural type: lucide-svelte 0.561 ships Svelte-4 class component
	 *  types (SvelteComponentTyped) which don't satisfy Svelte 5's `Component`
	 *  interface, so we accept any component constructor/function instead. */
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	icon?: any;
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
	subtitle = '',
	icon: Icon,
	showAddButton = false,
	addButtonLabel = 'Add',
	onAdd,
	class: className = '',
}: Props = $props();
</script>

<div
    class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between {className}"
>
    <div class="flex items-center gap-3.5 min-w-0">
        {#if Icon}
            <div
                class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary"
            >
                <Icon class="w-5.5 h-5.5" />
            </div>
        {/if}
        <div class="flex flex-col gap-0.5 min-w-0">
            <h1 class="text-2xl sm:text-[1.7rem] font-bold tracking-tight text-base-content leading-tight">
                {title}
            </h1>
            {#if subtitle}
                <p class="text-sm text-base-content/55 max-w-2xl">
                    {subtitle}
                </p>
            {/if}
        </div>
    </div>

    {#if showAddButton}
        <button
            class="btn btn-primary gap-1.5 rounded-xl shadow-sm hover:shadow transition-all self-start sm:self-auto"
            onclick={onAdd}
        >
            <Plus class="w-4 h-4" strokeWidth={2.5} />
            <span>{addButtonLabel}</span>
        </button>
    {/if}
</div>
