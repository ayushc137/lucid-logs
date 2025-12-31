<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import {
    Home,
    ListTodo,
    Target,
    Zap,
    Calendar,
    BarChart3,
    Settings,
    ChevronLeft,
    ChevronRight,
    LogOut,
    Palette,
    Check,
  } from "lucide-svelte";
  import { cn } from "$lib/utils";
  import { authStore } from "$lib/stores/auth.svelte";
  import { themeStore, THEMES } from "$lib/stores";

  interface Props {
    collapsed?: boolean;
    onToggle?: () => void;
  }

  let { collapsed = $bindable(false), onToggle }: Props = $props();

  const navItems = [
    { href: "/", icon: Home, label: "Dashboard" },
    { href: "/tasks", icon: ListTodo, label: "Tasks" },
    { href: "/categories", icon: Palette, label: "Categories" },
    { href: "/goals", icon: Target, label: "Goals" },
    { href: "/templates", icon: Zap, label: "Quick Log" },
    { href: "/retrospectives", icon: Calendar, label: "Retros" },
    { href: "/analytics", icon: BarChart3, label: "Analytics" },
  ] as const;

  function isActive(href: string): boolean {
    if (href === "/") return $page.url.pathname === "/";
    return $page.url.pathname.startsWith(href);
  }

  function handleLogout() {
    authStore.logout();
    goto("/login");
  }
</script>

<aside
  class={cn(
    "flex flex-col h-screen transition-all duration-300 bg-base-200 border-r border-base-300",
    collapsed ? "w-16" : "w-56",
  )}
>
  <!-- Logo -->
  <div
    class={cn(
      "flex items-center h-14 border-b border-base-300",
      collapsed ? "justify-center" : "px-3 gap-3",
    )}
  >
    <img
      src="/icon.png"
      alt="Lucid Logs"
      class="w-9 h-9 rounded-lg flex-shrink-0"
    />
    {#if !collapsed}
      <div class="flex flex-col overflow-hidden">
        <span class="font-bold text-sm">Lucid Logs</span>
        <span class="text-[10px] opacity-60">Daily Journey</span>
      </div>
    {/if}
  </div>

  <!-- Navigation -->
  <nav
    class={cn(
      "flex-1 flex flex-col gap-1",
      collapsed ? "p-1 items-center" : "p-2",
    )}
  >
    {#each navItems as item}
      {@const active = isActive(item.href)}
      <a
        href={item.href}
        class={cn(
          "btn btn-ghost btn-sm",
          collapsed
            ? "btn-square tooltip tooltip-right"
            : "w-full justify-start gap-3",
          active && "btn-active",
        )}
        data-tip={collapsed ? item.label : undefined}
      >
        <item.icon class="w-5 h-5 flex-shrink-0" />
        {#if !collapsed}
          <span>{item.label}</span>
        {/if}
      </a>
    {/each}
  </nav>

  <!-- Bottom -->
  <div
    class={cn(
      "border-t border-base-300 flex flex-col",
      collapsed ? "p-1 items-center" : "p-2",
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
          "btn btn-ghost btn-sm",
          collapsed
            ? "btn-square tooltip tooltip-right"
            : "w-full justify-start gap-3",
        )}
        data-tip={collapsed ? "Theme" : undefined}
      >
        <Palette class="w-5 h-5" />
        {#if !collapsed}
          <span class="flex-1 text-left">Theme</span>
          <span class="text-lg">{themeStore.currentTheme?.emoji}</span>
        {/if}
      </button>
      <ul
        class={cn(
          "dropdown-content menu bg-base-100 rounded-xl z-[100] w-52 p-1.5 shadow-xl border border-base-300",
          collapsed && "mb-0",
        )}
      >
        {#each THEMES as theme}
          <li>
            <button
              onclick={() => themeStore.set(theme.id)}
              class={cn(
                "flex items-center gap-2 px-3 py-2 rounded-lg",
                themeStore.current === theme.id && "bg-primary/10",
              )}
            >
              <span class="text-lg">{theme.emoji}</span>
              <span class="flex-1 font-medium text-sm">{theme.label}</span>
              {#if themeStore.current === theme.id}
                <Check class="w-4 h-4 text-primary" />
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    </div>

    <a
      href="/settings"
      class={cn(
        "btn btn-ghost btn-sm mt-1",
        collapsed
          ? "btn-square tooltip tooltip-right"
          : "w-full justify-start gap-3",
        $page.url.pathname === "/settings" && "active",
      )}
      data-tip={collapsed ? "Settings" : undefined}
    >
      <Settings class="w-5 h-5" />
      {#if !collapsed}
        <span>Settings</span>
      {/if}
    </a>

    <button
      class={cn(
        "btn btn-ghost btn-sm mt-1 text-error hover:bg-error/10",
        collapsed
          ? "btn-square tooltip tooltip-right"
          : "w-full justify-start gap-3",
      )}
      onclick={handleLogout}
      data-tip={collapsed ? "Logout" : undefined}
    >
      <LogOut class="w-5 h-5" />
      {#if !collapsed}
        <span>Logout</span>
      {/if}
    </button>

    <div class="divider my-1 w-full"></div>

    <button
      class={cn(
        "btn btn-ghost btn-xs opacity-60 hover:opacity-100 transition-opacity",
        collapsed ? "btn-square" : "w-full",
      )}
      onclick={onToggle}
      aria-label="Toggle sidebar"
    >
      {#if collapsed}
        <ChevronRight class="w-3.5 h-3.5" />
      {:else}
        <ChevronLeft class="w-3.5 h-3.5" />
        <span class="text-xs">Collapse</span>
      {/if}
    </button>
  </div>
</aside>
