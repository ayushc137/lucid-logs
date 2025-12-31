<script lang="ts">
    import { themeStore, THEMES } from "$lib/stores";
    import { Palette, Check, Moon, Sun } from "lucide-svelte";
    import { cn } from "$lib/utils";
</script>

<div class="card bg-base-100 shadow">
    <div class="card-body">
        <div class="flex items-center gap-3 mb-4">
            <div
                class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center"
            >
                <Palette class="w-5 h-5 text-primary" />
            </div>
            <div>
                <h2 class="font-bold">Theme</h2>
                <p class="text-sm opacity-50">Choose your favorite look</p>
            </div>
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {#each THEMES as theme}
                <button
                    onclick={() => themeStore.set(theme.id)}
                    class={cn(
                        "relative p-4 rounded-xl border-2 transition-all text-left",
                        themeStore.current === theme.id
                            ? "border-primary bg-primary/10"
                            : "border-base-300 hover:border-base-content/30",
                    )}
                >
                    <div class="flex items-center gap-2 mb-2">
                        <span class="text-2xl">{theme.emoji}</span>
                        {#if theme.isDark}
                            <Moon class="w-3 h-3 opacity-50" />
                        {:else}
                            <Sun class="w-3 h-3 opacity-50" />
                        {/if}
                    </div>
                    <p class="font-semibold text-sm">{theme.label}</p>
                    <p class="text-xs opacity-50">{theme.description}</p>

                    {#if themeStore.current === theme.id}
                        <div
                            class="absolute top-2 right-2 w-5 h-5 rounded-full bg-primary flex items-center justify-center"
                        >
                            <Check class="w-3 h-3 text-primary-content" />
                        </div>
                    {/if}
                </button>
            {/each}
        </div>
    </div>
</div>
