<script lang="ts">
import { THEMES, themeStore } from '$lib/stores';
import { cn } from '$lib/utils';
import { Check, Moon, Palette, Sun } from 'lucide-svelte';
</script>

<div class="bg-base-100 rounded-box border border-base-content/5 p-4 sm:p-5">
	<div class="flex items-center gap-3 mb-4">
		<div
			class="w-10 h-10 rounded-box bg-primary/10 flex items-center justify-center"
		>
			<Palette class="w-5 h-5 text-primary" />
		</div>
		<div>
			<h2 class="text-sm font-semibold">Appearance</h2>
			<p class="text-xs text-base-content/60">Choose light or dark</p>
		</div>
	</div>

	<div class="grid grid-cols-2 gap-3">
		{#each THEMES as theme}
			<button
				onclick={() => themeStore.set(theme.id)}
				class={cn(
					"relative p-4 rounded-box border-2 transition-all duration-200 text-left active:scale-[0.98]",
					themeStore.current === theme.id
						? "border-primary bg-primary/5"
						: "border-base-content/10 hover:border-base-content/25",
				)}
			>
				<div class="flex items-center gap-2 mb-1.5">
					{#if theme.isDark}
						<Moon class="w-4 h-4 text-base-content/60" />
					{:else}
						<Sun class="w-4 h-4 text-base-content/60" />
					{/if}
					<p class="text-sm font-semibold">{theme.label}</p>
				</div>
				<p class="text-xs text-base-content/60">{theme.description}</p>

				{#if themeStore.current === theme.id}
					<div
						class="absolute top-2.5 right-2.5 w-5 h-5 rounded-full bg-primary flex items-center justify-center"
					>
						<Check class="w-3 h-3 text-primary-content" />
					</div>
				{/if}
			</button>
		{/each}
	</div>
</div>
