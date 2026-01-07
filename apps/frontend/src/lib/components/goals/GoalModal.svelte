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
    } from "$lib/api";
    import { CategoryDropdown } from "$lib/components/ui";
    import { cn } from "$lib/utils";

    import {
        Save,
        Repeat,
        BarChart3,
        Calendar,
        Flag,
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
    let activeDays = $state<string[]>([]);

    // Target settings
    let isMeasurable = $state(false);
    let targetOperator = $state<"gte" | "lte" | "eq">("gte");
    let targetValue = $state(10);
    let targetUnit = $state("");
    let targetPerPeriod = $state(false);

    // Tab state
    let activeTab = $state<"details" | "tasks" | "history">("details");

    // Tasks tab filter
    let taskFilter = $state<"all" | "positive" | "negative" | "neutral">("all");

    // Emoji picker state
    let showEmojiPicker = $state(false);

    const isEditing = $derived(!!goal);

    // Common units
    const commonUnits = [
        { value: "count", label: "Times" },
        { value: "hours", label: "Hours" },
        { value: "minutes", label: "Minutes" },
        { value: "steps", label: "Steps" },
        { value: "km", label: "Kilometers" },
        { value: "miles", label: "Miles" },
        { value: "calories", label: "Calories" },
        { value: "pages", label: "Pages" },
        { value: "glasses", label: "Glasses" },
        { value: "liters", label: "Liters" },
        { value: "dollars", label: "Dollars" },
    ];

    // Operator options
    const operatorOptions = [
        {
            value: "gte",
            label: "At least",
            symbol: "≥",
            desc: "Achieve or exceed",
        },
        {
            value: "lte",
            label: "At most",
            symbol: "≤",
            desc: "Stay under limit",
        },
        {
            value: "eq",
            label: "Exactly",
            symbol: "=",
            desc: "Hit exact target",
        },
    ];

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
                activeDays = goal.recurrence.active_days || [];
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
                        ? activeDays
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

    function toggleDay(day: string) {
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
    const progressPercent = $derived(goal?.stats?.progress_percent || 0);
    const currentValue = $derived(goal?.stats?.current_value || 0);

    // Linked tasks
    const linkedTasks = $derived(goal?.linked_tasks || []);
    const filteredTasks = $derived(
        taskFilter === "all"
            ? linkedTasks
            : linkedTasks.filter((t) => t.impact_type === taskFilter),
    );
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
</script>

<!-- Modal -->
<dialog class="modal modal-bottom sm:modal-middle" class:modal-open={open}>
    <div class="modal-box max-w-3xl p-0 flex flex-col bg-base-100 max-h-[90vh]">
        <!-- Header -->
        <div
            class="flex items-center gap-4 px-6 py-5 border-b border-base-200 bg-gradient-to-r from-base-200/50 to-transparent"
        >
            <!-- Icon Picker -->
            <div class="relative">
                <button
                    type="button"
                    class="w-14 h-14 rounded-2xl bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center text-3xl hover:from-primary/30 hover:to-primary/10 transition-all shadow-sm border border-primary/10"
                    onclick={() => (showEmojiPicker = !showEmojiPicker)}
                >
                    {icon}
                </button>

                {#if showEmojiPicker}
                    <div
                        class="absolute top-16 left-0 z-50 bg-base-100 border border-base-300 rounded-2xl shadow-2xl p-4 w-80"
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
                                        "w-9 h-9 rounded-xl text-xl hover:bg-base-200 transition-all flex items-center justify-center",
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
                    {isEditing ? "Edit Goal" : "Create Goal"}
                </h3>
                {#if isEditing && goal}
                    <div class="flex items-center gap-2 mt-1">
                        <span
                            class={cn(
                                "badge badge-sm",
                                goal.status === "active"
                                    ? "badge-success"
                                    : goal.status === "completed"
                                      ? "badge-info"
                                      : "badge-neutral",
                            )}
                        >
                            {goal.status}
                        </span>
                        {#if currentStreak > 0}
                            <span class="badge badge-sm badge-warning gap-1">
                                <Flame class="w-3 h-3" />
                                {currentStreak} streak
                            </span>
                        {/if}
                    </div>
                {/if}
            </div>

            <button
                class="btn btn-ghost btn-sm btn-circle"
                onclick={handleClose}
            >
                <X class="w-5 h-5" />
            </button>
        </div>

        <!-- Tabs -->
        <div class="flex border-b border-base-200 px-6">
            <button
                class={cn(
                    "px-5 py-3 text-sm font-medium border-b-2 -mb-px transition-colors flex items-center gap-2",
                    activeTab === "details"
                        ? "border-primary text-primary"
                        : "border-transparent text-base-content/60 hover:text-base-content",
                )}
                onclick={() => (activeTab = "details")}
            >
                <Target class="w-4 h-4" />
                Details
            </button>
            {#if isEditing && linkedTasks.length > 0}
                <button
                    class={cn(
                        "px-5 py-3 text-sm font-medium border-b-2 -mb-px transition-colors flex items-center gap-2",
                        activeTab === "tasks"
                            ? "border-primary text-primary"
                            : "border-transparent text-base-content/60 hover:text-base-content",
                    )}
                    onclick={() => (activeTab = "tasks")}
                >
                    <ListTodo class="w-4 h-4" />
                    Tasks
                    <span class="badge badge-xs badge-primary"
                        >{linkedTasks.length}</span
                    >
                </button>
            {/if}
            <button
                class={cn(
                    "px-5 py-3 text-sm font-medium border-b-2 -mb-px transition-colors flex items-center gap-2",
                    activeTab === "history"
                        ? "border-primary text-primary"
                        : "border-transparent text-base-content/60 hover:text-base-content",
                )}
                onclick={() => (activeTab = "history")}
            >
                <History class="w-4 h-4" />
                History
            </button>
        </div>

        <!-- Content -->
        <div class="flex-1 overflow-y-auto">
            {#if activeTab === "details"}
                <div class="p-6 space-y-6">
                    <!-- Title & Description -->
                    <div class="space-y-4">
                        <div class="form-control">
                            <label class="label" for="goal-title">
                                <span class="label-text font-medium"
                                    >Goal Title <span class="text-error">*</span
                                    ></span
                                >
                            </label>
                            <input
                                id="goal-title"
                                type="text"
                                bind:value={title}
                                placeholder="What do you want to achieve?"
                                class="input input-bordered w-full text-lg"
                            />
                        </div>

                        <div class="form-control">
                            <label class="label" for="goal-description">
                                <span class="label-text font-medium"
                                    >Description</span
                                >
                            </label>
                            <textarea
                                id="goal-description"
                                bind:value={description}
                                placeholder="Add more details about your goal..."
                                class="textarea textarea-bordered w-full h-20 resize-none"
                            ></textarea>
                        </div>
                    </div>

                    <!-- Quick Settings Grid -->
                    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
                        <!-- Category -->
                        <div class="form-control">
                            <label class="label py-1">
                                <span
                                    class="label-text text-xs font-medium flex items-center gap-1.5"
                                >
                                    <Tag class="w-3.5 h-3.5 opacity-50" />
                                    Category
                                </span>
                            </label>
                            <CategoryDropdown
                                {categories}
                                bind:value={categoryId}
                                placeholder="Select..."
                                size="sm"
                            />
                        </div>

                        <!-- Priority -->
                        <div class="form-control">
                            <label class="label py-1">
                                <span
                                    class="label-text text-xs font-medium flex items-center gap-1.5"
                                >
                                    <Flag class="w-3.5 h-3.5 opacity-50" />
                                    Priority
                                </span>
                            </label>
                            <select
                                class="select select-bordered select-sm w-full"
                                bind:value={priority}
                            >
                                <option value={0}>None</option>
                                <option value={1}>Low</option>
                                <option value={2}>Medium</option>
                                <option value={3}>High</option>
                            </select>
                        </div>

                        <!-- Start Date -->
                        <div class="form-control">
                            <label class="label py-1">
                                <span
                                    class="label-text text-xs font-medium flex items-center gap-1.5"
                                >
                                    <Calendar class="w-3.5 h-3.5 opacity-50" />
                                    Start
                                </span>
                            </label>
                            <input
                                type="date"
                                bind:value={startDate}
                                class="input input-bordered input-sm w-full"
                            />
                        </div>

                        <!-- Deadline -->
                        <div class="form-control">
                            <label class="label py-1">
                                <span
                                    class="label-text text-xs font-medium flex items-center gap-1.5"
                                >
                                    <Clock class="w-3.5 h-3.5 opacity-50" />
                                    Deadline
                                </span>
                            </label>
                            <input
                                type="date"
                                bind:value={deadline}
                                class="input input-bordered input-sm w-full"
                            />
                        </div>
                    </div>

                    <!-- Goal Type Cards -->
                    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
                        <!-- Recurring Habit -->
                        <div
                            class={cn(
                                "rounded-2xl border-2 p-5 transition-all cursor-pointer",
                                isHabit
                                    ? "border-primary bg-primary/5 shadow-sm"
                                    : "border-base-200 hover:border-base-300 bg-base-50",
                            )}
                            role="button"
                            tabindex="0"
                            onclick={() => (isHabit = !isHabit)}
                            onkeydown={(e) =>
                                e.key === "Enter" && (isHabit = !isHabit)}
                        >
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <div
                                        class={cn(
                                            "w-10 h-10 rounded-xl flex items-center justify-center",
                                            isHabit
                                                ? "bg-primary text-primary-content"
                                                : "bg-base-200",
                                        )}
                                    >
                                        <Repeat class="w-5 h-5" />
                                    </div>
                                    <div>
                                        <p class="font-semibold">
                                            Recurring Habit
                                        </p>
                                        <p class="text-xs text-base-content/60">
                                            Track daily, weekly, or monthly
                                        </p>
                                    </div>
                                </div>
                                <input
                                    type="checkbox"
                                    class="checkbox checkbox-primary"
                                    checked={isHabit}
                                    onclick={(e) => e.stopPropagation()}
                                    onchange={() => (isHabit = !isHabit)}
                                />
                            </div>

                            {#if isHabit}
                                <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
                                <div
                                    class="mt-5 pt-4 border-t border-base-200 space-y-4"
                                    role="presentation"
                                    onclick={(e) => e.stopPropagation()}
                                >
                                    <div class="flex gap-3">
                                        <div class="form-control flex-1">
                                            <label
                                                class="label py-0 pb-1"
                                                for="habit-frequency"
                                            >
                                                <span class="label-text text-xs"
                                                    >Frequency</span
                                                >
                                            </label>
                                            <input
                                                id="habit-frequency"
                                                type="number"
                                                min="1"
                                                max="30"
                                                class="input input-bordered input-sm w-full"
                                                bind:value={frequency}
                                            />
                                        </div>
                                        <div class="form-control flex-1">
                                            <label
                                                class="label py-0 pb-1"
                                                for="habit-period"
                                            >
                                                <span class="label-text text-xs"
                                                    >Period</span
                                                >
                                            </label>
                                            <select
                                                id="habit-period"
                                                class="select select-bordered select-sm w-full"
                                                bind:value={period}
                                            >
                                                <option value="day"
                                                    >Daily</option
                                                >
                                                <option value="week"
                                                    >Weekly</option
                                                >
                                                <option value="month"
                                                    >Monthly</option
                                                >
                                            </select>
                                        </div>
                                    </div>

                                    {#if period === "week"}
                                        <div class="form-control">
                                            <span class="label py-0 pb-2">
                                                <span class="label-text text-xs"
                                                    >Active Days</span
                                                >
                                            </span>
                                            <div class="flex gap-1.5">
                                                {#each dayOptions as day}
                                                    <button
                                                        type="button"
                                                        class={cn(
                                                            "w-9 h-9 rounded-lg text-xs font-medium transition-all",
                                                            activeDays.includes(
                                                                day.value,
                                                            )
                                                                ? "bg-primary text-primary-content"
                                                                : "bg-base-200 hover:bg-base-300",
                                                        )}
                                                        onclick={() =>
                                                            toggleDay(
                                                                day.value,
                                                            )}
                                                        title={day.full}
                                                    >
                                                        {day.label}
                                                    </button>
                                                {/each}
                                            </div>
                                        </div>
                                    {/if}
                                </div>
                            {/if}
                        </div>

                        <!-- Measurable Target -->
                        <div
                            class={cn(
                                "rounded-2xl border-2 p-5 transition-all cursor-pointer",
                                isMeasurable
                                    ? "border-accent bg-accent/5 shadow-sm"
                                    : "border-base-200 hover:border-base-300 bg-base-50",
                            )}
                            role="button"
                            tabindex="0"
                            onclick={() => (isMeasurable = !isMeasurable)}
                            onkeydown={(e) =>
                                e.key === "Enter" &&
                                (isMeasurable = !isMeasurable)}
                        >
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <div
                                        class={cn(
                                            "w-10 h-10 rounded-xl flex items-center justify-center",
                                            isMeasurable
                                                ? "bg-accent text-accent-content"
                                                : "bg-base-200",
                                        )}
                                    >
                                        <TrendingUp class="w-5 h-5" />
                                    </div>
                                    <div>
                                        <p class="font-semibold">
                                            Measurable Target
                                        </p>
                                        <p class="text-xs text-base-content/60">
                                            Track progress with numbers
                                        </p>
                                    </div>
                                </div>
                                <input
                                    type="checkbox"
                                    class="checkbox checkbox-accent"
                                    checked={isMeasurable}
                                    onclick={(e) => e.stopPropagation()}
                                    onchange={() =>
                                        (isMeasurable = !isMeasurable)}
                                />
                            </div>

                            {#if isMeasurable}
                                <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
                                <div
                                    class="mt-5 pt-4 border-t border-base-200 space-y-4"
                                    role="presentation"
                                    onclick={(e) => e.stopPropagation()}
                                >
                                    <div class="form-control">
                                        <span class="label py-0 pb-1">
                                            <span class="label-text text-xs"
                                                >Target Type</span
                                            >
                                        </span>
                                        <div class="grid grid-cols-3 gap-2">
                                            {#each operatorOptions as op}
                                                <button
                                                    type="button"
                                                    class={cn(
                                                        "btn btn-sm",
                                                        targetOperator ===
                                                            op.value
                                                            ? "btn-accent"
                                                            : "btn-ghost border border-base-300",
                                                    )}
                                                    onclick={() =>
                                                        (targetOperator =
                                                            op.value as
                                                                | "gte"
                                                                | "lte"
                                                                | "eq")}
                                                >
                                                    <span class="font-bold"
                                                        >{op.symbol}</span
                                                    >
                                                    <span
                                                        class="hidden sm:inline text-xs"
                                                        >{op.label}</span
                                                    >
                                                </button>
                                            {/each}
                                        </div>
                                    </div>

                                    <div class="flex gap-3">
                                        <div class="form-control flex-1">
                                            <label
                                                class="label py-0 pb-1"
                                                for="target-value"
                                            >
                                                <span class="label-text text-xs"
                                                    >Value</span
                                                >
                                            </label>
                                            <input
                                                id="target-value"
                                                type="number"
                                                min="1"
                                                class="input input-bordered input-sm w-full"
                                                bind:value={targetValue}
                                            />
                                        </div>
                                        <div class="form-control flex-1">
                                            <label
                                                class="label py-0 pb-1"
                                                for="target-unit"
                                            >
                                                <span class="label-text text-xs"
                                                    >Unit</span
                                                >
                                            </label>
                                            <select
                                                id="target-unit"
                                                class="select select-bordered select-sm w-full"
                                                bind:value={targetUnit}
                                            >
                                                {#each commonUnits as unit}
                                                    <option value={unit.value}
                                                        >{unit.label}</option
                                                    >
                                                {/each}
                                            </select>
                                        </div>
                                    </div>

                                    <label
                                        class="flex items-center gap-2 cursor-pointer"
                                    >
                                        <input
                                            type="checkbox"
                                            class="checkbox checkbox-xs checkbox-accent"
                                            bind:checked={targetPerPeriod}
                                        />
                                        <span
                                            class="text-xs text-base-content/70"
                                            >Reset each period</span
                                        >
                                    </label>
                                </div>
                            {/if}
                        </div>
                    </div>

                    <!-- Status (only for editing) -->
                    {#if isEditing}
                        <div class="form-control">
                            <span class="label">
                                <span class="label-text font-medium"
                                    >Status</span
                                >
                            </span>
                            <div class="flex gap-2">
                                {#each [{ value: "active", label: "Active", icon: Circle, color: "btn-success" }, { value: "completed", label: "Completed", icon: CheckCircle2, color: "btn-info" }, { value: "archived", label: "Archived", icon: History, color: "btn-neutral" }] as opt}
                                    {@const Icon = opt.icon}
                                    <button
                                        type="button"
                                        class={cn(
                                            "btn btn-sm gap-1.5",
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
                </div>
            {:else if activeTab === "tasks"}
                <!-- Tasks Tab -->
                <div class="p-6">
                    {#if linkedTasks.length > 0}
                        <!-- Filter buttons -->
                        <div class="flex gap-2 mb-4">
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
                                    onclick={() => (taskFilter = "positive")}
                                >
                                    <Check class="w-3 h-3" />
                                    Positive
                                    <span class="badge badge-xs badge-success"
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
                                    onclick={() => (taskFilter = "negative")}
                                >
                                    <XIcon class="w-3 h-3" />
                                    Negative
                                    <span class="badge badge-xs badge-error"
                                        >{taskCounts.negative}</span
                                    >
                                </button>
                            {/if}
                            {#if taskCounts.neutral > 0}
                                <button
                                    class={cn(
                                        "btn btn-sm gap-1",
                                        taskFilter === "neutral"
                                            ? "btn-neutral"
                                            : "btn-ghost",
                                    )}
                                    onclick={() => (taskFilter = "neutral")}
                                >
                                    <Minus class="w-3 h-3" />
                                    Neutral
                                    <span class="badge badge-xs badge-neutral"
                                        >{taskCounts.neutral}</span
                                    >
                                </button>
                            {/if}
                        </div>

                        <!-- Task list -->
                        <div class="space-y-2 max-h-[400px] overflow-y-auto">
                            {#each filteredTasks as task (task.task_id)}
                                <div
                                    class="flex items-center gap-3 p-3 rounded-xl bg-base-200/50 hover:bg-base-200 transition-colors"
                                >
                                    <!-- Impact indicator -->
                                    <div
                                        class={cn(
                                            "w-8 h-8 rounded-lg flex items-center justify-center",
                                            task.impact_type === "positive"
                                                ? "bg-success/20 text-success"
                                                : task.impact_type ===
                                                    "negative"
                                                  ? "bg-error/20 text-error"
                                                  : "bg-base-300 text-base-content/50",
                                        )}
                                    >
                                        {#if task.impact_type === "positive"}
                                            <Check class="w-4 h-4" />
                                        {:else if task.impact_type === "negative"}
                                            <XIcon class="w-4 h-4" />
                                        {:else}
                                            <Minus class="w-4 h-4" />
                                        {/if}
                                    </div>

                                    <!-- Task info -->
                                    <div class="flex-1 min-w-0">
                                        <p class="font-medium truncate">
                                            {task.task_title}
                                        </p>
                                        <div
                                            class="flex items-center gap-2 text-xs text-base-content/60"
                                        >
                                            {#if task.quantity_value}
                                                <span
                                                    class="badge badge-xs badge-ghost"
                                                >
                                                    +{task.quantity_value}
                                                    {task.unit_id?.replace(
                                                        "units:",
                                                        "",
                                                    ) || ""}
                                                </span>
                                            {/if}
                                            {#if task.impact_magnitude && task.impact_magnitude > 0}
                                                <span class="opacity-50"
                                                    >Impact: {task.impact_magnitude}/5</span
                                                >
                                            {/if}
                                        </div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {:else}
                        <div class="text-center py-12 text-base-content/50">
                            <ListTodo
                                class="w-16 h-16 mx-auto mb-4 opacity-30"
                            />
                            <p class="font-medium">No linked tasks yet</p>
                            <p class="text-sm mt-1">
                                Tasks that contribute to this goal will appear
                                here
                            </p>
                        </div>
                    {/if}
                </div>
            {:else if activeTab === "history"}
                <!-- History Tab -->
                <div class="p-6">
                    {#if isEditing && goal}
                        <!-- Stats Summary -->
                        <div class="grid grid-cols-3 gap-4 mb-6">
                            <div class="stat bg-base-200/50 rounded-xl p-4">
                                <div class="stat-figure text-primary">
                                    <Flame class="w-8 h-8" />
                                </div>
                                <div class="stat-title text-xs">
                                    Current Streak
                                </div>
                                <div class="stat-value text-2xl text-primary">
                                    {currentStreak}
                                </div>
                            </div>
                            <div class="stat bg-base-200/50 rounded-xl p-4">
                                <div class="stat-figure text-success">
                                    <TrendingUp class="w-8 h-8" />
                                </div>
                                <div class="stat-title text-xs">Progress</div>
                                <div class="stat-value text-2xl text-success">
                                    {progressPercent}%
                                </div>
                            </div>
                            <div class="stat bg-base-200/50 rounded-xl p-4">
                                <div class="stat-figure text-info">
                                    <BarChart3 class="w-8 h-8" />
                                </div>
                                <div class="stat-title text-xs">
                                    Current Value
                                </div>
                                <div class="stat-value text-2xl text-info">
                                    {currentValue}
                                </div>
                            </div>
                        </div>

                        <div class="text-center py-8 text-base-content/50">
                            <History
                                class="w-12 h-12 mx-auto mb-3 opacity-30"
                            />
                            <p class="font-medium">
                                Activity history coming soon
                            </p>
                            <p class="text-sm mt-1">
                                Track your daily progress over time
                            </p>
                        </div>
                    {:else}
                        <div class="text-center py-12 text-base-content/50">
                            <History
                                class="w-16 h-16 mx-auto mb-4 opacity-30"
                            />
                            <p class="font-medium">Save your goal first</p>
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
            class="flex items-center justify-end gap-3 px-6 py-4 border-t border-base-200 bg-base-200/30"
        >
            <button
                class="btn btn-ghost"
                onclick={handleClose}
                disabled={isPending}
            >
                Cancel
            </button>
            <button
                class="btn btn-primary gap-2 min-w-[140px]"
                onclick={handleSubmit}
                disabled={isPending || !title.trim()}
            >
                {#if isPending}
                    <span class="loading loading-spinner loading-sm"></span>
                {:else}
                    <Save class="w-4 h-4" />
                {/if}
                {isEditing ? "Update Goal" : "Create Goal"}
            </button>
        </div>
    </div>

    <!-- Backdrop -->
    <form method="dialog" class="modal-backdrop">
        <button onclick={handleClose}>close</button>
    </form>
</dialog>
