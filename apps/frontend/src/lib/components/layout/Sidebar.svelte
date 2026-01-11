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
	{ href: '/templates', icon: Zap, label: 'Templates' },
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
    "flex flex-col h-screen transition-all duration-300 bg-base-200/95 backdrop-blur-sm border-r border-base-300 z-50 relative",
    collapsed ? "w-20" : "w-64",
  )}
>
  <!-- Logo -->
  <div
    class={cn(
      "flex items-center h-16 flex-shrink-0",
      collapsed ? "justify-center" : "px-4 gap-3",
    )}
  >
    <div class="relative group">
      <div
        class="absolute -inset-1 bg-gradient-to-r from-primary to-secondary rounded-xl blur opacity-25 group-hover:opacity-50 transition duration-200"
      ></div>
      <img
        src="/icon.png"
        alt="Lucid Logs"
        class="relative w-10 h-10 rounded-xl shadow-sm"
      />
    </div>
    {#if !collapsed}
      <div class="flex flex-col overflow-hidden">
        <span
          class="font-bold text-lg leading-tight bg-clip-text text-transparent bg-gradient-to-r from-base-content to-base-content/70"
        >
          Lucid Logs
        </span>
        <span
          class="text-[11px] font-medium text-base-content/50 uppercase tracking-wider"
        >
          Daily Journey
        </span>
      </div>
    {/if}
  </div>

  <!-- Navigation -->
  <nav
    class={cn(
      "flex-1 flex flex-col gap-2 py-4",
      collapsed
        ? "px-2 items-center"
        : "px-3 overflow-y-auto overflow-x-hidden",
    )}
  >
    {#each navItems as item}
      {@const active = isActive(item.href)}
      <a
        href={item.href}
        class={cn(
          "flex items-center transition-all duration-200 ease-out group relative",
          collapsed
            ? "justify-center w-12 h-12 rounded-xl"
            : "w-full px-3 py-2.5 rounded-xl gap-3",
          active
            ? "bg-primary text-primary-content shadow-md shadow-primary/20"
            : "text-base-content/60 hover:bg-base-200 hover:text-base-content",
        )}
        data-tip={collapsed ? item.label : undefined}
        class:tooltip={collapsed}
        class:tooltip-right={collapsed}
      >
        <item.icon
          class={cn(
            "transition-transform duration-200 group-hover:scale-110",
            collapsed ? "w-6 h-6" : "w-5 h-5",
          )}
        />
        {#if !collapsed}
          <span class="font-medium text-sm">{item.label}</span>
          {#if active}
            <div
              class="absolute right-2 w-1.5 h-1.5 rounded-full bg-white/40 animate-pulse"
            ></div>
          {/if}
        {/if}
      </a>
    {/each}
  </nav>

  <!-- Bottom -->
  <div
    class={cn(
      "flex-shrink-0 flex flex-col gap-1 border-t border-base-200 bg-base-50/50",
      collapsed ? "p-2 items-center" : "p-3",
    )}
  >
    <!-- Theme Picker -->
    <div
      class={cn(
        "dropdown",
        collapsed
          ? "dropdown-right dropdown-end"
          : "dropdown-top dropdown-end w-full",
      )}
    >
      <button
        tabindex="0"
        class={cn(
          "flex items-center transition-all duration-200",
          collapsed
            ? "justify-center w-12 h-12 rounded-xl hover:bg-base-200 text-base-content/70 hover:text-base-content"
            : "w-full p-2 rounded-xl hover:bg-base-200 text-base-content/70 hover:text-base-content gap-3",
        )}
        data-tip={collapsed ? "Theme" : undefined}
        class:tooltip={collapsed}
        class:tooltip-right={collapsed}
      >
        <Palette class={collapsed ? "w-5 h-5" : "w-4 h-4"} />
        {#if !collapsed}
          <span class="flex-1 text-left text-sm font-medium">Theme</span>
          <div
            class="flex items-center justify-center w-6 h-6 rounded-md bg-base-100 border border-base-200 shadow-sm text-sm"
          >
            {themeStore.currentTheme?.emoji}
          </div>
        {/if}
      </button>

      <ul
        class={cn(
          "dropdown-content menu bg-base-100 rounded-2xl z-[100] w-60 p-2 shadow-2xl shadow-black/5 border border-base-200",
          collapsed && "ml-2",
        )}
      >
        <li
          class="menu-title px-2 py-1 text-xs font-semibold text-base-content/40 uppercase tracking-wider"
        >
          Select Theme
        </li>
        <div class="h-64 overflow-y-auto">
          {#each THEMES as theme}
            <li>
              <button
                onclick={() => themeStore.set(theme.id)}
                class={cn(
                  "flex items-center gap-3 px-3 py-2 rounded-xl my-0.5",
                  themeStore.current === theme.id
                    ? "bg-primary/10 text-primary"
                    : "hover:bg-base-200",
                )}
              >
                <span class="text-xl">{theme.emoji}</span>
                <span class="flex-1 font-medium text-sm">{theme.label}</span>
                {#if themeStore.current === theme.id}
                  <Check class="w-4 h-4" />
                {/if}
              </button>
            </li>
          {/each}
        </div>
      </ul>
    </div>

    <!-- Settings & Logout Group -->
    <div class="flex flex-col gap-0.5">
      <a
        href="/settings"
        class={cn(
          "flex items-center transition-all duration-200",
          collapsed
            ? "justify-center w-12 h-12 rounded-xl hover:bg-base-200 text-base-content/70 hover:text-base-content"
            : "w-full p-2 rounded-xl hover:bg-base-200 text-base-content/70 hover:text-base-content gap-3",
          $page.url.pathname === "/settings" && "bg-base-200 text-base-content",
        )}
        data-tip={collapsed ? "Settings" : undefined}
        class:tooltip={collapsed}
        class:tooltip-right={collapsed}
      >
        <Settings class={collapsed ? "w-5 h-5" : "w-4 h-4"} />
        {#if !collapsed}
          <span class="text-sm font-medium">Settings</span>
        {/if}
      </a>

      <button
        class={cn(
          "flex items-center transition-all duration-200 text-error/80 hover:text-error",
          collapsed
            ? "justify-center w-12 h-12 rounded-xl hover:bg-error/10"
            : "w-full p-2 rounded-xl hover:bg-error/10 gap-3",
        )}
        onclick={handleLogout}
        data-tip={collapsed ? "Logout" : undefined}
        class:tooltip={collapsed}
        class:tooltip-right={collapsed}
      >
        <LogOut class={collapsed ? "w-5 h-5" : "w-4 h-4"} />
        {#if !collapsed}
          <span class="text-sm font-medium">Logout</span>
        {/if}
      </button>
    </div>

    <!-- Collapse Toggle -->
    <div class="divider my-1 w-full opacity-50"></div>
    <button
      class={cn(
        "flex items-center justify-center transition-all duration-200 text-base-content/40 hover:text-base-content hover:bg-base-200",
        collapsed
          ? "w-8 h-8 rounded-lg mx-auto"
          : "w-full py-1.5 rounded-lg gap-2",
      )}
      onclick={onToggle}
      aria-label="Toggle sidebar"
    >
      {#if collapsed}
        <ChevronRight class="w-4 h-4" />
      {:else}
        <ChevronLeft class="w-4 h-4" />
        <span class="text-xs font-medium uppercase tracking-wider"
          >Collapse</span
        >
      {/if}
    </button>
  </div>
</aside>
