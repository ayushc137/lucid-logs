<script lang="ts">
import { DEFAULT_CATEGORY_COLOR } from '$lib/utils';
    import { browser } from "$app/environment";
    import {
        type CreateGoalRequest,
        type Goal,
        createCategory,
        createGoal,
        createUnit,
        getCategories,
        getGoalLogs,
        getGoalTasks,
        getUnits,
        updateGoal,
    } from "$lib/api";
    import {
        GoalHistoryTab,
        GoalPrioritySlider,
        GoalRecurrenceSettings,
        GoalTargetSettings,
        GoalTasksTab,
        GoalProgressChart,
        GoalStreakHistory,
    } from "$lib/components/goals";
    import { CategoryDropdown, ColorPicker } from "$lib/components/ui";
    import { cn } from "$lib/utils";
    import {
        createMutation,
        createQuery,
        useQueryClient,
    } from "@tanstack/svelte-query";
    import { onMount } from "svelte";
    import { writable } from "svelte/store";

    import {
        BarChart3,
        Calendar,
        Check,
        CheckCircle2,
        Circle,
        Clock,
        Flame,
        History,
        ListTodo,
        Play,
        Plus,
        Repeat,
        Save,
        Tag,
        Target,
        X,
    } from "lucide-svelte";

    interface Props {
        open?: boolean;
        goal?: Goal | null;
        onClose?: () => void;
    }

    let { open = $bindable(false), goal = null, onClose }: Props = $props();

    const queryClient = useQueryClient();

    // Fetch categories
    const categoriesQuery = createQuery({
        queryKey: ["categories"],
        queryFn: () => getCategories({ limit: 100 }),
    });

    const categories = $derived($categoriesQuery.data?.items || []);

    // Form state
    let title = $state("");
    let description = $state("");
    let icon = $state("🎯");
    let categoryId = $state<string | undefined>(undefined);
    let priority = $state(2);
    let startDate = $state("");
    let deadline = $state("");
    let status = $state("active");

    // Habit settings
    let isHabit = $state(false);
    let frequency = $state(1);
    let period = $state<"day" | "week" | "month">("day");
    let activeDays = $state<number[]>([]);
    let activeMonthDay = $state<number>(1); // Day of the month for monthly habits

    // Target settings
    let isMeasurable = $state(false);
    let targetOperator = $state<"gte" | "lte" | "eq">("gte");
    let targetValue = $state(10);
    let targetUnit = $state("");

    // Tab state
    let activeTab = $state<"details" | "tasks" | "history" | "analytics">(
        "details",
    );

    // Emoji picker state
    let showEmojiPicker = $state(false);

    const isEditing = $derived(!!goal);
    const goalId = $derived(goal?.id);

    // Fetch goal logs when editing
    // Fetch goal logs when editing
    const logsOptions = writable({
        queryKey: ["goal-logs", null as string | null | undefined],
        queryFn: () =>
            Promise.resolve({ goal_id: "", logs: [] as any[], total: 0 }),
        enabled: false,
    });

    $effect(() => {
        logsOptions.set({
            queryKey: ["goal-logs", goalId],
            queryFn: () =>
                goalId
                    ? getGoalLogs(goalId, { limit: 50 })
                    : Promise.resolve({ goal_id: "", logs: [], total: 0 }),
            enabled: !!goalId && open,
        });
    });

    const logsQuery = createQuery(logsOptions);

    const logs = $derived($logsQuery.data?.logs || []);

    // Fetch goal tasks when editing
    const tasksOptions = writable({
        queryKey: ["goal-tasks", null as string | null | undefined],
        queryFn: () => Promise.resolve({ goal_id: "", tasks: [] as any[] }),
        enabled: false,
    });

    $effect(() => {
        tasksOptions.set({
            queryKey: ["goal-tasks", goalId],
            queryFn: () =>
                goalId
                    ? getGoalTasks(goalId)
                    : Promise.resolve({ goal_id: "", tasks: [] }),
            enabled: !!goalId && open,
        });
    });

    const tasksQuery = createQuery(tasksOptions);

    // Fetch units for summary
    const unitsQuery = createQuery({
        queryKey: ["units"],
        queryFn: () => getUnits(),
        staleTime: 1000 * 60 * 5, // Cache for 5 minutes
    });
    const units = $derived($unitsQuery.data?.items || []);
    const selectedUnit = $derived(units.find((u) => u.id === targetUnit));

    // Category creation state
    let showCreateCategory = $state(false);
    let newCategoryName = $state("");
    let newCategoryColor = $state(DEFAULT_CATEGORY_COLOR);
    let useCustomCategoryColor = $state(false);

    // Unit creation state
    let showCreateUnit = $state(false);

    // Emoji picker
    let emojiPickerRef = $state<HTMLDivElement | null>(null);
    let emojiPickerReady = $state(false);
    let showFullPicker = $state(false);

    // Suggested emojis organized by goal categories
    const suggestedEmojis = {
        "Fitness & Health": [
            "💪",
            "🏃",
            "🏋️",
            "🧘",
            "🚴",
            "🏊",
            "🥗",
            "💊",
            "😴",
            "🧠",
        ],
        "Learning & Growth": [
            "📚",
            "✍️",
            "🎓",
            "💡",
            "🔬",
            "📝",
            "🎯",
            "📖",
            "🧪",
            "💭",
        ],
        "Finance & Career": [
            "💰",
            "📈",
            "💼",
            "🏆",
            "⭐",
            "🚀",
            "💎",
            "📊",
            "🎖️",
            "👔",
        ],
        "Lifestyle & Habits": [
            "🌅",
            "🧹",
            "☕",
            "🌱",
            "⏰",
            "📱",
            "🎨",
            "🎵",
            "🏠",
            "✨",
        ],
        Relationships: [
            "❤️",
            "👨‍👩‍👧",
            "🤝",
            "💬",
            "📞",
            "🎁",
            "😊",
            "🙏",
            "💝",
            "👥",
        ],
    };

    // Flatten for quick access row
    const quickEmojis = [
        "🎯",
        "💪",
        "📚",
        "💰",
        "❤️",
        "🚀",
        "⭐",
        "🏆",
        "🔥",
        "✅",
        "🌟",
        "💡",
    ];

    // Initialize emoji-picker-element
    onMount(() => {
        if (browser) {
            import("emoji-picker-element").then(() => {
                emojiPickerReady = true;
            });
        }
    });

    // Handle emoji selection from picker
    function handleEmojiSelect(event: CustomEvent) {
        if (event.detail?.unicode) {
            icon = event.detail.unicode;
            showEmojiPicker = false;
            showFullPicker = false;
        }
    }

    // Handle quick emoji selection
    function selectQuickEmoji(emoji: string) {
        icon = emoji;
        showEmojiPicker = false;
        showFullPicker = false;
    }

    // Days
    const dayOptions = [
        { value: "mon", label: "M", full: "Monday" },
        { value: "tue", label: "T", full: "Tuesday" },
        { value: "wed", label: "W", full: "Wednesday" },
        { value: "thu", label: "T", full: "Thursday" },
        { value: "fri", label: "F", full: "Friday" },
        { value: "sat", label: "S", full: "Saturday" },
        { value: "sun", label: "S", full: "Sunday" },
    ];

    const createMut = createMutation({
        mutationFn: (data: CreateGoalRequest) => createGoal(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["goals"] });
            queryClient.invalidateQueries({ queryKey: ["goals-list"] });
            resetForm();
            onClose?.();
        },
    });

    const updateMut = createMutation({
        mutationFn: ({
            id,
            data,
        }: {
            id: string;
            data: Partial<CreateGoalRequest>;
        }) => updateGoal(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["goals"] });
            queryClient.invalidateQueries({ queryKey: ["goals-list"] });
            queryClient.invalidateQueries({ queryKey: ["goal-logs"] });
            resetForm();
            onClose?.();
        },
    });

    const isPending = $derived($createMut.isPending || $updateMut.isPending);

    // Load goal data when editing
    $effect(() => {
        if (open && goal) {
            title = goal.title;
            description = goal.description || "";
            icon = goal.icon || "🎯";
            categoryId =
                goal.category?.id && goal.category.id !== ":<nil>"
                    ? goal.category.id
                    : undefined;
            priority = goal.priority || 5;
            status = goal.status || "active";

            if (goal.start_date) {
                startDate = new Date(goal.start_date)
                    .toISOString()
                    .split("T")[0];
            }
            if (goal.deadline) {
                deadline = new Date(goal.deadline).toISOString().split("T")[0];
            }

            if (goal.recurrence) {
                isHabit = true;
                frequency = goal.recurrence.frequency || 1;
                period =
                    (goal.recurrence.period as "day" | "week" | "month") ||
                    "day";
                activeDays = (goal.recurrence.active_days || []).map((d) =>
                    typeof d === "string" ? Number.parseInt(d, 10) : d,
                );
            } else {
                isHabit = false;
            }

            if (goal.target && goal.target.value > 0) {
                isMeasurable = true;
                targetOperator =
                    (goal.target.operator as "gte" | "lte" | "eq") || "gte";
                targetValue = goal.target.value || 10;
                targetUnit = goal.target.unit_id || "";
            } else {
                isMeasurable = false;
            }
        } else if (open) {
            resetForm();
        }
    });

    function resetForm() {
        title = "";
        description = "";
        icon = "🎯";
        categoryId = undefined;
        priority = 5;
        startDate = "";
        deadline = "";
        status = "active";
        isHabit = false;
        frequency = 1;
        period = "day";
        activeDays = [];
        isMeasurable = false;
        targetOperator = "gte";
        targetValue = 10;
        targetUnit = "";
        activeTab = "details";
        showEmojiPicker = false;
    }

    function handleSubmit() {
        if (!title.trim()) return;

        const data: CreateGoalRequest = {
            title: title.trim(),
            description: description.trim() || undefined,
            icon,
            priority,
            category_id: categoryId,
            start_date: startDate || undefined,
            deadline: deadline || undefined,
        };

        if (isHabit) {
            data.recurrence = {
                frequency,
                period,
                active_days:
                    period === "week" && activeDays.length > 0
                        ? activeDays.map((d) => String(d))
                        : undefined,
            };
        }

        if (isMeasurable) {
            data.target = {
                value: targetValue,
                operator: targetOperator,
                unit_id: targetUnit || "count",
            };
        }

        if (isEditing && goal) {
            $updateMut.mutate({
                id: goal.id,
                data: { ...data, status } as any,
            });
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

    function toggleDay(day: number) {
        if (activeDays.includes(day)) {
            activeDays = activeDays.filter((d) => d !== day);
        } else {
            activeDays = [...activeDays, day];
        }
    }

    // Computed values for display
    const currentStreak = $derived(goal?.stats?.current_streak || 0);
    const longestStreak = $derived(goal?.stats?.longest_streak || 0);
    const progressPercent = $derived(goal?.stats?.progress_percent || 0);
    const currentValue = $derived(goal?.stats?.current_value || 0);
    const totalContributions = $derived(goal?.stats?.total_contributions || 0);

    // Linked tasks (fetched from separate API)
    const linkedTasks = $derived($tasksQuery.data?.tasks || []);

    function getPeriodLabel(p: string): string {
        const map: Record<string, string> = {
            day: "Daily",
            week: "Weekly",
            month: "Monthly",
        };
        return map[p] || p;
    }

    // Create category mutation
    const createCategoryMut = createMutation({
        mutationFn: (data: { name: string; color: string }) =>
            createCategory({ name: data.name, color: data.color }),
        onSuccess: (newCat) => {
            queryClient.invalidateQueries({ queryKey: ["categories"] });
            categoryId = newCat.id;
            showCreateCategory = false;
            newCategoryName = "";
            newCategoryColor = DEFAULT_CATEGORY_COLOR;
            useCustomCategoryColor = false;
        },
    });

    async function handleCreateCategory() {
        if (!newCategoryName.trim()) return;
        $createCategoryMut.mutate({
            name: newCategoryName.trim(),
            color: newCategoryColor,
        });
    }

    // Create unit mutation
    const createUnitMut = createMutation({
        mutationFn: (data: { name: string; symbol: string; type: string }) =>
            createUnit({
                name: data.name,
                symbol: data.symbol,
                type: data.type as
                    | "count"
                    | "time"
                    | "distance"
                    | "volume"
                    | "custom",
            }),
        onSuccess: (newUnit) => {
            queryClient.invalidateQueries({ queryKey: ["units"] });
            targetUnit = newUnit.id;
            showCreateUnit = false;
        },
    });

    function handleCreateUnit(data: {
        name: string;
        symbol: string;
        type: string;
    }) {
        $createUnitMut.mutate(data);
    }
</script>

<!-- Modal -->
<dialog class="modal modal-bottom sm:modal-middle" class:modal-open={open}>
    <div
        class="modal-box max-w-4xl p-0 flex flex-col max-h-[92vh] overflow-hidden"
    >
		<!-- Mobile drag handle -->
		<div class="sm:hidden flex justify-center pt-2.5 pb-1 shrink-0">
			<div class="w-10 h-1 rounded-full bg-base-content/15"></div>
		</div>
        <!-- Header -->
        <div
            class="flex items-center gap-4 px-6 py-5 border-b border-base-content/5"
        >
            <!-- Icon Picker -->
            <div class="relative">
                <button
                    type="button"
                    class="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center text-4xl hover:bg-primary/20 transition-all shadow-lg border border-primary/10 hover:scale-105"
                    onclick={() => (showEmojiPicker = !showEmojiPicker)}
                >
                    {icon}
                </button>

                {#if showEmojiPicker}
                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                    <div
                        class="fixed inset-0 z-40"
                        onclick={() => {
                            showEmojiPicker = false;
                            showFullPicker = false;
                        }}
                    ></div>
                    <div
                        bind:this={emojiPickerRef}
                        class="absolute top-18 left-0 z-50 bg-base-100 border border-base-300 rounded-2xl shadow-2xl overflow-hidden"
                    >
                        {#if showFullPicker && emojiPickerReady}
                            <!-- Full emoji picker -->
                            <div
                                class="flex items-center justify-between px-3 pt-3 pb-1"
                            >
                                <button
                                    type="button"
                                    class="btn btn-xs btn-ghost gap-1"
                                    onclick={() => (showFullPicker = false)}
                                >
                                    ← Back to Suggested
                                </button>
                            </div>
                            <emoji-picker
                                class="light"
                                onemoji-click={handleEmojiSelect}
                            ></emoji-picker>
                        {:else}
                            <!-- Suggested emojis panel -->
                            <div class="p-4 w-80 max-h-[400px] overflow-y-auto">
                                <!-- Button to open full picker - at top -->
                                <button
                                    type="button"
                                    class="btn btn-sm btn-ghost w-full mb-4 gap-2 text-base-content/60 hover:text-primary border border-base-300 hover:border-primary/50"
                                    onclick={() => (showFullPicker = true)}
                                >
                                    🔍 Search all emojis...
                                </button>

                                <!-- Quick access row -->
                                <div class="mb-4">
                                    <p
                                        class="text-xs font-medium text-base-content/50 mb-2"
                                    >
                                        Quick Pick
                                    </p>
                                    <div class="flex flex-wrap gap-1">
                                        {#each quickEmojis as emoji}
                                            <button
                                                type="button"
                                                class={cn(
                                                    "w-9 h-9 rounded-xl text-xl hover:bg-base-200 transition-all flex items-center justify-center hover:scale-110",
                                                    icon === emoji &&
                                                        "bg-primary/20 ring-2 ring-primary shadow-sm",
                                                )}
                                                onclick={() =>
                                                    selectQuickEmoji(emoji)}
                                            >
                                                {emoji}
                                            </button>
                                        {/each}
                                    </div>
                                </div>

                                <!-- Categorized suggestions -->
                                {#each Object.entries(suggestedEmojis) as [category, emojis]}
                                    <div class="mb-3">
                                        <p
                                            class="text-[10px] font-semibold text-base-content/40 uppercase mb-1.5"
                                        >
                                            {category}
                                        </p>
                                        <div class="flex flex-wrap gap-1">
                                            {#each emojis as emoji}
                                                <button
                                                    type="button"
                                                    class={cn(
                                                        "w-8 h-8 rounded-lg text-lg hover:bg-base-200 transition-all flex items-center justify-center hover:scale-110",
                                                        icon === emoji &&
                                                            "bg-primary/20 ring-2 ring-primary shadow-sm",
                                                    )}
                                                    onclick={() =>
                                                        selectQuickEmoji(emoji)}
                                                >
                                                    {emoji}
                                                </button>
                                            {/each}
                                        </div>
                                    </div>
                                {/each}
                            </div>
                        {/if}
                    </div>
                {/if}
            </div>

            <div class="flex-1">
                <h3 class="text-xl font-bold text-base-content">
                    {isEditing ? goal?.title || "Edit Goal" : "Create New Goal"}
                </h3>
                {#if isEditing && goal}
                    <div class="flex items-center gap-2 mt-1.5 flex-wrap">
                        <span
                            class={cn(
                                "badge badge-sm gap-1",
                                goal.status === "active"
                                    ? "badge-success"
                                    : goal.status === "completed"
                                      ? "badge-info"
                                      : "badge-neutral",
                            )}
                        >
                            {#if goal.status === "active"}
                                <Circle class="w-2.5 h-2.5 fill-current" />
                            {:else if goal.status === "completed"}
                                <CheckCircle2 class="w-3 h-3" />
                            {:else}
                                <History class="w-3 h-3" />
                            {/if}
                            {goal.status}
                        </span>
                        {#if currentStreak > 0}
                            <span
                                class="badge badge-sm gap-1 bg-gradient-to-r from-orange-500 to-yellow-500 text-white border-0"
                            >
                                <Flame class="w-3 h-3" />
                                {currentStreak} day streak
                            </span>
                        {/if}
                        {#if isHabit}
                            <span class="badge badge-sm badge-ghost gap-1">
                                <Repeat class="w-3 h-3" />
                                {getPeriodLabel(period)}
                            </span>
                        {/if}
                        {#if isMeasurable && progressPercent > 0}
                            <span class="badge badge-sm badge-ghost gap-1">
                                <Target class="w-3 h-3" />
                                {progressPercent}%
                            </span>
                        {/if}
                    </div>
                {/if}
            </div>

            <button
                class="btn btn-ghost btn-sm btn-circle hover:bg-base-200"
                onclick={handleClose}
            >
                <X class="w-5 h-5" />
            </button>
        </div>

        <!-- Tabs -->
        <div class="flex border-b border-base-content/5 px-6">
            <button
                class={cn(
                    "px-5 py-3.5 text-sm font-medium border-b-2 -mb-px transition-all flex items-center gap-2",
                    activeTab === "details"
                        ? "border-primary text-primary"
                        : "border-transparent text-base-content/60 hover:text-base-content hover:bg-base-200/30",
                )}
                onclick={() => (activeTab = "details")}
            >
                <Target class="w-4 h-4" />
                Details
            </button>
            {#if isEditing}
                <button
                    class={cn(
                        "px-5 py-3.5 text-sm font-medium border-b-2 -mb-px transition-all flex items-center gap-2",
                        activeTab === "tasks"
                            ? "border-primary text-primary"
                            : "border-transparent text-base-content/60 hover:text-base-content hover:bg-base-200/30",
                    )}
                    onclick={() => (activeTab = "tasks")}
                >
                    <ListTodo class="w-4 h-4" />
                    Tasks
                    {#if linkedTasks.length > 0}
                        <span class="badge badge-xs badge-primary"
                            >{linkedTasks.length}</span
                        >
                    {/if}
                </button>
                <button
                    class={cn(
                        "px-5 py-3.5 text-sm font-medium border-b-2 -mb-px transition-all flex items-center gap-2",
                        activeTab === "history"
                            ? "border-primary text-primary"
                            : "border-transparent text-base-content/60 hover:text-base-content hover:bg-base-200/30",
                    )}
                    onclick={() => (activeTab = "history")}
                >
                    <History class="w-4 h-4" />
                    History
                    {#if logs.length > 0}
                        <span class="badge badge-xs badge-ghost"
                            >{logs.length}</span
                        >
                    {/if}
                </button>
                <button
                    class={cn(
                        "px-5 py-3.5 text-sm font-medium border-b-2 -mb-px transition-all flex items-center gap-2",
                        activeTab === "analytics"
                            ? "border-primary text-primary"
                            : "border-transparent text-base-content/60 hover:text-base-content hover:bg-base-200/30",
                    )}
                    onclick={() => (activeTab = "analytics")}
                >
                    <BarChart3 class="w-4 h-4" />
                    Analytics
                </button>
            {/if}
        </div>

        <!-- Content -->
        <div class="flex-1 overflow-y-auto">
            {#if activeTab === "details"}
                <div class="p-6 space-y-5">
                    <!-- Title & Description -->
                    <div class="space-y-3">
                        <div class="form-control">
                            <label class="label py-0.5" for="goal-title">
                                <span class="label-text font-semibold"
                                    >What's your goal?</span
                                >
                            </label>
                            <input
                                id="goal-title"
                                type="text"
                                bind:value={title}
                                placeholder="e.g., Read 30 minutes daily, Run 5km..."
                                class="input input-bordered w-full bg-base-200/30 focus:bg-base-100 border-base-300 focus:border-primary"
                            />
                        </div>

                        <div class="form-control">
                            <label class="label py-0.5" for="goal-description">
                                <span
                                    class="label-text text-xs text-base-content/60"
                                    >Description (optional)</span
                                >
                            </label>
                            <textarea
                                id="goal-description"
                                bind:value={description}
                                placeholder="Add more details..."
                                class="textarea textarea-bordered w-full h-16 resize-none bg-base-200/30 focus:bg-base-100 text-sm"
                            ></textarea>
                        </div>
                    </div>

                    <!-- Status (prominent when editing) -->
                    {#if isEditing}
                        <div
                            class="flex items-center gap-2 p-3 rounded-xl bg-base-200/50 border border-base-300"
                        >
                            <span
                                class="text-sm font-semibold text-base-content/70"
                                >Status:</span
                            >
                            <div class="flex gap-1 flex-1">
                                {#each [{ value: "active", label: "Active", icon: Play, color: "btn-success" }, { value: "completed", label: "Completed", icon: CheckCircle2, color: "btn-info" }, { value: "archived", label: "Archived", icon: History, color: "btn-neutral" }] as opt}
                                    {@const Icon = opt.icon}
                                    <button
                                        type="button"
                                        class={cn(
                                            "btn btn-sm gap-1.5 flex-1",
                                            status === opt.value
                                                ? opt.color
                                                : "btn-ghost border border-base-300",
                                        )}
                                        onclick={() => (status = opt.value)}
                                    >
                                        <Icon class="w-4 h-4" />
                                        {opt.label}
                                    </button>
                                {/each}
                            </div>
                        </div>
                    {/if}

                    <!-- Dates Row -->
                    <div class="grid grid-cols-2 gap-3">
                        <!-- Start Date -->
                        <div class="form-control">
                            <label class="label py-0.5" for="goal-start-date">
                                <span
                                    class="label-text text-xs font-medium flex items-center gap-1"
                                >
                                    <Calendar class="w-3 h-3 opacity-50" />
                                    Start Date
                                </span>
                            </label>
                            <input
                                id="goal-start-date"
                                type="date"
                                class="input input-sm input-bordered"
                                bind:value={startDate}
                            />
                        </div>

                        <!-- Deadline -->
                        <div class="form-control">
                            <label class="label py-0.5" for="goal-deadline">
                                <span
                                    class="label-text text-xs font-medium flex items-center gap-1"
                                >
                                    <Clock class="w-3 h-3 opacity-50" />
                                    Deadline
                                </span>
                            </label>
                            <input
                                id="goal-deadline"
                                type="date"
                                class="input input-sm input-bordered"
                                bind:value={deadline}
                            />
                        </div>
                    </div>

                    <!-- Category & Priority Row -->
                    <div class="grid grid-cols-2 gap-4">
                        <!-- Category -->
                        <div class="space-y-2">
                            <div class="flex items-center gap-2">
                                <Tag class="w-4 h-4 opacity-50" />
                                <span
                                    class="text-xs font-semibold uppercase opacity-50"
                                    >Category</span
                                >
                            </div>
                            {#if showCreateCategory}
                                <div class="space-y-2">
                                    <input
                                        type="text"
                                        bind:value={newCategoryName}
                                        placeholder="Category name"
                                        class={cn(
                                            "input input-sm input-bordered w-full",
                                            newCategoryName.length > 40 &&
                                                "input-error",
                                        )}
                                        maxlength={40}
                                    />
                                    <div
                                        class="flex items-center justify-between"
                                    >
                                        <span class="text-xs opacity-60"
                                            >Color</span
                                        >
                                        <label
                                            class="flex items-center gap-2 cursor-pointer"
                                        >
                                            <span class="text-xs opacity-60"
                                                >Custom</span
                                            >
                                            <input
                                                type="checkbox"
                                                class="toggle toggle-xs toggle-primary"
                                                bind:checked={
                                                    useCustomCategoryColor
                                                }
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
                                            type="button"
                                            class="btn btn-sm btn-primary flex-1 gap-1"
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
                                            type="button"
                                            class="btn btn-sm btn-ghost"
                                            onclick={() =>
                                                (showCreateCategory = false)}
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
                                    onCreate={() => (showCreateCategory = true)}
                                />
                            {/if}
                        </div>

                        <!-- Priority -->
                        <div class="space-y-2">
                            <GoalPrioritySlider bind:value={priority} />
                        </div>
                    </div>

                    <!-- Goal Type Selection -->
                    <div
                        class="rounded-2xl bg-base-200/30 border border-base-300 p-4 space-y-3"
                    >
                        <div class="flex items-center gap-2">
                            <BarChart3 class="w-4 h-4 opacity-50" />
                            <span
                                class="text-xs font-semibold uppercase opacity-50"
                                >Goal Type</span
                            >
                        </div>

                        <!-- Recurring Habit Settings -->
                        <GoalRecurrenceSettings
                            bind:isHabit
                            bind:frequency
                            bind:period
                            bind:activeDays
                            bind:activeMonthDay
                            onIsHabitChange={(v) => (isHabit = v)}
                            onFrequencyChange={(v) => (frequency = v)}
                            onPeriodChange={(v) => (period = v)}
                            onActiveDaysChange={(days) => (activeDays = days)}
                            onActiveMonthDayChange={(d) => (activeMonthDay = d)}
                        />

                        <!-- Measurable Target Settings -->
                        <GoalTargetSettings
                            bind:isMeasurable
                            bind:targetOperator
                            bind:targetValue
                            bind:targetUnit
                            bind:showCreateUnit
                            {isHabit}
                            {period}
                            {units}
                            {selectedUnit}
                            onIsMeasurableChange={(v) => (isMeasurable = v)}
                            onTargetOperatorChange={(v) => (targetOperator = v)}
                            onTargetValueChange={(v) => (targetValue = v)}
                            onTargetUnitChange={(v) => (targetUnit = v)}
                            onShowCreateUnit={() => (showCreateUnit = true)}
                            onCancelCreateUnit={() => (showCreateUnit = false)}
                            onCreateUnit={handleCreateUnit}
                            isCreatingUnit={$createUnitMut.isPending}
                        />
                    </div>
                </div>
            {:else if activeTab === "tasks"}
                <GoalTasksTab {linkedTasks} {totalContributions} />
            {:else if activeTab === "history"}
                <GoalHistoryTab
                    {goal}
                    {logs}
                    isLoading={$logsQuery.isPending}
                    {currentStreak}
                    {longestStreak}
                    {progressPercent}
                    {currentValue}
                />
            {:else if activeTab === "analytics"}
                <div class="p-6 space-y-6">
                    {#if goal}
                        <!-- Progress Chart -->
                        <div>
                            <h4
                                class="font-semibold mb-4 flex items-center gap-2"
                            >
                                <BarChart3 class="w-4 h-4 text-primary" />
                                Progress Overview (Last 30 Days)
                            </h4>
                            <GoalProgressChart {goal} days={30} />
                        </div>

                        <!-- Streak History -->
                        <div class="border-t border-base-200 pt-6">
                            <h4
                                class="font-semibold mb-4 flex items-center gap-2"
                            >
                                <Flame class="w-4 h-4 text-orange-500" />
                                Streak History
                            </h4>
                            <div class="max-h-[300px] overflow-y-auto">
                                <GoalStreakHistory {goal} limit={20} />
                            </div>
                        </div>
                    {:else}
                        <div class="text-center py-16 text-base-content/50">
                            <BarChart3
                                class="w-12 h-12 mx-auto mb-3 opacity-30"
                            />
                            <p class="font-medium">Save your goal first</p>
                            <p class="text-sm mt-1">
                                Analytics will appear after you create the goal
                            </p>
                        </div>
                    {/if}
                </div>
            {/if}
        </div>

        <!-- Footer -->
        <div
            class="flex items-center justify-between gap-2 px-4 sm:px-6 py-4 border-t border-base-content/5"
        >
            <div class="text-xs text-base-content/40">
                {#if isEditing}
                    <span
                        >Last updated: {new Date(
                            goal?.updated_at || "",
                        ).toLocaleDateString()}</span
                    >
                {/if}
            </div>
            <div class="flex items-center gap-3">
                <button
                    class="btn btn-ghost"
                    onclick={handleClose}
                    disabled={isPending}
                >
                    Cancel
                </button>
                <button
                    class="btn btn-primary gap-2 min-w-[140px] shadow-lg"
                    onclick={handleSubmit}
                    disabled={isPending || !title.trim()}
                >
                    {#if isPending}
                        <span class="loading loading-spinner loading-sm"></span>
                    {:else}
                        <Save class="w-4 h-4" />
                    {/if}
                    {isEditing ? "Save Changes" : "Create Goal"}
                </button>
            </div>
        </div>
    </div>

    <!-- Backdrop -->
    <form method="dialog" class="modal-backdrop bg-base-content/20 backdrop-blur-sm">
        <button onclick={handleClose}>close</button>
    </form>
</dialog>
