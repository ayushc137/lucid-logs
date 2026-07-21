<script lang="ts">
import type { CategoryBreakdown } from '$lib/api';

interface Props {
	distribution: CategoryBreakdown[];
	totalHours: number;
}

let { distribution, totalHours }: Props = $props();

// Build SVG donut segments
const segments = $derived(
	distribution.slice(0, 8).map((cat, i) => {
		const pct = cat.percentage / 100;
		const angle = pct * 360;
		const startAngle = distribution
			.slice(0, i)
			.reduce((a, c) => a + (c.percentage / 100) * 360, -90);
		const endAngle = startAngle + angle;

		// Colors: cycle through a palette
		const colors = [
			'oklch(var(--p))',
			'oklch(var(--s))',
			'oklch(var(--a))',
			'oklch(var(--wa))',
			'oklch(var(--in))',
			'oklch(var(--er))',
			'oklch(var(--su))',
			'oklch(var(--nc))',
		];
		const color = colors[i % colors.length];

		// Donut path
		const largeArc = angle > 180 ? 1 : 0;
		const r = 40;
		const ir = 24;
		const cx = 50;
		const cy = 50;
		const rad = (a: number) => (a * Math.PI) / 180;

		const x1 = cx + r * Math.cos(rad(startAngle));
		const y1 = cy + r * Math.sin(rad(startAngle));
		const x2 = cx + r * Math.cos(rad(endAngle));
		const y2 = cy + r * Math.sin(rad(endAngle));
		const x3 = cx + ir * Math.cos(rad(endAngle));
		const y3 = cy + ir * Math.sin(rad(endAngle));
		const x4 = cx + ir * Math.cos(rad(startAngle));
		const y4 = cy + ir * Math.sin(rad(startAngle));

		const path = `M ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} L ${x3} ${y3} A ${ir} ${ir} 0 ${largeArc} 0 ${x4} ${y4} Z`;

		return {
			path,
			color,
			name: cat.category_name,
			hours: cat.hours,
			pct: cat.percentage,
		};
	}),
);
</script>

<div class="flex items-center gap-6">
	<div class="relative shrink-0">
		<svg viewBox="0 0 100 100" class="w-40 h-40 -rotate-0">
			{#each segments as seg}
				<path d={seg.path} fill={seg.color} opacity="0.85" />
			{/each}
		</svg>
		<div class="absolute inset-0 flex flex-col items-center justify-center">
			<span class="text-2xl font-bold">{totalHours.toFixed(1)}h</span>
			<span class="text-xs text-base-content/50">total</span>
		</div>
	</div>
	<div class="flex flex-col gap-1.5 min-w-0">
		{#each segments as seg}
			<div class="flex items-center gap-2 text-sm">
				<span class="w-2.5 h-2.5 rounded-full shrink-0" style="background: {seg.color}"></span>
				<span class="truncate font-medium">{seg.name}</span>
				<span class="text-base-content/50 ml-auto shrink-0">{seg.hours.toFixed(1)}h · {Math.round(seg.pct)}%</span>
			</div>
		{/each}
	</div>
</div>
