<script lang="ts">
	import { browser } from "$app/environment";
	import { goto } from "$app/navigation";
	import {
		type CreateTaskRequest,
		type Goal,
		type Task,
		type TaskItem,
		type UpdateTaskRequest,
		createTask,
		deleteTask,
		getLastTaskEndTime,
		getTaskGoals,
		updateTask,
	} from "$lib/api";
	import { mapGoalPriorityToTask } from "$lib/utils/priority";
	import { createCategory, getCategories } from "$lib/api/categories";
	import {
		type Emotion,
		type InferredEmotion,
		inferEmotion,
	} from "$lib/api/emotions";
	import { EmotionModal } from "$lib/components/emotions";
	import {
		QUADRANT_COLORS,
		QUADRANT_META,
		type Quadrant,
	} from "$lib/components/emotions/emotionData";
	import { GoalSelector } from "$lib/components/goals";
	import { RichEditor } from "$lib/components/rich-editor";
	import {
		EmotionSection,
		ReflectionSection,
		ScheduleSection,
		TaskPrioritySlider,
	} from "$lib/components/tasks";
	import {
		Card,
		CategoryDropdown,
		ColorPicker,
		ConfirmDialog,
		SectionHeader,
	} from "$lib/components/ui";
	import OpenMoji from "$lib/components/ui/OpenMoji.svelte";
	import { emotionStore } from "$lib/stores/emotions.svelte";
	import { cn } from "$lib/utils";
	import {
		createMutation,
		createQuery,
		useQueryClient,
	} from "@tanstack/svelte-query";

	import {
		Check,
		Circle,
		CircleCheck,
		Edit3,
		FileText,
		Info,
		Save,
		Tag,
		Target,
		Trash2,
		X,
	} from "lucide-svelte";

	import { onDestroy } from "svelte";
	import { writable } from "svelte/store";

	// Context for emotion selection
	type EmotionSelectionContext = {
		type: "task" | "positive" | "negative";
		index?: number;
	};

	interface Props {
		/** Task to edit (null for create mode) */
		task?: Task | null;
		/** Initial category ID for new tasks */
		initialCategoryId?: string;
		/** Referrer URL to navigate back to */
		referrer?: string;
		/** Called after successful save */
		onSuccess?: () => void;
	}

	let {
		task = null,
		initialCategoryId,
		referrer,
		onSuccess,
	}: Props = $props();

	const queryClient = useQueryClient();
	const isEditing = $derived(!!task);

	// Fetch last task end time only in create mode
	// Fetch last task end time only in create mode
	const lastTaskEndTimeOptions = writable({
		queryKey: ["tasks", "last-end-time"],
		queryFn: getLastTaskEndTime,
		enabled: false,
	});

	$effect(() => {
		lastTaskEndTimeOptions.set({
			queryKey: ["tasks", "last-end-time"],
			queryFn: getLastTaskEndTime,
			enabled: !isEditing,
		});
	});

	const lastTaskEndTimeQuery = createQuery<{ end_time: string | null }>(
		lastTaskEndTimeOptions,
	);

	const lastTaskEndTime = $derived(
		$lastTaskEndTimeQuery.data?.end_time
			? new Date($lastTaskEndTimeQuery.data.end_time)
			: null,
	);

	// Fetch task goals when editing
	const taskId = $derived(task?.id);
	const taskGoalsOptions = writable({
		queryKey: ["task-goals", null as string | null | undefined],
		queryFn: () => Promise.resolve({ task_id: "", goals: [] as any[] }),
		enabled: false,
	});

	$effect(() => {
		taskGoalsOptions.set({
			queryKey: ["task-goals", taskId],
			queryFn: () =>
				taskId
					? getTaskGoals(taskId)
					: Promise.resolve({ task_id: "", goals: [] }),
			enabled: !!taskId,
		});
	});

	const taskGoalsQuery = createQuery(taskGoalsOptions);

	// Form state
	let title = $state("");
	let journal = $state("");
	let categoryId = $state<string | undefined>(undefined);
	let priority = $state(3);
	let positives = $state<TaskItem[]>([]);
	let negatives = $state<TaskItem[]>([]);
	let completed = $state(false);

	// Goal links state
	interface GoalLink {
		goal_id: string;
		impact_type: "positive" | "negative";
		quantity_value?: number;
		quantity_unit?: string;
	}
	let goalLinks = $state<GoalLink[]>([]);

	// Emotion state
	let selectedEmotion = $state<Emotion | null>(null);
	let emotionContext = $state<EmotionSelectionContext | null>(null);
	let emotionModalOpen = $state(false);
	let liveInferredEmotion = $state<InferredEmotion | null>(null);
	let inferredEmotionFull = $state<Emotion | null>(null);
	let inferringEmotion = $state(false);
	let inferenceError = $state<string | null>(null);
	/** Initial emotion to pre-select when modal opens (e.g. suggested emotion) */
	let initialEmotionForModal = $state<Emotion | null>(null);
	/** Allowed quadrants for emotion selection (null = all allowed) */
	let allowedQuadrantsForModal = $state<Quadrant[] | null>(null);

	// Pending emotion for new reflection item (select emotion FIRST)
	let pendingPositiveEmotion = $state<Emotion | null>(null);
	let pendingNegativeEmotion = $state<Emotion | null>(null);
	let pendingEmotionType = $state<"positive" | "negative" | null>(null);

	// Date/time state
	let startDate = $state("");
	let endDate = $state("");
	let startTime = $state("09:00:00");
	let endTime = $state("10:00:00");

	// Category creation
	let newCategoryName = $state("");
	let newCategoryColor = $state("#6366f1");
	let showNewCategory = $state(false);
	let useCustomCategoryColor = $state(false);

	// Confirmation dialogs
	let showUncompleteConfirm = $state(false);
	let showDeleteConfirm = $state(false);

	// Live time tracking (only for create mode)
	let liveEndTime = $state(false);
	let useLastTaskStart = $state(false);
	let timeUpdateInterval: ReturnType<typeof setInterval> | null = null;

	// Track if form is initialized
	let formInitialized = $state(false);

	// Category & Priority inheritance tracking
	let categoryInheritedFrom = $state<{ id: string; title: string } | null>(
		null,
	);
	let priorityInheritedFrom = $state<{ id: string; title: string } | null>(
		null,
	);
	let categoryManuallySet = $state(false);
	let priorityManuallySet = $state(false);

	/**
	 * Handle goals being added - apply category/priority inheritance
	 * Per design: First goal sets category, highest priority wins
	 */
	function handleGoalsAdded(addedGoals: Goal[]) {
		if (addedGoals.length === 0) return;

		// First goal sets category (if not manually set and no category yet)
		if (!categoryManuallySet && !categoryId && addedGoals[0]?.category) {
			categoryId = addedGoals[0].category.id;
			categoryInheritedFrom = {
				id: addedGoals[0].id,
				title: addedGoals[0].title,
			};
		}

		// Priority: highest wins (lowest number = highest priority)
		if (!priorityManuallySet) {
			// Find the goal with highest priority among newly added goals
			const goalsWithPriority = addedGoals.filter(
				(g) => g.priority !== undefined && g.priority !== null,
			);

			if (goalsWithPriority.length > 0) {
				const highestGoal = goalsWithPriority.reduce((a, b) =>
					a.priority < b.priority ? a : b,
				);

				const mappedPriority = mapGoalPriorityToTask(highestGoal.priority);
				priority = mappedPriority;
				priorityInheritedFrom = {
					id: highestGoal.id,
					title: highestGoal.title,
				};
			}
		}
	}

	/** Mark category as manually set and clear inheritance indicator */
	function handleCategoryManualChange(newCategoryId: string | undefined) {
		categoryId = newCategoryId;
		categoryManuallySet = true;
		categoryInheritedFrom = null;
	}

	/** Mark priority as manually set and clear inheritance indicator */
	function handlePriorityManualChange(newPriority: number) {
		priority = newPriority;
		priorityManuallySet = true;
		priorityInheritedFrom = null;
	}

	function getTodayString() {
		return new Date().toISOString().split("T")[0];
	}

	// Hash of reflections to check if they have changed
	function getReflectionsHash(p: TaskItem[], n: TaskItem[]) {
		return JSON.stringify({
			p: p.map((x) => ({ t: x.text, e: x.emotion_id })),
			n: n.map((x) => ({ t: x.text, e: x.emotion_id })),
		});
	}

	// Logic to prevent initial inference call
	let lastInferredHash = $state("");

	function updateEndTimeToNow() {
		const now = new Date();
		endDate = now.toISOString().split("T")[0];
		endTime = now.toTimeString().slice(0, 8);
	}

	// Initialize form based on mode
	$effect(() => {
		if (formInitialized) return;

		if (task) {
			// Edit mode - populate from task
			title = task.title;
			journal = task.journal || "";
			categoryId = task.category?.id;
			priority = task.priority || 3;
			positives = task.positives || [];
			negatives = task.negatives || [];
			completed = task.completed || false;
			// Goal links are populated via separate effect from taskGoalsQuery

			const startDateObj = new Date(task.start_date);
			const endDateObj = new Date(task.end_date);

			startDate = startDateObj.toISOString().split("T")[0];
			endDate = endDateObj.toISOString().split("T")[0];
			startTime = startDateObj.toTimeString().slice(0, 8);
			endTime = endDateObj.toTimeString().slice(0, 8);

			if (task.emotion_id) {
				// Try to load immediately from store
				const e = emotionStore.get(task.emotion_id);
				if (e) {
					selectedEmotion = e;
				}
			}

			// Set initial hash to prevent immediate inference
			lastInferredHash = getReflectionsHash(positives, negatives);

			formInitialized = true;
		} else {
			// Create mode - set defaults
			const today = getTodayString();
			startDate = today;
			endDate = today;
			if (initialCategoryId) {
				categoryId = initialCategoryId;
			}

			// Initialize hash for empty arrays in create mode
			lastInferredHash = getReflectionsHash([], []);
		}
		formInitialized = true;
		console.log("[TaskForm Init] Initialized:", {
			isEditing,
			lastInferredHash,
			formInitialized,
		});
	});

	// Populate goal links when query data arrives (edit mode)
	let goalLinksInitialized = $state(false);
	$effect(() => {
		if (isEditing && $taskGoalsQuery.data?.goals && !goalLinksInitialized) {
			goalLinks = $taskGoalsQuery.data.goals.map((l) => ({
				goal_id: l.goal_id,
				impact_type:
					l.impact_type === "negative" ? "negative" : "positive",
				quantity_value: l.quantity_value,
				quantity_unit: l.quantity_unit,
			}));
			goalLinksInitialized = true;
		}
	});

	// Live time interval for create mode
	$effect(() => {
		if (!isEditing && liveEndTime) {
			updateEndTimeToNow();
			timeUpdateInterval = setInterval(updateEndTimeToNow, 1000);
		} else if (timeUpdateInterval) {
			clearInterval(timeUpdateInterval);
			timeUpdateInterval = null;
		}
	});

	// Effect to set start from last task
	$effect(() => {
		if (!isEditing && useLastTaskStart && lastTaskEndTime) {
			const lastEnd = new Date(lastTaskEndTime);
			startDate = lastEnd.toISOString().split("T")[0];
			startTime = lastEnd.toTimeString().slice(0, 8);
		}
	});

	// Effect to calculate live inferred emotion
	$effect(() => {
		const currentHash = getReflectionsHash(positives, negatives);
		const hasEmotions =
			positives.some((p) => p.emotion_id) ||
			negatives.some((n) => n.emotion_id);

		console.log("[Emotion Inference] Effect triggered:", {
			hasEmotions,
			currentHash,
			lastInferredHash,
			positivesCount: positives.length,
			negativesCount: negatives.length,
			positivesWithEmotions: positives.filter((p) => p.emotion_id).length,
			negativesWithEmotions: negatives.filter((n) => n.emotion_id).length,
		});

		if (!hasEmotions) {
			console.log("[Emotion Inference] No emotions, clearing inference");
			liveInferredEmotion = null;
			inferredEmotionFull = null;
			inferringEmotion = false;
			inferenceError = null;
			return;
		}

		if (currentHash !== lastInferredHash) {
			console.log("[Emotion Inference] Starting inference...");
			inferringEmotion = true;
			inferenceError = null;

			inferEmotion({ positives, negatives })
				.then((response) => {
					console.log("[Emotion Inference] Success:", response);
					lastInferredHash = currentHash;
					liveInferredEmotion = response.inferred_emotion;
					inferringEmotion = false;

					// Get full emotion data from store
					if (response.inferred_emotion?.closest_emotion_id) {
						const e = emotionStore.get(
							response.inferred_emotion.closest_emotion_id,
						);
						console.log(
							"[Emotion Inference] Found emotion in store:",
							e?.name,
						);
						inferredEmotionFull = e || null;
					} else {
						console.log(
							"[Emotion Inference] No closest emotion ID in response",
						);
						inferredEmotionFull = null;
					}
				})
				.catch((error) => {
					console.error("[Emotion Inference] Error:", error);
					inferringEmotion = false;
					inferenceError = error.message || "Failed to infer emotion";
					liveInferredEmotion = null;
					inferredEmotionFull = null;
				});
		} else {
			console.log(
				"[Emotion Inference] Hash unchanged, skipping inference",
			);
		}
	});

	// Reactively update selectedEmotion and inferredEmotionFull when store initializes
	$effect(() => {
		if (emotionStore.isInitialized) {
			if (task?.emotion_id && !selectedEmotion) {
				const e = emotionStore.get(task.emotion_id);
				if (e) selectedEmotion = e;
				// Note: this might override if user cleared it?
				// Actually formInitialized prevents re-running the main init block.
				// This block is specifically for when store loads AFTER task load.
				// But simply checking !selectedEmotion checks if it's empty.
				// If user actively cleared it, it is null.
				// Ideally we track if we have "attempted" to load the initial emotion.
			}
			// Update inferred emotion full if needed (e.g. store loaded late)
			if (
				liveInferredEmotion?.closest_emotion_id &&
				!inferredEmotionFull
			) {
				const e = emotionStore.get(
					liveInferredEmotion.closest_emotion_id,
				);
				if (e) inferredEmotionFull = e;
			}
			// Also need to handle task.inferred_emotion if not editing live
			if (
				task?.inferred_emotion?.closest_emotion_id &&
				!inferredEmotionFull &&
				!liveInferredEmotion
			) {
				const e = emotionStore.get(
					task.inferred_emotion.closest_emotion_id,
				);
				if (e) inferredEmotionFull = e;
			}
		}
	});

	onDestroy(() => {
		if (timeUpdateInterval) clearInterval(timeUpdateInterval);
	});

	// Queries
	const categoriesQuery = createQuery({
		queryKey: ["categories"],
		queryFn: () => getCategories({ limit: 100 }),
		retry: false,
	});

	const categories = $derived($categoriesQuery.data?.items || []);
	const selectedCategory = $derived(
		categories.find((c) => c.id === categoryId),
	);

	// Mutations
	const createMut = createMutation({
		mutationFn: (data: CreateTaskRequest) => createTask(data),
		onSuccess: handleSuccess,
	});

	const updateMut = createMutation({
		mutationFn: (data: UpdateTaskRequest) => updateTask(task!.id, data),
		onSuccess: handleSuccess,
	});

	const deleteMut = createMutation({
		mutationFn: () => deleteTask(task!.id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["tasks"] });
			navigateBack();
		},
	});

	const createCategoryMut = createMutation({
		mutationFn: (data: { name: string; color: string }) =>
			createCategory(data),
		onSuccess: (newCat) => {
			queryClient.invalidateQueries({ queryKey: ["categories"] });
			categoryId = newCat.id;
			newCategoryName = "";
			newCategoryColor = "#6366f1";
			useCustomCategoryColor = false;
			showNewCategory = false;
		},
	});

	const isPending = $derived($createMut.isPending || $updateMut.isPending);
	const inferredEmotion = $derived(
		liveInferredEmotion || task?.inferred_emotion,
	);

	function handleSuccess() {
		queryClient.invalidateQueries({ queryKey: ["tasks"] });
		onSuccess?.();
		navigateBack();
	}

	function navigateBack() {
		if (referrer) {
			goto(referrer);
		} else {
			goto("/tasks");
		}
	}

	// loadTaskEmotion removed as it's handled in effects/init

	// Emotion handlers
	function handleEmotionSelect(emotion: Emotion) {
		console.log("[Emotion Select]", {
			emotion: emotion.name,
			context: emotionContext,
		});

		if (emotionContext?.type === "task") {
			selectedEmotion = emotion;
		} else if (
			emotionContext?.type === "positive" &&
			emotionContext.index !== undefined
		) {
			if (emotionContext.index === -1) {
				// Pending new positive item
				pendingPositiveEmotion = emotion;
				console.log(
					"[Emotion Select] Set pending positive emotion:",
					emotion.name,
				);
			} else {
				positives = positives.map((p, i) =>
					i === emotionContext!.index
						? { ...p, emotion_id: emotion.id }
						: p,
				);
				console.log(
					"[Emotion Select] Updated positive item at index",
					emotionContext.index,
				);
			}
		} else if (
			emotionContext?.type === "negative" &&
			emotionContext.index !== undefined
		) {
			if (emotionContext.index === -1) {
				// Pending new negative item
				pendingNegativeEmotion = emotion;
				console.log(
					"[Emotion Select] Set pending negative emotion:",
					emotion.name,
				);
			} else {
				negatives = negatives.map((n, i) =>
					i === emotionContext!.index
						? { ...n, emotion_id: emotion.id }
						: n,
				);
				console.log(
					"[Emotion Select] Updated negative item at index",
					emotionContext.index,
				);
			}
		}
		emotionContext = null;
		emotionModalOpen = false;
		pendingEmotionType = null;
	}

	function openEmotionForTask() {
		emotionContext = { type: "task" };
		initialEmotionForModal = null;
		allowedQuadrantsForModal = null; // All quadrants allowed for task emotion
		emotionModalOpen = true;
	}

	/** Open emotion modal with the suggested emotion pre-selected and panned to */
	function openEmotionForSuggested() {
		emotionContext = { type: "task" };
		initialEmotionForModal = inferredEmotionFull;
		allowedQuadrantsForModal = null; // All quadrants allowed for task emotion
		emotionModalOpen = true;
	}

	function openEmotionForPositive(index: number) {
		emotionContext = { type: "positive", index };

		// Find current emotion if it exists
		const p = positives[index];
		initialEmotionForModal = p.emotion_id
			? emotionStore.get(p.emotion_id) || null
			: null;

		allowedQuadrantsForModal = ["yellow", "green"]; // Only pleasant emotions for positives
		emotionModalOpen = true;
	}

	function openEmotionForNegative(index: number) {
		emotionContext = { type: "negative", index };

		// Find current emotion if it exists
		const n = negatives[index];
		initialEmotionForModal = n.emotion_id
			? emotionStore.get(n.emotion_id) || null
			: null;

		allowedQuadrantsForModal = ["red", "blue"]; // Only unpleasant emotions for negatives
		emotionModalOpen = true;
	}

	// Open emotion picker for pending new item
	function openEmotionForPendingPositive() {
		pendingEmotionType = "positive";
		emotionContext = { type: "positive", index: -1 };
		initialEmotionForModal = pendingPositiveEmotion;
		allowedQuadrantsForModal = ["yellow", "green"]; // Only pleasant emotions for positives
		emotionModalOpen = true;
	}

	function openEmotionForPendingNegative() {
		pendingEmotionType = "negative";
		emotionContext = { type: "negative", index: -1 };
		initialEmotionForModal = pendingNegativeEmotion;
		allowedQuadrantsForModal = ["red", "blue"]; // Only unpleasant emotions for negatives
		emotionModalOpen = true;
	}

	function clearTaskEmotion() {
		selectedEmotion = null;
	}

	function clearPositiveEmotion(index: number) {
		positives = positives.map((p, i) =>
			i === index ? { ...p, emotion_id: undefined } : p,
		);
	}

	function clearNegativeEmotion(index: number) {
		negatives = negatives.map((n, i) =>
			i === index ? { ...n, emotion_id: undefined } : n,
		);
	}

	// Completion toggle
	function handleCompletionToggle() {
		if (completed && isEditing) {
			showUncompleteConfirm = true;
		} else {
			completed = !completed;
		}
	}

	function confirmUncomplete() {
		completed = false;
		showUncompleteConfirm = false;
	}

	// Category
	function handleCreateCategory() {
		if (newCategoryName.trim()) {
			$createCategoryMut.mutate({
				name: newCategoryName.trim(),
				color: newCategoryColor,
			});
		}
	}

	// Submit
	function handleSubmit() {
		if (!title.trim() || !startDate || !endDate) return;

		const [startYear, startMonth, startDay] = startDate
			.split("-")
			.map(Number);
		const [startHour, startMinute, startSecond] = startTime
			.split(":")
			.map(Number);
		const [endYear, endMonth, endDay] = endDate.split("-").map(Number);
		const [endHour, endMinute, endSecond] = endTime.split(":").map(Number);

		const startDateObj = new Date(
			startYear,
			startMonth - 1,
			startDay,
			startHour || 0,
			startMinute || 0,
			startSecond || 0,
		);
		const endDateObj = new Date(
			endYear,
			endMonth - 1,
			endDay,
			endHour || 0,
			endMinute || 0,
			endSecond || 0,
		);

		const data = {
			title: title.trim(),
			journal: journal || undefined,
			start_date: startDateObj.toISOString(),
			end_date: endDateObj.toISOString(),
			category_id: categoryId || undefined,
			priority: priority || undefined,
			positives: positives.length > 0 ? positives : undefined,
			negatives: negatives.length > 0 ? negatives : undefined,
			completed: completed,
			emotion_id: selectedEmotion?.id || undefined,
			goal_links:
				goalLinks.length > 0
					? goalLinks.map((l) => ({
							goal_id: l.goal_id,
							impact_type: l.impact_type,
							quantity_value: l.quantity_value,
							quantity_unit: l.quantity_unit,
						}))
					: undefined,
		};

		if (isEditing) {
			$updateMut.mutate(data);
		} else {
			$createMut.mutate(data);
		}
	}

	function handleDelete() {
		$deleteMut.mutate();
		showDeleteConfirm = false;
	}
</script>

<div class="flex flex-col min-h-[calc(100vh-6rem)] relative">
	<div class="space-y-6 flex-1 pb-4">
		<!-- Main Content Grid -->
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
			<!-- Left Column: Title + Journal (2/3 width on desktop) -->
			<div class="lg:col-span-2 space-y-4">
				<!-- Title Card -->
				<Card variant="bordered">
					<div class="flex items-center gap-3">
						<!-- Completion Toggle -->
						<button
							class={cn(
								"w-10 h-10 rounded-xl flex items-center justify-center transition-all duration-200 cursor-pointer shrink-0 hover:scale-105",
								completed
									? "bg-success text-success-content hover:bg-success/80 shadow-sm"
									: "bg-base-200 hover:bg-primary/10 hover:text-primary",
							)}
							onclick={handleCompletionToggle}
							title={completed
								? "Mark as incomplete"
								: "Mark as complete"}
						>
							{#if completed}
								<CircleCheck
									class="w-5 h-5 transition-transform duration-200"
								/>
							{:else}
								<Circle
									class="w-5 h-5 transition-transform duration-200"
								/>
							{/if}
						</button>

						<!-- Title Input -->
						<input
							type="text"
							bind:value={title}
							placeholder="What needs to be done?"
							class="input input-ghost text-xl md:text-2xl font-bold flex-1 px-0 focus:outline-none focus:bg-transparent placeholder:text-base-content/20 transition-all duration-200"
						/>

						<!-- Category Badge -->
						{#if selectedCategory}
							<span
								class="badge badge-lg shrink-0 transition-all duration-200 animate-fade-in"
								style="background-color: {selectedCategory.color}; color: white; border: none;"
							>
								{selectedCategory.name}
							</span>
						{/if}

						<!-- Completed Badge -->
						{#if completed}
							<span
								class="badge badge-success gap-1 transition-all duration-200 animate-fade-in"
							>
								<Check class="w-3 h-3" />
								Done
							</span>
						{/if}
					</div>
				</Card>

				<!-- Journal Card - Expanded -->
				<Card
					variant="bordered"
					class="min-h-[500px] flex flex-col transition-all duration-200 hover:shadow-md"
				>
					<div class="flex items-center gap-2 mb-3">
						<FileText class="w-4 h-4 opacity-50" />
						<span class="text-xs font-semibold uppercase opacity-50"
							>Notes & Details</span
						>
					</div>
					<div class="flex-1">
						<RichEditor
							bind:value={journal}
							placeholder="Add details, notes, or reflections..."
							minHeight="420px"
							showToolbar={true}
						/>
					</div>
				</Card>

				<!-- Reflection Section -->
				<ReflectionSection
					{positives}
					{negatives}
					{pendingPositiveEmotion}
					{pendingNegativeEmotion}
					onPositivesChange={(items) => (positives = items)}
					onNegativesChange={(items) => (negatives = items)}
					onOpenEmotionForPositive={openEmotionForPositive}
					onOpenEmotionForNegative={openEmotionForNegative}
					onOpenEmotionForPendingPositive={openEmotionForPendingPositive}
					onOpenEmotionForPendingNegative={openEmotionForPendingNegative}
					onClearPendingPositiveEmotion={() =>
						(pendingPositiveEmotion = null)}
					onClearPendingNegativeEmotion={() =>
						(pendingNegativeEmotion = null)}
				/>
			</div>

			<!-- Right Column: Meta Fields (1/3 width on desktop) -->
			<div class="space-y-4">
				<!-- Category -->
				<Card
					variant="bordered"
					class="transition-all duration-200 hover:shadow-md"
				>
					<div class="flex items-center gap-2 mb-3">
						<Tag class="w-4 h-4 opacity-50" />
						<span class="text-xs font-semibold uppercase opacity-50"
							>Category</span
						>
						{#if categoryInheritedFrom}
							<span class="badge badge-xs badge-ghost gap-1">
								<Target class="w-2.5 h-2.5" />
								from {categoryInheritedFrom.title}
							</span>
						{/if}
					</div>
					{#if showNewCategory}
						<div class="space-y-3">
							<input
								type="text"
								bind:value={newCategoryName}
								placeholder="Category name"
								class={cn(
									"input input-sm input-bordered w-full transition-all duration-200",
									newCategoryName.length > 40 &&
										"input-error",
								)}
								maxlength={40}
							/>
							<div class="flex items-center justify-between">
								<span class="text-xs opacity-60">Color</span>
								<label
									class="flex items-center gap-2 cursor-pointer"
								>
									<span class="text-xs opacity-60"
										>Custom</span
									>
									<input
										type="checkbox"
										class="toggle toggle-xs toggle-primary"
										bind:checked={useCustomCategoryColor}
									/>
								</label>
							</div>
							<ColorPicker
								bind:value={newCategoryColor}
								customMode={useCustomCategoryColor}
								size="sm"
							/>
							<div class="flex gap-2">
								<button
									class="btn btn-sm btn-primary flex-1 gap-1 transition-all duration-200 hover:shadow-md"
									onclick={handleCreateCategory}
									disabled={$createCategoryMut.isPending ||
										!newCategoryName.trim()}
								>
									{#if $createCategoryMut.isPending}
										<span
											class="loading loading-spinner loading-xs"
										></span>
									{:else}
										<Check class="w-3.5 h-3.5" />
									{/if}
									Create
								</button>
								<button
									class="btn btn-sm btn-ghost transition-all duration-200 hover:bg-base-200"
									onclick={() => (showNewCategory = false)}
									>Cancel</button
								>
							</div>
						</div>
					{:else}
						<CategoryDropdown
							{categories}
							bind:value={categoryId}
							onchange={handleCategoryManualChange}
							placeholder="Select category..."
							showCreateButton={true}
							onCreate={() => (showNewCategory = true)}
						/>
					{/if}
				</Card>

				<!-- Emotion Section -->
				<EmotionSection
					{selectedEmotion}
					inferredEmotion={liveInferredEmotion}
					{inferredEmotionFull}
					{inferringEmotion}
					{inferenceError}
					onOpenEmotionForTask={openEmotionForTask}
					onOpenEmotionForSuggested={openEmotionForSuggested}
					onClearTaskEmotion={clearTaskEmotion}
				/>

				<!-- Schedule -->
				<ScheduleSection
					bind:startDate
					bind:endDate
					bind:startTime
					bind:endTime
					bind:liveEndTime
					bind:useLastTaskStart
					{isEditing}
					{lastTaskEndTime}
					onStartDateChange={(v) => (startDate = v)}
					onEndDateChange={(v) => (endDate = v)}
					onStartTimeChange={(v) => (startTime = v)}
					onEndTimeChange={(v) => (endTime = v)}
					onLiveEndTimeChange={(v) => (liveEndTime = v)}
					onUseLastTaskStartChange={(v) => (useLastTaskStart = v)}
				/>
				<!-- Goals Section -->
				<Card
					variant="bordered"
					class="transition-all duration-200 hover:shadow-md"
				>
					<GoalSelector
						value={goalLinks}
						onChange={(links) => (goalLinks = links)}
						onGoalsAdded={handleGoalsAdded}
					/>
				</Card>

				<!-- Priority -->
				<TaskPrioritySlider
					bind:value={priority}
					onChange={handlePriorityManualChange}
					inheritedFrom={priorityInheritedFrom?.title}
				/>
			</div>
		</div>
	</div>

	<!-- Sticky Footer Actions -->
	<div
		class="sticky bottom-0 -mx-4 md:-mx-6 lg:-mx-8 px-4 md:px-6 lg:px-8 py-3 mt-auto bg-base-100/95 backdrop-blur-md border-t border-base-300 shadow-lg z-30"
	>
		<div class="flex items-center justify-between gap-4">
			<div class="flex items-center gap-3">
				{#if isEditing}
					<button
						class="btn btn-ghost btn-sm text-error hover:bg-error/10 gap-2 transition-all duration-200"
						onclick={() => (showDeleteConfirm = true)}
						disabled={$deleteMut.isPending}
					>
						<Trash2 class="w-4 h-4" />
						<span class="hidden sm:inline">Delete</span>
					</button>
				{/if}
				<span class="text-xs text-base-content/40 hidden lg:inline">
					<kbd class="kbd kbd-xs">⌘</kbd> +
					<kbd class="kbd kbd-xs">Enter</kbd> to save
				</span>
			</div>
			<div class="flex items-center gap-2">
				<button
					class="btn btn-ghost btn-sm hover:bg-base-200 transition-all duration-200"
					onclick={navigateBack}
					disabled={isPending}
				>
					Cancel
				</button>
				<button
					class="btn btn-primary btn-sm gap-2 min-w-[120px] shadow-md hover:shadow-lg shadow-primary/20 transition-all duration-200"
					onclick={handleSubmit}
					disabled={isPending || !title.trim()}
				>
					{#if isPending}
						<span class="loading loading-spinner loading-xs"></span>
						{isEditing ? "Saving..." : "Creating..."}
					{:else}
						<Save class="w-4 h-4" />
						{isEditing ? "Save Changes" : "Create Task"}
					{/if}
				</button>
			</div>
		</div>
	</div>
</div>

<!-- Emotion Modal -->
<EmotionModal
	bind:open={emotionModalOpen}
	initialEmotion={initialEmotionForModal}
	allowedQuadrants={allowedQuadrantsForModal}
	onSelect={handleEmotionSelect}
	onClose={() => {
		emotionModalOpen = false;
		emotionContext = null;
		initialEmotionForModal = null;
		allowedQuadrantsForModal = null;
	}}
/>

<!-- Uncomplete Confirmation -->
<ConfirmDialog
	bind:open={showUncompleteConfirm}
	title="Mark as Incomplete?"
	message="Are you sure you want to mark this task as incomplete?"
	confirmText="Mark Incomplete"
	cancelText="Keep Complete"
	destructive={false}
	onConfirm={confirmUncomplete}
	onCancel={() => (showUncompleteConfirm = false)}
/>

<!-- Delete Confirmation -->
{#if isEditing}
	<ConfirmDialog
		bind:open={showDeleteConfirm}
		title="Delete Task?"
		message="This action cannot be undone."
		confirmText="Delete"
		destructive={true}
		loading={$deleteMut.isPending}
		onConfirm={handleDelete}
		onCancel={() => (showDeleteConfirm = false)}
	/>
{/if}
