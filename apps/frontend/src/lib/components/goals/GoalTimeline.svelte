<script lang="ts">
    import {
        Sparkles,
        RotateCcw,
        Trophy,
        Archive,
        Flame,
        AlertCircle,
        Target,
        CalendarDays,
        Link2,
        Unlink,
        ArrowDown,
        ArrowUp,
        Clock,
        ChevronDown,
        ChevronUp,
    } from "lucide-svelte";
    import type { GoalLog, GoalLogEvent } from "$lib/api/goals";
    import { cn } from "$lib/utils";

    interface Props {
        logs: GoalLog[];
        isLoading?: boolean;
        filter?: "all" | "with-tasks" | "goal-only";
        onFilterChange?: (filter: "all" | "with-tasks" | "goal-only") => void;
    }

    let {
        logs = [],
        isLoading = false,
        filter = "all",
        onFilterChange,
    }: Props = $props();

    let showEventDetails = $state<string | null>(null);

    // Event display config
    const eventConfig: Record<
        GoalLogEvent,
        { icon: typeof Sparkles; color: string; label: string }
    > = {
        created: {
            icon: Sparkles,
            color: "text-success",
            label: "Goal created",
        },
        updated: { icon: RotateCcw, color: "text-info", label: "Goal updated" },
        completed: {
            icon: Trophy,
            color: "text-success",
            label: "Goal completed",
        },
        archived: {
            icon: Archive,
            color: "text-warning",
            label: "Goal archived",
        },
        reactivated: {
            icon: Sparkles,
            color: "text-success",
            label: "Goal reactivated",
        },
        deleted: {
            icon: AlertCircle,
            color: "text-error",
            label: "Goal deleted",
        },
        streak_updated: {
            icon: Flame,
            color: "text-warning",
            label: "Streak updated",
        },
        streak_broken: {
            icon: AlertCircle,
            color: "text-error",
            label: "Streak broken",
        },
        target_met: {
            icon: Target,
            color: "text-success",
            label: "Target reached!",
        },
        target_exceeded: {
            icon: AlertCircle,
            color: "text-error",
            label: "Target exceeded",
        },
        period_end: {
            icon: CalendarDays,
            color: "text-info",
            label: "Period ended",
        },
        task_linked: {
            icon: Link2,
            color: "text-primary",
            label: "Task linked",
        },
        task_unlinked: {
            icon: Unlink,
            color: "text-warning",
            label: "Task unlinked",
        },
        child_added: {
            icon: ArrowDown,
            color: "text-success",
            label: "Child added",
        },
        child_removed: {
            icon: ArrowUp,
            color: "text-error",
            label: "Child removed",
        },
    };

    // Filter logs based on filter prop
    const filteredLogs = $derived.by(() => {
        if (filter === "all") return logs;
        if (filter === "with-tasks") {
            return logs.filter(
                (l) =>
                    l.event === "task_linked" ||
                    l.event === "task_unlinked" ||
                    l.triggered_by_task_id,
            );
        }
        return logs.filter(
            (l) =>
                l.event !== "task_linked" &&
                l.event !== "task_unlinked" &&
                !l.triggered_by_task_id,
        );
    });

    function formatEventDate(dateStr: string): string {
        const date = new Date(dateStr);
        const now = new Date();
        const diff = now.getTime() - date.getTime();
        const days = Math.floor(diff / (1000 * 60 * 60 * 24));

        if (days === 0) {
            return date.toLocaleTimeString([], {
                hour: "2-digit",
                minute: "2-digit",
            });
        } else if (days === 1) {
            return "Yesterday";
        } else if (days < 7) {
            return `${days} days ago`;
        } else {
            return date.toLocaleDateString([], {
                month: "short",
                day: "numeric",
            });
        }
    }

    function formatChanges(changes: Record<string, unknown>): string {
        const entries = Object.entries(changes);
        if (entries.length === 0) return "";
        return entries
            .map(([key, value]) => {
                const formattedKey = key.replace(/_/g, " ");
                if (typeof value === "object") {
                    return `${formattedKey}: ${JSON.stringify(value)}`;
                }
                return `${formattedKey}: ${value}`;
            })
            .join(", ");
    }

    function toggleEventDetails(logId: string) {
        showEventDetails = showEventDetails === logId ? null : logId;
    }
</script>

<div class="space-y-4">
    <!-- Filter Buttons -->
    <div class="flex items-center gap-2 justify-center">
        <button
            class={cn(
                "btn btn-xs",
                filter === "all" ? "btn-primary" : "btn-ghost",
            )}
            onclick={() => onFilterChange?.("all")}
        >
            All
        </button>
        <button
            class={cn(
                "btn btn-xs gap-1",
                filter === "with-tasks" ? "btn-primary" : "btn-ghost",
            )}
            onclick={() => onFilterChange?.("with-tasks")}
        >
            <Link2 class="w-3 h-3" />
            With Tasks
        </button>
        <button
            class={cn(
                "btn btn-xs gap-1",
                filter === "goal-only" ? "btn-primary" : "btn-ghost",
            )}
            onclick={() => onFilterChange?.("goal-only")}
        >
            <Target class="w-3 h-3" />
            Goal Only
        </button>
    </div>

    <!-- Timeline -->
    {#if isLoading}
        <div class="flex items-center justify-center py-8">
            <span class="loading loading-spinner loading-md"></span>
        </div>
    {:else if filteredLogs.length === 0}
        <div class="text-center py-8 text-base-content/40">
            <Clock class="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p class="text-sm">No activity yet</p>
        </div>
    {:else}
        <ul class="timeline timeline-vertical timeline-compact">
            {#each filteredLogs as log, i (log.id)}
                {@const config =
                    eventConfig[log.event as GoalLogEvent] ||
                    eventConfig.updated}
                {@const IconComponent = config.icon}
                {@const hasChanges =
                    log.changes && Object.keys(log.changes).length > 0}

                <li>
                    {#if i > 0}
                        <hr class="bg-base-300" />
                    {/if}

                    <div
                        class="timeline-start text-xs text-base-content/50 pr-2 whitespace-nowrap"
                    >
                        {formatEventDate(log.created_at)}
                    </div>

                    <div class="timeline-middle">
                        <div
                            class={cn(
                                "w-7 h-7 rounded-full flex items-center justify-center",
                                config.color,
                                "bg-base-200",
                            )}
                        >
                            <IconComponent class="w-3.5 h-3.5" />
                        </div>
                    </div>

                    <div
                        class="timeline-end timeline-box py-2 px-3 ml-2 bg-base-100 border-base-300"
                    >
                        <div class="flex items-center justify-between gap-2">
                            <div class="flex items-center gap-2">
                                <span class="text-sm font-medium"
                                    >{config.label}</span
                                >
                                {#if log.triggered_by_task_id}
                                    <span class="badge badge-xs badge-ghost"
                                        >via task</span
                                    >
                                {/if}
                            </div>
                            {#if hasChanges}
                                <button
                                    class="btn btn-ghost btn-xs p-0.5"
                                    onclick={() => toggleEventDetails(log.id)}
                                >
                                    {#if showEventDetails === log.id}
                                        <ChevronUp class="w-3.5 h-3.5" />
                                    {:else}
                                        <ChevronDown class="w-3.5 h-3.5" />
                                    {/if}
                                </button>
                            {/if}
                        </div>

                        {#if showEventDetails === log.id && hasChanges}
                            <div class="mt-2 pt-2 border-t border-base-300">
                                <pre
                                    class="text-[10px] text-base-content/60 whitespace-pre-wrap break-all">{formatChanges(
                                        log.changes!,
                                    )}</pre>
                            </div>
                        {/if}
                    </div>

                    {#if i < filteredLogs.length - 1}
                        <hr class="bg-base-300" />
                    {/if}
                </li>
            {/each}
        </ul>
    {/if}
</div>
