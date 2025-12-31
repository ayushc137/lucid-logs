<script lang="ts">
    import { Check, ArrowRight } from "lucide-svelte";
    import { fly } from "svelte/transition";
    import { cubicOut } from "svelte/easing";
    import type { TimelineTask } from "./types";
    import { stripHtml } from "$lib/utils";

    interface Props {
        task: TimelineTask;
        status: "past" | "current" | "future";
        transitionDelay?: number;
        onTaskClick?: (taskId: string) => void;
    }

    let { task, status, transitionDelay = 0, onTaskClick }: Props = $props();

    const bg = $derived(task.categoryColor || "#6b7280");
    const isCompleted = $derived(task.completed);

    function formatTime(d: Date): string {
        if (!(d instanceof Date) || isNaN(d.getTime())) return "--:--";
        return d.toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
            hour12: true,
        });
    }

    function formatDuration(s: Date, e: Date): string {
        if (!(s instanceof Date) || !(e instanceof Date)) return "";
        const mins = Math.round((e.getTime() - s.getTime()) / 60000);
        if (mins < 1) return `<1m`;
        if (mins < 60) return `${mins}m`;
        const h = Math.floor(mins / 60),
            m = mins % 60;
        return m > 0 ? `${h}h ${m}m` : `${h}h`;
    }

    const descriptionPreview = $derived(
        task.description
            ? stripHtml(task.description, { newlineSeparator: " · " })
            : null,
    );
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_interactive_supports_focus -->
<div
    class="group relative flex items-stretch rounded-xl border-2 transition-all duration-200 overflow-hidden cursor-pointer shadow-sm
        {isCompleted
        ? 'bg-base-100/80 border-base-200/80 hover:border-base-300'
        : status === 'current'
          ? 'bg-base-100 border-primary/40 shadow-lg shadow-primary/10 ring-1 ring-primary/20'
          : 'bg-base-100 border-base-200/80 hover:border-base-300 hover:shadow-md'}"
    role="button"
    onclick={() => onTaskClick?.(task.id)}
    in:fly={{
        x: -8,
        duration: 250,
        delay: transitionDelay,
        easing: cubicOut,
    }}
>
    <!-- Left Color Bar -->
    <div
        class="w-1.5 shrink-0 transition-all duration-200"
        class:opacity-40={isCompleted}
        style="background-color: {bg};"
    ></div>

    <!-- Main Content -->
    <div class="flex-1 flex items-center gap-3 p-3 min-w-0">
        <!-- Time Badge -->
        <div
            class="flex flex-col items-center justify-center min-w-[72px] shrink-0 border-r border-base-200/60 pr-3"
        >
            <span
                class="text-sm font-bold font-mono {isCompleted
                    ? 'text-base-content/40'
                    : status === 'current'
                      ? 'text-primary'
                      : 'text-base-content/80'}"
            >
                {formatTime(task.startTime)}
            </span>
            <span class="text-[10px] font-medium text-base-content/40 mt-0.5">
                {formatDuration(task.startTime, task.endTime)}
            </span>
        </div>

        <!-- Task Details -->
        <div class="flex-1 min-w-0">
            <h4
                class="text-sm font-semibold truncate leading-snug transition-colors
                    {isCompleted
                    ? 'text-base-content/40 line-through decoration-base-content/20'
                    : 'text-base-content'}"
            >
                {task.title}
            </h4>

            <!-- Description Preview -->
            {#if descriptionPreview && !isCompleted}
                <p
                    class="text-[11px] text-base-content/50 truncate mt-0.5 leading-snug"
                >
                    {descriptionPreview}
                </p>
            {/if}

            {#if task.categoryName}
                <div class="flex items-center gap-1.5 mt-1">
                    <div
                        class="w-2 h-2 rounded-full shrink-0"
                        style="background-color: {bg};"
                    ></div>
                    <span
                        class="text-[11px] font-medium text-base-content/50 truncate"
                    >
                        {task.categoryName}
                    </span>
                </div>
            {/if}
        </div>

        <!-- Status Indicator -->
        <div class="flex items-center shrink-0 pl-2">
            {#if isCompleted}
                <div
                    class="flex items-center gap-1 text-success bg-success/10 px-2.5 py-1 rounded-lg"
                >
                    <Check class="w-3.5 h-3.5" strokeWidth={3} />
                    <span class="text-[11px] font-bold">Done</span>
                </div>
            {:else if status === "current"}
                <div
                    class="flex items-center gap-1.5 text-primary bg-primary/10 px-2.5 py-1 rounded-lg"
                >
                    <div class="relative">
                        <div class="w-1.5 h-1.5 rounded-full bg-primary"></div>
                        <div
                            class="absolute inset-0 w-1.5 h-1.5 rounded-full bg-primary animate-ping"
                        ></div>
                    </div>
                    <span class="text-[11px] font-bold">Now</span>
                </div>
            {:else}
                <div
                    class="opacity-0 group-hover:opacity-100 transition-all duration-200 text-base-content/30 group-hover:text-primary"
                >
                    <ArrowRight class="w-4 h-4" />
                </div>
            {/if}
        </div>
    </div>
</div>
