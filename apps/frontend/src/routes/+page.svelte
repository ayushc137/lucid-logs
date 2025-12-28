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
  import { Timeline } from "$lib/components/timeline";
  import { TaskModal } from "$lib/components/tasks";
  import {
    Card,
    IconBox,
    SectionHeader,
    ConfirmDialog,
  } from "$lib/components/ui";
  import { createQuery, createMutation } from "@tanstack/svelte-query";
  import { getTasks, updateTask, type Task } from "$lib/api";
  import { getCategories, type Category } from "$lib/api/categories";
  import { cn } from "$lib/utils";

  // Fetch today's tasks - use YYYY-MM-DD format for consistent filtering
  const today = new Date();
  const year = today.getFullYear();
  const month = String(today.getMonth() + 1).padStart(2, "0");
  const day = String(today.getDate()).padStart(2, "0");
  const startOfDay = `${year}-${month}-${day}T00:00:00Z`;
  const endOfDay = `${year}-${month}-${day}T23:59:59Z`;

  const tasksQuery = createQuery({
    queryKey: ["tasks", "today"],
    queryFn: () =>
      getTasks({
        limit: 50,
        start_date_from: startOfDay,
        start_date_to: endOfDay,
      }),
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
      return {
        id: task.id,
        title: task.title,
        startTime: new Date(task.start_date),
        endTime: new Date(task.end_date),
        categoryColor: task.category?.color,
        categoryName: task.category?.name,
        completed: task.completed,
        emoji: task.completed ? "✓" : undefined,
        categoryId: task.category?.id,
      };
    });
  }

  const allTasks = $derived(transformTasks($tasksQuery.data?.items || []));
  const timelineTasks = $derived(
    selectedCategoryId
      ? allTasks.filter((t) => t.categoryId === selectedCategoryId)
      : allTasks,
  );

  const categories = $derived($categoriesQuery.data?.items || []);
  const completedCount = $derived(
    ($tasksQuery.data?.items || []).filter((t) => t.completed).length,
  );
  const totalCount = $derived($tasksQuery.data?.items?.length || 0);
  const completionRate = $derived(
    totalCount > 0 ? Math.round((completedCount / totalCount) * 100) : 0,
  );

  // Task modal state
  let modalOpen = $state(false);
  let editingTask = $state<Task | null>(null);

  // Uncomplete confirmation state
  let showUncompleteConfirm = $state(false);
  let pendingUncompleteTaskId = $state<string | null>(null);

  // Toggle complete mutation
  const toggleCompleteMut = createMutation({
    mutationFn: ({ id, completed }: { id: string; completed: boolean }) =>
      updateTask(id, { completed }),
    onSuccess: () => {
      $tasksQuery.refetch();
    },
  });

  // Handle task click from timeline to open edit modal
  function handleTaskClick(taskId: string) {
    const task = $tasksQuery.data?.items?.find((t) => t.id === taskId);
    if (task) {
      editingTask = task;
      modalOpen = true;
    }
  }

  // Handle modal close
  function handleModalClose() {
    modalOpen = false;
    editingTask = null;
    $tasksQuery.refetch();
  }

  // Handle toggle complete from timeline
  function handleToggleComplete(taskId: string, newCompleted: boolean) {
    if (!newCompleted) {
      // Uncompleting - show confirmation
      pendingUncompleteTaskId = taskId;
      showUncompleteConfirm = true;
    } else {
      // Completing - just do it
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
    modalOpen = true;
  }
</script>

<svelte:head>
  <title>Dashboard - Lucid Logs</title>
</svelte:head>

<div class="h-full flex flex-col gap-4 lg:gap-6 relative">
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
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body p-4">
        <!-- Stats Row -->
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <!-- Streak -->
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-xl bg-warning flex items-center justify-center text-warning-content"
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
              class="w-10 h-10 rounded-xl bg-info flex items-center justify-center text-info-content"
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
              class="w-10 h-10 rounded-xl bg-accent flex items-center justify-center text-accent-content"
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
              class="w-10 h-10 rounded-xl bg-success flex items-center justify-center text-success-content"
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
              class="w-9 h-9 rounded-lg bg-secondary flex items-center justify-center text-secondary-content"
            >
              <Zap class="w-4 h-4" />
            </div>
            <span class="font-semibold text-sm">Quick Log</span>
          </div>
          <div class="flex items-center gap-2 flex-wrap">
            {#each quickLogs as log}
              <button
                class="btn btn-ghost btn-sm gap-1.5 px-3"
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

  <!-- Timeline Section - Full width below -->
  <div class="flex-1 min-h-0">
    <div class="card bg-base-100 shadow-sm h-full">
      <div class="card-body p-4 lg:p-5 flex flex-col h-full">
        <!-- Timeline Header -->
        <div class="flex items-center justify-between mb-3 flex-shrink-0">
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-xl bg-info/10 flex items-center justify-center"
            >
              <Calendar class="w-5 h-5 text-info" />
            </div>
            <div>
              <h3 class="font-semibold text-sm">Today's Timeline</h3>
              <p class="text-xs opacity-50">{totalCount} tasks scheduled</p>
            </div>
          </div>

          <!-- Category Filters -->
          <div class="flex flex-wrap gap-2">
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
                  class="w-2 h-2 rounded-full"
                  style="background-color: {category.color};"
                ></span>
                {category.name}
              </button>
            {/each}
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
            <Timeline
              tasks={timelineTasks}
              startHour={0}
              endHour={24}
              onTaskClick={handleTaskClick}
              onToggleComplete={handleToggleComplete}
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
  onClose={handleModalClose}
/>

<!-- Uncomplete Confirmation Dialog -->
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

<style>
  @keyframes fade-in {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .animate-fade-in {
    animation: fade-in 0.2s ease-out;
  }
</style>
