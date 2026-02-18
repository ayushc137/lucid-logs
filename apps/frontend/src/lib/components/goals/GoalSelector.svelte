<script lang="ts">
    import { type Goal, getGoals, getUnits } from "$lib/api";
    import { cn } from "$lib/utils";
    import { createQuery } from "@tanstack/svelte-query";
    import {
        Flame,
        Plus,
        Target,
        TrendingDown,
        TrendingUp,
        X,
    } from "lucide-svelte";
    import { fly } from "svelte/transition";
    import GoalSearchModal, {
        type GoalLinkConfig,
    } from "./GoalSearchModal.svelte";

    interface GoalLink {
        goal_id: string;
        impact_type: "positive" | "negative";
        quantity_value?: number;
        quantity_unit?: string;
    }

    interface Props {
        value: GoalLink[];
        onChange: (links: GoalLink[]) => void;
        /** Called when goals are added, with full goal objects for inheritance */
        onGoalsAdded?: (goals: Goal[]) => void;
        disabled?: boolean;
    }

    let { value = [], onChange, onGoalsAdded, disabled = false }: Props = $props();

    let modalOpen = $state(false);

    // Fetch goals for display
    const goalsQuery = createQuery({
        queryKey: ["goals", "active"],
        queryFn: () => getGoals({ status: "active", limit: 100 }),
    });

    // Fetch units for symbol display
    const unitsQuery = createQuery({
        queryKey: ["units"],
        queryFn: () => getUnits(),
        staleTime: 1000 * 60 * 5,
    });

    const goals = $derived($goalsQuery.data?.items || []);
    const units = $derived($unitsQuery.data?.items || []);

    // Get selected goals with full data
    const selectedGoals = $derived(
        value
            .map((link) => ({
                ...link,
                goal: goals.find((g) => g.id === link.goal_id),
            }))
            .filter((l) => l.goal),
    );

    // Get excluded goal IDs for modal
    const excludedGoalIds = $derived(value.map((v) => v.goal_id));

    function getUnitSymbol(unitId?: string): string {
        if (!unitId) return "";
        const unit = units.find((u) => u.id === unitId);
        return unit?.symbol || unitId.replace("units:", "");
    }

    function handleAddGoals(links: GoalLinkConfig[]) {
        const newLinks = links.map((link) => ({
            goal_id: link.goal_id,
            impact_type: link.impact_type,
            quantity_value: link.quantity_value,
            quantity_unit: link.quantity_unit,
        }));
        onChange([...value, ...newLinks]);

        // Notify parent with full goal objects for category/priority inheritance
        if (onGoalsAdded) {
            const addedGoals = links
                .map((link) => goals.find((g) => g.id === link.goal_id))
                .filter((g): g is Goal => !!g);
            if (addedGoals.length > 0) {
                onGoalsAdded(addedGoals);
            }
        }
    }

    function removeGoal(goalId: string) {
        onChange(value.filter((v) => v.goal_id !== goalId));
    }

    function updateLink(goalId: string, updates: Partial<GoalLink>) {
        onChange(
            value.map((v) => (v.goal_id === goalId ? { ...v, ...updates } : v)),
        );
    }

    const impactTypes = [
        {
            value: "positive" as const,
            label: "Positive",
            icon: TrendingUp,
            color: "text-success",
            bgColor: "bg-success",
        },
        {
            value: "negative" as const,
            label: "Negative",
            icon: TrendingDown,
            color: "text-error",
            bgColor: "bg-error",
        },
    ];
</script>

<div class="space-y-3">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
            <Target class="w-4 h-4 text-accent" />
            <span class="text-sm font-semibold">Linked Goals</span>
            {#if value.length > 0}
                <span
                    class="badge badge-sm badge-accent/20 text-accent font-bold"
                    >{value.length}</span
                >
            {/if}
        </div>
        <button
            type="button"
            class="btn btn-ghost btn-xs gap-1.5 hover:bg-accent/10 hover:text-accent transition-colors"
            onclick={() => (modalOpen = true)}
            {disabled}
        >
            <Plus class="w-3.5 h-3.5" />
            Add Goal
        </button>
    </div>

    <!-- Selected Goals List -->
    {#if selectedGoals.length > 0}
        <div class="space-y-2">
            {#each selectedGoals as link (link.goal_id)}
                {@const goal = link.goal}
                {@const impactConfig = impactTypes.find(
                    (t) => t.value === link.impact_type,
                )}
                {@const ImpactIcon = impactConfig?.icon || TrendingUp}
                {#if goal}
                    <div
                        class="relative p-3 rounded-xl bg-base-200/40 border border-base-200 hover:border-base-300 transition-colors group"
                        in:fly={{ y: -8, duration: 250 }}
                    >
                        <div class="flex items-start gap-3">
                            <!-- Goal Icon with Impact Indicator -->
                            <div class="relative shrink-0">
                                <div
                                    class="w-10 h-10 rounded-xl flex items-center justify-center text-lg"
                                    style="background-color: {goal.category
                                        ?.color || '#6b7280'}20;"
                                >
                                    {goal.icon || "🎯"}
                                </div>
                                <!-- Impact Badge -->
                                <div
                                    class={cn(
                                        "absolute -bottom-1 -right-1 w-5 h-5 rounded-full flex items-center justify-center border-2 border-base-100",
                                        impactConfig?.bgColor ||
                                            "bg-base-content/60",
                                    )}
                                >
                                    <ImpactIcon
                                        class="w-2.5 h-2.5 text-white"
                                    />
                                </div>
                            </div>

                            <!-- Goal Info -->
                            <div class="flex-1 min-w-0 space-y-2">
                                <div class="flex items-center gap-2">
                                    <span class="font-semibold text-sm truncate"
                                        >{goal.title}</span
                                    >
                                    {#if (goal.stats?.current_streak || 0) > 0}
                                        <div
                                            class="flex items-center gap-0.5 text-[10px] text-warning font-bold"
                                        >
                                            <Flame class="w-3 h-3" />
                                            {goal.stats?.current_streak || 0}
                                        </div>
                                    {/if}
                                </div>

                                <!-- Controls Row -->
                                <div class="flex items-center gap-3 flex-wrap">
                                    <!-- Impact Type Selector -->
                                    <div
                                        class="flex items-center gap-1 bg-base-100 rounded-lg p-0.5 border border-base-200"
                                    >
                                        {#each impactTypes as type}
                                            {@const Icon = type.icon}
                                            {@const isSelected =
                                                link.impact_type === type.value}
                                            <button
                                                type="button"
                                                class={cn(
                                                    "flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium transition-all duration-200",
                                                    isSelected
                                                        ? `${type.bgColor}/15 ${type.color}`
                                                        : "hover:bg-base-200 text-base-content/50",
                                                )}
                                                onclick={() =>
                                                    updateLink(link.goal_id, {
                                                        impact_type: type.value,
                                                    })}
                                                title={type.label}
                                            >
                                                <Icon class="w-3 h-3" />
                                                {#if isSelected}
                                                    <span
                                                        class="hidden sm:inline"
                                                        >{type.label}</span
                                                    >
                                                {/if}
                                            </button>
                                        {/each}
                                    </div>

                                    <!-- Quantity Input (for measurable goals) -->
                                    {#if goal.target}
                                        <div
                                            class="flex items-center gap-1.5 bg-base-100 rounded-lg px-2 py-1 border border-base-200"
                                        >
                                            <input
                                                type="number"
                                                step="0.1"
                                                min="0"
                                                class="w-14 bg-transparent text-sm font-medium focus:outline-none text-center [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                                                placeholder="0"
                                                value={link.quantity_value ??
                                                    ""}
                                                onchange={(e) => {
                                                    const val = parseFloat(
                                                        e.currentTarget.value,
                                                    );
                                                    updateLink(link.goal_id, {
                                                        quantity_value: isNaN(
                                                            val,
                                                        )
                                                            ? undefined
                                                            : val,
                                                    });
                                                }}
                                            />
                                            <span
                                                class="text-xs font-semibold text-base-content/50 border-l border-base-200 pl-1.5"
                                            >
                                                {getUnitSymbol(
                                                    goal.target.unit_id,
                                                )}
                                            </span>
                                        </div>
                                    {/if}
                                </div>
                            </div>

                            <!-- Remove Button -->
                            <button
                                type="button"
                                class="btn btn-ghost btn-xs btn-circle opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                                onclick={() => removeGoal(link.goal_id)}
                                title="Remove goal link"
                            >
                                <X class="w-4 h-4" />
                            </button>
                        </div>
                    </div>
                {/if}
            {/each}
        </div>
    {:else}
        <!-- Empty State -->
        <button
            type="button"
            class="w-full p-4 rounded-xl border-2 border-dashed border-base-300 hover:border-accent/50 hover:bg-accent/5 transition-all duration-200 group"
            onclick={() => (modalOpen = true)}
            {disabled}
        >
            <div
                class="flex flex-col items-center gap-2 text-base-content/50 group-hover:text-accent transition-colors"
            >
                <div
                    class="w-10 h-10 rounded-xl bg-base-200 group-hover:bg-accent/10 flex items-center justify-center transition-colors"
                >
                    <Target class="w-5 h-5" />
                </div>
                <span class="text-xs font-medium">Click to link goals</span>
            </div>
        </button>
    {/if}
</div>

<!-- Goal Search Modal -->
<GoalSearchModal
    bind:open={modalOpen}
    excludeGoalIds={excludedGoalIds}
    onAdd={handleAddGoals}
/>
