<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /** Table variant */
        variant?: "default" | "lg" | "compact";
        /** Zebra striping */
        zebra?: boolean;
        /** Hover effect on rows */
        hover?: boolean;
        /** Pinned rows (sticky header) */
        pinned?: boolean;
        /** Custom class */
        class?: string;
        /** Table content (thead, tbody) */
        children: Snippet;
        /** Footer content */
        footer?: Snippet;
    }

    let {
        variant = "default",
        zebra = false,
        hover = true,
        pinned = false,
        class: className = "",
        children,
        footer,
    }: Props = $props();

    const variantClasses: Record<string, string> = {
        default: "table",
        lg: "table table-lg",
        compact: "table table-sm",
    };
</script>

<div
    class="card bg-base-100 shadow-lg border border-base-200 overflow-hidden {className}"
>
    <div class="overflow-x-auto">
        <table
            class={variantClasses[variant]}
            class:table-zebra={zebra}
            class:table-pin-rows={pinned}
        >
            {@render children()}
        </table>
    </div>
    {#if footer}
        <div
            class="px-4 py-3 bg-base-200/50 border-t border-base-200 text-sm opacity-60 flex items-center justify-between"
        >
            {@render footer()}
        </div>
    {/if}
</div>
