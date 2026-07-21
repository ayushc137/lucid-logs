<script lang="ts">
import { Flame } from 'lucide-svelte';

interface Props {
	score: number; // 0-100
}

let { score }: Props = $props();

// Map score to color + label
const info = $derived(() => {
	if (score >= 70) return { color: 'text-success', label: 'Locked in', ring: 'stroke-success' };
	if (score >= 40) return { color: 'text-warning', label: 'Mixed focus', ring: 'stroke-warning' };
	return { color: 'text-error', label: 'Scattered', ring: 'stroke-error' };
});

// SVG arc
const r = 40;
const circumference = 2 * Math.PI * r;
const dashOffset = $derived(circumference - (score / 100) * circumference);
</script>

<div class="flex items-center gap-5">
	<div class="relative w-24 h-24 shrink-0">
		<svg viewBox="0 0 100 100" class="w-full h-full -rotate-90">
			<!-- Background circle -->
			<circle cx="50" cy="50" r={r} fill="none" stroke="currentColor" stroke-width="8" class="text-base-200" />
			<!-- Score arc -->
			<circle
				cx="50" cy="50" r={r} fill="none"
				stroke="currentColor" stroke-width="8"
				stroke-dasharray={circumference}
				stroke-dashoffset={dashOffset}
				stroke-linecap="round"
				class="{info().ring} transition-all duration-500"
			/>
		</svg>
		<div class="absolute inset-0 flex flex-col items-center justify-center">
			<span class="text-xl font-bold {info().color}">{Math.round(score)}%</span>
		</div>
	</div>
	<div class="flex flex-col gap-1">
		<span class="text-sm font-semibold {info().color}">{info().label}</span>
		<p class="text-xs text-base-content/50 leading-relaxed">
			{Math.round(score)}% of tracked time went to high-priority tasks (4+)
		</p>
	</div>
</div>
