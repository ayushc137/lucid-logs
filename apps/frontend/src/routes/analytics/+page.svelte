<script lang="ts">
import {
	getAnalyticsDashboard,
	getAnalyticsStreaks,
	type DashboardResponse,
	type StreaksResponse,
} from '$lib/api';
import ActivityHeatmap from '$lib/components/analytics/ActivityHeatmap.svelte';
import DonutChart from '$lib/components/analytics/DonutChart.svelte';
import MoodTrend from '$lib/components/analytics/MoodTrend.svelte';
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
import { writable } from 'svelte/store';
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

type PeriodOption = 'week' | 'month' | 'quarter' | 'custom';

let period = $state<PeriodOption>('week');

// Custom date range state (date input format: YYYY-MM-DD)
let customStart = $state('');
let customEnd = $state('');

const periodOptions: { value: PeriodOption; label: string; days: number }[] = [
	{ value: 'week', label: '7d', days: 7 },
	{ value: 'month', label: '30d', days: 30 },
	{ value: 'quarter', label: '90d', days: 90 },
	{ value: 'custom', label: 'Custom', days: 0 },
];

// ── Custom date range helpers ──────────────────────────────────

function toLocalDateInput(d: Date): string {
	const y = d.getFullYear();
	const m = String(d.getMonth() + 1).padStart(2, '0');
	const day = String(d.getDate()).padStart(2, '0');
	return `${y}-${m}-${day}`;
}

function initCustomDates() {
	const end = new Date();
	const start = new Date();
	start.setDate(start.getDate() - 30);
	customStart = toLocalDateInput(start);
	customEnd = toLocalDateInput(end);
}

// Initialize on first select of custom
let customInitialized = false;
$effect(() => {
	if (period === 'custom' && !customInitialized) {
		initCustomDates();
		customInitialized = true;
	}
});

// Build RFC3339 start/end for API from custom date inputs
const customRangeParams = $derived.by(() => {
	if (!customStart || !customEnd) return null;
	// Parse as local time
	const [sy, sm, sd] = customStart.split('-').map(Number);
	const [ey, em, ed] = customEnd.split('-').map(Number);
	if (!sy || !ey) return null;
	const startD = new Date(sy, sm - 1, sd, 0, 0, 0, 0);
	const endD = new Date(ey, em - 1, ed, 23, 59, 59, 0);
	if (startD > endD) return null;
	return {
		start_date: startD.toISOString(),
		end_date: endD.toISOString(),
	};
});

// The period string sent to the API
const apiPeriod = $derived(period === 'custom' ? 'custom' : period);

// ── Heatmap range ──────────────────────────────────────────────

const heatmapRange = $derived.by(() => {
	if (period === 'custom') {
		if (!customRangeParams) return null;
		return { startDate: customRangeParams.start_date, endDate: customRangeParams.end_date };
	}
	const days = periodOptions.find((o) => o.value === period)?.days ?? 7;
	const end = new Date();
	const start = new Date();
	start.setDate(start.getDate() - days);
	return { startDate: start.toISOString(), endDate: end.toISOString() };
});

// ── Dashboard query (writable-store bridge for reactivity) ─────

function makeOptions() {
	// For preset periods, call with just the period string.
	// For custom, call with start_date/end_date params.
	// Use explicit string types to avoid TypeScript literal-type union issues.
	const p: string = period;
	const s: string = period === 'custom' ? customStart : '';
	const e: string = period === 'custom' ? customEnd : '';
	if (period === 'custom') {
		const range = customRangeParams;
		return {
			queryKey: ['analytics', 'dashboard', p, s, e] as const,
			queryFn: () => {
				if (!range) return Promise.reject(new Error('Invalid date range'));
				return getAnalyticsDashboard('custom', {
					start_date: range.start_date,
					end_date: range.end_date,
				});
			},
			enabled: !!range,
		};
	}
	return {
		queryKey: ['analytics', 'dashboard', p, s, e] as const,
		queryFn: () => getAnalyticsDashboard(period),
	};
}

const dashboardOptions$ = writable(makeOptions());

// Re-emit when period or custom dates change so queryKey + queryFn capture the new value.
$effect(() => {
	period;
	customStart;
	customEnd;
	dashboardOptions$.set(makeOptions());
});

const dashboardQuery = createQuery(dashboardOptions$);

const streaksQuery = createQuery({
	queryKey: ['analytics', 'streaks'],
	queryFn: () => getAnalyticsStreaks(),
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

// ── Productivity proportions (completed/postponed/canceled) ────

const taskProportions = $derived.by(() => {
	const t = data?.tasks;
	if (!t) return null;
	const total = t.completed_tasks + t.postponed_tasks + t.abandoned_tasks;
	if (total === 0) return null;
	return {
		completed: (t.completed_tasks / total) * 100,
		postponed: (t.postponed_tasks / total) * 100,
		canceled: (t.abandoned_tasks / total) * 100,
		total,
	};
});

// ── Mood quadrant human-readable info ──────────────────────────
interface QuadrantInfo {
	label: string;
	emoji: string;
	description: string;
	words: string;
	tips: string[];
}

const QUADRANT_INFO: Record<string, QuadrantInfo> = {
	yellow: {
		label: 'Energized',
		emoji: '⚡',
		description: 'Positive and high-energy — you\'ve been feeling motivated and engaged.',
		words: 'excited, motivated, joyful',
		tips: [
			'Channel this energy into your hardest goals',
			'High-energy periods are great for starting new habits',
			'Use this momentum to tackle creative or challenging work',
		],
	},
	green: {
		label: 'Calm',
		emoji: '🌿',
		description: 'Positive and relaxed — a peaceful, content state.',
		words: 'relaxed, content, at peace',
		tips: [
			'Great balance — keep the rhythm going',
			'This is a good state for planning and reflection',
			'Enjoy it! Consistent calm states build resilience',
		],
	},
	red: {
		label: 'Stressed',
		emoji: '🔥',
		description: 'High-energy but negative — tension, frustration, or anxiety.',
		words: 'anxious, frustrated, tense',
		tips: [
			'Try shorter work blocks with frequent breaks',
			'Consider a walk between intense tasks',
			'Notice which task categories correlate with stress',
		],
	},
	blue: {
		label: 'Low',
		emoji: '💧',
		description: 'Low energy and low mood — tired, drained, or down.',
		words: 'tired, sad, drained',
		tips: [
			'Small wins matter — try breaking tasks into smaller pieces',
			'A short walk or some sunlight can help lift your energy',
			'Be gentle with yourself — low periods are temporary',
		],
	},
};

// Period label for mood summary
const periodLabel = $derived(period === 'week' ? 'week' : period === 'month' ? 'month' : period === 'quarter' ? '3 months' : 'period');

// Dominant quadrant info object
const dominantInfo = $derived.by(() => {
	const dq = data?.emotions?.dominant_quadrant;
	if (!dq) return null;
	return QUADRANT_INFO[dq] ?? null;
});

// Mood stability in words
const stabilityWord = $derived.by(() => {
	const s = data?.emotions?.mood_stability;
	if (s === undefined || s === null) return '';
	if (s > 0.7) return 'very stable';
	if (s >= 0.4) return 'fairly steady';
	return 'swinging between states';
});

// Valence in words
const valenceWord = $derived.by(() => {
	const v = data?.emotions?.average_valence ?? 0;
	if (v > 0.2) return 'positive';
	if (v < -0.2) return 'negative';
	return 'neutral';
});

// Arousal in words
const arousalWord = $derived.by(() => {
	const a = data?.emotions?.average_arousal ?? 0;
	if (a > 0.2) return 'high';
	if (a < -0.2) return 'low';
	return 'moderate';
});

// Trend direction from trend data
const trendWord = $derived.by(() => {
	const trend = data?.emotions?.trend;
	if (!trend || trend.length < 4) return '';
	const mid = Math.floor(trend.length / 2);
	const firstAvg = trend.slice(0, mid).reduce((s, d) => s + d.valence, 0) / mid;
	const secondAvg = trend.slice(mid).reduce((s, d) => s + d.valence, 0) / (trend.length - mid);
	const diff = secondAvg - firstAvg;
	if (diff > 0.1) return 'improving';
	if (diff < -0.1) return 'declining';
	return 'steady';
});

// Mood summary sentence
const moodSummary = $derived.by(() => {
	if (!dominantInfo) return '';
	const parts: string[] = [];
	parts.push(`This ${periodLabel} you've mostly felt **${dominantInfo.label}** ${dominantInfo.emoji} — ${dominantInfo.description}`);
	if (trendWord) {
		const trendClause = trendWord === 'steady' ? 'holding steady' : trendWord === 'improving' ? 'improving' : 'declining';
		parts.push(`Your mood has been ${stabilityWord} and ${trendClause}.`);
	} else {
		parts.push(`Your mood has been ${stabilityWord}.`);
	}
	return parts.join(' ');
});

// Quadrant segments enriched with info
const quadrantBreakdown = $derived.by(() => {
	const qd = data?.emotions?.quadrant_distribution;
	if (!qd) return [];
	const total = qd.yellow + qd.green + qd.red + qd.blue;
	if (total === 0) return [];
	const order = ['yellow', 'green', 'red', 'blue'] as const;
	return order
		.map((key) => {
			const value = qd[key];
			const info = QUADRANT_INFO[key];
			return {
				key,
				value,
				pct: (value / total) * 100,
				info,
			};
		})
		.filter((s) => s.value > 0)
		.sort((a, b) => b.value - a.value);
});
</script>

<svelte:head>
	<title>Analytics - Lucid Logs</title>
</svelte:head>

<div class="flex flex-col gap-6 px-4 py-4 sm:px-6 sm:py-6 max-w-5xl mx-auto w-full">
	<!-- Sticky Top Bar -->
	<div class="sticky top-0 z-10 -mx-4 sm:-mx-6 px-4 sm:px-6 py-3 bg-base-200/95 backdrop-blur-sm border-b border-base-content/5">
		<div class="flex items-center justify-between gap-3 flex-wrap">
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
		<!-- Custom date range pickers -->
		{#if period === 'custom'}
			<div class="flex items-center gap-2 mt-3 flex-wrap">
				<div class="flex items-center gap-1.5">
					<label for="custom-start" class="text-xs text-base-content/60 font-medium">From</label>
					<input
						id="custom-start"
						type="date"
						class="input input-sm input-bordered"
						bind:value={customStart}
					/>
				</div>
				<div class="flex items-center gap-1.5">
					<label for="custom-end" class="text-xs text-base-content/60 font-medium">To</label>
					<input
						id="custom-end"
						type="date"
						class="input input-sm input-bordered"
						bind:value={customEnd}
					/>
				</div>
			</div>
		{/if}
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

		<!-- Productivity + Time (2-col on desktop) -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
			<!-- Productivity Section -->
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Productivity" color="primary" subtitle="Task completion over time">
						{#snippet icon()}<TrendingUp class="w-4 h-4 text-primary" />{/snippet}
					</SectionHeader>

					<!-- Proportions bar (completed vs postponed vs canceled) -->
					{#if taskProportions}
						<div class="space-y-1.5">
							<div class="flex h-3 rounded-full overflow-hidden bg-base-200">
								<div
									class="bg-success transition-all"
									style="width: {taskProportions.completed}%"
									title="Completed: {data.tasks.completed_tasks} ({Math.round(taskProportions.completed)}%)"
								></div>
								<div
									class="bg-warning transition-all"
									style="width: {taskProportions.postponed}%"
									title="Postponed: {data.tasks.postponed_tasks} ({Math.round(taskProportions.postponed)}%)"
								></div>
								<div
									class="bg-error transition-all"
									style="width: {taskProportions.canceled}%"
									title="Canceled: {data.tasks.abandoned_tasks} ({Math.round(taskProportions.canceled)}%)"
								></div>
							</div>
						</div>
					{/if}

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
		</div>

		<!-- Mood Section (full width) -->
		{#if data.emotions.average_valence !== 0 || data.emotions.average_arousal !== 0}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Mood" color="error" subtitle="Emotional patterns">
						{#snippet icon()}<Heart class="w-4 h-4 text-error" />{/snippet}
					</SectionHeader>

					<!-- Mood summary sentence -->
					{#if moodSummary}
						<p class="text-sm leading-relaxed text-base-content/80">
							{#each moodSummary.split('**') as part, i}
								{#if i % 2 === 1}<strong>{part}</strong>{:else}{part}{/if}
							{/each}
						</p>
					{/if}

					<!-- Quadrant distribution bar (above the detailed breakdown) -->
					{#if quadrantSegments.length > 0}
						<div class="space-y-2">
							<div class="flex h-3 rounded-full overflow-hidden bg-base-200">
								{#each quadrantSegments as seg (seg.key)}
									{@const qi = QUADRANT_INFO[seg.key]}
									<div
										style="width: {seg.pct}%; background: {seg.color};"
										class="transition-all"
										title={qi ? `{qi.label}: ${Math.round(seg.pct)}% (${seg.value} entries)` : `${seg.key}: ${seg.value}`}
									></div>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Quadrant breakdown with meanings -->
					{#if quadrantBreakdown.length > 0}
						<div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
							{#each quadrantBreakdown as item (item.key)}
								<div class="flex items-start gap-2.5 p-2.5 rounded-lg bg-base-200/40">
									<span class="w-3 h-3 rounded-full mt-0.5 shrink-0" style="background: {QUADRANT_COLORS[item.key]}"></span>
									<div class="min-w-0">
										<div class="flex items-baseline gap-1.5 flex-wrap">
											<span class="text-sm font-semibold">{item.info.emoji} {item.info.label}</span>
											<span class="text-xs text-base-content/50 font-mono">{Math.round(item.pct)}%</span>
										</div>
										<p class="text-xs text-base-content/50 mt-0.5">{item.info.words}</p>
									</div>
								</div>
							{/each}
						</div>
					{/if}

					<!-- Dominant mood analysis paragraph -->
					{#if dominantInfo}
						<div class="space-y-2 text-sm">
							<p class="leading-relaxed">
								Your dominant mood this {periodLabel} was <strong style="color: {quadrantColor(data.emotions.dominant_quadrant)}">{dominantInfo.label}</strong> {dominantInfo.emoji}. {dominantInfo.description}
							</p>
							{#if data.emotions.top_emotions && data.emotions.top_emotions.length > 0}
								<p class="text-base-content/60">
									Most frequent feelings:
									{#each data.emotions.top_emotions.slice(0, 3) as emo, i}
										{#if i > 0}, {/if}{emo.emotion_name}
									{/each}
								</p>
							{/if}
							<p class="text-base-content/60 text-xs">
								Overall positivity: <span class="font-medium">{valenceWord}</span> · Energy level: <span class="font-medium">{arousalWord}</span>
							</p>
						</div>
					{/if}

					<!-- Friendly suggestions callout -->
					{#if dominantInfo && dominantInfo.tips.length > 0}
						<div class="bg-primary/5 border border-primary/10 rounded-box p-3 space-y-2">
							<p class="text-xs font-semibold text-base-content/60 uppercase tracking-wide">Suggestions</p>
							<ul class="space-y-1.5">
								{#each dominantInfo.tips as tip}
									<li class="text-sm text-base-content/70 flex items-start gap-2">
										<span class="text-primary mt-0.5">•</span>
										<span>{tip}</span>
									</li>
								{/each}
							</ul>
						</div>
					{/if}

					<!-- Mood trend chart -->
					{#if data.emotions.trend && data.emotions.trend.length > 1}
						<MoodTrend trend={data.emotions.trend} />
					{/if}
				</div>
			</Card>
		{/if}

		<!-- Goals + Streak Leaders (2-col on desktop) -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
			{#if data.goals.goal_progress.length > 0}
				<Card>
					<div class="flex flex-col gap-3">
						<SectionHeader title="Goals" color="secondary" subtitle="Progress toward your goals">
							{#snippet icon()}<Target class="w-4 h-4 text-secondary" />{/snippet}
						</SectionHeader>
						<div class="space-y-2">
							{#each data.goals.goal_progress.slice(0, 5) as goal, i}
								<div class="flex items-center gap-2.5">
									<div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-secondary/10 text-secondary">
										<Target class="w-3.5 h-3.5" />
									</div>
									<div class="flex-1 min-w-0">
										<div class="flex items-center justify-between gap-2 mb-1">
											<span class="text-sm font-medium truncate">{goal.goal_title}</span>
											<span class="text-xs text-base-content/50 shrink-0 font-mono">{Math.round(goal.progress)}%</span>
										</div>
										<div class="h-1.5 bg-base-200 rounded-full overflow-hidden">
											<div
												class="h-full rounded-full transition-all"
												style="width: {goal.progress}%; background: {CHART_COLORS[i % CHART_COLORS.length]};"
											></div>
										</div>
									</div>
								</div>
							{/each}
						</div>
						{#if data.goals.goal_progress.length > 5}
							<a href="/goals" class="text-xs text-secondary font-medium hover:underline mt-1">View all goals →</a>
						{/if}
					</div>
				</Card>
			{/if}

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
		</div>

		<!-- Activity Heatmap (full width) -->
		<Card>
			<div class="flex flex-col gap-3">
				<SectionHeader title="Activity" color="primary" subtitle="Daily activity over time">
					{#snippet icon()}<Activity class="w-4 h-4 text-primary" />{/snippet}
				</SectionHeader>
				<ActivityHeatmap startDate={heatmapRange?.startDate} endDate={heatmapRange?.endDate} />
			</div>
		</Card>
	{/if}
</div>
