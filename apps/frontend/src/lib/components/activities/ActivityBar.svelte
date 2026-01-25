<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { Loader2, Plus } from 'lucide-svelte';
	import { getPinnedActivities, type Activity, type InstantLogResponse } from '$lib/api';
	import InstantLogModal from './InstantLogModal.svelte';

	// Props
	export let onActivityLogged: (response: InstantLogResponse) => void = () => {};

	// State
	let selectedActivity: Activity | null = null;
	let isModalOpen = false;

	// Query for pinned activities
	const pinnedActivitiesQuery = createQuery({
		queryKey: ['activities', 'pinned'],
		queryFn: getPinnedActivities
	});

	function openModal(activity: Activity) {
		selectedActivity = activity;
		isModalOpen = true;
	}

	function closeModal() {
		isModalOpen = false;
		selectedActivity = null;
	}

	function handleSuccess(response: InstantLogResponse) {
		onActivityLogged(response);
		$pinnedActivitiesQuery.refetch();
	}
</script>

<div class="activity-bar">
	<!-- Header -->
	<div class="mb-3 flex items-center justify-between">
		<h3 class="text-sm font-semibold text-gray-600 dark:text-gray-400">Quick Log</h3>
		<a
			href="/activities"
			class="text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
		>
			Manage
		</a>
	</div>

	<!-- Activity Pills -->
	<div class="flex flex-wrap gap-2">
		{#if $pinnedActivitiesQuery.isLoading}
			<div class="flex items-center gap-2 text-gray-400">
				<Loader2 class="h-4 w-4 animate-spin" />
				<span class="text-sm">Loading...</span>
			</div>
		{:else if $pinnedActivitiesQuery.isError}
			<p class="text-sm text-red-500">Failed to load activities</p>
		{:else if $pinnedActivitiesQuery.data && $pinnedActivitiesQuery.data.length > 0}
			{#each $pinnedActivitiesQuery.data as activity (activity.id)}
				<button
					type="button"
					class="activity-pill group"
					on:click={() => openModal(activity)}
					title={activity.title}
				>
					{#if activity.icon}
						<span class="text-lg">{activity.icon}</span>
					{/if}
					<span class="max-w-[100px] truncate text-sm font-medium">
						{activity.title}
					</span>
					{#if activity.quantity_enabled && activity.quantity_default}
						<span class="rounded bg-gray-200 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-600 dark:text-gray-300">
							{activity.quantity_default}
						</span>
					{/if}
				</button>
			{/each}

			<!-- Add Activity Button -->
			<a
				href="/activities/new"
				class="activity-pill-add"
				title="Create new activity"
			>
				<Plus class="h-4 w-4" />
			</a>
		{:else}
			<!-- Empty State -->
			<div class="flex w-full flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-200 py-6 dark:border-gray-700">
				<p class="mb-2 text-sm text-gray-500 dark:text-gray-400">No pinned activities</p>
				<a
					href="/activities/new"
					class="inline-flex items-center gap-1 text-sm font-medium text-emerald-600 hover:text-emerald-700 dark:text-emerald-500 dark:hover:text-emerald-400"
				>
					<Plus class="h-4 w-4" />
					Create your first activity
				</a>
			</div>
		{/if}
	</div>
</div>

<!-- Instant Log Modal -->
{#if selectedActivity}
	<InstantLogModal
		activity={selectedActivity}
		bind:isOpen={isModalOpen}
		onClose={closeModal}
		onSuccess={handleSuccess}
	/>
{/if}

<style>
	.activity-pill {
		@apply flex items-center gap-2 rounded-full bg-white px-3 py-2 shadow-sm ring-1 ring-gray-200 transition-all;
		@apply hover:bg-gray-50 hover:shadow-md hover:ring-emerald-300;
		@apply active:scale-95;
		@apply dark:bg-gray-800 dark:ring-gray-700 dark:hover:bg-gray-750 dark:hover:ring-emerald-600;
	}

	.activity-pill-add {
		@apply flex h-10 w-10 items-center justify-center rounded-full border-2 border-dashed border-gray-300 text-gray-400 transition-colors;
		@apply hover:border-emerald-400 hover:text-emerald-500;
		@apply dark:border-gray-600 dark:hover:border-emerald-500 dark:hover:text-emerald-400;
	}
</style>
