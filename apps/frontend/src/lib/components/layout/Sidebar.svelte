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
    Sun,
    Moon,
  } from "lucide-svelte";
  import { cn } from "$lib/utils";
  import { authStore } from "$lib/stores/auth.svelte";

  interface Props {
    collapsed?: boolean;
    onToggle?: () => void;
  }

  let { collapsed = $bindable(false), onToggle }: Props = $props();
  let currentTheme = $state("dark");
  let themeDropdownOpen = $state(false);

  // Available DaisyUI themes - 10 curated themes
  const themes = [
    { id: "light", emoji: "☀️", label: "Light", description: "Clean & Bright" },
    { id: "dark", emoji: "🌙", label: "Dark", description: "Easy on Eyes" },
    { id: "nord", emoji: "❄️", label: "Nord", description: "Arctic Cool" },
    { id: "night", emoji: "🌃", label: "Night", description: "Deep Blue" },
    { id: "dim", emoji: "🌑", label: "Dim", description: "Soft Dark" },
    { id: "lofi", emoji: "📻", label: "Lofi", description: "Muted Tones" },
    { id: "winter", emoji: "⛄", label: "Winter", description: "Frosty Light" },
    {
      id: "corporate",
      emoji: "💼",
      label: "Corporate",
      description: "Professional",
    },
    { id: "retro", emoji: "🕹️", label: "Retro", description: "Vintage Vibes" },
    {
      id: "dracula",
      emoji: "🧛",
      label: "Dracula",
      description: "Dark Purple",
    },
  ] as const;

  // Load saved theme on mount
  $effect(() => {
    const saved = localStorage.getItem("theme") || "dark";
    currentTheme = saved;
    document.documentElement.setAttribute("data-theme", saved);
  });

  function applyTheme(theme: string) {
    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem("theme", theme);
    currentTheme = theme;
    themeDropdownOpen = false;
  }

  const navItems = [
    { href: "/", icon: Home, label: "Dashboard" },
    { href: "/tasks", icon: ListTodo, label: "Tasks" },
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
  <div class="flex items-center h-14 px-3 border-b border-base-300">
    <div class="flex items-center gap-3 w-full">
      <div
        class="w-9 h-9 rounded-lg bg-primary flex items-center justify-center flex-shrink-0"
      >
        <span class="text-primary-content font-bold text-sm">LL</span>
      </div>
      {#if !collapsed}
        <div class="flex flex-col overflow-hidden">
          <span class="font-bold text-sm">Lucid Logs</span>
          <span class="text-[10px] opacity-60">Daily Journey</span>
        </div>
      {/if}
    </div>
  </div>

  <!-- Navigation -->
  <nav class="flex-1 p-2 overflow-y-auto">
    <ul class="menu menu-sm gap-1 p-0">
      {#each navItems as item}
        {@const active = isActive(item.href)}
        <li>
          <a
            href={item.href}
            class={cn(
              "flex items-center gap-3",
              active && "active",
              collapsed && "tooltip tooltip-right",
            )}
            data-tip={collapsed ? item.label : undefined}
          >
            <item.icon class="w-5 h-5 flex-shrink-0" />
            {#if !collapsed}
              <span>{item.label}</span>
            {/if}
          </a>
        </li>
      {/each}
    </ul>
  </nav>

  <!-- Bottom -->
  <div class="p-2 border-t border-base-300">
    <!-- Theme Picker -->
    <div class="dropdown dropdown-top dropdown-end w-full">
      <button
        tabindex="0"
        class={cn(
          "btn btn-ghost btn-sm w-full justify-start gap-3",
          collapsed && "btn-square tooltip tooltip-right",
        )}
        data-tip={collapsed ? "Theme" : undefined}
      >
        <Palette class="w-5 h-5" />
        {#if !collapsed}
          <span class="flex-1 text-left">Theme</span>
          <span class="text-lg"
            >{themes.find((t) => t.id === currentTheme)?.emoji}</span
          >
        {/if}
      </button>
      <ul
        class="dropdown-content menu bg-base-100 rounded-xl z-50 w-52 p-1.5 shadow-xl border border-base-300"
      >
        {#each themes as theme}
          <li>
            <button
              onclick={() => applyTheme(theme.id)}
              class={cn(
                "flex items-center gap-2 px-3 py-2 rounded-lg",
                currentTheme === theme.id && "bg-primary/10",
              )}
            >
              <span class="text-lg">{theme.emoji}</span>
              <span class="flex-1 font-medium text-sm">{theme.label}</span>
              {#if currentTheme === theme.id}
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
        "btn btn-ghost btn-sm w-full justify-start gap-3 mt-1",
        collapsed && "btn-square tooltip tooltip-right",
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
        "btn btn-ghost btn-sm w-full justify-start gap-3 mt-1 text-error hover:bg-error/10",
        collapsed && "btn-square tooltip tooltip-right",
      )}
      onclick={handleLogout}
      data-tip={collapsed ? "Logout" : undefined}
    >
      <LogOut class="w-5 h-5" />
      {#if !collapsed}
        <span>Logout</span>
      {/if}
    </button>

    <div class="divider my-2"></div>

    <button
      class="btn btn-ghost btn-sm w-full"
      onclick={onToggle}
      aria-label="Toggle sidebar"
    >
      {#if collapsed}
        <ChevronRight class="w-4 h-4" />
      {:else}
        <ChevronLeft class="w-4 h-4" />
        <span class="text-xs opacity-60">Collapse</span>
      {/if}
    </button>
  </div>
</aside>
