<script lang="ts">
    import { Repeat, Target, Check, X } from "lucide-svelte";
    import { cn } from "$lib/utils";

    interface Props {
        isHabit: boolean;
        isMeasurable: boolean;
        onToggleHabit?: (value: boolean) => void;
        onToggleMeasurable?: (value: boolean) => void;
    }

    let {
        isHabit = $bindable(),
        isMeasurable = $bindable(),
        onToggleHabit,
        onToggleMeasurable,
    }: Props = $props();

    function toggleHabit() {
        isHabit = !isHabit;
        onToggleHabit?.(isHabit);
    }

    function toggleMeasurable() {
        isMeasurable = !isMeasurable;
        onToggleMeasurable?.(isMeasurable);
    }
</script>

<div class="grid grid-cols-2 gap-3">
    <!-- Recurring Habit Toggle -->
    <button
        type="button"
        class={cn(
            "group relative rounded-xl border-2 p-4 transition-all duration-300 text-left",
            isHabit
                ? "border-primary bg-gradient-to-br from-primary/15 via-primary/10 to-transparent shadow-sm"
                : "border-base-300 bg-base-100 hover:border-primary/30 hover:bg-primary/5",
        )}
        onclick={toggleHabit}
    >
        <div class="flex items-start gap-3">
            <div
                class={cn(
                    "w-10 h-10 rounded-lg flex items-center justify-center transition-all duration-300",
                    isHabit
                        ? "bg-primary text-primary-content shadow-sm"
                        : "bg-base-200 text-base-content/50 group-hover:bg-primary/20 group-hover:text-primary",
                )}
            >
                <Repeat class="w-5 h-5" />
            </div>
            <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                    <span class="text-sm font-semibold">Recurring Habit</span>
                    {#if isHabit}
                        <span class="badge badge-primary badge-xs gap-0.5">
                            <Check class="w-2.5 h-2.5" />
                        </span>
                    {/if}
                </div>
                <p class="text-xs text-base-content/60 mt-0.5">
                    Track daily, weekly, or monthly habits
                </p>
            </div>
        </div>

        <!-- Decorative corner -->
        {#if isHabit}
            <div
                class="absolute top-0 right-0 w-6 h-6 overflow-hidden rounded-tr-xl"
            >
                <div
                    class="absolute -top-3 -right-3 w-6 h-6 bg-primary rotate-45"
                ></div>
            </div>
        {/if}
    </button>

    <!-- Measurable Target Toggle -->
    <button
        type="button"
        class={cn(
            "group relative rounded-xl border-2 p-4 transition-all duration-300 text-left",
            isMeasurable
                ? "border-secondary bg-gradient-to-br from-secondary/15 via-secondary/10 to-transparent shadow-sm"
                : "border-base-300 bg-base-100 hover:border-secondary/30 hover:bg-secondary/5",
        )}
        onclick={toggleMeasurable}
    >
        <div class="flex items-start gap-3">
            <div
                class={cn(
                    "w-10 h-10 rounded-lg flex items-center justify-center transition-all duration-300",
                    isMeasurable
                        ? "bg-secondary text-secondary-content shadow-sm"
                        : "bg-base-200 text-base-content/50 group-hover:bg-secondary/20 group-hover:text-secondary",
                )}
            >
                <Target class="w-5 h-5" />
            </div>
            <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                    <span class="text-sm font-semibold">Measurable Target</span>
                    {#if isMeasurable}
                        <span class="badge badge-secondary badge-xs gap-0.5">
                            <Check class="w-2.5 h-2.5" />
                        </span>
                    {/if}
                </div>
                <p class="text-xs text-base-content/60 mt-0.5">
                    Set a specific numeric goal to hit
                </p>
            </div>
        </div>

        <!-- Decorative corner -->
        {#if isMeasurable}
            <div
                class="absolute top-0 right-0 w-6 h-6 overflow-hidden rounded-tr-xl"
            >
                <div
                    class="absolute -top-3 -right-3 w-6 h-6 bg-secondary rotate-45"
                ></div>
            </div>
        {/if}
    </button>
</div>

<!-- Info text -->
<p class="text-xs text-base-content/40 text-center mt-2">
    {#if isHabit && isMeasurable}
        Tracking a recurring habit with a measurable target per period
    {:else if isHabit}
        Tracking a recurring habit without a specific target
    {:else if isMeasurable}
        One-time measurable goal with a specific target
    {:else}
        Simple goal - select options above to configure
    {/if}
</p>
