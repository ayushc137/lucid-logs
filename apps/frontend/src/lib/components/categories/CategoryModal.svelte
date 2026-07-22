<script lang="ts">
import { ColorPicker, Modal } from '$lib/components/ui';
import { COLOR_PRESETS, DEFAULT_COLOR, getContrastColor } from '$lib/constants';
import { CircleAlert, Save, Tag } from 'lucide-svelte';

interface Props {
	open?: boolean;
	mode: 'create' | 'edit';
	initialName?: string;
	initialColor?: string;
	onSubmit: (data: { name: string; color: string }) => void | Promise<void>;
	isLoading?: boolean;
	error?: string;
}

let {
	open = $bindable(false),
	mode,
	initialName = '',
	initialColor = DEFAULT_COLOR,
	onSubmit,
	isLoading = false,
	error = '',
}: Props = $props();

let name = $state('');
let color = $state(DEFAULT_COLOR);
let useCustomColor = $state(false);
let validationError = $state('');

const isEditing = $derived(mode === 'edit');
const displayError = $derived(validationError || error);

$effect(() => {
	if (!open) return;
	name = initialName;
	color = initialColor;
	useCustomColor = !COLOR_PRESETS.includes(
		initialColor as (typeof COLOR_PRESETS)[number],
	);
	validationError = '';
});

function handleSubmit() {
	validationError = '';
	const trimmedName = name.trim();

	if (!trimmedName) {
		validationError = 'Category name is required';
		return;
	}
	if (trimmedName.length > 40) {
		validationError = 'Category name must be 40 characters or less';
		return;
	}

	onSubmit({ name: trimmedName, color });
}
</script>

<Modal
	bind:open
	size="md"
	title={isEditing ? 'Edit Category' : 'Create New Category'}
	subtitle={isEditing ? 'Update the category name and color' : 'Create a category to organize your tasks'}
>
	{#snippet icon()}
		<Tag class="w-5 h-5 text-primary" />
	{/snippet}

	<div class="space-y-5">
		{#if displayError}
			<div class="alert alert-error" role="alert">
				<CircleAlert class="w-4 h-4" />
				<span>{displayError}</span>
			</div>
		{/if}

		<div class="form-control">
			<label class="label py-1" for="category-name">
				<span class="label-text font-semibold">Name</span>
				<span class="label-text-alt text-base-content/50">{name.length}/40</span>
			</label>
			<input
				id="category-name"
				type="text"
				bind:value={name}
				maxlength={40}
				required
				placeholder="e.g., Work, Personal, Health"
				class="input input-bordered w-full"
				onkeydown={(event: KeyboardEvent) => event.key === 'Enter' && handleSubmit()}
			/>
		</div>

		<div class="form-control">
			<div class="label py-1">
				<span class="label-text font-semibold">Color</span>
				<label class="flex items-center gap-2 cursor-pointer">
					<span class="text-xs text-base-content/60">Custom</span>
					<input
						type="checkbox"
						class="toggle toggle-xs toggle-primary"
						bind:checked={useCustomColor}
					/>
				</label>
			</div>
			<ColorPicker bind:value={color} customMode={useCustomColor} />
		</div>

		<div class="flex items-center gap-3 pt-1">
			<span class="text-sm text-base-content/50">Preview</span>
			<span
				class="badge badge-lg font-medium"
				style="background-color: {color}; color: {getContrastColor(color)}; border: none;"
			>
				{name.trim() || 'Category Name'}
			</span>
		</div>
	</div>

	{#snippet actions()}
		<button type="button" class="btn btn-ghost" onclick={() => (open = false)} disabled={isLoading}>
			Cancel
		</button>
		<button
			type="button"
			class="btn btn-primary gap-2 min-w-[140px]"
			onclick={handleSubmit}
			disabled={isLoading || !name.trim()}
		>
			{#if isLoading}
				<span class="loading loading-spinner loading-sm"></span>
				{isEditing ? 'Saving...' : 'Creating...'}
			{:else}
				<Save class="w-4 h-4" />
				{isEditing ? 'Save Changes' : 'Create Category'}
			{/if}
		</button>
	{/snippet}
</Modal>
