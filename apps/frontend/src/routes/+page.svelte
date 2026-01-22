<script lang="ts">
	import { browser } from "$app/environment";
	import { goto } from "$app/navigation";
	import { page } from "$app/stores";
	import {
		type Goal,
		type Task,
		getGoals,
		getTasks,
		updateTask,
	} from "$lib/api";
	import { type Category, getCategories } from "$lib/api/categories";
	import type { Emotion } from "$lib/api/emotions";
	import { EmotionModal } from "$lib/components/emotions";
	import {
		type EmotionSelectionContext,
		TaskModal,
	} from "$lib/components/tasks";
	import {
		TimelineGantt,
		TimelineAgenda,
		type TimelineGoal,
	} from "$lib/components/timeline";
	import {
		Card,
		ConfirmDialog,
		IconBox,
		SectionHeader,
	} from "$lib/components/ui";
	import { emotionStore } from "$lib/stores/emotions.svelte";
	import { cn } from "$lib/utils";
	import {
		getUrlParams,
		parsers,
		updateUrlParams,
	} from "$lib/utils/navigation";
	import {
		createMutation,
		createQuery,
		useQueryClient,
	} from "@tanstack/svelte-query";
	import {
		AlignHorizontalJustifyStart,
		Calendar,
		ClipboardList,
		Flame,
		List,
		ListTodo,
		Plus,
		Smile,
		Target,
		X,
		Zap,
	} from "lucide-svelte";

	// Selected date state initialized from URL
	const initialParams = getUrlParams<{ date: Date; view: string }>({
		date: parsers.dateOnly(new Date()),
		view: parsers.string("timeline"),
	});

	let selectedDate = $state<Date>(initialParams.date);
	let currentView = $state<"timeline" | "agenda">(
		(initialParams.view as "timeline" | "agenda") || "timeline",
	);

	function handleViewChange(view: "timeline" | "agenda") {
		currentView = view;
	}

	// Sync state to URL
	$effect(() => {
		// Format as YYYY-MM-DD local time
		const year = selectedDate.getFullYear();
		const month = String(selectedDate.getMonth() + 1).padStart(2, "0");
		const day = String(selectedDate.getDate()).padStart(2, "0");

		updateUrlParams({
			date: `${year}-${month}-${day}`,
			view: currentView,
		});
	});

	// Helper to format date for API (Correctly handles local timezone to UTC)
	function formatDateForApi(date: Date): {
		startOfDay: string;
		endOfDay: string;
	} {
		// Create date at start of local day (00:00:00)
		const start = new Date(date);
		start.setHours(0, 0, 0, 0);

		// Create date at end of local day (23:59:59.999)
		const end = new Date(date);
		end.setHours(23, 59, 59, 999);

		return {
			startOfDay: start.toISOString(),
			endOfDay: end.toISOString(),
		};
	}

	// Helper to get previous day for sleep task query
	function getPreviousDayRange(date: Date): {
		startOfDay: string;
		endOfDay: string;
	} {
		const prevDay = new Date(date);
		prevDay.setDate(prevDay.getDate() - 1);
		return formatDateForApi(prevDay);
	}

	// Query key based on selected date
	const dateKey = $derived(
		`${selectedDate.getFullYear()}-${selectedDate.getMonth()}-${selectedDate.getDate()}`,
	);

	// Track dateKey changes to refetch
	let currentDateKey = "";

	// Current and previous day query params (computed from selectedDate)
	let queryParams = $derived.by(() => {
		const { startOfDay, endOfDay } = formatDateForApi(selectedDate);
		return { startOfDay, endOfDay };
	});

	let prevQueryParams = $derived.by(() => {
		const prevDay = new Date(selectedDate);
		prevDay.setDate(prevDay.getDate() - 1);
		return formatDateForApi(prevDay);
	});

	// Fetch tasks for the selected date
	const tasksQuery = createQuery({
		queryKey: ["tasks", "timeline"],
		queryFn: () => {
			const { startOfDay, endOfDay } = formatDateForApi(selectedDate);
			return getTasks({
				limit: 100,
				start_date_from: startOfDay,
				start_date_to: endOfDay,
				sort_field: "start_date",
				sort_order: "asc",
			});
		},
	});

	// Also fetch tasks from previous day that might extend into current day (sleep tasks)
	const prevDayTasksQuery = createQuery({
		queryKey: ["tasks", "timeline-prev"],
		queryFn: () => {
			const prevDay = new Date(selectedDate);
			prevDay.setDate(prevDay.getDate() - 1);
			const { startOfDay, endOfDay } = formatDateForApi(prevDay);
			return getTasks({
				limit: 50,
				start_date_from: startOfDay,
				start_date_to: endOfDay,
				sort_field: "start_date",
				sort_order: "asc",
			});
		},
	});

	// Refetch when date changes
	$effect(() => {
		if (currentDateKey !== "" && currentDateKey !== dateKey) {
			$tasksQuery.refetch();
			$prevDayTasksQuery.refetch();
		}
		currentDateKey = dateKey;
	});

	const categoriesQuery = createQuery({
		queryKey: ["categories"],
		queryFn: () => getCategories({ limit: 100 }),
	});

	// Fetch active goals
	const goalsQuery = createQuery({
		queryKey: ["goals", "active"],
		queryFn: () => getGoals({ status: "active", limit: 50 }),
	});

	// Transform goals for timeline display
	function transformGoals(apiGoals: Goal[]): TimelineGoal[] {
		return apiGoals.map((goal) => ({
			id: goal.id,
			title: goal.title,
			icon: goal.icon || "🎯",
			color: goal.category?.color || "#8b5cf6",
			startDate: goal.start_date ? new Date(goal.start_date) : undefined,
			deadline: goal.deadline ? new Date(goal.deadline) : undefined,
			recurrence: goal.recurrence
				? {
						frequency: goal.recurrence.frequency,
						period: goal.recurrence.period,
					}
				: undefined,
			progress: goal.stats?.progress_percent || 0,
			currentStreak: goal.stats?.current_streak || 0,
			todayStatus: goal.stats?.today_status,
			linkedTaskIds: goal.linked_task_ids || [],
			target: goal.target
				? {
						value: goal.target.value,
						currentValue: goal.stats?.current_value || 0,
						unitId: goal.target.unit_id,
					}
				: undefined,
		}));
	}

	// Transformed goals for timeline
	const timelineGoals = $derived(
		transformGoals($goalsQuery.data?.items || []),
	);

	function handleGoalClick(goalId: string) {
		goto(`/goals?modal=${goalId}`);
	}

	function handleCreateTaskFromGoal(goalId: string) {
		// Navigate to task creation with goal pre-linked
		const goal = ($goalsQuery.data?.items || []).find(
			(g) => g.id === goalId,
		);
		if (goal) {
			goto(`/tasks/create?goal=${goalId}`);
		}
	}

	// Filter state
	let selectedCategoryId = $state<string | null>(null);

	// FAB menu state
	let fabOpen = $state(false);

	// Transform API tasks to timeline format
	type TimelineTask = {
		id: string;
		title: string;
		description?: string;
		startTime: Date;
		endTime: Date;
		categoryColor?: string;
		categoryName?: string;
		completed?: boolean;
		emoji?: string;
		categoryId?: string;
		// Emotion fields (user-selected)
		emotionId?: string;
		emotionName?: string;
		emotionEmoji?: string;
		emotionQuadrant?: "yellow" | "green" | "red" | "blue";
		emotionDescription?: string;
		// Inferred emotion
		inferredEmotionId?: string;
		inferredEmotionName?: string;
		inferredEmotionEmoji?: string;
		inferredEmotionQuadrant?: "yellow" | "green" | "red" | "blue";
		inferredEmotionDescription?: string;
		// Linked goals for highlighting
		linkedGoals?: {
			id: string;
			title: string;
			icon?: string;
			color?: string;
			impactType: "positive" | "negative" | "neutral";
		}[];
	};

	function transformTasks(apiTasks: Task[]): TimelineTask[] {
		return apiTasks.map((task) => {
			const startTime = task.start_date
				? new Date(task.start_date)
				: new Date();
			const endTime = task.end_date
				? new Date(task.end_date)
				: new Date(startTime.getTime() + 30 * 60000);

			return {
				id: task.id,
				title: task.title,
				description: task.journal,
				startTime,
				endTime,
				categoryColor: task.category?.color,
				categoryName: task.category?.name,
				completed: task.completed,
				emoji: task.completed ? "✓" : undefined,
				categoryId: task.category?.id,
				emotionId: task.emotion_id,
				inferredEmotionId: task.inferred_emotion?.closest_emotion_id,
				linkedGoals: task.linked_goals?.map((g) => ({
					id: g.id,
					title: g.title,
					icon: g.icon,
					color: g.color,
					impactType: g.impact_type,
					quantityValue: g.quantity_value,
					unitSymbol: g.unit_symbol,
				})),
			};
		});
	}

	// Enrich tasks with emotion data (both selected and inferred) using the store
	function enrichWithEmotions(tasks: TimelineTask[]): TimelineTask[] {
		if (!emotionStore.isInitialized) return tasks;

		return tasks.map((task) => {
			let enrichedTask = { ...task };

			// Selected emotion data
			if (task.emotionId) {
				const emotion = emotionStore.get(task.emotionId);
				if (emotion) {
					enrichedTask = {
						...enrichedTask,
						emotionName: emotion.name,
						emotionEmoji: emotion.emoji,
						emotionQuadrant: emotion.quadrant,
						emotionDescription: emotion.description,
					};
				}
			}

			// Inferred emotion data
			if (task.inferredEmotionId) {
				const inferredEmotion = emotionStore.get(
					task.inferredEmotionId,
				);
				if (inferredEmotion) {
					enrichedTask = {
						...enrichedTask,
						inferredEmotionName: inferredEmotion.name,
						inferredEmotionEmoji: inferredEmotion.emoji,
						inferredEmotionQuadrant: inferredEmotion.quadrant,
						inferredEmotionDescription: inferredEmotion.description,
					};
				}
			}

			return enrichedTask;
		});
	}

	// Check if a task is relevant for the selected date view
	function isTaskVisibleOnDate(task: TimelineTask, date: Date): boolean {
		const viewStart = new Date(date);
		viewStart.setHours(0, 0, 0, 0);
		const viewEnd = new Date(date);
		viewEnd.setHours(23, 59, 59, 999);

		// Task is visible if it overlaps with the view window
		return task.startTime <= viewEnd && task.endTime >= viewStart;
	}

	// Combine current day tasks with overnight tasks from previous day
	const allTasksRaw = $derived.by(() => {
		const currentDayTasks = transformTasks($tasksQuery.data?.items || []);
		const prevDayTasks = transformTasks(
			$prevDayTasksQuery.data?.items || [],
		);

		// Filter previous day tasks to only include those that extend into current day
		const overnightTasks = prevDayTasks.filter((task) => {
			const viewStart = new Date(selectedDate);
			viewStart.setHours(0, 0, 0, 0);
			// Task ends after midnight of selected date
			return task.endTime > viewStart;
		});

		// Combine and deduplicate by ID
		const taskMap = new Map<string, TimelineTask>();
		[...overnightTasks, ...currentDayTasks].forEach((task) => {
			taskMap.set(task.id, task);
		});

		return Array.from(taskMap.values());
	});

	// Enrich tasks with emotions when raw tasks change
	// Enrich tasks with emotions reactively
	const emotionEnrichedTasks = $derived(enrichWithEmotions(allTasksRaw));

	// Use enriched tasks for display
	const allTasks = $derived(() =>
		emotionEnrichedTasks.length > 0 ? emotionEnrichedTasks : allTasksRaw,
	);

	const timelineTasks = $derived(
		selectedCategoryId
			? allTasks().filter((t) => t.categoryId === selectedCategoryId)
			: allTasks(),
	);

	const categories = $derived($categoriesQuery.data?.items || []);
	const completedCount = $derived(
		allTasks().filter((t) => t.completed).length,
	);
	const totalCount = $derived(allTasks().length);
	const completionRate = $derived(
		totalCount > 0 ? Math.round((completedCount / totalCount) * 100) : 0,
	);

	// Check if selected date is today
	const isSelectedDateToday = $derived(() => {
		const today = new Date();
		return (
			selectedDate.getFullYear() === today.getFullYear() &&
			selectedDate.getMonth() === today.getMonth() &&
			selectedDate.getDate() === today.getDate()
		);
	});

	// Task modal state
	let modalOpen = $state(false);
	let editingTask = $state<Task | null>(null);
	let initialCategoryId = $state<string | undefined>(undefined);

	// Uncomplete confirmation state
	let showUncompleteConfirm = $state(false);
	let pendingUncompleteTaskId = $state<string | null>(null);

	// Emotion modal state (for modal switching)
	let emotionModalOpen = $state(false);
	let pendingEmotion = $state<Emotion | null>(null);
	let emotionSelectionContext = $state<EmotionSelectionContext | null>(null);

	// Toggle complete mutation
	const queryClient = useQueryClient();
	const toggleCompleteMut = createMutation({
		mutationFn: ({ id, completed }: { id: string; completed: boolean }) =>
			updateTask(id, { completed }),
		onSuccess: () => {
			// Invalidate both queries
			queryClient.invalidateQueries({ queryKey: ["tasks", "timeline"] });
			queryClient.invalidateQueries({
				queryKey: ["tasks", "timeline-prev"],
			});
		},
	});

	// Handle date change from Timeline
	function handleDateChange(newDate: Date) {
		selectedDate = newDate;
		selectedCategoryId = null; // Reset filter on date change
	}

	// Handle task click from timeline - navigate to edit page
	function handleTaskClick(taskId: string) {
		if (browser) {
			sessionStorage.setItem(
				"task-form-referrer",
				$page.url.pathname + $page.url.search,
			);
		}
		goto(`/tasks/${taskId}`);
	}

	// Handle category label click from timeline - navigate to create page with category
	function handleCategoryClick(categoryId: string) {
		if (browser) {
			sessionStorage.setItem(
				"task-form-referrer",
				$page.url.pathname + $page.url.search,
			);
		}
		goto(`/tasks/create?category=${categoryId}`);
	}

	// Handle modal close
	function handleModalClose() {
		modalOpen = false;
		editingTask = null;
		initialCategoryId = undefined;
		pendingEmotion = null; // Clear pending emotion
		queryClient.invalidateQueries({ queryKey: ["tasks", "timeline"] });
		queryClient.invalidateQueries({ queryKey: ["tasks", "timeline-prev"] });
	}

	// Handle opening emotion modal from TaskModal
	function handleOpenEmotionModal(context: EmotionSelectionContext) {
		emotionSelectionContext = context;
		emotionModalOpen = true;
	}

	// Handle emotion selection from EmotionModal
	function handleEmotionSelect(emotion: Emotion) {
		pendingEmotion = emotion;
		emotionModalOpen = false;
		modalOpen = true; // Reopen task modal
	}

	// Handle emotion modal close (cancelled)
	function handleEmotionModalClose() {
		emotionModalOpen = false;
		modalOpen = true; // Reopen task modal without selection
		emotionSelectionContext = null;
	}

	// Handle toggle complete from timeline
	function handleToggleComplete(taskId: string, newCompleted: boolean) {
		if (!newCompleted) {
			pendingUncompleteTaskId = taskId;
			showUncompleteConfirm = true;
		} else {
			$toggleCompleteMut.mutate({ id: taskId, completed: true });
		}
	}

	function confirmUncomplete() {
		if (pendingUncompleteTaskId) {
			$toggleCompleteMut.mutate({
				id: pendingUncompleteTaskId,
				completed: false,
			});
			pendingUncompleteTaskId = null;
			showUncompleteConfirm = false;
		}
	}

	// Quick logs data
	const quickLogs = [
		{ emoji: "☕", label: "Morning", color: "#f97316" },
		{ emoji: "💪", label: "Workout", color: "#ef4444" },
		{ emoji: "📚", label: "Reading", color: "#3b82f6" },
		{ emoji: "🧘", label: "Meditation", color: "#8b5cf6" },
		{ emoji: "🍽️", label: "Meal", color: "#22c55e" },
		{ emoji: "💤", label: "Sleep", color: "#6366f1" },
		{ emoji: "💧", label: "Water", color: "#06b6d4" },
		{ emoji: "🚶", label: "Walk", color: "#10b981" },
	];

	// Get greeting based on time
	function getGreeting(): string {
		const hour = new Date().getHours();
		if (hour < 12) return "Good Morning";
		if (hour < 17) return "Good Afternoon";
		return "Good Evening";
	}

	function formatDate(): string {
		return new Date().toLocaleDateString("en-US", {
			weekday: "long",
			month: "long",
			day: "numeric",
		});
	}

	function handleQuickLog(log: { emoji: string; label: string }) {
		// TODO: Implement quick log creation
		console.log("Quick log:", log);
		fabOpen = false;
	}

	function openTaskModal() {
		fabOpen = false;
		if (browser) {
			sessionStorage.setItem(
				"task-form-referrer",
				$page.url.pathname + $page.url.search,
			);
		}
		goto("/tasks/create");
	}

	// Update task time mutation
	const updateTaskTimeMut = createMutation({
		mutationFn: ({
			id,
			start_date,
			end_date,
		}: {
			id: string;
			start_date: string;
			end_date: string;
		}) => updateTask(id, { start_date, end_date }),
		onSuccess: () => {
			// Invalidate both queries to refresh data
			queryClient.invalidateQueries({ queryKey: ["tasks", "timeline"] });
			queryClient.invalidateQueries({
				queryKey: ["tasks", "timeline-prev"],
			});
		},
	});

	function handleTaskTimeUpdate(taskId: string, start: Date, end: Date) {
		// Optimistically update is handled in the component
		// Here we just fire the API request
		const { startOfDay: startDateStr } = formatDateForApi(start);
		// Be careful with timezones. The API expects ISO strings.
		// The previous transform used new Date(task.start_date), so we should send back ISO.

		// We need to preserve the full timestamp
		$updateTaskTimeMut.mutate({
			id: taskId,
			start_date: start.toISOString(),
			end_date: end.toISOString(),
		});
	}
</script>

<svelte:head>
	<title>Dashboard - Lucid Logs</title>
</svelte:head>

<div class="h-full flex flex-col gap-4 lg:gap-5 relative">
	<!-- Top Section: Welcome + Stats + Quick Logs -->
	<div class="flex-shrink-0">
		<!-- Welcome Header -->
		<div class="flex items-center justify-between mb-4">
			<div>
				<p class="text-xs uppercase font-medium opacity-50">
					{formatDate()}
				</p>
				<h1 class="text-2xl sm:text-3xl font-bold mt-1">
					{getGreeting()}! 👋
				</h1>
			</div>
			<div class="flex items-center gap-3">
				<!-- View Toggle -->
				<div
					class="join border border-base-300/60 rounded-lg bg-base-100 h-10"
				>
					<button
						type="button"
						class="join-item btn btn-sm h-full px-3 sm:px-4 gap-2 {currentView ===
						'timeline'
							? 'btn-primary'
							: 'btn-ghost'}"
						onclick={() => handleViewChange("timeline")}
						title="Timeline View"
					>
						<AlignHorizontalJustifyStart class="w-4 h-4" />
						<span class="font-medium hidden sm:inline"
							>Timeline</span
						>
					</button>
					<button
						type="button"
						class="join-item btn btn-sm h-full px-3 sm:px-4 gap-2 {currentView ===
						'agenda'
							? 'btn-primary'
							: 'btn-ghost'}"
						onclick={() => handleViewChange("agenda")}
						title="Agenda View"
					>
						<List class="w-4 h-4" />
						<span class="font-medium hidden sm:inline">Agenda</span>
					</button>
				</div>

				<button
					class="btn btn-primary gap-2 h-10"
					onclick={openTaskModal}
				>
					<Plus class="w-4 h-4" />
					<span class="hidden sm:inline">New Task</span>
				</button>
			</div>
		</div>

		<!-- Stats & Quick Log Card -->
		<div
			class="card bg-base-100 shadow-md border border-base-200/50 rounded-2xl overflow-hidden"
		>
			<div class="card-body p-5">
				<!-- Stats Row -->
				<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
					<!-- Streak -->
					<div
						class="group flex items-center gap-3 p-2.5 rounded-xl bg-base-100 hover:bg-base-200/30 transition-all duration-300 cursor-pointer hover:shadow-[0_4px_20px_rgb(0,0,0,0.06)] hover:-translate-y-0.5"
					>
						<div
							class="w-11 h-11 rounded-xl bg-gradient-to-br from-warning to-warning/70 flex items-center justify-center text-warning-content shadow-md transition-transform duration-300 group-hover:scale-110"
						>
							<Flame class="w-5 h-5" />
						</div>
						<div>
							<p
								class="text-[10px] font-semibold uppercase tracking-wide opacity-50"
							>
								Streak
							</p>
							<p
								class="text-2xl font-bold text-warning leading-tight"
							>
								7<span
									class="text-xs font-medium opacity-60 ml-0.5"
									>d</span
								>
							</p>
						</div>
					</div>

					<!-- Tasks -->
					<div
						class="group flex items-center gap-3 p-2.5 rounded-xl bg-base-100 hover:bg-base-200/30 transition-all duration-300 cursor-pointer hover:shadow-[0_4px_20px_rgb(0,0,0,0.06)] hover:-translate-y-0.5"
					>
						<div
							class="w-11 h-11 rounded-xl bg-gradient-to-br from-info to-info/70 flex items-center justify-center text-info-content shadow-md transition-transform duration-300 group-hover:scale-110"
						>
							<ListTodo class="w-5 h-5" />
						</div>
						<div>
							<p
								class="text-[10px] font-semibold uppercase tracking-wide opacity-50"
							>
								Tasks
							</p>
							<p
								class="text-2xl font-bold text-info leading-tight"
							>
								{completedCount}<span
									class="text-xs font-medium opacity-60 ml-0.5"
									>/{totalCount}</span
								>
							</p>
						</div>
					</div>

					<!-- Goals -->
					<div
						class="group flex items-center gap-3 p-2.5 rounded-xl bg-base-100 hover:bg-base-200/30 transition-all duration-300 cursor-pointer hover:shadow-[0_4px_20px_rgb(0,0,0,0.06)] hover:-translate-y-0.5"
					>
						<div
							class="w-11 h-11 rounded-xl bg-gradient-to-br from-accent to-accent/70 flex items-center justify-center text-accent-content shadow-md transition-transform duration-300 group-hover:scale-110"
						>
							<Target class="w-5 h-5" />
						</div>
						<div>
							<p
								class="text-[10px] font-semibold uppercase tracking-wide opacity-50"
							>
								Goals
							</p>
							<p
								class="text-2xl font-bold text-accent leading-tight"
							>
								3<span
									class="text-xs font-medium opacity-60 ml-0.5"
									>active</span
								>
							</p>
						</div>
					</div>

					<!-- Mood -->
					<div
						class="group flex items-center gap-3 p-2.5 rounded-xl bg-base-100 hover:bg-base-200/30 transition-all duration-300 cursor-pointer hover:shadow-[0_4px_20px_rgb(0,0,0,0.06)] hover:-translate-y-0.5"
					>
						<div
							class="w-11 h-11 rounded-xl bg-gradient-to-br from-success to-success/70 flex items-center justify-center text-success-content shadow-md transition-transform duration-300 group-hover:scale-110"
						>
							<Smile class="w-5 h-5" />
						</div>
						<div>
							<p
								class="text-[10px] font-semibold uppercase tracking-wide opacity-50"
							>
								Mood
							</p>
							<div class="flex items-center gap-1.5">
								<span class="text-lg">😊</span>
								<span class="text-sm font-bold text-success"
									>Great</span
								>
							</div>
						</div>
					</div>
				</div>

				<div class="divider my-3 opacity-50"></div>

				<!-- Quick Log Row -->
				<div class="flex items-center justify-between gap-4">
					<div class="flex items-center gap-2.5">
						<div
							class="w-9 h-9 rounded-lg bg-gradient-to-br from-secondary to-secondary/70 flex items-center justify-center text-secondary-content shadow-md"
						>
							<Zap class="w-4 h-4" />
						</div>
						<span class="font-bold text-sm">Quick Log</span>
					</div>
					<div class="flex items-center gap-1.5 flex-wrap">
						{#each quickLogs as log}
							<button
								class="btn btn-ghost btn-sm gap-1.5 px-2.5 hover:bg-base-200/50 hover:shadow-sm hover:-translate-y-0.5 transition-all duration-200 rounded-lg"
								onclick={() => handleQuickLog(log)}
								title={log.label}
							>
								<span class="text-base">{log.emoji}</span>
								<span
									class="text-xs font-medium hidden sm:inline"
									>{log.label}</span
								>
							</button>
						{/each}
					</div>
				</div>
			</div>
		</div>
	</div>

	<!-- Timeline Section -->
	<div class="flex-1 min-h-0 flex flex-col">
		<div
			class="card bg-base-100 shadow-md border border-base-200/50 rounded-2xl h-full flex flex-col overflow-hidden"
		>
			<div class="card-body p-3 lg:p-4 flex flex-col h-full gap-3">
				<!-- Timeline -->

				<!-- Timeline -->
				<div class="flex-1 min-h-0">
					{#if $tasksQuery.isLoading}
						<div
							class="flex flex-col items-center justify-center h-full gap-3"
						>
							<span
								class="loading loading-spinner loading-lg text-primary"
							></span>
							<p class="text-sm opacity-50">
								Loading your day...
							</p>
						</div>
					{:else if currentView === "timeline"}
						<TimelineGantt
							tasks={timelineTasks}
							goals={timelineGoals}
							{selectedDate}
							showGoals={true}
							onTaskClick={handleTaskClick}
							onCategoryClick={handleCategoryClick}
							onToggleComplete={handleToggleComplete}
							onDateChange={handleDateChange}
							onTaskTimeUpdate={handleTaskTimeUpdate}
							onGoalClick={handleGoalClick}
							onCreateTaskFromGoal={handleCreateTaskFromGoal}
						/>
					{:else}
						<TimelineAgenda
							tasks={timelineTasks}
							goals={timelineGoals}
							{selectedDate}
							showGoals={true}
							onTaskClick={handleTaskClick}
							onCategoryClick={handleCategoryClick}
							onToggleComplete={handleToggleComplete}
							onDateChange={handleDateChange}
							onGoalClick={handleGoalClick}
							onCreateTaskFromGoal={handleCreateTaskFromGoal}
						/>
					{/if}
				</div>
			</div>
		</div>
	</div>

	<!-- Floating Action Button -->
	<div class="fixed bottom-6 right-6 z-50">
		<!-- FAB Menu -->
		{#if fabOpen}
			<!-- Backdrop -->
			<button
				class="fixed inset-0 bg-base-content/20 backdrop-blur-sm z-40"
				onclick={() => (fabOpen = false)}
				aria-label="Close menu"
			></button>

			<!-- Menu -->
			<div class="absolute bottom-16 right-0 z-50 animate-fade-in">
				<div
					class="card bg-base-100 shadow-2xl border border-base-200/50 rounded-2xl min-w-[280px] overflow-hidden"
				>
					<div class="card-body p-3">
						<!-- Create Task Option -->
						<button
							class="btn btn-ghost justify-start gap-3 h-auto py-3"
							onclick={openTaskModal}
						>
							<div
								class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center"
							>
								<ClipboardList class="w-5 h-5 text-primary" />
							</div>
							<div class="text-left">
								<p class="font-semibold text-sm">New Task</p>
								<p class="text-xs opacity-50">
									Create a detailed task
								</p>
							</div>
						</button>

						<div
							class="divider my-1 text-[10px] uppercase opacity-50"
						>
							Quick Logs
						</div>

						<!-- Quick Logs Grid -->
						<div class="grid grid-cols-4 gap-2">
							{#each quickLogs as log}
								<button
									class="btn btn-ghost btn-sm flex-col h-auto py-2 gap-1 hover:bg-base-200"
									onclick={() => handleQuickLog(log)}
								>
									<span class="text-xl">{log.emoji}</span>
									<span
										class="text-[9px] font-medium opacity-60"
										>{log.label}</span
									>
								</button>
							{/each}
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- FAB Button -->
		<button
			class={cn(
				"btn btn-circle btn-lg shadow-2xl transition-all duration-300",
				fabOpen ? "btn-error rotate-45" : "btn-primary hover:scale-110",
			)}
			onclick={() => (fabOpen = !fabOpen)}
			aria-label={fabOpen ? "Close menu" : "Open menu"}
		>
			{#if fabOpen}
				<X class="w-6 h-6" />
			{:else}
				<Plus class="w-6 h-6" />
			{/if}
		</button>
	</div>
</div>

<!-- Task Modal -->
<TaskModal
	bind:open={modalOpen}
	task={editingTask}
	{initialCategoryId}
	onClose={handleModalClose}
	onOpenEmotionModal={handleOpenEmotionModal}
	{pendingEmotion}
/>

<!-- Emotion Modal (separate from task modal) -->
<EmotionModal
	bind:open={emotionModalOpen}
	onSelect={handleEmotionSelect}
	onClose={handleEmotionModalClose}
/>
<ConfirmDialog
	bind:open={showUncompleteConfirm}
	title="Mark as Incomplete?"
	message="Are you sure you want to mark this task as incomplete? This will remove the completion status."
	confirmText="Mark Incomplete"
	cancelText="Keep Complete"
	destructive={false}
	loading={$toggleCompleteMut.isPending}
	onConfirm={confirmUncomplete}
	onCancel={() => {
		showUncompleteConfirm = false;
		pendingUncompleteTaskId = null;
	}}
/>
