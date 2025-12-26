<script lang="ts">
  import {
    ListTodo,
    Plus,
    Trash2,
    Edit,
    Check,
    Search,
    Filter,
    ChevronUp,
    ChevronDown,
    X,
    Calendar,
    Clock,
    Tag,
    ArrowUpDown,
    CircleDot,
    Loader2,
    CalendarDays,
  } from "lucide-svelte";
  import {
    createQuery,
    createMutation,
    useQueryClient,
  } from "@tanstack/svelte-query";
  import {
    getTasks,
    deleteTask,
    updateTask,
    getCategories,
    type Task,
    type Category,
    type TaskFilterParams,
  } from "$lib/api";
  import { TaskModal } from "$lib/components/tasks";
  import { cn } from "$lib/utils";

  const queryClient = useQueryClient();

  // Today's date for filtering
  const today = new Date();
  const year = today.getFullYear();
  const month = String(today.getMonth() + 1).padStart(2, "0");
  const day = String(today.getDate()).padStart(2, "0");
  const todayStart = `${year}-${month}-${day}T00:00:00Z`;
  const todayEnd = `${year}-${month}-${day}T23:59:59Z`;

  // Search and filter states
  let searchQuery = $state("");
  let debouncedSearch = $state("");
  let filterCategory = $state<string>("all");
  let filterStatus = $state<"all" | "completed" | "pending">("all");
  let filterPriority = $state<"all" | "high" | "medium" | "low">("all");
  let showFilters = $state(false);

  // Today only toggle - ON by default
  let todayOnly = $state(true);

  // Date range filter (when todayOnly is off)
  let dateFrom = $state("");
  let dateTo = $state("");

  // Sort states
  type SortField = "title" | "start_date" | "priority" | "created_at";
  type SortDirection = "asc" | "desc";
  let sortField = $state<SortField>("start_date");
  let sortDirection = $state<SortDirection>("desc");

  // Debounce search input
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;
  function handleSearchInput(value: string) {
    searchQuery = value;
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      debouncedSearch = value;
      // No need to manually trigger - $effect watching queryParams handles it
    }, 300);
  }

  // Compute query params reactively with $derived
  // This ensures all filter/search state is properly tracked
  const queryParams = $derived.by(() => {
    const params: TaskFilterParams = {
      limit: 100,
      sort_field: sortField,
      sort_order: sortDirection,
    };

    if (todayOnly) {
      params.start_date_from = todayStart;
      params.start_date_to = todayEnd;
    } else {
      if (dateFrom) params.start_date_from = `${dateFrom}T00:00:00Z`;
      if (dateTo) params.start_date_to = `${dateTo}T23:59:59Z`;
    }

    if (debouncedSearch) params.search = debouncedSearch;
    if (filterCategory !== "all") params.category_id = filterCategory;
    if (filterStatus !== "all") params.status = filterStatus;

    if (filterPriority === "high") {
      params.priority_min = 7;
      params.priority_max = 10;
    } else if (filterPriority === "medium") {
      params.priority_min = 4;
      params.priority_max = 6;
    } else if (filterPriority === "low") {
      params.priority_min = 1;
      params.priority_max = 3;
    }

    return params;
  });

  // Mutable refs for tracking param changes - initialized lazily
  let currentParams: TaskFilterParams | null = null;
  let currentParamsJson = "";

  // Fetch tasks - queryFn reads currentParams which is updated by $effect
  const tasksQuery = createQuery({
    queryKey: ["tasks-list"],
    queryFn: async () => {
      // Use currentParams if set, otherwise use queryParams directly
      const params = currentParams ?? queryParams;
      console.log("[Tasks] Fetching with params:", params);
      return getTasks(params);
    },
    staleTime: 0,
  });

  // Use $effect to trigger refetch when queryParams change
  // Update currentParams before calling refetch so queryFn sees fresh values
  $effect(() => {
    const newParamsJson = JSON.stringify(queryParams);
    // Skip initial run (first time currentParamsJson is empty)
    if (currentParamsJson === "") {
      currentParamsJson = newParamsJson;
      currentParams = queryParams;
      return;
    }
    if (newParamsJson !== currentParamsJson) {
      currentParams = queryParams;
      currentParamsJson = newParamsJson;
      console.log("[Tasks] Params changed, refetching...", queryParams);
      $tasksQuery.refetch();
    }
  });

  // Fetch categories
  const categoriesQuery = createQuery({
    queryKey: ["categories"],
    queryFn: () => getCategories({ limit: 50 }),
  });

  // Delete mutation
  const deleteMut = createMutation({
    mutationFn: (id: string) => deleteTask(id),
    onSuccess: () => {
      $tasksQuery.refetch();
    },
  });

  // Toggle complete mutation
  const toggleCompleteMut = createMutation({
    mutationFn: ({ id, completed }: { id: string; completed: boolean }) =>
      updateTask(id, { completed }),
    onSuccess: () => {
      $tasksQuery.refetch();
    },
  });

  // Modal states
  let modalOpen = $state(false);
  let editingTask = $state<Task | null>(null);
  let deleteConfirmOpen = $state(false);
  let deleteTaskId = $state<string | null>(null);

  function openCreateModal() {
    editingTask = null;
    modalOpen = true;
  }

  function openEditModal(task: Task) {
    editingTask = task;
    modalOpen = true;
  }

  function handleModalClose() {
    modalOpen = false;
    editingTask = null;
    $tasksQuery.refetch(); // Refresh after modal closes
  }

  function confirmDelete(id: string, e: Event) {
    e.stopPropagation();
    deleteTaskId = id;
    deleteConfirmOpen = true;
  }

  function handleDelete() {
    if (deleteTaskId) {
      $deleteMut.mutate(deleteTaskId);
      deleteConfirmOpen = false;
      deleteTaskId = null;
    }
  }

  function toggleComplete(task: Task, e: Event) {
    e.stopPropagation();
    $toggleCompleteMut.mutate({ id: task.id, completed: !task.completed });
  }

  function formatTime(dateStr: string): string {
    return new Date(dateStr).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function formatShortDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString([], {
      month: "short",
      day: "numeric",
    });
  }

  // Toggle sort on column header click
  function toggleSort(field: SortField) {
    if (sortField === field) {
      sortDirection = sortDirection === "asc" ? "desc" : "asc";
    } else {
      sortField = field;
      sortDirection = field === "title" ? "asc" : "desc";
    }
    // No manual refetch needed - $effect watches queryParams which includes sort state
  }

  function getPriorityLabel(priority: number): string {
    if (priority >= 7) return "High";
    if (priority >= 4) return "Medium";
    return "Low";
  }

  function getPriorityColor(priority: number): string {
    if (priority >= 7) return "badge-error";
    if (priority >= 4) return "badge-warning";
    return "badge-info";
  }

  function clearFilters() {
    searchQuery = "";
    debouncedSearch = "";
    filterCategory = "all";
    filterStatus = "all";
    filterPriority = "all";
    dateFrom = "";
    dateTo = "";
    // No manual refetch needed - $effect watches queryParams
  }

  // Handle filter changes - no longer needed since $effect watches queryParams
  function onFilterChange() {
    // State changes automatically trigger refetch via $effect
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

  const hasActiveFilters = $derived(
    debouncedSearch !== "" ||
      filterCategory !== "all" ||
      filterStatus !== "all" ||
      filterPriority !== "all" ||
      dateFrom !== "" ||
      dateTo !== "",
  );

  const tasks = $derived($tasksQuery.data?.items || []);
  const categories = $derived($categoriesQuery.data?.items || []);
  const totalTasks = $derived($tasksQuery.data?.total || 0);
  const isSearching = $derived(searchQuery !== debouncedSearch);
</script>

<svelte:head>
  <title>Tasks - Lucid Logs</title>
</svelte:head>

<div class="max-w-6xl mx-auto space-y-6">
  <!-- Header -->
  <div
    class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
  >
    <div>
      <h1 class="text-3xl font-bold">Tasks</h1>
      <p class="text-sm opacity-60 mt-1">
        {#if todayOnly}
          Today's tasks ({totalTasks})
        {:else if hasActiveFilters}
          Showing {tasks.length} filtered results
        {:else}
          {totalTasks} total tasks
        {/if}
      </p>
    </div>
    <button
      class="btn btn-primary gap-2 shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-all"
      onclick={openCreateModal}
    >
      <Plus class="w-4 h-4" />
      New Task
    </button>
  </div>

  <!-- Search and Filters Bar -->
  <div class="card bg-base-100 shadow-lg border border-base-200">
    <div class="card-body p-4">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center">
        <!-- Search Input -->
        <div class="relative flex-1">
          {#if isSearching || $tasksQuery.isFetching}
            <Loader2
              class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-50 animate-spin"
            />
          {:else}
            <Search
              class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 opacity-50"
            />
          {/if}
          <input
            type="text"
            placeholder="Search tasks..."
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
                // State change triggers refetch via $effect
              }}
            >
              <X class="w-3 h-3" />
            </button>
          {/if}
        </div>

        <!-- Today Only Toggle -->
        <div class="flex items-center gap-2 bg-base-200 rounded-lg px-3 py-2">
          <CalendarDays class="w-4 h-4 opacity-60" />
          <span class="text-sm font-medium">Today</span>
          <input
            type="checkbox"
            class="toggle toggle-primary toggle-sm"
            bind:checked={todayOnly}
            onchange={onFilterChange}
          />
        </div>

        <!-- Filter Toggle Button -->
        <button
          class={cn(
            "btn gap-2",
            showFilters ? "btn-primary" : "btn-ghost border border-base-300",
          )}
          onclick={() => (showFilters = !showFilters)}
        >
          <Filter class="w-4 h-4" />
          Filters
          {#if hasActiveFilters}
            <span class="badge badge-sm badge-secondary">Active</span>
          {/if}
        </button>
      </div>

      <!-- Expanded Filters -->
      {#if showFilters}
        <div
          class="grid grid-cols-2 md:grid-cols-4 gap-3 pt-4 border-t border-base-200 mt-4"
        >
          <!-- Date Range (shows when todayOnly is off) -->
          {#if !todayOnly}
            <div class="form-control">
              <label class="label py-1" for="dateFrom">
                <span class="label-text text-xs font-medium opacity-70">
                  From Date
                </span>
              </label>
              <input
                id="dateFrom"
                type="date"
                class="input input-bordered input-sm w-full"
                bind:value={dateFrom}
                onchange={onFilterChange}
              />
            </div>
            <div class="form-control">
              <label class="label py-1" for="dateTo">
                <span class="label-text text-xs font-medium opacity-70">
                  To Date
                </span>
              </label>
              <input
                id="dateTo"
                type="date"
                class="input input-bordered input-sm w-full"
                bind:value={dateTo}
                onchange={onFilterChange}
              />
            </div>
          {/if}

          <div class="form-control">
            <label class="label py-1" for="filterCategory">
              <span class="label-text text-xs font-medium opacity-70">
                Category
              </span>
            </label>
            <select
              id="filterCategory"
              class="select select-bordered select-sm w-full"
              bind:value={filterCategory}
              onchange={onFilterChange}
            >
              <option value="all">All Categories</option>
              {#each categories as cat}
                <option value={cat.id}>{cat.name}</option>
              {/each}
            </select>
          </div>

          <div class="form-control">
            <label class="label py-1" for="filterStatus">
              <span class="label-text text-xs font-medium opacity-70">
                Status
              </span>
            </label>
            <select
              id="filterStatus"
              class="select select-bordered select-sm w-full"
              bind:value={filterStatus}
              onchange={onFilterChange}
            >
              <option value="all">All Status</option>
              <option value="completed">Completed</option>
              <option value="pending">Pending</option>
            </select>
          </div>

          <div class="form-control">
            <label class="label py-1" for="filterPriority">
              <span class="label-text text-xs font-medium opacity-70">
                Priority
              </span>
            </label>
            <select
              id="filterPriority"
              class="select select-bordered select-sm w-full"
              bind:value={filterPriority}
              onchange={onFilterChange}
            >
              <option value="all">All Priorities</option>
              <option value="high">High (7-10)</option>
              <option value="medium">Medium (4-6)</option>
              <option value="low">Low (1-3)</option>
            </select>
          </div>

          <!-- Clear Filters -->
          {#if hasActiveFilters}
            <div class="form-control justify-end">
              <button class="btn btn-ghost btn-sm gap-1" onclick={clearFilters}>
                <X class="w-4 h-4" />
                Clear All
              </button>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>

  <!-- Tasks Table -->
  {#if $tasksQuery.isLoading}
    <div class="card bg-base-100 shadow-lg border border-base-200">
      <div class="card-body py-16 text-center">
        <span class="loading loading-spinner loading-lg text-primary mx-auto"
        ></span>
        <p class="opacity-60 mt-4">Loading tasks...</p>
      </div>
    </div>
  {:else if $tasksQuery.isError}
    <div class="alert alert-error shadow-lg">
      <span>Failed to load tasks. Please try again.</span>
    </div>
  {:else if tasks.length === 0}
    <div class="card bg-base-100 shadow-lg border border-base-200">
      <div class="card-body py-12 text-center">
        {#if todayOnly || hasActiveFilters}
          <Search class="w-12 h-12 opacity-30 mx-auto mb-4" />
          <h2 class="text-xl font-semibold">No matching tasks</h2>
          <p class="opacity-60 mt-1">
            {#if todayOnly}
              No tasks scheduled for today.
            {:else}
              Try adjusting your search or filter criteria
            {/if}
          </p>
          <div class="flex gap-2 justify-center mt-4">
            {#if todayOnly}
              <button
                class="btn btn-ghost btn-sm"
                onclick={() => {
                  todayOnly = false;
                  // State change triggers refetch via $effect
                }}
              >
                Show all dates
              </button>
            {/if}
            {#if hasActiveFilters}
              <button class="btn btn-ghost btn-sm" onclick={clearFilters}>
                Clear filters
              </button>
            {/if}
          </div>
        {:else}
          <div
            class="w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4"
          >
            <ListTodo class="w-10 h-10 text-primary" />
          </div>
          <h2 class="text-2xl font-bold">No tasks yet</h2>
          <p class="opacity-60 mt-2 max-w-md mx-auto">
            Create your first task to start tracking your activities
          </p>
          <button
            class="btn btn-primary mt-6 gap-2 shadow-lg shadow-primary/20"
            onclick={openCreateModal}
          >
            <Plus class="w-4 h-4" />
            Create Your First Task
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
          <!-- Table Header with Sortable Columns -->
          <thead class="bg-base-200">
            <tr>
              <th class="w-12"></th>
              <!-- Title - Sortable -->
              <th>
                <button
                  class={cn(
                    "flex items-center gap-2 font-semibold transition-colors",
                    sortField === "title"
                      ? "text-primary"
                      : "hover:text-primary",
                  )}
                  onclick={() => toggleSort("title")}
                >
                  Task
                  {#if sortField === "title"}
                    {#if sortDirection === "asc"}
                      <ChevronUp class="w-4 h-4" />
                    {:else}
                      <ChevronDown class="w-4 h-4" />
                    {/if}
                  {:else}
                    <ArrowUpDown class="w-3 h-3 opacity-40" />
                  {/if}
                </button>
              </th>
              <!-- Category -->
              <th class="hidden lg:table-cell">Category</th>
              <!-- Date - Sortable -->
              <th>
                <button
                  class={cn(
                    "flex items-center gap-2 font-semibold transition-colors",
                    sortField === "start_date"
                      ? "text-primary"
                      : "hover:text-primary",
                  )}
                  onclick={() => toggleSort("start_date")}
                >
                  Date
                  {#if sortField === "start_date"}
                    {#if sortDirection === "asc"}
                      <ChevronUp class="w-4 h-4" />
                    {:else}
                      <ChevronDown class="w-4 h-4" />
                    {/if}
                  {:else}
                    <ArrowUpDown class="w-3 h-3 opacity-40" />
                  {/if}
                </button>
              </th>
              <!-- Priority - Sortable -->
              <th class="hidden md:table-cell">
                <button
                  class={cn(
                    "flex items-center gap-2 font-semibold transition-colors",
                    sortField === "priority"
                      ? "text-primary"
                      : "hover:text-primary",
                  )}
                  onclick={() => toggleSort("priority")}
                >
                  Priority
                  {#if sortField === "priority"}
                    {#if sortDirection === "asc"}
                      <ChevronUp class="w-4 h-4" />
                    {:else}
                      <ChevronDown class="w-4 h-4" />
                    {/if}
                  {:else}
                    <ArrowUpDown class="w-3 h-3 opacity-40" />
                  {/if}
                </button>
              </th>
              <th class="w-24"></th>
            </tr>
          </thead>

          <!-- Table Body -->
          <tbody>
            {#each tasks as task (task.id)}
              <tr
                class={cn(
                  "hover:bg-base-200/50 transition-colors cursor-pointer group",
                  task.completed && "bg-success/5 opacity-70",
                )}
                onclick={() => openEditModal(task)}
              >
                <!-- Status -->
                <td>
                  <button
                    onclick={(e) => toggleComplete(task, e)}
                    disabled={$toggleCompleteMut.isPending}
                  >
                    <div
                      class={cn(
                        "w-6 h-6 rounded-full border-2 flex items-center justify-center transition-all",
                        task.completed
                          ? "bg-success border-success text-success-content"
                          : "border-base-content/20 hover:border-primary hover:scale-110",
                      )}
                    >
                      {#if task.completed}
                        <Check class="w-4 h-4" />
                      {/if}
                    </div>
                  </button>
                </td>

                <!-- Title with Search Highlight -->
                <td>
                  <div class="flex flex-col gap-1">
                    <span
                      class={cn(
                        "font-semibold",
                        task.completed && "line-through opacity-50",
                      )}
                    >
                      {#if debouncedSearch}
                        {@html highlightText(task.title, debouncedSearch)}
                      {:else}
                        {task.title}
                      {/if}
                    </span>
                    {#if task.journal}
                      <span class="text-sm opacity-50 line-clamp-1 max-w-md">
                        {#if debouncedSearch}
                          {@html highlightText(
                            task.journal.replace(/<[^>]*>/g, "").slice(0, 80),
                            debouncedSearch,
                          )}
                        {:else}
                          {task.journal.replace(/<[^>]*>/g, "").slice(0, 80)}
                        {/if}
                      </span>
                    {/if}
                    {#if task.category}
                      <div class="lg:hidden mt-1">
                        <span
                          class="badge badge-sm"
                          style="background-color: {task.category
                            .color}20; color: {task.category.color};"
                        >
                          {task.category.name}
                        </span>
                      </div>
                    {/if}
                  </div>
                </td>

                <!-- Category -->
                <td class="hidden lg:table-cell">
                  {#if task.category}
                    <span
                      class="badge"
                      style="background-color: {task.category
                        .color}20; color: {task.category.color};"
                    >
                      {task.category.name}
                    </span>
                  {:else}
                    <span class="opacity-30">—</span>
                  {/if}
                </td>

                <!-- Date -->
                <td>
                  <div class="flex flex-col">
                    <span class="text-sm font-medium">
                      {formatShortDate(task.start_date)}
                    </span>
                    <span class="text-xs opacity-50 flex items-center gap-1">
                      <Clock class="w-3 h-3" />
                      {formatTime(task.start_date)} - {formatTime(
                        task.end_date,
                      )}
                    </span>
                  </div>
                </td>

                <!-- Priority -->
                <td class="hidden md:table-cell">
                  <span
                    class={cn(
                      "badge badge-sm",
                      getPriorityColor(task.priority),
                    )}
                  >
                    {getPriorityLabel(task.priority)}
                  </span>
                </td>

                <!-- Actions -->
                <td>
                  <div
                    class="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <button
                      class="btn btn-ghost btn-sm btn-square"
                      onclick={(e) => {
                        e.stopPropagation();
                        openEditModal(task);
                      }}
                    >
                      <Edit class="w-4 h-4" />
                    </button>
                    <button
                      class="btn btn-ghost btn-sm btn-square text-error"
                      onclick={(e) => confirmDelete(task.id, e)}
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
        <span>{tasks.length} tasks</span>
        {#if $tasksQuery.isFetching}
          <span class="flex items-center gap-2 text-primary">
            <Loader2 class="w-4 h-4 animate-spin" />
            Loading...
          </span>
        {/if}
      </div>
    </div>
  {/if}
</div>

<!-- Task Modal -->
<TaskModal
  bind:open={modalOpen}
  task={editingTask}
  onClose={handleModalClose}
/>

<!-- Delete Dialog -->
<dialog class="modal" class:modal-open={deleteConfirmOpen}>
  <div class="modal-box max-w-sm">
    <h3 class="font-bold text-lg">Delete Task?</h3>
    <p class="py-4 opacity-70">This action cannot be undone.</p>
    <div class="modal-action">
      <button class="btn btn-ghost" onclick={() => (deleteConfirmOpen = false)}>
        Cancel
      </button>
      <button
        class="btn btn-error"
        onclick={handleDelete}
        disabled={$deleteMut.isPending}
      >
        Delete
      </button>
    </div>
  </div>
  <form method="dialog" class="modal-backdrop">
    <button onclick={() => (deleteConfirmOpen = false)}>close</button>
  </form>
</dialog>
