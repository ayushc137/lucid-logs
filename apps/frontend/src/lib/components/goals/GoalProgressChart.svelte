<script lang="ts">
    import { onMount } from "svelte";
    import type { Goal, DailyStats, DailyProgressResponse } from "$lib/api";
    import { getGoalDailyProgress } from "$lib/api/goals";
    import {
        BarChart3,
        TrendingUp,
        Calendar,
        Loader2,
        CheckCircle2,
        XCircle,
        MinusCircle,
    } from "lucide-svelte";

    interface Props {
        goal: Goal;
        days?: number; // Default to 30 days
    }

    let { goal, days = 30 }: Props = $props();

    let isLoading = $state(true);
    let error = $state<string | null>(null);
    let progressData = $state<DailyStats[]>([]);
    let targetValue = $state(0);

    // Calculate date range
    function getDateRange() {
        const end = new Date();
        const start = new Date();
        start.setDate(start.getDate() - days);
        return {
            start_date: start.toISOString(),
            end_date: end.toISOString(),
        };
    }

    // Fetch data on mount
    onMount(async () => {
        try {
            const { start_date, end_date } = getDateRange();
            const response = await getGoalDailyProgress(goal.id, {
                start_date,
                end_date,
            });
            progressData = response.data || [];
            targetValue = response.target_per_period || goal.target?.value || 0;
        } catch (err) {
            error =
                err instanceof Error
                    ? err.message
                    : "Failed to load progress data";
        } finally {
            isLoading = false;
        }
    });

    // Calculate stats from data
    const metCount = $derived(
        progressData.filter((d) => d.status === "met").length,
    );
    const missedCount = $derived(
        progressData.filter((d) => d.status === "missed").length,
    );
    const pendingCount = $derived(
        progressData.filter((d) => d.status === "pending").length,
    );
    const avgValue = $derived(
        progressData.length > 0
            ? progressData.reduce((sum, d) => sum + d.daily_value, 0) /
                  progressData.length
            : 0,
    );
    const maxValue = $derived(
        Math.max(...progressData.map((d) => d.daily_value), targetValue, 1),
    );

    // Get bar height percentage
    function getBarHeight(value: number): number {
        return Math.min((value / maxValue) * 100, 100);
    }

    // Get bar color based on status
    function getBarColor(stat: DailyStats): string {
        switch (stat.status) {
            case "met":
                return "bg-success";
            case "exceeded":
                return "bg-success";
            case "missed":
                return "bg-error/60";
            default:
                return "bg-base-content/30";
        }
    }

    // Format date for display
    function formatDate(dateStr: string): string {
        const date = new Date(dateStr);
        return date.toLocaleDateString("en-US", {
            weekday: "short",
            month: "short",
            day: "numeric",
        });
    }
</script>

<div class="space-y-6">
    {#if isLoading}
        <div class="flex items-center justify-center py-12">
            <Loader2 class="w-8 h-8 animate-spin text-primary" />
        </div>
    {:else if error}
        <div class="alert alert-error">
            <XCircle class="w-5 h-5" />
            <span>{error}</span>
        </div>
    {:else if progressData.length === 0}
        <div class="text-center py-12 text-base-content/50">
            <Calendar class="w-12 h-12 mx-auto mb-3 opacity-30" />
            <p class="font-medium">No progress data yet</p>
            <p class="text-sm mt-1">Data will appear as you log tasks</p>
        </div>
    {:else}
        <!-- Summary Stats -->
        <div class="grid grid-cols-3 gap-3">
            <div
                class="p-4 rounded-xl bg-success/10 border border-success/20 text-center"
            >
                <div class="flex items-center justify-center gap-2 mb-1">
                    <CheckCircle2 class="w-4 h-4 text-success" />
                </div>
                <p class="text-2xl font-bold text-success">{metCount}</p>
                <p class="text-xs text-base-content/60">Days Met</p>
            </div>
            <div
                class="p-4 rounded-xl bg-error/10 border border-error/20 text-center"
            >
                <div class="flex items-center justify-center gap-2 mb-1">
                    <XCircle class="w-4 h-4 text-error" />
                </div>
                <p class="text-2xl font-bold text-error">{missedCount}</p>
                <p class="text-xs text-base-content/60">Days Missed</p>
            </div>
            <div
                class="p-4 rounded-xl bg-primary/10 border border-primary/20 text-center"
            >
                <div class="flex items-center justify-center gap-2 mb-1">
                    <TrendingUp class="w-4 h-4 text-primary" />
                </div>
                <p class="text-2xl font-bold text-primary">
                    {avgValue.toFixed(1)}
                </p>
                <p class="text-xs text-base-content/60">Avg/Day</p>
            </div>
        </div>

        <!-- Progress Chart -->
        <div class="p-4 rounded-xl bg-base-200/50 border border-base-300">
            <div class="flex items-center justify-between mb-4">
                <h4 class="font-semibold flex items-center gap-2">
                    <BarChart3 class="w-4 h-4" />
                    Daily Progress
                </h4>
                <span class="text-xs text-base-content/50"
                    >Last {days} days</span
                >
            </div>

            <!-- Bar Chart -->
            <div class="relative h-32">
                <!-- Target line -->
                {#if targetValue > 0}
                    <div
                        class="absolute w-full border-t-2 border-dashed border-warning/50 z-10"
                        style="bottom: {getBarHeight(targetValue)}%"
                    >
                        <span
                            class="absolute right-0 -top-4 text-xs text-warning font-medium"
                        >
                            Target: {targetValue}
                        </span>
                    </div>
                {/if}

                <!-- Bars -->
                <div class="flex items-end justify-between h-full gap-px">
                    {#each progressData.slice(-days) as stat}
                        <div
                            class="flex-1 flex flex-col items-center justify-end h-full group relative"
                        >
                            <!-- Bar -->
                            <div
                                class="{getBarColor(
                                    stat,
                                )} w-full rounded-t transition-all duration-300 hover:opacity-80 cursor-pointer min-h-[2px]"
                                style="height: {Math.max(
                                    getBarHeight(stat.daily_value),
                                    2,
                                )}%"
                            ></div>

                            <!-- Tooltip on hover -->
                            <div
                                class="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-20"
                            >
                                <div
                                    class="bg-base-300 rounded-lg shadow-lg p-2 text-xs whitespace-nowrap border border-base-content/10"
                                >
                                    <p class="font-medium">
                                        {formatDate(stat.date)}
                                    </p>
                                    <p class="text-base-content/70">
                                        Value: {stat.daily_value}
                                    </p>
                                    <p
                                        class={stat.status === "met"
                                            ? "text-success"
                                            : stat.status === "missed"
                                              ? "text-error"
                                              : "text-base-content/50"}
                                    >
                                        {stat.status.charAt(0).toUpperCase() +
                                            stat.status.slice(1)}
                                    </p>
                                </div>
                            </div>
                        </div>
                    {/each}
                </div>
            </div>

            <!-- X-axis labels (show only first, middle, last) -->
            <div
                class="flex justify-between mt-2 text-[10px] text-base-content/40"
            >
                {#if progressData.length > 0}
                    <span>{formatDate(progressData[0].date)}</span>
                    {#if progressData.length > 2}
                        <span
                            >{formatDate(
                                progressData[
                                    Math.floor(progressData.length / 2)
                                ].date,
                            )}</span
                        >
                    {/if}
                    <span
                        >{formatDate(
                            progressData[progressData.length - 1].date,
                        )}</span
                    >
                {/if}
            </div>
        </div>
    {/if}
</div>
