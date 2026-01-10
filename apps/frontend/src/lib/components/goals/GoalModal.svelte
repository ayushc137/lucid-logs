<script lang="ts">
    import {
        createMutation,
        useQueryClient,
        createQuery,
    } from "@tanstack/svelte-query";
    import {
        createGoal,
        updateGoal,
        type Goal,
        type CreateGoalRequest,
        getCategories,
        createCategory,
        getGoalLogs,
        getUnits,
        createUnit,
    } from "$lib/api";
    import {
        CategoryDropdown,
        ColorPicker,
        UnitDropdown,
    } from "$lib/components/ui";
    import { GoalPrioritySlider, GoalTimeline } from "$lib/components/goals";
    import { cn } from "$lib/utils";

    import {
        Save,
        Repeat,
        BarChart3,
        Calendar,
        Tag,
        X,
        History,
        Target,
        TrendingUp,
        Clock,
        Flame,
        CheckCircle2,
        Circle,
        ListTodo,
        Check,
        X as XIcon,
        Minus,
        ArrowUp,
        ArrowDown,
        Filter,
        Search,
        Award,
        Plus,
        Play,
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
    let targetPerPeriod = $state(false);

    // Tab state
    let activeTab = $state<"details" | "tasks" | "history">("details");

    // Tasks tab state
    let taskFilter = $state<"all" | "positive" | "negative" | "neutral">("all");
    let taskSearch = $state("");
    let taskSortBy = $state<"date" | "impact">("date");
    let showTaskFilters = $state(false);

    // History tab state
    let historyFilter = $state<"all" | "with_tasks" | "without_tasks">("all");
    let showEventDetails = $state<string | null>(null);

    // Emoji picker state
    let showEmojiPicker = $state(false);

    const isEditing = $derived(!!goal);
    const goalId = $derived(goal?.id);

    // Fetch goal logs when editing
    const logsQuery = createQuery({
        queryKey: () => ["goal-logs", goalId],
        queryFn: () =>
            goalId
                ? getGoalLogs(goalId, { limit: 50 })
                : Promise.resolve({ goal_id: "", logs: [], total: 0 }),
        enabled: !!goalId && open,
    });

    const logs = $derived($logsQuery.data?.logs || []);

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
    let newCategoryColor = $state("#6366f1");
    let useCustomCategoryColor = $state(false);

    // Unit creation state
    let showCreateUnit = $state(false);
    let newUnitName = $state("");
    let newUnitSymbol = $state("");
    let newUnitType = $state<
        "count" | "time" | "distance" | "volume" | "custom"
    >("count");

    // Popular emojis
    const popularEmojis = [
        "🎯",
        "⭐",
        "🏆",
        "💪",
        "🚀",
        "📚",
        "💰",
        "❤️",
        "🏃",
        "🧘",
        "🎨",
        "✍️",
        "🎵",
        "🌱",
        "⚡",
        "🔥",
        "💎",
        "🌟",
        "🎉",
        "✅",
        "📈",
        "🧠",
        "💡",
        "🌈",
    ];

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
                    typeof d === "string" ? parseInt(d, 10) : d,
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
                targetPerPeriod = goal.target.per_period || false;
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
        targetPerPeriod = false;
        activeTab = "details";
        showEmojiPicker = false;
        historyFilter = "all";
        taskFilter = "all";
        taskSearch = "";
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
                per_period: targetPerPeriod,
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
    const currentStreak = $derived(
        goal?.stats?.current_streak || goal?.current_streak || 0,
    );
    const longestStreak = $derived(goal?.stats?.longest_streak || 0);
    const progressPercent = $derived(goal?.stats?.progress_percent || 0);
    const currentValue = $derived(goal?.stats?.current_value || 0);
    const totalContributions = $derived(goal?.stats?.total_contributions || 0);

    // Linked tasks
    const linkedTasks = $derived(goal?.linked_tasks || []);

    // Filter and search tasks
    const filteredTasks = $derived.by(() => {
        let tasks = linkedTasks;

        if (taskFilter !== "all") {
            tasks = tasks.filter((t) => t.impact_type === taskFilter);
        }

        if (taskSearch.trim()) {
            const search = taskSearch.toLowerCase();
            tasks = tasks.filter((t) =>
                t.task_title.toLowerCase().includes(search),
            );
        }

        // Sort
        if (taskSortBy === "impact") {
            tasks = [...tasks].sort(
                (a, b) => (b.impact_magnitude || 0) - (a.impact_magnitude || 0),
            );
        }

        return tasks;
    });

    const taskCounts = $derived({
        all: linkedTasks.length,
        positive: linkedTasks.filter((t) => t.impact_type === "positive")
            .length,
        negative: linkedTasks.filter((t) => t.impact_type === "negative")
            .length,
        neutral: linkedTasks.filter((t) => t.impact_type === "neutral").length,
    });

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
            newCategoryColor = "#6366f1";
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
            newUnitName = "";
            newUnitSymbol = "";
            newUnitType = "count";
        },
    });

    async function handleCreateUnit() {
        if (!newUnitName.trim()) return;
        $createUnitMut.mutate({
            name: newUnitName.trim(),
            symbol: newUnitSymbol.trim() || newUnitName.trim().substring(0, 3),
            type: newUnitType,
        });
    }
</script>

<!-- Modal -->
<dialog class="modal modal-bottom sm:modal-middle" class:modal-open={open}>
    <div
        class="modal-box max-w-4xl p-0 flex flex-col bg-base-100 max-h-[92vh] rounded-3xl shadow-2xl"
    >
        <!-- Header -->
        <div
            class="flex items-center gap-4 px-6 py-5 border-b border-base-200/50 bg-gradient-to-r from-primary/5 via-base-100 to-secondary/5"
        >
            <!-- Icon Picker -->
            <div class="relative">
                <button
                    type="button"
                    class="w-16 h-16 rounded-2xl bg-gradient-to-br from-primary/20 via-primary/10 to-secondary/10 flex items-center justify-center text-4xl hover:from-primary/30 hover:to-secondary/20 transition-all shadow-lg border border-primary/10 hover:scale-105"
                    onclick={() => (showEmojiPicker = !showEmojiPicker)}
                >
                    {icon}
                </button>

                {#if showEmojiPicker}
                    <div
                        class="absolute top-18 left-0 z-50 bg-base-100 border border-base-300 rounded-2xl shadow-2xl p-4 w-80"
                    >
                        <p
                            class="text-xs font-medium text-base-content/50 mb-3"
                        >
                            Choose an icon
                        </p>
                        <div class="grid grid-cols-8 gap-1.5">
                            {#each popularEmojis as emoji}
                                <button
                                    type="button"
                                    class={cn(
                                        "w-9 h-9 rounded-xl text-xl hover:bg-base-200 transition-all flex items-center justify-center hover:scale-110",
                                        icon === emoji &&
                                            "bg-primary/20 ring-2 ring-primary shadow-sm",
                                    )}
                                    onclick={() => {
                                        icon = emoji;
                                        showEmojiPicker = false;
                                    }}
                                >
                                    {emoji}
                                </button>
                            {/each}
                        </div>
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
        <div class="flex border-b border-base-200/50 px-6 bg-base-100/50">
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

                        <!-- Recurring Habit - Expandable -->
                        <div
                            class={cn(
                                "rounded-xl border-2 transition-all",
                                isHabit
                                    ? "border-primary bg-primary/5"
                                    : "border-base-300 bg-base-100 hover:border-primary/30",
                            )}
                        >
                            <button
                                type="button"
                                class="w-full p-3 flex items-center gap-3 text-left"
                                onclick={() => (isHabit = !isHabit)}
                            >
                                <div
                                    class={cn(
                                        "w-8 h-8 rounded-lg flex items-center justify-center transition-all",
                                        isHabit
                                            ? "bg-primary text-primary-content"
                                            : "bg-base-200 text-base-content/50",
                                    )}
                                >
                                    <Repeat class="w-4 h-4" />
                                </div>
                                <div class="flex-1">
                                    <span class="text-sm font-semibold"
                                        >Recurring Habit</span
                                    >
                                    <p class="text-xs text-base-content/50">
                                        Track daily, weekly, or monthly
                                    </p>
                                </div>
                                <div
                                    class={cn(
                                        "w-5 h-5 rounded border-2 flex items-center justify-center transition-all",
                                        isHabit
                                            ? "bg-primary border-primary text-primary-content"
                                            : "border-base-300",
                                    )}
                                >
                                    {#if isHabit}
                                        <Check class="w-3 h-3" />
                                    {/if}
                                </div>
                            </button>
                            {#if isHabit}
                                <div
                                    class="px-3 pb-3 pt-3 border-t border-primary/20 space-y-3"
                                >
                                    <!-- Recurrence Settings inline -->
                                    <div class="flex items-center gap-3">
                                        <span
                                            class="text-xs font-medium opacity-70"
                                            >Repeat every</span
                                        >
                                        <input
                                            type="number"
                                            min="1"
                                            max="365"
                                            bind:value={frequency}
                                            class="input input-sm input-bordered w-16 text-center"
                                        />
                                        <select
                                            bind:value={period}
                                            class="select select-sm select-bordered"
                                        >
                                            <option value="day">day(s)</option>
                                            <option value="week">week(s)</option
                                            >
                                            <option value="month"
                                                >month(s)</option
                                            >
                                        </select>
                                    </div>

                                    {#if period === "week"}
                                        <div class="flex items-center gap-3">
                                            <span
                                                class="text-xs font-medium opacity-70"
                                                >Active on</span
                                            >
                                            <div
                                                class="flex gap-1.5 flex-wrap flex-1"
                                            >
                                                {#each [{ value: 0, label: "Sun" }, { value: 1, label: "Mon" }, { value: 2, label: "Tue" }, { value: 3, label: "Wed" }, { value: 4, label: "Thu" }, { value: 5, label: "Fri" }, { value: 6, label: "Sat" }] as day}
                                                    <button
                                                        type="button"
                                                        class={cn(
                                                            "btn btn-sm px-2.5",
                                                            activeDays.includes(
                                                                day.value,
                                                            )
                                                                ? "btn-primary"
                                                                : "bg-base-200 hover:bg-base-300 border-base-300 text-base-content/70",
                                                        )}
                                                        onclick={() =>
                                                            toggleDay(
                                                                day.value,
                                                            )}
                                                    >
                                                        {day.label}
                                                    </button>
                                                {/each}
                                            </div>
                                        </div>
                                    {/if}

                                    {#if period === "month"}
                                        <div class="flex items-center gap-3">
                                            <span
                                                class="text-xs font-medium opacity-70"
                                                >Day of month</span
                                            >
                                            <select
                                                bind:value={activeMonthDay}
                                                class="select select-sm select-bordered"
                                            >
                                                {#each Array.from({ length: 31 }, (_, i) => i + 1) as d}
                                                    <option value={d}
                                                        >{d}</option
                                                    >
                                                {/each}
                                            </select>
                                        </div>
                                    {/if}

                                    <!-- Habit Summary -->
                                    <div
                                        class="pt-3 mt-3 border-t border-primary/20 flex items-center gap-2"
                                    >
                                        <div
                                            class="w-5 h-5 rounded-md bg-primary/10 flex items-center justify-center"
                                        >
                                            <Repeat
                                                class="w-3 h-3 text-primary"
                                            />
                                        </div>
                                        <p class="text-xs text-base-content/60">
                                            Repeats every <span
                                                class="font-semibold text-primary"
                                                >{frequency > 1
                                                    ? `${frequency} ${period}s`
                                                    : period}</span
                                            >
                                            {#if period === "week" && activeDays.length > 0 && activeDays.length < 7}
                                                on <span class="font-medium"
                                                    >{activeDays
                                                        .map(
                                                            (d) =>
                                                                [
                                                                    "Sun",
                                                                    "Mon",
                                                                    "Tue",
                                                                    "Wed",
                                                                    "Thu",
                                                                    "Fri",
                                                                    "Sat",
                                                                ][d],
                                                        )
                                                        .join(", ")}</span
                                                >
                                            {/if}
                                        </p>
                                    </div>
                                </div>
                            {/if}
                        </div>

                        <!-- Measurable Target - Expandable -->
                        <div
                            class={cn(
                                "rounded-xl border-2 transition-all",
                                isMeasurable
                                    ? "border-secondary bg-secondary/5"
                                    : "border-base-300 bg-base-100 hover:border-secondary/30",
                            )}
                        >
                            <button
                                type="button"
                                class="w-full p-3 flex items-center gap-3 text-left"
                                onclick={() => (isMeasurable = !isMeasurable)}
                            >
                                <div
                                    class={cn(
                                        "w-8 h-8 rounded-lg flex items-center justify-center transition-all",
                                        isMeasurable
                                            ? "bg-secondary text-secondary-content"
                                            : "bg-base-200 text-base-content/50",
                                    )}
                                >
                                    <Target class="w-4 h-4" />
                                </div>
                                <div class="flex-1">
                                    <span class="text-sm font-semibold"
                                        >Measurable Target</span
                                    >
                                    <p class="text-xs text-base-content/50">
                                        Set a specific numeric goal
                                    </p>
                                </div>
                                <div
                                    class={cn(
                                        "w-5 h-5 rounded border-2 flex items-center justify-center transition-all",
                                        isMeasurable
                                            ? "bg-secondary border-secondary text-secondary-content"
                                            : "border-base-300",
                                    )}
                                >
                                    {#if isMeasurable}
                                        <Check class="w-3 h-3" />
                                    {/if}
                                </div>
                            </button>
                            {#if isMeasurable}
                                <div
                                    class="px-3 pb-3 pt-3 border-t border-secondary/20 space-y-3"
                                >
                                    <!-- Target Settings inline -->
                                    <div class="flex items-center gap-2">
                                        <select
                                            bind:value={targetOperator}
                                            class="select select-sm select-bordered"
                                        >
                                            <option value="gte"
                                                >At least (≥)</option
                                            >
                                            <option value="eq"
                                                >Exactly (=)</option
                                            >
                                            <option value="lte"
                                                >At most (≤)</option
                                            >
                                        </select>
                                        <input
                                            type="number"
                                            min="0"
                                            step="1"
                                            bind:value={targetValue}
                                            class="input input-sm input-bordered w-20 text-center"
                                        />
                                        <div class="flex-1">
                                            {#if showCreateUnit}
                                                <!-- Unit Creation Form (replaces dropdown like Category) -->
                                                <div class="space-y-2">
                                                    <input
                                                        type="text"
                                                        bind:value={newUnitName}
                                                        placeholder="Unit name (e.g., hours)"
                                                        class={cn(
                                                            "input input-sm input-bordered w-full",
                                                            newUnitName.length >
                                                                30 &&
                                                                "input-error",
                                                        )}
                                                        maxlength={30}
                                                    />
                                                    <div class="flex gap-2">
                                                        <input
                                                            type="text"
                                                            bind:value={
                                                                newUnitSymbol
                                                            }
                                                            placeholder="Symbol (e.g., h)"
                                                            class="input input-sm input-bordered w-20"
                                                            maxlength={10}
                                                        />
                                                        <select
                                                            bind:value={
                                                                newUnitType
                                                            }
                                                            class="select select-sm select-bordered flex-1"
                                                        >
                                                            <option
                                                                value="count"
                                                                >Count</option
                                                            >
                                                            <option value="time"
                                                                >Time</option
                                                            >
                                                            <option
                                                                value="distance"
                                                                >Distance</option
                                                            >
                                                            <option
                                                                value="volume"
                                                                >Volume</option
                                                            >
                                                            <option
                                                                value="custom"
                                                                >Custom</option
                                                            >
                                                        </select>
                                                    </div>
                                                    <div class="flex gap-2">
                                                        <button
                                                            type="button"
                                                            class="btn btn-sm btn-primary flex-1 gap-1"
                                                            onclick={handleCreateUnit}
                                                            disabled={$createUnitMut.isPending ||
                                                                !newUnitName.trim()}
                                                        >
                                                            {#if $createUnitMut.isPending}
                                                                <span
                                                                    class="loading loading-spinner loading-xs"
                                                                ></span>
                                                            {:else}
                                                                <Check
                                                                    class="w-3.5 h-3.5"
                                                                />
                                                            {/if}
                                                            Create
                                                        </button>
                                                        <button
                                                            type="button"
                                                            class="btn btn-sm btn-ghost"
                                                            onclick={() =>
                                                                (showCreateUnit = false)}
                                                            >Cancel</button
                                                        >
                                                    </div>
                                                </div>
                                            {:else}
                                                <UnitDropdown
                                                    {units}
                                                    bind:value={targetUnit}
                                                    size="sm"
                                                    showCreateButton={true}
                                                    onCreate={() =>
                                                        (showCreateUnit = true)}
                                                />
                                            {/if}
                                        </div>
                                    </div>

                                    {#if isHabit}
                                        <label
                                            class="flex items-center gap-3 cursor-pointer"
                                        >
                                            <input
                                                type="checkbox"
                                                class="checkbox checkbox-sm checkbox-secondary"
                                                bind:checked={targetPerPeriod}
                                            />
                                            <span class="text-sm"
                                                >Reset per {period}</span
                                            >
                                        </label>
                                    {/if}

                                    <!-- Target Summary -->
                                    <div
                                        class="pt-3 mt-3 border-t border-secondary/20 flex items-center gap-2"
                                    >
                                        <div
                                            class="w-5 h-5 rounded-md bg-secondary/10 flex items-center justify-center"
                                        >
                                            <Target
                                                class="w-3 h-3 text-secondary"
                                            />
                                        </div>
                                        <p class="text-xs text-base-content/60">
                                            Target: <span
                                                class="font-semibold text-secondary"
                                                >{targetOperator === "gte"
                                                    ? "at least"
                                                    : targetOperator === "lte"
                                                      ? "at most"
                                                      : "exactly"}
                                                {targetValue || 0}{selectedUnit
                                                    ? ` ${selectedUnit.symbol}`
                                                    : ""}</span
                                            >
                                            {#if isHabit && targetPerPeriod}
                                                <span class="opacity-70"
                                                    >per {period}</span
                                                >
                                            {/if}
                                        </p>
                                    </div>
                                </div>
                            {/if}
                        </div>
                    </div>
                </div>
            {:else if activeTab === "tasks"}
                <!-- Tasks Tab -->
                <div class="p-6">
                    <!-- Stats Bar -->
                    <div
                        class="flex items-center gap-4 mb-4 p-3 bg-base-200/50 rounded-xl"
                    >
                        <div class="flex items-center gap-2 text-sm">
                            <ListTodo class="w-4 h-4 text-primary" />
                            <span class="font-medium">{linkedTasks.length}</span
                            >
                            <span class="text-base-content/60"
                                >linked tasks</span
                            >
                        </div>
                        {#if totalContributions > 0}
                            <div class="flex items-center gap-2 text-sm">
                                <TrendingUp class="w-4 h-4 text-success" />
                                <span class="font-medium"
                                    >{totalContributions}</span
                                >
                                <span class="text-base-content/60"
                                    >contributions</span
                                >
                            </div>
                        {/if}
                    </div>

                    {#if linkedTasks.length > 0}
                        <!-- Filter Bar -->
                        <div class="flex items-center gap-3 mb-4">
                            <!-- Search -->
                            <div class="relative flex-1">
                                <Search
                                    class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-base-content/40"
                                />
                                <input
                                    type="text"
                                    placeholder="Search tasks..."
                                    class="input input-sm input-bordered w-full pl-9 bg-base-200/50"
                                    bind:value={taskSearch}
                                />
                            </div>

                            <!-- Quick Filters -->
                            <div class="flex gap-1">
                                <button
                                    class={cn(
                                        "btn btn-sm gap-1",
                                        taskFilter === "all"
                                            ? "btn-primary"
                                            : "btn-ghost",
                                    )}
                                    onclick={() => (taskFilter = "all")}
                                >
                                    All
                                    <span class="badge badge-xs"
                                        >{taskCounts.all}</span
                                    >
                                </button>
                                {#if taskCounts.positive > 0}
                                    <button
                                        class={cn(
                                            "btn btn-sm gap-1",
                                            taskFilter === "positive"
                                                ? "btn-success"
                                                : "btn-ghost text-success",
                                        )}
                                        onclick={() =>
                                            (taskFilter = "positive")}
                                    >
                                        <Check class="w-3 h-3" />
                                        <span
                                            class="badge badge-xs badge-success"
                                            >{taskCounts.positive}</span
                                        >
                                    </button>
                                {/if}
                                {#if taskCounts.negative > 0}
                                    <button
                                        class={cn(
                                            "btn btn-sm gap-1",
                                            taskFilter === "negative"
                                                ? "btn-error"
                                                : "btn-ghost text-error",
                                        )}
                                        onclick={() =>
                                            (taskFilter = "negative")}
                                    >
                                        <XIcon class="w-3 h-3" />
                                        <span class="badge badge-xs badge-error"
                                            >{taskCounts.negative}</span
                                        >
                                    </button>
                                {/if}
                            </div>
                        </div>

                        <!-- Task List -->
                        <div
                            class="space-y-2 max-h-[400px] overflow-y-auto pr-2"
                        >
                            {#each filteredTasks as task (task.task_id)}
                                <div
                                    class="flex items-center gap-3 p-4 rounded-xl bg-base-200/30 hover:bg-base-200/60 transition-all border border-transparent hover:border-base-300 group"
                                >
                                    <!-- Impact indicator -->
                                    <div
                                        class={cn(
                                            "w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 transition-transform group-hover:scale-110",
                                            task.impact_type === "positive"
                                                ? "bg-success/20 text-success"
                                                : task.impact_type ===
                                                    "negative"
                                                  ? "bg-error/20 text-error"
                                                  : "bg-base-300 text-base-content/50",
                                        )}
                                    >
                                        {#if task.impact_type === "positive"}
                                            <Check class="w-5 h-5" />
                                        {:else if task.impact_type === "negative"}
                                            <XIcon class="w-5 h-5" />
                                        {:else}
                                            <Minus class="w-5 h-5" />
                                        {/if}
                                    </div>

                                    <!-- Task info -->
                                    <div class="flex-1 min-w-0">
                                        <p class="font-medium truncate">
                                            {task.task_title}
                                        </p>
                                        <div
                                            class="flex items-center gap-2 mt-1 flex-wrap"
                                        >
                                            {#if task.quantity_value}
                                                <span
                                                    class="badge badge-sm bg-base-300 gap-1"
                                                >
                                                    <ArrowUp
                                                        class="w-3 h-3 text-success"
                                                    />
                                                    +{task.quantity_value}
                                                    {task.unit_id?.replace(
                                                        "units:",
                                                        "",
                                                    ) || ""}
                                                </span>
                                            {/if}
                                            {#if task.impact_magnitude && task.impact_magnitude > 0}
                                                <span
                                                    class="text-xs text-base-content/50"
                                                >
                                                    Impact: {task.impact_magnitude}/5
                                                </span>
                                            {/if}
                                        </div>
                                    </div>
                                </div>
                            {:else}
                                <div
                                    class="text-center py-8 text-base-content/50"
                                >
                                    <Search
                                        class="w-8 h-8 mx-auto mb-2 opacity-30"
                                    />
                                    <p class="text-sm">
                                        No tasks match your filters
                                    </p>
                                </div>
                            {/each}
                        </div>
                    {:else}
                        <div class="text-center py-16 text-base-content/50">
                            <div
                                class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-base-200 flex items-center justify-center"
                            >
                                <ListTodo class="w-8 h-8 opacity-30" />
                            </div>
                            <p class="font-medium text-lg">
                                No linked tasks yet
                            </p>
                            <p class="text-sm mt-1 max-w-xs mx-auto">
                                Tasks that contribute to this goal will appear
                                here. Link tasks from the task details page.
                            </p>
                        </div>
                    {/if}
                </div>
            {:else if activeTab === "history"}
                <!-- History Tab -->
                <div class="p-6">
                    {#if isEditing && goal}
                        <!-- Stats Summary -->
                        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
                            <div
                                class="p-4 rounded-2xl bg-gradient-to-br from-orange-500/10 to-yellow-500/10 border border-orange-500/20"
                            >
                                <div class="flex items-center gap-2 mb-2">
                                    <Flame class="w-5 h-5 text-orange-500" />
                                </div>
                                <p class="text-2xl font-bold text-orange-500">
                                    {currentStreak}
                                </p>
                                <p class="text-xs text-base-content/60">
                                    Current Streak
                                </p>
                            </div>
                            <div
                                class="p-4 rounded-2xl bg-gradient-to-br from-purple-500/10 to-pink-500/10 border border-purple-500/20"
                            >
                                <div class="flex items-center gap-2 mb-2">
                                    <Award class="w-5 h-5 text-purple-500" />
                                </div>
                                <p class="text-2xl font-bold text-purple-500">
                                    {longestStreak}
                                </p>
                                <p class="text-xs text-base-content/60">
                                    Best Streak
                                </p>
                            </div>
                            <div
                                class="p-4 rounded-2xl bg-gradient-to-br from-green-500/10 to-emerald-500/10 border border-green-500/20"
                            >
                                <div class="flex items-center gap-2 mb-2">
                                    <TrendingUp
                                        class="w-5 h-5 text-green-500"
                                    />
                                </div>
                                <p class="text-2xl font-bold text-green-500">
                                    {progressPercent}%
                                </p>
                                <p class="text-xs text-base-content/60">
                                    Progress
                                </p>
                            </div>
                            <div
                                class="p-4 rounded-2xl bg-gradient-to-br from-blue-500/10 to-cyan-500/10 border border-blue-500/20"
                            >
                                <div class="flex items-center gap-2 mb-2">
                                    <BarChart3 class="w-5 h-5 text-blue-500" />
                                </div>
                                <p class="text-2xl font-bold text-blue-500">
                                    {currentValue}
                                </p>
                                <p class="text-xs text-base-content/60">
                                    Current Value
                                </p>
                            </div>
                        </div>

                        <!-- Activity Timeline -->
                        <div
                            class="border border-base-300 rounded-2xl p-4 bg-base-200/30 max-h-[350px] overflow-y-auto"
                        >
                            <h4 class="font-semibold mb-4 text-center">
                                Activity Timeline
                            </h4>
                            <GoalTimeline
                                {logs}
                                isLoading={$logsQuery.isPending}
                                filter={historyFilter === "with_tasks"
                                    ? "with-tasks"
                                    : historyFilter === "without_tasks"
                                      ? "goal-only"
                                      : "all"}
                                onFilterChange={(f) =>
                                    (historyFilter =
                                        f === "with-tasks"
                                            ? "with_tasks"
                                            : f === "goal-only"
                                              ? "without_tasks"
                                              : "all")}
                            />
                        </div>
                    {:else}
                        <div class="text-center py-16 text-base-content/50">
                            <div
                                class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-base-200 flex items-center justify-center"
                            >
                                <History class="w-8 h-8 opacity-30" />
                            </div>
                            <p class="font-medium text-lg">
                                Save your goal first
                            </p>
                            <p class="text-sm mt-1">
                                History will appear after you create the goal
                            </p>
                        </div>
                    {/if}
                </div>
            {/if}
        </div>

        <!-- Footer -->
        <div
            class="flex items-center justify-between gap-3 px-6 py-4 border-t border-base-200/50 bg-gradient-to-r from-base-200/30 via-transparent to-base-200/30"
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
    <form method="dialog" class="modal-backdrop bg-black/50 backdrop-blur-sm">
        <button onclick={handleClose}>close</button>
    </form>
</dialog>
