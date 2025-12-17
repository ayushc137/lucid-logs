<script lang="ts">
  import { ListTodo, Plus, Trash2, Edit, Check } from "lucide-svelte";
  import {
    createQuery,
    createMutation,
    useQueryClient,
  } from "@tanstack/svelte-query";
  import { getTasks, deleteTask, updateTask, type Task } from "$lib/api";
  import { TaskModal } from "$lib/components/tasks";
  import { cn } from "$lib/utils";

  const queryClient = useQueryClient();

  // Fetch tasks
  const tasksQuery = createQuery({
    queryKey: ["tasks"],
    queryFn: () => getTasks({ limit: 50 }),
  });

  // Delete mutation
  const deleteMut = createMutation({
    mutationFn: (id: string) => deleteTask(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
    },
  });

  // Toggle complete mutation
  const toggleCompleteMut = createMutation({
    mutationFn: ({ id, completed }: { id: string; completed: boolean }) =>
      updateTask(id, { completed }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
    },
  });

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
  }

  function confirmDelete(id: string) {
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

  function toggleComplete(task: Task) {
    $toggleCompleteMut.mutate({ id: task.id, completed: !task.completed });
  }

  function formatTime(dateStr: string): string {
    return new Date(dateStr).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString([], {
      month: "short",
      day: "numeric",
    });
  }

  const tasks = $derived($tasksQuery.data?.items || []);
</script>

<svelte:head>
  <title>Tasks - Lucid Logs</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold">Tasks</h1>
    <button class="btn btn-primary btn-sm gap-2" onclick={openCreateModal}>
      <Plus class="w-4 h-4" />
      New Task
    </button>
  </div>

  {#if $tasksQuery.isLoading}
    <div class="card bg-base-100 shadow">
      <div class="card-body py-12 text-center">
        <span class="loading loading-spinner loading-lg text-primary mx-auto"
        ></span>
        <p class="opacity-60 mt-2">Loading tasks...</p>
      </div>
    </div>
  {:else if $tasksQuery.isError}
    <div class="alert alert-error">
      <span>Failed to load tasks. Please try again.</span>
    </div>
  {:else if tasks.length === 0}
    <div class="card bg-base-100 shadow">
      <div class="card-body py-12 text-center">
        <div
          class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mx-auto mb-4"
        >
          <ListTodo class="w-8 h-8 opacity-50" />
        </div>
        <h2 class="text-xl font-semibold">No tasks yet</h2>
        <p class="opacity-60 mt-1 max-w-md mx-auto">
          Create your first task to start tracking your activities
        </p>
        <button class="btn btn-primary mt-4 gap-2" onclick={openCreateModal}>
          <Plus class="w-4 h-4" />
          Create Task
        </button>
      </div>
    </div>
  {:else}
    <div class="space-y-3">
      {#each tasks as task (task.id)}
        <div
          class="card bg-base-100 shadow-sm hover:shadow-md transition-all border border-base-300 hover:border-primary/30"
        >
          <div class="card-body p-4">
            <div class="flex items-start gap-3">
              <!-- Complete checkbox -->
              <button
                class="mt-1 flex-shrink-0"
                onclick={() => toggleComplete(task)}
                disabled={$toggleCompleteMut.isPending}
              >
                <div
                  class={cn(
                    "w-5 h-5 rounded-full border-2 flex items-center justify-center transition-all",
                    task.completed
                      ? "bg-success border-success text-success-content"
                      : "border-base-content/30 hover:border-primary",
                  )}
                >
                  {#if task.completed}
                    <Check class="w-3 h-3" />
                  {/if}
                </div>
              </button>

              <!-- Task content -->
              <div class="flex-1 min-w-0">
                <h3
                  class={cn(
                    "font-semibold",
                    task.completed && "line-through opacity-50",
                  )}
                >
                  {task.title}
                </h3>
                <div class="flex flex-wrap gap-2 mt-1 text-xs opacity-60">
                  <span>{formatDate(task.start_date)}</span>
                  <span>•</span>
                  <span
                    >{formatTime(task.start_date)} - {formatTime(
                      task.end_date,
                    )}</span
                  >
                  {#if task.category}
                    <span>•</span>
                    <span
                      class="badge badge-xs badge-outline"
                      style="border-color: {task.category.color}; color: {task
                        .category.color}"
                    >
                      {task.category.name}
                    </span>
                  {/if}
                </div>
                {#if task.journal}
                  <p class="text-sm opacity-60 mt-2 line-clamp-2">
                    {task.journal}
                  </p>
                {/if}
                {#if (task.positives?.length || 0) > 0 || (task.negatives?.length || 0) > 0}
                  <div class="flex flex-wrap gap-1 mt-2">
                    {#each task.positives || [] as p}
                      <span class="badge badge-sm badge-success badge-outline"
                        >{p.text}</span
                      >
                    {/each}
                    {#each task.negatives || [] as n}
                      <span class="badge badge-sm badge-error badge-outline"
                        >{n.text}</span
                      >
                    {/each}
                  </div>
                {/if}
              </div>

              <!-- Actions -->
              <div class="flex gap-1">
                <button
                  class="btn btn-ghost btn-sm btn-square"
                  onclick={() => openEditModal(task)}
                  aria-label="Edit task"
                >
                  <Edit class="w-4 h-4" />
                </button>
                <button
                  class="btn btn-ghost btn-sm btn-square text-error"
                  onclick={() => confirmDelete(task.id)}
                  aria-label="Delete task"
                >
                  <Trash2 class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Task Modal -->
<TaskModal
  bind:open={modalOpen}
  task={editingTask}
  onClose={handleModalClose}
/>

<!-- Delete Confirmation Dialog -->
<dialog class="modal" class:modal-open={deleteConfirmOpen}>
  <div class="modal-box max-w-sm">
    <h3 class="font-bold text-lg">Delete Task?</h3>
    <p class="py-4 opacity-70">
      This action cannot be undone. The task will be permanently deleted.
    </p>
    <div class="modal-action">
      <button class="btn btn-ghost" onclick={() => (deleteConfirmOpen = false)}>
        Cancel
      </button>
      <button
        class="btn btn-error"
        onclick={handleDelete}
        disabled={$deleteMut.isPending}
      >
        {#if $deleteMut.isPending}
          <span class="loading loading-spinner loading-sm"></span>
        {/if}
        Delete
      </button>
    </div>
  </div>
  <form method="dialog" class="modal-backdrop">
    <button onclick={() => (deleteConfirmOpen = false)}>close</button>
  </form>
</dialog>
