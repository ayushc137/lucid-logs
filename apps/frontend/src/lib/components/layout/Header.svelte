<script lang="ts">
  import { themeStore, THEMES } from '$lib/stores';
  import { Bell, Search, Plus, Palette, Check, LogOut, Settings, User, Menu } from 'lucide-svelte';
  import { cn } from '$lib/utils';

  interface Props {
    onMenuClick?: () => void;
  }

  let { onMenuClick }: Props = $props();
</script>

<header class="h-14 flex items-center justify-between px-4 bg-base-100 border-b border-base-content/10">
  <!-- Left -->
  <div class="flex items-center gap-3">
    <button class="lg:hidden btn btn-ghost btn-sm btn-square" onclick={onMenuClick} aria-label="Menu">
      <Menu class="w-5 h-5" />
    </button>

    <div class="hidden sm:flex items-center gap-2 px-3 py-2 rounded-lg bg-base-200 w-64">
      <Search class="w-4 h-4 opacity-50" />
      <input type="text" placeholder="Search logs..." class="bg-transparent text-sm outline-none flex-1 placeholder:opacity-50" />
      <kbd class="text-[10px] opacity-40 px-1.5 py-0.5 rounded bg-base-300">⌘K</kbd>
    </div>
  </div>

  <!-- Right -->
  <div class="flex items-center gap-2">
    <!-- Quick Log Button - fixed padding -->
    <button class="btn btn-primary btn-sm px-4 py-2 gap-2">
      <Plus class="w-4 h-4" />
      <span class="hidden sm:inline">Quick Log</span>
    </button>

    <!-- Notifications -->
    <button class="btn btn-ghost btn-sm btn-square relative" aria-label="Notifications">
      <Bell class="w-5 h-5" />
      <span class="absolute top-1 right-1 w-2 h-2 bg-error rounded-full"></span>
    </button>

    <!-- Theme Picker -->
    <div class="dropdown dropdown-end">
      <button class="btn btn-ghost btn-sm btn-square" aria-label="Theme">
        <Palette class="w-5 h-5" />
      </button>
      <div class="dropdown-content mt-2 p-2 bg-base-100 rounded-xl border border-base-content/10 shadow-xl w-56 z-50">
        <p class="text-xs font-semibold uppercase opacity-50 px-2 py-1 mb-1">Choose Theme</p>
        {#each THEMES as theme}
          <button
            onclick={() => themeStore.set(theme.id)}
            class={cn(
              'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-all',
              themeStore.current === theme.id ? 'bg-primary/15 text-primary font-medium' : 'hover:bg-base-200'
            )}
          >
            <span class="text-xl">{theme.emoji}</span>
            <div class="flex-1 text-left">
              <div>{theme.label}</div>
              <div class="text-[10px] opacity-50">{theme.description}</div>
            </div>
            {#if themeStore.current === theme.id}
              <Check class="w-4 h-4" />
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <!-- User Avatar - fixed with proper styling -->
    <div class="dropdown dropdown-end">
      <button class="avatar placeholder cursor-pointer" aria-label="User menu">
        <div class="bg-primary text-primary-content rounded-full w-8 h-8">
          <span class="text-xs font-semibold">U</span>
        </div>
      </button>
      <div class="dropdown-content mt-2 p-2 bg-base-100 rounded-xl border border-base-content/10 shadow-xl w-44 z-50">
        <a href="/settings" class="flex items-center gap-2 px-2 py-2 rounded-lg text-sm hover:bg-base-200">
          <Settings class="w-4 h-4" /> Settings
        </a>
        <button class="w-full flex items-center gap-2 px-2 py-2 rounded-lg text-sm text-error hover:bg-error/10">
          <LogOut class="w-4 h-4" /> Logout
        </button>
      </div>
    </div>
  </div>
</header>
