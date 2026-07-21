<script lang="ts">
import { type DashboardResponse, getAnalyticsDashboard } from '$lib/api';
import ActivityHeatmap from '$lib/components/analytics/ActivityHeatmap.svelte';
import DonutChart from '$lib/components/analytics/DonutChart.svelte';
import FocusScore from '$lib/components/analytics/FocusScore.svelte';
import MoodTrend from '$lib/components/analytics/MoodTrend.svelte';
import { EmptyState, ErrorAlert, LoadingCard } from '$lib/components/ui';
import { createQuery } from '@tanstack/svelte-query';
import {
	Activity,
	BarChart3,
	Brain,
	CalendarDays,
	CheckCircle2,
	Clock,
	Flame,
	Heart,
	ListTodo,
	PieChart,
	Target,
	TrendingUp,
	Zap,
} from 'lucide-svelte';

let period = $state<'week' | 'month' | 'year'>('week');

const analyticsQuery = createQuery({
	get queryKey() {
		return ['analytics', period];
	},
	queryFn: () => getAnalyticsDashboard(period),
});

const data = $derived($analyticsQuery.data);

function pct(value: number): string {
	return `${Math.round(value * 100)}%`;
}

function hours(value: number): string {
	return `${value.toFixed(1)}h`;
}

const periodOptions = [
	{ value: 'week' as const, label: 'This Week' },
	{ value: 'month' as const, label: 'This Month' },
	{ value: 'year' as const, label: 'This Year' },
];
</script>

<svelte:head>
  <title>Analytics - Lucid Logs</title>
</svelte:head>

<div class="flex flex-col gap-6 px-4 py-4 sm:px-6 sm:py-6 max-w-5xl mx-auto w-full">
  <!-- Header -->
  <div class="flex items-center gap-3.5">
    <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
      <BarChart3 class="w-5.5 h-5.5" />
    </div>
    <div>
      <h1 class="text-2xl sm:text-[1.7rem] font-bold tracking-tight leading-tight">Analytics</h1>
      <p class="text-sm text-base-content/55">See how your days actually stack up</p>
    </div>
  </div>

  <!-- Period Selector -->
  <div class="flex gap-2">
    {#each periodOptions as opt}
      <button
        class="btn btn-sm rounded-full {period === opt.value ? 'btn-primary' : 'btn-ghost'}"
        onclick={() => (period = opt.value)}
      >
        {opt.label}
      </button>
    {/each}
  </div>

  {#if $analyticsQuery.isPending}
    <LoadingCard />
  {:else if $analyticsQuery.isError}
    <ErrorAlert message="Failed to load analytics. Try again in a moment." />
  {:else if !data || (data.tasks.total_tasks === 0 && data.goals.active_goals === 0)}
    <EmptyState
      title="Not enough data yet"
      description="Log a few tasks and goals first — analytics will appear here once you have something to measure."
      showButton={false}
    >
      {#snippet icon()}
        <BarChart3 class="w-10 h-10 text-primary" />
      {/snippet}
    </EmptyState>
  {:else}
    <!-- Top Stats Row -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <div class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 gap-1">
          <div class="flex items-center gap-2 text-info">
            <ListTodo class="w-4 h-4" />
            <span class="text-xs font-medium text-base-content/50">Tasks</span>
          </div>
          <p class="text-2xl font-bold stat-value">{data.tasks.completed_tasks}<span class="text-sm font-medium text-base-content/50">/{data.tasks.total_tasks}</span></p>
          <p class="text-xs text-base-content/50">completed</p>
        </div>
      </div>

      <div class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 gap-1">
          <div class="flex items-center gap-2 text-success">
            <CheckCircle2 class="w-4 h-4" />
            <span class="text-xs font-medium text-base-content/50">Completion</span>
          </div>
          <p class="text-2xl font-bold stat-value">{pct(data.tasks.completion_rate)}</p>
          <p class="text-xs text-base-content/50">of tasks done</p>
        </div>
      </div>

      <div class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 gap-1">
          <div class="flex items-center gap-2 text-warning">
            <Flame class="w-4 h-4" />
            <span class="text-xs font-medium text-base-content/50">Streak</span>
          </div>
          <p class="text-2xl font-bold stat-value">{data.goals.total_streak_days}<span class="text-sm font-medium text-base-content/50">d</span></p>
          <p class="text-xs text-base-content/50">total across goals</p>
        </div>
      </div>

      <div class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 gap-1">
          <div class="flex items-center gap-2 text-accent">
            <Clock class="w-4 h-4" />
            <span class="text-xs font-medium text-base-content/50">Time</span>
          </div>
          <p class="text-2xl font-bold stat-value">{hours(data.tasks.total_duration_hours)}</p>
          <p class="text-xs text-base-content/50">tracked</p>
        </div>
      </div>
    </div>

    <!-- Focus Score -->
    <section class="card bg-base-100 border border-base-200 shadow-sm">
      <div class="card-body p-4 sm:p-5">
        <h2 class="font-semibold text-[15px] flex items-center gap-2 mb-3">
          <Zap class="w-4 h-4 text-warning" />
          Focus Score
        </h2>
        <FocusScore score={data.tasks.focus_score} />
      </div>
    </section>

    <!-- Time per Category Donut -->
    {#if data.categories.distribution.length > 0}
      <section class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 sm:p-5 gap-3">
          <h2 class="font-semibold text-[15px] flex items-center gap-2">
            <PieChart class="w-4 h-4 text-secondary" />
            Where your time went
          </h2>
          <DonutChart distribution={data.categories.distribution} totalHours={data.categories.total_hours} />
        </div>
      </section>
    {/if}

    <!-- Activity Heatmap -->
    <section class="card bg-base-100 border border-base-200 shadow-sm">
      <div class="card-body p-4 sm:p-5 gap-3">
        <h2 class="font-semibold text-[15px] flex items-center gap-2">
          <Activity class="w-4 h-4 text-primary" />
          Activity Heatmap
        </h2>
        <ActivityHeatmap />
      </div>
    </section>

    <!-- Mood Trend Line -->
    {#if data.emotions.trend && data.emotions.trend.length > 1}
      <section class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 sm:p-5 gap-3">
          <h2 class="font-semibold text-[15px] flex items-center gap-2">
            <Heart class="w-4 h-4 text-error" />
            Mood Trend
          </h2>
          <MoodTrend trend={data.emotions.trend} />
        </div>
      </section>
    {/if}

    <!-- Goals Progress -->
    {#if data.goals.goal_progress.length > 0}
      <section class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 sm:p-5 gap-3">
          <h2 class="font-semibold text-[15px] flex items-center gap-2">
            <Target class="w-4 h-4 text-primary" />
            Goal Progress
          </h2>
          <div class="space-y-3">
            {#each data.goals.goal_progress as goal}
              <div class="space-y-1.5">
                <div class="flex items-center justify-between text-sm">
                  <span class="font-medium truncate">{goal.goal_title}</span>
                  <span class="text-base-content/50 shrink-0 ml-2">{Math.round(goal.progress)}%</span>
                </div>
                <progress
                  class="progress progress-primary w-full h-2"
                  value={goal.progress}
                  max="100"
                ></progress>
              </div>
            {/each}
          </div>
        </div>
      </section>
    {/if}

    <!-- Emotion Summary -->
    {#if data.emotions.top_emotions.length > 0}
      <section class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 sm:p-5 gap-3">
          <h2 class="font-semibold text-[15px] flex items-center gap-2">
            <Heart class="w-4 h-4 text-error" />
            Mood check
          </h2>
          <div class="flex flex-wrap gap-2">
            {#each data.emotions.top_emotions.slice(0, 5) as emotion}
              <span class="badge badge-lg gap-1.5 bg-base-200/60">
                <Brain class="w-3.5 h-3.5" />
                {emotion.emotion_name}
                <span class="text-xs opacity-60">×{emotion.count}</span>
              </span>
            {/each}
          </div>
        </div>
      </section>
    {/if}

    <!-- Streak Leaders -->
    {#if data.goals.streak_leaders.length > 0}
      <section class="card bg-base-100 border border-base-200 shadow-sm">
        <div class="card-body p-4 sm:p-5 gap-3">
          <h2 class="font-semibold text-[15px] flex items-center gap-2">
            <Zap class="w-4 h-4 text-warning" />
            Streak leaders
          </h2>
          <div class="space-y-2">
            {#each data.goals.streak_leaders as streak}
              <div class="flex items-center justify-between py-2 border-b border-base-200 last:border-0">
                <span class="font-medium text-sm truncate">{streak.goal_title}</span>
                <div class="flex items-center gap-3 text-sm shrink-0">
                  <span class="text-warning font-bold flex items-center gap-1">
                    <Flame class="w-3.5 h-3.5" />
                    {streak.current_streak}d
                  </span>
                  <span class="text-base-content/40 text-xs">best {streak.longest_streak}d</span>
                </div>
              </div>
            {/each}
          </div>
        </div>
      </section>
    {/if}
  {/if}
</div>
