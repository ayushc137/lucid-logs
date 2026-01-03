<script lang="ts">
  import {
    createQuery,
    createMutation,
    useQueryClient,
  } from "@tanstack/svelte-query";
  import { getGoals, deleteGoal, updateGoal, type Goal } from "$lib/api";
  import { ErrorAlert, ConfirmDialog } from "$lib/components/ui";
  import { GoalModal } from "$lib/components/goals";
  import {
    Target,
    Plus,
    Trash2,
    SquarePen,
    Check,
    X,
    Search,
    LoaderCircle,
    Flame,
    Pause,
    Ban,
    ChevronUp,
    ChevronDown,
    ArrowUpDown,
    Filter,
    Repeat,
    Crosshair,
    BarChart3,
    Layers,
    Shield,
  } from "lucide-svelte";
  import { cn } from "$lib/utils";
  import {
    getUrlParams,
    updateUrlParams,
    parsers,
  } from "$lib/utils/navigation";

  const queryClient = useQueryClient();

  // Initialize state from URL
  const initialParams = getUrlParams({
    q: parsers.string(""),
    status: parsers.string(""),
    type: parsers.string(""),
    sort: parsers.string("created_at-desc"),
  });

  // Search state
  let searchQuery = $state(initialParams.q);
  let debouncedSearch = $state(initialParams.q);
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;

  // Filter state
  let statusFilter = $state(initialParams.status);
  let typeFilter = $state(initialParams.type);
  let sortBy = $state(initialParams.sort);

  function handleSearchInput(value: string) {
    searchQuery = value;
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      debouncedSearch = value;
    }, 300);
  }

  // Sync state to URL
  $effect(() => {
    updateUrlParams(
      {
        q: debouncedSearch,
        status: statusFilter,
        type: typeFilter,
        sort: sortBy,
      },
      { replace: true, keepFocus: true },
    );
  });

  // Query params
  const queryParams = $derived({
    limit: 100,
    search: debouncedSearch || undefined,
    status: statusFilter || undefined,
    goal_type: typeFilter || undefined,
    sort_by: sortBy || undefined,
  });

  // Mutable ref for tracking param changes
  let currentParamsJson = "";

  // Goals query
  const query = createQuery({
    queryKey: ["goals-list"],
    queryFn: () => getGoals(queryParams),
    retry: false,
  });

  // Refetch when params change
  $effect(() => {
    const newParamsJson = JSON.stringify(queryParams);
    if (currentParamsJson === "") {
      currentParamsJson = newParamsJson;
      return;
    }
    if (newParamsJson !== currentParamsJson) {
      currentParamsJson = newParamsJson;
      $query.refetch();
    }
  });

  const goals = $derived($query.data?.items || []);
  const totalGoals = $derived($query.data?.total || 0);

  // Modal states
  let modalOpen = $state(false);
  let editingGoal = $state<Goal | null>(null);
  let deleteConfirmOpen = $state(false);
  let deleteTargetId = $state<string | null>(null);

  // Mutations
  const deleteMut = createMutation({
    mutationFn: (id: string) => deleteGoal(id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["goals-list"] });
      await queryClient.invalidateQueries({ queryKey: ["goals"] });
      deleteConfirmOpen = false;
      deleteTargetId = null;
    },
  });

  const statusMut = createMutation({
    mutationFn: ({
      id,
      status,
    }: {
      id: string;
      status: "active" | "completed" | "paused" | "abandoned";
    }) => updateGoal(id, { status }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["goals-list"] });
      await queryClient.invalidateQueries({ queryKey: ["goals"] });
    },
  });

  function handleCreate() {
    editingGoal = null;
    modalOpen = true;
  }

  function startEdit(goal: Goal) {
    editingGoal = goal;
    modalOpen = true;
  }

  function handleModalClose() {
    modalOpen = false;
    editingGoal = null;
    queryClient.invalidateQueries({ queryKey: ["goals-list"] });
    queryClient.invalidateQueries({ queryKey: ["goals"] });
  }

  function confirmDelete(id: string, e: Event) {
    e.stopPropagation();
    deleteTargetId = id;
    deleteConfirmOpen = true;
  }

  function handleDelete() {
    if (deleteTargetId) {
      $deleteMut.mutate(deleteTargetId);
    }
  }

  function toggleSort(field: string) {
    if (sortBy === `${field}-desc`) {
      sortBy = `${field}-asc`;
    } else if (sortBy === `${field}-asc`) {
      sortBy = `${field}-desc`;
    } else {
      sortBy = `${field}-desc`;
    }
  }

  function getSortIcon(field: string) {
    if (sortBy === `${field}-desc`) return ChevronDown;
    if (sortBy === `${field}-asc`) return ChevronUp;
    return ArrowUpDown;
  }

  const isSearching = $derived(searchQuery !== debouncedSearch);

  const statusOptions = [
    { value: "", label: "All Statuses" },
    { value: "active", label: "Active" },
    { value: "completed", label: "Completed" },
    { value: "paused", label: "Paused" },
    { value: "abandoned", label: "Abandoned" },
  ];

  const typeOptions = [
    { value: "", label: "All Types" },
    { value: "discrete", label: "One-time" },
    { value: "measurable", label: "Measurable" },
    { value: "avoidance", label: "Avoidance" },
    { value: "epic", label: "Epic" },
  ];

  const typeIcons: Record<string, typeof Target> = {
    discrete: Crosshair,
    measurable: BarChart3,
    epic: Layers,
    avoidance: Shield,
  };

  const statusColors: Record<string, string> = {
    active: "badge-success",
    completed: "badge-info",
    paused: "badge-warning",
    abandoned: "badge-error",
  };

  function getProgress(goal: Goal): number {
    if (!goal.target) return 0;
    return Math.min(
      100,
      Math.round((goal.target.current_value / goal.target.value) * 100),
    );
  }

  function formatRecurrence(rec: Goal["recurrence"]): string {
    if (!rec) return "—";
    const times = rec.frequency === 1 ? "" : `${rec.frequency}× `;
    return `${times}${rec.period}ly`;
  }

  // Highlight search matches
  function highlightText(text: string, query: string): string {
    if (!query || !text) return text;
    const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const regex = new RegExp(`(${escapedQuery})`, "gi");
    return text.replace(
      regex,
      '<mark class="bg-warning text-warning-content rounded px-0.5">$1</mark>',
    );
  }
</script>

<svelte:head>
  <title>Goals - Lucid Logs</title>
</svelte:head>

<div class="max-w-6xl mx-auto space-y-6">
  <!-- Header -->
  <div
    class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
  >
    <div>
      <h1 class="text-3xl font-bold">Goals</h1>
      <p class="text-sm opacity-60 mt-1">
        {#if debouncedSearch || statusFilter || typeFilter}
          Found {goals.length} matching goals
        {:else}
          {totalGoals} goals
        {/if}
      </p>
    </div>
    <button
      class="btn btn-primary gap-2 shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-all"
      onclick={handleCreate}
    >
      <Plus class="w-4 h-4" />
      New Goal
    </button>
  </div>

  <!-- Search & Filters Card -->
  <div class="card bg-base-100 shadow-lg border border-base-200">
    <div class="card-body p-4">
      <div class="flex flex-col sm:flex-row gap-3">
        <!-- Search -->
        <div class="relative flex-1">
          {#if isSearching || $query.isFetching}
            <LoaderCircle
              class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-50 animate-spin"
            />
          {:else}
            <Search
              class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-50"
            />
          {/if}
          <input
            type="text"
            placeholder="Search goals..."
            class="input input-bordered w-full pl-10"
            value={searchQuery}
            oninput={(e) => handleSearchInput(e.currentTarget.value)}
          />
          {#if searchQuery}
            <button
              class="absolute right-3 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle"
              onclick={() => {
                searchQuery = "";
                debouncedSearch = "";
              }}
            >
              <X class="w-3 h-3" />
            </button>
          {/if}
        </div>

        <!-- Status Filter -->
        <select
          class="select select-bordered w-full sm:w-40"
          bind:value={statusFilter}
        >
          {#each statusOptions as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <!-- Type Filter -->
        <select
          class="select select-bordered w-full sm:w-40"
          bind:value={typeFilter}
        >
          {#each typeOptions as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>
      </div>
    </div>
  </div>

  <!-- Goals Table -->
  {#if $query.isLoading}
    <div class="card bg-base-100 shadow-lg border border-base-200">
      <div class="card-body py-16 text-center">
        <span class="loading loading-spinner loading-lg text-primary mx-auto"
        ></span>
        <p class="opacity-60 mt-4">Loading goals...</p>
      </div>
    </div>
  {:else if $query.isError}
    <ErrorAlert
      message="Failed to load goals. Please try again."
      onRetry={() => $query.refetch()}
    />
  {:else if goals.length === 0}
    <div class="card bg-base-100 shadow-lg border border-base-200">
      <div class="card-body py-12 text-center">
        {#if debouncedSearch || statusFilter || typeFilter}
          <Search class="w-12 h-12 opacity-30 mx-auto mb-4" />
          <h2 class="text-xl font-semibold">No matching goals</h2>
          <p class="opacity-60 mt-1">Try adjusting your filters</p>
          <button
            class="btn btn-ghost btn-sm mt-4"
            onclick={() => {
              searchQuery = "";
              debouncedSearch = "";
              statusFilter = "";
              typeFilter = "";
            }}
          >
            Clear filters
          </button>
        {:else}
          <div
            class="w-20 h-20 rounded-full bg-accent/10 flex items-center justify-center mx-auto mb-4"
          >
            <Target class="w-10 h-10 text-accent" />
          </div>
          <h2 class="text-2xl font-bold">No goals yet</h2>
          <p class="opacity-60 mt-2 max-w-md mx-auto">
            Set goals to track your progress and stay motivated
          </p>
          <button
            class="btn btn-primary mt-6 gap-2 shadow-lg shadow-primary/20"
            onclick={handleCreate}
          >
            <Plus class="w-4 h-4" />
            Create Your First Goal
          </button>
        {/if}
      </div>
    </div>
  {:else}
    <div
      class="card bg-base-100 shadow-lg border border-base-200 overflow-hidden"
    >
      <div class="overflow-x-auto">
        <table class="table table-lg">
          <thead class="bg-base-100 border-b border-base-300">
            <tr>
              <th class="w-12"></th>
              <th>
                <button
                  class="flex items-center gap-1 hover:text-primary transition-colors"
                  onclick={() => toggleSort("title")}
                >
                  Goal
                  <svelte:component
                    this={getSortIcon("title")}
                    class="w-3 h-3 opacity-50"
                  />
                </button>
              </th>
              <th class="hidden md:table-cell">Type</th>
              <th class="hidden lg:table-cell">Recurrence</th>
              <th class="hidden sm:table-cell">
                <button
                  class="flex items-center gap-1 hover:text-primary transition-colors"
                  onclick={() => toggleSort("streak")}
                >
                  Streak
                  <svelte:component
                    this={getSortIcon("streak")}
                    class="w-3 h-3 opacity-50"
                  />
                </button>
              </th>
              <th>Progress</th>
              <th class="hidden sm:table-cell">Status</th>
              <th class="w-24 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each goals as goal (goal.id)}
              {@const progress = getProgress(goal)}
              {@const TypeIcon = typeIcons[goal.goal_type] || Target}
              {@const goalColor = goal.color || "#8b5cf6"}
              <tr
                class="relative hover:bg-base-200/50 transition-colors group cursor-pointer overflow-hidden"
                onclick={() => startEdit(goal)}
              >
                <!-- Full-width progress bar background -->
                {#if goal.target && progress > 0}
                  <td
                    class="absolute inset-0 p-0 pointer-events-none"
                    style="background: transparent;"
                  >
                    <div
                      class="absolute inset-y-0 left-0 opacity-10 transition-all duration-500"
                      style="width: {progress}%; background-color: {goalColor};"
                    ></div>
                  </td>
                {/if}

                <!-- Icon -->
                <td class="relative z-10">
                  <div
                    class="w-10 h-10 rounded-lg flex items-center justify-center text-lg"
                    style="background-color: {goalColor}20;"
                  >
                    {goal.icon || "🎯"}
                  </div>
                </td>

                <!-- Title -->
                <td class="relative z-10">
                  <div class="flex flex-col gap-0.5">
                    <span class="font-semibold">
                      {#if debouncedSearch}
                        {@html highlightText(goal.title, debouncedSearch)}
                      {:else}
                        {goal.title}
                      {/if}
                    </span>
                    {#if goal.description}
                      <span class="text-xs opacity-50 truncate max-w-[200px]">
                        {#if debouncedSearch}
                          {@html highlightText(
                            goal.description,
                            debouncedSearch,
                          )}
                        {:else}
                          {goal.description}
                        {/if}
                      </span>
                    {/if}
                  </div>
                </td>

                <!-- Type -->
                <td class="hidden md:table-cell relative z-10">
                  <div class="flex items-center gap-1.5">
                    <TypeIcon class="w-4 h-4 opacity-50" />
                    <span class="text-sm capitalize">{goal.goal_type}</span>
                  </div>
                </td>

                <!-- Recurrence -->
                <td class="hidden lg:table-cell relative z-10">
                  {#if goal.recurrence}
                    <div class="flex items-center gap-1.5">
                      <Repeat class="w-3.5 h-3.5 opacity-50" />
                      <span class="text-sm"
                        >{formatRecurrence(goal.recurrence)}</span
                      >
                    </div>
                  {:else}
                    <span class="text-sm opacity-40">—</span>
                  {/if}
                </td>

                <!-- Streak -->
                <td class="hidden sm:table-cell relative z-10">
                  {#if goal.current_streak > 0}
                    <div class="flex items-center gap-1 text-warning font-bold">
                      <Flame class="w-4 h-4" />
                      <span>{goal.current_streak}</span>
                    </div>
                  {:else}
                    <span class="text-sm opacity-40">—</span>
                  {/if}
                </td>

                <!-- Progress -->
                <td class="relative z-10">
                  {#if goal.target}
                    <div class="flex items-center gap-2">
                      <div
                        class="w-24 h-2 bg-base-300 rounded-full overflow-hidden"
                      >
                        <div
                          class="h-full rounded-full transition-all duration-500"
                          style="width: {progress}%; background-color: {goal.color ||
                            '#8b5cf6'};"
                        ></div>
                      </div>
                      <span class="text-xs font-medium w-10">
                        {progress}%
                      </span>
                    </div>
                  {:else}
                    <span class="text-sm opacity-40">—</span>
                  {/if}
                </td>

                <!-- Status -->
                <td class="hidden sm:table-cell relative z-10">
                  <span
                    class={cn(
                      "badge badge-sm",
                      statusColors[goal.status] || "badge-ghost",
                    )}
                  >
                    {goal.status}
                  </span>
                </td>

                <!-- Actions -->
                <td class="relative z-10">
                  <div
                    class="flex gap-1 justify-end opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <button
                      class="btn btn-ghost btn-sm btn-square"
                      onclick={(e) => {
                        e.stopPropagation();
                        startEdit(goal);
                      }}
                    >
                      <SquarePen class="w-4 h-4" />
                    </button>
                    <button
                      class="btn btn-ghost btn-sm btn-square text-error"
                      onclick={(e) => confirmDelete(goal.id, e)}
                    >
                      <Trash2 class="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Footer -->
      <div
        class="px-4 py-3 bg-base-200/50 border-t border-base-200 text-sm opacity-60 flex items-center justify-between"
      >
        <span>{goals.length} goals</span>
        {#if $query.isFetching}
          <span class="flex items-center gap-2 text-primary">
            <LoaderCircle class="w-4 h-4 animate-spin" />
            Loading...
          </span>
        {/if}
      </div>
    </div>
  {/if}
</div>

<!-- Goal Modal -->
<GoalModal
  bind:open={modalOpen}
  goal={editingGoal}
  onClose={handleModalClose}
/>

<!-- Delete Confirmation Modal -->
<ConfirmDialog
  bind:open={deleteConfirmOpen}
  title="Delete Goal"
  message="Are you sure you want to delete this goal? This action cannot be undone. All progress and streaks will be lost."
  confirmText="Delete"
  destructive={true}
  loading={$deleteMut.isPending}
  onConfirm={handleDelete}
  onCancel={() => {
    deleteConfirmOpen = false;
    deleteTargetId = null;
  }}
/>
