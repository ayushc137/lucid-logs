<script lang="ts">
import {
	deleteActivity,
	getActivities,
	instantLog,
	type Activity,
	type InstantLogResponse,
} from '$lib/api/activities';
import { ActivityModal, InstantLogModal } from '$lib/components/activities';
import { ConfirmDialog, ErrorAlert } from '$lib/components/ui';
import { cn } from '$lib/utils';
import { getUrlParams, parsers, updateUrlParams } from '$lib/utils/navigation';
import {
	createMutation,
	createQuery,
	useQueryClient,
} from '@tanstack/svelte-query';
import {
	Check,
	Clock,
	Hash,
	LoaderCircle,
	MoreVertical,
	Pin,
	PinOff,
	Play,
	Plus,
	Search,
	SquarePen,
	Target,
	Trash2,
	X,
	Zap,
} from 'lucide-svelte';

const queryClient = useQueryClient();

// Initialize state from URL
const initialParams = getUrlParams<{ q: string; pinned: string }>({
	q: parsers.string(''),
	pinned: parsers.string(''),
});

// Search state
let searchQuery = $state(initialParams.q);
let debouncedSearch = $state(initialParams.q);
let searchTimeout: ReturnType<typeof setTimeout> | null = null;
let filterPinned = $state(initialParams.pinned === 'true' ? true : initialParams.pinned === 'false' ? false : null);

function handleSearchInput(value: string) {
	searchQuery = value;
	if (searchTimeout) clearTimeout(searchTimeout);
	searchTimeout = setTimeout(() => {
		debouncedSearch = value;
	}, 300);
}

// Sync state to URL
$effect(() => {
	updateUrlParams(
		{ 
			q: debouncedSearch,
			pinned: filterPinned === null ? '' : String(filterPinned)
		},
		{ replace: true, keepFocus: true }
	);
});

// Query
const query = createQuery({
	queryKey: ['activities'],
	queryFn: () => getActivities({ limit: 100 }),
	retry: false,
});

const allActivities = $derived($query.data?.items || []);

// Filter activities client-side
const activities = $derived.by(() => {
	let filtered = allActivities;
	
	// Search filter
	if (debouncedSearch) {
		const q = debouncedSearch.toLowerCase();
		filtered = filtered.filter(a => 
			a.title.toLowerCase().includes(q) ||
			a.description?.toLowerCase().includes(q)
		);
	}
	
	// Pinned filter
	if (filterPinned !== null) {
		filtered = filtered.filter(a => a.pinned === filterPinned);
	}
	
	// Sort: pinned first, then by sort_order, then by title
	return filtered.sort((a, b) => {
		if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
		if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order;
		return a.title.localeCompare(b.title);
	});
});

const totalActivities = $derived(allActivities.length);

// Modal states
let createModalOpen = $state(false);
let editingActivity = $state<Activity | null>(null);
let deleteConfirmOpen = $state(false);
let deleteTargetId = $state<string | null>(null);

// Instant log modal
let instantLogActivity = $state<Activity | null>(null);
let instantLogOpen = $state(false);

// Success toast state
let successMessage = $state('');
let successTimeout: ReturnType<typeof setTimeout> | null = null;

function showSuccess(message: string) {
	successMessage = message;
	if (successTimeout) clearTimeout(successTimeout);
	successTimeout = setTimeout(() => {
		successMessage = '';
	}, 3000);
}

// Mutations
const deleteMut = createMutation({
	mutationFn: (id: string) => deleteActivity(id),
	onSuccess: async () => {
		await queryClient.invalidateQueries({ queryKey: ['activities'] });
		deleteConfirmOpen = false;
		deleteTargetId = null;
		showSuccess('Activity deleted');
	},
});

function handleCreate() {
	editingActivity = null;
	createModalOpen = true;
}

function startEdit(activity: Activity) {
	editingActivity = activity;
	createModalOpen = true;
}

function handleModalClose() {
	createModalOpen = false;
	editingActivity = null;
	queryClient.invalidateQueries({ queryKey: ['activities'] });
}

function confirmDelete(id: string, e: Event) {
	e.stopPropagation();
	deleteTargetId = id;
	deleteConfirmOpen = true;
}

function handleDelete() {
	if (deleteTargetId) {
		$deleteMut.mutate(deleteTargetId);
	}
}

function openInstantLog(activity: Activity, e: Event) {
	e.stopPropagation();
	instantLogActivity = activity;
	instantLogOpen = true;
}

function handleInstantLogSuccess(response: InstantLogResponse) {
	const goalsUpdated = response.goals_updated?.length ?? 0;
	if (goalsUpdated > 0) {
		showSuccess(`Logged! Updated ${goalsUpdated} goal${goalsUpdated > 1 ? 's' : ''}`);
	} else {
		showSuccess('Activity logged!');
	}
}

function formatDuration(seconds: number): string {
	const mins = Math.round(seconds / 60);
	if (mins < 60) return `${mins}m`;
	const hrs = Math.floor(mins / 60);
	const remainMins = mins % 60;
	return remainMins > 0 ? `${hrs}h ${remainMins}m` : `${hrs}h`;
}

function formatLastUsed(dateStr?: string): string {
	if (!dateStr) return 'Never';
	const date = new Date(dateStr);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
	
	if (diffDays === 0) return 'Today';
	if (diffDays === 1) return 'Yesterday';
	if (diffDays < 7) return `${diffDays}d ago`;
	if (diffDays < 30) return `${Math.floor(diffDays / 7)}w ago`;
	return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

const isSearching = $derived(searchQuery !== debouncedSearch);

// Highlight search matches
function highlightText(text: string, query: string): string {
	if (!query || !text) return text;
	const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	const regex = new RegExp(`(${escapedQuery})`, 'gi');
	return text.replace(
		regex,
		'<mark class="bg-warning text-warning-content rounded px-0.5">$1</mark>'
	);
}
</script>

<svelte:head>
	<title>Activities - Lucid Logs</title>
</svelte:head>

<!-- Success Toast -->
{#if successMessage}
	<div class="toast toast-top toast-end z-50">
		<div class="alert alert-success shadow-lg">
			<Check class="w-4 h-4" />
			<span>{successMessage}</span>
		</div>
	</div>
{/if}

<div class="max-w-6xl mx-auto space-y-6">
	<!-- Header -->
	<div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
		<div>
			<h1 class="text-3xl font-bold flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
					<Zap class="w-5 h-5 text-primary" />
				</div>
				Activities
			</h1>
			<p class="text-sm opacity-60 mt-1">
				{#if debouncedSearch || filterPinned !== null}
					Found {activities.length} matching activities
				{:else}
					{totalActivities} quick-log activities
				{/if}
			</p>
		</div>
		<button
			class="btn btn-primary gap-2 shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-all"
			onclick={handleCreate}
		>
			<Plus class="w-4 h-4" />
			New Activity
		</button>
	</div>

	<!-- Search & Filters -->
	<div class="card bg-base-100 shadow-lg border border-base-200">
		<div class="card-body p-4">
			<div class="flex flex-col sm:flex-row gap-3">
				<!-- Search -->
				<div class="relative flex-1">
					{#if isSearching || $query.isFetching}
						<LoaderCircle class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-50 animate-spin" />
					{:else}
						<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-50" />
					{/if}
					<input
						type="text"
						placeholder="Search activities..."
						class="input input-bordered w-full pl-10"
						value={searchQuery}
						oninput={(e) => handleSearchInput(e.currentTarget.value)}
					/>
					{#if searchQuery}
						<button
							class="absolute right-3 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle"
							onclick={() => {
								searchQuery = '';
								debouncedSearch = '';
							}}
						>
							<X class="w-3 h-3" />
						</button>
					{/if}
				</div>

				<!-- Filter: Pinned -->
				<div class="flex gap-2">
					<button
						class={cn(
							"btn btn-sm gap-1",
							filterPinned === null && "btn-ghost",
							filterPinned === true && "btn-primary",
							filterPinned === false && "btn-outline"
						)}
						onclick={() => {
							if (filterPinned === null) filterPinned = true;
							else if (filterPinned === true) filterPinned = false;
							else filterPinned = null;
						}}
					>
						<Pin class="w-3 h-3" />
						{filterPinned === null ? 'All' : filterPinned ? 'Pinned' : 'Unpinned'}
					</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Activities Grid/Table -->
	{#if $query.isLoading}
		<div class="card bg-base-100 shadow-lg border border-base-200">
			<div class="card-body py-16 text-center">
				<span class="loading loading-spinner loading-lg text-primary mx-auto"></span>
				<p class="opacity-60 mt-4">Loading activities...</p>
			</div>
		</div>
	{:else if $query.isError}
		<ErrorAlert
			message="Failed to load activities. Please try again."
			onRetry={() => $query.refetch()}
		/>
	{:else if activities.length === 0}
		<div class="card bg-base-100 shadow-lg border border-base-200">
			<div class="card-body py-12 text-center">
				{#if debouncedSearch || filterPinned !== null}
					<Search class="w-12 h-12 opacity-30 mx-auto mb-4" />
					<h2 class="text-xl font-semibold">No matching activities</h2>
					<p class="opacity-60 mt-1">Try adjusting your search or filters</p>
					<button
						class="btn btn-ghost btn-sm mt-4"
						onclick={() => {
							searchQuery = '';
							debouncedSearch = '';
							filterPinned = null;
						}}
					>
						Clear filters
					</button>
				{:else}
					<div class="w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
						<Zap class="w-10 h-10 text-primary" />
					</div>
					<h2 class="text-2xl font-bold">No activities yet</h2>
					<p class="opacity-60 mt-2 max-w-md mx-auto">
						Create activities to quickly log tasks and track progress towards your goals
					</p>
					<button
						class="btn btn-primary mt-6 gap-2 shadow-lg shadow-primary/20"
						onclick={handleCreate}
					>
						<Plus class="w-4 h-4" />
						Create Your First Activity
					</button>
				{/if}
			</div>
		</div>
	{:else}
		<!-- Activities Table -->
		<div class="card bg-base-100 shadow-lg border border-base-200 overflow-hidden">
			<div class="overflow-x-auto">
				<table class="table table-lg">
					<thead class="bg-base-100 border-b border-base-300">
						<tr>
							<th class="w-12"></th>
							<th>Activity</th>
							<th class="hidden md:table-cell">Settings</th>
							<th class="hidden lg:table-cell">Goals</th>
							<th class="hidden sm:table-cell">Usage</th>
							<th class="w-32 text-right">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each activities as activity (activity.id)}
							<tr
								class="hover:bg-base-200/50 transition-colors group cursor-pointer"
								onclick={() => startEdit(activity)}
							>
								<!-- Icon -->
								<td>
									<div class="relative">
										<span class="text-2xl">{activity.icon || '⚡'}</span>
										{#if activity.pinned}
											<Pin class="w-3 h-3 text-primary absolute -top-1 -right-1" />
										{/if}
									</div>
								</td>

								<!-- Title & Description -->
								<td>
									<div>
										<span class="font-semibold">
											{#if debouncedSearch}
												{@html highlightText(activity.title, debouncedSearch)}
											{:else}
												{activity.title}
											{/if}
										</span>
										{#if activity.description}
											<p class="text-sm opacity-60 line-clamp-1 mt-0.5">
												{#if debouncedSearch}
													{@html highlightText(activity.description, debouncedSearch)}
												{:else}
													{activity.description}
												{/if}
											</p>
										{/if}
										{#if activity.category}
											<span
												class="badge badge-sm mt-1"
												style="background-color: {activity.category.color}20; color: {activity.category.color};"
											>
												{activity.category.name}
											</span>
										{/if}
									</div>
								</td>

								<!-- Settings -->
								<td class="hidden md:table-cell">
									<div class="flex flex-wrap gap-1">
										{#if activity.default_duration}
											<span class="badge badge-sm badge-ghost gap-1">
												<Clock class="w-3 h-3" />
												{formatDuration(activity.default_duration)}
											</span>
										{/if}
										{#if activity.quantity_enabled}
											<span class="badge badge-sm badge-ghost gap-1">
												<Hash class="w-3 h-3" />
												{activity.quantity_default ?? 1}
												{#if activity.quantity_unit_id}
													{activity.quantity_unit_id.replace('units:', '')}
												{/if}
											</span>
										{/if}
										<span class={cn(
											"badge badge-sm",
											activity.default_impact === 'positive' && "badge-success",
											activity.default_impact === 'negative' && "badge-error",
											activity.default_impact === 'neutral' && "badge-ghost"
										)}>
											{activity.default_impact}
										</span>
									</div>
								</td>

								<!-- Goals -->
								<td class="hidden lg:table-cell">
									{#if activity.goals && activity.goals.length > 0}
										<div class="flex items-center gap-1">
											<Target class="w-3 h-3 opacity-50" />
											<span class="text-sm">{activity.goals.length} linked</span>
										</div>
									{:else}
										<span class="text-sm opacity-40">None</span>
									{/if}
								</td>

								<!-- Usage -->
								<td class="hidden sm:table-cell">
									<div class="text-sm">
										<span class="font-medium">{activity.use_count}</span>
										<span class="opacity-50"> uses</span>
										<p class="text-xs opacity-50">
											{formatLastUsed(activity.last_used_at)}
										</p>
									</div>
								</td>

								<!-- Actions -->
								<td>
									<div class="flex gap-1 justify-end">
										<!-- Quick Log Button -->
										<button
											class="btn btn-primary btn-sm gap-1"
											onclick={(e) => openInstantLog(activity, e)}
											title="Quick log"
										>
											<Play class="w-3 h-3" />
											Log
										</button>

										<!-- Dropdown -->
										<div class="dropdown dropdown-end">
											<button
												tabindex="0"
												class="btn btn-ghost btn-sm btn-square opacity-0 group-hover:opacity-100 transition-opacity"
												onclick={(e) => e.stopPropagation()}
											>
												<MoreVertical class="w-4 h-4" />
											</button>
											<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
											<ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow-lg bg-base-100 rounded-box w-44 border border-base-300">
												<li>
													<button onclick={(e) => { e.stopPropagation(); startEdit(activity); }}>
														<SquarePen class="w-4 h-4" />
														Edit
													</button>
												</li>
												<li>
													<button
														class="text-error"
														onclick={(e) => confirmDelete(activity.id, e)}
													>
														<Trash2 class="w-4 h-4" />
														Delete
													</button>
												</li>
											</ul>
										</div>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- Footer -->
			<div class="px-4 py-3 bg-base-200/50 border-t border-base-200 text-sm opacity-60 flex items-center justify-between">
				<span>{activities.length} activities</span>
				{#if $query.isFetching}
					<span class="flex items-center gap-2 text-primary">
						<LoaderCircle class="w-4 h-4 animate-spin" />
						Loading...
					</span>
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Activity Modal -->
<ActivityModal
	bind:open={createModalOpen}
	activity={editingActivity}
	onClose={handleModalClose}
/>

<!-- Instant Log Modal -->
{#if instantLogActivity}
	<InstantLogModal
		activity={instantLogActivity}
		isOpen={instantLogOpen}
		onClose={() => {
			instantLogOpen = false;
			instantLogActivity = null;
		}}
		onSuccess={handleInstantLogSuccess}
	/>
{/if}

<!-- Delete Confirmation Modal -->
<ConfirmDialog
	bind:open={deleteConfirmOpen}
	title="Delete Activity"
	message="Are you sure you want to delete this activity? This action cannot be undone."
	confirmText="Delete"
	destructive={true}
	loading={$deleteMut.isPending}
	onConfirm={handleDelete}
	onCancel={() => {
		deleteConfirmOpen = false;
		deleteTargetId = null;
	}}
/>
