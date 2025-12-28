<script lang="ts">
    import { Plus } from "lucide-svelte";
    import type { Snippet } from "svelte";

    interface Props {
        /** Main title */
        title: string;
        /** Description text */
        description: string;
        /** Show action button */
        showButton?: boolean;
        /** Button label */
        buttonLabel?: string;
        /** Button click handler */
        onButtonClick?: () => void;
        /** Custom class for container */
        class?: string;
        /** Icon slot */
        icon?: Snippet;
    }

    let {
        title,
        description,
        showButton = true,
        buttonLabel = "Get Started",
        onButtonClick,
        class: className = "",
        icon,
    }: Props = $props();
</script>

<div class="card bg-base-100 shadow-lg border border-base-200 {className}">
    <div class="card-body py-12 text-center">
        {#if icon}
            <div
                class="w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4"
            >
                {@render icon()}
            </div>
        {/if}
        <h2 class="text-2xl font-bold">{title}</h2>
        <p class="opacity-60 mt-2 max-w-md mx-auto">
            {description}
        </p>
        {#if showButton}
            <button
                class="btn btn-primary mt-6 gap-2 shadow-lg shadow-primary/20"
                onclick={onButtonClick}
            >
                <Plus class="w-4 h-4" />
                {buttonLabel}
            </button>
        {/if}
    </div>
</div>
