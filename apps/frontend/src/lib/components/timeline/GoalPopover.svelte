<script lang="ts">
import {
	Check,
	Calendar,
	Flame,
	Repeat,
	Target,
	TrendingUp,
} from 'lucide-svelte';
import { scale } from 'svelte/transition';
import type { TimelineGoal } from './types';

interface Props {
	goal: TimelineGoal;
	position: { x: number; y: number };
}

let { goal, position }: Props = $props();

// Calculate smart positioning - always open upward to not hide timeline
const smartPosition = $derived.by(() => {
	if (typeof window === 'undefined') {
		return { x: position.x, y: position.y };
	}

	const POPOVER_WIDTH = 280;
	const OFFSET = 16;
	const PADDING = 8;

	const viewportWidth = window.innerWidth;

	const spaceRight = viewportWidth - position.x;

	const shouldFlipX = spaceRight < POPOVER_WIDTH + OFFSET + PADDING;

	let x = shouldFlipX
		? Math.max(PADDING, position.x - POPOVER_WIDTH - OFFSET)
		: position.x + OFFSET;

	// Always position above the cursor
	let y = position.y - OFFSET;

	return { x, y };
});

const isHabit = $derived(!!goal.recurrence);
const isMet = $derived(
	goal.todayStatus === 'met' || goal.todayStatus === 'exceeded',
);

function formatRecurrence(
	rec: { frequency: number; period: string } | undefined,
): string {
	if (!rec) return '';
	const freq = rec.frequency === 1 ? '' : `${rec.frequency}x `;
	const periodMap: Record<string, string> = {
		day: 'Daily',
		week: 'Weekly',
		month: 'Monthly',
	};
	return freq + (periodMap[rec.period] || `${rec.period}ly`);
}

function formatDate(date: Date | undefined): string {
	if (!date) return '';
	return date.toLocaleDateString([], {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
	});
}
</script>

<div
    class="fixed z-50 pointer-events-none transition-all duration-150 ease-out origin-bottom"
    style="top: {smartPosition.y}px; left: {smartPosition.x}px; transform: translateY(-100%);"
    in:scale={{ duration: 200, start: 0.9, opacity: 0, easing: (t) => t * (2 - t) }}
    out:scale={{ duration: 150, start: 1, opacity: 0, easing: (t) => t * t }}
>
    <div class="w-[300px] bg-base-100/95 backdrop-blur-xl rounded-2xl shadow-[0_20px_60px_rgba(0,0,0,0.15),0_0_30px_rgba(0,0,0,0.05)] border border-base-200/50 overflow-hidden">
        <!-- Header with goal color accent -->
        <div
            class="h-1.5 w-full"
            style="background-color: {goal.color};"
        ></div>

        <div class="p-4">
            <!-- Title Section -->
            <div class="flex gap-3 mb-3">
                <div
                    class="w-3 h-3 rounded-full shrink-0 mt-1 ring-2 ring-base-200 shadow-sm"
                    style="background-color: {goal.color}"
                ></div>
                <div class="min-w-0 flex-1">
                    <p class="font-bold text-sm leading-snug line-clamp-2">{goal.title}</p>
                    <p class="text-[10px] font-semibold uppercase tracking-wider text-base-content/40 mt-1">
                        {isHabit ? "Habit" : "Goal"}
                    </p>
                </div>
            </div>

            <!-- Recurrence Info -->
            {#if goal.recurrence}
                <div class="flex items-start gap-2 mb-3 p-2.5 bg-base-200/50 rounded-lg">
                    <Repeat class="w-3.5 h-3.5 text-base-content/40 mt-0.5 shrink-0" />
                    <p class="text-xs text-base-content/60 line-clamp-2 leading-relaxed">
                        {formatRecurrence(goal.recurrence)}
                    </p>
                </div>
            {/if}

            <!-- Target Progress -->
            {#if goal.target}
                <div class="mb-3 p-2.5 bg-base-200/30 rounded-lg">
                    <div class="flex items-center gap-1.5 text-[10px] font-semibold uppercase text-base-content/40 mb-2">
                        <Target class="w-3 h-3" />
                        <span>Progress</span>
                    </div>
                    <div class="flex items-center gap-2 mb-1.5">
                        <div class="flex-1 h-2 bg-base-200/50 rounded-full overflow-hidden">
                            <div
                                class="h-full rounded-full transition-all duration-300"
                                style="width: {Math.min(goal.progress, 100)}%; background-color: {goal.color};"
                            ></div>
                        </div>
                    </div>
                    <div class="flex items-center justify-between">
                        <span class="text-[10px] text-base-content/50">
                            {Math.round(goal.target.currentValue * 100) / 100} / {goal.target.value}
                        </span>
                        <span class="text-xs font-bold" style="color: {goal.color};">
                            {Math.round(goal.progress)}%
                        </span>
                    </div>
                </div>
            {/if}

            <!-- Streak Info -->
            {#if goal.currentStreak > 0}
                <div class="flex items-center gap-2 mb-3 p-2.5 bg-warning/10 rounded-lg">
                    <Flame class="w-3.5 h-3.5 text-warning" />
                    <span class="text-xs font-bold text-warning">
                        {goal.currentStreak} day streak
                    </span>
                </div>
            {/if}

            <!-- Dates -->
            {#if goal.startDate || goal.deadline}
                <div class="space-y-2 pt-3 border-t border-base-200/50 mb-3">
                    {#if goal.startDate}
                        <div class="flex items-center gap-2">
                            <Calendar class="w-3 h-3 text-base-content/40" />
                            <span class="text-xs text-base-content/50">
                                Started {formatDate(goal.startDate)}
                            </span>
                        </div>
                    {/if}
                    {#if goal.deadline}
                        <div class="flex items-center gap-2">
                            <TrendingUp class="w-3 h-3 text-base-content/40" />
                            <span class="text-xs text-base-content/50">
                                Due {formatDate(goal.deadline)}
                            </span>
                        </div>
                    {/if}
                </div>
            {/if}

            <!-- Today's Status -->
            {#if isMet}
                <div class="flex items-center gap-1.5 mt-0 text-success bg-success/10 px-3 py-1.5 rounded-lg w-fit">
                    <Check class="w-3.5 h-3.5" strokeWidth={3} />
                    <span class="text-[11px] font-bold uppercase tracking-wide">Completed</span>
                </div>
            {/if}
        </div>
    </div>
</div>
