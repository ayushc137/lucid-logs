<script lang="ts">
    import {
        Palette,
        Plus,
        Trash2,
        SquarePen,
        Check,
        X,
        Search,
        LoaderCircle,
        Pipette,
        CircleAlert,
    } from "lucide-svelte";
    import { ErrorAlert, ConfirmDialog, ColorPicker } from "$lib/components/ui";
    import {
        createQuery,
        createMutation,
        useQueryClient,
    } from "@tanstack/svelte-query";
    import {
        getCategories,
        createCategory,
        updateCategory,
        deleteCategory,
        type Category,
    } from "$lib/api/categories";
    import { cn } from "$lib/utils";
    import {
        COLOR_PRESETS,
        DEFAULT_COLOR,
        getContrastColor,
    } from "$lib/constants";

    const queryClient = useQueryClient();

    // Search state
    let searchQuery = $state("");
    let debouncedSearch = $state("");
    let searchTimeout: ReturnType<typeof setTimeout> | null = null;

    function handleSearchInput(value: string) {
        searchQuery = value;
        if (searchTimeout) clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            debouncedSearch = value;
        }, 300);
    }

    // Query params
    const queryParams = $derived({
        limit: 100,
        search: debouncedSearch || undefined,
    });

    // Mutable ref for tracking param changes
    let currentParamsJson = "";

    // Category query
    const query = createQuery({
        queryKey: ["categories-list"],
        queryFn: () => getCategories(queryParams),
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

    const categories = $derived($query.data?.items || []);
    const totalCategories = $derived($query.data?.total || 0);

    // Modal states
    let isCreating = $state(false);
    let editingId = $state<string | null>(null);
    let deleteConfirmOpen = $state(false);
    let deleteTargetId = $state<string | null>(null);

    // Create form state
    let newName = $state("");
    let newColor = $state(DEFAULT_COLOR);
    let useCustomColor = $state(false);
    let createError = $state("");

    // Edit form state
    let editName = $state("");
    let editColor = $state("");
    let editUseCustomColor = $state(false);

    // Mutations
    const createMut = createMutation({
        mutationFn: (data: { name: string; color: string }) =>
            createCategory(data),
        onSuccess: async () => {
            await queryClient.invalidateQueries({
                queryKey: ["categories-list"],
            });
            await queryClient.invalidateQueries({ queryKey: ["categories"] });
            resetCreateForm();
        },
        onError: (error: Error) => {
            // Use the error message directly from the API
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
            await queryClient.invalidateQueries({
                queryKey: ["categories-list"],
            });
            await queryClient.invalidateQueries({ queryKey: ["categories"] });
            editingId = null;
        },
    });

    const deleteMut = createMutation({
        mutationFn: (id: string) => deleteCategory(id),
        onSuccess: async () => {
            await queryClient.invalidateQueries({
                queryKey: ["categories-list"],
            });
            await queryClient.invalidateQueries({ queryKey: ["categories"] });
            deleteConfirmOpen = false;
            deleteTargetId = null;
        },
    });

    function resetCreateForm() {
        newName = "";
        newColor = DEFAULT_COLOR;
        useCustomColor = false;
        createError = "";
        isCreating = false;
    }

    function startEdit(cat: Category) {
        editingId = cat.id;
        editName = cat.name;
        editColor = cat.color;
        editUseCustomColor = !COLOR_PRESETS.includes(
            cat.color as (typeof COLOR_PRESETS)[number],
        );
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
        if (newName.trim().length > 40) {
            createError = "Category name must be 40 characters or less";
            return;
        }
        if (newName.trim()) {
            $createMut.mutate({ name: newName.trim(), color: newColor });
        }
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

    const isSearching = $derived(searchQuery !== debouncedSearch);
</script>

<svelte:head>
    <title>Categories - Lucid Logs</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
    <!-- Header -->
    <div
        class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
    >
        <div>
            <h1 class="text-3xl font-bold">Categories</h1>
            <p class="text-sm opacity-60 mt-1">
                {#if debouncedSearch}
                    Found {categories.length} matching categories
                {:else}
                    {totalCategories} categories
                {/if}
            </p>
        </div>
        <button
            class="btn btn-primary gap-2 shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-all"
            onclick={() => (isCreating = true)}
        >
            <Plus class="w-4 h-4" />
            New Category
        </button>
    </div>

    <!-- Search Bar -->
    <div class="card bg-base-100 shadow-lg border border-base-200">
        <div class="card-body p-4">
            <div class="relative">
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
                    placeholder="Search categories..."
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
        </div>
    </div>

    <!-- Create Modal -->
    {#if isCreating}
        <div class="card bg-base-100 shadow-lg border border-primary/20">
            <div class="card-body">
                <h3 class="font-bold text-lg flex items-center gap-2">
                    <Plus class="w-5 h-5 text-primary" />
                    Create New Category
                </h3>

                {#if createError}
                    <div class="alert alert-error">
                        <CircleAlert class="w-4 h-4" />
                        <span>{createError}</span>
                    </div>
                {/if}

                <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                    <!-- Name Input -->
                    <div class="form-control">
                        <label class="label" for="new-category-name">
                            <span class="label-text font-medium">Name</span>
                            <span class="label-text-alt opacity-50"
                                >{newName.length}/40</span
                            >
                        </label>
                        <input
                            id="new-category-name"
                            type="text"
                            placeholder="e.g., Work, Personal, Health"
                            class={cn(
                                "input input-bordered w-full",
                                newName.length > 40 && "input-error",
                            )}
                            maxlength={40}
                            bind:value={newName}
                            onkeydown={(e: KeyboardEvent) =>
                                e.key === "Enter" && handleCreate()}
                        />
                    </div>

                    <!-- Color Picker -->
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text font-medium">Color</span>
                            <button
                                type="button"
                                class={cn(
                                    "btn btn-xs gap-1",
                                    useCustomColor
                                        ? "btn-primary"
                                        : "btn-ghost",
                                )}
                                onclick={() =>
                                    (useCustomColor = !useCustomColor)}
                            >
                                <Pipette class="w-3 h-3" />
                                Custom
                            </button>
                        </label>

                        <ColorPicker
                            bind:value={newColor}
                            customMode={useCustomColor}
                        />
                    </div>
                </div>

                <!-- Preview -->
                <div
                    class="flex items-center gap-3 mt-4 pt-4 border-t border-base-300"
                >
                    <span class="text-sm opacity-60">Preview:</span>
                    <span
                        class="badge badge-lg font-medium"
                        style="background-color: {newColor}; color: {getContrastColor(
                            newColor,
                        )}; border: none;"
                    >
                        {newName || "Category Name"}
                    </span>
                </div>

                <!-- Actions -->
                <div class="flex justify-end gap-2 mt-4">
                    <button class="btn btn-ghost" onclick={resetCreateForm}>
                        Cancel
                    </button>
                    <button
                        class="btn btn-primary gap-2"
                        onclick={handleCreate}
                        disabled={!newName.trim() || $createMut.isPending}
                    >
                        {#if $createMut.isPending}
                            <span class="loading loading-spinner loading-sm"
                            ></span>
                        {:else}
                            <Check class="w-4 h-4" />
                        {/if}
                        Create Category
                    </button>
                </div>
            </div>
        </div>
    {/if}

    <!-- Categories Table -->
    {#if $query.isLoading}
        <div class="card bg-base-100 shadow-lg border border-base-200">
            <div class="card-body py-16 text-center">
                <span
                    class="loading loading-spinner loading-lg text-primary mx-auto"
                ></span>
                <p class="opacity-60 mt-4">Loading categories...</p>
            </div>
        </div>
    {:else if $query.isError}
        <ErrorAlert
            message="Failed to load categories. Please try again."
            onRetry={() => $query.refetch()}
        />
    {:else if categories.length === 0}
        <div class="card bg-base-100 shadow-lg border border-base-200">
            <div class="card-body py-12 text-center">
                {#if debouncedSearch}
                    <Search class="w-12 h-12 opacity-30 mx-auto mb-4" />
                    <h2 class="text-xl font-semibold">
                        No matching categories
                    </h2>
                    <p class="opacity-60 mt-1">
                        Try adjusting your search term
                    </p>
                    <button
                        class="btn btn-ghost btn-sm mt-4"
                        onclick={() => {
                            searchQuery = "";
                            debouncedSearch = "";
                        }}
                    >
                        Clear search
                    </button>
                {:else}
                    <div
                        class="w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4"
                    >
                        <Palette class="w-10 h-10 text-primary" />
                    </div>
                    <h2 class="text-2xl font-bold">No categories yet</h2>
                    <p class="opacity-60 mt-2 max-w-md mx-auto">
                        Create categories to organize your tasks
                    </p>
                    <button
                        class="btn btn-primary mt-6 gap-2 shadow-lg shadow-primary/20"
                        onclick={() => (isCreating = true)}
                    >
                        <Plus class="w-4 h-4" />
                        Create Your First Category
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
                    <thead class="bg-base-200">
                        <tr>
                            <th class="w-16">Color</th>
                            <th>Name</th>
                            <th class="hidden sm:table-cell">Created</th>
                            <th class="w-24 text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each categories as category (category.id)}
                            {#if editingId === category.id}
                                <!-- Edit Row -->
                                <tr class="bg-base-200/50">
                                    <td colspan="4">
                                        <div class="flex flex-col gap-4 py-2">
                                            <div
                                                class="grid grid-cols-1 md:grid-cols-2 gap-4"
                                            >
                                                <!-- Name Input -->
                                                <div class="form-control">
                                                    <label
                                                        class="label py-1"
                                                        for="edit-name"
                                                    >
                                                        <span
                                                            class="label-text text-xs font-medium"
                                                            >Name</span
                                                        >
                                                    </label>
                                                    <input
                                                        id="edit-name"
                                                        type="text"
                                                        class="input input-bordered input-sm"
                                                        maxlength={40}
                                                        bind:value={editName}
                                                    />
                                                </div>

                                                <!-- Color Picker -->
                                                <div class="form-control">
                                                    <label class="label py-1">
                                                        <span
                                                            class="label-text text-xs font-medium"
                                                            >Color</span
                                                        >
                                                        <button
                                                            type="button"
                                                            class={cn(
                                                                "btn btn-xs gap-1",
                                                                editUseCustomColor
                                                                    ? "btn-primary"
                                                                    : "btn-ghost",
                                                            )}
                                                            onclick={() =>
                                                                (editUseCustomColor =
                                                                    !editUseCustomColor)}
                                                        >
                                                            <Pipette
                                                                class="w-3 h-3"
                                                            />
                                                            Custom
                                                        </button>
                                                    </label>
                                                    <ColorPicker
                                                        bind:value={editColor}
                                                        customMode={editUseCustomColor}
                                                    />
                                                </div>
                                            </div>
                                            <div class="flex justify-end gap-2">
                                                <button
                                                    class="btn btn-ghost btn-sm"
                                                    onclick={() =>
                                                        (editingId = null)}
                                                >
                                                    Cancel
                                                </button>
                                                <button
                                                    class="btn btn-primary btn-sm gap-1"
                                                    onclick={saveEdit}
                                                    disabled={$updateMut.isPending}
                                                >
                                                    {#if $updateMut.isPending}
                                                        <span
                                                            class="loading loading-spinner loading-xs"
                                                        ></span>
                                                    {:else}
                                                        <Check
                                                            class="w-4 h-4"
                                                        />
                                                    {/if}
                                                    Save
                                                </button>
                                            </div>
                                        </div>
                                    </td>
                                </tr>
                            {:else}
                                <!-- View Row -->
                                <tr
                                    class="hover:bg-base-200/50 transition-colors group"
                                >
                                    <td>
                                        <div
                                            class="w-8 h-8 rounded-lg shadow-sm"
                                            style="background-color: {category.color};"
                                        ></div>
                                    </td>
                                    <td>
                                        <div class="flex items-center gap-3">
                                            <span class="font-semibold"
                                                >{category.name}</span
                                            >
                                            <span
                                                class="badge badge-sm"
                                                style="background-color: {category.color}20; color: {category.color};"
                                            >
                                                {category.color}
                                            </span>
                                        </div>
                                    </td>
                                    <td class="hidden sm:table-cell">
                                        <span class="text-sm opacity-60">
                                            {new Date(
                                                category.created_at,
                                            ).toLocaleDateString([], {
                                                month: "short",
                                                day: "numeric",
                                                year: "numeric",
                                            })}
                                        </span>
                                    </td>
                                    <td>
                                        <div
                                            class="flex gap-1 justify-end opacity-0 group-hover:opacity-100 transition-opacity"
                                        >
                                            <button
                                                class="btn btn-ghost btn-sm btn-square"
                                                onclick={() =>
                                                    startEdit(category)}
                                            >
                                                <SquarePen class="w-4 h-4" />
                                            </button>
                                            <button
                                                class="btn btn-ghost btn-sm btn-square text-error"
                                                onclick={(e) =>
                                                    confirmDelete(
                                                        category.id,
                                                        e,
                                                    )}
                                            >
                                                <Trash2 class="w-4 h-4" />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            {/if}
                        {/each}
                    </tbody>
                </table>
            </div>

            <!-- Footer -->
            <div
                class="px-4 py-3 bg-base-200/50 border-t border-base-200 text-sm opacity-60 flex items-center justify-between"
            >
                <span>{categories.length} categories</span>
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

<!-- Delete Confirmation Modal -->
<ConfirmDialog
    bind:open={deleteConfirmOpen}
    title="Delete Category"
    message="Are you sure you want to delete this category? This action cannot be undone. Tasks using this category will be unassigned."
    confirmText="Delete"
    destructive={true}
    loading={$deleteMut.isPending}
    onConfirm={handleDelete}
    onCancel={() => {
        deleteConfirmOpen = false;
        deleteTargetId = null;
    }}
/>
