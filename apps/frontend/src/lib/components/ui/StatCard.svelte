<script lang="ts">
    import type { Snippet } from "svelte";
    import IconBox from "./IconBox.svelte";

    interface Props {
        /** Main value to display */
        value: string | number;
        /** Label text */
        label: string;
        /** Optional unit/suffix */
        unit?: string;
        /** Semantic color */
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
        /** Custom class */
        class?: string;
    }

    let {
        value,
        label,
        unit = "",
        color = "primary",
        icon,
        class: className = "",
    }: Props = $props();
</script>

<div class="flex items-center gap-3 {className}">
    {#if icon}
        <IconBox size="md" variant="solid" {color}>
            {@render icon()}
        </IconBox>
    {/if}
    <div>
        <p class="text-xs font-medium uppercase opacity-50">{label}</p>
        <p class="text-xl font-bold text-{color}">
            {value}{#if unit}<span class="text-sm font-medium opacity-60 ml-0.5"
                    >{unit}</span
                >{/if}
        </p>
    </div>
</div>
