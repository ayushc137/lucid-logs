<script lang="ts">
    import { CircleAlert } from "lucide-svelte";

    interface Props {
        /** Error message to display */
        message?: string;
        /** Show retry button */
        showRetry?: boolean;
        /** Retry button click handler */
        onRetry?: () => void;
        /** Custom class for container */
        class?: string;
    }

    let {
        message = "Something went wrong. Please try again.",
        showRetry = true,
        onRetry,
        class: className = "",
    }: Props = $props();
</script>

<div class="alert alert-error shadow-lg {className}">
    <CircleAlert class="w-5 h-5" />
    <span>{message}</span>
    {#if showRetry && onRetry}
        <button class="btn btn-sm btn-ghost" onclick={onRetry}> Retry </button>
    {/if}
</div>
