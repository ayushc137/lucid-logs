<script lang="ts">
import {
	getAnalyticsDashboard,
	getAnalyticsStreaks,
	getActivityHeatmap,
	type DashboardResponse,
	type StreaksResponse,
} from '$lib/api';
import ActivityHeatmap from '$lib/components/analytics/ActivityHeatmap.svelte';
import DonutChart from '$lib/components/analytics/DonutChart.svelte';
import {
	Card,
	EmptyState,
	ErrorAlert,
	LoadingCard,
	SectionHeader,
	StatCard,
} from '$lib/components/ui';
import { CHART_COLORS, QUADRANT_COLORS, quadrantColor } from '$lib/utils/chart-colors';
import { createQuery } from '@tanstack/svelte-query';
import {
	Activity,
	BarChart3,
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
import { cn } from '$lib/utils';

type PeriodOption = 'week' | 'month' | 'quarter';

let period = $state<PeriodOption>('week');

const periodOptions: { value: PeriodOption; label: string; days: number }[] = [
	{ value: 'week', label: '7d', days: 7 },
	{ value: 'month', label: '30d', days: 30 },
	{ value: 'quarter', label: '90d', days: 90 },
];

const dashboardQuery = createQuery({
	get queryKey() {
		return ['analytics', 'dashboard', period];
	},
	queryFn: () => getAnalyticsDashboard(period),
});

const streaksQuery = createQuery({
	queryKey: ['analytics', 'streaks'],
	queryFn: () => getAnalyticsStreaks(),
});

const heatmapQuery = createQuery({
	get queryKey() {
		return ['analytics', 'heatmap', period];
	},
	queryFn: () => getActivityHeatmap(),
});

const data = $derived($dashboardQuery.data);
const streaks = $derived($streaksQuery.data);
const isLoading = $derived($dashboardQuery.isPending);

function pct(value: number): string {
	return `${Math.round(value)}%`;
}

function hours(value: number): string {
	return `${value.toFixed(1)}h`;
}

// Quadrant distribution segments for mood
const quadrantSegments = $derived.by(() => {
	const qd = data?.emotions?.quadrant_distribution;
	if (!qd) return [];
	const total = qd.yellow + qd.green + qd.red + qd.blue;
	if (total === 0) return [];
	return [
		{ key: 'yellow', value: qd.yellow, color: QUADRANT_COLORS.yellow, pct: (qd.yellow / total) * 100 },
		{ key: 'green', value: qd.green, color: QUADRANT_COLORS.green, pct: (qd.green / total) * 100 },
		{ key: 'red', value: qd.red, color: QUADRANT_COLORS.red, pct: (qd.red / total) * 100 },
		{ key: 'blue', value: qd.blue, color: QUADRANT_COLORS.blue, pct: (qd.blue / total) * 100 },
	].filter((s) => s.value > 0);
});

const hasData = $derived(!!data && (data.tasks.total_tasks > 0 || data.goals.active_goals > 0));
</script>

<svelte:head>
	<title>Analytics - Lucid Logs</title>
</svelte:head>

<div class="flex flex-col gap-6 px-4 py-4 sm:px-6 sm:py-6 max-w-5xl mx-auto w-full">
	<!-- Sticky Top Bar -->
	<div class="sticky top-0 z-10 -mx-4 sm:-mx-6 px-4 sm:px-6 py-3 bg-base-200/95 backdrop-blur-sm border-b border-base-content/5">
		<div class="flex items-center justify-between gap-3">
			<div class="flex items-center gap-3 min-w-0">
				<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
					<BarChart3 class="w-5 h-5" />
				</div>
				<h1 class="text-xl font-bold tracking-tight">Analytics</h1>
			</div>
			<!-- Period Selector (segmented control) -->
			<div class="flex items-center gap-1 bg-base-100 rounded-full border border-base-content/5 p-0.5">
				{#each periodOptions as opt}
					<button
						class={cn(
							'px-3 py-1.5 text-xs font-medium rounded-full transition-all',
							period === opt.value
								? 'bg-primary text-primary-content'
								: 'text-base-content/60 hover:text-base-content',
						)}
						onclick={() => (period = opt.value)}
					>
						{opt.label}
					</button>
				{/each}
			</div>
		</div>
	</div>

	{#if isLoading}
		<LoadingCard />
	{:else if $dashboardQuery.isError}
		<ErrorAlert message="Failed to load analytics. Try again in a moment." />
	{:else if !data || !hasData}
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
		<!-- Overview Stat Cards -->
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
			<Card>
				<StatCard value={pct(data.tasks.completion_rate)} label="Completion Rate" color="success">
					{#snippet icon()}<CheckCircle2 class="w-4 h-4" />{/snippet}
				</StatCard>
			</Card>
			<Card>
				<StatCard value={hours(data.tasks.total_duration_hours)} label="Hours Tracked" color="info">
					{#snippet icon()}<Clock class="w-4 h-4" />{/snippet}
				</StatCard>
			</Card>
			<Card>
				<StatCard value={streaks?.total_current_streak_days ?? data.goals.total_streak_days} label="Active Days" unit="d" color="warning">
					{#snippet icon()}<CalendarDays class="w-4 h-4" />{/snippet}
				</StatCard>
			</Card>
			<Card>
				<StatCard value={data.goals.total_streak_days} label="Current Streaks" unit="d" color="primary">
					{#snippet icon()}<Flame class="w-4 h-4" />{/snippet}
				</StatCard>
			</Card>
		</div>

		<!-- Productivity Section -->
		<Card>
			<div class="flex flex-col gap-4">
				<SectionHeader title="Productivity" color="primary" subtitle="Task completion over time">
					{#snippet icon()}<TrendingUp class="w-4 h-4 text-primary" />{/snippet}
				</SectionHeader>

				<!-- Task counts row -->
				<div class="grid grid-cols-3 gap-3">
					<div class="bg-base-200/40 rounded-box p-3 text-center">
						<div class="flex items-center justify-center gap-1 text-success mb-1">
							<CheckCircle2 class="w-3.5 h-3.5" />
							<span class="text-xs font-medium">Completed</span>
						</div>
						<p class="text-xl font-bold font-mono">{data.tasks.completed_tasks}</p>
					</div>
					<div class="bg-base-200/40 rounded-box p-3 text-center">
						<div class="flex items-center justify-center gap-1 text-warning mb-1">
							<Clock class="w-3.5 h-3.5" />
							<span class="text-xs font-medium">Postponed</span>
						</div>
						<p class="text-xl font-bold font-mono">{data.tasks.postponed_tasks}</p>
					</div>
					<div class="bg-base-200/40 rounded-box p-3 text-center">
						<div class="flex items-center justify-center gap-1 text-error mb-1">
							<ListTodo class="w-3.5 h-3.5" />
							<span class="text-xs font-medium">Canceled</span>
						</div>
						<p class="text-xl font-bold font-mono">{data.tasks.abandoned_tasks}</p>
					</div>
				</div>

				<!-- Focus Score -->
				<div class="flex items-center gap-3">
					<div class="flex-1">
						<div class="flex items-center justify-between text-xs mb-1">
							<span class="text-base-content/60">Focus Score</span>
							<span class="font-mono font-bold">{Math.round(data.tasks.focus_score)}/100</span>
						</div>
						<div class="h-2 bg-base-200 rounded-full overflow-hidden">
							<div class="h-full rounded-full bg-primary transition-all" style="width: {data.tasks.focus_score}%"></div>
						</div>
					</div>
				</div>
			</div>
		</Card>

		<!-- Time Distribution -->
		{#if data.categories.distribution.length > 0}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Time" color="secondary" subtitle="Where your hours went">
						{#snippet icon()}<PieChart class="w-4 h-4 text-secondary" />{/snippet}
					</SectionHeader>
					<DonutChart distribution={data.categories.distribution} totalHours={data.categories.total_hours} />
					<!-- Per-category bars -->
					<div class="space-y-2">
						{#each data.categories.distribution.slice(0, 8) as cat, i}
							<div class="flex items-center gap-2">
								<span class="text-sm font-medium w-24 truncate">{cat.category_name}</span>
								<div class="flex-1 h-2.5 bg-base-200 rounded-full overflow-hidden">
									<div
										class="h-full rounded-full transition-all"
										style="width: {cat.percentage}%; background: {CHART_COLORS[i % CHART_COLORS.length]};"
									></div>
								</div>
								<span class="text-xs text-base-content/50 font-mono shrink-0 w-16 text-right">{cat.hours.toFixed(1)}h · {Math.round(cat.percentage)}%</span>
							</div>
						{/each}
					</div>
				</div>
			</Card>
		{/if}

		<!-- Mood Section -->
		{#if data.emotions.average_valence !== 0 || data.emotions.average_arousal !== 0}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Mood" color="error" subtitle="Emotional patterns">
						{#snippet icon()}<Heart class="w-4 h-4 text-error" />{/snippet}
					</SectionHeader>

					<!-- Valence/Arousal -->
					<div class="grid grid-cols-2 gap-3 text-sm">
						<div class="bg-base-200/40 rounded-box px-3 py-2">
							<p class="text-xs text-base-content/50">Avg Valence</p>
							<p class="font-mono text-lg font-bold">{data.emotions.average_valence.toFixed(2)}</p>
						</div>
						<div class="bg-base-200/40 rounded-box px-3 py-2">
							<p class="text-xs text-base-content/50">Avg Arousal</p>
							<p class="font-mono text-lg font-bold">{data.emotions.average_arousal.toFixed(2)}</p>
						</div>
					</div>

					<!-- Quadrant distribution bar -->
					{#if quadrantSegments.length > 0}
						<div class="space-y-2">
							<div class="flex h-3 rounded-full overflow-hidden bg-base-200">
								{#each quadrantSegments as seg (seg.key)}
									<div
										style="width: {seg.pct}%; background: {seg.color};"
										class="transition-all"
										title="{seg.key}: {seg.value}"
									></div>
								{/each}
							</div>
							<div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-base-content/60">
								{#each quadrantSegments as seg (seg.key)}
									<span class="flex items-center gap-1.5 capitalize">
										<span class="w-2 h-2 rounded-full" style="background: {seg.color}"></span>
										{seg.key} ({Math.round(seg.pct)}%)
									</span>
								{/each}
							</div>
						</div>
					{/if}

					{#if data.emotions.dominant_quadrant}
						<p class="text-sm">
							<span class="text-base-content/50">Dominant:</span>
							<span class="font-medium capitalize ml-1" style="color: {quadrantColor(data.emotions.dominant_quadrant)}">{data.emotions.dominant_quadrant}</span>
						</p>
					{/if}

					{#if data.emotions.top_emotions && data.emotions.top_emotions.length > 0}
						<div class="flex flex-wrap gap-2">
							{#each data.emotions.top_emotions.slice(0, 5) as emotion}
								<span class="badge badge-sm gap-1 bg-base-200/60">
									{emotion.emotion_name}
									<span class="text-xs opacity-60">×{emotion.count}</span>
								</span>
							{/each}
						</div>
					{/if}
				</div>
			</Card>
		{/if}

		<!-- Goals Section -->
		{#if data.goals.goal_progress.length > 0}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Goals" color="secondary" subtitle="Progress toward your goals">
						{#snippet icon()}<Target class="w-4 h-4 text-secondary" />{/snippet}
					</SectionHeader>
					<div class="space-y-3">
						{#each data.goals.goal_progress as goal}
							<div class="space-y-1.5">
								<div class="flex items-center justify-between text-sm">
									<span class="font-medium truncate">{goal.goal_title}</span>
									<span class="text-base-content/50 shrink-0 ml-2 font-mono">{Math.round(goal.progress)}%</span>
								</div>
								<div class="h-2 bg-base-200 rounded-full overflow-hidden">
									<div
										class="h-full rounded-full transition-all"
										style="width: {goal.progress}%; background: {CHART_COLORS[0]};"
									></div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</Card>
		{/if}

		<!-- Activity Heatmap (full width) -->
		<Card>
			<div class="flex flex-col gap-3">
				<SectionHeader title="Activity" color="primary" subtitle="Daily activity over time">
					{#snippet icon()}<Activity class="w-4 h-4 text-primary" />{/snippet}
				</SectionHeader>
				<ActivityHeatmap />
			</div>
		</Card>

		<!-- Streaks Leaderboard -->
		{#if data.goals.streak_leaders.length > 0}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Streak Leaders" color="warning" subtitle="Longest active streaks">
						{#snippet icon()}<Zap class="w-4 h-4 text-warning" />{/snippet}
					</SectionHeader>
					<div class="space-y-2">
						{#each data.goals.streak_leaders as streak, i}
							<div class="flex items-center justify-between py-2 border-b border-base-content/5 last:border-0">
								<div class="flex items-center gap-3">
									{#if i === 0}
										<span class="w-6 h-6 rounded-full bg-warning/20 flex items-center justify-center text-xs font-bold text-warning">1</span>
									{:else}
										<span class="w-6 h-6 rounded-full bg-base-200 flex items-center justify-center text-xs font-medium text-base-content/50">{i + 1}</span>
									{/if}
									<span class="font-medium text-sm truncate">{streak.goal_title}</span>
								</div>
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
			</Card>
		{/if}
	{/if}
</div>
