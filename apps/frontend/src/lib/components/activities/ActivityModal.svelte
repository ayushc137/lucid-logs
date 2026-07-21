<script lang="ts">
	import { browser } from '$app/environment';
	import {
		createActivity,
		updateActivity,
		type Activity,
		type CreateActivityRequest,
	} from '$lib/api/activities';
	import { createCategory, getCategories } from '$lib/api/categories';
	import { getGoals, type Goal } from '$lib/api/goals';
	import { getUnits } from '$lib/api/units';
	import { CategoryDropdown, ColorPicker } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { onMount } from 'svelte';
	import {
		Check,
		ChevronRight,
		CircleAlert,
		Clock,
		Flame,
		Pin,
		Save,
		Search,
		Tag,
		Target,
		TrendingDown,
		TrendingUp,
		X,
		Zap,
	} from 'lucide-svelte';
	import { scale } from 'svelte/transition';

	// Per-goal link configuration (simplified - no default_impact at activity level)
	interface GoalLinkConfig {
		goal_id: string;
		auto_link_tasks: boolean;
		default_quantity?: number;
		impact_type: 'positive' | 'negative';
	}

	// Props
	interface Props {
		open: boolean;
		activity?: Activity | null;
		onClose: () => void;
	}

	let { open = $bindable(), activity = null, onClose }: Props = $props();

	const queryClient = useQueryClient();
	const isEditing = $derived(!!activity);

	// Form state - Basic
	let title = $state('');
	let icon = $state('⚡');
	let description = $state('');
	let categoryId = $state<string | undefined>(undefined);
	let pinned = $state(false);
	let defaultCompleted = $state(true);

	// Form state - Duration
	let durationEnabled = $state(false);
	let defaultDuration = $state(30); // minutes

	// Form state - Goals
	let goalConfigs = $state<GoalLinkConfig[]>([]);
	let goalSearchQuery = $state('');

	// UI state
	let error = $state('');
	let activeTab = $state<'details' | 'goals'>('details');

	// Queries
	const categoriesQuery = createQuery({
		queryKey: ['categories'],
		queryFn: () => getCategories({ limit: 100 }),
	});

	const goalsQuery = createQuery({
		queryKey: ['goals', 'active'],
		queryFn: () => getGoals({ status: 'active', limit: 100 }),
	});

	const unitsQuery = createQuery({
		queryKey: ['units'],
		queryFn: () => getUnits(),
	});

	const categories = $derived($categoriesQuery.data?.items || []);
	const goals = $derived($goalsQuery.data?.items || []);
	const units = $derived($unitsQuery.data?.items || []);

	// Emoji picker state
	let showEmojiPicker = $state(false);
	let emojiPickerReady = $state(false);
	let showFullPicker = $state(false);

	// Category creation state
	let showCreateCategory = $state(false);
	let newCategoryName = $state('');
	let newCategoryColor = $state('#6366f1');
	let useCustomCategoryColor = $state(false);

	// Suggested emojis for activities
	const suggestedEmojis = {
		'Quick Actions': ['⚡', '✅', '📝', '🔔', '⏰', '📌', '🎯', '💡', '🚀', '✨'],
		'Health & Fitness': ['💧', '🏃', '🧘', '💪', '🚴', '🏊', '😴', '🥗', '💊', '🧠'],
		'Learning & Work': ['📚', '✍️', '💼', '📊', '🖥️', '📧', '📞', '🎓', '📖', '🔬'],
		'Lifestyle': ['☕', '🎨', '🎵', '🌿', '🏠', '🧹', '🛒', '🍳', '🚗', '✈️'],
	};

	const quickEmojis = ['⚡', '💧', '🏃', '📚', '🧘', '💪', '☕', '🎯', '✍️', '🎨', '🎵', '🌿'];

	// Initialize emoji-picker-element
	onMount(() => {
		if (browser) {
			import('emoji-picker-element').then(() => {
				emojiPickerReady = true;
			});
		}
	});

	function handleEmojiSelect(event: CustomEvent) {
		if (event.detail?.unicode) {
			icon = event.detail.unicode;
			showEmojiPicker = false;
			showFullPicker = false;
		}
	}

	function selectQuickEmoji(emoji: string) {
		icon = emoji;
		showEmojiPicker = false;
		showFullPicker = false;
	}

	// Derived: selected goal IDs and available goals
	const selectedGoalIds = $derived(goalConfigs.map((gc) => gc.goal_id));
	const availableGoals = $derived(
		goals.filter(
			(g) =>
				!selectedGoalIds.includes(g.id) &&
				g.title.toLowerCase().includes(goalSearchQuery.toLowerCase())
		)
	);
	const linkedGoals = $derived(
		goalConfigs
			.map((gc) => ({ ...gc, goal: goals.find((g) => g.id === gc.goal_id) }))
			.filter((gc) => gc.goal)
	);

	// Auto-pick category from first linked goal (if not already set)
	const inferredCategory = $derived(() => {
		if (categoryId) return null;
		const firstGoalWithCategory = linkedGoals.find((lg) => lg.goal?.category);
		return firstGoalWithCategory?.goal?.category || null;
	});

	// Reset form when modal opens
	$effect(() => {
		if (open) {
			if (activity) {
				// Edit mode - populate form
				title = activity.title;
				icon = activity.icon || '⚡';
				description = activity.description || '';
				categoryId = activity.category?.id;
				pinned = activity.pinned;
				defaultCompleted = activity.default_completed;
				durationEnabled = !!activity.default_duration;
				defaultDuration = activity.default_duration ? Math.round(activity.default_duration / 60) : 30;
				goalConfigs =
					activity.goals?.map((g) => ({
						goal_id: g.goal_id,
						auto_link_tasks: g.auto_link_tasks,
						default_quantity: g.default_quantity,
						impact_type: (g.default_impact === 'negative' ? 'negative' : 'positive') as 'positive' | 'negative',
					})) || [];
			} else {
				// Create mode - reset form
				title = '';
				icon = '⚡';
				description = '';
				categoryId = undefined;
				pinned = false;
				defaultCompleted = true;
				durationEnabled = false;
				defaultDuration = 30;
				goalConfigs = [];
			}
			error = '';
			activeTab = 'details';
			goalSearchQuery = '';
			showEmojiPicker = false;
			showFullPicker = false;
			showCreateCategory = false;
			newCategoryName = '';
		}
	});

	// Mutations
	const createMut = createMutation({
		mutationFn: (data: CreateActivityRequest) => createActivity(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['activities'] });
			onClose();
		},
		onError: (err: Error) => {
			error = err.message || 'Failed to create activity';
		},
	});

	const updateMut = createMutation({
		mutationFn: (data: CreateActivityRequest) => updateActivity(activity!.id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['activities'] });
			onClose();
		},
		onError: (err: Error) => {
			error = err.message || 'Failed to update activity';
		},
	});

	const isPending = $derived($createMut.isPending || $updateMut.isPending);

	function handleSubmit() {
		if (!title.trim()) {
			error = 'Title is required';
			activeTab = 'details';
			return;
		}

		// Use inferred category if none selected
		const finalCategoryId = categoryId || inferredCategory()?.id;

		const data: CreateActivityRequest = {
			title: title.trim(),
			icon,
			description: description.trim() || undefined,
			category_id: finalCategoryId,
			pinned,
			default_completed: defaultCompleted,
			default_duration: durationEnabled ? defaultDuration * 60 : undefined,
			quantity_enabled: goalConfigs.some((gc) => {
				const goal = goals.find((g) => g.id === gc.goal_id);
				return goal?.target?.unit_id;
			}),
			goal_links: goalConfigs.map((gc) => ({
				goal_id: gc.goal_id,
				auto_link_tasks: gc.auto_link_tasks,
				default_quantity: gc.default_quantity,
				default_impact: gc.impact_type,
			})),
		};

		if (isEditing) {
			$updateMut.mutate(data);
		} else {
			$createMut.mutate(data);
		}
	}

	function addGoal(goal: Goal) {
		goalConfigs = [
			...goalConfigs,
			{
				goal_id: goal.id,
				auto_link_tasks: true,
				default_quantity: goal.target ? 1 : undefined,
				impact_type: 'positive',
			},
		];
	}

	function removeGoal(goalId: string) {
		goalConfigs = goalConfigs.filter((gc) => gc.goal_id !== goalId);
	}

	function toggleImpact(goalId: string) {
		goalConfigs = goalConfigs.map((gc) =>
			gc.goal_id === goalId
				? { ...gc, impact_type: gc.impact_type === 'positive' ? 'negative' : 'positive' }
				: gc
		);
	}

	function updateQuantity(goalId: string, value: number | undefined) {
		goalConfigs = goalConfigs.map((gc) =>
			gc.goal_id === goalId ? { ...gc, default_quantity: value } : gc
		);
	}

	function getUnitSymbol(unitId: string | undefined): string {
		if (!unitId) return '';
		const unit = units.find((u) => u.id === unitId);
		return unit?.symbol || unitId.replace('units:', '');
	}

	// Create category mutation
	const createCategoryMut = createMutation({
		mutationFn: (data: { name: string; color: string }) =>
			createCategory({ name: data.name, color: data.color }),
		onSuccess: (newCat) => {
			queryClient.invalidateQueries({ queryKey: ['categories'] });
			categoryId = newCat.id;
			showCreateCategory = false;
			newCategoryName = '';
			newCategoryColor = '#6366f1';
			useCustomCategoryColor = false;
		},
	});

	async function handleCreateCategory() {
		if (!newCategoryName.trim()) return;
		$createCategoryMut.mutate({
			name: newCategoryName.trim(),
			color: newCategoryColor,
		});
	}

	function handleClose() {
		if (!isPending) {
			open = false;
			onClose();
		}
	}
</script>

<!-- Modal -->
<dialog class="modal modal-bottom sm:modal-middle" class:modal-open={open}>
	<div class="modal-box max-w-4xl p-0 flex flex-col bg-base-100 max-h-[90vh] rounded-b-none sm:rounded-3xl shadow-2xl">
		<!-- Header -->
		<div class="flex items-center gap-4 px-6 py-5 border-b border-base-200/50 bg-gradient-to-r from-primary/5 via-base-100 to-secondary/5">
			<!-- Icon Picker -->
			<div class="relative">
				<button
					type="button"
					class="w-14 h-14 rounded-2xl bg-gradient-to-br from-primary/20 via-primary/10 to-secondary/10 flex items-center justify-center text-3xl hover:from-primary/30 hover:to-secondary/20 transition-all shadow-lg border border-primary/10 hover:scale-105"
					onclick={() => (showEmojiPicker = !showEmojiPicker)}
				>
					{icon}
				</button>

				{#if showEmojiPicker}
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class="fixed inset-0 z-40"
						onclick={() => { showEmojiPicker = false; showFullPicker = false; }}
					></div>
					<div class="absolute top-16 left-0 z-50 bg-base-100 border border-base-300 rounded-2xl shadow-2xl overflow-hidden">
						{#if showFullPicker && emojiPickerReady}
							<div class="flex items-center justify-between px-3 pt-3 pb-1">
								<button type="button" class="btn btn-xs btn-ghost gap-1" onclick={() => (showFullPicker = false)}>
									← Back
								</button>
							</div>
							<emoji-picker class="light" onemoji-click={handleEmojiSelect}></emoji-picker>
						{:else}
							<div class="p-4 w-72 max-h-[350px] overflow-y-auto">
								<button
									type="button"
									class="btn btn-sm btn-ghost w-full mb-3 gap-2 text-base-content/60"
									onclick={() => (showFullPicker = true)}
								>
									🔍 Search all emojis...
								</button>
								<div class="flex flex-wrap gap-1 mb-3">
									{#each quickEmojis as emoji}
										<button
											type="button"
											class={cn(
												"w-8 h-8 rounded-lg text-lg hover:bg-base-200 transition-all flex items-center justify-center",
												icon === emoji && "bg-primary/20 ring-2 ring-primary"
											)}
											onclick={() => selectQuickEmoji(emoji)}
										>
											{emoji}
										</button>
									{/each}
								</div>
								{#each Object.entries(suggestedEmojis) as [cat, emojis]}
									<div class="mb-2">
										<p class="text-[10px] font-semibold text-base-content/40 uppercase mb-1">{cat}</p>
										<div class="flex flex-wrap gap-1">
											{#each emojis as emoji}
												<button
													type="button"
													class={cn(
														"w-7 h-7 rounded text-base hover:bg-base-200 transition-all flex items-center justify-center",
														icon === emoji && "bg-primary/20 ring-2 ring-primary"
													)}
													onclick={() => selectQuickEmoji(emoji)}
												>
													{emoji}
												</button>
											{/each}
										</div>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<div class="flex-1">
				<h3 class="text-xl font-bold text-base-content">
					{isEditing ? 'Edit Activity' : 'New Activity'}
				</h3>
				{#if isEditing && activity?.use_count}
					<p class="text-sm text-base-content/50">{activity.use_count} uses</p>
				{/if}
			</div>

			<button class="btn btn-ghost btn-sm btn-circle" onclick={handleClose}>
				<X class="w-5 h-5" />
			</button>
		</div>

		<!-- Error Alert -->
		{#if error}
			<div class="mx-6 mt-4 alert alert-error">
				<CircleAlert class="w-4 h-4" />
				<span>{error}</span>
				<button type="button" class="btn btn-ghost btn-xs btn-circle" onclick={() => (error = '')}>
					<X class="w-3 h-3" />
				</button>
			</div>
		{/if}

		<!-- Tabs -->
		<div class="flex border-b border-base-200/50 px-6 bg-base-100/50">
			<button
				class={cn(
					"px-5 py-3.5 text-sm font-medium border-b-2 -mb-px transition-all flex items-center gap-2",
					activeTab === 'details'
						? "border-primary text-primary"
						: "border-transparent text-base-content/60 hover:text-base-content"
				)}
				onclick={() => (activeTab = 'details')}
			>
				<Zap class="w-4 h-4" />
				Details
			</button>
			<button
				class={cn(
					"px-5 py-3.5 text-sm font-medium border-b-2 -mb-px transition-all flex items-center gap-2",
					activeTab === 'goals'
						? "border-primary text-primary"
						: "border-transparent text-base-content/60 hover:text-base-content"
				)}
				onclick={() => (activeTab = 'goals')}
			>
				<Target class="w-4 h-4" />
				Goals
				{#if goalConfigs.length > 0}
					<span class="badge badge-xs badge-primary">{goalConfigs.length}</span>
				{/if}
			</button>
		</div>

		<!-- Content -->
		<div class="flex-1 overflow-y-auto">
			{#if activeTab === 'details'}
				<div class="p-6 space-y-5">
					<!-- Title -->
					<div class="form-control">
						<label class="label py-0.5" for="activity-title">
							<span class="label-text font-semibold">Activity Name</span>
						</label>
						<input
							id="activity-title"
							type="text"
							bind:value={title}
							placeholder="e.g., Log Water, Morning Run..."
							class="input input-bordered w-full bg-base-200/30 focus:bg-base-100"
						/>
					</div>

					<!-- Description -->
					<div class="form-control">
						<label class="label py-0.5" for="activity-description">
							<span class="label-text text-xs text-base-content/60">Description (optional)</span>
						</label>
						<textarea
							id="activity-description"
							bind:value={description}
							placeholder="Brief description..."
							class="textarea textarea-bordered w-full h-16 resize-none bg-base-200/30 focus:bg-base-100 text-sm"
						></textarea>
					</div>

					<!-- Category (with inferred from goal) -->
					<div class="space-y-2">
						<div class="flex items-center gap-2">
							<Tag class="w-4 h-4 opacity-50" />
							<span class="text-xs font-semibold uppercase opacity-50">Category</span>
							{#if !categoryId && inferredCategory()}
								<span class="badge badge-xs badge-ghost">from goal</span>
							{/if}
						</div>
						{#if showCreateCategory}
							<div class="space-y-2">
								<input
									type="text"
									bind:value={newCategoryName}
									placeholder="Category name"
									class={cn("input input-sm input-bordered w-full", newCategoryName.length > 40 && "input-error")}
									maxlength={40}
								/>
								<div class="flex items-center justify-between">
									<span class="text-xs opacity-60">Color</span>
									<label class="flex items-center gap-2 cursor-pointer">
										<span class="text-xs opacity-60">Custom</span>
										<input type="checkbox" class="toggle toggle-xs toggle-primary" bind:checked={useCustomCategoryColor} />
									</label>
								</div>
								<ColorPicker bind:value={newCategoryColor} customMode={useCustomCategoryColor} size="sm" />
								<div class="flex gap-2">
									<button
										type="button"
										class="btn btn-sm btn-primary flex-1 gap-1"
										onclick={handleCreateCategory}
										disabled={$createCategoryMut.isPending || !newCategoryName.trim()}
									>
										{#if $createCategoryMut.isPending}
											<span class="loading loading-spinner loading-xs"></span>
										{:else}
											<Check class="w-3.5 h-3.5" />
										{/if}
										Create
									</button>
									<button type="button" class="btn btn-sm btn-ghost" onclick={() => (showCreateCategory = false)}>
										Cancel
									</button>
								</div>
							</div>
						{:else}
							<CategoryDropdown
								{categories}
								bind:value={categoryId}
								placeholder={inferredCategory() ? `${inferredCategory()?.name} (from goal)` : 'Select category...'}
								showCreateButton={true}
								onCreate={() => (showCreateCategory = true)}
							/>
						{/if}
					</div>

					<!-- Duration -->
					<div class="p-4 rounded-xl bg-base-200/50 border border-base-300 space-y-3">
						<label class="flex items-center justify-between cursor-pointer">
							<span class="flex items-center gap-2 font-medium">
								<Clock class="w-4 h-4 text-primary" />
								Default Duration
							</span>
							<input type="checkbox" class="toggle toggle-primary toggle-sm" bind:checked={durationEnabled} />
						</label>
						{#if durationEnabled}
							<div class="flex items-center gap-2">
								<input type="number" class="input input-bordered input-sm w-24" min="1" bind:value={defaultDuration} />
								<span class="text-sm opacity-60">minutes</span>
							</div>
						{/if}
					</div>

					<!-- Pinned & Completed Row -->
					<div class="grid grid-cols-2 gap-3">
						<div class="flex items-center gap-3 p-3 rounded-xl bg-base-200/50 border border-base-300">
							<Pin class="w-4 h-4 text-primary shrink-0" />
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium">Pin to Quick Bar</p>
							</div>
							<input type="checkbox" class="toggle toggle-primary toggle-sm" bind:checked={pinned} />
						</div>
						<div class="flex items-center gap-3 p-3 rounded-xl bg-base-200/50 border border-base-300">
							<Check class="w-4 h-4 text-success shrink-0" />
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium">Mark Completed</p>
							</div>
							<input type="checkbox" class="toggle toggle-success toggle-sm" bind:checked={defaultCompleted} />
						</div>
					</div>
				</div>
			{/if}

			{#if activeTab === 'goals'}
				<div class="flex h-[450px]">
					<!-- Available Goals List -->
					<div class="flex-1 flex flex-col border-r border-base-200">
						<div class="px-4 py-3 border-b border-base-200">
							<div class="relative">
								<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/40" />
								<input
									type="text"
									class="input input-bordered input-sm w-full pl-9"
									placeholder="Search goals..."
									bind:value={goalSearchQuery}
								/>
							</div>
						</div>
						<div class="flex-1 overflow-y-auto p-3">
							{#if $goalsQuery.isLoading}
								<div class="flex items-center justify-center h-full">
									<span class="loading loading-spinner loading-md text-primary"></span>
								</div>
							{:else if availableGoals.length === 0}
								<div class="flex flex-col items-center justify-center h-full text-center">
									<Target class="w-10 h-10 text-base-content/20 mb-2" />
									<p class="text-sm text-base-content/50">
										{goalSearchQuery ? 'No goals match' : 'All goals linked'}
									</p>
								</div>
							{:else}
								<div class="space-y-2">
									{#each availableGoals as goal (goal.id)}
										<button
											type="button"
											class="w-full flex items-center gap-3 p-3 rounded-xl bg-base-50 hover:bg-base-200/60 border border-transparent hover:border-base-300 transition-all group text-left"
											onclick={() => addGoal(goal)}
										>
											<div
												class="w-10 h-10 rounded-lg flex items-center justify-center text-xl shrink-0 group-hover:scale-110 transition-transform"
												style="background-color: {goal.category?.color || '#6b7280'}20;"
											>
												{goal.icon || '🎯'}
											</div>
											<div class="flex-1 min-w-0">
												<p class="font-medium text-sm truncate">{goal.title}</p>
												<div class="flex items-center gap-2 text-xs text-base-content/50">
													{#if goal.category}
														<span class="flex items-center gap-1">
															<span class="w-2 h-2 rounded-full" style="background-color: {goal.category.color}"></span>
															{goal.category.name}
														</span>
													{/if}
													{#if goal.target}
														<span>{goal.stats?.current_value || 0}/{goal.target.value} {getUnitSymbol(goal.target.unit_id)}</span>
													{/if}
													{#if (goal.stats?.current_streak || 0) > 0}
														<span class="flex items-center gap-0.5 text-warning font-semibold">
															<Flame class="w-3 h-3" />
															{goal.stats?.current_streak}
														</span>
													{/if}
												</div>
											</div>
											<ChevronRight class="w-4 h-4 text-base-content/30 group-hover:text-primary transition-colors" />
										</button>
									{/each}
								</div>
							{/if}
						</div>
					</div>

					<!-- Linked Goals Panel -->
					<div class="w-80 shrink-0 flex flex-col bg-base-200/30">
						<div class="px-4 py-3 border-b border-base-200">
							<h3 class="font-semibold">Linked Goals</h3>
							<p class="text-xs text-base-content/50">
								{linkedGoals.length} goal{linkedGoals.length !== 1 ? 's' : ''} selected
							</p>
						</div>

						<div class="flex-1 overflow-y-auto p-3">
							{#if linkedGoals.length === 0}
								<div class="flex flex-col items-center justify-center h-full text-center px-4">
									<p class="text-sm text-base-content/40">Click goals to add them</p>
								</div>
							{:else}
								<div class="space-y-3">
									{#each linkedGoals as linked (linked.goal_id)}
										{@const goal = linked.goal}
										{@const isPositive = linked.impact_type === 'positive'}
										{#if goal}
											<div class="p-3 rounded-xl bg-base-100 border border-base-200 shadow-sm" transition:scale={{ duration: 150 }}>
												<div class="flex items-start gap-2 mb-3">
													<span class="text-lg">{goal.icon || '🎯'}</span>
													<div class="flex-1 min-w-0">
														<p class="font-medium text-sm truncate">{goal.title}</p>
														{#if goal.target}
															<p class="text-xs text-base-content/50">
																{goal.stats?.current_value || 0}/{goal.target.value} {getUnitSymbol(goal.target.unit_id)}
															</p>
														{/if}
													</div>
													<button
														class="p-1 rounded hover:bg-base-200 text-base-content/40 hover:text-error"
														onclick={() => removeGoal(linked.goal_id)}
													>
														<X class="w-4 h-4" />
													</button>
												</div>

												<!-- Impact Toggle -->
												<div class="flex gap-2 mb-3">
													<button
														class={cn(
															"flex-1 flex items-center justify-center gap-1.5 py-1.5 rounded-lg text-xs font-medium transition-all",
															isPositive
																? "bg-success/20 text-success border border-success/30"
																: "bg-base-200 text-base-content/50 hover:bg-base-300"
														)}
														onclick={() => !isPositive && toggleImpact(linked.goal_id)}
													>
														<TrendingUp class="w-3.5 h-3.5" />
														Positive
													</button>
													<button
														class={cn(
															"flex-1 flex items-center justify-center gap-1.5 py-1.5 rounded-lg text-xs font-medium transition-all",
															!isPositive
																? "bg-error/20 text-error border border-error/30"
																: "bg-base-200 text-base-content/50 hover:bg-base-300"
														)}
														onclick={() => isPositive && toggleImpact(linked.goal_id)}
													>
														<TrendingDown class="w-3.5 h-3.5" />
														Negative
													</button>
												</div>

												<!-- Quantity -->
												{#if goal.target}
													<div class="flex items-center gap-2">
														<input
															type="number"
															step="0.1"
															min="0"
															class="input input-bordered input-xs flex-1"
															placeholder="Qty"
															value={linked.default_quantity ?? ''}
															onchange={(e) => {
																const val = parseFloat(e.currentTarget.value);
																updateQuantity(linked.goal_id, isNaN(val) ? undefined : val);
															}}
														/>
														<span class="text-xs font-medium text-base-content/60 min-w-[35px]">
															{getUnitSymbol(goal.target.unit_id)}
														</span>
													</div>
												{/if}
											</div>
										{/if}
									{/each}
								</div>
							{/if}
						</div>
					</div>
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="flex items-center justify-end gap-3 px-6 py-4 border-t border-base-200/50 bg-base-100/80">
			<button class="btn btn-ghost btn-sm" onclick={handleClose} disabled={isPending}>
				Cancel
			</button>
			<button
				class="btn btn-primary btn-sm gap-2 min-w-[120px]"
				onclick={handleSubmit}
				disabled={isPending || !title.trim()}
			>
				{#if isPending}
					<span class="loading loading-spinner loading-xs"></span>
					{isEditing ? 'Saving...' : 'Creating...'}
				{:else}
					<Save class="w-4 h-4" />
					{isEditing ? 'Save' : 'Create'}
				{/if}
			</button>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop bg-base-content/20">
		<button onclick={handleClose}>close</button>
	</form>
</dialog>
