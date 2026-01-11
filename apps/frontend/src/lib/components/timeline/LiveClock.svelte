<script lang="ts">
import { onDestroy, onMount } from 'svelte';
import { fade } from 'svelte/transition';

let {
	showOnlyToday = true,
	selectedDate = new Date(),
}: {
	showOnlyToday?: boolean;
	selectedDate?: Date;
} = $props();

let currentTime = $state(new Date());
let timeInterval: ReturnType<typeof setInterval>;

onMount(() => {
	timeInterval = setInterval(() => (currentTime = new Date()), 1000);
});

onDestroy(() => {
	if (timeInterval) clearInterval(timeInterval);
});

const isToday = $derived(() => {
	const t = new Date();
	return (
		selectedDate.getFullYear() === t.getFullYear() &&
		selectedDate.getMonth() === t.getMonth() &&
		selectedDate.getDate() === t.getDate()
	);
});

const nowFormatted = $derived(
	currentTime.toLocaleTimeString([], {
		hour: 'numeric',
		minute: '2-digit',
		hour12: true,
	}),
);

const shouldShow = $derived(!showOnlyToday || isToday());
</script>

{#if shouldShow}
    <div
        class="flex items-center gap-2 px-3 py-1.5 bg-primary/5 rounded-lg border border-primary/10"
        in:fade={{ duration: 300 }}
    >
        <div class="w-2 h-2 rounded-full bg-primary animate-pulse"></div>
        <span class="text-xs font-mono font-bold text-primary"
            >{nowFormatted}</span
        >
    </div>
{/if}
