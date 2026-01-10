<script lang="ts">
    import {
        createMutation,
        createQuery,
        useQueryClient,
    } from "@tanstack/svelte-query";
    import { getUnits, createUnit, type Unit } from "$lib/api";
    import { ChevronDown, Plus, Check } from "lucide-svelte";
    import { cn } from "$lib/utils";

    interface Props {
        value: string;
        label?: string;
        size?: "xs" | "sm" | "md";
        class?: string;
    }

    let {
        value = $bindable(""),
        label,
        size = "sm",
        class: className,
    }: Props = $props();

    const queryClient = useQueryClient();

    // Fetch units from API
    const unitsQuery = createQuery({
        queryKey: ["units"],
        queryFn: () => getUnits(),
        staleTime: 1000 * 60 * 5, // Cache for 5 minutes
    });

    const units = $derived($unitsQuery.data?.items || []);
    const selectedUnit = $derived(units.find((u) => u.id === value));

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
            value = newUnit.id;
            showCreate = false;
            newName = "";
            newSymbol = "";
            newType = "count";
        },
    });

    let isOpen = $state(false);
    let showCreate = $state(false);
    let newName = $state("");
    let newSymbol = $state("");
    let newType = $state<"count" | "time" | "distance" | "volume" | "custom">(
        "count",
    );
    let dropdownRef = $state<HTMLDivElement | null>(null);

    // Group units by type
    const groupedUnits = $derived.by(() => {
        const groups: Record<string, Unit[]> = {
            time: [],
            distance: [],
            volume: [],
            count: [],
            custom: [],
        };
        for (const unit of units) {
            if (groups[unit.type]) {
                groups[unit.type].push(unit);
            } else {
                groups.custom.push(unit);
            }
        }
        return groups;
    });

    function handleSelect(unitId: string) {
        value = unitId;
        isOpen = false;
    }

    function handleCreate() {
        if (!newName.trim()) return;
        $createUnitMut.mutate({
            name: newName.trim(),
            symbol: newSymbol.trim() || newName.trim().substring(0, 3),
            type: newType,
        });
    }

    function handleClickOutside(e: MouseEvent) {
        if (dropdownRef && !dropdownRef.contains(e.target as Node)) {
            isOpen = false;
        }
    }

    $effect(() => {
        if (isOpen) {
            document.addEventListener("click", handleClickOutside);
            return () =>
                document.removeEventListener("click", handleClickOutside);
        }
    });

    const sizeClasses = {
        xs: "h-6 text-xs px-2",
        sm: "h-8 text-sm px-3",
        md: "h-10 px-3",
    };
</script>

<div class={cn("form-control", className)}>
    {#if label}
        <div class="label py-0.5">
            <span class="label-text text-xs font-medium opacity-70"
                >{label}</span
            >
        </div>
    {/if}

    <div class="relative" bind:this={dropdownRef}>
        <!-- Trigger -->
        <button
            type="button"
            class={cn(
                "flex items-center justify-between w-full gap-2 rounded-lg border border-base-300 bg-base-100 transition-all hover:border-base-content/30",
                sizeClasses[size],
                isOpen && "border-primary ring-1 ring-primary/20",
            )}
            onclick={() => (isOpen = !isOpen)}
        >
            <span class="truncate">
                {#if selectedUnit}
                    <span class="font-medium">{selectedUnit.symbol}</span>
                    <span class="opacity-50 ml-1">({selectedUnit.name})</span>
                {:else}
                    <span class="opacity-50">Select unit...</span>
                {/if}
            </span>
            <ChevronDown
                class={cn(
                    "w-4 h-4 opacity-50 flex-shrink-0 transition-transform",
                    isOpen && "rotate-180",
                )}
            />
        </button>

        <!-- Dropdown -->
        {#if isOpen}
            <div
                class="absolute z-50 top-full left-0 right-0 mt-1 bg-base-100 border border-base-300 rounded-lg shadow-lg overflow-hidden animate-dropdown min-w-[200px]"
            >
                <!-- Create new unit toggle -->
                <button
                    type="button"
                    class="w-full px-3 py-2 text-left flex items-center gap-2 text-sm border-b border-base-200 hover:bg-base-200"
                    onclick={() => (showCreate = !showCreate)}
                >
                    <Plus class="w-4 h-4 text-primary" />
                    <span class="text-primary font-medium">Create new unit</span
                    >
                </button>

                <!-- Create form -->
                {#if showCreate}
                    <div
                        class="p-3 border-b border-base-200 space-y-2 bg-base-200/30"
                    >
                        <div class="flex gap-2">
                            <input
                                type="text"
                                bind:value={newName}
                                placeholder="Name (e.g., hours)"
                                class="input input-bordered input-sm flex-1"
                            />
                            <input
                                type="text"
                                bind:value={newSymbol}
                                placeholder="Symbol"
                                class="input input-bordered input-sm w-16"
                            />
                        </div>
                        <div class="flex gap-2 items-center">
                            <select
                                class="select select-bordered select-sm flex-1"
                                bind:value={newType}
                            >
                                <option value="count">Count</option>
                                <option value="time">Time</option>
                                <option value="distance">Distance</option>
                                <option value="volume">Volume</option>
                                <option value="custom">Custom</option>
                            </select>
                            <button
                                type="button"
                                class="btn btn-sm btn-primary"
                                onclick={handleCreate}
                                disabled={!newName.trim() ||
                                    $createUnitMut.isPending}
                            >
                                {$createUnitMut.isPending ? "..." : "Add"}
                            </button>
                        </div>
                    </div>
                {/if}

                <!-- Unit list by group -->
                <div class="max-h-60 overflow-y-auto py-1">
                    {#each Object.entries(groupedUnits) as [type, typeUnits]}
                        {#if typeUnits.length > 0}
                            <div
                                class="px-3 py-1 text-[10px] font-bold uppercase opacity-40 bg-base-200/50"
                            >
                                {type}
                            </div>
                            {#each typeUnits as unit (unit.id)}
                                <button
                                    type="button"
                                    class={cn(
                                        "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                                        value === unit.id
                                            ? "bg-primary/10"
                                            : "hover:bg-base-200",
                                    )}
                                    onclick={() => handleSelect(unit.id)}
                                >
                                    <span class="font-medium w-10"
                                        >{unit.symbol}</span
                                    >
                                    <span class="opacity-60 flex-1"
                                        >{unit.name}</span
                                    >
                                    {#if value === unit.id}
                                        <Check
                                            class="w-3.5 h-3.5 text-primary"
                                        />
                                    {/if}
                                </button>
                            {/each}
                        {/if}
                    {/each}
                </div>
            </div>
        {/if}
    </div>
</div>

<style>
    @keyframes dropdown-in {
        from {
            opacity: 0;
            transform: translateY(-4px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    .animate-dropdown {
        animation: dropdown-in 0.15s ease-out;
    }
</style>
