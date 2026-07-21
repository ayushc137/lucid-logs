<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Check, Loader2, Target, X } from 'lucide-svelte';
	import { instantLog, type Activity, type InstantLogResponse } from '$lib/api/activities';
	import { getUnits } from '$lib/api/units';
	import { cn } from '$lib/utils';

	// Props
	interface Props {
		activity: Activity;
		isOpen: boolean;
		onClose: () => void;
		onSuccess?: (response: InstantLogResponse) => void;
	}

	let { activity, isOpen = $bindable(), onClose, onSuccess }: Props = $props();

	const queryClient = useQueryClient();

	// Local state
	let notes = $state('');
	let error = $state('');

	// Units query for symbol lookup
	const unitsQuery = createQuery({
		queryKey: ['units'],
		queryFn: () => getUnits(),
	});

	const units = $derived($unitsQuery.data?.items || []);

	// Get unit symbol
	function getUnitSymbol(unitId: string | undefined): string {
		if (!unitId) return '';
		const unit = units.find((u) => u.id === unitId);
		return unit?.symbol || unitId.replace('units:', '');
	}

	// Reset state when activity changes or modal opens
	$effect(() => {
		if (isOpen && activity) {
			notes = '';
			error = '';
		}
	});

	// Mutation for instant logging
	const instantLogMutation = createMutation({
		mutationFn: async () => {
			// Don't send quantity - backend uses per-goal defaults from activity_goals
			return instantLog(activity.id, {
				notes: notes.trim() || undefined,
			});
		},
		onSuccess: (response) => {
			queryClient.invalidateQueries({ queryKey: ['activities'] });
			queryClient.invalidateQueries({ queryKey: ['tasks'] });
			queryClient.invalidateQueries({ queryKey: ['goals'] });
			onSuccess?.(response);
			onClose();
		},
		onError: (err: Error) => {
			error = err.message || 'Failed to log activity';
		},
	});

	function handleSubmit() {
		$instantLogMutation.mutate();
	}

	// Keyboard handler
	function handleKeydown(event: KeyboardEvent) {
		if (!isOpen) return;
		if (event.key === 'Escape') {
			onClose();
		} else if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			handleSubmit();
		}
	}

	const isPending = $derived($instantLogMutation.isPending);
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Modal: bottom sheet on mobile, centered dialog on desktop -->
<dialog class={cn('modal modal-bottom sm:modal-middle', isOpen && 'modal-open')}>
	<div class="modal-box max-w-sm w-full sm:w-auto">
		<!-- Drag handle (mobile) -->
		<div class="sm:hidden flex justify-center pt-1 pb-3">
			<div class="w-10 h-1 rounded-full bg-base-content/15"></div>
		</div>
		<!-- Header -->
		<div class="flex items-center gap-3 mb-6">
			<div class="w-10 h-10 rounded-box bg-primary/10 flex items-center justify-center text-xl shrink-0">
				{activity.icon || '⚡'}
			</div>
			<div class="flex-1">
				<h3 class="font-bold text-lg">{activity.title}</h3>
				{#if activity.description}
					<p class="text-sm opacity-60 line-clamp-1">{activity.description}</p>
				{/if}
			</div>
			<button class="btn btn-ghost btn-sm btn-circle" onclick={onClose}>
				<X class="w-4 h-4" />
			</button>
		</div>

		<!-- Error Alert -->
		{#if error}
			<div class="alert alert-error mb-4">
				<span>{error}</span>
				<button class="btn btn-ghost btn-xs btn-circle" onclick={() => (error = '')}>
					<X class="w-3 h-3" />
				</button>
			</div>
		{/if}

		<!-- Per-Goal Quantities Preview -->
		{#if activity.goals && activity.goals.length > 0 && activity.quantity_enabled}
			<div class="mb-6 space-y-3">
				<p class="text-sm font-medium flex items-center gap-2">
					<Target class="w-4 h-4 text-primary" />
					Will update goals:
				</p>
				{#each activity.goals as goalLink}
					{@const goalQuantity = goalLink.default_quantity ??
						(activity.quantity_default ?? 1) * (goalLink.quantity_multiplier ?? 1)}
					{@const goalUnit = goalLink.goal?.target?.unit_id}
					<div class="bg-base-200 rounded-xl p-3">
						<div class="flex items-center gap-3">
							<span class="text-xl">{goalLink.goal?.icon || '🎯'}</span>
							<div class="flex-1">
								<p class="font-medium text-sm">{goalLink.goal?.title ?? 'Goal'}</p>
								<span class={cn(
									'badge badge-xs',
									goalLink.default_impact === 'positive' && 'badge-success',
									goalLink.default_impact === 'negative' && 'badge-error',
									goalLink.default_impact === 'neutral' && 'badge-ghost'
								)}>
									{goalLink.default_impact}
								</span>
							</div>
							<!-- Per-goal quantity display (read-only) -->
							<div class="text-right">
								<span class="text-lg font-bold text-primary">+{goalQuantity}</span>
								{#if goalUnit}
									<span class="text-sm opacity-60 ml-1">{getUnitSymbol(goalUnit)}</span>
								{/if}
							</div>
						</div>
					</div>
				{/each}
				<p class="text-xs opacity-50 text-center">
					Edit activity settings to change default quantities
				</p>
			</div>
		{:else if activity.goals && activity.goals.length > 0}
			<!-- Goals without quantity tracking -->
			<div class="mb-6">
				<p class="text-sm font-medium flex items-center gap-2 mb-3">
					<Target class="w-4 h-4 text-primary" />
					Will update goals:
				</p>
				<div class="flex flex-wrap gap-2">
					{#each activity.goals as goalLink}
						<div
							class={cn(
								'badge gap-1',
								goalLink.default_impact === 'positive' && 'badge-success',
								goalLink.default_impact === 'negative' && 'badge-error',
								goalLink.default_impact === 'neutral' && 'badge-ghost'
							)}
						>
							{#if goalLink.goal?.icon}
								<span>{goalLink.goal.icon}</span>
							{/if}
							<span>{goalLink.goal?.title ?? 'Goal'}</span>
						</div>
					{/each}
				</div>
			</div>
		{:else}
			<div class="text-center py-4 mb-4">
				<p class="opacity-70">Log this activity now?</p>
				<p class="text-xs opacity-50 mt-1">No goals linked - entry will be logged as a task</p>
			</div>
		{/if}

		<!-- Notes (optional) -->
		<div class="mb-4">
			<label class="label" for="notes-input">
				<span class="label-text font-medium">Notes</span>
				<span class="label-text-alt opacity-50">Optional</span>
			</label>
			<input
				id="notes-input"
				type="text"
				class="input input-bordered w-full"
				placeholder="Add a quick note..."
				bind:value={notes}
			/>
		</div>

		<!-- Actions -->
		<div class="modal-action mt-0">
			<button type="button" class="btn btn-ghost" onclick={onClose}>Cancel</button>
			<button
				type="button"
				class="btn btn-primary flex-1 gap-2"
				onclick={handleSubmit}
				disabled={isPending}
			>
				{#if isPending}
					<Loader2 class="w-4 h-4 animate-spin" />
					Logging...
				{:else}
					<Check class="w-4 h-4" />
					Log Now
				{/if}
			</button>
		</div>
	</div>

	<!-- Click outside to close -->
	<form method="dialog" class="modal-backdrop bg-base-content/20 backdrop-blur-sm">
		<button onclick={onClose}>close</button>
	</form>
</dialog>
