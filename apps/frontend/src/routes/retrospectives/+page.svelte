<script lang="ts">
import { browser } from '$app/environment';
import {
	generateRetrospective,
	getRetrospectives,
	type Retrospective,
} from '$lib/api';
import {
	EmptyState,
	ErrorAlert,
	LoadingCard,
	PageHeader,
} from '$lib/components/ui';
import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
import { Calendar, ChevronRight, Sparkles } from 'lucide-svelte';

const queryClient = useQueryClient();

let page = $state(1);
const limit = 20;

const retrosQuery = createQuery({
	queryKey: ['retrospectives', page],
	queryFn: () => getRetrospectives({ limit, offset: (page - 1) * limit }),
	enabled: browser,
});

const generateMutation = createMutation({
	mutationFn: generateRetrospective,
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ['retrospectives'] });
	},
});

const retros = $derived($retrosQuery.data?.retrospectives ?? []);
const total = $derived($retrosQuery.data?.total ?? 0);
const totalPages = $derived(Math.max(1, Math.ceil(total / limit)));

function formatRange(r: Retrospective): string {
	const s = new Date(r.start_date);
	const e = new Date(r.end_date);
	const fmt = (d: Date) =>
		d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	return `${fmt(s)} → ${fmt(e)}`;
}

function typeLabel(t: Retrospective['retro_type']): string {
	return t.charAt(0).toUpperCase() + t.slice(1);
}

function handleGenerateDaily() {
	const today = new Date();
	const start = new Date(today.getFullYear(), today.getMonth(), today.getDate());
	const end = new Date(start);
	end.setDate(end.getDate() + 1);
	$generateMutation.mutate({
		retro_type: 'daily',
		start_date: start.toISOString(),
		end_date: end.toISOString(),
	});
}

function handleGenerateWeekly() {
	const today = new Date();
	const dayOfWeek = today.getDay();
	const start = new Date(today);
	start.setDate(today.getDate() - dayOfWeek);
	start.setHours(0, 0, 0, 0);
	const end = new Date(start);
	end.setDate(start.getDate() + 7);
	$generateMutation.mutate({
		retro_type: 'weekly',
		start_date: start.toISOString(),
		end_date: end.toISOString(),
	});
}
</script>

<div class="container mx-auto max-w-5xl px-4 py-6">
	<PageHeader title="Retrospectives" subtitle="Reflect on your days and weeks" />

	<div class="mb-6 flex flex-wrap gap-2">
		<button
			class="btn btn-primary gap-2"
			disabled={$generateMutation.isPending}
			onclick={handleGenerateDaily}
		>
			<Sparkles class="h-4 w-4" />
			Generate Today's Retro
		</button>
		<button
			class="btn btn-outline gap-2"
			disabled={$generateMutation.isPending}
			onclick={handleGenerateWeekly}
		>
			<Calendar class="h-4 w-4" />
			Generate This Week
		</button>
	</div>

	{#if $retrosQuery.isLoading}
		<LoadingCard />
	{:else if $retrosQuery.isError}
		<ErrorAlert
			message="Failed to load retrospectives"
			onRetry={() => $retrosQuery.refetch()}
		/>
	{:else if retros.length === 0}
		<EmptyState
			title="No retrospectives yet"
			description="Generate your first daily or weekly retrospective to start reflecting."
		/>
	{:else}
		<div class="space-y-3">
			{#each retros as retro (retro.id)}
				<a
					href={`/retrospectives/${encodeURIComponent(retro.id)}`}
					class="card bg-base-100 shadow-sm border border-base-200 hover:border-primary transition-colors"
				>
					<div class="card-body flex-row items-center justify-between p-4">
						<div class="flex flex-col gap-1">
							<div class="flex items-center gap-2">
								<span class="badge badge-primary badge-sm">{typeLabel(retro.retro_type)}</span>
								{#if retro.status === 'completed'}
									<span class="badge badge-success badge-sm">Completed</span>
								{:else}
									<span class="badge badge-ghost badge-sm">Draft</span>
								{/if}
							</div>
							<div class="text-sm opacity-70">{formatRange(retro)}</div>
						</div>
						<ChevronRight class="h-5 w-5 opacity-40" />
					</div>
				</a>
			{/each}
		</div>

		{#if totalPages > 1}
			<div class="mt-6 flex justify-center gap-2">
				<button
					class="btn btn-sm"
					disabled={page === 1}
					onclick={() => (page -= 1)}
				>
					Previous
				</button>
				<span class="flex items-center px-3 text-sm opacity-70">
					Page {page} of {totalPages}
				</span>
				<button
					class="btn btn-sm"
					disabled={page >= totalPages}
					onclick={() => (page += 1)}
				>
					Next
				</button>
			</div>
		{/if}
	{/if}
</div>
