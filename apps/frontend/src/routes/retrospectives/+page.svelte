<script lang="ts">
import { goto } from '$app/navigation';
import {
	deleteRetrospective,
	generateRetrospective,
	getRetrospectives,
	getUserPreferences,
	type Retrospective,
	type GenerateRetroRequest,
} from '$lib/api';
import {
	ConfirmDialog,
	EmptyState,
	ErrorAlert,
	LoadingCard,
	Modal,
} from '$lib/components/ui';
import { cn } from '$lib/utils';
import { quadrantColor } from '$lib/utils/chart-colors';
import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
import {
	BookOpen,
	CalendarDays,
	CheckCircle2,
	ChevronRight,
	CircleDashed,
	Clock,
	Heart,
	ListTodo,
	Plus,
	Sparkles,
	Trash2,
} from 'lucide-svelte';

const queryClient = useQueryClient();

const retrosQuery = createQuery({
	queryKey: ['retrospectives'],
	queryFn: () => getRetrospectives({ limit: 50 }),
});

// Fetch preferences to check if AI is configured
const prefsQuery = createQuery({
	queryKey: ['user-preferences'],
	queryFn: () => getUserPreferences(),
});
const aiHasKey = $derived($prefsQuery.data?.preferences?.ai?.has_key ?? false);

// Modal state
let showCreateModal = $state(false);
let selectedType = $state<'daily' | 'weekly' | 'monthly'>('daily');
let customStart = $state('');
let customEnd = $state('');
let useAIRange = $state(true);
let generateError = $state<string | null>(null);

let deleteTarget = $state<Retrospective | null>(null);

const generateMutation = createMutation({
	mutationFn: (req: GenerateRetroRequest) => generateRetrospective(req),
	onSuccess: (retro) => {
		queryClient.invalidateQueries({ queryKey: ['retrospectives'] });
		showCreateModal = false;
		generateError = null;
		if (retro.id) goto(`/retrospectives/${retro.id}`);
	},
	onError: (err) => {
		generateError = err instanceof Error ? err.message : 'Failed to generate retrospective';
	},
});

const deleteMutation = createMutation({
	mutationFn: (id: string) => deleteRetrospective(id),
	onSuccess: () => queryClient.invalidateQueries({ queryKey: ['retrospectives'] }),
});

function handleGenerate() {
	generateError = null;
	const req: GenerateRetroRequest = { retro_type: selectedType };
	if (selectedType !== 'daily' && customStart && customEnd) {
		req.start_date = new Date(customStart).toISOString();
		req.end_date = new Date(customEnd + 'T23:59:59').toISOString();
	}
	$generateMutation.mutate(req);
}

function openCreateModal() {
	generateError = null;
	showCreateModal = true;
}

function fmtRange(retro: Retrospective): string {
	const start = new Date(retro.start_date);
	const end = new Date(retro.end_date);
	const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' };
	const startStr = start.toLocaleDateString(undefined, opts);
	const endStr = end.toLocaleDateString(undefined, { ...opts, year: 'numeric' });
	return retro.retro_type === 'daily' ? endStr : `${startStr} – ${endStr}`;
}

const typeLabels: Record<string, string> = {
	daily: 'Daily',
	weekly: 'Weekly',
	monthly: 'Monthly',
	quarterly: 'Quarterly',
	yearly: 'Yearly',
	custom: 'Custom',
};

const typeOptions = [
	{ type: 'daily' as const, label: 'Today', desc: "Today's tasks, mood, and habits" },
	{ type: 'weekly' as const, label: 'This Week', desc: 'The last 7 days in review' },
	{ type: 'monthly' as const, label: 'This Month', desc: 'A month-level look back' },
];

const retros = $derived($retrosQuery.data?.retrospectives ?? []);
</script>

<svelte:head>
	<title>Retrospectives - Lucid Logs</title>
</svelte:head>

<div class="flex flex-col gap-6 px-4 py-4 sm:px-6 sm:py-6 max-w-3xl mx-auto w-full">
	<!-- Header -->
	<div class="flex items-center justify-between gap-3.5">
		<div class="flex items-center gap-3.5 min-w-0">
			<div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
				<BookOpen class="w-5.5 h-5.5" />
			</div>
			<div class="min-w-0">
				<h1 class="text-2xl font-semibold tracking-tight">Retrospectives</h1>
				<p class="text-sm text-base-content/55">Auto-generated reflections on your days and weeks</p>
			</div>
		</div>
		<button class="btn btn-primary btn-sm gap-2 shrink-0" onclick={openCreateModal}>
			<Plus class="w-4 h-4" />
			<span class="hidden sm:inline">New</span>
		</button>
	</div>

	<!-- List -->
	{#if $retrosQuery.isPending}
		<LoadingCard />
	{:else if $retrosQuery.isError}
		<ErrorAlert message="Failed to load retrospectives" />
	{:else if retros.length === 0}
		<EmptyState
			title="No retrospectives yet"
			description="Generate your first retrospective to get an auto-built summary of your tasks, moods, habits, and goals."
			showButton={true}
			buttonLabel="New Retrospective"
			onButtonClick={openCreateModal}
		>
			{#snippet icon()}
				<BookOpen class="w-7 h-7" />
			{/snippet}
		</EmptyState>
	{:else}
		<div class="flex flex-col gap-2.5">
			{#each retros as retro (retro.id)}
				{@const tasks = retro.auto_summary?.tasks}
				{@const mood = retro.auto_summary?.mood}
				<article
					class="card bg-base-100 border border-base-content/5 shadow-sm active:scale-[0.99] transition-all cursor-pointer hover:border-primary/30"
					onclick={() => goto(`/retrospectives/${retro.id}`)}
					role="button"
					tabindex="0"
					onkeydown={(e) => e.key === 'Enter' && goto(`/retrospectives/${retro.id}`)}
				>
					<div class="card-body p-4 gap-2">
						<div class="flex items-center gap-3">
							<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-secondary/10 text-secondary">
								<CalendarDays class="w-5 h-5" />
							</div>
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 flex-wrap">
									<span class="badge badge-sm badge-outline">{typeLabels[retro.retro_type] ?? retro.retro_type}</span>
									<span
										class={cn(
											'badge badge-sm gap-1',
											retro.status === 'completed' ? 'badge-success' : 'badge-ghost',
										)}
									>
										{#if retro.status === 'completed'}
											<CheckCircle2 class="w-3 h-3" /> Completed
										{:else}
											<CircleDashed class="w-3 h-3" /> Draft
										{/if}
									</span>
								</div>
								<p class="font-semibold text-[15px] mt-1">{fmtRange(retro)}</p>
								<!-- Inline mini-stats -->
								<div class="flex items-center gap-3 text-xs text-base-content/55 mt-0.5">
									{#if tasks}
										<span class="flex items-center gap-1">
											<ListTodo class="w-3 h-3" />
											{tasks.completed ?? 0}
										</span>
										<span class="flex items-center gap-1">
											<Clock class="w-3 h-3" />
											{(tasks.total_duration_hours ?? 0).toFixed(1)}h
										</span>
									{/if}
									{#if mood && mood.dominant_quadrant}
										<span class="flex items-center gap-1">
											<span class="w-2 h-2 rounded-full" style="background: {quadrantColor(mood.dominant_quadrant)}"></span>
											<span class="capitalize">{mood.dominant_quadrant}</span>
										</span>
									{/if}
								</div>
							</div>
							<button
								class="btn btn-ghost btn-xs btn-square text-error/70 shrink-0"
								aria-label="Delete retrospective"
								onclick={(e) => {
									e.stopPropagation();
									deleteTarget = retro;
								}}
							>
								<Trash2 class="w-4 h-4" />
							</button>
							<ChevronRight class="w-4 h-4 text-base-content/30 shrink-0" />
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</div>

<!-- Create Modal -->
<Modal bind:open={showCreateModal} title="New Retrospective" subtitle="Choose a period to analyze" size="md">
	{#snippet icon()}<Sparkles class="w-5 h-5 text-primary" />{/snippet}

	<div class="flex flex-col gap-4">
		<!-- Period type picker -->
		<div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
			{#each typeOptions as opt (opt.type)}
				<button
					class={cn(
						'flex flex-col items-start gap-1 rounded-box border-2 px-4 py-3 text-left transition-all active:scale-[0.98]',
						selectedType === opt.type
							? 'border-primary bg-primary/5'
							: 'border-base-content/10 hover:border-base-content/25',
					)}
					onclick={() => (selectedType = opt.type)}
				>
					<span class="font-semibold text-sm">{opt.label}</span>
					<span class="text-xs text-base-content/55">{opt.desc}</span>
				</button>
			{/each}
		</div>

		<!-- Custom date range (weekly/monthly only) -->
		{#if selectedType !== 'daily'}
			<div class="grid grid-cols-2 gap-3">
				<label class="form-control">
					<span class="label-text font-medium text-xs text-base-content/60 mb-1">Start Date</span>
					<input type="date" class="input input-bordered input-sm" bind:value={customStart} />
				</label>
				<label class="form-control">
					<span class="label-text font-medium text-xs text-base-content/60 mb-1">End Date</span>
					<input type="date" class="input input-bordered input-sm" bind:value={customEnd} />
				</label>
			</div>
			<p class="text-xs text-base-content/40">Leave blank to auto-detect this period's range.</p>
		{/if}

		<!-- AI insights toggle -->
		{#if aiHasKey}
			<label class="flex items-center gap-3 cursor-pointer">
				<input type="checkbox" class="toggle toggle-primary toggle-sm" bind:checked={useAIRange} />
				<span class="text-sm flex items-center gap-1.5">
					<Sparkles class="w-3.5 h-3.5 text-primary" />
					Include AI insights
				</span>
			</label>
		{/if}

		{#if generateError}
			<ErrorAlert message={generateError} />
		{/if}
	</div>

	{#snippet actions()}
		<button class="btn btn-ghost btn-sm" onclick={() => (showCreateModal = false)}>Cancel</button>
		<button
			class="btn btn-primary btn-sm gap-2"
			disabled={$generateMutation.isPending}
			onclick={handleGenerate}
		>
			{#if $generateMutation.isPending}
				<span class="loading loading-spinner loading-xs"></span>
			{:else}
				<Sparkles class="w-4 h-4" />
			{/if}
			Generate
		</button>
	{/snippet}
</Modal>

<ConfirmDialog
	open={deleteTarget !== null}
	title="Delete retrospective?"
	message="This will permanently remove this retrospective and its reflection."
	confirmText="Delete"
	onConfirm={() => {
		if (deleteTarget?.id) $deleteMutation.mutate(deleteTarget.id);
		deleteTarget = null;
	}}
	onCancel={() => (deleteTarget = null)}
/>
