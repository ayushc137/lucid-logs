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
        type TaskTemplate,
    } from "$lib/api";
    import { getCategories, createCategory } from "$lib/api/categories";
    import {
        inferEmotion,
        type InferredEmotion,
        type Emotion,
    } from "$lib/api/emotions";
    import { emotionStore } from "$lib/stores/emotions.svelte";
    import {
        QUADRANT_COLORS,
        QUADRANT_META,
        type Quadrant,
    } from "$lib/components/emotions/emotionData";
    import OpenMoji from "$lib/components/ui/OpenMoji.svelte";
    import { RichEditor } from "$lib/components/rich-editor";
    import {
        CategoryDropdown,
        ColorPicker,
        ConfirmDialog,
        Card,
        SectionHeader,
    } from "$lib/components/ui";
    import { EmotionModal } from "$lib/components/emotions";
    import { GoalSelector } from "$lib/components/goals";
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
        Target,
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

    const lastTaskEndTimeQuery = createQuery(() => queryOptions);

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

    // Goal links state
    interface GoalLink {
        goal_id: string;
        impact_type: "positive" | "negative" | "neutral";
        impact_magnitude: number;
        quantity_value?: number;
        quantity_unit?: string;
    }
    let goalLinks = $state<GoalLink[]>([]);

    // Emotion state
    let selectedEmotion = $state<Emotion | null>(null);
    let emotionContext = $state<EmotionSelectionContext | null>(null);
    let emotionModalOpen = $state(false);
    let liveInferredEmotion = $state<InferredEmotion | null>(null);
    let inferredEmotionFull = $state<Emotion | null>(null);
    let inferringEmotion = $state(false);
    let inferenceError = $state<string | null>(null);
    /** Initial emotion to pre-select when modal opens (e.g. suggested emotion) */
    let initialEmotionForModal = $state<Emotion | null>(null);
    /** Allowed quadrants for emotion selection (null = all allowed) */
    let allowedQuadrantsForModal = $state<Quadrant[] | null>(null);

    // Pending emotion for new reflection item (select emotion FIRST)
    let pendingPositiveEmotion = $state<Emotion | null>(null);
    let pendingNegativeEmotion = $state<Emotion | null>(null);
    let pendingEmotionType = $state<"positive" | "negative" | null>(null);

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

    // Hash of reflections to check if they have changed
    function getReflectionsHash(p: TaskItem[], n: TaskItem[]) {
        return JSON.stringify({
            p: p.map((x) => ({ t: x.text, e: x.emotion_id })),
            n: n.map((x) => ({ t: x.text, e: x.emotion_id })),
        });
    }

    // Logic to prevent initial inference call
    let lastInferredHash = $state("");

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
            goalLinks = (task.linked_goals || []).map((l) => ({
                goal_id: l.goal_id,
                impact_type:
                    (l.impact_type as "positive" | "negative" | "neutral") ||
                    "positive",
                impact_magnitude: l.impact_magnitude,
                quantity_value: l.quantity_value,
                quantity_unit: l.quantity_unit,
            }));

            const startDateObj = new Date(task.start_date);
            const endDateObj = new Date(task.end_date);

            startDate = startDateObj.toISOString().split("T")[0];
            endDate = endDateObj.toISOString().split("T")[0];
            startTime = startDateObj.toTimeString().slice(0, 8);
            endTime = endDateObj.toTimeString().slice(0, 8);

            if (task.emotion_id) {
                // Try to load immediately from store
                const e = emotionStore.get(task.emotion_id);
                if (e) {
                    selectedEmotion = e;
                }
            }

            // Set initial hash to prevent immediate inference
            lastInferredHash = getReflectionsHash(positives, negatives);
        } else {
            // Create mode - set defaults
            const today = getTodayString();
            startDate = today;
            endDate = today;
            if (initialCategoryId) {
                categoryId = initialCategoryId;
            }

            // Check for template
            if (browser) {
                const templateJson = sessionStorage.getItem("task-template");
                if (templateJson) {
                    try {
                        const tmpl = JSON.parse(templateJson) as TaskTemplate;
                        title = tmpl.title;
                        if (tmpl.description) journal = tmpl.description;
                        if (tmpl.icon) title = `${tmpl.icon} ${title}`; // Optional: prepend icon?

                        if (tmpl.category) categoryId = tmpl.category.id;

                        // Calculate end time based on default duration
                        if (tmpl.default_duration) {
                            const start = new Date(); // Or whatever start time is set to (now by default)
                            // Actually start time is set to 'now' implicitly by user interaction or default state?
                            // Default state for startTime/endTime is set to 09:00/10:00 in declarations...
                            // If we want "Now" + duration:
                            const now = new Date();
                            const end = new Date(
                                now.getTime() + tmpl.default_duration * 1000,
                            );

                            // Update start/end strings
                            startDate = now.toISOString().split("T")[0];
                            startTime = now.toTimeString().slice(0, 8);
                            endDate = end.toISOString().split("T")[0];
                            endTime = end.toTimeString().slice(0, 8);
                        }

                        if (tmpl.default_emotion_id) {
                            const e = emotionStore.get(tmpl.default_emotion_id);
                            if (e) selectedEmotion = e;
                        }

                        if (tmpl.goals) {
                            goalLinks = tmpl.goals.map((g) => ({
                                goal_id: g.id,
                                impact_type: "positive",
                                impact_magnitude: 3,
                                quantity_value: tmpl.quantity_enabled
                                    ? tmpl.quantity_default
                                    : undefined,
                                quantity_unit: g.target?.unit_id,
                            }));
                        }

                        sessionStorage.removeItem("task-template");
                    } catch (e) {
                        console.error("Failed to parse task template", e);
                    }
                }
            }

            // Initialize hash for empty arrays in create mode
            lastInferredHash = getReflectionsHash([], []);
        }
        formInitialized = true;
        console.log("[TaskForm Init] Initialized:", {
            isEditing,
            lastInferredHash,
            formInitialized,
        });
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
        const currentHash = getReflectionsHash(positives, negatives);
        const hasEmotions =
            positives.some((p) => p.emotion_id) ||
            negatives.some((n) => n.emotion_id);

        console.log("[Emotion Inference] Effect triggered:", {
            hasEmotions,
            currentHash,
            lastInferredHash,
            positivesCount: positives.length,
            negativesCount: negatives.length,
            positivesWithEmotions: positives.filter((p) => p.emotion_id).length,
            negativesWithEmotions: negatives.filter((n) => n.emotion_id).length,
        });

        if (!hasEmotions) {
            console.log("[Emotion Inference] No emotions, clearing inference");
            liveInferredEmotion = null;
            inferredEmotionFull = null;
            inferringEmotion = false;
            inferenceError = null;
            return;
        }

        if (currentHash !== lastInferredHash) {
            console.log("[Emotion Inference] Starting inference...");
            inferringEmotion = true;
            inferenceError = null;

            inferEmotion({ positives, negatives })
                .then((response) => {
                    console.log("[Emotion Inference] Success:", response);
                    lastInferredHash = currentHash;
                    liveInferredEmotion = response.inferred_emotion;
                    inferringEmotion = false;

                    // Get full emotion data from store
                    if (response.inferred_emotion?.closest_emotion_id) {
                        const e = emotionStore.get(
                            response.inferred_emotion.closest_emotion_id,
                        );
                        console.log(
                            "[Emotion Inference] Found emotion in store:",
                            e?.name,
                        );
                        inferredEmotionFull = e || null;
                    } else {
                        console.log(
                            "[Emotion Inference] No closest emotion ID in response",
                        );
                        inferredEmotionFull = null;
                    }
                })
                .catch((error) => {
                    console.error("[Emotion Inference] Error:", error);
                    inferringEmotion = false;
                    inferenceError = error.message || "Failed to infer emotion";
                    liveInferredEmotion = null;
                    inferredEmotionFull = null;
                });
        } else {
            console.log(
                "[Emotion Inference] Hash unchanged, skipping inference",
            );
        }
    });

    // Reactively update selectedEmotion and inferredEmotionFull when store initializes
    $effect(() => {
        if (emotionStore.isInitialized) {
            if (task?.emotion_id && !selectedEmotion) {
                const e = emotionStore.get(task.emotion_id);
                if (e) selectedEmotion = e;
                // Note: this might override if user cleared it?
                // Actually formInitialized prevents re-running the main init block.
                // This block is specifically for when store loads AFTER task load.
                // But simply checking !selectedEmotion checks if it's empty.
                // If user actively cleared it, it is null.
                // Ideally we track if we have "attempted" to load the initial emotion.
            }
            // Update inferred emotion full if needed (e.g. store loaded late)
            if (
                liveInferredEmotion?.closest_emotion_id &&
                !inferredEmotionFull
            ) {
                const e = emotionStore.get(
                    liveInferredEmotion.closest_emotion_id,
                );
                if (e) inferredEmotionFull = e;
            }
            // Also need to handle task.inferred_emotion if not editing live
            if (
                task?.inferred_emotion?.closest_emotion_id &&
                !inferredEmotionFull &&
                !liveInferredEmotion
            ) {
                const e = emotionStore.get(
                    task.inferred_emotion.closest_emotion_id,
                );
                if (e) inferredEmotionFull = e;
            }
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
    const inferredEmotion = $derived(
        liveInferredEmotion || task?.inferred_emotion,
    );

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

    // loadTaskEmotion removed as it's handled in effects/init

    // Emotion handlers
    function handleEmotionSelect(emotion: Emotion) {
        console.log("[Emotion Select]", {
            emotion: emotion.name,
            context: emotionContext,
        });

        if (emotionContext?.type === "task") {
            selectedEmotion = emotion;
        } else if (
            emotionContext?.type === "positive" &&
            emotionContext.index !== undefined
        ) {
            if (emotionContext.index === -1) {
                // Pending new positive item
                pendingPositiveEmotion = emotion;
                console.log(
                    "[Emotion Select] Set pending positive emotion:",
                    emotion.name,
                );
            } else {
                positives = positives.map((p, i) =>
                    i === emotionContext!.index
                        ? { ...p, emotion_id: emotion.id }
                        : p,
                );
                console.log(
                    "[Emotion Select] Updated positive item at index",
                    emotionContext.index,
                );
            }
        } else if (
            emotionContext?.type === "negative" &&
            emotionContext.index !== undefined
        ) {
            if (emotionContext.index === -1) {
                // Pending new negative item
                pendingNegativeEmotion = emotion;
                console.log(
                    "[Emotion Select] Set pending negative emotion:",
                    emotion.name,
                );
            } else {
                negatives = negatives.map((n, i) =>
                    i === emotionContext!.index
                        ? { ...n, emotion_id: emotion.id }
                        : n,
                );
                console.log(
                    "[Emotion Select] Updated negative item at index",
                    emotionContext.index,
                );
            }
        }
        emotionContext = null;
        emotionModalOpen = false;
        pendingEmotionType = null;
    }

    function openEmotionForTask() {
        emotionContext = { type: "task" };
        initialEmotionForModal = null;
        allowedQuadrantsForModal = null; // All quadrants allowed for task emotion
        emotionModalOpen = true;
    }

    /** Open emotion modal with the suggested emotion pre-selected and panned to */
    function openEmotionForSuggested() {
        emotionContext = { type: "task" };
        initialEmotionForModal = inferredEmotionFull;
        allowedQuadrantsForModal = null; // All quadrants allowed for task emotion
        emotionModalOpen = true;
    }

    function openEmotionForPositive(index: number) {
        emotionContext = { type: "positive", index };

        // Find current emotion if it exists
        const p = positives[index];
        initialEmotionForModal = p.emotion_id
            ? emotionStore.get(p.emotion_id) || null
            : null;

        allowedQuadrantsForModal = ["yellow", "green"]; // Only pleasant emotions for positives
        emotionModalOpen = true;
    }

    function openEmotionForNegative(index: number) {
        emotionContext = { type: "negative", index };

        // Find current emotion if it exists
        const n = negatives[index];
        initialEmotionForModal = n.emotion_id
            ? emotionStore.get(n.emotion_id) || null
            : null;

        allowedQuadrantsForModal = ["red", "blue"]; // Only unpleasant emotions for negatives
        emotionModalOpen = true;
    }

    // Open emotion picker for pending new item
    function openEmotionForPendingPositive() {
        pendingEmotionType = "positive";
        emotionContext = { type: "positive", index: -1 };
        initialEmotionForModal = pendingPositiveEmotion;
        allowedQuadrantsForModal = ["yellow", "green"]; // Only pleasant emotions for positives
        emotionModalOpen = true;
    }

    function openEmotionForPendingNegative() {
        pendingEmotionType = "negative";
        emotionContext = { type: "negative", index: -1 };
        initialEmotionForModal = pendingNegativeEmotion;
        allowedQuadrantsForModal = ["red", "blue"]; // Only unpleasant emotions for negatives
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
            const newItem = {
                text: newPositive.trim(),
                emotion_id: pendingPositiveEmotion?.id,
            };
            positives = [...positives, newItem];
            console.log("[Add Positive]", {
                newItem,
                totalPositives: positives.length,
                hasEmotion: !!newItem.emotion_id,
            });
            newPositive = "";
            pendingPositiveEmotion = null;
        }
    }

    function addNegative() {
        if (newNegative.trim()) {
            const newItem = {
                text: newNegative.trim(),
                emotion_id: pendingNegativeEmotion?.id,
            };
            negatives = [...negatives, newItem];
            console.log("[Add Negative]", {
                newItem,
                totalNegatives: negatives.length,
                hasEmotion: !!newItem.emotion_id,
            });
            newNegative = "";
            pendingNegativeEmotion = null;
        }
    }

    function clearPendingPositiveEmotion() {
        pendingPositiveEmotion = null;
    }

    function clearPendingNegativeEmotion() {
        pendingNegativeEmotion = null;
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
            linked_goals:
                goalLinks.length > 0
                    ? goalLinks.map((l) => ({
                          goal_id: l.goal_id,
                          impact_type: l.impact_type,
                          impact_magnitude: l.impact_magnitude,
                          quantity_value: l.quantity_value,
                          quantity_unit: l.quantity_unit,
                      }))
                    : undefined,
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

                <!-- Reflection Section - Redesigned -->
                <Card
                    variant="bordered"
                    class="transition-all duration-300 hover:shadow-lg"
                >
                    <div class="flex items-center gap-2 mb-4">
                        <Sparkles class="w-4 h-4 opacity-50" />
                        <span class="text-xs font-semibold uppercase opacity-50"
                            >Reflection</span
                        >
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <!-- Positives Section - Redesigned -->
                        <div class="flex flex-col h-full">
                            <div class="flex items-center gap-2 mb-3">
                                <div
                                    class="w-6 h-6 rounded-lg bg-success/20 flex items-center justify-center"
                                >
                                    <ThumbsUp
                                        class="w-3.5 h-3.5 text-success"
                                    />
                                </div>
                                <span class="text-sm font-semibold text-success"
                                    >What went well?</span
                                >
                            </div>

                            <!-- New item input with emotion selector -->
                            <div
                                class="rounded-xl border border-success/30 bg-success/5 p-3 space-y-2"
                            >
                                <!-- Emotion selector (optional, select BEFORE adding) -->
                                <div class="flex items-center gap-2">
                                    {#if pendingPositiveEmotion}
                                        {@const colors =
                                            QUADRANT_COLORS[
                                                pendingPositiveEmotion.quadrant as Quadrant
                                            ]}
                                        <div
                                            class="tooltip tooltip-bottom"
                                            data-tip={pendingPositiveEmotion.description}
                                        >
                                            <button
                                                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 hover:scale-105"
                                                style="
                                                background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
                                                color: {colors.secondary};
                                                border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                            "
                                                onclick={openEmotionForPendingPositive}
                                            >
                                                <span
                                                    class="w-6 h-6 rounded-md flex items-center justify-center"
                                                    style="background: {colors.gradient};"
                                                >
                                                    <OpenMoji
                                                        emoji={pendingPositiveEmotion.emoji}
                                                        alt={pendingPositiveEmotion.name}
                                                        size="sm"
                                                    />
                                                </span>
                                                <span
                                                    class="truncate max-w-[80px]"
                                                    >{pendingPositiveEmotion.name}</span
                                                >
                                                <!-- svelte-ignore a11y_click_events_have_key_events -->
                                                <!-- svelte-ignore a11y_no_static_element_interactions -->
                                                <span
                                                    class="p-0.5 rounded-full hover:bg-base-content/10 transition-colors cursor-pointer"
                                                    onclick={(e) => {
                                                        e.stopPropagation();
                                                        clearPendingPositiveEmotion();
                                                    }}
                                                    role="button"
                                                    tabindex="0"
                                                >
                                                    <X class="w-3 h-3" />
                                                </span>
                                            </button>
                                        </div>
                                    {:else}
                                        <button
                                            class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-medium bg-success/10 text-success/70 hover:bg-success/20 hover:text-success transition-all duration-200"
                                            onclick={openEmotionForPendingPositive}
                                        >
                                            <Heart class="w-3 h-3" />
                                            <span>Add feeling</span>
                                        </button>
                                    {/if}
                                </div>

                                <!-- Text input -->
                                <div class="flex items-start gap-2">
                                    <textarea
                                        bind:value={newPositive}
                                        placeholder="What went well? What are you grateful for?"
                                        class="textarea textarea-sm textarea-bordered flex-1 bg-transparent border-success/20 focus:border-success min-h-[60px] text-sm resize-none"
                                        onkeydown={(e) => {
                                            if (
                                                e.key === "Enter" &&
                                                !e.shiftKey
                                            ) {
                                                e.preventDefault();
                                                addPositive();
                                            }
                                        }}
                                    ></textarea>
                                    <button
                                        class="btn btn-sm btn-success btn-outline shrink-0 gap-1"
                                        onclick={addPositive}
                                        disabled={!newPositive.trim()}
                                    >
                                        <Plus class="w-4 h-4" />
                                    </button>
                                </div>
                            </div>

                            <!-- Existing positives list -->
                            {#if positives.length > 0}
                                <div class="flex flex-col gap-2 mt-3">
                                    {#each positives as p, i (i)}
                                        {@const hasEmotion = !!p.emotion_id}
                                        <div
                                            class="rounded-xl bg-gradient-to-r from-success/10 to-success/5 border border-success/20 p-3 group hover:shadow-md transition-all duration-300 animate-fade-in"
                                        >
                                            <div class="flex items-start gap-3">
                                                <ThumbsUp
                                                    class="w-4 h-4 text-success shrink-0 mt-0.5"
                                                />
                                                <div class="flex-1 min-w-0">
                                                    <p
                                                        class="text-sm leading-relaxed break-words"
                                                    >
                                                        {p.text}
                                                    </p>
                                                    <!-- Emotion display -->
                                                    <div class="mt-2">
                                                        {#if hasEmotion}
                                                            {@const emotion =
                                                                emotionStore.get(
                                                                    p.emotion_id!,
                                                                )}
                                                            {#if emotion}
                                                                {@const colors =
                                                                    QUADRANT_COLORS[
                                                                        emotion.quadrant as Quadrant
                                                                    ]}
                                                                <div
                                                                    class="tooltip tooltip-bottom"
                                                                    data-tip={emotion.description}
                                                                >
                                                                    <button
                                                                        class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 hover:scale-105"
                                                                        style="
                                                                        background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
                                                                        color: {colors.secondary};
                                                                        border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                                                    "
                                                                        onclick={() =>
                                                                            openEmotionForPositive(
                                                                                i,
                                                                            )}
                                                                    >
                                                                        <span
                                                                            class="w-6 h-6 rounded-md flex items-center justify-center"
                                                                            style="background: {colors.gradient};"
                                                                        >
                                                                            <OpenMoji
                                                                                emoji={emotion.emoji}
                                                                                alt={emotion.name}
                                                                                size="sm"
                                                                            />
                                                                        </span>
                                                                        <span
                                                                            class="truncate max-w-[80px]"
                                                                            >{emotion.name}</span
                                                                        >
                                                                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                                                                        <!-- svelte-ignore a11y_no_static_element_interactions -->
                                                                        <span
                                                                            class="p-0.5 rounded-full hover:bg-base-content/10 transition-colors cursor-pointer"
                                                                            onclick={(
                                                                                e,
                                                                            ) => {
                                                                                e.stopPropagation();
                                                                                clearPositiveEmotion(
                                                                                    i,
                                                                                );
                                                                            }}
                                                                            role="button"
                                                                            tabindex="0"
                                                                        >
                                                                            <X
                                                                                class="w-3 h-3"
                                                                            />
                                                                        </span>
                                                                    </button>
                                                                </div>
                                                            {:else}
                                                                <span
                                                                    class="text-xs text-success/50"
                                                                    >Loading...</span
                                                                >
                                                            {/if}
                                                        {:else}
                                                            <button
                                                                class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] text-base-content/40 hover:text-success hover:bg-success/10 transition-all duration-200"
                                                                onclick={() =>
                                                                    openEmotionForPositive(
                                                                        i,
                                                                    )}
                                                            >
                                                                <Plus
                                                                    class="w-3 h-3"
                                                                />
                                                                <span
                                                                    >Add feeling</span
                                                                >
                                                            </button>
                                                        {/if}
                                                    </div>
                                                </div>
                                                <button
                                                    onclick={() =>
                                                        removePositive(i)}
                                                    class="btn btn-ghost btn-xs text-base-content/30 hover:text-error hover:bg-error/10 transition-all duration-200 shrink-0 opacity-0 group-hover:opacity-100"
                                                >
                                                    <X class="w-3.5 h-3.5" />
                                                </button>
                                            </div>
                                        </div>
                                    {/each}
                                </div>
                            {/if}
                        </div>

                        <!-- Negatives Section - Redesigned -->
                        <div class="flex flex-col h-full">
                            <div class="flex items-center gap-2 mb-3">
                                <div
                                    class="w-6 h-6 rounded-lg bg-error/20 flex items-center justify-center"
                                >
                                    <ThumbsDown
                                        class="w-3.5 h-3.5 text-error"
                                    />
                                </div>
                                <span class="text-sm font-semibold text-error"
                                    >What could improve?</span
                                >
                            </div>

                            <!-- New item input with emotion selector -->
                            <div
                                class="rounded-xl border border-error/30 bg-error/5 p-3 space-y-2"
                            >
                                <!-- Emotion selector (optional, select BEFORE adding) -->
                                <div class="flex items-center gap-2">
                                    {#if pendingNegativeEmotion}
                                        {@const colors =
                                            QUADRANT_COLORS[
                                                pendingNegativeEmotion.quadrant as Quadrant
                                            ]}
                                        <div
                                            class="tooltip tooltip-bottom"
                                            data-tip={pendingNegativeEmotion.description}
                                        >
                                            <button
                                                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 hover:scale-105"
                                                style="
                                                background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
                                                color: {colors.secondary};
                                                border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                            "
                                                onclick={openEmotionForPendingNegative}
                                            >
                                                <span
                                                    class="w-6 h-6 rounded-md flex items-center justify-center"
                                                    style="background: {colors.gradient};"
                                                >
                                                    <OpenMoji
                                                        emoji={pendingNegativeEmotion.emoji}
                                                        alt={pendingNegativeEmotion.name}
                                                        size="sm"
                                                    />
                                                </span>
                                                <span
                                                    class="truncate max-w-[80px]"
                                                    >{pendingNegativeEmotion.name}</span
                                                >
                                                <!-- svelte-ignore a11y_click_events_have_key_events -->
                                                <!-- svelte-ignore a11y_no_static_element_interactions -->
                                                <span
                                                    class="p-0.5 rounded-full hover:bg-base-content/10 transition-colors cursor-pointer"
                                                    onclick={(e) => {
                                                        e.stopPropagation();
                                                        clearPendingNegativeEmotion();
                                                    }}
                                                    role="button"
                                                    tabindex="0"
                                                >
                                                    <X class="w-3 h-3" />
                                                </span>
                                            </button>
                                        </div>
                                    {:else}
                                        <button
                                            class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-medium bg-error/10 text-error/70 hover:bg-error/20 hover:text-error transition-all duration-200"
                                            onclick={openEmotionForPendingNegative}
                                        >
                                            <Heart class="w-3 h-3" />
                                            <span>Add feeling</span>
                                        </button>
                                    {/if}
                                </div>

                                <!-- Text input -->
                                <div class="flex items-start gap-2">
                                    <textarea
                                        bind:value={newNegative}
                                        placeholder="What could be better? What challenged you?"
                                        class="textarea textarea-sm textarea-bordered flex-1 bg-transparent border-error/20 focus:border-error min-h-[60px] text-sm resize-none"
                                        onkeydown={(e) => {
                                            if (
                                                e.key === "Enter" &&
                                                !e.shiftKey
                                            ) {
                                                e.preventDefault();
                                                addNegative();
                                            }
                                        }}
                                    ></textarea>
                                    <button
                                        class="btn btn-sm btn-error btn-outline shrink-0 gap-1"
                                        onclick={addNegative}
                                        disabled={!newNegative.trim()}
                                    >
                                        <Plus class="w-4 h-4" />
                                    </button>
                                </div>
                            </div>

                            <!-- Existing negatives list -->
                            {#if negatives.length > 0}
                                <div class="flex flex-col gap-2 mt-3">
                                    {#each negatives as n, i (i)}
                                        {@const hasEmotion = !!n.emotion_id}
                                        <div
                                            class="rounded-xl bg-gradient-to-r from-error/10 to-error/5 border border-error/20 p-3 group hover:shadow-md transition-all duration-300 animate-fade-in"
                                        >
                                            <div class="flex items-start gap-3">
                                                <ThumbsDown
                                                    class="w-4 h-4 text-error shrink-0 mt-0.5"
                                                />
                                                <div class="flex-1 min-w-0">
                                                    <p
                                                        class="text-sm leading-relaxed break-words"
                                                    >
                                                        {n.text}
                                                    </p>
                                                    <!-- Emotion display -->
                                                    <div class="mt-2">
                                                        {#if hasEmotion}
                                                            {@const emotion =
                                                                emotionStore.get(
                                                                    n.emotion_id!,
                                                                )}
                                                            {#if emotion}
                                                                {@const colors =
                                                                    QUADRANT_COLORS[
                                                                        emotion.quadrant as Quadrant
                                                                    ]}
                                                                <div
                                                                    class="tooltip tooltip-bottom"
                                                                    data-tip={emotion.description}
                                                                >
                                                                    <button
                                                                        class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 hover:scale-105"
                                                                        style="
                                                                        background: linear-gradient(135deg, color-mix(in srgb, {colors.primary} 15%, transparent), color-mix(in srgb, {colors.secondary} 10%, transparent));
                                                                        color: {colors.secondary};
                                                                        border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent);
                                                                    "
                                                                        onclick={() =>
                                                                            openEmotionForNegative(
                                                                                i,
                                                                            )}
                                                                    >
                                                                        <span
                                                                            class="w-6 h-6 rounded-md flex items-center justify-center"
                                                                            style="background: {colors.gradient};"
                                                                        >
                                                                            <OpenMoji
                                                                                emoji={emotion.emoji}
                                                                                alt={emotion.name}
                                                                                size="sm"
                                                                            />
                                                                        </span>
                                                                        <span
                                                                            class="truncate max-w-[80px]"
                                                                            >{emotion.name}</span
                                                                        >
                                                                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                                                                        <!-- svelte-ignore a11y_no_static_element_interactions -->
                                                                        <span
                                                                            class="p-0.5 rounded-full hover:bg-base-content/10 transition-colors cursor-pointer"
                                                                            onclick={(
                                                                                e,
                                                                            ) => {
                                                                                e.stopPropagation();
                                                                                clearNegativeEmotion(
                                                                                    i,
                                                                                );
                                                                            }}
                                                                            role="button"
                                                                            tabindex="0"
                                                                        >
                                                                            <X
                                                                                class="w-3 h-3"
                                                                            />
                                                                        </span>
                                                                    </button>
                                                                </div>
                                                            {:else}
                                                                <span
                                                                    class="text-xs text-error/50"
                                                                    >Loading...</span
                                                                >
                                                            {/if}
                                                        {:else}
                                                            <button
                                                                class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] text-base-content/40 hover:text-error hover:bg-error/10 transition-all duration-200"
                                                                onclick={() =>
                                                                    openEmotionForNegative(
                                                                        i,
                                                                    )}
                                                            >
                                                                <Plus
                                                                    class="w-3 h-3"
                                                                />
                                                                <span
                                                                    >Add feeling</span
                                                                >
                                                            </button>
                                                        {/if}
                                                    </div>
                                                </div>
                                                <button
                                                    onclick={() =>
                                                        removeNegative(i)}
                                                    class="btn btn-ghost btn-xs text-base-content/30 hover:text-error hover:bg-error/10 transition-all duration-200 shrink-0 opacity-0 group-hover:opacity-100"
                                                >
                                                    <X class="w-3.5 h-3.5" />
                                                </button>
                                            </div>
                                        </div>
                                    {/each}
                                </div>
                            {/if}
                        </div>
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

                <!-- Emotion Section - Redesigned -->
                <Card
                    variant="bordered"
                    class="transition-all duration-300 hover:shadow-lg overflow-visible"
                >
                    <div class="flex items-center gap-2 mb-4">
                        <Heart class="w-4 h-4 opacity-50" />
                        <span class="text-xs font-semibold uppercase opacity-50"
                            >Feeling</span
                        >
                    </div>

                    <!-- Two-column display: Selected | Inferred -->
                    <div class="grid grid-cols-1 gap-3">
                        <!-- User Selected Emotion -->
                        <div class="relative">
                            <div
                                class="text-[10px] uppercase font-semibold text-base-content/40 mb-2 flex items-center gap-1.5"
                            >
                                <span>Your Selection</span>
                            </div>

                            {#if selectedEmotion}
                                {@const colors =
                                    QUADRANT_COLORS[
                                        selectedEmotion.quadrant as Quadrant
                                    ]}
                                {@const meta =
                                    QUADRANT_META[
                                        selectedEmotion.quadrant as Quadrant
                                    ]}

                                <div
                                    class="tooltip tooltip-bottom w-full"
                                    data-tip={selectedEmotion.description}
                                >
                                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                                    <div
                                        class="w-full rounded-xl p-3 flex items-center gap-3 transition-all duration-300 hover:shadow-lg cursor-pointer group/card relative overflow-hidden border"
                                        style="
                                                background: linear-gradient(135deg, 
                                                    color-mix(in srgb, {colors.primary} 12%, var(--b1)) 0%, 
                                                    color-mix(in srgb, {colors.secondary} 8%, var(--b1)) 100%);
                                                border-color: color-mix(in srgb, {colors.primary} 25%, transparent);
                                            "
                                        onclick={openEmotionForTask}
                                        onkeydown={(e) =>
                                            e.key === "Enter" &&
                                            openEmotionForTask()}
                                        role="button"
                                        tabindex="0"
                                    >
                                        <!-- Background glow on hover -->
                                        <div
                                            class="absolute inset-0 opacity-0 group-hover/card:opacity-100 transition-opacity duration-500 pointer-events-none"
                                            style="background: radial-gradient(circle at 30% 50%, {colors.glow}, transparent 60%);"
                                        ></div>

                                        <!-- Emoji -->
                                        <div
                                            class="w-11 h-11 rounded-xl flex items-center justify-center shrink-0 shadow-md group-hover/card:scale-110 transition-transform duration-300 relative z-10"
                                            style="background: {colors.gradient};"
                                        >
                                            <OpenMoji
                                                emoji={selectedEmotion.emoji}
                                                alt={selectedEmotion.name}
                                                size="md"
                                            />
                                        </div>

                                        <!-- Info -->
                                        <div
                                            class="flex-1 min-w-0 text-left relative z-10"
                                        >
                                            <div
                                                class="flex items-center gap-2"
                                            >
                                                <span
                                                    class="text-sm font-bold truncate"
                                                    style="color: {colors.secondary};"
                                                >
                                                    {selectedEmotion.name}
                                                </span>
                                                <span
                                                    class="badge badge-xs font-bold uppercase tracking-wider px-1.5 shrink-0"
                                                    style="background: {colors.primary}; color: white; border: none;"
                                                >
                                                    ✓
                                                </span>
                                            </div>
                                            <div
                                                class="text-[10px] opacity-60 mt-0.5"
                                            >
                                                {meta?.energyLabel} Energy • {meta?.pleasantnessLabel}
                                            </div>
                                        </div>

                                        <!-- Clear button -->
                                        <button
                                            class="p-1.5 rounded-full hover:bg-base-content/10 transition-colors relative z-10 opacity-0 group-hover/card:opacity-100"
                                            onclick={(e) => {
                                                e.stopPropagation();
                                                clearTaskEmotion();
                                            }}
                                            title="Remove"
                                        >
                                            <X
                                                class="w-4 h-4 opacity-60 hover:opacity-100"
                                            />
                                        </button>
                                    </div>
                                </div>
                            {:else}
                                <!-- Empty State -->
                                <button
                                    class="w-full rounded-xl border-2 border-dashed border-base-300 p-4 flex items-center justify-center gap-3 hover:border-primary hover:bg-primary/5 transition-all duration-300 text-base-content/50 hover:text-primary group/empty"
                                    onclick={openEmotionForTask}
                                >
                                    <div
                                        class="w-10 h-10 rounded-xl bg-base-200 flex items-center justify-center group-hover/empty:bg-primary/20 transition-colors duration-300"
                                    >
                                        <Heart
                                            class="w-5 h-5 group-hover/empty:scale-110 transition-transform duration-300"
                                        />
                                    </div>
                                    <div class="text-left">
                                        <span class="text-sm font-semibold"
                                            >Select Emotion</span
                                        >
                                        <p class="text-xs opacity-60">
                                            How are you feeling?
                                        </p>
                                    </div>
                                    <Plus
                                        class="w-5 h-5 ml-auto opacity-40 group-hover/empty:opacity-100 transition-opacity"
                                    />
                                </button>
                            {/if}
                        </div>

                        <!-- Inferred Emotion (side by side feel) -->
                        {#if inferringEmotion || inferredEmotion || inferenceError}
                            <div class="relative">
                                <div
                                    class="text-[10px] uppercase font-semibold text-primary/70 mb-2 flex items-center gap-1.5"
                                >
                                    <Sparkles
                                        class="w-3 h-3 text-primary {inferringEmotion
                                            ? 'animate-pulse'
                                            : ''}"
                                    />
                                    <span>Suggested</span>
                                    <div
                                        class="tooltip tooltip-right cursor-help normal-case"
                                        data-tip="Based on your reflections"
                                    >
                                        <Info class="w-3 h-3 text-primary/50" />
                                    </div>
                                    {#if inferringEmotion}
                                        <span
                                            class="loading loading-spinner loading-xs text-primary"
                                        ></span>
                                    {/if}
                                </div>

                                {#if inferenceError}
                                    <!-- Error State -->
                                    <div
                                        class="w-full rounded-xl p-2.5 flex items-center gap-2 bg-error/10 border border-error/30"
                                    >
                                        <CircleAlert
                                            class="w-4 h-4 text-error shrink-0"
                                        />
                                        <span class="text-xs text-error"
                                            >{inferenceError}</span
                                        >
                                    </div>
                                {:else if inferringEmotion}
                                    <!-- Loading State -->
                                    <div
                                        class="w-full rounded-xl p-2.5 flex items-center gap-3 bg-base-200 border border-base-300 animate-pulse"
                                    >
                                        <div
                                            class="w-9 h-9 rounded-lg bg-base-300"
                                        ></div>
                                        <div class="flex-1">
                                            <div
                                                class="h-3 bg-base-300 rounded w-24 mb-1"
                                            ></div>
                                            <div
                                                class="h-2 bg-base-300 rounded w-32"
                                            ></div>
                                        </div>
                                    </div>
                                {:else if inferredEmotion}
                                    {@const inferredQuadrant =
                                        (inferredEmotionFull?.quadrant ||
                                            inferredEmotion.quadrant ||
                                            "green") as Quadrant}
                                    {@const colors =
                                        QUADRANT_COLORS[inferredQuadrant]}
                                    {@const meta =
                                        QUADRANT_META[inferredQuadrant]}
                                    <!-- Emotion Display -->

                                    <div
                                        class="tooltip tooltip-bottom w-full"
                                        data-tip={inferredEmotionFull?.description ||
                                            "Calculated from your reflections"}
                                    >
                                        <button
                                            class="w-full rounded-xl p-2.5 flex items-center gap-3 transition-all duration-300 hover:shadow-md cursor-pointer group/inferred relative overflow-hidden"
                                            style="
                                                background: linear-gradient(135deg,
                                                    color-mix(in srgb, {colors.primary} 6%, var(--b2)) 0%,
                                                    color-mix(in srgb, {colors.secondary} 4%, var(--b2)) 100%);
                                                border: 1px dashed color-mix(in srgb, {colors.primary} 35%, transparent);
                                            "
                                            onclick={openEmotionForSuggested}
                                        >
                                            <!-- Emoji -->
                                            <div
                                                class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 shadow-sm group-hover/inferred:scale-105 transition-transform duration-300"
                                                style="background: {colors.gradient}; opacity: 0.9;"
                                            >
                                                {#if inferredEmotionFull}
                                                    <OpenMoji
                                                        emoji={inferredEmotionFull.emoji}
                                                        alt={inferredEmotionFull?.name ||
                                                            "Inferred Emotion"}
                                                        size="md"
                                                    />
                                                {:else}
                                                    <Sparkles
                                                        class="w-4 h-4 text-white/80"
                                                    />
                                                {/if}
                                            </div>

                                            <!-- Info -->
                                            <div
                                                class="flex-1 min-w-0 text-left"
                                            >
                                                <div
                                                    class="flex items-center gap-2"
                                                >
                                                    <span
                                                        class="text-sm font-semibold opacity-80"
                                                    >
                                                        {inferredEmotionFull?.name ||
                                                            "Suggesting..."}
                                                    </span>
                                                </div>
                                                <div
                                                    class="text-[10px] opacity-50 mt-0.5"
                                                >
                                                    {meta?.energyLabel} Energy •
                                                    {meta?.pleasantnessLabel}
                                                </div>
                                            </div>

                                            <ChevronRight
                                                class="w-4 h-4 text-primary/40 group-hover/inferred:text-primary transition-colors"
                                            />
                                        </button>
                                    </div>
                                {/if}
                            </div>
                        {/if}
                    </div>
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

                <!-- Goals Section -->
                <Card
                    variant="bordered"
                    class="transition-all duration-200 hover:shadow-md"
                >
                    <GoalSelector
                        value={goalLinks}
                        onChange={(links) => (goalLinks = links)}
                    />
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
                        class="range range-sm range-primary w-full transition-all duration-200"
                        step="1"
                    />
                    <div
                        class="relative h-5 text-[10px] mt-1.5 opacity-50 mx-0.5"
                    >
                        {#each priorityLabels as label, i}
                            <span
                                class="absolute top-0 whitespace-nowrap"
                                style="
                                    left: {(i / (priorityLabels.length - 1)) *
                                    100}%;
                                    transform: translateX({i === 0
                                    ? '0%'
                                    : i === priorityLabels.length - 1
                                      ? '-100%'
                                      : '-50%'});
                                "
                            >
                                {label}
                            </span>
                        {/each}
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
    initialEmotion={initialEmotionForModal}
    allowedQuadrants={allowedQuadrantsForModal}
    onSelect={handleEmotionSelect}
    onClose={() => {
        emotionModalOpen = false;
        emotionContext = null;
        initialEmotionForModal = null;
        allowedQuadrantsForModal = null;
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
