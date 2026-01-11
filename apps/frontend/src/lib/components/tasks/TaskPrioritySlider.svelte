<script lang="ts">
    import { Card } from "$lib/components/ui";
    import { CircleAlert } from "lucide-svelte";
    import { cn } from "$lib/utils";

    interface Props {
        value: number;
        onChange?: (value: number) => void;
    }

    let { value = $bindable(3), onChange }: Props = $props();

    const priorityLabels = [
        "None",
        "Low",
        "Medium",
        "High",
        "Critical",
        "Urgent",
    ];
    const priorityColors = [
        "badge-ghost",
        "badge-info",
        "badge-success",
        "badge-warning",
        "badge-error",
        "badge-error",
    ];
</script>

<Card variant="bordered" class="transition-all duration-200 hover:shadow-md">
    <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-2">
            <CircleAlert class="w-4 h-4 opacity-50" />
            <span class="text-xs font-semibold uppercase opacity-50"
                >Priority</span
            >
        </div>
        <span class={cn("badge badge-sm", priorityColors[value])}
            >{priorityLabels[value]}</span
        >
    </div>
    <input
        type="range"
        min="0"
        max="5"
        bind:value
        oninput={(e) => onChange?.(parseInt(e.currentTarget.value))}
        class="range range-sm range-primary w-full transition-all duration-200"
        step="1"
    />
    <div class="relative h-5 text-[10px] mt-1.5 opacity-50 mx-0.5">
        {#each priorityLabels as label, i}
            <span
                class="absolute top-0 whitespace-nowrap"
                style="
                    left: {(i / (priorityLabels.length - 1)) * 100}%;
                    transform: translateX({i === 0
                    ? '0%'
                    : i === priorityLabels.length - 1
                      ? '-100%'
                      : '-50%'});
                "
            >
                {label}
            </span>
        {/each}
    </div>
</Card>
