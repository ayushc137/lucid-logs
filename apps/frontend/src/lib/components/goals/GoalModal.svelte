<script lang="ts">
    import { createMutation, useQueryClient } from "@tanstack/svelte-query";
    import {
        createGoal,
        updateGoal,
        type Goal,
        type CreateGoalRequest,
        type Recurrence,
    } from "$lib/api";
    import { Modal, ColorPicker, CategoryDropdown } from "$lib/components/ui";
    import {
        Target,
        Save,
        Repeat,
        Calendar,
        Flag,
        Sparkles,
        Trash2,
        AlertCircle,
        X,
        Plus,
        Check,
    } from "lucide-svelte";
    import { cn } from "$lib/utils";

    interface Props {
        open: boolean;
        goal?: Goal | null;
        onClose: () => void;
    }

    let { open = $bindable(), goal = null, onClose }: Props = $props();

    const queryClient = useQueryClient();
    const isEditing = $derived(!!goal);

    // Form state
    let title = $state("");
    let description = $state("");
    let why = $state("");
    let icon = $state("");
    let color = $state("#3B82F6");
    let goalType = $state<"discrete" | "measurable" | "epic" | "avoidance">(
        "discrete",
    );
    let status = $state<"active" | "completed" | "paused" | "abandoned">(
        "active",
    );
    let priority = $state(2);
    let valueScore = $state(3);
    let categoryId = $state<string | undefined>(undefined);
    let lifeDomain = $state("");
    let isPrivate = $state(false);

    // Recurrence state
    let isRecurring = $state(false);
    let frequency = $state(1);
    let period = $state<"day" | "week" | "month">("day");
    let activeDays = $state<string[]>([]);
    let graceDays = $state(0);

    // Target state (for measurable goals)
    let targetValue = $state<number | undefined>(undefined);
    let targetUnit = $state("");
    let targetPerPeriod = $state(false);

    // Date state
    let startDate = $state("");
    let deadline = $state("");

    // UI state
    let showAdvanced = $state(false);
    let errors = $state<Record<string, string>>({});

    // Initialize form when goal changes
    $effect(() => {
        if (open) {
            if (goal) {
                title = goal.title;
                description = goal.description || "";
                why = goal.why || "";
                icon = goal.icon || "";
                color = goal.color || "#3B82F6";
                goalType = goal.goal_type;
                status = goal.status;
                priority = goal.priority || 2;
                valueScore = goal.value_score || 3;
                categoryId = goal.category_id;
                lifeDomain = goal.life_domain || "";
                isPrivate = goal.is_private;

                // Recurrence
                isRecurring = !!goal.recurrence;
                if (goal.recurrence) {
                    frequency = goal.recurrence.frequency;
                    period = goal.recurrence.period;
                    activeDays = goal.recurrence.active_days || [];
                    graceDays = goal.recurrence.grace_days || 0;
                }

                // Target
                if (goal.target) {
                    targetValue = goal.target.value;
                    targetUnit = goal.target.unit;
                    targetPerPeriod = goal.target.per_period || false;
                }

                // Dates
                startDate = goal.start_date
                    ? goal.start_date.split("T")[0]
                    : "";
                deadline = goal.deadline ? goal.deadline.split("T")[0] : "";
            } else {
                resetForm();
            }
        }
    });

    function resetForm() {
        title = "";
        description = "";
        why = "";
        icon = "";
        color = "#3B82F6";
        goalType = "discrete";
        status = "active";
        priority = 2;
        valueScore = 3;
        categoryId = undefined;
        lifeDomain = "";
        isPrivate = false;
        isRecurring = false;
        frequency = 1;
        period = "day";
        activeDays = [];
        graceDays = 0;
        targetValue = undefined;
        targetUnit = "";
        targetPerPeriod = false;
        startDate = "";
        deadline = "";
        showAdvanced = false;
        errors = {};
    }

    // Create mutation
    const createMut = createMutation({
        mutationFn: (data: CreateGoalRequest) => createGoal(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["goals"] });
            handleClose();
        },
        onError: (err: Error) => {
            errors = { submit: err.message };
        },
    });

    // Update mutation
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
            handleClose();
        },
        onError: (err: Error) => {
            errors = { submit: err.message };
        },
    });

    function handleClose() {
        resetForm();
        open = false;
        onClose();
    }

    function validateForm(): boolean {
        const newErrors: Record<string, string> = {};

        if (!title.trim()) {
            newErrors.title = "Title is required";
        }

        if (goalType === "measurable" && !targetValue) {
            newErrors.targetValue =
                "Target value is required for measurable goals";
        }

        if (goalType === "measurable" && !targetUnit.trim()) {
            newErrors.targetUnit =
                "Target unit is required for measurable goals";
        }

        errors = newErrors;
        return Object.keys(newErrors).length === 0;
    }

    function handleSubmit() {
        if (!validateForm()) return;

        const recurrence: Recurrence | undefined = isRecurring
            ? {
                  frequency,
                  period,
                  active_days: activeDays.length > 0 ? activeDays : undefined,
                  grace_days: graceDays > 0 ? graceDays : undefined,
              }
            : undefined;

        const target =
            goalType === "measurable" && targetValue
                ? {
                      value: targetValue,
                      unit: targetUnit,
                      per_period: targetPerPeriod,
                  }
                : undefined;

        const data: CreateGoalRequest = {
            title: title.trim(),
            description: description.trim() || undefined,
            why: why.trim() || undefined,
            icon: icon || undefined,
            color: color || undefined,
            goal_type: goalType,
            recurrence,
            target,
            start_date: startDate ? `${startDate}T00:00:00Z` : undefined,
            deadline: deadline ? `${deadline}T23:59:59Z` : undefined,
            priority: priority as 1 | 2 | 3,
            value_score: valueScore as 1 | 2 | 3 | 4 | 5,
            category_id: categoryId,
            life_domain: lifeDomain || undefined,
            is_private: isPrivate,
        };

        if (isEditing && goal) {
            $updateMut.mutate({ id: goal.id, data });
        } else {
            $createMut.mutate(data);
        }
    }

    const isPending = $derived($createMut.isPending || $updateMut.isPending);

    const goalTypes = [
        {
            value: "discrete",
            label: "One-time",
            desc: "Complete once",
            icon: Check,
        },
        {
            value: "measurable",
            label: "Measurable",
            desc: "Track progress",
            icon: Target,
        },
        {
            value: "avoidance",
            label: "Avoidance",
            desc: "Don't do",
            icon: X,
        },
        { value: "epic", label: "Epic", desc: "Multi-step", icon: Flag },
    ];

    const periods = [
        { value: "day", label: "Daily" },
        { value: "week", label: "Weekly" },
        { value: "month", label: "Monthly" },
    ];

    const weekdays = [
        { value: "mon", label: "M" },
        { value: "tue", label: "T" },
        { value: "wed", label: "W" },
        { value: "thu", label: "T" },
        { value: "fri", label: "F" },
        { value: "sat", label: "S" },
        { value: "sun", label: "S" },
    ];

    function toggleDay(day: string) {
        if (activeDays.includes(day)) {
            activeDays = activeDays.filter((d) => d !== day);
        } else {
            activeDays = [...activeDays, day];
        }
    }

    const lifeDomains = [
        { value: "health", label: "🏃 Health" },
        { value: "work", label: "💼 Work" },
        { value: "learning", label: "📚 Learning" },
        { value: "relationships", label: "❤️ Relationships" },
        { value: "finance", label: "💰 Finance" },
        { value: "hobbies", label: "🎨 Hobbies" },
        { value: "mindfulness", label: "🧘 Mindfulness" },
        { value: "other", label: "✨ Other" },
    ];
</script>

<Modal
    bind:open
    size="lg"
    title={isEditing ? "Edit Goal" : "Create Goal"}
    onClose={handleClose}
>
    {#snippet icon()}
        <Target class="w-5 h-5 text-accent" />
    {/snippet}

    <form
        onsubmit={(e) => {
            e.preventDefault();
            handleSubmit();
        }}
        class="space-y-5"
    >
        {#if errors.submit}
            <div class="alert alert-error gap-2">
                <AlertCircle class="w-4 h-4" />
                <span class="text-sm">{errors.submit}</span>
            </div>
        {/if}

        <!-- Title & Icon Row -->
        <div class="flex gap-3">
            <!-- Icon Picker -->
            <div class="flex flex-col gap-1">
                <label class="text-xs font-semibold uppercase opacity-50"
                    >Icon</label
                >
                <button
                    type="button"
                    class="w-12 h-12 rounded-xl border-2 border-dashed border-base-300 hover:border-primary/50 flex items-center justify-center text-2xl transition-colors"
                    onclick={() => {
                        const emojis = [
                            "🎯",
                            "💪",
                            "📚",
                            "🏃",
                            "💧",
                            "🧘",
                            "💤",
                            "🍎",
                            "✍️",
                            "🎨",
                        ];
                        icon =
                            emojis[Math.floor(Math.random() * emojis.length)];
                    }}
                >
                    {icon || "🎯"}
                </button>
            </div>

            <!-- Title -->
            <div class="flex-1">
                <label
                    class="text-xs font-semibold uppercase opacity-50"
                    for="goal-title"
                >
                    Title *
                </label>
                <input
                    id="goal-title"
                    type="text"
                    class="input input-bordered w-full mt-1"
                    class:input-error={errors.title}
                    placeholder="e.g., Run 5km every day"
                    bind:value={title}
                />
                {#if errors.title}
                    <p class="text-error text-xs mt-1">{errors.title}</p>
                {/if}
            </div>
        </div>

        <!-- Goal Type Selection -->
        <div>
            <label class="text-xs font-semibold uppercase opacity-50"
                >Goal Type</label
            >
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 mt-2">
                {#each goalTypes as type}
                    {@const Icon = type.icon}
                    <button
                        type="button"
                        class={cn(
                            "flex flex-col items-center gap-1 p-3 rounded-xl border-2 transition-all",
                            goalType === type.value
                                ? "border-primary bg-primary/10 text-primary"
                                : "border-base-200 hover:border-base-300",
                        )}
                        onclick={() =>
                            (goalType = type.value as typeof goalType)}
                    >
                        <Icon class="w-5 h-5" />
                        <span class="text-sm font-semibold">{type.label}</span>
                        <span class="text-[10px] opacity-60">{type.desc}</span>
                    </button>
                {/each}
            </div>
        </div>

        <!-- Measurable Target -->
        {#if goalType === "measurable"}
            <div
                class="p-4 rounded-xl bg-base-200/50 border border-base-200 space-y-3"
            >
                <div class="flex items-center gap-2 text-sm font-semibold">
                    <Target class="w-4 h-4 text-accent" />
                    Target Settings
                </div>
                <div class="grid grid-cols-2 gap-3">
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Value *</label
                        >
                        <input
                            type="number"
                            class="input input-bordered input-sm w-full mt-1"
                            class:input-error={errors.targetValue}
                            placeholder="e.g., 100"
                            bind:value={targetValue}
                        />
                    </div>
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Unit *</label
                        >
                        <input
                            type="text"
                            class="input input-bordered input-sm w-full mt-1"
                            class:input-error={errors.targetUnit}
                            placeholder="e.g., km, pages"
                            bind:value={targetUnit}
                        />
                    </div>
                </div>
                <label class="label cursor-pointer justify-start gap-2">
                    <input
                        type="checkbox"
                        class="checkbox checkbox-sm"
                        bind:checked={targetPerPeriod}
                    />
                    <span class="label-text text-sm"
                        >Per period (e.g., 3L per day)</span
                    >
                </label>
            </div>
        {/if}

        <!-- Recurrence Toggle -->
        <div
            class="p-4 rounded-xl bg-base-200/50 border border-base-200 space-y-3"
        >
            <label class="flex items-center gap-3 cursor-pointer">
                <input
                    type="checkbox"
                    class="toggle toggle-primary"
                    bind:checked={isRecurring}
                />
                <div class="flex items-center gap-2">
                    <Repeat class="w-4 h-4 text-primary" />
                    <span class="font-semibold text-sm">Make it a Habit</span>
                </div>
            </label>

            {#if isRecurring}
                <div class="space-y-3 pt-2 border-t border-base-200">
                    <!-- Frequency -->
                    <div class="grid grid-cols-2 gap-3">
                        <div>
                            <label
                                class="text-xs font-semibold uppercase opacity-50"
                                >Times</label
                            >
                            <input
                                type="number"
                                min="1"
                                max="365"
                                class="input input-bordered input-sm w-full mt-1"
                                bind:value={frequency}
                            />
                        </div>
                        <div>
                            <label
                                class="text-xs font-semibold uppercase opacity-50"
                                >Period</label
                            >
                            <select
                                class="select select-bordered select-sm w-full mt-1"
                                bind:value={period}
                            >
                                {#each periods as p}
                                    <option value={p.value}>{p.label}</option>
                                {/each}
                            </select>
                        </div>
                    </div>

                    <!-- Active Days (for weekly) -->
                    {#if period === "week"}
                        <div>
                            <label
                                class="text-xs font-semibold uppercase opacity-50"
                                >Active Days</label
                            >
                            <div class="flex gap-1 mt-2">
                                {#each weekdays as day}
                                    <button
                                        type="button"
                                        class={cn(
                                            "w-8 h-8 rounded-full text-xs font-bold transition-all",
                                            activeDays.includes(day.value)
                                                ? "bg-primary text-primary-content"
                                                : "bg-base-200 hover:bg-base-300",
                                        )}
                                        onclick={() => toggleDay(day.value)}
                                    >
                                        {day.label}
                                    </button>
                                {/each}
                            </div>
                        </div>
                    {/if}

                    <!-- Grace Days -->
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Grace Days</label
                        >
                        <input
                            type="number"
                            min="0"
                            max="7"
                            class="input input-bordered input-sm w-full mt-1"
                            placeholder="Days you can miss without breaking streak"
                            bind:value={graceDays}
                        />
                    </div>
                </div>
            {/if}
        </div>

        <!-- Description & Why -->
        <div class="space-y-3">
            <div>
                <label
                    class="text-xs font-semibold uppercase opacity-50"
                    for="goal-desc"
                >
                    Description
                </label>
                <textarea
                    id="goal-desc"
                    class="textarea textarea-bordered w-full mt-1 h-20"
                    placeholder="What do you want to achieve?"
                    bind:value={description}
                ></textarea>
            </div>
            <div>
                <label
                    class="text-xs font-semibold uppercase opacity-50"
                    for="goal-why"
                >
                    <Sparkles class="w-3 h-3 inline mr-1" />
                    Why does this matter?
                </label>
                <textarea
                    id="goal-why"
                    class="textarea textarea-bordered w-full mt-1 h-16"
                    placeholder="Helpful for retrospectives..."
                    bind:value={why}
                ></textarea>
            </div>
        </div>

        <!-- Advanced Options Toggle -->
        <button
            type="button"
            class="btn btn-ghost btn-sm gap-2"
            onclick={() => (showAdvanced = !showAdvanced)}
        >
            {showAdvanced ? "Hide" : "Show"} Advanced Options
            <Plus
                class={cn(
                    "w-4 h-4 transition-transform",
                    showAdvanced && "rotate-45",
                )}
            />
        </button>

        {#if showAdvanced}
            <div
                class="space-y-4 p-4 rounded-xl bg-base-200/30 border border-base-200"
            >
                <!-- Dates -->
                <div class="grid grid-cols-2 gap-3">
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Start Date</label
                        >
                        <input
                            type="date"
                            class="input input-bordered input-sm w-full mt-1"
                            bind:value={startDate}
                        />
                    </div>
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Deadline</label
                        >
                        <input
                            type="date"
                            class="input input-bordered input-sm w-full mt-1"
                            bind:value={deadline}
                        />
                    </div>
                </div>

                <!-- Priority & Value -->
                <div class="grid grid-cols-2 gap-3">
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Priority</label
                        >
                        <select
                            class="select select-bordered select-sm w-full mt-1"
                            bind:value={priority}
                        >
                            <option value={1}>Low</option>
                            <option value={2}>Medium</option>
                            <option value={3}>High</option>
                        </select>
                    </div>
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Value Score</label
                        >
                        <select
                            class="select select-bordered select-sm w-full mt-1"
                            bind:value={valueScore}
                        >
                            <option value={1}>1 - Low</option>
                            <option value={2}>2</option>
                            <option value={3}>3 - Medium</option>
                            <option value={4}>4</option>
                            <option value={5}>5 - High</option>
                        </select>
                    </div>
                </div>

                <!-- Life Domain -->
                <div>
                    <label class="text-xs font-semibold uppercase opacity-50"
                        >Life Domain</label
                    >
                    <select
                        class="select select-bordered select-sm w-full mt-1"
                        bind:value={lifeDomain}
                    >
                        <option value="">Select domain...</option>
                        {#each lifeDomains as domain}
                            <option value={domain.value}>{domain.label}</option>
                        {/each}
                    </select>
                </div>

                <!-- Category & Color -->
                <div class="grid grid-cols-2 gap-3">
                    <CategoryDropdown
                        bind:selectedId={categoryId}
                        size="sm"
                        showLabel={true}
                    />
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Color</label
                        >
                        <div class="mt-1">
                            <ColorPicker bind:value={color} cols={8} />
                        </div>
                    </div>
                </div>

                <!-- Privacy -->
                <label class="label cursor-pointer justify-start gap-2">
                    <input
                        type="checkbox"
                        class="checkbox checkbox-sm"
                        bind:checked={isPrivate}
                    />
                    <span class="label-text text-sm"
                        >Private goal (hidden from shares)</span
                    >
                </label>
            </div>
        {/if}
    </form>

    {#snippet actions()}
        <button type="button" class="btn btn-ghost" onclick={handleClose}
            >Cancel</button
        >
        <button
            type="button"
            class="btn btn-primary gap-2"
            disabled={isPending}
            onclick={handleSubmit}
        >
            {#if isPending}
                <span class="loading loading-spinner loading-sm"></span>
            {:else}
                <Save class="w-4 h-4" />
            {/if}
            {isEditing ? "Update Goal" : "Create Goal"}
        </button>
    {/snippet}
</Modal>
