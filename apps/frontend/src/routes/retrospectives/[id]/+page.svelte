<script lang="ts">
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { page } from '$app/stores';
import {
	deleteRetrospective,
	getRetrospective,
	updateRetrospective,
	type UserReflection,
} from '$lib/api';
import {
	ConfirmDialog,
	ErrorAlert,
	LoadingCard,
} from '$lib/components/ui';
import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
import { ArrowLeft, Save, Trash2 } from 'lucide-svelte';

const queryClient = useQueryClient();

const retroId = $derived($page.params.id as string);

let showDeleteConfirm = $state(false);

// Editable reflection state — populated when retro loads.
let whatWentWell = $state('');
let whatDidntGoWell = $state('');
let whatLearned = $state('');
let proudOf = $state('');
let changeTomorrow = $state('');
let additionalNotes = $state('');

const retroQuery = createQuery({
	queryKey: ['retrospective', retroId],
	queryFn: () => getRetrospective(retroId),
	enabled: browser && !!retroId,
});

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
	mutationFn: (data: UpdateRetroRequest) => updateRetrospective(retroId, data),
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ['retrospective', retroId] });
		queryClient.invalidateQueries({ queryKey: ['retrospectives'] });
	},
});

const deleteMutation = createMutation({
	mutationFn: () => deleteRetrospective(retroId),
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ['retrospectives'] });
		goto('/retrospectives');
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
</script>

<div class="container mx-auto max-w-3xl px-4 py-6">
	<button class="btn btn-ghost btn-sm gap-2 mb-4" onclick={() => goto('/retrospectives')}>
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
	{:else if $retroQuery.data}
		{@const retro = $retroQuery.data}
		<div class="mb-6">
			<h1 class="text-2xl font-bold capitalize">{retro.retro_type} Retrospective</h1>
			<p class="text-sm opacity-70">
				{formatDate(retro.start_date)} → {formatDate(retro.end_date)}
				<span class="badge badge-sm ml-2" class:badge-success={retro.status === 'completed'}>
					{retro.status}
				</span>
			</p>
		</div>

		{#if retro.auto_summary}
			<div class="card bg-base-100 border border-base-200 mb-6">
				<div class="card-body">
					<h2 class="card-title text-base">Auto Summary</h2>
					{#if retro.auto_summary.insights && retro.auto_summary.insights.length > 0}
						<ul class="list-disc pl-5 space-y-1 text-sm">
							{#each retro.auto_summary.insights as insight}
								<li>{insight}</li>
							{/each}
						</ul>
					{:else}
						<p class="text-sm opacity-70">No insights generated for this period.</p>
					{/if}
				</div>
			</div>
		{/if}

		<div class="card bg-base-100 border border-base-200">
			<div class="card-body space-y-4">
				<h2 class="card-title text-base">Your Reflection</h2>

				<label class="form-control">
					<span class="label-text font-medium">What went well?</span>
					<textarea class="textarea textarea-bordered" rows="3" bind:value={whatWentWell}></textarea>
				</label>

				<label class="form-control">
					<span class="label-text font-medium">What didn't go well?</span>
					<textarea class="textarea textarea-bordered" rows="3" bind:value={whatDidntGoWell}></textarea>
				</label>

				<label class="form-control">
					<span class="label-text font-medium">What did you learn?</span>
					<textarea class="textarea textarea-bordered" rows="3" bind:value={whatLearned}></textarea>
				</label>

				<label class="form-control">
					<span class="label-text font-medium">What are you proud of?</span>
					<textarea class="textarea textarea-bordered" rows="2" bind:value={proudOf}></textarea>
				</label>

				<label class="form-control">
					<span class="label-text font-medium">What will you change tomorrow?</span>
					<textarea class="textarea textarea-bordered" rows="2" bind:value={changeTomorrow}></textarea>
				</label>

				<label class="form-control">
					<span class="label-text font-medium">Additional notes</span>
					<textarea class="textarea textarea-bordered" rows="3" bind:value={additionalNotes}></textarea>
				</label>

				<div class="flex gap-2 pt-4">
					<button
						class="btn btn-outline gap-2"
						disabled={$updateMutation.isPending}
						onclick={() => handleSave(false)}
					>
						<Save class="h-4 w-4" />
						Save Draft
					</button>
					<button
						class="btn btn-primary gap-2"
						disabled={$updateMutation.isPending}
						onclick={() => handleSave(true)}
					>
						<Save class="h-4 w-4" />
						Mark Complete
					</button>
					<button
						class="btn btn-error btn-outline gap-2 ml-auto"
						onclick={() => (showDeleteConfirm = true)}
					>
						<Trash2 class="h-4 w-4" />
						Delete
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Delete retrospective?"
	message="This will permanently delete this retrospective."
	confirmText="Delete"
	destructive={true}
	loading={$deleteMutation.isPending}
	onConfirm={() => deleteMutation.mutate()}
	onCancel={() => (showDeleteConfirm = false)}
/>
