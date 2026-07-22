<script lang="ts">
import {
	QUADRANT_COLORS,
	type Quadrant,
} from '$lib/components/emotions/emotionData';
import { OpenMoji } from '$lib/components/ui';
import {
	cn,
	stripHtml,
	FALLBACK_CATEGORY_COLOR,
	FALLBACK_GOAL_COLOR,
} from '$lib/utils';
import { Sparkles } from 'lucide-svelte';
import type { TimelineTask } from './types';

interface Props {
	task: TimelineTask;
	onclick?: () => void;
	ontoggle?: (completed: boolean) => void;
}

let { task, onclick, ontoggle }: Props = $props();

const completed = $derived(task.completed ?? false);

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

const railColor = $derived(task.categoryColor || FALLBACK_CATEGORY_COLOR);

const descriptionPreview = $derived(
	task.description
		? stripHtml(task.description, { newlineSeparator: ' · ' })
		: null,
);

// Emotion presence flags (match TaskPopover logic)
const hasSelectedEmotion = $derived(
	!!task.emotionId &&
		!!task.emotionName &&
		!!task.emotionEmoji &&
		!!task.emotionQuadrant,
);
const hasInferredEmotion = $derived(
	!!task.inferredEmotionId &&
		!!task.inferredEmotionName &&
		!!task.inferredEmotionEmoji &&
		!!task.inferredEmotionQuadrant,
);

// Whether any rich detail exists — used to toggle the meta wrapper
const hasMeta = $derived(
	!!descriptionPreview ||
		hasSelectedEmotion ||
		(!!task.inferredEmotionName && !hasSelectedEmotion) ||
		(task.linkedGoals ? task.linkedGoals.length > 0 : false),
);

// Goal helpers
const visibleGoals = $derived(task.linkedGoals?.slice(0, 2) ?? []);
const extraGoalsCount = $derived(
	task.linkedGoals && task.linkedGoals.length > 2
		? task.linkedGoals.length - 2
		: 0,
);
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
			{formatStart(task.startTime)}
		</span>
		<span class="text-[11px] tabular-nums leading-tight text-base-content/40">
			{formatEnd(task.endTime)}
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
			{task.title}
		</p>

		<!-- Line 1: duration · category dot+name (always present) -->
		<p class="text-xs text-base-content/50 mt-0.5 flex items-center gap-1.5">
			<span class="tabular-nums">{formatDuration(task.startTime, task.endTime)}</span>
			{#if task.categoryName}
				<span aria-hidden="true">·</span>
				<span
					class="w-2 h-2 rounded-full shrink-0"
					style="background-color: {railColor}"
				></span>
				<span class="truncate">{task.categoryName}</span>
			{/if}
		</p>

		<!-- Rich meta area (only when data exists) -->
		{#if hasMeta}
			<div class="mt-1.5 flex flex-wrap items-center gap-1.5">
				<!-- Description preview -->
				{#if descriptionPreview}
					<p class="w-full text-xs text-base-content/50 truncate">
						{descriptionPreview}
					</p>
				{/if}

				<!-- Selected emotion chip -->
				{#if hasSelectedEmotion}
					{@const colors = QUADRANT_COLORS[task.emotionQuadrant as Quadrant]}
					<span
						class="flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[11px] font-medium"
						style="background: color-mix(in srgb, {colors.primary} 12%, transparent); border: 1px solid color-mix(in srgb, {colors.primary} 30%, transparent); color: {colors.primary};"
					>
						<OpenMoji emoji={task.emotionEmoji ?? ''} size={14} />
						{task.emotionName}
					</span>
				{:else if task.inferredEmotionName}
					<!-- Inferred emotion chip (dashed border + Sparkles icon) -->
					{@const colors = QUADRANT_COLORS[(task.inferredEmotionQuadrant ?? 'blue') as Quadrant]}
					<span
						class="flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[11px] font-medium opacity-80"
						style="background: color-mix(in srgb, {colors.primary} 12%, transparent); border: 1px dashed color-mix(in srgb, {colors.primary} 30%, transparent); color: {colors.primary};"
					>
						<Sparkles class="w-3 h-3" />
						{task.inferredEmotionName}
					</span>
				{/if}

				<!-- Linked goal chips -->
				{#if task.linkedGoals && task.linkedGoals.length > 0}
					{#each visibleGoals as goal}
						{@const unitDisplay = goal.unitSymbol?.replace('units:', '') || ''}
						{@const formattedQty = goal.quantityValue
							? Number.isInteger(goal.quantityValue)
								? goal.quantityValue
								: goal.quantityValue.toFixed(1)
							: null}
						{@const impactSign = goal.impactType === 'negative' ? '−' : '+'}
						<span
							class="flex items-center gap-1 text-[11px] bg-base-200/50 rounded-md px-1.5 py-0.5"
						>
							<span>{goal.icon || '🎯'}</span>
							<span
								class="truncate max-w-[120px] font-medium"
								style="color: {goal.color || FALLBACK_GOAL_COLOR};"
							>{goal.title}</span>
							{#if formattedQty}
								<span
									class={cn(
										'font-semibold',
										goal.impactType === 'positive'
											? 'text-success'
											: goal.impactType === 'negative'
												? 'text-error'
												: 'text-base-content/60',
									)}
								>{impactSign}{formattedQty}<span class="ml-0.5">{unitDisplay}</span></span>
							{/if}
						</span>
					{/each}
					{#if extraGoalsCount > 0}
						<span class="text-[11px] text-base-content/50 px-1">
							+{extraGoalsCount} more
						</span>
					{/if}
				{/if}
			</div>
		{/if}
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
