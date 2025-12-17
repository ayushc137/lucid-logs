<script lang="ts">
  import {
    Calendar,
    Target,
    ListTodo,
    Plus,
    Zap,
    Smile,
    Flame,
    TrendingUp,
  } from "lucide-svelte";
  import { Timeline } from "$lib/components/timeline";
  import { TaskModal } from "$lib/components/tasks";
  import { createQuery } from "@tanstack/svelte-query";
  import { getTasks, type Task } from "$lib/api";
  import { getCategories, type Category } from "$lib/api/categories";
  import { cn } from "$lib/utils";

  // Fetch today's tasks
  const today = new Date();
  const startOfDay = new Date(
    today.getFullYear(),
    today.getMonth(),
    today.getDate(),
  ).toISOString();
  const endOfDay = new Date(
    today.getFullYear(),
    today.getMonth(),
    today.getDate() + 1,
  ).toISOString();

  const tasksQuery = createQuery({
    queryKey: ["tasks", "today"],
    queryFn: () =>
      getTasks({ limit: 50, start_date: startOfDay, end_date: endOfDay }),
  });

  const categoriesQuery = createQuery({
    queryKey: ["categories"],
    queryFn: () => getCategories({ limit: 100 }),
  });

  // Filter state
  let selectedCategoryId = $state<string | null>(null);

  // Transform API tasks to timeline format
  type TimelineTask = {
    id: string;
    title: string;
    startHour: number;
    endHour: number;
    color: "primary" | "secondary" | "accent" | "info" | "success" | "warning";
    emoji?: string;
    categoryId?: string;
  };

  function transformTasks(apiTasks: Task[]): TimelineTask[] {
    const colors: TimelineTask["color"][] = [
      "primary",
      "secondary",
      "accent",
      "info",
      "success",
      "warning",
    ];
    return apiTasks.map((task, index) => {
      const start = new Date(task.start_date);
      const end = new Date(task.end_date);
      return {
        id: task.id,
        title: task.title,
        startHour: start.getHours() + start.getMinutes() / 60,
        endHour: end.getHours() + end.getMinutes() / 60,
        color: colors[index % colors.length],
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

  const quickLogs = [
    { emoji: "☕", label: "Morning" },
    { emoji: "💪", label: "Workout" },
    { emoji: "📚", label: "Reading" },
    { emoji: "🧘", label: "Meditation" },
    { emoji: "🍽️", label: "Meal" },
    { emoji: "💤", label: "Sleep" },
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
</script>

<svelte:head>
  <title>Dashboard - Lucid Logs</title>
</svelte:head>

<div class="h-full flex flex-col lg:flex-row gap-4 lg:gap-6">
  <!-- Left Column: Stats + Quick Log -->
  <div class="lg:w-80 xl:w-96 flex-shrink-0 space-y-4">
    <!-- Welcome Card - Subtle design -->
    <div class="card bg-base-100">
      <div class="card-body p-5">
        <p class="text-xs uppercase font-medium opacity-50">{formatDate()}</p>
        <h1 class="text-xl sm:text-2xl font-bold mt-1">{getGreeting()}! 👋</h1>
        <p class="text-sm opacity-60 mt-1">Ready to capture today's moments?</p>
        <button
          class="btn btn-primary w-full mt-4 gap-2"
          onclick={() => (modalOpen = true)}
        >
          <Plus class="w-4 h-4" />
          Start Logging
        </button>
      </div>
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-2 gap-3">
      <div class="stat-card">
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div>
              <p
                class="text-[10px] sm:text-xs font-medium uppercase opacity-50"
              >
                Streak
              </p>
              <p class="text-2xl sm:text-3xl font-bold text-primary">7</p>
            </div>
            <div class="stat-icon bg-primary/10">
              <Flame class="w-5 h-5 text-primary" />
            </div>
          </div>
          <p
            class="text-[10px] sm:text-xs opacity-50 mt-1 flex items-center gap-1"
          >
            <TrendingUp class="w-3 h-3 text-success" />
            days in a row
          </p>
        </div>
      </div>

      <div class="stat-card">
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div>
              <p
                class="text-[10px] sm:text-xs font-medium uppercase opacity-50"
              >
                Tasks
              </p>
              <p class="text-2xl sm:text-3xl font-bold text-info">
                {totalCount}
              </p>
            </div>
            <div class="stat-icon bg-info/10">
              <ListTodo class="w-5 h-5 text-info" />
            </div>
          </div>
          <div class="flex items-center gap-2 mt-1">
            <progress
              class="progress progress-success w-full h-1.5"
              value={completionRate}
              max="100"
            ></progress>
            <span class="text-[10px] font-medium text-success"
              >{completionRate}%</span
            >
          </div>
        </div>
      </div>

      <div class="stat-card">
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div>
              <p
                class="text-[10px] sm:text-xs font-medium uppercase opacity-50"
              >
                Goals
              </p>
              <p class="text-2xl sm:text-3xl font-bold text-warning">--</p>
            </div>
            <div class="stat-icon bg-warning/10">
              <Target class="w-5 h-5 text-warning" />
            </div>
          </div>
          <p class="text-[10px] sm:text-xs opacity-50 mt-1">this week</p>
        </div>
      </div>

      <div class="stat-card">
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div>
              <p
                class="text-[10px] sm:text-xs font-medium uppercase opacity-50"
              >
                Mood
              </p>
              <p class="text-2xl sm:text-3xl">😊</p>
            </div>
            <div class="stat-icon bg-success/10">
              <Smile class="w-5 h-5 text-success" />
            </div>
          </div>
          <p class="text-[10px] sm:text-xs opacity-50 mt-1">feeling great</p>
        </div>
      </div>
    </div>

    <!-- Quick Log -->
    <div class="card bg-base-100">
      <div class="card-body p-4">
        <div class="flex items-center gap-3 mb-4">
          <div class="stat-icon bg-warning/10">
            <Zap class="w-5 h-5 text-warning" />
          </div>
          <div>
            <h3 class="font-semibold text-sm">Quick Log</h3>
            <p class="text-xs opacity-50">One-tap logging</p>
          </div>
        </div>
        <div class="grid grid-cols-3 gap-2">
          {#each quickLogs as log}
            <button
              class="btn btn-ghost btn-sm flex-col h-auto py-3 gap-1 hover:bg-base-200"
            >
              <span class="text-xl">{log.emoji}</span>
              <span class="text-[10px] font-medium opacity-60">{log.label}</span
              >
            </button>
          {/each}
        </div>
      </div>
    </div>
  </div>

  <!-- Right Column: Timeline -->
  <div class="flex-1 min-w-0">
    <div class="card bg-base-100 h-full">
      <div class="card-body p-4 lg:p-6 flex flex-col h-full">
        <div class="flex items-center justify-between mb-4 flex-shrink-0">
          <div class="flex items-center gap-3">
            <div class="stat-icon bg-info/10">
              <Calendar class="w-5 h-5 text-info" />
            </div>
            <div>
              <h3 class="font-semibold text-sm">Today's Timeline</h3>
              <p class="text-xs opacity-50">{totalCount} tasks scheduled</p>
            </div>
          </div>
          <button
            class="btn btn-primary btn-sm gap-1.5"
            onclick={() => (modalOpen = true)}
          >
            <Plus class="w-4 h-4" />
            <span class="hidden sm:inline">Add Task</span>
          </button>
        </div>

        <!-- Category Filters -->
        <div class="flex flex-wrap gap-2 mb-4">
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

        <div class="flex-1 overflow-y-auto min-h-0 -mx-2 px-2">
          {#if $tasksQuery.isLoading}
            <div class="flex flex-col items-center justify-center h-full gap-3">
              <span class="loading loading-spinner loading-lg text-primary"
              ></span>
              <p class="text-sm opacity-50">Loading your day...</p>
            </div>
          {:else}
            <Timeline tasks={timelineTasks} startHour={6} endHour={22} />
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>

<!-- Task Modal -->
<TaskModal bind:open={modalOpen} onClose={() => (modalOpen = false)} />
