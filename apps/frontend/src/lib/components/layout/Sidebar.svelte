<script lang="ts">
  import { page } from '$app/stores';
  import { Home, ListTodo, Target, Zap, Calendar, BarChart3, Settings, ChevronLeft, ChevronRight } from 'lucide-svelte';
  import { cn } from '$lib/utils';

  interface Props {
    collapsed?: boolean;
    onToggle?: () => void;
  }

  let { collapsed = $bindable(false), onToggle }: Props = $props();

  const navItems = [
    { href: '/', icon: Home, label: 'Dashboard' },
    { href: '/tasks', icon: ListTodo, label: 'Tasks' },
    { href: '/goals', icon: Target, label: 'Goals' },
    { href: '/templates', icon: Zap, label: 'Quick Log' },
    { href: '/retrospectives', icon: Calendar, label: 'Retros' },
    { href: '/analytics', icon: BarChart3, label: 'Analytics' },
  ] as const;

  function isActive(href: string): boolean {
    if (href === '/') return $page.url.pathname === '/';
    return $page.url.pathname.startsWith(href);
  }
</script>

<aside
  class={cn(
    'flex flex-col h-screen transition-all duration-300 bg-base-100 border-r border-base-content/10',
    collapsed ? 'w-16' : 'w-56'
  )}
>
  <!-- Logo with custom icon -->
  <div class="flex items-center h-14 px-3 border-b border-base-content/10">
    <div class="flex items-center gap-3 w-full">
      <img src="/icon.svg" alt="Lucid Logs" class="w-9 h-9 flex-shrink-0" />
      {#if !collapsed}
        <div class="flex flex-col overflow-hidden">
          <span class="font-bold text-sm">Lucid Logs</span>
          <span class="text-[10px] opacity-50">Daily Journey</span>
        </div>
      {/if}
    </div>
  </div>

  <!-- Navigation -->
  <nav class="flex-1 p-2 overflow-y-auto">
    <ul class="flex flex-col gap-1">
      {#each navItems as item}
        {@const active = isActive(item.href)}
        <li>
          <a
            href={item.href}
            class={cn(
              'flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all',
              active ? 'bg-primary/15 text-primary font-semibold' : 'hover:bg-base-content/5'
            )}
            title={collapsed ? item.label : undefined}
          >
            <item.icon class="w-5 h-5 flex-shrink-0" />
            {#if !collapsed}
              <span class="text-sm">{item.label}</span>
            {/if}
          </a>
        </li>
      {/each}
    </ul>
  </nav>

  <!-- Bottom -->
  <div class="p-2 border-t border-base-content/10">
    <a
      href="/settings"
      class={cn(
        'flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all',
        $page.url.pathname === '/settings' ? 'bg-primary/15 text-primary' : 'hover:bg-base-content/5'
      )}
      title={collapsed ? 'Settings' : undefined}
    >
      <Settings class="w-5 h-5" />
      {#if !collapsed}
        <span class="text-sm">Settings</span>
      {/if}
    </a>

    <button
      class="w-full flex items-center justify-center gap-2 px-3 py-2 mt-2 rounded-lg text-xs opacity-50 hover:opacity-100 hover:bg-base-content/5 transition-all"
      onclick={onToggle}
      aria-label="Toggle sidebar"
    >
      {#if collapsed}
        <ChevronRight class="w-4 h-4" />
      {:else}
        <ChevronLeft class="w-4 h-4" />
        <span>Collapse</span>
      {/if}
    </button>
  </div>
</aside>
