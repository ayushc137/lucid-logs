<script lang="ts">
import { goto } from '$app/navigation';
import {
	deleteRetrospective,
	generateRetrospective,
	getRetrospectives,
	type Retrospective,
} from '$lib/api/retrospectives';
import { ConfirmDialog, EmptyState, ErrorAlert, LoadingCard } from '$lib/components/ui';
import { cn } from '$lib/utils';
import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
import {
	BookOpen,
	CalendarDays,
	CheckCircle2,
	ChevronRight,
	CircleDashed,
	Sparkles,
	Trash2,
} from 'lucide-svelte';

const queryClient = useQueryClient();

const retrosQuery = createQuery({
	queryKey: ['retrospectives'],
	queryFn: () => getRetrospectives({ limit: 50 }),
});

let generating = $state<string | null>(null);
let generateError = $state<string | null>(null);
let deleteTarget = $state<Retrospective | null>(null);

const generateMutation = createMutation({
	mutationFn: (retro_type: string) =>
		generateRetrospective({ retro_type: retro_type as Retrospective['retro_type'] }),
	onSuccess: (retro) => {
		queryClient.invalidateQueries({ queryKey: ['retrospectives'] });
		generating = null;
		if (retro.id) goto(`/retrospectives/${retro.id}`);
	},
	onError: (err) => {
		generateError = err instanceof Error ? err.message : 'Failed to generate retrospective';
		generating = null;
	},
});

const deleteMutation = createMutation({
	mutationFn: (id: string) => deleteRetrospective(id),
	onSuccess: () => queryClient.invalidateQueries({ queryKey: ['retrospectives'] }),
});

function generate(retroType: string) {
	generateError = null;
	generating = retroType;
	$generateMutation.mutate(retroType);
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

const generateOptions = [
	{ type: 'daily', label: 'Today', desc: "Today's tasks, mood, and habits" },
	{ type: 'weekly', label: 'This Week', desc: 'The last 7 days in review' },
	{ type: 'monthly', label: 'This Month', desc: 'A month-level look back' },
];

const retros = $derived($retrosQuery.data?.retrospectives ?? []);
</script>

<svelte:head>
	<title>Retrospectives - Lucid Logs</title>
</svelte:head>

<div class="flex flex-col gap-6 px-4 py-4 sm:px-6 sm:py-6 max-w-3xl mx-auto w-full">
	<!-- Header -->
	<div class="flex items-center gap-3.5">
		<div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
			<BookOpen class="w-5.5 h-5.5" />
		</div>
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Retrospectives</h1>
			<p class="text-sm text-base-content/55">Auto-generated reflections on your days and weeks</p>
		</div>
	</div>

	<!-- Generate -->
	<section class="card bg-base-100 border border-base-200 shadow-sm">
		<div class="card-body p-4 sm:p-5 gap-3">
			<h2 class="font-semibold text-[15px] flex items-center gap-2">
				<Sparkles class="w-4 h-4 text-primary" />
				Generate a retrospective
			</h2>
			<div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
				{#each generateOptions as opt (opt.type)}
					<button
						class="flex flex-col items-start gap-0.5 rounded-xl border border-base-300 bg-base-200/40 px-4 py-3 text-left transition-all hover:border-primary/50 hover:bg-primary/5 active:scale-[0.98] disabled:opacity-50"
						disabled={generating !== null}
						onclick={() => generate(opt.type)}
					>
						<span class="font-semibold text-sm flex items-center gap-2">
							{opt.label}
							{#if generating === opt.type}
								<span class="loading loading-spinner loading-xs text-primary"></span>
							{/if}
						</span>
						<span class="text-xs text-base-content/55">{opt.desc}</span>
					</button>
				{/each}
			</div>
			{#if generateError}
				<ErrorAlert message={generateError} />
			{/if}
		</div>
	</section>

	<!-- List -->
	{#if $retrosQuery.isPending}
		<LoadingCard />
	{:else if $retrosQuery.isError}
		<ErrorAlert message="Failed to load retrospectives" />
	{:else if retros.length === 0}
		<EmptyState
			title="No retrospectives yet"
			description="Generate your first retrospective above to get an auto-built summary of your tasks, moods, habits, and goals."
			showButton={false}
		>
			{#snippet icon()}
				<BookOpen class="w-7 h-7" />
			{/snippet}
		</EmptyState>
	{:else}
		<div class="flex flex-col gap-2.5">
			{#each retros as retro (retro.id)}
				<article
					class="card bg-base-100 border border-base-200 shadow-sm active:scale-[0.99] transition-all cursor-pointer"
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
								<p class="text-xs text-base-content/55">
									{#if retro.auto_summary?.tasks}
										{retro.auto_summary.tasks.completed ?? 0} tasks completed
										· {(retro.auto_summary.tasks.total_duration_hours ?? 0).toFixed(1)}h tracked
									{/if}
								</p>
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
