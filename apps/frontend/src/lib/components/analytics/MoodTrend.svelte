<script lang="ts">
import type { DailyMood } from '$lib/api';
import { QUADRANT_COLORS } from '$lib/utils/chart-colors';

interface Props {
	trend: DailyMood[];
}

let { trend }: Props = $props();

// Chart dimensions
const W = 320;
const H = 120;
const PAD_X = 8;
const PAD_Y = 10;
const plotW = W - PAD_X * 2;
const plotH = H - PAD_Y * 2;

// Scale valence (-1 to +1) to Y (inverted: higher valence = higher up)
function y(valence: number): number {
	return PAD_Y + plotH - ((valence + 1) / 2) * plotH;
}

function x(index: number): number {
	return PAD_X + (index / Math.max(trend.length - 1, 1)) * plotW;
}

// Build path
const path = $derived(
	trend.length > 0
		? trend
				.map(
					(d, i) =>
						`${i === 0 ? 'M' : 'L'} ${x(i).toFixed(1)} ${y(d.valence).toFixed(1)}`,
				)
				.join(' ')
		: '',
);

const areaPath = $derived(
	trend.length > 0
		? `${path} L ${x(trend.length - 1).toFixed(1)} ${y(-1)} L ${x(0).toFixed(1)} ${y(-1)} Z`
		: '',
);

// Date labels: first, middle, last
const dateLabels = $derived(
	trend.length > 0
		? [
				{ x: x(0), label: trend[0].date.slice(5) },
				{
					x: x(Math.floor(trend.length / 2)),
					label: trend[Math.floor(trend.length / 2)].date.slice(5),
				},
				{
					x: x(trend.length - 1),
					label: trend[trend.length - 1].date.slice(5),
				},
			]
		: [],
);
</script>

<div class="flex flex-col gap-2">
	<svg viewBox="0 0 {W} {H}" class="w-full h-32">
		<!-- Zero line -->
		<line x1={PAD_X} y1={y(0)} x2={W - PAD_X} y2={y(0)} stroke="currentColor" stroke-width="0.5" class="text-base-300" stroke-dasharray="3 3" />

		<!-- Area fill -->
		{#if areaPath}
			<path d={areaPath} fill="oklch(var(--p))" opacity="0.08" />
		{/if}

		<!-- Line -->
		{#if path}
			<path d={path} fill="none" stroke="oklch(var(--p))" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
		{/if}

		<!-- Data points -->
		{#each trend as d, i}
			<circle cx={x(i)} cy={y(d.valence)} r="3" fill="oklch(var(--p))" />
		{/each}

		<!-- Date labels -->
		{#each dateLabels as dl}
			<text x={dl.x} y={H - 2} text-anchor="middle" class="text-[8px] fill-base-content/40">{dl.label}</text>
		{/each}
	</svg>

	<!-- Mood legend: 4 quadrant colors with labels -->
	<div class="flex items-center justify-between gap-2 flex-wrap text-xs text-base-content/50">
		<span>mood over time</span>
		<div class="flex items-center gap-2.5 flex-wrap">
			<span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full inline-block" style="background: {QUADRANT_COLORS.yellow}"></span>⚡ Energized</span>
			<span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full inline-block" style="background: {QUADRANT_COLORS.green}"></span>🌿 Calm</span>
			<span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full inline-block" style="background: {QUADRANT_COLORS.red}"></span>🔥 Stressed</span>
			<span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full inline-block" style="background: {QUADRANT_COLORS.blue}"></span>💧 Low</span>
		</div>
	</div>
</div>
