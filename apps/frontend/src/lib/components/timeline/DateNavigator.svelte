<script lang="ts">
import { Calendar, ChevronLeft, ChevronRight, Clock } from 'lucide-svelte';
import { onDestroy, onMount } from 'svelte';
import { scale } from 'svelte/transition';

let {
	selectedDate = new Date(),
	onDateChange,
}: {
	selectedDate: Date;
	onDateChange?: (date: Date) => void;
} = $props();

let dateInputRef: HTMLInputElement = null!;
let currentTime = $state(new Date());
let timeInterval: ReturnType<typeof setInterval>;

onMount(() => {
	timeInterval = setInterval(() => (currentTime = new Date()), 1000);
});

onDestroy(() => {
	if (timeInterval) clearInterval(timeInterval);
});

function goToPreviousDay() {
	const d = new Date(selectedDate);
	d.setDate(d.getDate() - 1);
	onDateChange?.(d);
}

function goToNextDay() {
	const d = new Date(selectedDate);
	d.setDate(d.getDate() + 1);
	onDateChange?.(d);
}

function goToToday() {
	onDateChange?.(new Date());
}

function openDatePicker() {
	dateInputRef?.showPicker?.();
}

function handleDateSelect(e: Event) {
	const input = e.target as HTMLInputElement;
	if (input.value) {
		const [year, month, day] = input.value.split('-').map(Number);
		const newDate = new Date(year, month - 1, day);
		onDateChange?.(newDate);
	}
}

const isToday = $derived(() => {
	const t = new Date();
	return (
		selectedDate.getFullYear() === t.getFullYear() &&
		selectedDate.getMonth() === t.getMonth() &&
		selectedDate.getDate() === t.getDate()
	);
});

const dateLabel = $derived(() => {
	const t = new Date();
	const y = new Date(t);
	y.setDate(y.getDate() - 1);
	const tm = new Date(t);
	tm.setDate(tm.getDate() + 1);
	if (isToday()) return 'Today';
	if (selectedDate.toDateString() === y.toDateString()) return 'Yesterday';
	if (selectedDate.toDateString() === tm.toDateString()) return 'Tomorrow';
	return selectedDate.toLocaleDateString('en-US', {
		weekday: 'short',
		month: 'short',
		day: 'numeric',
	});
});

const datePickerValue = $derived(
	`${selectedDate.getFullYear()}-${String(selectedDate.getMonth() + 1).padStart(2, '0')}-${String(selectedDate.getDate()).padStart(2, '0')}`,
);

const nowFormatted = $derived(
	currentTime.toLocaleTimeString([], {
		hour: 'numeric',
		minute: '2-digit',
		hour12: true,
	}),
);
</script>

<div class="flex items-center gap-3">
    <!-- Left: Date Navigation -->
    <div class="flex items-center bg-base-200/40 rounded-xl p-1 shadow-inner">
        <button
            class="btn btn-sm btn-ghost btn-square rounded-lg hover:bg-base-100 transition-all duration-200"
            onclick={goToPreviousDay}
            aria-label="Previous Day"
        >
            <ChevronLeft class="w-4 h-4" />
        </button>
        <div class="relative">
            <button
                class="btn btn-sm btn-ghost px-3 font-bold hover:bg-base-100 rounded-lg transition-all duration-200 gap-2"
                class:text-primary={isToday()}
                onclick={openDatePicker}
            >
                <Calendar class="w-4 h-4 opacity-60" />
                <span class="text-sm">{dateLabel()}</span>
            </button>
            <!-- Hidden date input that opens native picker directly -->
            <input
                type="date"
                class="absolute inset-0 opacity-0 cursor-pointer w-full h-full"
                style="pointer-events: none;"
                bind:this={dateInputRef}
                value={datePickerValue}
                onchange={handleDateSelect}
            />
        </div>
        <button
            class="btn btn-sm btn-ghost btn-square rounded-lg hover:bg-base-100 transition-all duration-200"
            onclick={goToNextDay}
            aria-label="Next Day"
        >
            <ChevronRight class="w-4 h-4" />
        </button>
    </div>

    <!-- Right: Live Clock (when today) or Go to Today button (when not today) -->
    {#if isToday()}
        <!-- Live Clock -->
        <div
            class="flex items-center gap-2 px-3 py-1.5 bg-primary/5 rounded-lg border border-primary/10"
        >
            <div class="w-2 h-2 rounded-full bg-primary animate-pulse"></div>
            <span class="text-xs font-mono font-bold text-primary"
                >{nowFormatted}</span
            >
        </div>
    {:else}
        <!-- Go to Today Button -->
        <button
            class="btn btn-sm btn-ghost text-primary hover:bg-primary/10 rounded-lg gap-1.5 transition-all duration-200"
            onclick={goToToday}
            in:scale={{ duration: 200, start: 0.9 }}
        >
            <Clock class="w-4 h-4" />
            Today
        </button>
    {/if}
</div>
