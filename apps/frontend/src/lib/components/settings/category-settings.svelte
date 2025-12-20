<script lang="ts">
  import {
    Plus,
    Pencil,
    Trash2,
    X,
    Check,
    AlertCircle,
    Palette,
  } from "lucide-svelte";
  import {
    getCategories,
    createCategory,
    updateCategory,
    deleteCategory,
    type Category,
  } from "$lib/api/categories";
  import {
    createQuery,
    createMutation,
    useQueryClient,
  } from "@tanstack/svelte-query";
  import { cn } from "$lib/utils";

  const queryClient = useQueryClient();

  // 16 Color presets - same as TaskModal
  const colorPresets = [
    "#ef4444",
    "#f97316",
    "#eab308",
    "#22c55e",
    "#10b981",
    "#06b6d4",
    "#3b82f6",
    "#6366f1",
    "#8b5cf6",
    "#a855f7",
    "#ec4899",
    "#f43f5e",
    "#64748b",
    "#78716c",
    "#0ea5e9",
    "#14b8a6",
  ];

  // Queries
  const query = createQuery({
    queryKey: ["categories"],
    queryFn: () => getCategories({ limit: 100 }),
    retry: false,
  });

  // Local state
  let isCreating = $state(false);
  let newName = $state("");
  let newColor = $state("#6366f1");
  let createError = $state("");

  let editingId = $state<string | null>(null);
  let editName = $state("");
  let editColor = $state("");

  const categories = $derived($query.data?.items || []);

  // Mutations
  const createMut = createMutation({
    mutationFn: (data: { name: string; color: string }) => createCategory(data),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["categories"] });
      newName = "";
      newColor = "#6366f1";
      createError = "";
      isCreating = false;
    },
    onError: (error: Error) => {
      createError = error.message || "Failed to create category";
    },
  });

  const updateMut = createMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: { name?: string; color?: string };
    }) => updateCategory(id, data),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["categories"] });
      editingId = null;
    },
  });

  const deleteMut = createMutation({
    mutationFn: (id: string) => deleteCategory(id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["categories"] }),
  });

  function startEdit(cat: Category) {
    editingId = cat.id;
    editName = cat.name;
    editColor = cat.color;
  }

  function saveEdit() {
    if (editingId && editName.trim()) {
      $updateMut.mutate({
        id: editingId,
        data: { name: editName.trim(), color: editColor },
      });
    }
  }

  function handleCreate() {
    createError = "";
    if (newName.trim()) {
      $createMut.mutate({ name: newName.trim(), color: newColor });
    }
  }

  function cancelCreate() {
    isCreating = false;
    createError = "";
    newName = "";
  }
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <div class="stat-icon bg-primary/10">
        <Palette class="w-5 h-5 text-primary" />
      </div>
      <div>
        <h3 class="text-lg font-bold">Manage Categories</h3>
        <p class="text-sm opacity-50">
          Organize your tasks with custom categories
        </p>
      </div>
    </div>
    {#if !isCreating}
      <button
        class="btn btn-primary btn-sm gap-2"
        onclick={() => (isCreating = true)}
      >
        <Plus class="w-4 h-4" />
        Add
      </button>
    {/if}
  </div>

  {#if isCreating}
    <div
      class="p-4 rounded-lg bg-base-200/50 border border-base-300 space-y-4 animate-fade-in"
    >
      {#if createError}
        <div class="alert alert-error alert-sm">
          <AlertCircle class="w-4 h-4" />
          <span>{createError}</span>
        </div>
      {/if}

      <div class="space-y-4">
        <div class="form-control">
          <label class="label" for="new-category-name">
            <span class="label-text font-medium">Name</span>
          </label>
          <input
            id="new-category-name"
            type="text"
            placeholder="e.g., Work, Personal, Health"
            class="input input-bordered input-sm w-full"
            bind:value={newName}
            onkeydown={(e: KeyboardEvent) =>
              e.key === "Enter" && handleCreate()}
          />
        </div>

        <div class="form-control">
          <span class="label py-1">
            <span class="label-text font-medium">Color</span>
          </span>
          <!-- 16 Colors in 8x2 grid -->
          <div class="grid grid-cols-8 gap-1.5">
            {#each colorPresets as color}
              <button
                type="button"
                class={cn("color-picker-btn", newColor === color && "active")}
                style="background-color: {color};"
                onclick={() => (newColor = color)}
                aria-label="Select color"
              ></button>
            {/each}
          </div>
        </div>

        <div class="flex items-center gap-2 pt-2">
          <button
            class="btn btn-primary btn-sm gap-1"
            onclick={handleCreate}
            disabled={!newName.trim() || $createMut.isPending}
          >
            {#if $createMut.isPending}
              <span class="loading loading-spinner loading-xs"></span>
            {:else}
              <Check class="w-4 h-4" />
            {/if}
            Create
          </button>
          <button class="btn btn-ghost btn-sm" onclick={cancelCreate}>
            Cancel
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Categories List -->
  <div class="space-y-2">
    {#each categories as category (category.id)}
      <div
        class={cn(
          "flex items-center gap-3 p-3 rounded-lg border transition-all",
          editingId === category.id
            ? "bg-base-200 border-primary/30"
            : "bg-base-100 border-base-300 hover:border-base-content/20",
        )}
      >
        {#if editingId === category.id}
          <!-- Edit Mode -->
          <div class="flex-1 space-y-2">
            <div class="flex items-center gap-2">
              <input
                type="text"
                bind:value={editName}
                class="input input-sm input-bordered flex-1"
              />
              <div class="flex gap-1">
                <button
                  class="btn btn-ghost btn-sm btn-square text-success"
                  onclick={saveEdit}
                >
                  <Check class="w-4 h-4" />
                </button>
                <button
                  class="btn btn-ghost btn-sm btn-square"
                  onclick={() => (editingId = null)}
                >
                  <X class="w-4 h-4" />
                </button>
              </div>
            </div>
            <!-- All 16 colors in edit mode -->
            <div class="grid grid-cols-8 gap-1">
              {#each colorPresets as color}
                <button
                  type="button"
                  class={cn(
                    "color-picker-btn",
                    editColor === color && "active",
                  )}
                  style="background-color: {color};"
                  onclick={() => (editColor = color)}
                  aria-label="Select color"
                ></button>
              {/each}
            </div>
          </div>
        {:else}
          <!-- View Mode -->
          <div
            class="w-5 h-5 rounded flex-shrink-0"
            style="background-color: {category.color};"
          ></div>
          <span class="font-medium flex-1">{category.name}</span>
          <div class="flex gap-1">
            <button
              class="btn btn-ghost btn-sm btn-square opacity-50 hover:opacity-100 tooltip tooltip-left"
              onclick={() => startEdit(category)}
              data-tip="Edit"
            >
              <Pencil class="w-4 h-4" />
            </button>
            <button
              class="btn btn-ghost btn-sm btn-square text-error opacity-50 hover:opacity-100 tooltip tooltip-left"
              onclick={() => $deleteMut.mutate(category.id)}
              data-tip="Delete"
            >
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        {/if}
      </div>
    {:else}
      <div class="text-center py-10">
        <div
          class="w-14 h-14 mx-auto rounded-xl bg-base-200 flex items-center justify-center mb-4"
        >
          <Palette class="w-7 h-7 opacity-30" />
        </div>
        <h4 class="font-semibold opacity-60">No categories yet</h4>
        <p class="text-sm opacity-40 mt-1">
          Create your first category to get started!
        </p>
        <button
          class="btn btn-primary btn-sm gap-2 mt-4"
          onclick={() => (isCreating = true)}
        >
          <Plus class="w-4 h-4" />
          Create Category
        </button>
      </div>
    {/each}
  </div>
</div>
