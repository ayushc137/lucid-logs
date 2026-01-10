<script lang="ts">
    import {
        ChevronDown,
        Search,
        X,
        Plus,
        Check,
        Clock,
        Ruler,
        Droplet,
        Hash,
        Shapes,
    } from "lucide-svelte";
    import { cn } from "$lib/utils";
    import type { Unit } from "$lib/api/units";

    interface Props {
        units: Unit[];
        value: string | undefined;
        onchange?: (value: string | undefined) => void;
        label?: string;
        size?: "xs" | "sm" | "md";
        showSearch?: boolean;
        showNoUnitOption?: boolean;
        noUnitLabel?: string;
        placeholder?: string;
        showCreateButton?: boolean;
        onCreate?: () => void;
        class?: string;
    }

    let {
        units = [],
        value = $bindable(),
        onchange,
        label,
        size = "sm",
        showSearch = true,
        showNoUnitOption = false,
        noUnitLabel = "No unit",
        placeholder = "Select unit...",
        showCreateButton = false,
        onCreate,
        class: className,
    }: Props = $props();

    let isOpen = $state(false);
    let search = $state("");
    let dropdownRef = $state<HTMLDivElement | null>(null);
    let searchInputRef = $state<HTMLInputElement | null>(null);

    const selectedUnit = $derived(units?.find((u) => u.id === value));

    const filteredUnits = $derived(
        search.trim()
            ? (units || []).filter(
                  (u) =>
                      u.name.toLowerCase().includes(search.toLowerCase()) ||
                      u.symbol.toLowerCase().includes(search.toLowerCase()),
              )
            : units || [],
    );

    // Group units by type
    const groupedUnits = $derived.by(() => {
        const groups: Record<string, Unit[]> = {
            time: [],
            distance: [],
            volume: [],
            count: [],
            custom: [],
        };
        for (const unit of filteredUnits) {
            if (groups[unit.type]) {
                groups[unit.type].push(unit);
            } else {
                groups.custom.push(unit);
            }
        }
        return groups;
    });

    // Check if there are any units to display
    const hasUnits = $derived(filteredUnits.length > 0);

    // Type icons and colors
    const typeConfig: Record<
        string,
        { icon: typeof Clock; color: string; label: string }
    > = {
        time: { icon: Clock, color: "text-blue-500", label: "Time" },
        distance: { icon: Ruler, color: "text-green-500", label: "Distance" },
        volume: { icon: Droplet, color: "text-cyan-500", label: "Volume" },
        count: { icon: Hash, color: "text-orange-500", label: "Count" },
        custom: { icon: Shapes, color: "text-purple-500", label: "Custom" },
    };

    function handleSelect(unitId: string | undefined) {
        value = unitId;
        onchange?.(unitId);
        isOpen = false;
        search = "";
    }

    function handleClickOutside(e: MouseEvent) {
        if (dropdownRef && !dropdownRef.contains(e.target as Node)) {
            isOpen = false;
            search = "";
        }
    }

    function handleOpen() {
        isOpen = !isOpen;
        if (isOpen && showSearch) {
            setTimeout(() => searchInputRef?.focus(), 50);
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
        <div class="label py-1">
            <span class="label-text text-xs font-medium opacity-70"
                >{label}</span
            >
        </div>
    {/if}

    <div class="relative" bind:this={dropdownRef}>
        <!-- Trigger Button -->
        <button
            type="button"
            class={cn(
                "flex items-center justify-between w-full gap-2 rounded-lg border border-base-300 bg-base-100 transition-all hover:border-base-content/30",
                sizeClasses[size],
                isOpen && "border-primary ring-1 ring-primary/20",
            )}
            onclick={handleOpen}
        >
            <div class="flex items-center gap-2 truncate">
                {#if selectedUnit}
                    {@const config =
                        typeConfig[selectedUnit.type] || typeConfig.custom}
                    <config.icon
                        class={cn("w-3.5 h-3.5 flex-shrink-0", config.color)}
                    />
                    <span class="font-medium">{selectedUnit.symbol}</span>
                    <span class="opacity-50 truncate"
                        >({selectedUnit.name})</span
                    >
                {:else}
                    <span class="opacity-50 truncate">{placeholder}</span>
                {/if}
            </div>
            <ChevronDown
                class={cn(
                    "w-4 h-4 opacity-50 flex-shrink-0 transition-transform",
                    isOpen && "rotate-180",
                )}
            />
        </button>

        <!-- Dropdown Menu -->
        {#if isOpen}
            <div
                class="absolute z-50 top-full left-0 right-0 mt-1 bg-base-100 border border-base-300 rounded-lg shadow-lg overflow-hidden animate-dropdown min-w-[220px]"
            >
                <!-- Search -->
                {#if showSearch && (units?.length || 0) > 5}
                    <div class="p-2 border-b border-base-200">
                        <div class="relative">
                            <Search
                                class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 opacity-40"
                            />
                            <input
                                bind:this={searchInputRef}
                                type="text"
                                placeholder="Search units..."
                                class="input input-sm input-bordered w-full pl-8 pr-8"
                                bind:value={search}
                                onclick={(e) => e.stopPropagation()}
                            />
                            {#if search}
                                <button
                                    class="absolute right-2 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        search = "";
                                    }}
                                >
                                    <X class="w-3 h-3" />
                                </button>
                            {/if}
                        </div>
                    </div>
                {/if}

                <!-- Options -->
                <div class="max-h-52 overflow-y-auto py-1">
                    <!-- No Unit Option -->
                    {#if showNoUnitOption}
                        <button
                            type="button"
                            class={cn(
                                "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                                value === undefined
                                    ? "bg-primary/10 text-primary"
                                    : "hover:bg-base-200",
                            )}
                            onclick={() => handleSelect(undefined)}
                        >
                            <span
                                class="w-3.5 h-3.5 rounded flex-shrink-0 bg-base-300"
                            ></span>
                            <span class="truncate flex-1 opacity-60"
                                >{noUnitLabel}</span
                            >
                            {#if value === undefined}
                                <Check class="w-3.5 h-3.5 text-primary" />
                            {/if}
                        </button>
                    {/if}

                    <!-- Units by Type -->
                    {#if hasUnits}
                        {#each Object.entries(groupedUnits) as [type, typeUnits]}
                            {#if typeUnits.length > 0}
                                {@const config =
                                    typeConfig[type] || typeConfig.custom}
                                <div
                                    class="px-3 py-1.5 text-[10px] font-bold uppercase tracking-wide flex items-center gap-1.5 bg-base-200/50"
                                >
                                    <config.icon
                                        class={cn("w-3 h-3", config.color)}
                                    />
                                    <span class="opacity-60"
                                        >{config.label}</span
                                    >
                                </div>
                                {#each typeUnits as unit (unit.id)}
                                    <button
                                        type="button"
                                        class={cn(
                                            "w-full px-3 py-2 text-left flex items-center gap-2 transition-colors text-sm",
                                            value === unit.id
                                                ? "bg-primary/10 text-primary"
                                                : "hover:bg-base-200",
                                        )}
                                        onclick={() => handleSelect(unit.id)}
                                    >
                                        <span class="font-medium w-12"
                                            >{unit.symbol}</span
                                        >
                                        <span class="opacity-60 flex-1 truncate"
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
                    {:else}
                        <!-- Empty State -->
                        <div class="px-3 py-4 text-center text-sm opacity-50">
                            {#if search}
                                No matching units
                            {:else}
                                No units available
                            {/if}
                        </div>
                    {/if}
                </div>

                <!-- Create Button -->
                {#if showCreateButton && onCreate}
                    <div class="border-t border-base-200 p-1">
                        <button
                            type="button"
                            class="w-full px-3 py-2 text-left flex items-center gap-2 text-sm text-primary hover:bg-base-200 rounded transition-colors"
                            onclick={() => {
                                isOpen = false;
                                onCreate();
                            }}
                        >
                            <Plus class="w-4 h-4" />
                            Create new unit
                        </button>
                    </div>
                {/if}
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
