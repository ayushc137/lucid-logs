<script lang="ts">
import { type ActivityHeatmapDay, getActivityHeatmap } from '$lib/api';
import { createQuery } from '@tanstack/svelte-query';
import { Flame } from 'lucide-svelte';

const heatmapQuery = createQuery({
	queryKey: ['activity-heatmap'],
	queryFn: () => getActivityHeatmap(),
});

const data = $derived($heatmapQuery.data);

const days = $derived(data?.days ?? []);
const currentStreak = $derived(data?.current_streak ?? 0);
const longestStreak = $derived(data?.longest_streak ?? 0);

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
		<div class="flex items-center gap-4 text-sm">
			<div class="flex items-center gap-1.5 text-warning font-semibold">
				<Flame class="w-4 h-4" />
				<span>{currentStreak}d current</span>
			</div>
			<span class="text-base-content/40">best {longestStreak}d</span>
		</div>

		<!-- Heatmap grid -->
		<div class="overflow-x-auto">
			<div class="inline-flex flex-col gap-1 min-w-fit">
				<!-- Month labels -->
				<div class="flex gap-1 text-[10px] text-base-content/40 pl-6">
					{#each monthLabels as m}
						<span style="width: calc(12px * 4 + 4px * 3);">{m.label}</span>
					{/each}
				</div>

				<div class="flex gap-1">
					<!-- Day-of-week labels -->
					<div class="flex flex-col gap-1 text-[10px] text-base-content/40 pr-1">
						{#each weekDays as d}
							<span class="h-3 leading-3">{d}</span>
						{/each}
					</div>

					<!-- Weeks -->
					{#each weeks as week}
						<div class="flex flex-col gap-1">
							{#each weekDays as _, dayIndex}
								{@const day = week.find(d => new Date(d.date).getDay() === dayIndex)}
								{#if day}
									<div
										class="w-3 h-3 rounded-sm {intensityClass(day.intensity)}"
										title="{day.date}: {day.count} tasks, {Math.round(day.minutes)}min"
									></div>
								{:else}
									<div class="w-3 h-3 rounded-sm bg-transparent"></div>
								{/if}
							{/each}
						</div>
					{/each}
				</div>

				<!-- Legend -->
				<div class="flex items-center gap-1.5 text-[10px] text-base-content/40 pl-6 pt-1">
					<span>Less</span>
					{#each [0, 1, 2, 3, 4] as level}
						<div class="w-2.5 h-2.5 rounded-sm {intensityClass(level)}"></div>
					{/each}
					<span>More</span>
				</div>
			</div>
		</div>
	</div>
{:else}
	<p class="text-sm text-base-content/50 py-4">No activity data yet.</p>
{/if}
