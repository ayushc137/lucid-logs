<script lang="ts">
    import { onMount } from "svelte";
    import type { Goal, StreakEvent } from "$lib/api";
    import { getGoalStreakHistory } from "$lib/api/goals";
    import {
        Flame,
        TrendingUp,
        TrendingDown,
        Minus,
        Loader2,
        Activity,
        Calendar,
    } from "lucide-svelte";

    interface Props {
        goal: Goal;
        limit?: number;
    }

    let { goal, limit = 20 }: Props = $props();

    let isLoading = $state(true);
    let error = $state<string | null>(null);
    let events = $state<StreakEvent[]>([]);

    onMount(async () => {
        try {
            events = await getGoalStreakHistory(goal.id, limit);
        } catch (err) {
            error =
                err instanceof Error
                    ? err.message
                    : "Failed to load streak history";
        } finally {
            isLoading = false;
        }
    });

    // Get icon and color for event type
    function getEventStyle(event: StreakEvent) {
        switch (event.event) {
            case "increment":
                return {
                    icon: TrendingUp,
                    color: "text-success",
                    bgColor: "bg-success/10",
                    borderColor: "border-success/20",
                    label: "Streak +1",
                };
            case "broken":
                return {
                    icon: TrendingDown,
                    color: "text-error",
                    bgColor: "bg-error/10",
                    borderColor: "border-error/20",
                    label: "Streak Broken",
                };
            case "reset":
                return {
                    icon: Activity,
                    color: "text-warning",
                    bgColor: "bg-warning/10",
                    borderColor: "border-warning/20",
                    label: "Streak Reset",
                };
            default:
                return {
                    icon: Minus,
                    color: "text-base-content/50",
                    bgColor: "bg-base-200",
                    borderColor: "border-base-300",
                    label: "Maintained",
                };
        }
    }

    // Format date for display
    function formatDate(dateStr: string): string {
        const date = new Date(dateStr);
        return date.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
            year:
                date.getFullYear() !== new Date().getFullYear()
                    ? "numeric"
                    : undefined,
        });
    }

    function formatTime(dateStr: string): string {
        const date = new Date(dateStr);
        return date.toLocaleTimeString("en-US", {
            hour: "numeric",
            minute: "2-digit",
        });
    }
</script>

<div class="space-y-4">
    {#if isLoading}
        <div class="flex items-center justify-center py-8">
            <Loader2 class="w-6 h-6 animate-spin text-primary" />
        </div>
    {:else if error}
        <div class="text-center py-8 text-error">
            <p>{error}</p>
        </div>
    {:else if events.length === 0}
        <div class="text-center py-8 text-base-content/50">
            <Calendar class="w-10 h-10 mx-auto mb-2 opacity-30" />
            <p class="font-medium">No streak history yet</p>
            <p class="text-sm mt-1">Complete tasks to build your streak</p>
        </div>
    {:else}
        <!-- Timeline -->
        <div class="relative">
            <!-- Vertical line -->
            <div
                class="absolute left-[19px] top-0 bottom-0 w-0.5 bg-base-300"
            ></div>

            <!-- Events -->
            <div class="space-y-4">
                {#each events as event}
                    {@const style = getEventStyle(event)}
                    {@const IconComponent = style.icon}
                    <div class="flex gap-4 relative">
                        <!-- Icon bubble -->
                        <div
                            class="{style.bgColor} {style.borderColor} border rounded-full w-10 h-10 flex items-center justify-center flex-shrink-0 z-10"
                        >
                            <IconComponent class="w-5 h-5 {style.color}" />
                        </div>

                        <!-- Content -->
                        <div class="flex-1 pt-1">
                            <div class="flex items-center gap-2 flex-wrap">
                                <span class="font-medium {style.color}"
                                    >{style.label}</span
                                >
                                <span class="text-base-content/40 text-xs">
                                    {formatDate(event.date)} at {formatTime(
                                        event.date,
                                    )}
                                </span>
                            </div>

                            <div class="flex items-center gap-2 mt-1">
                                <div
                                    class="flex items-center gap-1 text-sm text-base-content/60"
                                >
                                    <Flame class="w-4 h-4 text-orange-500" />
                                    <span>{event.streak_before}</span>
                                    <span class="mx-1">→</span>
                                    <span
                                        class="font-semibold {event.streak_after >
                                        event.streak_before
                                            ? 'text-success'
                                            : event.streak_after <
                                                event.streak_before
                                              ? 'text-error'
                                              : ''}"
                                    >
                                        {event.streak_after}
                                    </span>
                                </div>
                            </div>

                            {#if event.notes}
                                <p class="text-sm text-base-content/50 mt-1">
                                    {event.notes}
                                </p>
                            {/if}
                        </div>
                    </div>
                {/each}
            </div>
        </div>
    {/if}
</div>
