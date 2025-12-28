<script lang="ts">
    import type { Snippet } from "svelte";
    import IconBox from "./IconBox.svelte";

    interface Props {
        /** Section title */
        title: string;
        /** Optional subtitle */
        subtitle?: string;
        /** Semantic color for icon box */
        color?:
            | "primary"
            | "secondary"
            | "accent"
            | "info"
            | "success"
            | "warning"
            | "error";
        /** Icon slot */
        icon?: Snippet;
        /** Actions slot (right side) */
        actions?: Snippet;
        /** Custom class */
        class?: string;
    }

    let {
        title,
        subtitle = "",
        color = "primary",
        icon,
        actions,
        class: className = "",
    }: Props = $props();
</script>

<div class="flex items-center justify-between {className}">
    <div class="flex items-center gap-3">
        {#if icon}
            <IconBox size="md" variant="subtle" {color}>
                {@render icon()}
            </IconBox>
        {/if}
        <div>
            <h3 class="font-semibold text-sm">{title}</h3>
            {#if subtitle}
                <p class="text-xs opacity-50">{subtitle}</p>
            {/if}
        </div>
    </div>
    {#if actions}
        <div class="flex items-center gap-2">
            {@render actions()}
        </div>
    {/if}
</div>
