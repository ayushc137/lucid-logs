<script lang="ts">
    import { createMutation, useQueryClient } from "@tanstack/svelte-query";
    import {
        createTemplate,
        updateTemplate,
        type TaskTemplate,
        type CreateTemplateRequest,
    } from "$lib/api";
    import { Modal, ColorPicker, CategoryDropdown } from "$lib/components/ui";
    import {
        Zap,
        Save,
        Clock,
        Flag,
        AlertCircle,
        Plus,
        Heart,
        FileText,
        Timer,
        Hash,
        Sparkles,
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
    let icon = $state("");
    let color = $state("#10B981");

    // Defaults
    let defaultDuration = $state<number | undefined>(undefined);
    let defaultPriority = $state(2);
    let defaultCategoryId = $state<string | undefined>(undefined);

    // Quick log
    let isQuickLog = $state(false);
    let quickLogOrder = $state(0);

    // Quantity
    let quantityEnabled = $state(false);
    let quantityDefault = $state<number | undefined>(undefined);
    let quantityUnit = $state("");
    let quantityStep = $state(1);

    // Emotion
    let expectedQuadrant = $state<
        "green" | "yellow" | "red" | "blue" | undefined
    >(undefined);

    // Show fields
    let showJournal = $state(true);
    let showDuration = $state(true);
    let showQuantity = $state(false);
    let showEmotion = $state(true);
    let showPositivesNegatives = $state(false);
    let showNotes = $state(true);

    // Activity key
    let activityKey = $state("");

    // UI state
    let showAdvanced = $state(false);
    let errors = $state<Record<string, string>>({});

    // Initialize form when template changes
    $effect(() => {
        if (open) {
            if (template) {
                title = template.title;
                description = template.description || "";
                icon = template.icon || "";
                color = template.color || "#10B981";
                defaultDuration = template.default_duration;
                defaultPriority = template.default_priority || 2;
                defaultCategoryId = template.default_category?.id;
                isQuickLog = template.is_quick_log;
                quickLogOrder = template.quick_log_order || 0;
                quantityEnabled = template.quantity_enabled;
                quantityDefault = template.quantity_default;
                quantityUnit = template.quantity_unit || "";
                quantityStep = template.quantity_step || 1;
                expectedQuadrant = template.expected_quadrant;
                activityKey = template.activity_key || "";

                if (template.show_fields) {
                    showJournal = template.show_fields.journal ?? true;
                    showDuration = template.show_fields.duration ?? true;
                    showQuantity = template.show_fields.quantity ?? false;
                    showEmotion = template.show_fields.emotion ?? true;
                    showPositivesNegatives =
                        template.show_fields.positives_negatives ?? false;
                    showNotes = template.show_fields.notes ?? true;
                }
            } else {
                resetForm();
            }
        }
    });

    function resetForm() {
        title = "";
        description = "";
        icon = "";
        color = "#10B981";
        defaultDuration = undefined;
        defaultPriority = 2;
        defaultCategoryId = undefined;
        isQuickLog = false;
        quickLogOrder = 0;
        quantityEnabled = false;
        quantityDefault = undefined;
        quantityUnit = "";
        quantityStep = 1;
        expectedQuadrant = undefined;
        activityKey = "";
        showJournal = true;
        showDuration = true;
        showQuantity = false;
        showEmotion = true;
        showPositivesNegatives = false;
        showNotes = true;
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

        if (quantityEnabled && !quantityUnit.trim()) {
            newErrors.quantityUnit =
                "Unit is required when quantity is enabled";
        }

        errors = newErrors;
        return Object.keys(newErrors).length === 0;
    }

    function handleSubmit() {
        if (!validateForm()) return;

        const data: CreateTemplateRequest = {
            title: title.trim(),
            description: description.trim() || undefined,
            icon: icon || undefined,
            color: color || undefined,
            default_duration: defaultDuration,
            default_priority: defaultPriority,
            default_category_id: defaultCategoryId,
            is_quick_log: isQuickLog,
            quick_log_order: isQuickLog ? quickLogOrder : undefined,
            quantity_enabled: quantityEnabled,
            quantity_default: quantityEnabled ? quantityDefault : undefined,
            quantity_unit: quantityEnabled ? quantityUnit : undefined,
            quantity_step: quantityEnabled ? quantityStep : undefined,
            expected_quadrant: expectedQuadrant,
            activity_key: activityKey || undefined,
            show_fields: {
                journal: showJournal,
                duration: showDuration,
                quantity: showQuantity,
                emotion: showEmotion,
                positives_negatives: showPositivesNegatives,
                notes: showNotes,
            },
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
        // Parse formats like "30", "30m", "1h", "1h 30m"
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
            <!-- Icon Picker -->
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
                        icon =
                            emojis[Math.floor(Math.random() * emojis.length)];
                    }}
                >
                    {icon || "⚡"}
                </button>
            </div>

            <!-- Title -->
            <div class="flex-1">
                <label
                    class="text-xs font-semibold uppercase opacity-50"
                    for="template-title"
                >
                    Title *
                </label>
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
                for="template-desc"
            >
                Description
            </label>
            <textarea
                id="template-desc"
                class="textarea textarea-bordered w-full mt-1 h-16"
                placeholder="Brief description of this template..."
                bind:value={description}
            ></textarea>
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
                <p class="text-xs opacity-60">
                    Quick log templates appear in the dashboard for one-tap
                    logging.
                </p>
                <div>
                    <label class="text-xs font-semibold uppercase opacity-50"
                        >Display Order</label
                    >
                    <input
                        type="number"
                        min="0"
                        class="input input-bordered input-sm w-24 mt-1"
                        bind:value={quickLogOrder}
                    />
                </div>
            {/if}
        </div>

        <!-- Default Settings -->
        <div class="grid grid-cols-2 gap-4">
            <!-- Duration -->
            <div>
                <label
                    class="text-xs font-semibold uppercase opacity-50 flex items-center gap-1"
                >
                    <Timer class="w-3 h-3" />
                    Default Duration
                </label>
                <input
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

            <!-- Priority -->
            <div>
                <label
                    class="text-xs font-semibold uppercase opacity-50 flex items-center gap-1"
                >
                    <Flag class="w-3 h-3" />
                    Default Priority
                </label>
                <select
                    class="select select-bordered select-sm w-full mt-1"
                    bind:value={defaultPriority}
                >
                    <option value={1}>Low</option>
                    <option value={2}>Medium</option>
                    <option value={3}>High</option>
                </select>
            </div>
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
                    class="grid grid-cols-3 gap-3 pt-2 border-t border-base-200"
                >
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Default</label
                        >
                        <input
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
                            >Unit *</label
                        >
                        <input
                            type="text"
                            class="input input-bordered input-sm w-full mt-1"
                            class:input-error={errors.quantityUnit}
                            placeholder="km, L, pages"
                            bind:value={quantityUnit}
                        />
                    </div>
                    <div>
                        <label
                            class="text-xs font-semibold uppercase opacity-50"
                            >Step</label
                        >
                        <input
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

        <!-- Expected Quadrant -->
        <div>
            <label
                class="text-xs font-semibold uppercase opacity-50 flex items-center gap-1"
            >
                <Heart class="w-3 h-3" />
                Expected Emotion Quadrant
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
                <!-- Category & Color -->
                <div class="grid grid-cols-2 gap-3">
                    <CategoryDropdown
                        bind:selectedId={defaultCategoryId}
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

                <!-- Activity Key -->
                <div>
                    <label class="text-xs font-semibold uppercase opacity-50"
                        >Activity Key</label
                    >
                    <input
                        type="text"
                        class="input input-bordered input-sm w-full mt-1"
                        placeholder="For auto-linking to goals"
                        bind:value={activityKey}
                    />
                    <p class="text-[10px] opacity-50 mt-1">
                        Tasks created from this template will auto-link to goals
                        with matching activity key.
                    </p>
                </div>

                <!-- Show Fields -->
                <div>
                    <label class="text-xs font-semibold uppercase opacity-50"
                        >Fields to Show in Quick Log</label
                    >
                    <div class="grid grid-cols-2 gap-2 mt-2">
                        <label
                            class="label cursor-pointer justify-start gap-2 py-1"
                        >
                            <input
                                type="checkbox"
                                class="checkbox checkbox-xs"
                                bind:checked={showJournal}
                            />
                            <span class="label-text text-sm">Journal</span>
                        </label>
                        <label
                            class="label cursor-pointer justify-start gap-2 py-1"
                        >
                            <input
                                type="checkbox"
                                class="checkbox checkbox-xs"
                                bind:checked={showDuration}
                            />
                            <span class="label-text text-sm">Duration</span>
                        </label>
                        <label
                            class="label cursor-pointer justify-start gap-2 py-1"
                        >
                            <input
                                type="checkbox"
                                class="checkbox checkbox-xs"
                                bind:checked={showQuantity}
                            />
                            <span class="label-text text-sm">Quantity</span>
                        </label>
                        <label
                            class="label cursor-pointer justify-start gap-2 py-1"
                        >
                            <input
                                type="checkbox"
                                class="checkbox checkbox-xs"
                                bind:checked={showEmotion}
                            />
                            <span class="label-text text-sm">Emotion</span>
                        </label>
                        <label
                            class="label cursor-pointer justify-start gap-2 py-1"
                        >
                            <input
                                type="checkbox"
                                class="checkbox checkbox-xs"
                                bind:checked={showPositivesNegatives}
                            />
                            <span class="label-text text-sm">Reflections</span>
                        </label>
                        <label
                            class="label cursor-pointer justify-start gap-2 py-1"
                        >
                            <input
                                type="checkbox"
                                class="checkbox checkbox-xs"
                                bind:checked={showNotes}
                            />
                            <span class="label-text text-sm">Notes</span>
                        </label>
                    </div>
                </div>
            </div>
        {/if}
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
