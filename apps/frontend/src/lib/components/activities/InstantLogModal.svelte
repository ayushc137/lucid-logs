<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { X, Minus, Plus, Check, Loader2 } from 'lucide-svelte';
	import { instantLog, type Activity, type InstantLogResponse } from '$lib/api';

	// Props
	export let activity: Activity;
	export let isOpen = false;
	export let onClose: () => void = () => {};
	export let onSuccess: (response: InstantLogResponse) => void = () => {};

	const queryClient = useQueryClient();

	// Local state
	let quantity = activity.quantity_default ?? 1;
	let error = '';

	// Reset quantity when activity changes
	$: if (activity) {
		quantity = activity.quantity_default ?? 1;
		error = '';
	}

	// Mutation for instant logging
	const instantLogMutation = createMutation({
		mutationFn: async () => {
			return instantLog(activity.id, {
				quantity: activity.quantity_enabled ? quantity : undefined
			});
		},
		onSuccess: (response) => {
			queryClient.invalidateQueries({ queryKey: ['activities'] });
			queryClient.invalidateQueries({ queryKey: ['tasks'] });
			queryClient.invalidateQueries({ queryKey: ['goals'] });
			onSuccess(response);
			onClose();
		},
		onError: (err: Error) => {
			error = err.message || 'Failed to log activity';
		}
	});

	function handleSubmit() {
		$instantLogMutation.mutate();
	}

	function incrementQuantity() {
		const step = activity.quantity_step ?? 1;
		quantity = Math.round((quantity + step) * 100) / 100;
	}

	function decrementQuantity() {
		const step = activity.quantity_step ?? 1;
		const newValue = quantity - step;
		if (newValue >= 0) {
			quantity = Math.round(newValue * 100) / 100;
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			onClose();
		} else if (event.key === 'Enter') {
			handleSubmit();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if isOpen}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm"
		on:click={onClose}
		on:keydown={(e) => e.key === 'Escape' && onClose()}
		role="button"
		tabindex="-1"
	></div>

	<!-- Modal -->
	<div
		class="fixed left-1/2 top-1/2 z-50 w-full max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-2xl bg-white p-6 shadow-2xl dark:bg-gray-800"
		role="dialog"
		aria-modal="true"
		aria-labelledby="modal-title"
	>
		<!-- Header -->
		<div class="mb-6 flex items-center justify-between">
			<div class="flex items-center gap-3">
				{#if activity.icon}
					<span class="text-3xl">{activity.icon}</span>
				{/if}
				<h2 id="modal-title" class="text-xl font-semibold text-gray-900 dark:text-white">
					{activity.title}
				</h2>
			</div>
			<button
				type="button"
				class="rounded-full p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
				on:click={onClose}
			>
				<X class="h-5 w-5" />
			</button>
		</div>

		<!-- Error Message -->
		{#if error}
			<div class="mb-4 rounded-lg bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400">
				{error}
			</div>
		{/if}

		<!-- Quantity Picker (only if quantity enabled) -->
		{#if activity.quantity_enabled}
			<div class="mb-6">
				<span class="mb-2 block text-sm font-medium text-gray-600 dark:text-gray-400">
					Quantity
				</span>
				<div class="flex items-center justify-center gap-4">
					<button
						type="button"
						class="flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 text-gray-700 transition-colors hover:bg-gray-200 active:bg-gray-300 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
						on:click={decrementQuantity}
						disabled={quantity <= 0}
					>
						<Minus class="h-5 w-5" />
					</button>

					<div class="min-w-[100px] text-center">
						<input
							type="number"
							bind:value={quantity}
							step={activity.quantity_step ?? 1}
							min="0"
							class="w-full bg-transparent text-center text-4xl font-bold text-gray-900 focus:outline-none dark:text-white"
						/>
						{#if activity.quantity_unit_id}
							<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
								{activity.quantity_unit_id.replace('units:', '')}
							</p>
						{/if}
					</div>

					<button
						type="button"
						class="flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 text-gray-700 transition-colors hover:bg-gray-200 active:bg-gray-300 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
						on:click={incrementQuantity}
					>
						<Plus class="h-5 w-5" />
					</button>
				</div>
			</div>
		{:else}
			<p class="mb-6 text-center text-gray-600 dark:text-gray-400">
				Log this activity now?
			</p>
		{/if}

		<!-- Goal Links Preview -->
		{#if activity.goals && activity.goals.length > 0}
			<div class="mb-6 rounded-lg bg-gray-50 p-3 dark:bg-gray-700/50">
				<p class="mb-2 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
					Will update goals:
				</p>
				<div class="flex flex-wrap gap-2">
					{#each activity.goals as goalLink}
						<span
							class="inline-flex items-center gap-1 rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400"
						>
							{#if goalLink.goal?.icon}
								<span>{goalLink.goal.icon}</span>
							{/if}
							{goalLink.goal?.title ?? goalLink.goal_id}
						</span>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Submit Button -->
		<button
			type="button"
			class="flex w-full items-center justify-center gap-2 rounded-xl bg-emerald-600 py-3.5 font-semibold text-white transition-colors hover:bg-emerald-700 active:bg-emerald-800 disabled:opacity-50"
			on:click={handleSubmit}
			disabled={$instantLogMutation.isPending}
		>
			{#if $instantLogMutation.isPending}
				<Loader2 class="h-5 w-5 animate-spin" />
				Logging...
			{:else}
				<Check class="h-5 w-5" />
				Log Now
			{/if}
		</button>
	</div>
{/if}
