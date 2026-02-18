<script lang="ts">
    import type { Unit } from "$lib/api";
    import { UnitDropdown } from "$lib/components/ui";
    import { cn } from "$lib/utils";
    import { Check, Target } from "lucide-svelte";

    interface Props {
        isMeasurable: boolean;
        targetOperator: "gte" | "lte" | "eq";
        targetValue: number;
        targetUnit: string;
        isHabit: boolean;
        period: "day" | "week" | "month";
        units: Unit[];
        selectedUnit: Unit | undefined;
        showCreateUnit: boolean;
        onIsMeasurableChange: (value: boolean) => void;
        onTargetOperatorChange: (value: "gte" | "lte" | "eq") => void;
        onTargetValueChange: (value: number) => void;
        onTargetUnitChange: (value: string) => void;
        onShowCreateUnit: () => void;
        onCreateUnit?: (data: {
            name: string;
            symbol: string;
            type: string;
        }) => void;
        onCancelCreateUnit?: () => void;
        isCreatingUnit?: boolean;
    }

    let {
        isMeasurable = $bindable(false),
        targetOperator = $bindable("gte"),
        targetValue = $bindable(10),
        targetUnit = $bindable(""),
        isHabit = false,
        period = "day",
        units = [],
        selectedUnit,
        showCreateUnit = $bindable(false),
        onIsMeasurableChange,
        onTargetOperatorChange,
        onTargetValueChange,
        onTargetUnitChange,
        onShowCreateUnit,
        onCreateUnit,
        onCancelCreateUnit,
        isCreatingUnit = false,
    }: Props = $props();

    let newUnitName = $state("");
    let newUnitSymbol = $state("");
    let newUnitType = $state<
        "count" | "time" | "distance" | "volume" | "custom"
    >("count");

    function handleCreateUnit() {
        if (!newUnitName.trim() || !onCreateUnit) return;
        onCreateUnit({
            name: newUnitName.trim(),
            symbol: newUnitSymbol.trim() || newUnitName.trim().substring(0, 3),
            type: newUnitType,
        });
        newUnitName = "";
        newUnitSymbol = "";
        newUnitType = "count";
    }
</script>

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
        onclick={() => onIsMeasurableChange(!isMeasurable)}
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
            <span class="text-sm font-semibold">Measurable Target</span>
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
        <div class="px-3 pb-3 pt-3 border-t border-secondary/20 space-y-3">
            <!-- Target Settings Row -->
            <div class="flex items-start gap-2">
                <select
                    bind:value={targetOperator}
                    onchange={(e) =>
                        onTargetOperatorChange(
                            e.currentTarget.value as "gte" | "lte" | "eq",
                        )}
                    class="select select-sm select-bordered"
                >
                    <option value="gte">At least (≥)</option>
                    <option value="eq">Exactly (=)</option>
                    <option value="lte">At most (≤)</option>
                </select>
                <input
                    type="number"
                    min="0"
                    step="1"
                    bind:value={targetValue}
                    oninput={(e) =>
                        onTargetValueChange(
                            parseFloat(e.currentTarget.value) || 0,
                        )}
                    class="input input-sm input-bordered w-20 text-center"
                />
                <div class="flex-1">
                    {#if showCreateUnit}
                        <!-- Unit Creation Form - Unit name replaces dropdown -->
                        <div class="space-y-2">
                            <input
                                type="text"
                                bind:value={newUnitName}
                                placeholder="Unit name (e.g., hours)"
                                class={cn(
                                    "input input-sm input-bordered w-full",
                                    newUnitName.length > 30 && "input-error",
                                )}
                                maxlength={30}
                            />
                            <div class="flex gap-2">
                                <input
                                    type="text"
                                    bind:value={newUnitSymbol}
                                    placeholder="Symbol"
                                    class="input input-sm input-bordered w-20"
                                    maxlength={10}
                                />
                                <select
                                    bind:value={newUnitType}
                                    class="select select-sm select-bordered flex-1"
                                >
                                    <option value="count">Count</option>
                                    <option value="time">Time</option>
                                    <option value="distance">Distance</option>
                                    <option value="volume">Volume</option>
                                    <option value="custom">Custom</option>
                                </select>
                            </div>
                            <div class="flex gap-2">
                                <button
                                    type="button"
                                    class="btn btn-sm btn-primary flex-1 gap-1"
                                    onclick={handleCreateUnit}
                                    disabled={isCreatingUnit ||
                                        !newUnitName.trim() ||
                                        !newUnitSymbol.trim()}
                                >
                                    {#if isCreatingUnit}
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
                                    onclick={onCancelCreateUnit}
                                >
                                    Cancel
                                </button>
                            </div>
                        </div>
                    {:else}
                        <UnitDropdown
                            {units}
                            bind:value={targetUnit}
                            size="sm"
                            showCreateButton={true}
                            onCreate={onShowCreateUnit}
                        />
                    {/if}
                </div>
            </div>

            <!-- Target Summary -->
            <div
                class="pt-3 mt-3 border-t border-secondary/20 flex items-center gap-2"
            >
                <div
                    class="w-5 h-5 rounded-md bg-secondary/10 flex items-center justify-center"
                >
                    <Target class="w-3 h-3 text-secondary" />
                </div>
                <p class="text-xs text-base-content/60">
                    Target: <span class="font-semibold text-secondary">
                        {targetOperator === "gte"
                            ? "at least"
                            : targetOperator === "lte"
                              ? "at most"
                              : "exactly"}
                        {targetValue || 0}{selectedUnit
                            ? ` ${selectedUnit.symbol}`
                            : ""}
                    </span>
                    {#if isHabit}
                        <span class="opacity-70">per {period}</span>
                    {/if}
                </p>
            </div>
        </div>
    {/if}
</div>
