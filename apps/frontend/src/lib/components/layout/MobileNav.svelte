<script lang="ts">
import { page } from '$app/stores';
import { cn } from '$lib/utils';
import {
	BarChart3,
	Home,
	ListTodo,
	Target,
	Zap,
} from 'lucide-svelte';

const tabs = [
	{ href: '/', icon: Home, label: 'Home' },
	{ href: '/tasks', icon: ListTodo, label: 'Tasks' },
	{ href: '/activities', icon: Zap, label: 'Log' },
	{ href: '/goals', icon: Target, label: 'Goals' },
	{ href: '/analytics', icon: BarChart3, label: 'Stats' },
] as const;

function isActive(href: string): boolean {
	if (href === '/') return $page.url.pathname === '/';
	return $page.url.pathname.startsWith(href);
}
</script>

<nav
  class="lg:hidden fixed bottom-0 inset-x-0 z-40 bg-base-100/90 backdrop-blur-lg border-t border-base-300 pb-safe"
  aria-label="Primary"
>
  <div class="grid grid-cols-5 h-16">
    {#each tabs as tab}
      {@const active = isActive(tab.href)}
      <a
        href={tab.href}
        class={cn(
          "relative flex flex-col items-center justify-center gap-0.5 transition-colors",
          active ? "text-primary" : "text-base-content/50 active:text-base-content"
        )}
        aria-current={active ? 'page' : undefined}
      >
        {#if active}
          <span class="absolute top-0 h-0.5 w-8 rounded-full bg-primary"></span>
        {/if}
        <tab.icon class={cn("w-5 h-5 transition-transform", active && "scale-110")} strokeWidth={active ? 2.4 : 2} />
        <span class={cn("text-[10px] font-medium leading-none", active && "font-semibold")}>
          {tab.label}
        </span>
      </a>
    {/each}
  </div>
</nav>
