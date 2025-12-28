<script lang="ts">
    import { cn } from "$lib/utils";
    import { COLOR_PRESETS, type ColorPreset } from "$lib/constants";

    type Size = "sm" | "md";

    interface Props {
        /** Currently selected color */
        value: string;
        /** Callback when color changes */
        onchange?: (color: string) => void;
        /** Show custom color picker toggle */
        allowCustom?: boolean;
        /** Is custom color mode enabled */
        customMode?: boolean;
        /** Toggle custom color mode */
        onCustomModeChange?: (enabled: boolean) => void;
        /** Grid columns */
        cols?: 8 | 4;
        /** Size of color buttons */
        size?: Size;
        /** Custom class */
        class?: string;
    }

    let {
        value = $bindable(),
        onchange,
        allowCustom = true,
        customMode = false,
        onCustomModeChange,
        cols = 8,
        size = "md",
        class: className = "",
    }: Props = $props();

    function selectColor(color: string) {
        value = color;
        onchange?.(color);
    }

    const sizeClasses = {
        sm: "w-6 h-6 rounded",
        md: "w-8 h-8 rounded-lg",
    };

    const inputSizeClasses = {
        sm: "w-10 h-8",
        md: "w-12 h-10",
    };
</script>

{#if customMode && allowCustom}
    <div class="flex items-center gap-2 {className}">
        <input
            type="color"
            class="{inputSizeClasses[
                size
            ]} rounded-lg cursor-pointer border-2 border-base-300"
            bind:value
            onchange={() => onchange?.(value)}
        />
        <input
            type="text"
            class="input input-bordered input-sm flex-1 font-mono"
            bind:value
            placeholder="#000000"
            onchange={() => onchange?.(value)}
        />
    </div>
{:else}
    <div
        class="grid gap-2 {cols === 8
            ? 'grid-cols-8'
            : 'grid-cols-4'} {className}"
    >
        {#each COLOR_PRESETS as color}
            <button
                type="button"
                class={cn(
                    "transition-all border-2",
                    sizeClasses[size],
                    value === color
                        ? "ring-2 ring-primary ring-offset-2 ring-offset-base-100 border-white scale-110"
                        : "border-transparent hover:scale-105",
                )}
                style="background-color: {color};"
                onclick={() => selectColor(color)}
                aria-label="Select color {color}"
            ></button>
        {/each}
    </div>
{/if}
