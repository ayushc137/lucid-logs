<script lang="ts">
import { cn, FALLBACK_CATEGORY_COLOR } from '$lib/utils';

interface Props {
	title: string;
	startTime: Date;
	endTime: Date;
	categoryColor?: string;
	categoryName?: string;
	completed?: boolean;
	onclick?: () => void;
	ontoggle?: (completed: boolean) => void;
}

let {
		title,
		startTime,
		endTime,
		categoryColor,
		categoryName,
		completed = false,
		onclick,
		ontoggle,
	}: Props = $props();

function formatStart(d: Date): string {
	if (!(d instanceof Date) || Number.isNaN(d.getTime())) return '--:--';
	const hours = d.getHours();
	const mins = d.getMinutes();
	const displayHour = hours === 0 ? 12 : hours > 12 ? hours - 12 : hours;
	return `${displayHour}:${String(mins).padStart(2, '0')}`;
}

function formatEnd(d: Date): string {
	if (!(d instanceof Date) || Number.isNaN(d.getTime())) return '';
	const hours = d.getHours();
	const mins = d.getMinutes();
	const period = hours < 12 ? 'AM' : 'PM';
	const displayHour = hours === 0 ? 12 : hours > 12 ? hours - 12 : hours;
	return `${displayHour}:${String(mins).padStart(2, '0')} ${period}`;
}

function formatDuration(s: Date, e: Date): string {
	if (!(s instanceof Date) || !(e instanceof Date)) return '';
	const mins = Math.round((e.getTime() - s.getTime()) / 60000);
	if (mins < 1) return `${Math.round((e.getTime() - s.getTime()) / 1000)}s`;
	if (mins < 60) return `${mins}m`;
	const h = Math.floor(mins / 60);
	const m = mins % 60;
	return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

const railColor = $derived(categoryColor || FALLBACK_CATEGORY_COLOR);
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class={cn(
		'group flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors cursor-pointer',
		completed ? 'opacity-60' : 'hover:bg-base-200',
	)}
	onclick={onclick}
	role="button"
	tabindex="0"
>
	<!-- Time column -->
	<div class="flex flex-col items-end justify-center shrink-0 w-14">
		<span class="text-[13px] font-medium tabular-nums leading-tight text-base-content/80">
			{formatStart(startTime)}
		</span>
		<span class="text-[11px] tabular-nums leading-tight text-base-content/40">
			{formatEnd(endTime)}
		</span>
	</div>

	<!-- Category color rail -->
	<div
		class="w-[3px] self-stretch rounded-full shrink-0"
		style="background-color: {railColor}"
	></div>

	<!-- Title + meta -->
	<div class="flex-1 min-w-0">
		<p
			class={cn(
				'text-sm font-medium truncate text-left',
				completed && 'line-through text-base-content/40',
			)}
		>
			{title}
		</p>
		<p class="text-xs text-base-content/50 mt-0.5 flex items-center gap-1.5">
			<span class="tabular-nums">{formatDuration(startTime, endTime)}</span>
			{#if categoryName}
				<span aria-hidden="true">·</span>
				<span
					class="w-2 h-2 rounded-full shrink-0"
					style="background-color: {railColor}"
				></span>
				<span class="truncate">{categoryName}</span>
			{/if}
		</p>
	</div>

	<!-- Checkbox -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_interactive_supports_focus -->
	<input
		type="checkbox"
		class="checkbox checkbox-sm shrink-0"
		checked={completed}
		onclick={(e) => {
			e.stopPropagation();
			ontoggle?.(!completed);
		}}
	/>
</div>
