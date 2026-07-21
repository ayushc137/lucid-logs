<script lang="ts">
import {
	type CreateTaskRequest,
	createTask,
	getLastTaskEndTime,
	updateTask,
} from '$lib/api';
import { getCategories } from '$lib/api/categories';
import { Card, CategoryDropdown } from '$lib/components/ui';
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

<Card variant="bordered" class="border-primary/20 bg-primary/5">
	<div class="flex items-center gap-2 mb-3">
		<Zap class="w-4 h-4 text-primary" />
		<span class="text-xs font-semibold uppercase text-primary/70">Quick Capture</span>
	</div>

	<div class="space-y-3">
		<!-- Title -->
		<input
			bind:this={titleInput}
			type="text"
			bind:value={title}
			placeholder="What did you just do?"
			class="input input-bordered w-full text-lg font-medium bg-base-100 focus:border-primary focus:outline-none"
			onkeydown={handleKeydown}
		/>

		<!-- Category (optional, collapsible) -->
		<details class="group">
			<summary class="text-xs text-base-content/50 cursor-pointer select-none hover:text-base-content/70 transition-colors">
				Add category <span class="group-open:hidden">▸</span><span class="hidden group-open:inline">▾</span>
			</summary>
			<div class="mt-2">
				<CategoryDropdown
					{categories}
					bind:value={categoryId}
					placeholder="Select category..."
					showCreateButton={false}
				/>
			</div>
		</details>

		<!-- Actions -->
		<div class="flex items-center justify-between pt-1">
			<span class="text-[10px] text-base-content/40">
				<kbd class="kbd kbd-xs">⌘</kbd> + <kbd class="kbd kbd-xs">Enter</kbd>
			</span>
			<button
				class="btn btn-primary btn-sm gap-1.5 min-w-[100px]"
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
				{showSuccess ? "Saved!" : "Log it"}
			</button>
		</div>
	</div>
</Card>
