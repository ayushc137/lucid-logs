<script lang="ts">
    import { createMutation, useQueryClient } from "@tanstack/svelte-query";
    import {
        createTask,
        updateTask,
        type Task,
        type CreateTaskRequest,
        type UpdateTaskRequest,
        type TaskItem,
    } from "$lib/api";
    import { createQuery } from "@tanstack/svelte-query";
    import { getCategories, createCategory } from "$lib/api/categories";
    import { RichEditor } from "$lib/components/rich-editor";
    import {
        CategoryDropdown,
        ColorPicker,
        ConfirmDialog,
    } from "$lib/components/ui";
    import { cn } from "$lib/utils";

    import {
        Sparkles,
        Check,
        X,
        Plus,
        ThumbsUp,
        ThumbsDown,
        Clock,
        CircleAlert,
        Save,
        Tag,
        Calendar,
        CircleCheck,
        Circle,
    } from "lucide-svelte";

    import { onMount, onDestroy } from "svelte";

    interface Props {
        open?: boolean;
        task?: Task | null;
        initialCategoryId?: string;
        lastTaskEndTime?: Date | null;
        onClose?: () => void;
    }

    let {
        open = $bindable(false),
        task = null,
        initialCategoryId,
        lastTaskEndTime = null,
        onClose,
    }: Props = $props();

    const queryClient = useQueryClient();

    // Form state
    let title = $state("");
    let journal = $state("");
    let categoryId = $state<string | undefined>(undefined);
    let priority = $state(3);
    let positives = $state<TaskItem[]>([]);
    let negatives = $state<TaskItem[]>([]);
    let newPositive = $state("");
    let newNegative = $state("");
    let completed = $state(false);

    // Date/time state (times include seconds HH:MM:SS)
    let startDate = $state("");
    let endDate = $state("");
    let startTime = $state("09:00:00");
    let endTime = $state("10:00:00");

    // Category creation
    let newCategoryName = $state("");
    let newCategoryColor = $state("#6366f1");
    let showNewCategory = $state(false);
    let useCustomCategoryColor = $state(false);

    // Uncomplete confirmation
    let showUncompleteConfirm = $state(false);

    // Live time tracking (only for create mode)
    let liveEndTime = $state(false);
    let useLastTaskStart = $state(false);
    let timeUpdateInterval: ReturnType<typeof setInterval> | null = null;

    // Function to update end time/date to now
    function updateEndTimeToNow() {
        const now = new Date();
        endDate = now.toISOString().split("T")[0];
        endTime = now.toTimeString().slice(0, 8);
    }

    // Effect to manage the live time interval
    $effect(() => {
        if (open && !task && liveEndTime) {
            // Start the interval
            updateEndTimeToNow();
            timeUpdateInterval = setInterval(updateEndTimeToNow, 1000);
        } else {
            // Clear the interval
            if (timeUpdateInterval) {
                clearInterval(timeUpdateInterval);
                timeUpdateInterval = null;
            }
        }
    });

    // Clean up interval on destroy
    onDestroy(() => {
        if (timeUpdateInterval) {
            clearInterval(timeUpdateInterval);
        }
    });

    // Effect to set start from last task
    $effect(() => {
        if (open && !task && useLastTaskStart && lastTaskEndTime) {
            const lastEnd = new Date(lastTaskEndTime);
            startDate = lastEnd.toISOString().split("T")[0];
            startTime = lastEnd.toTimeString().slice(0, 8);
        }
    });

    const categoriesQuery = createQuery({
        queryKey: ["categories"],
        queryFn: () => getCategories({ limit: 100 }),
        retry: false,
    });

    const categories = $derived($categoriesQuery.data?.items || []);
    const selectedCategory = $derived(
        categories.find((c) => c.id === categoryId),
    );

    const isEditing = $derived(!!task);

    const createMut = createMutation({
        mutationFn: (data: CreateTaskRequest) => createTask(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["tasks"] });
            resetForm();
            onClose?.();
        },
    });

    const updateMut = createMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateTaskRequest }) =>
            updateTask(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["tasks"] });
            resetForm();
            onClose?.();
        },
    });

    const createCategoryMut = createMutation({
        mutationFn: (data: { name: string; color: string }) =>
            createCategory(data),
        onSuccess: (newCat) => {
            queryClient.invalidateQueries({ queryKey: ["categories"] });
            categoryId = newCat.id;
            newCategoryName = "";
            newCategoryColor = "#6366f1";
            useCustomCategoryColor = false;
            showNewCategory = false;
        },
    });

    const isPending = $derived($createMut.isPending || $updateMut.isPending);

    function getTodayString() {
        const today = new Date();
        return today.toISOString().split("T")[0];
    }

    $effect(() => {
        if (open && task) {
            resetForm();
            title = task.title;
            journal = task.journal || "";
            categoryId = task.category?.id;
            priority = task.priority || 3;
            positives = task.positives || [];
            negatives = task.negatives || [];
            completed = task.completed || false;

            const startDateObj = new Date(task.start_date);
            const endDateObj = new Date(task.end_date);

            startDate = startDateObj.toISOString().split("T")[0];
            endDate = endDateObj.toISOString().split("T")[0];
            // Include seconds in time (HH:MM:SS)
            startTime = startDateObj.toTimeString().slice(0, 8);
            endTime = endDateObj.toTimeString().slice(0, 8);
        } else if (open) {
            resetForm();
            // Apply initial category if provided
            if (initialCategoryId) {
                categoryId = initialCategoryId;
            }
            if (categories.length === 0 && !$categoriesQuery.isLoading) {
                showNewCategory = true;
            }
        }
    });

    $effect(() => {
        if (
            open &&
            !task &&
            categories.length === 0 &&
            !$categoriesQuery.isLoading
        ) {
            showNewCategory = true;
        }
    });

    function resetForm() {
        title = "";
        journal = "";
        categoryId = undefined;
        priority = 3;
        positives = [];
        negatives = [];
        newPositive = "";
        newNegative = "";
        completed = false;
        startDate = getTodayString();
        endDate = getTodayString();
        startTime = "09:00:00";
        endTime = "10:00:00";
        liveEndTime = false;
        useLastTaskStart = false;
    }

    // Handle completion toggle
    function handleCompletionToggle() {
        completed = !completed;
    }

    function addPositive() {
        if (newPositive.trim()) {
            positives = [...positives, { text: newPositive.trim() }];
            newPositive = "";
        }
    }

    function addNegative() {
        if (newNegative.trim()) {
            negatives = [...negatives, { text: newNegative.trim() }];
            newNegative = "";
        }
    }

    function removePositive(index: number) {
        positives = positives.filter((_, i) => i !== index);
    }
    function removeNegative(index: number) {
        negatives = negatives.filter((_, i) => i !== index);
    }

    function handleSubmit() {
        if (!title.trim() || !startDate || !endDate) return;

        const [startYear, startMonth, startDay] = startDate
            .split("-")
            .map(Number);
        const startTimeParts = startTime.split(":").map(Number);
        const startHour = startTimeParts[0] || 0;
        const startMinute = startTimeParts[1] || 0;
        const startSecond = startTimeParts[2] || 0;

        const [endYear, endMonth, endDay] = endDate.split("-").map(Number);
        const endTimeParts = endTime.split(":").map(Number);
        const endHour = endTimeParts[0] || 0;
        const endMinute = endTimeParts[1] || 0;
        const endSecond = endTimeParts[2] || 0;

        const startDateObj = new Date(
            startYear,
            startMonth - 1,
            startDay,
            startHour,
            startMinute,
            startSecond,
        );
        const endDateObj = new Date(
            endYear,
            endMonth - 1,
            endDay,
            endHour,
            endMinute,
            endSecond,
        );

        const data = {
            title: title.trim(),
            journal: journal || undefined,
            start_date: startDateObj.toISOString(),
            end_date: endDateObj.toISOString(),
            category_id: categoryId || undefined,
            priority: priority || undefined,
            positives: positives.length > 0 ? positives : undefined,
            negatives: negatives.length > 0 ? negatives : undefined,
            completed: completed, // Add completed here
        };

        if (isEditing && task) {
            // Include completed status when updating
            $updateMut.mutate({ id: task.id, data: { ...data, completed } });
        } else {
            $createMut.mutate(data);
        }
    }

    function handleClose() {
        if (!isPending) {
            open = false;
            onClose?.();
        }
    }

    function handleCreateCategory() {
        if (newCategoryName.trim()) {
            $createCategoryMut.mutate({
                name: newCategoryName.trim(),
                color: newCategoryColor,
            });
        }
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
</script>

<!-- DaisyUI Modal -->
<dialog class="modal modal-bottom sm:modal-middle" class:modal-open={open}>
    <div
        class="modal-box max-w-5xl h-[85vh] max-h-[800px] p-0 flex flex-col overflow-hidden"
    >
        <!-- Header -->
        <div
            class="flex items-center justify-between px-6 py-4 border-b border-base-300 bg-base-200/50"
        >
            <div class="flex items-center gap-3">
                <!-- Completion Checkbox -->
                <button
                    class={cn(
                        "w-10 h-10 rounded-xl flex items-center justify-center transition-all duration-200 cursor-pointer",
                        completed
                            ? "bg-success text-success-content hover:bg-success/80"
                            : "bg-base-300 hover:bg-primary/10 hover:text-primary",
                    )}
                    onclick={handleCompletionToggle}
                    title={completed
                        ? "Mark as incomplete"
                        : "Mark as complete"}
                >
                    {#if completed}
                        <CircleCheck class="w-5 h-5" />
                    {:else}
                        <Circle class="w-5 h-5" />
                    {/if}
                </button>
                <div>
                    <div class="flex items-center gap-2">
                        <h3
                            class={cn(
                                "font-bold text-lg",
                                completed && "text-success",
                            )}
                        >
                            {isEditing ? "Edit Task" : "Create New Task"}
                        </h3>
                        {#if completed}
                            <span class="badge badge-success badge-sm gap-1">
                                <Check class="w-3 h-3" />
                                Completed
                            </span>
                        {/if}
                    </div>
                </div>
            </div>
            <button
                class="btn btn-ghost btn-sm btn-circle"
                onclick={handleClose}
            >
                <X class="w-5 h-5" />
            </button>
        </div>

        <!-- Content -->
        <div class="flex flex-1 overflow-hidden">
            <!-- Left: Title + Journal -->
            <div
                class="flex-1 flex flex-col p-6 overflow-y-auto overflow-x-hidden"
            >
                <!-- Title Input with Category Tag -->
                <div class="flex items-center gap-3">
                    <input
                        type="text"
                        bind:value={title}
                        placeholder="What needs to be done?"
                        class="input input-ghost text-2xl md:text-3xl font-bold flex-1 px-0 focus:outline-none focus:bg-transparent placeholder:text-base-content/20"
                    />
                    {#if selectedCategory}
                        <span
                            class="badge badge-lg shrink-0"
                            style="background-color: {selectedCategory.color}; color: white; border: none;"
                        >
                            {selectedCategory.name}
                        </span>
                    {/if}
                </div>

                <div class="divider my-3"></div>

                <!-- Rich Text Editor -->
                <div class="flex-1 min-h-0">
                    <RichEditor
                        bind:value={journal}
                        placeholder="Add details, notes, or reflections..."
                        minHeight="100%"
                        showToolbar={true}
                        class="h-full"
                    />
                </div>
            </div>

            <!-- Right: Meta Sidebar -->
            <div
                class="w-[380px] flex-shrink-0 bg-base-200/30 border-l border-base-300 p-5 overflow-y-auto"
            >
                <!-- Category Selector -->
                <div class="form-control mb-5">
                    <div class="label pb-1">
                        <span
                            class="label-text text-xs font-semibold uppercase opacity-50 flex items-center gap-1.5"
                        >
                            <Tag class="w-3 h-3" />
                            Category
                        </span>
                    </div>
                    {#if showNewCategory}
                        <div
                            class="space-y-3 p-3 rounded-lg bg-base-100 border border-base-300"
                        >
                            <input
                                type="text"
                                bind:value={newCategoryName}
                                placeholder="Category name (max 40 chars)"
                                class={cn(
                                    "input input-sm input-bordered w-full",
                                    newCategoryName.length > 40 &&
                                        "input-error",
                                )}
                                maxlength={40}
                            />
                            <!-- Custom color toggle -->
                            <div class="flex items-center justify-between">
                                <span class="text-xs opacity-60">Color</span>
                                <label
                                    class="flex items-center gap-2 cursor-pointer"
                                >
                                    <span class="text-xs opacity-60"
                                        >Custom</span
                                    >
                                    <input
                                        type="checkbox"
                                        class="toggle toggle-xs toggle-primary"
                                        bind:checked={useCustomCategoryColor}
                                    />
                                </label>
                            </div>
                            <ColorPicker
                                bind:value={newCategoryColor}
                                customMode={useCustomCategoryColor}
                                size="sm"
                                class="mt-2"
                            />
                            <!-- Preview -->
                            <div
                                class="flex items-center gap-2 text-xs opacity-60"
                            >
                                <span>Preview:</span>
                                <span
                                    class="badge badge-sm"
                                    style="background-color: {newCategoryColor}; color: white; border: none;"
                                >
                                    {newCategoryName || "Category"}
                                </span>
                            </div>
                            <div class="flex gap-2">
                                <button
                                    class="btn btn-sm btn-primary flex-1 gap-1"
                                    onclick={handleCreateCategory}
                                    disabled={$createCategoryMut.isPending ||
                                        !newCategoryName.trim() ||
                                        newCategoryName.length > 40}
                                >
                                    {#if $createCategoryMut.isPending}
                                        <span
                                            class="loading loading-spinner loading-xs"
                                        ></span>
                                    {:else}
                                        <Check class="w-3.5 h-3.5" />
                                    {/if}
                                    Create
                                </button>
                                <button
                                    class="btn btn-sm btn-ghost"
                                    onclick={() => (showNewCategory = false)}
                                >
                                    Cancel
                                </button>
                            </div>
                        </div>
                    {:else}
                        <CategoryDropdown
                            {categories}
                            bind:value={categoryId}
                            placeholder="Select category..."
                            showCreateButton={true}
                            onCreate={() => (showNewCategory = true)}
                        />
                    {/if}
                </div>

                <!-- Schedule -->
                <div class="form-control mb-5">
                    <label class="label pb-1">
                        <span
                            class="label-text text-xs font-semibold uppercase opacity-50 flex items-center gap-1.5"
                        >
                            <Calendar class="w-3 h-3" />
                            Schedule
                        </span>
                    </label>
                    <div
                        class="bg-base-100 rounded-lg border border-base-300 p-3 space-y-3"
                    >
                        <!-- Quick toggles (only in create mode) -->
                        {#if !isEditing}
                            <div
                                class="flex flex-wrap gap-2 pb-2 border-b border-base-200"
                            >
                                {#if lastTaskEndTime}
                                    <label
                                        class="flex items-center gap-2 cursor-pointer"
                                    >
                                        <input
                                            type="checkbox"
                                            class="toggle toggle-xs toggle-primary"
                                            bind:checked={useLastTaskStart}
                                        />
                                        <span
                                            class="text-xs {useLastTaskStart
                                                ? 'text-primary font-medium'
                                                : 'opacity-60'}"
                                        >
                                            From last task
                                        </span>
                                    </label>
                                {/if}
                                <label
                                    class="flex items-center gap-2 cursor-pointer ml-auto"
                                >
                                    <input
                                        type="checkbox"
                                        class="toggle toggle-xs toggle-success"
                                        bind:checked={liveEndTime}
                                    />
                                    <span
                                        class="text-xs flex items-center gap-1 {liveEndTime
                                            ? 'text-success font-medium'
                                            : 'opacity-60'}"
                                    >
                                        {#if liveEndTime}
                                            <span
                                                class="w-1.5 h-1.5 rounded-full bg-success animate-pulse"
                                            ></span>
                                        {/if}
                                        End at Now
                                    </span>
                                </label>
                            </div>
                        {/if}

                        <div class="grid grid-cols-2 gap-3">
                            <div class="form-control">
                                <label class="label py-0 pb-1" for="start-date">
                                    <span class="label-text text-xs opacity-50"
                                        >Start Date</span
                                    >
                                </label>
                                <input
                                    id="start-date"
                                    type="date"
                                    bind:value={startDate}
                                    class="input input-sm input-bordered w-full"
                                    disabled={useLastTaskStart}
                                />
                            </div>
                            <div class="form-control">
                                <label class="label py-0 pb-1" for="end-date">
                                    <span class="label-text text-xs opacity-50"
                                        >End Date</span
                                    >
                                </label>
                                <input
                                    id="end-date"
                                    type="date"
                                    bind:value={endDate}
                                    class="input input-sm input-bordered w-full {liveEndTime
                                        ? 'input-success'
                                        : ''}"
                                    disabled={liveEndTime}
                                />
                            </div>
                        </div>
                        <div class="grid grid-cols-2 gap-3">
                            <div class="form-control">
                                <label class="label py-0 pb-1" for="start-time">
                                    <span
                                        class="label-text text-xs opacity-50 flex items-center gap-1"
                                    >
                                        <Clock class="w-3 h-3" />
                                        From
                                    </span>
                                </label>
                                <input
                                    id="start-time"
                                    type="time"
                                    step="1"
                                    bind:value={startTime}
                                    class="input input-sm input-bordered w-full"
                                    disabled={useLastTaskStart}
                                />
                            </div>
                            <div class="form-control">
                                <label class="label py-0 pb-1" for="end-time">
                                    <span
                                        class="label-text text-xs opacity-50 flex items-center gap-1"
                                    >
                                        <Clock class="w-3 h-3" />
                                        To
                                        {#if liveEndTime}
                                            <span
                                                class="badge badge-xs badge-success"
                                                >LIVE</span
                                            >
                                        {/if}
                                    </span>
                                </label>
                                <input
                                    id="end-time"
                                    type="time"
                                    step="1"
                                    bind:value={endTime}
                                    class="input input-sm input-bordered w-full {liveEndTime
                                        ? 'input-success font-mono'
                                        : ''}"
                                    disabled={liveEndTime}
                                />
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Priority -->
                <div class="form-control mb-5">
                    <label class="label pb-1">
                        <span
                            class="label-text text-xs font-semibold uppercase opacity-50 flex items-center gap-1.5"
                        >
                            <CircleAlert class="w-3 h-3" />
                            Priority
                        </span>
                        <span
                            class={cn(
                                "badge badge-sm",
                                priorityColors[priority],
                            )}
                        >
                            {priorityLabels[priority]}
                        </span>
                    </label>
                    <input
                        type="range"
                        min="0"
                        max="5"
                        bind:value={priority}
                        class="range range-sm range-primary"
                        step="1"
                    />
                    <div
                        class="flex justify-between text-[10px] px-0.5 mt-1 opacity-40"
                    >
                        <span>None</span>
                        <span>Low</span>
                        <span>Med</span>
                        <span>High</span>
                        <span>Crit</span>
                        <span>Urg</span>
                    </div>
                </div>

                <!-- Reflection -->
                <div class="form-control">
                    <label class="label pb-1">
                        <span
                            class="label-text text-xs font-semibold uppercase opacity-50 flex items-center gap-1.5"
                        >
                            <Sparkles class="w-3 h-3" />
                            Reflection
                        </span>
                    </label>

                    <!-- Positives -->
                    <div class="mb-3">
                        <div class="join w-full">
                            <input
                                type="text"
                                bind:value={newPositive}
                                placeholder="Add a win..."
                                class="input input-sm input-bordered join-item flex-1 bg-success/5 border-success/30 focus:border-success"
                                onkeydown={(e) =>
                                    e.key === "Enter" &&
                                    (e.preventDefault(), addPositive())}
                            />
                            <button
                                class="btn btn-sm btn-ghost join-item text-success"
                                onclick={addPositive}
                            >
                                <Plus class="w-4 h-4" />
                            </button>
                        </div>
                        {#if positives.length > 0}
                            <div class="flex flex-wrap gap-1.5 mt-2">
                                {#each positives as p, i}
                                    <span
                                        class="badge badge-sm bg-success/10 text-success gap-1 pr-1 animate-fade-in"
                                    >
                                        <ThumbsUp class="w-2.5 h-2.5" />
                                        {p.text}
                                        <button
                                            onclick={() => removePositive(i)}
                                            class="hover:bg-success/20 rounded p-0.5"
                                        >
                                            <X class="w-2.5 h-2.5" />
                                        </button>
                                    </span>
                                {/each}
                            </div>
                        {/if}
                    </div>

                    <!-- Negatives -->
                    <div>
                        <div class="join w-full">
                            <input
                                type="text"
                                bind:value={newNegative}
                                placeholder="Add an improvement..."
                                class="input input-sm input-bordered join-item flex-1 bg-error/5 border-error/30 focus:border-error"
                                onkeydown={(e) =>
                                    e.key === "Enter" &&
                                    (e.preventDefault(), addNegative())}
                            />
                            <button
                                class="btn btn-sm btn-ghost join-item text-error"
                                onclick={addNegative}
                            >
                                <Plus class="w-4 h-4" />
                            </button>
                        </div>
                        {#if negatives.length > 0}
                            <div class="flex flex-wrap gap-1.5 mt-2">
                                {#each negatives as n, i}
                                    <span
                                        class="badge badge-sm bg-error/10 text-error gap-1 pr-1 animate-fade-in"
                                    >
                                        <ThumbsDown class="w-2.5 h-2.5" />
                                        {n.text}
                                        <button
                                            onclick={() => removeNegative(i)}
                                            class="hover:bg-error/20 rounded p-0.5"
                                        >
                                            <X class="w-2.5 h-2.5" />
                                        </button>
                                    </span>
                                {/each}
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
        </div>

        <!-- Footer -->
        <div
            class="flex items-center justify-between gap-3 px-6 py-4 border-t border-base-300 bg-base-200/50"
        >
            <div class="text-xs opacity-40">
                <kbd class="kbd kbd-xs">⌘</kbd> +
                <kbd class="kbd kbd-xs">Enter</kbd> to save
            </div>
            <div class="flex items-center gap-2">
                <button
                    class="btn btn-ghost btn-sm"
                    onclick={handleClose}
                    disabled={isPending}
                >
                    Cancel
                </button>
                <button
                    class="btn btn-primary btn-sm gap-2 min-w-[100px]"
                    onclick={handleSubmit}
                    disabled={isPending || !title.trim()}
                >
                    {#if isPending}
                        <span class="loading loading-spinner loading-xs"></span>
                        Saving...
                    {:else}
                        <Save class="w-4 h-4" />
                        {isEditing ? "Save" : "Create"}
                    {/if}
                </button>
            </div>
        </div>
    </div>
    <form method="dialog" class="modal-backdrop bg-base-content/20">
        <button onclick={handleClose}>close</button>
    </form>
</dialog>
