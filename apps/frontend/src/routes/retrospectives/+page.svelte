<script lang="ts">
import { goto } from '$app/navigation';
import {
	deleteRetrospective,
	generateRetrospective,
	getRetrospectives,
	getUserPreferences,
	type GenerateRetroRequest,
	type Retrospective,
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
	CheckCircle2,
	CircleDashed,
	Funnel,
	LoaderCircle,
	Plus,
	Search,
	Sparkles,
	Trash2,
	X,
} from 'lucide-svelte';

const queryClient = useQueryClient();

let searchQuery = $state('');
let debouncedSearch = $state('');
let statusFilter = $state<'all' | 'draft' | 'completed'>('all');
let periodFilter = $state<'all' | 'daily' | 'weekly' | 'monthly'>('all');
let showFilters = $state(false);
let searchTimeout: ReturnType<typeof setTimeout> | null = null;

const retrosQuery = createQuery({
	queryKey: ['retrospectives'],
	queryFn: () => getRetrospectives({ limit: 200 }),
});

const prefsQuery = createQuery({
	queryKey: ['user-preferences'],
	queryFn: () => getUserPreferences(),
});
const aiHasKey = $derived($prefsQuery.data?.preferences?.ai?.has_key ?? false);

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
		req.end_date = new Date(`${customEnd}T23:59:59`).toISOString();
	}
	$generateMutation.mutate(req);
}

function openCreateModal() {
	generateError = null;
	showCreateModal = true;
}

function handleSearchInput(value: string) {
	searchQuery = value;
	if (searchTimeout) clearTimeout(searchTimeout);
	searchTimeout = setTimeout(() => {
		debouncedSearch = value.trim().toLowerCase();
	}, 300);
}

function clearSearch() {
	if (searchTimeout) clearTimeout(searchTimeout);
	searchQuery = '';
	debouncedSearch = '';
}

function clearFilters() {
	clearSearch();
	statusFilter = 'all';
	periodFilter = 'all';
}

function fmtRange(retro: Retrospective): string {
	const start = new Date(retro.start_date);
	const end = new Date(retro.end_date);
	const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' };
	const startStr = start.toLocaleDateString(undefined, opts);
	const endStr = end.toLocaleDateString(undefined, { ...opts, year: 'numeric' });
	return retro.retro_type === 'daily' ? endStr : `${startStr} – ${endStr}`;
}

function fmtGenerated(date: string): string {
	return new Date(date).toLocaleDateString(undefined, {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
	});
}

function taskCount(retro: Retrospective): number {
	const tasks = retro.auto_summary?.tasks;
	if (!tasks) return 0;
	return tasks.completed + tasks.postponed + tasks.canceled + (tasks.not_started ?? 0);
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

function searchBlob(retro: Retrospective): string {
	const summary = retro.auto_summary;
	return [
		fmtRange(retro),
		typeLabels[retro.retro_type] ?? retro.retro_type,
		retro.status,
		summary?.ai_narrative ?? '',
		...(summary?.insights ?? []),
		summary?.mood?.dominant_quadrant ?? '',
		retro.user_content?.what_went_well ?? '',
		retro.user_content?.what_didnt_go_well ?? '',
		retro.user_content?.what_learned ?? '',
		retro.user_content?.proud_of ?? '',
		retro.user_content?.change_tomorrow ?? '',
		retro.user_content?.additional_notes ?? '',
		...(retro.user_content?.gratitude ?? []),
	]
		.join(' ')
		.toLowerCase();
}

const retros = $derived($retrosQuery.data?.retrospectives ?? []);
const filteredRetros = $derived(
	retros.filter((retro) => {
		const matchesStatus = statusFilter === 'all' || retro.status === statusFilter;
		const matchesPeriod = periodFilter === 'all' || retro.retro_type === periodFilter;
		const matchesSearch = !debouncedSearch || searchBlob(retro).includes(debouncedSearch);
		return matchesStatus && matchesPeriod && matchesSearch;
	}),
);
const hasActiveFilters = $derived(
	debouncedSearch !== '' || statusFilter !== 'all' || periodFilter !== 'all',
);
const isSearching = $derived(searchQuery.trim().toLowerCase() !== debouncedSearch);
</script>

<svelte:head>
	<title>Retrospectives - Lucid Logs</title>
</svelte:head>

<div class="max-w-6xl mx-auto space-y-6 w-full">
	<!-- Header -->
	<div class="flex items-center justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Retrospectives</h1>
			<p class="text-sm text-base-content/60 mt-1">
				{hasActiveFilters
					? `${filteredRetros.length} of ${retros.length} retrospectives`
					: `${retros.length} retrospectives`}
			</p>
		</div>
		<button
			class="btn btn-primary gap-2 shadow-lg shadow-primary/20 transition-all hover:shadow-primary/40"
			onclick={openCreateModal}
		>
			<Plus class="w-4 h-4" />
			New
		</button>
	</div>

	<!-- Search and Filters -->
	<div class="card bg-base-100 shadow-lg border border-base-200">
		<div class="card-body p-3 sm:p-4">
			<div class="flex flex-col gap-3 lg:flex-row lg:items-center">
				<div class="relative flex-1">
					{#if isSearching || $retrosQuery.isFetching}
						<LoaderCircle class="absolute left-3 top-1/2 w-4 h-4 -translate-y-1/2 animate-spin text-base-content/50" />
					{:else}
						<Search class="absolute left-3 top-1/2 w-4 h-4 -translate-y-1/2 text-base-content/50" />
					{/if}
					<input
						type="search"
						placeholder="Search retrospectives..."
						class="input input-bordered w-full pl-10 pr-10"
						value={searchQuery}
						oninput={(event) => handleSearchInput(event.currentTarget.value)}
					/>
					{#if searchQuery}
						<button
							class="absolute right-2 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle"
							onclick={clearSearch}
							aria-label="Clear search"
						>
							<X class="w-3 h-3" />
						</button>
					{/if}
				</div>

				<div class="flex items-center gap-2">
					<button
						class={cn(
							'btn gap-2',
							showFilters ? 'btn-primary' : 'btn-ghost border border-base-300',
						)}
						onclick={() => (showFilters = !showFilters)}
						aria-expanded={showFilters}
					>
						<Funnel class="w-4 h-4" />
						Filters
						{#if hasActiveFilters}
							<span class="badge badge-sm badge-secondary">Active</span>
						{/if}
					</button>
					{#if hasActiveFilters}
						<button class="btn btn-ghost btn-sm gap-1" onclick={clearFilters}>
							<X class="w-4 h-4" />
							Clear
						</button>
					{/if}
				</div>
			</div>

			{#if showFilters}
				<div class="grid gap-3 pt-4 mt-4 border-t border-base-200 sm:grid-cols-2">
					<label class="form-control">
						<span class="label-text text-xs font-medium text-base-content/60 mb-1">Status</span>
						<select class="select select-bordered w-full" bind:value={statusFilter}>
							<option value="all">All</option>
							<option value="draft">Draft</option>
							<option value="completed">Completed</option>
						</select>
					</label>
					<label class="form-control">
						<span class="label-text text-xs font-medium text-base-content/60 mb-1">Period</span>
						<select class="select select-bordered w-full" bind:value={periodFilter}>
							<option value="all">All</option>
							<option value="daily">Daily</option>
							<option value="weekly">Weekly</option>
							<option value="monthly">Monthly</option>
						</select>
					</label>
				</div>
			{/if}
		</div>
	</div>

	<!-- List -->
	{#if $retrosQuery.isPending}
		<LoadingCard message="Loading retrospectives..." />
	{:else if $retrosQuery.isError}
		<ErrorAlert message="Failed to load retrospectives" onRetry={() => $retrosQuery.refetch()} />
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
	{:else if filteredRetros.length === 0}
		<div class="card bg-base-100 shadow-lg border border-base-200">
			<div class="card-body py-12 text-center">
				<Search class="w-12 h-12 text-base-content/30 mx-auto mb-4" />
				<h2 class="text-lg font-semibold">No matching retrospectives</h2>
				<p class="text-sm text-base-content/60 mt-1">Try another search or adjust your filters.</p>
				<div class="mt-4">
					<button class="btn btn-ghost btn-sm" onclick={clearFilters}>Clear filters</button>
				</div>
			</div>
		</div>
	{:else}
		<div class="hidden lg:block card bg-base-100 shadow-lg border border-base-200 overflow-hidden">
			<div class="overflow-x-auto">
				<table class="table table-lg">
					<thead class="bg-base-100 border-b border-base-300">
						<tr>
							<th>Type</th>
							<th>Period</th>
							<th>Status</th>
							<th class="text-right">Tasks</th>
							<th class="text-right">Hours</th>
							<th>Mood</th>
							<th>Generated</th>
							<th class="w-24 text-right">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each filteredRetros as retro (retro.id)}
							{@const tasks = retro.auto_summary?.tasks}
							{@const mood = retro.auto_summary?.mood}
							<tr
								class="hover:bg-base-200/50 transition-colors group cursor-pointer"
								onclick={() => goto(`/retrospectives/${retro.id}`)}
								role="link"
								tabindex="0"
								onkeydown={(event) => event.key === 'Enter' && goto(`/retrospectives/${retro.id}`)}
							>
								<td>
									<span class="badge badge-sm badge-outline whitespace-nowrap">
										{typeLabels[retro.retro_type] ?? retro.retro_type}
									</span>
								</td>
								<td class="whitespace-nowrap font-semibold">{fmtRange(retro)}</td>
								<td>
									<span
										class={cn(
											'badge badge-sm gap-1 whitespace-nowrap',
											retro.status === 'completed' ? 'badge-success' : 'badge-ghost',
										)}
									>
										{#if retro.status === 'completed'}
											<CheckCircle2 class="h-3 w-3" /> Completed
										{:else}
											<CircleDashed class="h-3 w-3" /> Draft
										{/if}
									</span>
								</td>
								<td class="text-right font-mono tabular-nums">{taskCount(retro)}</td>
								<td class="text-right font-mono tabular-nums">
									{(tasks?.total_duration_hours ?? 0).toFixed(1)}h
								</td>
								<td>
									{#if mood?.dominant_quadrant}
										<span class="flex items-center gap-2 whitespace-nowrap">
											<span
												class="h-2 w-2 rounded-full"
												style="background: {quadrantColor(mood.dominant_quadrant)}"
											></span>
											<span class="capitalize">{mood.dominant_quadrant}</span>
										</span>
									{:else}
										<span class="text-base-content/40">—</span>
									{/if}
								</td>
								<td class="whitespace-nowrap text-sm text-base-content/60">{fmtGenerated(retro.generated_at)}</td>
								<td>
									<div class="flex gap-1 justify-end opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
										<button
											class="btn btn-ghost btn-sm btn-square text-error"
											aria-label="Delete retrospective"
											onclick={(event) => {
												event.stopPropagation();
												deleteTarget = retro;
											}}
										>
											<Trash2 class="w-4 h-4" />
										</button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<div class="px-4 py-3 bg-base-200/50 border-t border-base-200 text-sm opacity-60 flex items-center justify-between">
				<span>{filteredRetros.length} retrospectives</span>
				{#if $retrosQuery.isFetching}
					<span class="flex items-center gap-2 text-primary"><LoaderCircle class="w-4 h-4 animate-spin" /> Loading...</span>
				{/if}
			</div>
		</div>

		<!-- Mobile card view -->
		<div class="lg:hidden space-y-2">
			{#each filteredRetros as retro (retro.id)}
				{@const tasks = retro.auto_summary?.tasks}
				{@const mood = retro.auto_summary?.mood}
				<article
					class="card bg-base-100 border border-base-200 shadow-sm active:scale-[0.99] transition-all cursor-pointer"
					onclick={() => goto(`/retrospectives/${retro.id}`)}
					role="button"
					tabindex="0"
					onkeydown={(event) => event.key === 'Enter' && goto(`/retrospectives/${retro.id}`)}
				>
					<div class="card-body p-3">
						<div class="flex items-center justify-between gap-2">
							<div class="flex items-center gap-2 min-w-0">
								<span class="badge badge-sm badge-outline shrink-0">{typeLabels[retro.retro_type] ?? retro.retro_type}</span>
								<span class="font-semibold text-[15px] truncate">{fmtRange(retro)}</span>
							</div>
							<span
								class={cn(
									'badge badge-sm gap-1 shrink-0',
									retro.status === 'completed' ? 'badge-success' : 'badge-ghost',
								)}
							>
								{#if retro.status === 'completed'}
									<CheckCircle2 class="h-3 w-3" /> Done
								{:else}
									<CircleDashed class="h-3 w-3" /> Draft
								{/if}
							</span>
						</div>
						<div class="flex items-center gap-4 mt-1 text-xs text-base-content/60">
							<span>{taskCount(retro)} tasks</span>
							<span>{(tasks?.total_duration_hours ?? 0).toFixed(1)}h</span>
							{#if mood?.dominant_quadrant}
								<span class="flex items-center gap-1.5">
									<span class="h-1.5 w-1.5 rounded-full" style="background: {quadrantColor(mood.dominant_quadrant)}"></span>
									<span class="capitalize">{mood.dominant_quadrant}</span>
								</span>
							{/if}
							<span class="ml-auto">{fmtGenerated(retro.generated_at)}</span>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</div>

<Modal bind:open={showCreateModal} title="New Retrospective" subtitle="Choose a period to analyze" size="md">
	{#snippet icon()}<Sparkles class="w-5 h-5 text-primary" />{/snippet}

	<div class="flex flex-col gap-4">
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
					<span class="text-xs text-base-content/60">{opt.desc}</span>
				</button>
			{/each}
		</div>

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
	destructive={true}
	loading={$deleteMutation.isPending}
	onConfirm={() => {
		if (deleteTarget?.id) $deleteMutation.mutate(deleteTarget.id);
		deleteTarget = null;
	}}
	onCancel={() => (deleteTarget = null)}
/>
