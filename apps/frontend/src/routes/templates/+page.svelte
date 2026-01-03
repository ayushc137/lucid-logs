<script lang="ts">
  import {
    createQuery,
    createMutation,
    useQueryClient,
  } from "@tanstack/svelte-query";
  import { getTemplates, deleteTemplate, type TaskTemplate } from "$lib/api";
  import { ErrorAlert, ConfirmDialog } from "$lib/components/ui";
  import { TemplateModal } from "$lib/components/templates";
  import {
    Zap,
    Plus,
    Trash2,
    SquarePen,
    Check,
    X,
    Search,
    LoaderCircle,
    Clock,
    Hash,
    Play,
    ChevronUp,
    ChevronDown,
    ArrowUpDown,
  } from "lucide-svelte";
  import { cn } from "$lib/utils";
  import {
    getUrlParams,
    updateUrlParams,
    parsers,
  } from "$lib/utils/navigation";
  import { goto } from "$app/navigation";
  import { browser } from "$app/environment";
  import { page } from "$app/stores";

  const queryClient = useQueryClient();

  // Initialize state from URL
  const initialParams = getUrlParams({
    q: parsers.string(""),
    quick_log: parsers.string(""),
  });

  // Search state
  let searchQuery = $state(initialParams.q);
  let debouncedSearch = $state(initialParams.q);
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;

  // Filter state
  let quickLogFilter = $state(initialParams.quick_log);

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
        quick_log: quickLogFilter,
      },
      { replace: true, keepFocus: true },
    );
  });

  // Templates query
  const query = createQuery({
    queryKey: ["templates-list"],
    queryFn: () => getTemplates({ limit: 100 }),
    retry: false,
  });

  // Client-side filtering since backend might not support search
  const templates = $derived(() => {
    let items = $query.data?.items || [];

    // Search filter
    if (debouncedSearch) {
      const search = debouncedSearch.toLowerCase();
      items = items.filter(
        (t) =>
          t.title.toLowerCase().includes(search) ||
          t.description?.toLowerCase().includes(search),
      );
    }

    // Quick log filter
    if (quickLogFilter === "true") {
      items = items.filter((t) => t.is_quick_log);
    } else if (quickLogFilter === "false") {
      items = items.filter((t) => !t.is_quick_log);
    }

    return items;
  });

  const totalTemplates = $derived($query.data?.total || 0);

  // Modal states
  let modalOpen = $state(false);
  let editingTemplate = $state<TaskTemplate | null>(null);
  let deleteConfirmOpen = $state(false);
  let deleteTargetId = $state<string | null>(null);

  // Mutations
  const deleteMut = createMutation({
    mutationFn: (id: string) => deleteTemplate(id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["templates-list"] });
      await queryClient.invalidateQueries({ queryKey: ["templates"] });
      deleteConfirmOpen = false;
      deleteTargetId = null;
    },
  });

  function handleCreate() {
    editingTemplate = null;
    modalOpen = true;
  }

  function startEdit(template: TaskTemplate) {
    editingTemplate = template;
    modalOpen = true;
  }

  function handleModalClose() {
    modalOpen = false;
    editingTemplate = null;
    queryClient.invalidateQueries({ queryKey: ["templates-list"] });
    queryClient.invalidateQueries({ queryKey: ["templates"] });
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

  function handleUse(template: TaskTemplate, e: Event) {
    e.stopPropagation();
    // Navigate to create task with template pre-filled
    if (browser) {
      sessionStorage.setItem(
        "task-form-referrer",
        $page.url.pathname + $page.url.search,
      );
      sessionStorage.setItem("task-template", JSON.stringify(template));
    }
    goto("/tasks/create");
  }

  const isSearching = $derived(searchQuery !== debouncedSearch);

  const quickLogOptions = [
    { value: "", label: "All Templates" },
    { value: "true", label: "Quick Log Only" },
    { value: "false", label: "Standard Only" },
  ];

  function formatDuration(seconds?: number): string {
    if (!seconds) return "—";
    const mins = Math.floor(seconds / 60);
    if (mins < 60) return `${mins}m`;
    const hours = Math.floor(mins / 60);
    const remainMins = mins % 60;
    return remainMins > 0 ? `${hours}h ${remainMins}m` : `${hours}h`;
  }

  const priorityLabels = [
    "None",
    "Low",
    "Medium",
    "High",
    "Critical",
    "Urgent",
  ];
  const priorityColors = [
    "badge-ghost",
    "badge-info",
    "badge-success",
    "badge-warning",
    "badge-error",
    "badge-error",
  ];

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
  <title>Templates - Lucid Logs</title>
</svelte:head>

<div class="max-w-6xl mx-auto space-y-6">
  <!-- Header -->
  <div
    class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
  >
    <div>
      <h1 class="text-3xl font-bold">Templates</h1>
      <p class="text-sm opacity-60 mt-1">
        {#if debouncedSearch || quickLogFilter}
          Found {templates().length} matching templates
        {:else}
          {totalTemplates} templates
        {/if}
      </p>
    </div>
    <button
      class="btn btn-secondary gap-2 shadow-lg shadow-secondary/20 hover:shadow-secondary/40 transition-all"
      onclick={handleCreate}
    >
      <Plus class="w-4 h-4" />
      New Template
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
            placeholder="Search templates..."
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

        <!-- Quick Log Filter -->
        <select
          class="select select-bordered w-full sm:w-44"
          bind:value={quickLogFilter}
        >
          {#each quickLogOptions as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>
      </div>
    </div>
  </div>

  <!-- Templates Table -->
  {#if $query.isLoading}
    <div class="card bg-base-100 shadow-lg border border-base-200">
      <div class="card-body py-16 text-center">
        <span class="loading loading-spinner loading-lg text-secondary mx-auto"
        ></span>
        <p class="opacity-60 mt-4">Loading templates...</p>
      </div>
    </div>
  {:else if $query.isError}
    <ErrorAlert
      message="Failed to load templates. Please try again."
      onRetry={() => $query.refetch()}
    />
  {:else if templates().length === 0}
    <div class="card bg-base-100 shadow-lg border border-base-200">
      <div class="card-body py-12 text-center">
        {#if debouncedSearch || quickLogFilter}
          <Search class="w-12 h-12 opacity-30 mx-auto mb-4" />
          <h2 class="text-xl font-semibold">No matching templates</h2>
          <p class="opacity-60 mt-1">Try adjusting your filters</p>
          <button
            class="btn btn-ghost btn-sm mt-4"
            onclick={() => {
              searchQuery = "";
              debouncedSearch = "";
              quickLogFilter = "";
            }}
          >
            Clear filters
          </button>
        {:else}
          <div
            class="w-20 h-20 rounded-full bg-secondary/10 flex items-center justify-center mx-auto mb-4"
          >
            <Zap class="w-10 h-10 text-secondary" />
          </div>
          <h2 class="text-2xl font-bold">No templates yet</h2>
          <p class="opacity-60 mt-2 max-w-md mx-auto">
            Create templates to speed up your task logging
          </p>
          <button
            class="btn btn-secondary mt-6 gap-2 shadow-lg shadow-secondary/20"
            onclick={handleCreate}
          >
            <Plus class="w-4 h-4" />
            Create Your First Template
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
              <th>Template</th>
              <th class="hidden md:table-cell">Duration</th>
              <th class="hidden lg:table-cell">Priority</th>
              <th class="hidden sm:table-cell">Quick Log</th>
              <th class="hidden md:table-cell">Uses</th>
              <th class="w-32 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each templates() as template (template.id)}
              <tr
                class="hover:bg-base-200/50 transition-colors group cursor-pointer"
                onclick={(e) => handleUse(template, e)}
              >
                <!-- Icon -->
                <td>
                  <div
                    class="w-10 h-10 rounded-lg flex items-center justify-center text-lg"
                    style="background-color: {template.color || '#6366f1'}20;"
                  >
                    {template.icon || "⚡"}
                  </div>
                </td>

                <!-- Title -->
                <td>
                  <div class="flex flex-col gap-0.5">
                    <span class="font-semibold">
                      {#if debouncedSearch}
                        {@html highlightText(template.title, debouncedSearch)}
                      {:else}
                        {template.title}
                      {/if}
                    </span>
                    {#if template.description}
                      <span class="text-xs opacity-50 truncate max-w-[200px]">
                        {#if debouncedSearch}
                          {@html highlightText(
                            template.description,
                            debouncedSearch,
                          )}
                        {:else}
                          {template.description}
                        {/if}
                      </span>
                    {/if}
                  </div>
                </td>

                <!-- Duration -->
                <td class="hidden md:table-cell">
                  <div class="flex items-center gap-1.5">
                    <Clock class="w-4 h-4 opacity-50" />
                    <span class="text-sm"
                      >{formatDuration(template.default_duration)}</span
                    >
                  </div>
                </td>

                <!-- Priority -->
                <td class="hidden lg:table-cell">
                  {#if template.default_priority !== undefined && template.default_priority !== null}
                    <span
                      class={cn(
                        "badge badge-sm",
                        priorityColors[template.default_priority],
                      )}
                    >
                      {priorityLabels[template.default_priority]}
                    </span>
                  {:else}
                    <span class="text-sm opacity-40">—</span>
                  {/if}
                </td>

                <!-- Quick Log -->
                <td class="hidden sm:table-cell">
                  {#if template.is_quick_log}
                    <div class="flex items-center gap-1.5">
                      <Zap class="w-4 h-4 text-secondary" />
                      <span class="text-sm text-secondary font-medium">Yes</span
                      >
                    </div>
                  {:else}
                    <span class="text-sm opacity-40">No</span>
                  {/if}
                </td>

                <!-- Uses -->
                <td class="hidden md:table-cell">
                  <div class="flex items-center gap-1.5">
                    <Hash class="w-3.5 h-3.5 opacity-50" />
                    <span class="text-sm">{template.use_count}</span>
                  </div>
                </td>

                <!-- Actions -->
                <td>
                  <div
                    class="flex gap-1 justify-end opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <button
                      class="btn btn-ghost btn-sm gap-1 text-secondary"
                      onclick={(e) => handleUse(template, e)}
                      title="Use this template"
                    >
                      <Play class="w-4 h-4" />
                      <span class="hidden lg:inline">Use</span>
                    </button>
                    <button
                      class="btn btn-ghost btn-sm btn-square"
                      onclick={(e) => {
                        e.stopPropagation();
                        startEdit(template);
                      }}
                    >
                      <SquarePen class="w-4 h-4" />
                    </button>
                    <button
                      class="btn btn-ghost btn-sm btn-square text-error"
                      onclick={(e) => confirmDelete(template.id, e)}
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
        <span>{templates().length} templates</span>
        {#if $query.isFetching}
          <span class="flex items-center gap-2 text-secondary">
            <LoaderCircle class="w-4 h-4 animate-spin" />
            Loading...
          </span>
        {/if}
      </div>
    </div>
  {/if}
</div>

<!-- Template Modal -->
<TemplateModal
  bind:open={modalOpen}
  template={editingTemplate}
  onClose={handleModalClose}
/>

<!-- Delete Confirmation Modal -->
<ConfirmDialog
  bind:open={deleteConfirmOpen}
  title="Delete Template"
  message="Are you sure you want to delete this template? This action cannot be undone."
  confirmText="Delete"
  destructive={true}
  loading={$deleteMut.isPending}
  onConfirm={handleDelete}
  onCancel={() => {
    deleteConfirmOpen = false;
    deleteTargetId = null;
  }}
/>
