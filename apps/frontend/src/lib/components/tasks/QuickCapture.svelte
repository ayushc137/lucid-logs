<script lang="ts">
import {
	type CreateTaskRequest,
	createTask,
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

function getCurrentTimeString(offsetMs = 0): string {
	const now = new Date(Date.now() + offsetMs);
	return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;
}

let title = $state('');
let categoryId = $state<string | undefined>(undefined);
let timeValue = $state(getCurrentTimeString());
let endTimeValue = $state(getCurrentTimeString(30 * 60000));
let titleInput = $state<HTMLInputElement | null>(null);
let showSuccess = $state(false);

const categoriesQuery = createQuery({
	queryKey: ['categories'],
	queryFn: () => getCategories({ limit: 100 }),
	retry: false,
});

const categories = $derived($categoriesQuery.data?.items || []);

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
		timeValue = getCurrentTimeString();
		endTimeValue = getCurrentTimeString(30 * 60000);
		categoryId = undefined;
		onSaved?.();
		// Re-focus for rapid entry
		setTimeout(() => titleInput?.focus(), 50);
	},
});

function handleSubmit() {
	if (!title.trim()) return;

	const now = new Date();
	const [hours, minutes] = timeValue.split(':').map(Number);
	const startTime = new Date(now);
	startTime.setHours(hours, minutes, 0, 0);
	const [endHours, endMinutes] = endTimeValue.split(':').map(Number);
	let endTime = new Date(now);
	endTime.setHours(endHours, endMinutes, 0, 0);
	if (endTime <= startTime) {
		endTime = new Date(startTime.getTime() + 30 * 60000);
	}

	$createMut.mutate({
		title: title.trim(),
		start_date: startTime.toISOString(),
		end_date: endTime.toISOString(),
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

<section aria-label="Quick capture">
	<div class="bg-base-100 border border-base-200 rounded-xl p-3 shadow-sm">
		<div class="flex items-center gap-2 mb-2.5">
			<Zap class="h-3.5 w-3.5 text-primary" />
			<span class="text-xs font-semibold uppercase tracking-wide text-base-content/60">Quick Capture</span>
			<span class="hidden text-[10px] text-base-content/40 sm:inline ml-auto">
				<kbd class="kbd kbd-xs">⌘</kbd> + <kbd class="kbd kbd-xs">Enter</kbd>
			</span>
		</div>

		<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
			<input
				bind:this={titleInput}
				type="text"
				bind:value={title}
				placeholder="What did you just do?"
				class="input input-sm input-bordered flex-1 min-w-0"
				onkeydown={handleKeydown}
			/>
			<div class="flex items-center gap-2">
				<input
					type="time"
					bind:value={timeValue}
					class="input input-sm input-bordered w-24 shrink-0"
					aria-label="Start time"
				/>
				<span class="text-base-content/30 text-xs">→</span>
				<input
					type="time"
					bind:value={endTimeValue}
					class="input input-sm input-bordered w-24 shrink-0"
					aria-label="End time"
				/>
				<CategoryDropdown
					{categories}
					bind:value={categoryId}
					placeholder="Category"
					showCreateButton={false}
					class="flex-1 sm:w-40"
				/>
				<button
					class="btn btn-sm btn-primary gap-1 shrink-0"
					onclick={handleSubmit}
					disabled={$createMut.isPending || !title.trim()}
				>
					{#if $createMut.isPending}
						<span class="loading loading-spinner loading-xs"></span>
					{:else if showSuccess}
						<Check class="w-3.5 h-3.5" />
					{:else}
						<Zap class="w-3.5 h-3.5" />
					{/if}
					{showSuccess ? 'Saved!' : 'Log it'}
				</button>
			</div>
		</div>
	</div>
</section>
