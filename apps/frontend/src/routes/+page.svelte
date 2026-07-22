<script lang="ts">
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { page } from '$app/stores';
import { type Goal, type Task, getGoals, getTasks, updateTask } from '$lib/api';
import { getCategories } from '$lib/api/categories';
import QuickCapture from '$lib/components/tasks/QuickCapture.svelte';
import { TimelineGantt, type TimelineGoal } from '$lib/components/timeline';
import AgendaRow from '$lib/components/timeline/AgendaRow.svelte';
import { ConfirmDialog, EmptyState } from '$lib/components/ui';
import { emotionStore } from '$lib/stores/emotions.svelte';
import { cn, FALLBACK_GOAL_COLOR } from '$lib/utils';
import { getUrlParams, parsers, updateUrlParams } from '$lib/utils/navigation';
import {
	createMutation,
	createQuery,
	useQueryClient,
} from '@tanstack/svelte-query';
import {
	ClipboardList,
	Flame,
	GanttChart,
	List,
	ListTodo,
	Plus,
	Smile,
	Target,
	X,
	Zap,
} from 'lucide-svelte';
import { onMount } from 'svelte';

// Selected date state initialized from URL
const initialParams = getUrlParams<{ date: Date }>({
	date: parsers.dateOnly(new Date()),
});

let selectedDate = $state<Date>(initialParams.date);

// Mobile detection and view mode toggle
let isMobile = $state(false);
let viewMode = $state<'timeline' | 'list'>('timeline');

function detectMobile() {
	if (typeof window === 'undefined') return false;
	return window.innerWidth < 768;
}

function initMobileState() {
	const mobile = detectMobile();
	isMobile = mobile;
	if (browser) {
		const stored = localStorage.getItem('dashboard-view-mode');
		if (stored === 'timeline' || stored === 'list') {
			viewMode = stored;
		} else {
			// Default based on viewport
			viewMode = mobile ? 'list' : 'timeline';
		}
	}
}

function setViewMode(mode: 'timeline' | 'list') {
	viewMode = mode;
	if (browser) {
		localStorage.setItem('dashboard-view-mode', mode);
	}
}

// Handle window resize for mobile detection
function handleResize() {
	isMobile = detectMobile();
}

onMount(() => {
	initMobileState();
	window.addEventListener('resize', handleResize);
	return () => window.removeEventListener('resize', handleResize);
});

// Sync state to URL
$effect(() => {
	// Format as YYYY-MM-DD local time
	const year = selectedDate.getFullYear();
	const month = String(selectedDate.getMonth() + 1).padStart(2, '0');
	const day = String(selectedDate.getDate()).padStart(2, '0');

	updateUrlParams({
		date: `${year}-${month}-${day}`,
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
let currentDateKey = '';

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
	queryKey: ['tasks', 'timeline'],
	queryFn: () => {
		const { startOfDay, endOfDay } = formatDateForApi(selectedDate);
		return getTasks({
			limit: 100,
			start_date_from: startOfDay,
			start_date_to: endOfDay,
			sort_field: 'start_date',
			sort_order: 'asc',
		});
	},
});

// Also fetch tasks from previous day that might extend into current day (sleep tasks)
const prevDayTasksQuery = createQuery({
	queryKey: ['tasks', 'timeline-prev'],
	queryFn: () => {
		const prevDay = new Date(selectedDate);
		prevDay.setDate(prevDay.getDate() - 1);
		const { startOfDay, endOfDay } = formatDateForApi(prevDay);
		return getTasks({
			limit: 50,
			start_date_from: startOfDay,
			start_date_to: endOfDay,
			sort_field: 'start_date',
			sort_order: 'asc',
		});
	},
});

// Refetch when date changes
$effect(() => {
	if (currentDateKey !== '' && currentDateKey !== dateKey) {
		$tasksQuery.refetch();
		$prevDayTasksQuery.refetch();
	}
	currentDateKey = dateKey;
});

const categoriesQuery = createQuery({
	queryKey: ['categories'],
	queryFn: () => getCategories({ limit: 100 }),
});

// Fetch active goals
const goalsQuery = createQuery({
	queryKey: ['goals', 'active'],
	queryFn: () => getGoals({ status: 'active', limit: 50 }),
});

// Transform goals for timeline display
function transformGoals(apiGoals: Goal[]): TimelineGoal[] {
	return apiGoals.map((goal) => ({
		id: goal.id,
		title: goal.title,
		icon: goal.icon || '🎯',
		color: goal.category?.color || FALLBACK_GOAL_COLOR,
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
const timelineGoals = $derived(transformGoals($goalsQuery.data?.items || []));

function handleGoalClick(goalId: string) {
	goto(`/goals?modal=${goalId}`);
}

function handleCreateTaskFromGoal(goalId: string) {
	// Navigate to task creation with goal pre-linked
	const goal = ($goalsQuery.data?.items || []).find((g) => g.id === goalId);
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
	emotionQuadrant?: 'yellow' | 'green' | 'red' | 'blue';
	emotionDescription?: string;
	// Inferred emotion
	inferredEmotionId?: string;
	inferredEmotionName?: string;
	inferredEmotionEmoji?: string;
	inferredEmotionQuadrant?: 'yellow' | 'green' | 'red' | 'blue';
	inferredEmotionDescription?: string;
	// Linked goals for highlighting
	linkedGoals?: {
		id: string;
		title: string;
		icon?: string;
		color?: string;
		impactType: 'positive' | 'negative' | 'neutral';
	}[];
};

function transformTasks(apiTasks: Task[]): TimelineTask[] {
	return apiTasks.map((task) => {
		const startTime = task.start_date ? new Date(task.start_date) : new Date();
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
			emoji: task.completed ? '✓' : undefined,
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
			const inferredEmotion = emotionStore.get(task.inferredEmotionId);
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
	const prevDayTasks = transformTasks($prevDayTasksQuery.data?.items || []);

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

// Group tasks by time-of-day for the agenda view
const agendaGroups = $derived.by(() => {
	const groups = [
		{ key: 'morning', label: 'Morning', tasks: [] as typeof timelineTasks },
		{ key: 'afternoon', label: 'Afternoon', tasks: [] as typeof timelineTasks },
		{ key: 'evening', label: 'Evening', tasks: [] as typeof timelineTasks },
	];
	for (const task of timelineTasks) {
		const h = task.startTime.getHours();
		if (h < 12) groups[0].tasks.push(task);
		else if (h < 17) groups[1].tasks.push(task);
		else groups[2].tasks.push(task);
	}
	return groups;
});

const categories = $derived($categoriesQuery.data?.items || []);
const completedCount = $derived(allTasks().filter((t) => t.completed).length);
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

// Uncomplete confirmation state
let showUncompleteConfirm = $state(false);
let pendingUncompleteTaskId = $state<string | null>(null);

// Toggle complete mutation
const queryClient = useQueryClient();
const toggleCompleteMut = createMutation({
	mutationFn: ({ id, completed }: { id: string; completed: boolean }) =>
		updateTask(id, { completed }),
	onSuccess: () => {
		// Invalidate both queries
		queryClient.invalidateQueries({ queryKey: ['tasks', 'timeline'] });
		queryClient.invalidateQueries({
			queryKey: ['tasks', 'timeline-prev'],
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
			'task-form-referrer',
			$page.url.pathname + $page.url.search,
		);
	}
	goto(`/tasks/${taskId}`);
}

// Handle category label click from timeline - navigate to create page with category
function handleCategoryClick(categoryId: string) {
	if (browser) {
		sessionStorage.setItem(
			'task-form-referrer',
			$page.url.pathname + $page.url.search,
		);
	}
	goto(`/tasks/create?category=${categoryId}`);
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

// Quick logs data — colors from COLOR_PRESETS (design language §3.4)
const quickLogs = [
	{ emoji: '☕', label: 'Morning', color: '#f97316' },
	{ emoji: '💪', label: 'Workout', color: '#ef4444' },
	{ emoji: '📚', label: 'Reading', color: '#3b82f6' },
	{ emoji: '🧘', label: 'Meditation', color: '#8b5cf6' },
	{ emoji: '🍽️', label: 'Meal', color: '#22c55e' },
	{ emoji: '💤', label: 'Sleep', color: '#6366f1' },
	{ emoji: '💧', label: 'Water', color: '#06b6d4' },
	{ emoji: '🚶', label: 'Walk', color: '#10b981' },
];

// Get greeting based on time
function getGreeting(): string {
	const hour = new Date().getHours();
	if (hour < 12) return 'Good Morning';
	if (hour < 17) return 'Good Afternoon';
	return 'Good Evening';
}

function formatDate(): string {
	return new Date().toLocaleDateString('en-US', {
		weekday: 'long',
		month: 'long',
		day: 'numeric',
	});
}

// Helpers for agenda view
function formatTimeCompact(d: Date): string {
	if (!(d instanceof Date) || Number.isNaN(d.getTime())) return '--:--';
	const hours = d.getHours();
	const mins = d.getMinutes();
	const period = hours < 12 ? 'AM' : 'PM';
	const displayHour = hours === 0 ? 12 : hours > 12 ? hours - 12 : hours;
	return `${displayHour}:${String(mins).padStart(2, '0')} ${period}`;
}

function formatDuration(s: Date, e: Date): string {
	if (!(s instanceof Date) || !(e instanceof Date)) return '';
	const mins = Math.round((e.getTime() - s.getTime()) / 60000);
	if (mins < 1) return `${Math.round((e.getTime() - s.getTime()) / 1000)}s`;
	if (mins < 60) return `${mins}m`;
	const h = Math.floor(mins / 60);
	const m = mins % 60;
	return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

function handleQuickLog(log: { emoji: string; label: string }) {
	// Navigate to activities page — quick log via activity instant-log
	fabOpen = false;
	goto('/activities');
}

function openTaskModal() {
	fabOpen = false;
	if (browser) {
		sessionStorage.setItem(
			'task-form-referrer',
			$page.url.pathname + $page.url.search,
		);
	}
	goto('/tasks/create');
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
		queryClient.invalidateQueries({ queryKey: ['tasks', 'timeline'] });
		queryClient.invalidateQueries({
			queryKey: ['tasks', 'timeline-prev'],
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
		<div class="flex items-end justify-between gap-3 mb-5">
			<div class="min-w-0">
				<p class="text-[13px] font-medium text-base-content/50">
					{formatDate()}
				</p>
				<h1 class="text-2xl font-semibold tracking-tight mt-0.5">
					{getGreeting()}
				</h1>
			</div>
			<button
				class="btn btn-primary gap-1.5 rounded-xl shadow-sm shrink-0"
				onclick={openTaskModal}
			>
				<Plus class="w-4 h-4" strokeWidth={2.5} />
				<span class="hidden sm:inline">New Task</span>
				<span class="sm:hidden">New</span>
			</button>
		</div>

		<!-- Stats & Quick Log Card -->
		<div
			class="card bg-base-100 border border-base-300/60 rounded-2xl shadow-sm overflow-hidden"
		>
			<div class="card-body p-4 sm:p-5 gap-0">
				<!-- Stats Row -->
				<div class="grid grid-cols-2 sm:grid-cols-4 gap-x-3 gap-y-4">
					<!-- Streak -->
					<div class="flex items-center gap-3">
						<div
							class="w-10 h-10 rounded-xl bg-warning/10 text-warning flex items-center justify-center shrink-0"
						>
							<Flame class="w-5 h-5" />
						</div>
						<div class="min-w-0">
							<p class="text-xs font-medium text-base-content/50 leading-tight">Streak</p>
							<p class="text-2xl font-bold tracking-tight leading-tight stat-value mt-0.5">
								7<span class="text-sm font-medium text-base-content/50 ml-0.5">d</span>
							</p>
						</div>
					</div>

					<!-- Tasks -->
					<div class="flex items-center gap-3">
						<div
							class="w-10 h-10 rounded-xl bg-info/10 text-info flex items-center justify-center shrink-0"
						>
							<ListTodo class="w-5 h-5" />
						</div>
						<div class="min-w-0">
							<p class="text-xs font-medium text-base-content/50 leading-tight">Tasks</p>
							<p class="text-2xl font-bold tracking-tight leading-tight stat-value mt-0.5">
								{completedCount}<span class="text-sm font-medium text-base-content/50 ml-0.5">/{totalCount}</span>
							</p>
						</div>
					</div>

					<!-- Goals -->
					<div class="flex items-center gap-3">
						<div
							class="w-10 h-10 rounded-xl bg-accent/10 text-accent flex items-center justify-center shrink-0"
						>
							<Target class="w-5 h-5" />
						</div>
						<div class="min-w-0">
							<p class="text-xs font-medium text-base-content/50 leading-tight">Goals</p>
							<p class="text-2xl font-bold tracking-tight leading-tight stat-value mt-0.5">
								3<span class="text-sm font-medium text-base-content/50 ml-0.5">active</span>
							</p>
						</div>
					</div>

					<!-- Mood -->
					<div class="flex items-center gap-3">
						<div
							class="w-10 h-10 rounded-xl bg-success/10 text-success flex items-center justify-center shrink-0"
						>
							<Smile class="w-5 h-5" />
						</div>
						<div class="min-w-0">
							<p class="text-xs font-medium text-base-content/50 leading-tight">Mood</p>
							<div class="flex items-center gap-1.5 mt-0.5">
								<span class="text-xl leading-none">😊</span>
								<span class="text-sm font-semibold">Great</span>
							</div>
						</div>
					</div>
				</div>

				<div class="border-t border-base-300/60 my-4"></div>

				<!-- Quick Log Row -->
				<div class="flex flex-col gap-3">
					<div class="flex items-center gap-2">
						<Zap class="w-4 h-4 text-primary" />
						<span class="font-semibold text-sm">Quick log</span>
					</div>
					<div class="flex items-center gap-1.5 flex-wrap">
						{#each quickLogs as log}
							<button
								class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-base-300/70 bg-base-100 hover:bg-base-200 hover:border-base-300 transition-colors"
								onclick={() => handleQuickLog(log)}
								title={log.label}
							>
								<span class="text-base leading-none">{log.emoji}</span>
								<span class="text-xs font-medium hidden sm:inline">{log.label}</span>
							</button>
						{/each}
					</div>
				</div>

				<!-- Quick Capture — inline inside top card, only on today's view -->
				{#if isSelectedDateToday()}
					<div class="border-t border-base-300/60 my-4"></div>
					<QuickCapture onSaved={() => {
						$tasksQuery.refetch();
					}} />
				{/if}
				</div>
				</div>
				</div>

	<!-- Timeline Section -->
	<div class="flex-1 min-h-0 flex flex-col">
		<div
			class="card bg-base-100 border border-base-300/60 rounded-2xl shadow-sm h-full flex flex-col overflow-hidden"
		>
			<div class="card-body p-3 lg:p-4 flex flex-col h-full gap-3">
				<!-- View Mode Toggle -->
				<div class="flex items-center justify-between gap-2">
					<!-- Segmented Toggle -->
					<div class="join bg-base-200 rounded-lg p-0.5 shrink-0">
						<button
							class="join-item btn btn-xs {viewMode === 'timeline' ? 'btn-primary' : 'btn-ghost'}"
							onclick={() => setViewMode('timeline')}
							aria-label="Timeline view"
						>
							<GanttChart class="w-3.5 h-3.5" />
							<span class="hidden sm:inline">Timeline</span>
						</button>
						<button
							class="join-item btn btn-xs {viewMode === 'list' ? 'btn-primary' : 'btn-ghost'}"
							onclick={() => setViewMode('list')}
							aria-label="Agenda view"
						>
							<List class="w-3.5 h-3.5" />
							<span class="hidden sm:inline">Agenda</span>
						</button>
					</div>
				</div>

				<!-- Goals strip — informative chips, own row -->
				{#if timelineGoals.length > 0}
					<div class="flex items-center gap-1.5 overflow-x-auto scrollbar-hide -mx-1 px-1">
						<span class="text-[10px] uppercase tracking-wide text-base-content/40 font-semibold shrink-0 mr-0.5">Goals</span>
						{#each timelineGoals as goal}
							<button
								class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-base-200/60 hover:bg-base-200 text-xs font-medium shrink-0 transition-colors"
								onclick={() => handleGoalClick(goal.id)}
							>
								<span class="text-sm leading-none">{goal.icon}</span>
								<span class="truncate max-w-[140px]">{goal.title}</span>
								{#if goal.target}
									<span class="text-[10px] text-base-content/50 tabular-nums">
										{goal.target.currentValue}/{goal.target.value}
									</span>
								{:else if goal.progress > 0}
									<span class="text-[10px] text-base-content/50 tabular-nums">{goal.progress}%</span>
								{/if}
							</button>
						{/each}
					</div>
				{/if}

				<!-- Content -->
				<div class="flex-1 min-h-0">
					{#if $tasksQuery.isLoading}
						<div
							class="flex flex-col items-center justify-center h-full gap-3"
						>
							<span
								class="loading loading-spinner loading-lg text-primary"
							></span>
							<p class="text-sm text-base-content/50">
								Loading your day...
							</p>
						</div>
					{:else if viewMode === 'list'}
							<!-- Agenda/List View: dense day-agenda grouped by time-of-day -->
							<div class="flex-1 overflow-y-auto px-1 pb-2 max-h-full">
								{#if timelineTasks.length === 0}
									<div class="py-12">
										<EmptyState
											title="No tasks yet"
											description="Capture what you just did or plan your day."
											buttonLabel="New Task"
											onButtonClick={openTaskModal}
										/>
									</div>
								{:else}
									{#each agendaGroups as group (group.key)}
										{#if group.tasks.length > 0}
											<div class="px-3 pt-3 pb-1">
												<span class="text-[11px] font-semibold uppercase tracking-wide text-base-content/40">
													{group.label}
												</span>
											</div>
											<div class="divide-y divide-base-200/60">
												{#each group.tasks as task (task.id)}
													<AgendaRow
														{task}
														onclick={() => handleTaskClick(task.id)}
														ontoggle={(completed) => handleToggleComplete(task.id, completed)}
													/>
												{/each}
											</div>
										{/if}
									{/each}
								{/if}
							</div>
					{:else}
						<TimelineGantt
							tasks={timelineTasks}
							goals={timelineGoals}
							{selectedDate}
							showGoals={false}
							onTaskClick={handleTaskClick}
							onCategoryClick={handleCategoryClick}
							onToggleComplete={handleToggleComplete}
							onDateChange={handleDateChange}
							onTaskTimeUpdate={handleTaskTimeUpdate}
							onGoalClick={handleGoalClick}
							onCreateTaskFromGoal={handleCreateTaskFromGoal}
						/>
					{/if}
				</div>
			</div>
		</div>
	</div>

	<!-- Floating Action Button (lifted above mobile tab bar) -->
	<div class="fixed bottom-20 lg:bottom-6 right-4 lg:right-6 z-40">
		<!-- FAB Menu -->
		{#if fabOpen}
			<!-- Backdrop -->
			<button
				class="fixed inset-0 bg-base-content/20 backdrop-blur-sm z-40"
				onclick={() => (fabOpen = false)}
				aria-label="Close menu"
			></button>

			<!-- Menu -->
			<div class="absolute bottom-16 right-0 z-50">
				<div
					class="card bg-base-100 shadow-xl border border-base-300/70 rounded-2xl min-w-[270px] overflow-hidden"
				>
					<div class="card-body p-2.5">
						<!-- Create Task Option -->
						<button
							class="flex items-center gap-3 w-full px-3 py-2.5 rounded-xl hover:bg-base-200 transition-colors text-left"
							onclick={openTaskModal}
						>
							<div
								class="w-9 h-9 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0"
							>
								<ClipboardList class="w-5 h-5" />
							</div>
							<div class="text-left">
								<p class="font-semibold text-sm">New Task</p>
								<p class="text-xs text-base-content/50">
									Create a detailed task
								</p>
							</div>
						</button>

						<div class="border-t border-base-300/60 my-1.5"></div>
						<p class="px-3 pb-1.5 text-[10px] font-semibold uppercase tracking-wider text-base-content/40">
							Quick logs
						</p>

						<!-- Quick Logs Grid -->
						<div class="grid grid-cols-4 gap-1.5">
							{#each quickLogs as log}
								<button
									class="flex flex-col items-center py-2 rounded-lg hover:bg-base-200 transition-colors gap-1"
									onclick={() => handleQuickLog(log)}
								>
									<span class="text-xl leading-none">{log.emoji}</span>
									<span class="text-[9px] font-medium text-base-content/60">{log.label}</span>
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
				"btn btn-circle btn-lg shadow-xl transition-all duration-200",
				fabOpen ? "btn-neutral rotate-45" : "btn-primary hover:scale-105 active:scale-95",
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
