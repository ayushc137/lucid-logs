<script lang="ts">
  import {
    Calendar,
    Target,
    ListTodo,
    Plus,
    Zap,
    Smile,
    Flame,
    X,
    ClipboardList,
  } from "lucide-svelte";
  import {
    TimelineMultiView,
    type TimelineView,
  } from "$lib/components/timeline";
  import { AlignHorizontalJustifyStart, List } from "lucide-svelte";
  import {
    TaskModal,
    type EmotionSelectionContext,
  } from "$lib/components/tasks";
  import { EmotionModal } from "$lib/components/emotions";
  import {
    Card,
    IconBox,
    SectionHeader,
    ConfirmDialog,
  } from "$lib/components/ui";
  import {
    createQuery,
    createMutation,
    useQueryClient,
  } from "@tanstack/svelte-query";
  import { getTasks, updateTask, type Task } from "$lib/api";
  import { getCategories, type Category } from "$lib/api/categories";
  import { getEmotion, type Emotion } from "$lib/api/emotions";
  import { cn } from "$lib/utils";
  import { goto } from "$app/navigation";
  import { browser } from "$app/environment";
  import { page } from "$app/stores";
  import {
    getUrlParams,
    updateUrlParams,
    parsers,
  } from "$lib/utils/navigation";

  // Selected date state initialized from URL
  const initialParams = getUrlParams<{ date: Date; view: string }>({
    date: parsers.dateOnly(new Date()),
    view: parsers.string("timeline"),
  });

  let selectedDate = $state<Date>(initialParams.date);
  let currentTimelineView = $state<TimelineView>(
    (initialParams.view as TimelineView) || "timeline",
  );

  function handleViewChange(view: TimelineView) {
    currentTimelineView = view;
  }

  // Sync state to URL
  $effect(() => {
    // Format as YYYY-MM-DD local time
    const year = selectedDate.getFullYear();
    const month = String(selectedDate.getMonth() + 1).padStart(2, "0");
    const day = String(selectedDate.getDate()).padStart(2, "0");

    updateUrlParams({
      date: `${year}-${month}-${day}`,
      view: currentTimelineView,
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
  };

  // Cache for emotion data to avoid refetching
  let emotionCache = new Map<string, Emotion>();
  let emotionEnrichedTasks = $state<TimelineTask[]>([]);

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
        inferredEmotionName: task.inferred_emotion?.closest_emotion_name,
        inferredEmotionId: task.inferred_emotion?.closest_emotion_id,
      };
    });
  }

  // Enrich tasks with emotion data (both selected and inferred)
  async function enrichTasksWithEmotions(
    tasks: TimelineTask[],
  ): Promise<TimelineTask[]> {
    const enrichedTasks = await Promise.all(
      tasks.map(async (task) => {
        let enrichedTask = { ...task };

        // Fetch selected emotion data
        if (task.emotionId) {
          let emotion = emotionCache.get(task.emotionId);
          if (!emotion) {
            try {
              emotion = await getEmotion(task.emotionId);
              emotionCache.set(task.emotionId, emotion);
            } catch (e) {
              console.error(`Failed to fetch emotion ${task.emotionId}:`, e);
            }
          }
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

        // Fetch inferred emotion data
        if (task.inferredEmotionId) {
          let inferredEmotion = emotionCache.get(task.inferredEmotionId);
          if (!inferredEmotion) {
            try {
              inferredEmotion = await getEmotion(task.inferredEmotionId);
              emotionCache.set(task.inferredEmotionId, inferredEmotion);
            } catch (e) {
              console.error(
                `Failed to fetch inferred emotion ${task.inferredEmotionId}:`,
                e,
              );
            }
          }
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
      }),
    );
    return enrichedTasks;
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
  $effect(() => {
    const rawTasks = allTasksRaw;
    enrichTasksWithEmotions(rawTasks).then((enriched) => {
      emotionEnrichedTasks = enriched;
    });
  });

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
      queryClient.invalidateQueries({ queryKey: ["tasks", "timeline-prev"] });
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
      queryClient.invalidateQueries({ queryKey: ["tasks", "timeline-prev"] });
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
        <p class="text-xs uppercase font-medium opacity-50">{formatDate()}</p>
        <h1 class="text-2xl sm:text-3xl font-bold mt-1">{getGreeting()}! 👋</h1>
      </div>
      <div class="flex items-center gap-3">
        <!-- View Toggle moved here -->
        <div class="join border border-base-300/60 rounded-lg bg-base-100 h-10">
          <button
            type="button"
            class="join-item btn btn-sm h-full px-3 sm:px-4 gap-2 {currentTimelineView ===
            'timeline'
              ? 'btn-primary'
              : 'btn-ghost'}"
            onclick={() => handleViewChange("timeline")}
            title="Timeline View"
          >
            <AlignHorizontalJustifyStart class="w-4 h-4" />
            <span class="font-medium hidden sm:inline">Timeline</span>
          </button>
          <button
            type="button"
            class="join-item btn btn-sm h-full px-3 sm:px-4 gap-2 {currentTimelineView ===
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

        <button class="btn btn-primary gap-2 h-10" onclick={openTaskModal}>
          <Plus class="w-4 h-4" />
          <span class="hidden sm:inline">New Task</span>
        </button>
      </div>
    </div>

    <!-- Stats & Quick Log Card -->
    <div class="card bg-base-100 shadow-sm border border-base-200/50">
      <div class="card-body p-4">
        <!-- Stats Row -->
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <!-- Streak -->
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-xl bg-gradient-to-br from-warning to-warning/70 flex items-center justify-center text-warning-content shadow-sm"
            >
              <Flame class="w-5 h-5" />
            </div>
            <div>
              <p class="text-xs font-medium uppercase opacity-50">Streak</p>
              <p class="text-xl font-bold text-warning">
                7<span class="text-sm font-medium opacity-60 ml-0.5">d</span>
              </p>
            </div>
          </div>

          <!-- Tasks -->
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-xl bg-gradient-to-br from-info to-info/70 flex items-center justify-center text-info-content shadow-sm"
            >
              <ListTodo class="w-5 h-5" />
            </div>
            <div>
              <p class="text-xs font-medium uppercase opacity-50">Tasks</p>
              <p class="text-xl font-bold text-info">
                {completedCount}<span
                  class="text-sm font-medium opacity-60 ml-0.5"
                  >/{totalCount}</span
                >
              </p>
            </div>
          </div>

          <!-- Goals -->
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-xl bg-gradient-to-br from-accent to-accent/70 flex items-center justify-center text-accent-content shadow-sm"
            >
              <Target class="w-5 h-5" />
            </div>
            <div>
              <p class="text-xs font-medium uppercase opacity-50">Goals</p>
              <p class="text-xl font-bold text-accent">
                3<span class="text-sm font-medium opacity-60 ml-0.5"
                  >active</span
                >
              </p>
            </div>
          </div>

          <!-- Mood -->
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-xl bg-gradient-to-br from-success to-success/70 flex items-center justify-center text-success-content shadow-sm"
            >
              <Smile class="w-5 h-5" />
            </div>
            <div>
              <p class="text-xs font-medium uppercase opacity-50">Mood</p>
              <div class="flex items-center gap-1.5">
                <span class="text-lg">😊</span>
                <span class="text-sm font-bold text-success">Great</span>
              </div>
            </div>
          </div>
        </div>

        <div class="divider my-2"></div>

        <!-- Quick Log Row -->
        <div class="flex items-center justify-between gap-4">
          <div class="flex items-center gap-2.5">
            <div
              class="w-9 h-9 rounded-lg bg-gradient-to-br from-secondary to-secondary/70 flex items-center justify-center text-secondary-content shadow-sm"
            >
              <Zap class="w-4 h-4" />
            </div>
            <span class="font-semibold text-sm">Quick Log</span>
          </div>
          <div class="flex items-center gap-1.5 flex-wrap">
            {#each quickLogs as log}
              <button
                class="btn btn-ghost btn-sm gap-1.5 px-2.5 hover:bg-base-200"
                onclick={() => handleQuickLog(log)}
                title={log.label}
              >
                <span class="text-base">{log.emoji}</span>
                <span class="text-xs font-medium hidden sm:inline"
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
      class="card bg-base-100 shadow-sm border border-base-200/50 h-full flex flex-col"
    >
      <div class="card-body p-3 lg:p-4 flex flex-col h-full gap-3">
        <!-- Timeline -->

        <!-- Timeline -->
        <div class="flex-1 min-h-0">
          {#if $tasksQuery.isLoading}
            <div class="flex flex-col items-center justify-center h-full gap-3">
              <span class="loading loading-spinner loading-lg text-primary"
              ></span>
              <p class="text-sm opacity-50">Loading your day...</p>
            </div>
          {:else}
            <TimelineMultiView
              tasks={timelineTasks}
              {selectedDate}
              currentView={currentTimelineView}
              onTaskClick={handleTaskClick}
              onCategoryClick={handleCategoryClick}
              onToggleComplete={handleToggleComplete}
              onDateChange={handleDateChange}
              onTaskTimeUpdate={handleTaskTimeUpdate}
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
          class="card bg-base-100 shadow-2xl border border-base-300 min-w-[280px]"
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
                <p class="text-xs opacity-50">Create a detailed task</p>
              </div>
            </button>

            <div class="divider my-1 text-[10px] uppercase opacity-50">
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
                  <span class="text-[9px] font-medium opacity-60"
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
