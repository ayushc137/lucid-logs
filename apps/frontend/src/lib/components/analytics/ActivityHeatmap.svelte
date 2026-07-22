<script lang="ts">
import { type ActivityHeatmapDay, getActivityHeatmap } from '$lib/api';
import { createQuery } from '@tanstack/svelte-query';
import { writable } from 'svelte/store';
import { Flame } from 'lucide-svelte';

interface Props {
	startDate?: string;
	endDate?: string;
}

let { startDate, endDate }: Props = $props();

function makeQueryFn() {
	// If props are provided, use them; otherwise default to 365 days
	if (startDate && endDate) {
		return () => getActivityHeatmap({ start_date: startDate, end_date: endDate });
	}
	return () =>
		getActivityHeatmap({
			start_date: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
			end_date: new Date().toISOString(),
		});
}

function makeOptions() {
	return {
		queryKey: ['activity-heatmap', startDate ?? 'default', endDate ?? 'default'] as const,
		queryFn: makeQueryFn(),
	};
}

const heatmapOptions$ = writable(makeOptions());

// Re-emit when props change so the query refetches
$effect(() => {
	startDate;
	endDate;
	heatmapOptions$.set(makeOptions());
});

const heatmapQuery = createQuery(heatmapOptions$);

const data = $derived($heatmapQuery.data);

const days = $derived(data?.days ?? []);

// Adaptive sizing based on the number of days in range
const rangeDays = $derived(days.length);
const isShortRange = $derived(rangeDays <= 14);
const isMediumRange = $derived(rangeDays > 14 && rangeDays <= 62);

const CELL = $derived(isShortRange ? 36 : isMediumRange ? 20 : 24);
const GAP = $derived(isShortRange ? 6 : isMediumRange ? 4 : 3);
const cellRoundClass = $derived(isShortRange ? 'rounded-md' : 'rounded');
const showMonthLabels = $derived(rangeDays > 62);
const weeksJustifyClass = $derived(isShortRange ? 'justify-center' : '');

const currentStreak = $derived(data?.current_streak ?? 0);
const longestStreak = $derived(data?.longest_streak ?? 0);

// Compute total active days + total hours from the days array
const rangeStats = $derived.by(() => {
	const activeDays = days.filter((d) => d.count > 0).length;
	const totalMinutes = days.reduce((sum, d) => sum + d.minutes, 0);
	const totalHours = totalMinutes / 60;
	return { activeDays, totalHours };
});

// Today's date string for highlight
const todayStr = $derived(new Date().toISOString().slice(0, 10));

// Group days by week for the grid
const weeks = $derived.by(() => {
	const result: ActivityHeatmapDay[][] = [];
	let currentWeek: ActivityHeatmapDay[] = [];
	for (const day of days) {
		const date = new Date(day.date);
		const dayOfWeek = date.getDay(); // 0 = Sunday
		if (dayOfWeek === 0 && currentWeek.length > 0) {
			result.push(currentWeek);
			currentWeek = [];
		}
		currentWeek.push(day);
	}
	if (currentWeek.length > 0) {
		result.push(currentWeek);
	}
	return result;
});

const monthLabels = $derived.by(() => {
	const labels: { label: string; weekIndex: number }[] = [];
	let lastMonth = -1;
	weeks.forEach((week, i) => {
		const firstDay = week[0];
		if (!firstDay) return;
		const month = new Date(firstDay.date).getMonth();
		if (month !== lastMonth) {
			labels.push({
				label: new Date(firstDay.date).toLocaleDateString('en-US', {
					month: 'short',
				}),
				weekIndex: i,
			});
			lastMonth = month;
		}
	});
	return labels;
});

const weekDays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

function intensityClass(intensity: number): string {
	switch (intensity) {
		case 0:
			return 'bg-base-300/50';
		case 1:
			return 'bg-primary/20';
		case 2:
			return 'bg-primary/40';
		case 3:
			return 'bg-primary/60';
		case 4:
			return 'bg-primary/80';
		default:
			return 'bg-base-300/50';
	}
}

// Human-readable tooltip for a day
function dayTooltip(day: ActivityHeatmapDay): string {
	const d = new Date(day.date);
	const dayName = d.toLocaleDateString('en-US', { weekday: 'short' });
	const dateStr = d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	const h = Math.floor(day.minutes / 60);
	const m = Math.round(day.minutes % 60);
	const timeStr = h > 0 ? `${h}h${m > 0 ? ` ${m}min` : ''}` : `${m}min`;
	return `${dayName}, ${dateStr}: ${day.count} task${day.count !== 1 ? 's' : ''}, ${timeStr}`;
}

// Format minutes for legend/summary
function fmtMinutes(min: number): string {
	const h = Math.floor(min / 60);
	const m = Math.round(min % 60);
	if (h > 0) return `${h}h${m > 0 ? ` ${m}min` : ''}`;
	return `${m}min`;
}

// Scroll container ref for auto-scroll to most recent
let scrollEl: HTMLDivElement | undefined = $state();

// Auto-scroll to end (most recent) on mount / when data loads — only if content overflows
$effect(() => {
	if (scrollEl && weeks.length > 0 && scrollEl.scrollWidth > scrollEl.clientWidth) {
		scrollEl.scrollLeft = scrollEl.scrollWidth;
	}
});
</script>

{#if $heatmapQuery.isPending}
	<div class="flex items-center justify-center py-8">
		<span class="loading loading-spinner loading-sm text-primary"></span>
	</div>
{:else if $heatmapQuery.isError}
	<p class="text-sm text-base-content/50 py-4">Could not load activity data.</p>
{:else if days.length > 0}
	<div class="space-y-3">
		<!-- Streak summary -->
		<div class="flex items-center gap-4 text-sm flex-wrap">
			<div class="flex items-center gap-1.5 text-warning font-semibold">
				<Flame class="w-4 h-4" />
				<span>{currentStreak}d current</span>
			</div>
			<span class="text-base-content/40">best {longestStreak}d</span>
			<span class="text-base-content/40">·</span>
			<span class="text-base-content/60">{rangeStats.activeDays} active days · {rangeStats.totalHours.toFixed(0)}h tracked</span>
		</div>

		<!-- Heatmap grid (scrollable, auto-scrolls to most recent) -->
		<div class="overflow-x-auto pb-2" bind:this={scrollEl}>
			<div class="inline-flex flex-col gap-1 min-w-fit pr-2 w-full">
				<!-- Month labels (hidden for short/medium ranges) -->
				{#if showMonthLabels}
					<div class="flex gap-1 text-[10px] text-base-content/40 mb-0.5 relative" style="padding-left: {CELL + GAP + 4}px; height: 16px;">
						{#each monthLabels as m}
							<span
								class="absolute whitespace-nowrap"
								style="left: {m.weekIndex * (CELL + GAP) + CELL + GAP + 4}px;"
							>
								{m.label}
							</span>
						{/each}
					</div>
				{/if}

				<div class="flex gap-1 {weeksJustifyClass}">
					<!-- Day-of-week labels -->
					<div class="flex flex-col gap-1 text-[10px] text-base-content/40 pr-1 shrink-0" style="width: {CELL}px; gap: {GAP}px;">
						{#each weekDays as d}
							<div class="flex items-center justify-end" style="height: {CELL}px; line-height: {CELL}px;">{d}</div>
						{/each}
					</div>

					<!-- Weeks -->
					{#each weeks as week}
						<div class="flex flex-col shrink-0" style="gap: {GAP}px;">
							{#each weekDays as _, dayIndex}
								{@const day = week.find(d => new Date(d.date).getDay() === dayIndex)}
								{#if day}
									{@const isToday = day.date.slice(0, 10) === todayStr}
									<div
										class="{cellRoundClass} flex items-center justify-center shrink-0 {intensityClass(day.intensity)} {isToday ? 'ring-2 ring-primary ring-offset-1 ring-offset-base-100' : ''}"
										style="width: {CELL}px; height: {CELL}px;"
										title={dayTooltip(day)}
									>
										<span class="font-medium {day.intensity >= 2 ? 'text-primary-content' : 'text-base-content/40'}" style="font-size: {isShortRange ? 12 : 9}px;">
											{new Date(day.date).getDate()}
										</span>
									</div>
								{:else}
									<div class="shrink-0" style="width: {CELL}px; height: {CELL}px;"></div>
								{/if}
							{/each}
						</div>
					{/each}
				</div>

				<!-- Legend -->
				<div class="flex items-center gap-1.5 text-[10px] text-base-content/40 pt-2" style="padding-left: {CELL + GAP + 4}px;">
					<span>Less</span>
					{#each [0, 1, 2, 3, 4] as level}
						<div class="rounded {intensityClass(level)}" style="width: 12px; height: 12px;"></div>
					{/each}
					<span>More</span>
				</div>
			</div>
		</div>

		<!-- Recent activity summary -->
		{#if days.length > 0}
			{@const recent = days.slice(-7)}
			{@const recentActive = recent.filter(d => d.count > 0)}
			{@const recentTotal = recent.reduce((sum, d) => sum + d.count, 0)}
			{@const recentMin = recent.reduce((sum, d) => sum + d.minutes, 0)}
			<p class="text-xs text-base-content/40">
				Last 7 days: {recentActive.length} active · {recentTotal} tasks · {fmtMinutes(recentMin)}
			</p>
		{/if}
	</div>
{:else}
	<p class="text-sm text-base-content/50 py-4">No activity data yet.</p>
{/if}
