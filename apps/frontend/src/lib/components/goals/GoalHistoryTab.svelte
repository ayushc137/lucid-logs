<script lang="ts">
    import type { Goal, GoalLog } from "$lib/api";
    import { GoalTimeline } from "$lib/components/goals";
    import {
        Award,
        BarChart3,
        Flame,
        History,
        TrendingUp,
    } from "lucide-svelte";

    interface Props {
        goal: Goal | null;
        logs: GoalLog[];
        isLoading: boolean;
        currentStreak: number;
        longestStreak: number;
        progressPercent: number;
        currentValue: number;
    }

    let {
        goal = null,
        logs = [],
        isLoading = false,
        currentStreak = 0,
        longestStreak = 0,
        progressPercent = 0,
        currentValue = 0,
    }: Props = $props();
</script>

<div class="p-6">
    {#if goal}
        <!-- Stats Summary -->
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
            <div
                class="p-4 rounded-2xl bg-gradient-to-br from-orange-500/10 to-yellow-500/10 border border-orange-500/20"
            >
                <div class="flex items-center gap-2 mb-2">
                    <Flame class="w-5 h-5 text-orange-500" />
                </div>
                <p class="text-2xl font-bold text-orange-500">
                    {currentStreak}
                </p>
                <p class="text-xs text-base-content/60">Current Streak</p>
            </div>
            <div
                class="p-4 rounded-2xl bg-gradient-to-br from-purple-500/10 to-pink-500/10 border border-purple-500/20"
            >
                <div class="flex items-center gap-2 mb-2">
                    <Award class="w-5 h-5 text-purple-500" />
                </div>
                <p class="text-2xl font-bold text-purple-500">
                    {longestStreak}
                </p>
                <p class="text-xs text-base-content/60">Best Streak</p>
            </div>
            <div
                class="p-4 rounded-2xl bg-gradient-to-br from-green-500/10 to-emerald-500/10 border border-green-500/20"
            >
                <div class="flex items-center gap-2 mb-2">
                    <TrendingUp class="w-5 h-5 text-green-500" />
                </div>
                <p class="text-2xl font-bold text-green-500">
                    {progressPercent}%
                </p>
                <p class="text-xs text-base-content/60">Progress</p>
            </div>
            <div
                class="p-4 rounded-2xl bg-gradient-to-br from-blue-500/10 to-cyan-500/10 border border-blue-500/20"
            >
                <div class="flex items-center gap-2 mb-2">
                    <BarChart3 class="w-5 h-5 text-blue-500" />
                </div>
                <p class="text-2xl font-bold text-blue-500">{currentValue}</p>
                <p class="text-xs text-base-content/60">Current Value</p>
            </div>
        </div>

        <!-- Activity Timeline -->
        <div
            class="border border-base-300 rounded-2xl p-4 bg-base-200/30 max-h-[350px] overflow-y-auto"
        >
            <h4 class="font-semibold mb-4 text-center">Activity Timeline</h4>
            <GoalTimeline {logs} {isLoading} />
        </div>
    {:else}
        <div class="text-center py-16 text-base-content/50">
            <div
                class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-base-200 flex items-center justify-center"
            >
                <History class="w-8 h-8 opacity-30" />
            </div>
            <p class="font-medium text-lg">Save your goal first</p>
            <p class="text-sm mt-1">
                History will appear after you create the goal
            </p>
        </div>
    {/if}
</div>
