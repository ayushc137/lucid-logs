<script lang="ts">
    import {
        createMutation,
        useQueryClient,
        createQuery,
    } from "@tanstack/svelte-query";
    import {
        createTask,
        updateTask,
        deleteTask,
        type Task,
        type CreateTaskRequest,
        type UpdateTaskRequest,
        type TaskItem,
        getLastTaskEndTime,
    } from "$lib/api";
    import { getCategories, createCategory } from "$lib/api/categories";
    import { getEmotion, type Emotion, inferEmotion, type InferredEmotion } from "$lib/api/emotions";
    import { RichEditor } from "$lib/components/rich-editor";
    import {
        CategoryDropdown,
        ColorPicker,
        ConfirmDialog,
        Card,
        SectionHeader,
    } from "$lib/components/ui";
    import { EmotionBadge } from "$lib/components/emotions";
    import { EmotionModal } from "$lib/components/emotions";
    import { cn } from "$lib/utils";
    import { goto } from "$app/navigation";
    import { browser } from "$app/environment";

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
        Heart,
        ChevronRight,
        Trash2,
        Edit3,
        FileText,
        Info,
    } from "lucide-svelte";

    import { onDestroy } from "svelte";

    // Context for emotion selection
    type EmotionSelectionContext = {
        type: "task" | "positive" | "negative";
        index?: number;
    };

    interface Props {
        /** Task to edit (null for create mode) */
        task?: Task | null;
        /** Initial category ID for new tasks */
        initialCategoryId?: string;
        /** Referrer URL to navigate back to */
        referrer?: string;
        /** Called after successful save */
        onSuccess?: () => void;
    }

    let {
        task = null,
        initialCategoryId,
        referrer,
        onSuccess,
    }: Props = $props();

    const queryClient = useQueryClient();
    const isEditing = $derived(!!task);

    // Fetch last task end time only in create mode
    const queryOptions = $derived({
        queryKey: ["tasks", "last-end-time"],
        queryFn: getLastTaskEndTime,
        enabled: !isEditing,
    });

    const lastTaskEndTimeQuery = createQuery(queryOptions);

    const lastTaskEndTime = $derived(
        $lastTaskEndTimeQuery.data?.end_time
            ? new Date($lastTaskEndTimeQuery.data.end_time)
            : null,
    );

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

    // Emotion state
    let selectedEmotion = $state<Emotion | null>(null);
    let emotionContext = $state<EmotionSelectionContext | null>(null);
    let emotionModalOpen = $state(false);
    let liveInferredEmotion = $state<InferredEmotion | null>(null);

    // Date/time state
    let startDate = $state("");
    let endDate = $state("");
    let startTime = $state("09:00:00");
    let endTime = $state("10:00:00");

    // Category creation
    let newCategoryName = $state("");
    let newCategoryColor = $state("#6366f1");
    let showNewCategory = $state(false);
    let useCustomCategoryColor = $state(false);

    // Confirmation dialogs
    let showUncompleteConfirm = $state(false);
    let showDeleteConfirm = $state(false);

    // Live time tracking (only for create mode)
    let liveEndTime = $state(false);
    let useLastTaskStart = $state(false);
    let timeUpdateInterval: ReturnType<typeof setInterval> | null = null;

    // Track if form is initialized
    let formInitialized = $state(false);

    function getTodayString() {
        return new Date().toISOString().split("T")[0];
    }

    function updateEndTimeToNow() {
        const now = new Date();
        endDate = now.toISOString().split("T")[0];
        endTime = now.toTimeString().slice(0, 8);
    }

    // Initialize form based on mode
    $effect(() => {
        if (formInitialized) return;

        if (task) {
            // Edit mode - populate from task
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
            startTime = startDateObj.toTimeString().slice(0, 8);
            endTime = endDateObj.toTimeString().slice(0, 8);

            if (task.emotion_id) {
                loadTaskEmotion(task.emotion_id);
            }
        } else {
            // Create mode - set defaults
            const today = getTodayString();
            startDate = today;
            endDate = today;
            if (initialCategoryId) {
                categoryId = initialCategoryId;
            }
        }
        formInitialized = true;
    });

    // Live time interval for create mode
    $effect(() => {
        if (!isEditing && liveEndTime) {
            updateEndTimeToNow();
            timeUpdateInterval = setInterval(updateEndTimeToNow, 1000);
        } else if (timeUpdateInterval) {
            clearInterval(timeUpdateInterval);
            timeUpdateInterval = null;
        }
    });

    // Effect to set start from last task
    $effect(() => {
        if (!isEditing && useLastTaskStart && lastTaskEndTime) {
            const lastEnd = new Date(lastTaskEndTime);
            startDate = lastEnd.toISOString().split("T")[0];
            startTime = lastEnd.toTimeString().slice(0, 8);
        }
    });

    // Effect to calculate live inferred emotion
    $effect(() => {
        // Only calculate if we have items with emotions
        const hasEmotions = positives.some(p => p.emotion_id) || negatives.some(n => n.emotion_id);

        if (hasEmotions) {
            inferEmotion({ positives, negatives })
                .then(response => {
                    liveInferredEmotion = response.inferred_emotion;
                })
                .catch(error => {
                    console.error("Failed to infer emotion:", error);
                    liveInferredEmotion = null;
                });
        } else {
            liveInferredEmotion = null;
        }
    });

    onDestroy(() => {
        if (timeUpdateInterval) clearInterval(timeUpdateInterval);
    });

    // Queries
    const categoriesQuery = createQuery({
        queryKey: ["categories"],
        queryFn: () => getCategories({ limit: 100 }),
        retry: false,
    });

    const categories = $derived($categoriesQuery.data?.items || []);
    const selectedCategory = $derived(
        categories.find((c) => c.id === categoryId),
    );

    // Mutations
    const createMut = createMutation({
        mutationFn: (data: CreateTaskRequest) => createTask(data),
        onSuccess: handleSuccess,
    });

    const updateMut = createMutation({
        mutationFn: (data: UpdateTaskRequest) => updateTask(task!.id, data),
        onSuccess: handleSuccess,
    });

    const deleteMut = createMutation({
        mutationFn: () => deleteTask(task!.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["tasks"] });
            navigateBack();
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
    const inferredEmotion = $derived(liveInferredEmotion || task?.inferred_emotion);

    function handleSuccess() {
        queryClient.invalidateQueries({ queryKey: ["tasks"] });
        onSuccess?.();
        navigateBack();
    }

    function navigateBack() {
        if (referrer) {
            goto(referrer);
        } else {
            goto("/tasks");
        }
    }

    async function loadTaskEmotion(emotionId: string) {
        try {
            selectedEmotion = await getEmotion(emotionId);
        } catch (e) {
            console.error("Failed to load task emotion:", e);
        }
    }

    // Emotion handlers
    function handleEmotionSelect(emotion: Emotion) {
        if (emotionContext?.type === "task") {
            selectedEmotion = emotion;
        } else if (
            emotionContext?.type === "positive" &&
            emotionContext.index !== undefined
        ) {
            positives = positives.map((p, i) =>
                i === emotionContext!.index
                    ? { ...p, emotion_id: emotion.id }
                    : p,
            );
        } else if (
            emotionContext?.type === "negative" &&
            emotionContext.index !== undefined
        ) {
            negatives = negatives.map((n, i) =>
                i === emotionContext!.index
                    ? { ...n, emotion_id: emotion.id }
                    : n,
            );
        }
        emotionContext = null;
        emotionModalOpen = false;
    }

    function openEmotionForTask() {
        emotionContext = { type: "task" };
        emotionModalOpen = true;
    }

    function openEmotionForPositive(index: number) {
        emotionContext = { type: "positive", index };
        emotionModalOpen = true;
    }

    function openEmotionForNegative(index: number) {
        emotionContext = { type: "negative", index };
        emotionModalOpen = true;
    }

    function clearTaskEmotion() {
        selectedEmotion = null;
    }

    function clearPositiveEmotion(index: number) {
        positives = positives.map((p, i) =>
            i === index ? { ...p, emotion_id: undefined } : p,
        );
    }

    function clearNegativeEmotion(index: number) {
        negatives = negatives.map((n, i) =>
            i === index ? { ...n, emotion_id: undefined } : n,
        );
    }

    // Completion toggle
    function handleCompletionToggle() {
        if (completed && isEditing) {
            showUncompleteConfirm = true;
        } else {
            completed = !completed;
        }
    }

    function confirmUncomplete() {
        completed = false;
        showUncompleteConfirm = false;
    }

    // Reflection items
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

    // Category
    function handleCreateCategory() {
        if (newCategoryName.trim()) {
            $createCategoryMut.mutate({
                name: newCategoryName.trim(),
                color: newCategoryColor,
            });
        }
    }

    // Submit
    function handleSubmit() {
        if (!title.trim() || !startDate || !endDate) return;

        const [startYear, startMonth, startDay] = startDate
            .split("-")
            .map(Number);
        const [startHour, startMinute, startSecond] = startTime
            .split(":")
            .map(Number);
        const [endYear, endMonth, endDay] = endDate.split("-").map(Number);
        const [endHour, endMinute, endSecond] = endTime.split(":").map(Number);

        const startDateObj = new Date(
            startYear,
            startMonth - 1,
            startDay,
            startHour || 0,
            startMinute || 0,
            startSecond || 0,
        );
        const endDateObj = new Date(
            endYear,
            endMonth - 1,
            endDay,
            endHour || 0,
            endMinute || 0,
            endSecond || 0,
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
            completed: completed,
            emotion_id: selectedEmotion?.id || undefined,
        };

        if (isEditing) {
            $updateMut.mutate(data);
        } else {
            $createMut.mutate(data);
        }
    }

    function handleDelete() {
        $deleteMut.mutate();
        showDeleteConfirm = false;
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

<div class="flex flex-col min-h-[calc(100vh-6rem)] relative">
    <div class="space-y-6 flex-1 pb-4">
        <!-- Main Content Grid -->
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <!-- Left Column: Title + Journal (2/3 width on desktop) -->
            <div class="lg:col-span-2 space-y-4">
                <!-- Title Card -->
                <Card variant="bordered">
                    <div class="flex items-center gap-3">
                        <!-- Completion Toggle -->
                        <button
                            class={cn(
                                "w-10 h-10 rounded-xl flex items-center justify-center transition-all duration-200 cursor-pointer shrink-0 hover:scale-105",
                                completed
                                    ? "bg-success text-success-content hover:bg-success/80 shadow-sm"
                                    : "bg-base-200 hover:bg-primary/10 hover:text-primary",
                            )}
                            onclick={handleCompletionToggle}
                            title={completed
                                ? "Mark as incomplete"
                                : "Mark as complete"}
                        >
                            {#if completed}
                                <CircleCheck
                                    class="w-5 h-5 transition-transform duration-200"
                                />
                            {:else}
                                <Circle
                                    class="w-5 h-5 transition-transform duration-200"
                                />
                            {/if}
                        </button>

                        <!-- Title Input -->
                        <input
                            type="text"
                            bind:value={title}
                            placeholder="What needs to be done?"
                            class="input input-ghost text-xl md:text-2xl font-bold flex-1 px-0 focus:outline-none focus:bg-transparent placeholder:text-base-content/20 transition-all duration-200"
                        />

                        <!-- Category Badge -->
                        {#if selectedCategory}
                            <span
                                class="badge badge-lg shrink-0 transition-all duration-200 animate-fade-in"
                                style="background-color: {selectedCategory.color}; color: white; border: none;"
                            >
                                {selectedCategory.name}
                            </span>
                        {/if}

                        <!-- Completed Badge -->
                        {#if completed}
                            <span
                                class="badge badge-success gap-1 transition-all duration-200 animate-fade-in"
                            >
                                <Check class="w-3 h-3" />
                                Done
                            </span>
                        {/if}
                    </div>
                </Card>

                <!-- Journal Card - Expanded -->
                <Card
                    variant="bordered"
                    class="min-h-[500px] flex flex-col transition-all duration-200 hover:shadow-md"
                >
                    <div class="flex items-center gap-2 mb-3">
                        <FileText class="w-4 h-4 opacity-50" />
                        <span class="text-xs font-semibold uppercase opacity-50"
                            >Notes & Details</span
                        >
                    </div>
                    <div class="flex-1">
                        <RichEditor
                            bind:value={journal}
                            placeholder="Add details, notes, or reflections..."
                            minHeight="420px"
                            showToolbar={true}
                        />
                    </div>
                </Card>
            </div>

            <!-- Right Column: Meta Fields (1/3 width on desktop) -->
            <div class="space-y-4">
                <!-- Category -->
                <Card
                    variant="bordered"
                    class="transition-all duration-200 hover:shadow-md"
                >
                    <div class="flex items-center gap-2 mb-3">
                        <Tag class="w-4 h-4 opacity-50" />
                        <span class="text-xs font-semibold uppercase opacity-50"
                            >Category</span
                        >
                    </div>
                    {#if showNewCategory}
                        <div class="space-y-3">
                            <input
                                type="text"
                                bind:value={newCategoryName}
                                placeholder="Category name"
                                class={cn(
                                    "input input-sm input-bordered w-full transition-all duration-200",
                                    newCategoryName.length > 40 &&
                                        "input-error",
                                )}
                                maxlength={40}
                            />
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
                            />
                            <div class="flex gap-2">
                                <button
                                    class="btn btn-sm btn-primary flex-1 gap-1 transition-all duration-200 hover:shadow-md"
                                    onclick={handleCreateCategory}
                                    disabled={$createCategoryMut.isPending ||
                                        !newCategoryName.trim()}
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
                                    class="btn btn-sm btn-ghost transition-all duration-200 hover:bg-base-200"
                                    onclick={() => (showNewCategory = false)}
                                    >Cancel</button
                                >
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
                </Card>

                <!-- Schedule -->
                <Card
                    variant="bordered"
                    class="transition-all duration-200 hover:shadow-md"
                >
                    <div class="flex items-center gap-2 mb-3">
                        <Calendar class="w-4 h-4 opacity-50" />
                        <span class="text-xs font-semibold uppercase opacity-50"
                            >Schedule</span
                        >
                    </div>

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

                        <div class="space-y-3">
                            <div class="grid grid-cols-2 gap-3">
                                <div class="form-control">
                                    <label
                                        class="label py-0 pb-1"
                                        for="start-date"
                                    >
                                        <span
                                            class="label-text text-xs opacity-50"
                                            >Start Date</span
                                        >
                                    </label>
                                    <input
                                        id="start-date"
                                        type="date"
                                        bind:value={startDate}
                                        class="input input-sm input-bordered w-full transition-all duration-200 focus:border-primary"
                                        disabled={useLastTaskStart}
                                    />
                                </div>
                                <div class="form-control">
                                    <label
                                        class="label py-0 pb-1"
                                        for="end-date"
                                    >
                                        <span
                                            class="label-text text-xs opacity-50"
                                            >End Date</span
                                        >
                                    </label>
                                    <input
                                        id="end-date"
                                        type="date"
                                        bind:value={endDate}
                                        class="input input-sm input-bordered w-full transition-all duration-200 focus:border-primary {liveEndTime
                                            ? 'input-success'
                                            : ''}"
                                        disabled={liveEndTime}
                                    />
                                </div>
                            </div>
                            <div class="grid grid-cols-2 gap-3">
                                <div class="form-control">
                                    <label
                                        class="label py-0 pb-1"
                                        for="start-time"
                                    >
                                        <span
                                            class="label-text text-xs opacity-50 flex items-center gap-1"
                                        >
                                            <Clock class="w-3 h-3" /> From
                                        </span>
                                    </label>
                                    <input
                                        id="start-time"
                                        type="time"
                                        step="1"
                                        bind:value={startTime}
                                        class="input input-sm input-bordered w-full transition-all duration-200 focus:border-primary"
                                        disabled={useLastTaskStart}
                                    />
                                </div>
                                <div class="form-control">
                                    <label
                                        class="label py-0 pb-1"
                                        for="end-time"
                                    >
                                        <span
                                            class="label-text text-xs opacity-50 flex items-center gap-1"
                                        >
                                            <Clock class="w-3 h-3" /> To
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
                                        class="input input-sm input-bordered w-full transition-all duration-200 focus:border-primary {liveEndTime
                                            ? 'input-success font-mono'
                                            : ''}"
                                        disabled={liveEndTime}
                                    />
                                </div>
                            </div>
                        </div>
                    </div>
                </Card>

                <!-- Priority -->
                <Card
                    variant="bordered"
                    class="transition-all duration-200 hover:shadow-md"
                >
                    <div class="flex items-center justify-between mb-3">
                        <div class="flex items-center gap-2">
                            <CircleAlert class="w-4 h-4 opacity-50" />
                            <span
                                class="text-xs font-semibold uppercase opacity-50"
                                >Priority</span
                            >
                        </div>
                        <span
                            class={cn(
                                "badge badge-sm",
                                priorityColors[priority],
                            )}>{priorityLabels[priority]}</span
                        >
                    </div>
                    <input
                        type="range"
                        min="0"
                        max="5"
                        bind:value={priority}
                        class="range range-sm range-primary transition-all duration-200"
                        step="1"
                    />
                    <div
                        class="flex justify-between text-[10px] px-0.5 mt-1 opacity-40"
                    >
                        {#each priorityLabels as label}
                            <span>{label.slice(0, 3)}</span>
                        {/each}
                    </div>
                </Card>

                <!-- Emotion -->
                <Card
                    variant="bordered"
                    class="transition-all duration-200 hover:shadow-md"
                >
                    <div class="flex items-center gap-2 mb-3">
                        <Heart class="w-4 h-4 opacity-50" />
                        <span class="text-xs font-semibold uppercase opacity-50"
                            >Emotion</span
                        >
                    </div>

                    <div class="space-y-3">
                        <!-- User Selected Emotion -->
                        <div>
                            <div
                                class="text-[10px] uppercase font-semibold text-base-content/40 mb-1.5"
                            >
                                Your Selection
                            </div>
                            {#if selectedEmotion}
                                <div
                                    class="transition-all duration-200 animate-fade-in"
                                >
                                    <EmotionBadge
                                        name={selectedEmotion.name}
                                        emoji={selectedEmotion.emoji}
                                        quadrant={selectedEmotion.quadrant}
                                        size="md"
                                        removable={true}
                                        onRemove={clearTaskEmotion}
                                        onclick={openEmotionForTask}
                                    />
                                </div>
                            {:else}
                                <button
                                    class="w-full rounded-lg border border-dashed border-base-300 p-3 flex items-center justify-center gap-2 hover:border-primary hover:bg-primary/5 transition-all duration-200 text-base-content/50 hover:text-primary"
                                    onclick={openEmotionForTask}
                                >
                                    <Plus class="w-4 h-4" />
                                    <span class="text-sm font-medium"
                                        >Select Emotion</span
                                    >
                                </button>
                            {/if}
                        </div>

                        <!-- Inferred Emotion (always show when available) -->
                        {#if inferredEmotion}
                            <div
                                class="pt-2 border-t border-base-200 transition-all duration-200"
                            >
                                <div class="flex items-center gap-1.5 mb-1.5">
                                    <Sparkles class="w-3 h-3 text-primary" />
                                    <span
                                        class="text-[10px] uppercase font-semibold text-primary"
                                        >Inferred Emotion</span
                                    >
                                    <div
                                        class="tooltip tooltip-right"
                                        data-tip="Calculated based on emotions from your reflections (positives & negatives)"
                                    >
                                        <Info
                                            class="w-3 h-3 text-primary/60 cursor-help"
                                        />
                                    </div>
                                </div>
                                <button
                                    class="w-full rounded-lg bg-primary/5 border border-primary/20 p-2.5 flex items-center justify-between gap-2 hover:bg-primary/10 transition-all duration-200 group text-left"
                                    onclick={openEmotionForTask}
                                >
                                    <span
                                        class="text-sm font-medium text-base-content/80"
                                    >
                                        {inferredEmotion.closest_emotion_name}
                                    </span>
                                    <ChevronRight
                                        class="w-3.5 h-3.5 text-primary/50 group-hover:text-primary transition-colors"
                                    />
                                </button>
                            </div>
                        {/if}
                    </div>
                </Card>

                <!-- Reflection -->
                <Card
                    variant="bordered"
                    class="transition-all duration-200 hover:shadow-md"
                >
                    <div class="flex items-center gap-2 mb-4">
                        <Sparkles class="w-4 h-4 opacity-50" />
                        <span class="text-xs font-semibold uppercase opacity-50"
                            >Reflection</span
                        >
                    </div>

                    <!-- Positives Section -->
                    <div class="mb-4">
                        <div class="flex items-center gap-2 mb-2">
                            <ThumbsUp class="w-3.5 h-3.5 text-success" />
                            <span class="text-xs font-medium text-success"
                                >What went well?</span
                            >
                        </div>
                        <div class="flex items-start gap-2">
                            <input
                                type="text"
                                bind:value={newPositive}
                                placeholder="Add a positive reflection..."
                                class="input input-sm input-bordered flex-1 bg-success/5 border-success/30 focus:border-success text-sm"
                                onkeydown={(e) => {
                                    if (e.key === "Enter") {
                                        e.preventDefault();
                                        addPositive();
                                    }
                                }}
                            />
                            <button
                                class="btn btn-sm btn-ghost text-success shrink-0"
                                onclick={addPositive}
                                disabled={!newPositive.trim()}
                                title="Add positive"
                            >
                                <Plus class="w-4 h-4" />
                            </button>
                        </div>
                        {#if positives.length > 0}
                            <div class="flex flex-col gap-2 mt-2">
                                {#each positives as p, i (i)}
                                    <div
                                        class="rounded-lg bg-success/10 border border-success/20 p-2.5 group hover:bg-success/15 transition-all duration-200 animate-fade-in"
                                    >
                                        <div class="flex items-start gap-2">
                                            <ThumbsUp
                                                class="w-3.5 h-3.5 text-success shrink-0 mt-0.5"
                                            />
                                            <p
                                                class="flex-1 text-sm text-success-content/90 leading-relaxed break-words"
                                            >
                                                {p.text}
                                            </p>
                                            <button
                                                onclick={() =>
                                                    removePositive(i)}
                                                class="btn btn-ghost btn-xs text-success/60 hover:text-success hover:bg-success/20 transition-all duration-200 shrink-0"
                                                title="Remove"
                                            >
                                                <X class="w-3 h-3" />
                                            </button>
                                        </div>
                                        <!-- Emotion for this positive -->
                                        <div class="mt-2 pl-5">
                                            {#if p.emotion_id}
                                                <button
                                                    class="text-xs text-success/70 hover:text-success flex items-center gap-1 transition-colors duration-200"
                                                    onclick={() =>
                                                        openEmotionForPositive(
                                                            i,
                                                        )}
                                                >
                                                    <Heart
                                                        class="w-2.5 h-2.5"
                                                    />
                                                    <span>Emotion set</span>
                                                    <X
                                                        class="w-2.5 h-2.5 ml-1 hover:text-error"
                                                        onclick={(e) => {
                                                            e.stopPropagation();
                                                            clearPositiveEmotion(
                                                                i,
                                                            );
                                                        }}
                                                    />
                                                </button>
                                            {:else}
                                                <button
                                                    class="text-[10px] text-base-content/40 hover:text-success flex items-center gap-1 transition-colors duration-200"
                                                    onclick={() =>
                                                        openEmotionForPositive(
                                                            i,
                                                        )}
                                                >
                                                    <Plus class="w-2.5 h-2.5" />
                                                    <span>Add emotion</span>
                                                </button>
                                            {/if}
                                        </div>
                                    </div>
                                {/each}
                            </div>
                        {/if}
                    </div>

                    <!-- Divider -->
                    <div class="border-t border-base-200 my-4"></div>

                    <!-- Negatives Section -->
                    <div>
                        <div class="flex items-center gap-2 mb-2">
                            <ThumbsDown class="w-3.5 h-3.5 text-error" />
                            <span class="text-xs font-medium text-error"
                                >What could improve?</span
                            >
                        </div>
                        <div class="flex items-start gap-2">
                            <input
                                type="text"
                                bind:value={newNegative}
                                placeholder="Add an area for improvement..."
                                class="input input-sm input-bordered flex-1 bg-error/5 border-error/30 focus:border-error text-sm"
                                onkeydown={(e) => {
                                    if (e.key === "Enter") {
                                        e.preventDefault();
                                        addNegative();
                                    }
                                }}
                            />
                            <button
                                class="btn btn-sm btn-ghost text-error shrink-0"
                                onclick={addNegative}
                                disabled={!newNegative.trim()}
                                title="Add improvement"
                            >
                                <Plus class="w-4 h-4" />
                            </button>
                        </div>
                        {#if negatives.length > 0}
                            <div class="flex flex-col gap-2 mt-2">
                                {#each negatives as n, i (i)}
                                    <div
                                        class="rounded-lg bg-error/10 border border-error/20 p-2.5 group hover:bg-error/15 transition-all duration-200 animate-fade-in"
                                    >
                                        <div class="flex items-start gap-2">
                                            <ThumbsDown
                                                class="w-3.5 h-3.5 text-error shrink-0 mt-0.5"
                                            />
                                            <p
                                                class="flex-1 text-sm text-error-content/90 leading-relaxed break-words"
                                            >
                                                {n.text}
                                            </p>
                                            <button
                                                onclick={() =>
                                                    removeNegative(i)}
                                                class="btn btn-ghost btn-xs text-error/60 hover:text-error hover:bg-error/20 transition-all duration-200 shrink-0"
                                                title="Remove"
                                            >
                                                <X class="w-3 h-3" />
                                            </button>
                                        </div>
                                        <!-- Emotion for this negative -->
                                        <div class="mt-2 pl-5">
                                            {#if n.emotion_id}
                                                <button
                                                    class="text-xs text-error/70 hover:text-error flex items-center gap-1 transition-colors duration-200"
                                                    onclick={() =>
                                                        openEmotionForNegative(
                                                            i,
                                                        )}
                                                >
                                                    <Heart
                                                        class="w-2.5 h-2.5"
                                                    />
                                                    <span>Emotion set</span>
                                                    <X
                                                        class="w-2.5 h-2.5 ml-1 hover:text-base-content"
                                                        onclick={(e) => {
                                                            e.stopPropagation();
                                                            clearNegativeEmotion(
                                                                i,
                                                            );
                                                        }}
                                                    />
                                                </button>
                                            {:else}
                                                <button
                                                    class="text-[10px] text-base-content/40 hover:text-error flex items-center gap-1 transition-colors duration-200"
                                                    onclick={() =>
                                                        openEmotionForNegative(
                                                            i,
                                                        )}
                                                >
                                                    <Plus class="w-2.5 h-2.5" />
                                                    <span>Add emotion</span>
                                                </button>
                                            {/if}
                                        </div>
                                    </div>
                                {/each}
                            </div>
                        {/if}
                    </div>
                </Card>
            </div>
        </div>
    </div>

    <!-- Sticky Footer Actions -->
    <div
        class="sticky bottom-0 -mx-4 md:-mx-6 lg:-mx-8 px-4 md:px-6 lg:px-8 py-3 mt-auto bg-base-100/95 backdrop-blur-md border-t border-base-300 shadow-lg z-30"
    >
        <div class="flex items-center justify-between gap-4">
            <div class="flex items-center gap-3">
                {#if isEditing}
                    <button
                        class="btn btn-ghost btn-sm text-error hover:bg-error/10 gap-2 transition-all duration-200"
                        onclick={() => (showDeleteConfirm = true)}
                        disabled={$deleteMut.isPending}
                    >
                        <Trash2 class="w-4 h-4" />
                        <span class="hidden sm:inline">Delete</span>
                    </button>
                {/if}
                <span class="text-xs text-base-content/40 hidden lg:inline">
                    <kbd class="kbd kbd-xs">⌘</kbd> +
                    <kbd class="kbd kbd-xs">Enter</kbd> to save
                </span>
            </div>
            <div class="flex items-center gap-2">
                <button
                    class="btn btn-ghost btn-sm hover:bg-base-200 transition-all duration-200"
                    onclick={navigateBack}
                    disabled={isPending}
                >
                    Cancel
                </button>
                <button
                    class="btn btn-primary btn-sm gap-2 min-w-[120px] shadow-md hover:shadow-lg shadow-primary/20 transition-all duration-200"
                    onclick={handleSubmit}
                    disabled={isPending || !title.trim()}
                >
                    {#if isPending}
                        <span class="loading loading-spinner loading-xs"></span>
                        {isEditing ? "Saving..." : "Creating..."}
                    {:else}
                        <Save class="w-4 h-4" />
                        {isEditing ? "Save Changes" : "Create Task"}
                    {/if}
                </button>
            </div>
        </div>
    </div>
</div>

<!-- Emotion Modal -->
<EmotionModal
    bind:open={emotionModalOpen}
    onSelect={handleEmotionSelect}
    onClose={() => {
        emotionModalOpen = false;
        emotionContext = null;
    }}
/>

<!-- Uncomplete Confirmation -->
<ConfirmDialog
    bind:open={showUncompleteConfirm}
    title="Mark as Incomplete?"
    message="Are you sure you want to mark this task as incomplete?"
    confirmText="Mark Incomplete"
    cancelText="Keep Complete"
    destructive={false}
    onConfirm={confirmUncomplete}
    onCancel={() => (showUncompleteConfirm = false)}
/>

<!-- Delete Confirmation -->
{#if isEditing}
    <ConfirmDialog
        bind:open={showDeleteConfirm}
        title="Delete Task?"
        message="This action cannot be undone."
        confirmText="Delete"
        destructive={true}
        loading={$deleteMut.isPending}
        onConfirm={handleDelete}
        onCancel={() => (showDeleteConfirm = false)}
    />
{/if}
