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
	StickyFooter,
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
import { quadrantColor, CHART_COLORS } from '$lib/utils/chart-colors';

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

function saveEdits() {
	handleSave(retro?.status === 'completed');
}

function togglePublished() {
	handleSave(retro?.status !== 'completed');
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

const quadrantMoodMeta: Record<string, { label: string; emoji: string }> = {
	yellow: { label: 'Energized', emoji: '😊' },
	green: { label: 'Content', emoji: '😌' },
	red: { label: 'Tense', emoji: '😤' },
	blue: { label: 'Low energy', emoji: '😔' },
};

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
		{ key: 'yellow', value: qd.yellow, color: quadrantColor('yellow'), pct: (qd.yellow / total) * 100 },
		{ key: 'green', value: qd.green, color: quadrantColor('green'), pct: (qd.green / total) * 100 },
		{ key: 'red', value: qd.red, color: quadrantColor('red'), pct: (qd.red / total) * 100 },
		{ key: 'blue', value: qd.blue, color: quadrantColor('blue'), pct: (qd.blue / total) * 100 },
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

const hasMoodData = $derived(!!mood && (mood.average_valence !== 0 || mood.average_arousal !== 0 || mood.dominant_quadrant !== ''));
const hasTimeData = $derived(!!categories?.time_distribution && categories.time_distribution.length > 0);
const hasTaskData = $derived(!!tasks && (tasks.completed > 0 || tasks.postponed > 0 || tasks.canceled > 0));
const hasHabitData = $derived(!!habits && ((habits.met?.length ?? 0) > 0 || (habits.partially_met?.length ?? 0) > 0 || (habits.missed?.length ?? 0) > 0));
const hasStreakData = $derived(!!streaks && ((streaks.continued?.length ?? 0) > 0 || (streaks.broken?.length ?? 0) > 0 || (streaks.started?.length ?? 0) > 0));
const hasGoalData = $derived(!!goals?.net_impact && goals.net_impact.length > 0);

const hasAIContent = $derived(!!summary && ((summary.insights?.length ?? 0) > 0 || !!summary.ai_narrative));

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

<div class="flex flex-col gap-6 max-w-6xl mx-auto w-full pb-28">
	{#if $retroQuery.isLoading}
		<LoadingCard message="Loading retrospective..." />
	{:else if $retroQuery.isError}
		<ErrorAlert
			message="Failed to load retrospective"
			onRetry={() => $retroQuery.refetch()}
		/>
	{:else if retro}
		<!-- Detail header: back action, title, and metadata -->
		<div class="flex items-start gap-3">
			<button
				class="btn btn-ghost btn-square btn-sm mt-0.5 shrink-0"
				onclick={() => goto('/retrospectives')}
				aria-label="Back to retrospectives"
			>
				<ArrowLeft class="h-4 w-4" />
			</button>
			<div class="min-w-0 flex-1">
				<h1 class="text-2xl font-semibold tracking-tight capitalize">
					{retro.retro_type} Retrospective
				</h1>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					<span class="badge badge-sm badge-outline capitalize">{retro.retro_type}</span>
					<span
						class="badge badge-sm gap-1"
						class:badge-success={retro.status === 'completed'}
						class:badge-ghost={retro.status !== 'completed'}
					>
						{#if retro.status === 'completed'}
							<CheckCircle2 class="w-3 h-3" /> Published
						{:else}
							<CircleDashed class="w-3 h-3" /> Draft
						{/if}
					</span>
					<span class="badge badge-sm badge-ghost">
						{fmtDateShort(retro.start_date)} – {formatDate(retro.end_date)}
					</span>
					<span class="text-xs text-base-content/50">Generated {formatDate(retro.generated_at)}</span>
				</div>
			</div>
		</div>

		<!-- Prominent period and emotion summary -->
		<Card variant="bordered">
			<div class="flex flex-col gap-5">
				<SectionHeader title="Mood & period summary" subtitle="Your emotional pattern and key outcomes">
					{#snippet icon()}<Heart class="w-4 h-4 text-error" />{/snippet}
				</SectionHeader>

				<div class="grid gap-5 lg:grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)]">
					<div class="rounded-box bg-base-content/5 p-4">
						{#if hasMoodData && mood?.dominant_quadrant}
							{@const dominantMood = quadrantMoodMeta[mood.dominant_quadrant] ?? {
								label: mood.dominant_quadrant,
								emoji: '🙂',
							}}
							<div class="flex items-center gap-3">
								<div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-box bg-base-100 text-3xl shadow-sm">
									<span aria-hidden="true">{dominantMood.emoji}</span>
								</div>
								<div>
									<p class="text-xs font-medium uppercase tracking-wide text-base-content/50">Dominant mood</p>
									<p class="text-lg font-semibold">{dominantMood.label}</p>
									<p class="flex items-center gap-1.5 text-xs text-base-content/60 capitalize">
										<span
											class="h-2.5 w-2.5 rounded-full"
											style="background: {quadrantColor(mood.dominant_quadrant)}"
										></span>
										{mood.dominant_quadrant} quadrant
									</p>
								</div>
							</div>
						{:else}
							<div class="flex items-center gap-3 text-base-content/50">
								<div class="flex h-12 w-12 items-center justify-center rounded-full bg-base-200">
									<Heart class="h-5 w-5" />
								</div>
								<p class="text-sm">No mood data was recorded for this period.</p>
							</div>
						{/if}

						{#if quadrantSegments.length > 0}
							<div class="mt-4 space-y-2">
								<p class="text-xs font-medium text-base-content/60">Quadrant distribution</p>
								<div class="flex h-3 overflow-hidden rounded-full bg-base-200" aria-label="Mood quadrant distribution">
									{#each quadrantSegments as segment (segment.key)}
										<div
											style="width: {segment.pct}%; background: {segment.color};"
											title="{segment.key}: {Math.round(segment.pct)}%"
										></div>
									{/each}
								</div>
								<div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-base-content/60">
									{#each quadrantSegments as segment (segment.key)}
										<span class="flex items-center gap-1.5 capitalize">
											<span class="h-2 w-2 rounded-full" style="background: {segment.color}"></span>
											{segment.key} {Math.round(segment.pct)}%
										</span>
									{/each}
								</div>
							</div>
						{/if}
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-3 lg:grid-cols-1">
						<div class="rounded-box bg-base-content/5 p-3">
							<StatCard value={tasks?.completed ?? 0} label="Tasks completed" color="primary">
								{#snippet icon()}<ListTodo class="w-4 h-4" />{/snippet}
							</StatCard>
						</div>
						<div class="rounded-box bg-base-content/5 p-3">
							<StatCard value={`${(tasks?.total_duration_hours ?? 0).toFixed(1)}h`} label="Hours logged" color="info">
								{#snippet icon()}<Clock class="w-4 h-4" />{/snippet}
							</StatCard>
						</div>
						<div class="rounded-box bg-base-content/5 p-3">
							<StatCard value={activeStreaks} label="Current streaks" color="warning">
								{#snippet icon()}<Flame class="w-4 h-4" />{/snippet}
							</StatCard>
						</div>
					</div>
				</div>
			</div>
		</Card>

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
		<div id="reflection" class="scroll-mt-6">
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
								class="textarea textarea-bordered w-full bg-base-100 text-sm"
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
			</div>

			{#if $updateMutation.isError}
			<ErrorAlert message="Failed to save retrospective" />
		{/if}
	{/if}
</div>

<!-- Sticky Bottom Action Bar -->
{#if retro}
	<StickyFooter>
		{#snippet left()}
			<button
				class="btn btn-ghost btn-sm gap-2 text-error hover:bg-error/10"
				disabled={$deleteMutation.isPending}
				onclick={() => (showDeleteConfirm = true)}
			>
				<Trash2 class="h-4 w-4" />
				<span class="hidden sm:inline">Delete</span>
			</button>
		{/snippet}
		{#snippet right()}
				<button
					class="btn btn-ghost btn-sm"
					disabled={$updateMutation.isPending}
					onclick={() => goto('/retrospectives')}
				>
					Back
				</button>
				<button
					class="btn btn-outline btn-sm gap-2"
					disabled={$updateMutation.isPending}
					onclick={saveEdits}
				>
					<Save class="h-4 w-4" />
					<span class="hidden sm:inline">Save edits</span>
				</button>
				<button
					class={retro.status === 'completed'
						? 'btn btn-outline btn-sm gap-2 min-w-[7.5rem]'
						: 'btn btn-primary btn-sm gap-2 min-w-[7.5rem] shadow-md shadow-primary/20'}
					disabled={$updateMutation.isPending}
					onclick={togglePublished}
				>
					{#if $updateMutation.isPending}
						<span class="loading loading-spinner loading-xs"></span>
					{:else if retro.status === 'completed'}
						<CircleDashed class="h-4 w-4" />
					{:else}
						<CheckCircle2 class="h-4 w-4" />
					{/if}
					{retro.status === 'completed' ? 'Unpublish' : 'Publish'}
				</button>
		{/snippet}
	</StickyFooter>
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
