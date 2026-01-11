<script lang="ts">
import { cn } from '$lib/utils';
import { Check, Repeat } from 'lucide-svelte';

interface Props {
	isHabit: boolean;
	frequency: number;
	period: 'day' | 'week' | 'month';
	activeDays: number[];
	activeMonthDay: number;
	onIsHabitChange: (value: boolean) => void;
	onFrequencyChange: (value: number) => void;
	onPeriodChange: (value: 'day' | 'week' | 'month') => void;
	onActiveDaysChange: (days: number[]) => void;
	onActiveMonthDayChange: (day: number) => void;
}

let {
	isHabit = $bindable(false),
	frequency = $bindable(1),
	period = $bindable('day'),
	activeDays = $bindable([]),
	activeMonthDay = $bindable(1),
	onIsHabitChange,
	onFrequencyChange,
	onPeriodChange,
	onActiveDaysChange,
	onActiveMonthDayChange,
}: Props = $props();

const dayOptions = [
	{ value: 0, label: 'Sun' },
	{ value: 1, label: 'Mon' },
	{ value: 2, label: 'Tue' },
	{ value: 3, label: 'Wed' },
	{ value: 4, label: 'Thu' },
	{ value: 5, label: 'Fri' },
	{ value: 6, label: 'Sat' },
];

function toggleDay(day: number) {
	if (activeDays.includes(day)) {
		const newDays = activeDays.filter((d) => d !== day);
		onActiveDaysChange(newDays);
	} else {
		const newDays = [...activeDays, day];
		onActiveDaysChange(newDays);
	}
}
</script>

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
        onclick={() => onIsHabitChange(!isHabit)}
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
            <span class="text-sm font-semibold">Recurring Habit</span>
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
        <div class="px-3 pb-3 pt-3 border-t border-primary/20 space-y-3">
            <!-- Recurrence Settings -->
            <div class="flex items-center gap-3">
                <span class="text-xs font-medium opacity-70">Repeat every</span>
                <input
                    type="number"
                    min="1"
                    max="365"
                    bind:value={frequency}
                    oninput={(e) =>
                        onFrequencyChange(parseInt(e.currentTarget.value) || 1)}
                    class="input input-sm input-bordered w-16 text-center"
                />
                <select
                    bind:value={period}
                    onchange={(e) =>
                        onPeriodChange(
                            e.currentTarget.value as "day" | "week" | "month",
                        )}
                    class="select select-sm select-bordered"
                >
                    <option value="day">day(s)</option>
                    <option value="week">week(s)</option>
                    <option value="month">month(s)</option>
                </select>
            </div>

            {#if period === "week"}
                <div class="flex items-center gap-3">
                    <span class="text-xs font-medium opacity-70">Active on</span
                    >
                    <div class="flex gap-1.5 flex-wrap flex-1">
                        {#each dayOptions as day}
                            <button
                                type="button"
                                class={cn(
                                    "btn btn-sm px-2.5",
                                    activeDays.includes(day.value)
                                        ? "btn-primary"
                                        : "bg-base-200 hover:bg-base-300 border-base-300 text-base-content/70",
                                )}
                                onclick={() => toggleDay(day.value)}
                            >
                                {day.label}
                            </button>
                        {/each}
                    </div>
                </div>
            {/if}

            {#if period === "month"}
                <div class="flex items-center gap-3">
                    <span class="text-xs font-medium opacity-70"
                        >Day of month</span
                    >
                    <select
                        bind:value={activeMonthDay}
                        onchange={(e) =>
                            onActiveMonthDayChange(
                                parseInt(e.currentTarget.value),
                            )}
                        class="select select-sm select-bordered"
                    >
                        {#each Array.from({ length: 31 }, (_, i) => i + 1) as d}
                            <option value={d}>{d}</option>
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
                    <Repeat class="w-3 h-3 text-primary" />
                </div>
                <p class="text-xs text-base-content/60">
                    Repeats every <span class="font-semibold text-primary">
                        {frequency > 1 ? `${frequency} ${period}s` : period}
                    </span>
                    {#if period === "week" && activeDays.length > 0 && activeDays.length < 7}
                        on <span class="font-medium">
                            {activeDays
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
                                .join(", ")}
                        </span>
                    {/if}
                </p>
            </div>
        </div>
    {/if}
</div>
