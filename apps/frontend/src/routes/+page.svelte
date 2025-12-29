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
  import { TaskModal } from "$lib/components/tasks";
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
  import { cn } from "$lib/utils";

  // Selected date state
  let selectedDate = $state(new Date());

  // Helper to format date for API (YYYY-MM-DD)
  function formatDateForApi(date: Date): {
    startOfDay: string;
    endOfDay: string;
  } {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return {
      startOfDay: `${year}-${month}-${day}T00:00:00Z`,
      endOfDay: `${year}-${month}-${day}T23:59:59Z`,
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

  // Timeline view state
  let currentTimelineView = $state<TimelineView>("timeline");

  function handleViewChange(view: TimelineView) {
    currentTimelineView = view;
    if (typeof window !== "undefined") {
      localStorage.setItem("timeline-view-preference", view);
    }
  }

  // FAB menu state
  let fabOpen = $state(false);

  // Transform API tasks to timeline format
  type TimelineTask = {
    id: string;
    title: string;
    startTime: Date;
    endTime: Date;
    categoryColor?: string;
    categoryName?: string;
    completed?: boolean;
    emoji?: string;
    categoryId?: string;
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
        startTime,
        endTime,
        categoryColor: task.category?.color,
        categoryName: task.category?.name,
        completed: task.completed,
        emoji: task.completed ? "✓" : undefined,
        categoryId: task.category?.id,
      };
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
  const allTasks = $derived(() => {
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

  // Get the end time of the last task that ended before now (for "start from last task" feature)
  const lastTaskEndTime = $derived(() => {
    const tasks = allTasks();
    if (tasks.length === 0) return null;

    const now = new Date();

    // Filter to only tasks that have ended before now
    const pastTasks = tasks.filter((t) => t.endTime.getTime() <= now.getTime());
    if (pastTasks.length === 0) return null;

    // Sort by end time and get the most recent
    const sorted = [...pastTasks].sort(
      (a, b) => b.endTime.getTime() - a.endTime.getTime(),
    );
    return sorted[0]?.endTime || null;
  });

  // Task modal state
  let modalOpen = $state(false);
  let editingTask = $state<Task | null>(null);
  let initialCategoryId = $state<string | undefined>(undefined);

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
      queryClient.invalidateQueries({ queryKey: ["tasks", "timeline"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "timeline-prev"] });
    },
  });

  // Handle date change from Timeline
  function handleDateChange(newDate: Date) {
    selectedDate = newDate;
    selectedCategoryId = null; // Reset filter on date change
  }

  // Handle task click from timeline to open edit modal
  function handleTaskClick(taskId: string) {
    const allTasksList = allTasks();
    const timelineTask = allTasksList.find((t) => t.id === taskId);
    if (timelineTask) {
      // Find the original API task
      const apiTask = [
        ...($tasksQuery.data?.items || []),
        ...($prevDayTasksQuery.data?.items || []),
      ].find((t) => t.id === taskId);
      if (apiTask) {
        editingTask = apiTask;
        initialCategoryId = undefined;
        modalOpen = true;
      }
    }
  }

  // Handle category label click from timeline
  function handleCategoryClick(categoryId: string) {
    editingTask = null;
    initialCategoryId = categoryId;
    modalOpen = true;
  }

  // Handle modal close
  function handleModalClose() {
    modalOpen = false;
    editingTask = null;
    initialCategoryId = undefined;
    queryClient.invalidateQueries({ queryKey: ["tasks", "timeline"] });
    queryClient.invalidateQueries({ queryKey: ["tasks", "timeline-prev"] });
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
    editingTask = null;
    initialCategoryId = undefined;
    modalOpen = true;
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
      <div class="flex items-center gap-2">
        <button class="btn btn-primary gap-2" onclick={openTaskModal}>
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
        <!-- Filter Row with View Toggle -->
        <div class="flex items-center justify-between gap-2 flex-shrink-0">
          <!-- Category Filters -->
          <div class="flex items-center gap-2 flex-wrap">
            <span class="text-xs font-semibold uppercase text-base-content/50"
              >Filter:</span
            >
            <button
              class={cn(
                "btn btn-xs",
                selectedCategoryId === null ? "btn-primary" : "btn-ghost",
              )}
              onclick={() => (selectedCategoryId = null)}
            >
              All
            </button>
            {#each categories as category}
              <button
                class={cn(
                  "btn btn-xs gap-1.5",
                  selectedCategoryId === category.id
                    ? "btn-primary btn-outline"
                    : "btn-ghost",
                )}
                onclick={() => (selectedCategoryId = category.id)}
              >
                <span
                  class="w-2 h-2 rounded-full ring-1 ring-black/10"
                  style="background-color: {category.color};"
                ></span>
                {category.name}
              </button>
            {/each}
          </div>

          <!-- View Toggle -->
          <div class="join border border-base-300/60 rounded-lg">
            <div class="tooltip tooltip-bottom" data-tip="Timeline View">
              <button
                type="button"
                class="join-item btn btn-xs btn-square {currentTimelineView ===
                'timeline'
                  ? 'btn-primary'
                  : 'btn-ghost'}"
                onclick={() => handleViewChange("timeline")}
              >
                <AlignHorizontalJustifyStart class="w-3.5 h-3.5" />
              </button>
            </div>
            <div class="tooltip tooltip-bottom" data-tip="Agenda View">
              <button
                type="button"
                class="join-item btn btn-xs btn-square {currentTimelineView ===
                'agenda'
                  ? 'btn-primary'
                  : 'btn-ghost'}"
                onclick={() => handleViewChange("agenda")}
              >
                <List class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>

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
  lastTaskEndTime={lastTaskEndTime()}
  onClose={handleModalClose}
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
