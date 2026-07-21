<script lang="ts">
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { page } from '$app/stores';
import {
	deleteRetrospective,
	getRetrospective,
	regenerateInsights,
	updateRetrospective,
	getUserPreferences,
	type UpdateRetroRequest,
} from '$lib/api';
import {
	Card,
	ConfirmDialog,
	ErrorAlert,
	LoadingCard,
	SectionHeader,
	StatCard,
} from '$lib/components/ui';
import DonutChart from '$lib/components/analytics/DonutChart.svelte';
import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
import {
	ArrowLeft,
	BookOpen,
	CheckCircle2,
	CircleDashed,
	Clock,
	Flame,
	Heart,
	Lightbulb,
	ListTodo,
	Plus,
	Sparkles,
	Target,
	TrendingDown,
	TrendingUp,
	Trash2,
	Save,
	ThumbsDown,
	ThumbsUp,
	Award,
} from 'lucide-svelte';
import { QUADRANT_COLORS, quadrantColor, CHART_COLORS } from '$lib/utils/chart-colors';

const queryClient = useQueryClient();

function getRetroId(): string {
	return $page.params.id as string;
}

let showDeleteConfirm = $state(false);

// Editable reflection state
let whatWentWell = $state('');
let whatDidntGoWell = $state('');
let whatLearned = $state('');
let proudOf = $state('');
let changeTomorrow = $state('');
let additionalNotes = $state('');

const retroQuery = createQuery({
	queryKey: ['retrospective', $page.params.id],
	queryFn: () => getRetrospective($page.params.id as string),
	enabled: browser && !!$page.params.id,
});

// Fetch user preferences once to check if AI is configured
const prefsQuery = createQuery({
	queryKey: ['user-preferences'],
	queryFn: () => getUserPreferences(),
	enabled: browser,
});
const aiConfigured = $derived($prefsQuery.data?.preferences?.ai?.has_key ?? false);

// Populate form when retro loads
$effect(() => {
	const retro = $retroQuery.data;
	if (retro && retro.user_content) {
		const uc = retro.user_content;
		whatWentWell = uc.what_went_well ?? '';
		whatDidntGoWell = uc.what_didnt_go_well ?? '';
		whatLearned = uc.what_learned ?? '';
		proudOf = uc.proud_of ?? '';
		changeTomorrow = uc.change_tomorrow ?? '';
		additionalNotes = uc.additional_notes ?? '';
	}
});

const updateMutation = createMutation({
	mutationFn: (data: UpdateRetroRequest) => updateRetrospective(getRetroId(), data),
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ['retrospective', getRetroId()] });
		queryClient.invalidateQueries({ queryKey: ['retrospectives'] });
	},
});

const deleteMutation = createMutation({
	mutationFn: () => deleteRetrospective(getRetroId()),
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ['retrospectives'] });
		goto('/retrospectives');
	},
});

const regenerateMutation = createMutation({
	mutationFn: () => regenerateInsights(getRetroId()),
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ['retrospective', getRetroId()] });
	},
});

function handleSave(markComplete: boolean) {
	$updateMutation.mutate({
		user_content: {
			what_went_well: whatWentWell,
			what_didnt_go_well: whatDidntGoWell,
			what_learned: whatLearned,
			proud_of: proudOf,
			change_tomorrow: changeTomorrow,
			additional_notes: additionalNotes,
		},
		status: markComplete ? 'completed' : 'draft',
	});
}

function formatDate(s: string): string {
	return new Date(s).toLocaleDateString(undefined, {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
	});
}

function fmtDateShort(s: string): string {
	return new Date(s).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

// Derived data — non-null asserted where guarded by has* checks in template
const retro = $derived($retroQuery.data);
const summary = $derived(retro?.auto_summary);
const mood = $derived(summary?.mood);
const tasks = $derived(summary?.tasks);
const goals = $derived(summary?.goals);
const habits = $derived(summary?.habits);
const categories = $derived(summary?.categories);
const streaks = $derived(habits?.streaks);

// Quadrant distribution bar segments
const quadrantSegments = $derived.by(() => {
	const qd = mood?.quadrant_distribution;
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

// Donut chart data for categories
const categoryDonutData = $derived.by(() => {
	const td = categories?.time_distribution;
	if (!td || td.length === 0) return null;
	return {
		distribution: td.map((c) => ({
			category_id: c.category,
			category_name: c.category,
			task_count: 0,
			hours: c.hours,
			percentage: c.percentage,
		})),
		totalHours: td.reduce((sum, c) => sum + c.hours, 0),
	};
});

// Active streaks count
const activeStreaks = $derived.by(() => {
	const s = streaks;
	if (!s) return 0;
	return (s.continued?.length ?? 0) + (s.started?.length ?? 0);
});

const hasMoodData = !!mood && (mood.average_valence !== 0 || mood.average_arousal !== 0 || mood.dominant_quadrant !== '');
const hasTimeData = !!categories?.time_distribution && categories.time_distribution.length > 0;
const hasTaskData = !!tasks && (tasks.completed > 0 || tasks.postponed > 0 || tasks.canceled > 0);
const hasHabitData = !!habits && ((habits.met?.length ?? 0) > 0 || (habits.partially_met?.length ?? 0) > 0 || (habits.missed?.length ?? 0) > 0);
const hasStreakData = !!streaks && ((streaks.continued?.length ?? 0) > 0 || (streaks.broken?.length ?? 0) > 0 || (streaks.started?.length ?? 0) > 0);
const hasGoalData = !!goals?.net_impact && goals.net_impact.length > 0;

const hasAIContent = !!summary && ((summary.insights?.length ?? 0) > 0 || !!summary.ai_narrative);

// Reflection prompts config
const reflectionPrompts = [
	{ key: 'what_went_well', label: 'What went well?', icon: ThumbsUp, color: 'text-success' },
	{ key: 'what_didnt_go_well', label: "What didn't go well?", icon: ThumbsDown, color: 'text-error' },
	{ key: 'what_learned', label: 'What did you learn?', icon: Lightbulb, color: 'text-info' },
	{ key: 'proud_of', label: 'What are you proud of?', icon: Award, color: 'text-warning' },
	{ key: 'change_tomorrow', label: 'What will you change tomorrow?', icon: TrendingUp, color: 'text-primary' },
	{ key: 'additional_notes', label: 'Additional notes', icon: BookOpen, color: 'text-base-content/50' },
] as const;
</script>

<svelte:head>
	<title>Retrospective - Lucid Logs</title>
</svelte:head>

<div class="flex flex-col gap-6 px-4 py-4 sm:px-6 sm:py-6 max-w-3xl mx-auto w-full pb-24">
	<button class="btn btn-ghost btn-sm gap-2 self-start" onclick={() => goto('/retrospectives')}>
		<ArrowLeft class="h-4 w-4" />
		Back
	</button>

	{#if $retroQuery.isLoading}
		<LoadingCard />
	{:else if $retroQuery.isError}
		<ErrorAlert
			message="Failed to load retrospective"
			onRetry={() => $retroQuery.refetch()}
		/>
	{:else if retro}
		<!-- Hero -->
		<div class="flex items-center gap-3.5">
			<div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
				<BookOpen class="w-5.5 h-5.5" />
			</div>
			<div class="flex-1 min-w-0">
				<div class="flex items-center gap-2 flex-wrap">
					<h1 class="text-2xl font-semibold tracking-tight capitalize">{retro.retro_type} Retrospective</h1>
					<span class="badge badge-sm badge-outline capitalize">{retro.retro_type}</span>
					<span class="badge badge-sm gap-1" class:badge-success={retro.status === 'completed'} class:badge-ghost={retro.status !== 'completed'}>
						{#if retro.status === 'completed'}
							<CheckCircle2 class="w-3 h-3" /> Completed
						{:else}
							<CircleDashed class="w-3 h-3" /> Draft
						{/if}
					</span>
				</div>
				<p class="text-sm text-base-content/55 mt-0.5">{fmtDateShort(retro.start_date)} – {formatDate(retro.end_date)}</p>
			</div>
		</div>

		<!-- Stat Strip -->
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
			<div class="card bg-base-100 border border-base-content/5 shadow-sm">
				<div class="card-body p-3 gap-0.5">
					<StatCard value={tasks?.completed ?? 0} label="Tasks Done" color="primary">
						{#snippet icon()}<ListTodo class="w-4 h-4" />{/snippet}
					</StatCard>
				</div>
			</div>
			<div class="card bg-base-100 border border-base-content/5 shadow-sm">
				<div class="card-body p-3 gap-0.5">
					<StatCard value={`${(tasks?.total_duration_hours ?? 0).toFixed(1)}h`} label="Tracked" color="info">
						{#snippet icon()}<Clock class="w-4 h-4" />{/snippet}
					</StatCard>
				</div>
			</div>
			<div class="card bg-base-100 border border-base-content/5 shadow-sm">
				<div class="card-body p-3 gap-0.5">
					<StatCard value={(mood?.average_valence ?? 0).toFixed(2)} label="Avg Mood" color="error">
						{#snippet icon()}<Heart class="w-4 h-4" />{/snippet}
					</StatCard>
				</div>
			</div>
			<div class="card bg-base-100 border border-base-content/5 shadow-sm">
				<div class="card-body p-3 gap-0.5">
					<StatCard value={activeStreaks} label="Active Streaks" color="warning">
						{#snippet icon()}<Flame class="w-4 h-4" />{/snippet}
					</StatCard>
				</div>
			</div>
		</div>

		<!-- AI Insights Card -->
		<Card>
			<div class="flex flex-col gap-3">
				<SectionHeader title="AI Insights" color="primary">
					{#snippet icon()}<Sparkles class="w-4 h-4 text-primary" />{/snippet}
					{#snippet actions()}
						{#if aiConfigured}
							<button
								class="btn btn-ghost btn-xs gap-1"
								disabled={$regenerateMutation.isPending}
								onclick={() => $regenerateMutation.mutate()}
							>
								{#if $regenerateMutation.isPending}
									<span class="loading loading-spinner loading-xs"></span>
								{:else}
									<Sparkles class="w-3 h-3" />
								{/if}
								Regenerate
							</button>
						{/if}
					{/snippet}
				</SectionHeader>

				{#if hasAIContent}
					{#if summary?.insights && summary.insights.length > 0}
						<ul class="space-y-1.5">
							{#each summary.insights as insight}
								<li class="flex items-start gap-2 text-sm">
									<Sparkles class="w-3.5 h-3.5 text-primary/60 shrink-0 mt-0.5" />
									<span>{insight}</span>
								</li>
							{/each}
						</ul>
					{/if}
					{#if summary?.ai_narrative}
						<p class="text-sm text-base-content/70 leading-relaxed">{summary.ai_narrative}</p>
					{/if}
				{:else}
					<div class="flex items-center gap-2 text-sm text-base-content/50">
						<Sparkles class="w-4 h-4 shrink-0" />
						<span>AI insights unavailable —</span>
						<a href="/settings" class="link link-primary">configure in Settings</a>
					</div>
				{/if}
				{#if $regenerateMutation.isError}
					<p class="text-xs text-error">Failed to regenerate insights. Check your AI configuration.</p>
				{/if}
			</div>
		</Card>

		<!-- Mood Section -->
		{#if hasMoodData && mood}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Mood" color="error" subtitle="Emotional patterns this period">
						{#snippet icon()}<Heart class="w-4 h-4 text-error" />{/snippet}
					</SectionHeader>

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

					<div class="grid grid-cols-2 gap-3 text-sm">
						<div class="bg-base-200/40 rounded-box px-3 py-2">
							<p class="text-xs text-base-content/50">Avg Valence</p>
							<p class="font-mono text-lg font-bold">{mood.average_valence.toFixed(2)}</p>
						</div>
						<div class="bg-base-200/40 rounded-box px-3 py-2">
							<p class="text-xs text-base-content/50">Avg Arousal</p>
							<p class="font-mono text-lg font-bold">{mood.average_arousal.toFixed(2)}</p>
						</div>
					</div>

					{#if mood.dominant_quadrant}
						<p class="text-sm">
							<span class="text-base-content/50">Dominant:</span>
							<span class="font-medium capitalize ml-1" style="color: {quadrantColor(mood.dominant_quadrant)}">{mood.dominant_quadrant}</span>
						</p>
					{/if}

					{#if mood.notable_spikes && mood.notable_spikes.length > 0}
						<div class="space-y-1">
							<p class="text-xs font-medium text-base-content/60">Notable Spikes</p>
							{#each mood.notable_spikes as spike}
								<div class="flex items-center gap-2 text-sm">
									<TrendingUp class="w-3.5 h-3.5 text-success shrink-0" />
									<span>{spike.emotion}</span>
									<span class="text-xs text-base-content/40">{fmtDateShort(spike.date)}</span>
								</div>
							{/each}
						</div>
					{/if}
					{#if mood.notable_dips && mood.notable_dips.length > 0}
						<div class="space-y-1">
							<p class="text-xs font-medium text-base-content/60">Notable Dips</p>
							{#each mood.notable_dips as dip}
								<div class="flex items-center gap-2 text-sm">
									<TrendingDown class="w-3.5 h-3.5 text-error shrink-0" />
									<span>{dip.emotion}</span>
									<span class="text-xs text-base-content/40">{fmtDateShort(dip.date)}</span>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</Card>
		{/if}

		<!-- Time Section -->
		{#if hasTimeData && categoryDonutData}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Time Distribution" color="secondary" subtitle="Where your hours went">
						{#snippet icon()}<Clock class="w-4 h-4 text-secondary" />{/snippet}
					</SectionHeader>
					<DonutChart distribution={categoryDonutData.distribution} totalHours={categoryDonutData.totalHours} />
					{#if categories?.neglected && categories.neglected.length > 0}
						<div class="flex flex-wrap gap-2">
							{#each categories.neglected as neg}
								<span class="badge badge-sm badge-ghost gap-1">
									<Clock class="w-3 h-3" />
									{neg.category} ({neg.days_since_last_task}d idle)
								</span>
							{/each}
						</div>
					{/if}
				</div>
			</Card>
		{/if}

		<!-- Tasks Section -->
		{#if hasTaskData && tasks}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Tasks" color="primary" subtitle="Completion breakdown">
						{#snippet icon()}<ListTodo class="w-4 h-4 text-primary" />{/snippet}
					</SectionHeader>
					<div class="grid grid-cols-4 gap-2 text-center">
						<div class="bg-base-200/40 rounded-box py-2">
							<p class="text-xs text-base-content/50">Completed</p>
							<p class="font-mono text-lg font-bold text-success">{tasks.completed}</p>
						</div>
						<div class="bg-base-200/40 rounded-box py-2">
							<p class="text-xs text-base-content/50">Postponed</p>
							<p class="font-mono text-lg font-bold text-warning">{tasks.postponed}</p>
						</div>
						<div class="bg-base-200/40 rounded-box py-2">
							<p class="text-xs text-base-content/50">Canceled</p>
							<p class="font-mono text-lg font-bold text-error">{tasks.canceled}</p>
						</div>
						<div class="bg-base-200/40 rounded-box py-2">
							<p class="text-xs text-base-content/50">Not Started</p>
							<p class="font-mono text-lg font-bold text-base-content/40">{tasks.not_started ?? 0}</p>
						</div>
					</div>
					{#if tasks.by_category && tasks.by_category.length > 0}
						<div class="space-y-2">
							<p class="text-xs font-medium text-base-content/60">By Category</p>
							{#each tasks.by_category as cat, i}
								<div class="flex items-center gap-2">
									<span class="text-sm font-medium w-24 truncate">{cat.category}</span>
									<div class="flex-1 h-2 bg-base-200 rounded-full overflow-hidden">
										<div
											class="h-full rounded-full transition-all"
											style="width: {Math.min(100, cat.count * 10)}%; background: {CHART_COLORS[i % CHART_COLORS.length]};"
										></div>
									</div>
									<span class="text-xs text-base-content/50 font-mono shrink-0 w-12 text-right">{cat.count}</span>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</Card>
		{/if}

		<!-- Habits Section -->
		{#if hasHabitData && habits}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Habits" color="success" subtitle="Consistency this period">
						{#snippet icon()}<CheckCircle2 class="w-4 h-4 text-success" />{/snippet}
					</SectionHeader>
					{#if habits.met && habits.met.length > 0}
						<div class="space-y-1.5">
							<p class="text-xs font-medium text-success">Met ({habits.met.length})</p>
							<div class="flex flex-wrap gap-1.5">
								{#each habits.met as h}
									<span class="badge badge-sm badge-success badge-outline gap-1">
										<CheckCircle2 class="w-3 h-3" />
										{h.name}
										{#if h.success_rate}<span class="opacity-60">{Math.round(h.success_rate * 100)}%</span>{/if}
									</span>
								{/each}
							</div>
						</div>
					{/if}
					{#if habits.partially_met && habits.partially_met.length > 0}
						<div class="space-y-1.5">
							<p class="text-xs font-medium text-warning">Partially Met ({habits.partially_met.length})</p>
							<div class="flex flex-wrap gap-1.5">
								{#each habits.partially_met as h}
									<span class="badge badge-sm badge-warning badge-outline gap-1">
										{h.name}
										{#if h.success_rate}<span class="opacity-60">{Math.round(h.success_rate * 100)}%</span>{/if}
									</span>
								{/each}
							</div>
						</div>
					{/if}
					{#if habits.missed && habits.missed.length > 0}
						<div class="space-y-1.5">
							<p class="text-xs font-medium text-error">Missed ({habits.missed.length})</p>
							<div class="flex flex-wrap gap-1.5">
								{#each habits.missed as h}
									<span class="badge badge-sm badge-error badge-outline gap-1">
										{h.name}
									</span>
								{/each}
							</div>
						</div>
					{/if}
				</div>
			</Card>
		{/if}

		<!-- Streaks Section -->
		{#if hasStreakData && streaks}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Streaks" color="warning" subtitle="Habit streak changes">
						{#snippet icon()}<Flame class="w-4 h-4 text-warning" />{/snippet}
					</SectionHeader>
					<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
						{#if streaks.continued && streaks.continued.length > 0}
							<div class="bg-base-200/40 rounded-box p-3 space-y-2">
								<p class="text-xs font-medium text-warning flex items-center gap-1">
									<Flame class="w-3.5 h-3.5" /> Continued
								</p>
								{#each streaks.continued as s}
									<div class="text-sm">
										<span class="font-medium">{s.name}</span>
										<span class="text-base-content/50 ml-1">{s.now ?? s.streak}d</span>
									</div>
								{/each}
							</div>
						{/if}
						{#if streaks.broken && streaks.broken.length > 0}
							<div class="bg-base-200/40 rounded-box p-3 space-y-2">
								<p class="text-xs font-medium text-error flex items-center gap-1">
									<TrendingDown class="w-3.5 h-3.5" /> Broken
								</p>
								{#each streaks.broken as s}
									<div class="text-sm">
										<span class="font-medium">{s.name}</span>
										<span class="text-base-content/50 ml-1">was {s.was}d</span>
									</div>
								{/each}
							</div>
						{/if}
						{#if streaks.started && streaks.started.length > 0}
							<div class="bg-base-200/40 rounded-box p-3 space-y-2">
								<p class="text-xs font-medium text-success flex items-center gap-1">
									<Plus class="w-3.5 h-3.5" /> Started
								</p>
								{#each streaks.started as s}
									<div class="text-sm">
										<span class="font-medium">{s.name}</span>
										<span class="text-base-content/50 ml-1">{s.now ?? 1}d</span>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			</Card>
		{/if}

		<!-- Goals Section -->
		{#if hasGoalData && goals}
			<Card>
				<div class="flex flex-col gap-4">
					<SectionHeader title="Goals" color="secondary" subtitle="Net impact on your goals">
						{#snippet icon()}<Target class="w-4 h-4 text-secondary" />{/snippet}
					</SectionHeader>
					{#if goals.net_impact && goals.net_impact.length > 0}
						<div class="space-y-2">
							{#each goals.net_impact as goal}
								<div class="space-y-1">
									<div class="flex items-center justify-between text-sm">
										<span class="font-medium truncate">{goal.name}</span>
									</div>
									<!-- Diverging bar: positive right, negative left -->
									<div class="flex items-center gap-1 h-3">
										<div class="flex-1 flex justify-end">
											{#if goal.negative > 0}
												<div
													class="h-full rounded-l-full bg-error/70"
													style="width: {Math.min(100, goal.negative * 10)}%"
												></div>
											{/if}
										</div>
										<div class="w-px h-4 bg-base-content/10"></div>
										<div class="flex-1">
											{#if goal.positive > 0}
												<div
													class="h-full rounded-r-full bg-success/70"
													style="width: {Math.min(100, goal.positive * 10)}%"
												></div>
											{/if}
										</div>
									</div>
									<div class="flex items-center justify-between text-xs text-base-content/40">
										<span class="text-error">-{goal.negative}</span>
										<span class="text-success">+{goal.positive}</span>
									</div>
								</div>
							{/each}
						</div>
					{/if}
					{#if goals.significantly_advanced && goals.significantly_advanced.length > 0}
						<div class="space-y-1">
							<p class="text-xs font-medium text-success">Significantly Advanced</p>
							{#each goals.significantly_advanced as g}
								<div class="flex items-center gap-2 text-sm">
									<TrendingUp class="w-3.5 h-3.5 text-success shrink-0" />
									<span>{g.name}</span>
								</div>
							{/each}
						</div>
					{/if}
					{#if goals.negatively_impacted && goals.negatively_impacted.length > 0}
						<div class="space-y-1">
							<p class="text-xs font-medium text-error">Negatively Impacted</p>
							{#each goals.negatively_impacted as g}
								<div class="flex items-center gap-2 text-sm">
									<TrendingDown class="w-3.5 h-3.5 text-error shrink-0" />
									<span>{g.name}</span>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</Card>
		{/if}

		<!-- Reflection Section -->
		<Card>
			<div class="flex flex-col gap-4">
				<SectionHeader title="Your Reflection" color="secondary" subtitle="Guided prompts for honest self-review">
					{#snippet icon()}<BookOpen class="w-4 h-4 text-secondary" />{/snippet}
				</SectionHeader>

				<div class="grid grid-cols-1 gap-3">
					{#each reflectionPrompts as prompt (prompt.key)}
						<div class="bg-base-200/40 rounded-box p-3 space-y-1.5">
							<label class="flex items-center gap-2 text-xs font-medium text-base-content/60">
								<prompt.icon class="w-3.5 h-3.5 {prompt.color}" />
								{prompt.label}
							</label>
							<textarea
								class="textarea textarea-bordered bg-base-100 text-sm"
								rows="2"
								value={prompt.key === 'what_went_well' ? whatWentWell
									: prompt.key === 'what_didnt_go_well' ? whatDidntGoWell
									: prompt.key === 'what_learned' ? whatLearned
									: prompt.key === 'proud_of' ? proudOf
									: prompt.key === 'change_tomorrow' ? changeTomorrow
									: additionalNotes}
								oninput={(e) => {
									const v = (e.currentTarget as HTMLTextAreaElement).value;
									if (prompt.key === 'what_went_well') whatWentWell = v;
									else if (prompt.key === 'what_didnt_go_well') whatDidntGoWell = v;
									else if (prompt.key === 'what_learned') whatLearned = v;
									else if (prompt.key === 'proud_of') proudOf = v;
									else if (prompt.key === 'change_tomorrow') changeTomorrow = v;
									else additionalNotes = v;
								}}
							></textarea>
						</div>
					{/each}
				</div>
			</div>
		</Card>

		{#if $updateMutation.isError}
			<ErrorAlert message="Failed to save retrospective" />
		{/if}
	{/if}
</div>

<!-- Sticky Bottom Action Bar -->
{#if retro}
	<div class="fixed bottom-0 left-0 right-0 z-50 bg-base-100/95 backdrop-blur-sm border-t border-base-content/5 px-4 py-3 sm:px-6">
		<div class="max-w-3xl mx-auto flex items-center gap-2">
			<button
				class="btn btn-outline btn-sm gap-2 flex-1 sm:flex-none"
				disabled={$updateMutation.isPending}
				onclick={() => handleSave(false)}
			>
				<Save class="h-4 w-4" />
				Save Draft
			</button>
			<button
				class="btn btn-primary btn-sm gap-2 flex-1 sm:flex-none"
				disabled={$updateMutation.isPending}
				onclick={() => handleSave(true)}
			>
				{#if $updateMutation.isPending}
					<span class="loading loading-spinner loading-xs"></span>
				{:else}
					<CheckCircle2 class="h-4 w-4" />
				{/if}
				Mark Complete
			</button>
			<button
				class="btn btn-ghost btn-sm gap-2 text-error sm:ml-auto"
				onclick={() => (showDeleteConfirm = true)}
			>
				<Trash2 class="h-4 w-4" />
				<span class="hidden sm:inline">Delete</span>
			</button>
		</div>
	</div>
{/if}

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Delete retrospective?"
	message="This will permanently delete this retrospective."
	confirmText="Delete"
	destructive={true}
	loading={$deleteMutation.isPending}
	onConfirm={() => $deleteMutation.mutate()}
	onCancel={() => (showDeleteConfirm = false)}
/>
