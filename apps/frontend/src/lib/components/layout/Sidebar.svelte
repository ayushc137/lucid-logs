<script lang="ts">
import { goto } from '$app/navigation';
import { page } from '$app/stores';
import { THEMES, themeStore } from '$lib/stores';
import { authStore } from '$lib/stores/auth.svelte';
import { cn } from '$lib/utils';
import {
	BarChart3,
	Calendar,
	Check,
	ChevronLeft,
	ChevronRight,
	Home,
	ListTodo,
	LogOut,
	Palette,
	Settings,
	Target,
	Zap,
} from 'lucide-svelte';

interface Props {
	collapsed?: boolean;
	onToggle?: () => void;
}

let { collapsed = $bindable(false), onToggle }: Props = $props();

const navItems = [
	{ href: '/', icon: Home, label: 'Dashboard' },
	{ href: '/tasks', icon: ListTodo, label: 'Tasks' },
	{ href: '/categories', icon: Palette, label: 'Categories' },
	{ href: '/goals', icon: Target, label: 'Goals' },
	{ href: '/activities', icon: Zap, label: 'Activities' },
	{ href: '/retrospectives', icon: Calendar, label: 'Retrospectives' },
	{ href: '/analytics', icon: BarChart3, label: 'Analytics' },
] as const;

function isActive(href: string): boolean {
	if (href === '/') return $page.url.pathname === '/';
	return $page.url.pathname.startsWith(href);
}

function handleLogout() {
	authStore.logout();
	goto('/login');
}
</script>

<aside
  class={cn(
    "flex flex-col h-dvh transition-[width] duration-200 ease-out bg-base-100 border-r border-base-300/70 z-50 relative",
    collapsed ? "w-[76px]" : "w-64",
  )}
>
  <!-- Logo -->
  <div
    class={cn(
      "flex items-center h-16 flex-shrink-0 border-b border-base-300/60",
      collapsed ? "justify-center" : "px-5 gap-3",
    )}
  >
    <img
      src="/icon.png"
      alt="Lucid Logs"
      class="w-8 h-8 rounded-lg ring-1 ring-base-300/60"
    />
    {#if !collapsed}
      <div class="flex flex-col overflow-hidden">
        <span class="font-semibold text-[15px] leading-tight tracking-tight">
          Lucid Logs
        </span>
        <span class="text-[11px] font-medium text-base-content/45 tracking-wide">
          Daily journey
        </span>
      </div>
    {/if}
  </div>

  <!-- Navigation -->
  <nav
    class={cn(
      "flex-1 flex flex-col gap-1 py-4",
      collapsed ? "px-3 items-center" : "px-3 overflow-y-auto overflow-x-hidden",
    )}
  >
    {#each navItems as item}
      {@const active = isActive(item.href)}
      <a
        href={item.href}
        class={cn(
          "flex items-center rounded-lg transition-colors group relative",
          collapsed ? "justify-center w-11 h-11" : "w-full px-3 py-2 gap-3",
          active
            ? "bg-primary/10 text-primary font-medium"
            : "text-base-content/60 hover:bg-base-200 hover:text-base-content",
        )}
        data-tip={collapsed ? item.label : undefined}
        class:tooltip={collapsed}
        class:tooltip-right={collapsed}
        aria-current={active ? 'page' : undefined}
      >
        {#if active && !collapsed}
          <span class="absolute left-0 top-1/2 -translate-y-1/2 h-5 w-1 rounded-r-full bg-primary"></span>
        {/if}
        <item.icon
          class={cn("shrink-0", collapsed ? "w-5 h-5" : "w-[18px] h-[18px]")}
          strokeWidth={active ? 2.3 : 2}
        />
        {#if !collapsed}
          <span class="text-sm">{item.label}</span>
        {/if}
      </a>
    {/each}
  </nav>

  <!-- Bottom -->
  <div
    class={cn(
      "flex-shrink-0 flex flex-col gap-1 border-t border-base-300/60",
      collapsed ? "p-3 items-center" : "p-3",
    )}
  >
    <!-- Theme Picker -->
    <div
      class={cn(
        "dropdown",
        collapsed ? "dropdown-right dropdown-end" : "dropdown-top dropdown-end w-full",
      )}
    >
      <button
        tabindex="0"
        class={cn(
          "flex items-center rounded-lg transition-colors text-base-content/60 hover:text-base-content hover:bg-base-200",
          collapsed ? "justify-center w-11 h-11" : "w-full px-3 py-2 gap-3",
        )}
        data-tip={collapsed ? "Theme" : undefined}
        class:tooltip={collapsed}
        class:tooltip-right={collapsed}
      >
        <Palette class={collapsed ? "w-5 h-5" : "w-[18px] h-[18px]"} />
        {#if !collapsed}
          <span class="flex-1 text-left text-sm">Theme</span>
          <span class="text-base">{themeStore.currentTheme?.emoji}</span>
        {/if}
      </button>

      <ul
        class={cn(
          "dropdown-content menu bg-base-100 rounded-xl z-[100] w-60 p-1.5 shadow-xl border border-base-300/70",
          collapsed && "ml-2",
        )}
      >
        <li class="menu-title px-2 py-1.5 text-[11px] font-semibold text-base-content/40 uppercase tracking-wider">
          Theme
        </li>
        <div class="max-h-64 overflow-y-auto">
          {#each THEMES as theme}
            <li>
              <button
                onclick={() => themeStore.set(theme.id)}
                class={cn(
                  "flex items-center gap-3 px-2.5 py-2 rounded-lg",
                  themeStore.current === theme.id
                    ? "bg-primary/10 text-primary"
                    : "hover:bg-base-200",
                )}
              >
                <span class="text-lg">{theme.emoji}</span>
                <span class="flex-1 text-sm font-medium">{theme.label}</span>
                {#if themeStore.current === theme.id}
                  <Check class="w-4 h-4" />
                {/if}
              </button>
            </li>
          {/each}
        </div>
      </ul>
    </div>

    <!-- Settings & Logout -->
    <a
      href="/settings"
      class={cn(
        "flex items-center rounded-lg transition-colors text-base-content/60 hover:text-base-content hover:bg-base-200",
        collapsed ? "justify-center w-11 h-11" : "w-full px-3 py-2 gap-3",
        $page.url.pathname === "/settings" && "bg-base-200 text-base-content",
      )}
      data-tip={collapsed ? "Settings" : undefined}
      class:tooltip={collapsed}
      class:tooltip-right={collapsed}
    >
      <Settings class={collapsed ? "w-5 h-5" : "w-[18px] h-[18px]"} />
      {#if !collapsed}
        <span class="text-sm">Settings</span>
      {/if}
    </a>

    <button
      class={cn(
        "flex items-center rounded-lg transition-colors text-base-content/60 hover:text-error hover:bg-error/10",
        collapsed ? "justify-center w-11 h-11" : "w-full px-3 py-2 gap-3",
      )}
      onclick={handleLogout}
      data-tip={collapsed ? "Logout" : undefined}
      class:tooltip={collapsed}
      class:tooltip-right={collapsed}
    >
      <LogOut class={collapsed ? "w-5 h-5" : "w-[18px] h-[18px]"} />
      {#if !collapsed}
        <span class="text-sm">Logout</span>
      {/if}
    </button>

    <!-- Collapse Toggle -->
    <div class="border-t border-base-300/60 mt-1 pt-1 w-full">
      <button
        class={cn(
          "flex items-center justify-center rounded-lg transition-colors text-base-content/40 hover:text-base-content hover:bg-base-200",
          collapsed ? "w-8 h-8 mx-auto" : "w-full py-1.5 gap-2",
        )}
        onclick={onToggle}
        aria-label="Toggle sidebar"
      >
        {#if collapsed}
          <ChevronRight class="w-4 h-4" />
        {:else}
          <ChevronLeft class="w-4 h-4" />
          <span class="text-xs font-medium">Collapse</span>
        {/if}
      </button>
    </div>
  </div>
</aside>
