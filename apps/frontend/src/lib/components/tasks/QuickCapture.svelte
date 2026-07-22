<script lang="ts">
import {
	type CreateTaskRequest,
	createTask,
	getLastTaskEndTime,
	updateTask,
} from '$lib/api';
import { getCategories } from '$lib/api/categories';
import { CategoryDropdown } from '$lib/components/ui';
import {
	createMutation,
	createQuery,
	useQueryClient,
} from '@tanstack/svelte-query';
import { Check, Zap } from 'lucide-svelte';
import { onMount } from 'svelte';

interface Props {
	/** Called after successful save */
	onSaved?: () => void;
	/** Auto-focus the title input on mount */
	autoFocus?: boolean;
}

let { onSaved, autoFocus = true }: Props = $props();

const queryClient = useQueryClient();

let title = $state('');
let categoryId = $state<string | undefined>(undefined);
let titleInput = $state<HTMLInputElement | null>(null);
let showSuccess = $state(false);

const categoriesQuery = createQuery({
	queryKey: ['categories'],
	queryFn: () => getCategories({ limit: 100 }),
	retry: false,
});

const categories = $derived($categoriesQuery.data?.items || []);

// Get last task end time to pre-fill start/end
const lastTaskEndQuery = createQuery({
	queryKey: ['tasks', 'last-end-time'],
	queryFn: getLastTaskEndTime,
	retry: false,
});

const createMut = createMutation({
	mutationFn: async (data: CreateTaskRequest) => {
		// Quick capture = already done → create then mark complete in one flow
		const task = await createTask(data);
		return updateTask(task.id, { completed: true });
	},
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ['tasks'] });
		showSuccess = true;
		setTimeout(() => {
			showSuccess = false;
		}, 2000);
		title = '';
		categoryId = undefined;
		onSaved?.();
		// Re-focus for rapid entry
		setTimeout(() => titleInput?.focus(), 50);
	},
});

function handleSubmit() {
	if (!title.trim()) return;

	const now = new Date();
	const startDate = new Date(now);
	const endDate = new Date(now);

	// If we have a last task end time, chain from it
	const lastEnd = $lastTaskEndQuery.data?.end_time;
	if (lastEnd) {
		const lastEndDate = new Date(lastEnd);
		// Only chain if it was within the last 4 hours (same session)
		if (now.getTime() - lastEndDate.getTime() < 4 * 60 * 60 * 1000) {
			startDate.setTime(lastEndDate.getTime());
		}
	}

	$createMut.mutate({
		title: title.trim(),
		start_date: startDate.toISOString(),
		end_date: endDate.toISOString(),
		category_id: categoryId || undefined,
	});
}

function handleKeydown(e: KeyboardEvent) {
	if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
		e.preventDefault();
		handleSubmit();
	}
}

onMount(() => {
	if (autoFocus) {
		titleInput?.focus();
	}
});
</script>

<section class="rounded-box bg-base-200/30 p-4" aria-labelledby="quick-capture-label">
	<div class="mb-3 flex items-center justify-between gap-3">
		<div
			id="quick-capture-label"
			class="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-base-content/50"
		>
			<Zap class="h-3.5 w-3.5 text-primary" />
			Quick Capture
		</div>
		<span class="hidden text-[10px] text-base-content/40 sm:inline">
			<kbd class="kbd kbd-xs">⌘</kbd> + <kbd class="kbd kbd-xs">Enter</kbd>
		</span>
	</div>

	<div class="flex flex-col gap-3 md:flex-row md:items-center">
		<div class="min-w-0 flex-1">
			<input
				bind:this={titleInput}
				type="text"
				bind:value={title}
				placeholder="What did you just do?"
				class="input input-bordered w-full"
				onkeydown={handleKeydown}
			/>
		</div>

		<div class="w-full md:w-64 md:shrink-0">
			<CategoryDropdown
				{categories}
				bind:value={categoryId}
				placeholder="Category (optional)"
				showCreateButton={false}
			/>
		</div>

		<button
			class="btn btn-primary gap-1.5 md:shrink-0"
			onclick={handleSubmit}
			disabled={$createMut.isPending || !title.trim()}
		>
			{#if $createMut.isPending}
				<span class="loading loading-spinner loading-xs"></span>
			{:else if showSuccess}
				<Check class="w-4 h-4" />
			{:else}
				<Zap class="w-4 h-4" />
			{/if}
			{showSuccess ? 'Saved!' : 'Log it'}
		</button>
	</div>
</section>
