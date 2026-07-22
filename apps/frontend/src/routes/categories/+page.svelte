<script lang="ts">
import {
	type Category,
	createCategory,
	deleteCategory,
	getCategories,
	updateCategory,
} from '$lib/api/categories';
import { CategoryModal } from '$lib/components/categories';
import { ConfirmDialog, ErrorAlert, Fab } from '$lib/components/ui';
import { getUrlParams, parsers, updateUrlParams } from '$lib/utils/navigation';
import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
import {
	LoaderCircle,
	Palette,
	Plus,
	Search,
	SquarePen,
	Trash2,
	X,
} from 'lucide-svelte';

const queryClient = useQueryClient();
const initialParams = getUrlParams<{ q: string }>({ q: parsers.string('') });

let searchQuery = $state(initialParams.q);
let debouncedSearch = $state(initialParams.q);
let searchTimeout: ReturnType<typeof setTimeout> | null = null;
let currentParamsJson = '';

let modalOpen = $state(false);
let editingCategory = $state<Category | null>(null);
let modalError = $state('');
let deleteConfirmOpen = $state(false);
let deleteTargetId = $state<string | null>(null);

function handleSearchInput(value: string) {
	searchQuery = value;
	if (searchTimeout) clearTimeout(searchTimeout);
	searchTimeout = setTimeout(() => {
		debouncedSearch = value;
	}, 300);
}

function clearSearch() {
	searchQuery = '';
	debouncedSearch = '';
}

$effect(() => {
	updateUrlParams({ q: debouncedSearch }, { replace: true, keepFocus: true });
});

const queryParams = $derived({
	limit: 100,
	search: debouncedSearch || undefined,
});

const query = createQuery({
	queryKey: ['categories-list'],
	queryFn: () => getCategories(queryParams),
	retry: false,
});

$effect(() => {
	const newParamsJson = JSON.stringify(queryParams);
	if (currentParamsJson === '') {
		currentParamsJson = newParamsJson;
		return;
	}
	if (newParamsJson !== currentParamsJson) {
		currentParamsJson = newParamsJson;
		$query.refetch();
	}
});

const categories = $derived($query.data?.items || []);
const totalCategories = $derived($query.data?.total || 0);
const isSearching = $derived(searchQuery !== debouncedSearch);

async function finishSave() {
	await queryClient.invalidateQueries({ queryKey: ['categories-list'] });
	await queryClient.invalidateQueries({ queryKey: ['categories'] });
	modalOpen = false;
	editingCategory = null;
}

const createMut = createMutation({
	mutationFn: (data: { name: string; color: string }) => createCategory(data),
	onSuccess: finishSave,
	onError: (error: Error) => {
		modalError = error.message || 'Failed to create category';
	},
});

const updateMut = createMutation({
	mutationFn: ({ id, data }: { id: string; data: { name: string; color: string } }) =>
		updateCategory(id, data),
	onSuccess: finishSave,
	onError: (error: Error) => {
		modalError = error.message || 'Failed to update category';
	},
});

const deleteMut = createMutation({
	mutationFn: (id: string) => deleteCategory(id),
	onSuccess: async () => {
		await queryClient.invalidateQueries({ queryKey: ['categories-list'] });
		await queryClient.invalidateQueries({ queryKey: ['categories'] });
		deleteConfirmOpen = false;
		deleteTargetId = null;
	},
});

function handleCreate() {
	editingCategory = null;
	modalError = '';
	modalOpen = true;
}

function startEdit(category: Category) {
	editingCategory = category;
	modalError = '';
	modalOpen = true;
}

function handleCategorySubmit(data: { name: string; color: string }) {
	modalError = '';
	if (editingCategory) {
		$updateMut.mutate({ id: editingCategory.id, data });
	} else {
		$createMut.mutate(data);
	}
}

function confirmDelete(id: string, event: Event) {
	event.stopPropagation();
	deleteTargetId = id;
	deleteConfirmOpen = true;
}

function handleDelete() {
	if (deleteTargetId) $deleteMut.mutate(deleteTargetId);
}

function highlightText(text: string, search: string): string {
	if (!search || !text) return text;
	const escapedSearch = search.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	return text.replace(
		new RegExp(`(${escapedSearch})`, 'gi'),
		'<mark class="bg-warning text-warning-content rounded px-0.5">$1</mark>',
	);
}
</script>

<svelte:head>
	<title>Categories - Lucid Logs</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
	<div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Categories</h1>
			<p class="text-sm opacity-60 mt-1">
				{#if debouncedSearch}
					Found {categories.length} matching categories
				{:else}
					{totalCategories} categories
				{/if}
			</p>
		</div>
		<button
			class="hidden lg:inline-flex btn btn-primary gap-2 shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-all"
			onclick={handleCreate}
		>
			<Plus class="w-4 h-4" />
			New Category
		</button>
	</div>

	<div class="card bg-base-100 shadow-lg border border-base-200">
		<div class="card-body p-4">
			<div class="relative">
				{#if isSearching || $query.isFetching}
					<LoaderCircle class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-50 animate-spin" />
				{:else}
					<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-50" />
				{/if}
				<input
					type="text"
					placeholder="Search categories..."
					class="input input-bordered w-full pl-10"
					value={searchQuery}
					oninput={(event) => handleSearchInput(event.currentTarget.value)}
				/>
				{#if searchQuery}
					<button
						class="absolute right-3 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle"
						onclick={clearSearch}
						aria-label="Clear search"
					>
						<X class="w-3 h-3" />
					</button>
				{/if}
			</div>
		</div>
	</div>

	{#if $query.isLoading}
		<div class="card bg-base-100 shadow-lg border border-base-200">
			<div class="card-body py-16 text-center">
				<span class="loading loading-spinner loading-lg text-primary mx-auto"></span>
				<p class="opacity-60 mt-4">Loading categories...</p>
			</div>
		</div>
	{:else if $query.isError}
		<ErrorAlert
			message="Failed to load categories. Please try again."
			onRetry={() => $query.refetch()}
		/>
	{:else if categories.length === 0}
		<div class="card bg-base-100 shadow-lg border border-base-200">
			<div class="card-body py-12 text-center">
				{#if debouncedSearch}
					<Search class="w-12 h-12 opacity-30 mx-auto mb-4" />
					<h2 class="text-xl font-semibold">No matching categories</h2>
					<p class="opacity-60 mt-1">Try adjusting your search term</p>
					<button class="btn btn-ghost btn-sm mt-4" onclick={clearSearch}>Clear search</button>
				{:else}
					<div class="w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
						<Palette class="w-10 h-10 text-primary" />
					</div>
					<h2 class="text-2xl font-bold">No categories yet</h2>
					<p class="opacity-60 mt-2 max-w-md mx-auto">Create categories to organize your tasks</p>
					<button class="btn btn-primary mt-6 gap-2 shadow-lg shadow-primary/20" onclick={handleCreate}>
						<Plus class="w-4 h-4" />
						Create Your First Category
					</button>
				{/if}
			</div>
		</div>
	{:else}
		<div class="hidden lg:block card bg-base-100 shadow-lg border border-base-200 overflow-hidden">
			<div class="overflow-x-auto">
				<table class="table table-lg">
					<thead class="bg-base-100 border-b border-base-300">
						<tr>
							<th class="w-16">Color</th>
							<th>Name</th>
							<th>Created</th>
							<th class="w-24 text-right">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each categories as category (category.id)}
							<tr class="hover:bg-base-200/50 transition-colors group cursor-pointer" onclick={() => startEdit(category)}>
								<td>
									<div class="w-8 h-8 rounded-lg shadow-sm" style="background-color: {category.color};"></div>
								</td>
								<td>
									<div class="font-semibold">
										{#if debouncedSearch}
											{@html highlightText(category.name, debouncedSearch)}
										{:else}
											{category.name}
										{/if}
									</div>
								</td>
								<td class="text-sm text-base-content/60">
									{new Date(category.created_at).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' })}
								</td>
								<td>
									<div class="flex gap-1 justify-end opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
										<button class="btn btn-ghost btn-sm btn-square" onclick={(event) => { event.stopPropagation(); startEdit(category); }} aria-label="Edit category">
											<SquarePen class="w-4 h-4" />
										</button>
										<button class="btn btn-ghost btn-sm btn-square text-error" onclick={(event) => confirmDelete(category.id, event)} aria-label="Delete category">
											<Trash2 class="w-4 h-4" />
										</button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<div class="px-4 py-3 bg-base-200/50 border-t border-base-200 text-sm opacity-60 flex items-center justify-between">
				<span>{categories.length} categories</span>
				{#if $query.isFetching}
					<span class="flex items-center gap-2 text-primary"><LoaderCircle class="w-4 h-4 animate-spin" /> Loading...</span>
				{/if}
			</div>
		</div>

		<div class="lg:hidden space-y-2">
			{#each categories as category (category.id)}
				<article
					class="card bg-base-100 border border-base-200 shadow-sm active:scale-[0.99] transition-all cursor-pointer"
					onclick={() => startEdit(category)}
					role="button"
					tabindex="0"
					onkeydown={(event) => event.key === 'Enter' && startEdit(category)}
				>
					<div class="card-body p-3">
						<div class="flex items-center gap-3">
							<div class="w-9 h-9 rounded-lg shadow-sm shrink-0" style="background-color: {category.color};"></div>
							<div class="flex-1 min-w-0">
								<p class="font-semibold text-[15px] truncate">
									{#if debouncedSearch}
										{@html highlightText(category.name, debouncedSearch)}
									{:else}
										{category.name}
									{/if}
								</p>
								<p class="text-xs text-base-content/50 mt-0.5">
									{new Date(category.created_at).toLocaleDateString()}
								</p>
							</div>
							<button class="btn btn-ghost btn-sm btn-square" onclick={(event) => { event.stopPropagation(); startEdit(category); }} aria-label="Edit category">
								<SquarePen class="w-4 h-4" />
							</button>
							<button class="btn btn-ghost btn-sm btn-square text-error" onclick={(event) => confirmDelete(category.id, event)} aria-label="Delete category">
								<Trash2 class="w-4 h-4" />
							</button>
						</div>
					</div>
				</article>
			{/each}
			<div class="flex items-center justify-between px-1 pt-1 text-sm text-base-content/60">
				<span>{categories.length} categories</span>
				{#if $query.isFetching}<LoaderCircle class="w-4 h-4 animate-spin text-primary" />{/if}
			</div>
		</div>
	{/if}
</div>

<Fab onclick={handleCreate} label="New Category" />

<CategoryModal
	bind:open={modalOpen}
	mode={editingCategory ? 'edit' : 'create'}
	initialName={editingCategory?.name || ''}
	initialColor={editingCategory?.color}
	onSubmit={handleCategorySubmit}
	isLoading={$createMut.isPending || $updateMut.isPending}
	error={modalError}
/>

<ConfirmDialog
	bind:open={deleteConfirmOpen}
	title="Delete Category"
	message="Are you sure you want to delete this category? This action cannot be undone. Tasks using this category will be unassigned."
	confirmText="Delete"
	destructive={true}
	loading={$deleteMut.isPending}
	onConfirm={handleDelete}
	onCancel={() => {
		deleteConfirmOpen = false;
		deleteTargetId = null;
	}}
/>
