<script lang="ts">
    import {
        createMutation,
        createQuery,
        useQueryClient,
    } from "@tanstack/svelte-query";
    import {
        createTemplate,
        updateTemplate,
        getGoals,
        getCategories,
        type TaskTemplate,
        type CreateTemplateRequest,
        type Goal,
    } from "$lib/api";
    import { Modal, CategoryDropdown } from "$lib/components/ui";
    import {
        Zap,
        Save,
        Timer,
        Flag,
        AlertCircle,
        Plus,
        Heart,
        Hash,
        Target,
        Trash2,
        X,
    } from "lucide-svelte";
    import { cn } from "$lib/utils";

    interface Props {
        open: boolean;
        template?: TaskTemplate | null;
        onClose: () => void;
    }

    let { open = $bindable(), template = null, onClose }: Props = $props();

    const queryClient = useQueryClient();
    const isEditing = $derived(!!template);

    // Form state
    let title = $state("");
    let description = $state("");
    let templateIcon = $state("");

    // Defaults
    let defaultDuration = $state<number | undefined>(undefined);
    let defaultCategoryId = $state<string | undefined>(undefined);
    let defaultEmotionId = $state<string | undefined>(undefined);
    let expectedQuadrant = $state<
        "green" | "yellow" | "red" | "blue" | undefined
    >(undefined);

    // Quick log
    let isQuickLog = $state(false);
    let quickLogOrder = $state(0);

    // Quantity
    let quantityEnabled = $state(false);
    let quantityDefault = $state<number | undefined>(undefined);
    let quantityStep = $state(1);

    // Goals linking
    interface TemplateGoalLinkState {
        goalId: string;
        autoLink: boolean;
        quantityMultiplier: number;
    }
    let goalLinks = $state<TemplateGoalLinkState[]>([]);

    // Load available goals for linking
    const goalsQuery = createQuery({
        queryKey: ["goals", "active"],
        queryFn: () => getGoals({ status: "active", limit: 100 }),
        enabled: open, // Only fetch when modal is open
    });

    const availableGoals = $derived($goalsQuery.data?.items || []);

    // Load categories
    const categoriesQuery = createQuery({
        queryKey: ["categories"],
        queryFn: () => getCategories({ limit: 100 }),
        enabled: open,
    });

    const categories = $derived($categoriesQuery.data?.items || []);

    // UI state
    let showAdvanced = $state(false);
    let errors = $state<Record<string, string>>({});

    // Initialize form when template changes
    $effect(() => {
        if (open) {
            if (template) {
                title = template.title;
                description = template.description || "";
                templateIcon = template.icon || "";
                defaultDuration = template.default_duration;
                defaultCategoryId = template.category?.id;
                defaultEmotionId = template.default_emotion_id;
                expectedQuadrant = template.expected_quadrant;

                isQuickLog = template.is_quick_log;
                quickLogOrder = template.quick_log_order || 0;

                quantityEnabled = template.quantity_enabled;
                quantityDefault = template.quantity_default;
                quantityStep = template.quantity_step || 1;

                // Map existing goals to link state
                // Note: The TaskTemplate struct has `goals: Goal[]` but we need to know the link properties (auto_link, multiplier)
                // If the backend doesn't return link details in `goals`, we might default them for now or need a richer return type.
                // Assuming `goals` in Template just lists them, we can just list them with default link props.
                // Ideally backend returns `template_goals` relation info. For now, we'll just prepopulate IDs.
                if (template.goals) {
                    goalLinks = template.goals.map((g) => ({
                        goalId: g.id,
                        autoLink: true, // Default to true if we don't have this info
                        quantityMultiplier: 1.0,
                    }));
                } else {
                    goalLinks = [];
                }
            } else {
                resetForm();
            }
        }
    });

    function resetForm() {
        title = "";
        description = "";
        templateIcon = "";
        defaultDuration = undefined;
        defaultCategoryId = undefined;
        defaultEmotionId = undefined;
        expectedQuadrant = undefined;

        isQuickLog = false;
        quickLogOrder = 0;

        quantityEnabled = false;
        quantityDefault = undefined;
        quantityStep = 1;

        goalLinks = [];
        showAdvanced = false;
        errors = {};
    }

    // Create mutation
    const createMut = createMutation({
        mutationFn: (data: CreateTemplateRequest) => createTemplate(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["templates"] });
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
            data: Partial<CreateTemplateRequest>;
        }) => updateTemplate(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["templates"] });
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
        errors = newErrors;
        return Object.keys(newErrors).length === 0;
    }

    function handleSubmit() {
        if (!validateForm()) return;

        const data: CreateTemplateRequest = {
            title: title.trim(),
            description: description.trim() || undefined,
            icon: templateIcon || undefined,

            default_duration: defaultDuration,
            expected_quadrant: expectedQuadrant,
            default_emotion_id: defaultEmotionId,

            category_id: defaultCategoryId,

            is_quick_log: isQuickLog,
            quick_log_order: isQuickLog ? quickLogOrder : undefined,

            quantity_enabled: quantityEnabled,
            quantity_default: quantityEnabled ? quantityDefault : undefined,
            quantity_step: quantityEnabled ? quantityStep : undefined,

            goal_links: goalLinks.map((l) => ({
                goal_id: l.goalId,
                auto_link_tasks: l.autoLink,
                quantity_multiplier: l.quantityMultiplier,
            })),
        };

        if (isEditing && template) {
            $updateMut.mutate({ id: template.id, data });
        } else {
            $createMut.mutate(data);
        }
    }

    const isPending = $derived($createMut.isPending || $updateMut.isPending);

    const quadrants = [
        { value: "green", label: "🟢 Green (Calm+)", color: "#22c55e" },
        { value: "yellow", label: "🟡 Yellow (Active+)", color: "#eab308" },
        { value: "red", label: "🔴 Red (Active-)", color: "#ef4444" },
        { value: "blue", label: "🔵 Blue (Calm-)", color: "#3b82f6" },
    ] as const;

    function formatDuration(seconds: number | undefined): string {
        if (!seconds) return "";
        const mins = Math.floor(seconds / 60);
        if (mins < 60) return `${mins}m`;
        const hrs = Math.floor(mins / 60);
        const remMins = mins % 60;
        return remMins > 0 ? `${hrs}h ${remMins}m` : `${hrs}h`;
    }

    function parseDuration(str: string): number | undefined {
        if (!str) return undefined;
        const match = str.match(
            /^(\d+)\s*(h|hr|hour|m|min|minute)?s?\s*(\d+)?\s*(m|min|minute)?s?$/i,
        );
        if (!match) {
            const num = parseInt(str);
            return isNaN(num) ? undefined : num * 60; // Assume minutes
        }
        let total = 0;
        if (match[2]?.toLowerCase().startsWith("h")) {
            total += parseInt(match[1]) * 3600;
            if (match[3]) {
                total += parseInt(match[3]) * 60;
            }
        } else {
            total += parseInt(match[1]) * 60;
        }
        return total;
    }

    let durationInput = $state("");
    $effect(() => {
        if (defaultDuration) {
            durationInput = formatDuration(defaultDuration);
        }
    });

    function addGoalLink(goalId: string) {
        if (!goalLinks.some((l) => l.goalId === goalId)) {
            goalLinks = [
                ...goalLinks,
                { goalId, autoLink: true, quantityMultiplier: 1.0 },
            ];
        }
    }

    function removeGoalLink(goalId: string) {
        goalLinks = goalLinks.filter((l) => l.goalId !== goalId);
    }
</script>

<Modal
    bind:open
    size="lg"
    title={isEditing ? "Edit Template" : "Create Template"}
    onClose={handleClose}
>
    {#snippet icon()}
        <Zap class="w-5 h-5 text-secondary" />
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
            <div class="flex flex-col gap-1">
                <label class="text-xs font-semibold uppercase opacity-50"
                    >Icon</label
                >
                <button
                    type="button"
                    class="w-12 h-12 rounded-xl border-2 border-dashed border-base-300 hover:border-secondary/50 flex items-center justify-center text-2xl transition-colors"
                    onclick={() => {
                        const emojis = [
                            "⚡",
                            "🏃",
                            "💪",
                            "📚",
                            "🧘",
                            "💤",
                            "🍎",
                            "💧",
                            "✍️",
                            "🎨",
                            "☕",
                            "🎯",
                        ];
                        templateIcon =
                            emojis[Math.floor(Math.random() * emojis.length)];
                    }}
                >
                    {templateIcon || "⚡"}
                </button>
            </div>
            <div class="flex-1">
                <label
                    class="text-xs font-semibold uppercase opacity-50"
                    for="template-title">Title *</label
                >
                <input
                    id="template-title"
                    type="text"
                    class="input input-bordered w-full mt-1"
                    class:input-error={errors.title}
                    placeholder="e.g., Morning Run"
                    bind:value={title}
                />
                {#if errors.title}
                    <p class="text-error text-xs mt-1">{errors.title}</p>
                {/if}
            </div>
        </div>

        <!-- Description -->
        <div>
            <label
                class="text-xs font-semibold uppercase opacity-50"
                for="template-desc">Description</label
            >
            <textarea
                id="template-desc"
                class="textarea textarea-bordered w-full mt-1 h-16"
                placeholder="Brief description..."
                bind:value={description}
            ></textarea>
        </div>

        <!-- Default Settings -->
        <div class="grid grid-cols-2 gap-4">
            <div>
                <label
                    class="text-xs font-semibold uppercase opacity-50 flex items-center gap-1"
                    for="default-duration"
                >
                    <Timer class="w-3 h-3" />
                    Default Duration
                </label>
                <input
                    id="default-duration"
                    type="text"
                    class="input input-bordered input-sm w-full mt-1"
                    placeholder="e.g., 30m, 1h"
                    value={durationInput}
                    onblur={(e) => {
                        defaultDuration = parseDuration(e.currentTarget.value);
                        durationInput = formatDuration(defaultDuration);
                    }}
                />
            </div>
            <CategoryDropdown
                {categories}
                bind:value={defaultCategoryId}
                size="sm"
                label="Category"
            />
        </div>

        <!-- Emotion -->
        <div>
            <label
                class="text-xs font-semibold uppercase opacity-50 flex items-center gap-1"
            >
                <Heart class="w-3 h-3" />
                Expected Quadrant
            </label>
            <div class="grid grid-cols-4 gap-2 mt-2">
                {#each quadrants as q}
                    <button
                        type="button"
                        class={cn(
                            "p-2 rounded-lg border-2 text-xs font-medium transition-all",
                            expectedQuadrant === q.value
                                ? "border-base-content/40"
                                : "border-base-200 hover:border-base-300",
                        )}
                        style={expectedQuadrant === q.value
                            ? `background-color: ${q.color}20`
                            : ""}
                        onclick={() => {
                            expectedQuadrant =
                                expectedQuadrant === q.value
                                    ? undefined
                                    : q.value;
                        }}
                    >
                        {q.label}
                    </button>
                {/each}
            </div>
        </div>

        <!-- Quick Log Toggle -->
        <div
            class="p-4 rounded-xl bg-secondary/10 border border-secondary/30 space-y-3"
        >
            <label class="flex items-center gap-3 cursor-pointer">
                <input
                    type="checkbox"
                    class="toggle toggle-secondary"
                    bind:checked={isQuickLog}
                />
                <div class="flex items-center gap-2">
                    <Zap class="w-4 h-4 text-secondary" />
                    <span class="font-semibold text-sm">Quick Log Template</span
                    >
                </div>
            </label>
            {#if isQuickLog}
                <div>
                    <label
                        class="text-xs font-semibold uppercase opacity-50"
                        for="quick-log-order">Display Order</label
                    >
                    <input
                        id="quick-log-order"
                        type="number"
                        min="0"
                        class="input input-bordered input-sm w-24 mt-1"
                        bind:value={quickLogOrder}
                    />
                </div>
            {/if}
        </div>

        <!-- Quantity Settings -->
        <div
            class="p-4 rounded-xl bg-base-200/50 border border-base-200 space-y-3"
        >
            <label class="flex items-center gap-3 cursor-pointer">
                <input
                    type="checkbox"
                    class="toggle toggle-primary"
                    bind:checked={quantityEnabled}
                />
                <div class="flex items-center gap-2">
                    <Hash class="w-4 h-4 text-primary" />
                    <span class="font-semibold text-sm">Track Quantity</span>
                </div>
            </label>

            {#if quantityEnabled}
                <div
                    class="grid grid-cols-2 gap-3 pt-2 border-t border-base-200"
                >
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            for="qty-default">Default</label
                        >
                        <input
                            id="qty-default"
                            type="number"
                            step="0.1"
                            min="0"
                            class="input input-bordered input-sm w-full mt-1"
                            bind:value={quantityDefault}
                        />
                    </div>
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            for="qty-step">Step</label
                        >
                        <input
                            id="qty-step"
                            type="number"
                            step="0.1"
                            min="0.1"
                            class="input input-bordered input-sm w-full mt-1"
                            bind:value={quantityStep}
                        />
                    </div>
                </div>
            {/if}
        </div>

        <!-- Goal Linking -->
        <div class="space-y-2">
            <div class="flex items-center justify-between">
                <label
                    class="text-xs font-semibold uppercase opacity-50 flex items-center gap-1"
                >
                    <Target class="w-3 h-3" />
                    Linked Goals
                </label>

                <div class="dropdown dropdown-end">
                    <button
                        type="button"
                        tabindex="0"
                        class="btn btn-ghost btn-xs gap-1"
                    >
                        <Plus class="w-3 h-3" /> Add
                    </button>
                    <ul
                        class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52 max-h-60 overflow-y-auto"
                    >
                        {#each availableGoals.filter((g) => !goalLinks.some((l) => l.goalId === g.id)) as goal}
                            <li>
                                <button
                                    type="button"
                                    onclick={() => addGoalLink(goal.id)}
                                >
                                    {goal.icon}
                                    {goal.title}
                                </button>
                            </li>
                        {/each}
                        {#if availableGoals.filter((g) => !goalLinks.some((l) => l.goalId === g.id)).length === 0}
                            <li class="p-2 text-xs opacity-50 text-center">
                                No more active goals
                            </li>
                        {/if}
                    </ul>
                </div>
            </div>

            {#if goalLinks.length > 0}
                <div class="space-y-2">
                    {#each goalLinks as link (link.goalId)}
                        {@const goal = availableGoals.find(
                            (g) => g.id === link.goalId,
                        )}
                        {#if goal}
                            <div
                                class="flex items-center gap-3 p-3 rounded-lg border border-base-200 bg-base-100"
                            >
                                <div
                                    class="w-8 h-8 rounded bg-base-200 flex items-center justify-center text-lg"
                                >
                                    {goal.icon}
                                </div>
                                <div class="flex-1">
                                    <div class="font-medium text-sm">
                                        {goal.title}
                                    </div>
                                    <div class="flex items-center gap-2 mt-1">
                                        <label
                                            class="label cursor-pointer justify-start gap-2 p-0"
                                        >
                                            <input
                                                type="checkbox"
                                                class="checkbox checkbox-xs"
                                                bind:checked={link.autoLink}
                                            />
                                            <span class="label-text text-xs"
                                                >Auto-link</span
                                            >
                                        </label>
                                    </div>
                                </div>
                                <button
                                    type="button"
                                    class="btn btn-ghost btn-xs btn-square text-error"
                                    onclick={() => removeGoalLink(link.goalId)}
                                >
                                    <X class="w-3 h-3" />
                                </button>
                            </div>
                        {/if}
                    {/each}
                </div>
            {:else}
                <div
                    class="p-4 text-center rounded-lg border border-dashed border-base-200 text-xs opacity-50"
                >
                    No goals linked
                </div>
            {/if}
        </div>
    </form>

    {#snippet actions()}
        <button type="button" class="btn btn-ghost" onclick={handleClose}
            >Cancel</button
        >
        <button
            type="button"
            class="btn btn-secondary gap-2"
            disabled={isPending}
            onclick={handleSubmit}
        >
            {#if isPending}
                <span class="loading loading-spinner loading-sm"></span>
            {:else}
                <Save class="w-4 h-4" />
            {/if}
            {isEditing ? "Update Template" : "Create Template"}
        </button>
    {/snippet}
</Modal>
