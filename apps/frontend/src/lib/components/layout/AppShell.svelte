<script lang="ts">
import { Header, MobileNav, Sidebar } from '$lib/components/layout';
import type { Snippet } from 'svelte';

interface Props {
	children: Snippet;
	showSearch?: boolean;
	headerContent?: Snippet;
}

let { children, showSearch = true, headerContent }: Props = $props();
let sidebarCollapsed = $state(false);
let mobileMenuOpen = $state(false);
</script>

<div class="flex h-dvh overflow-hidden bg-base-200">
  <!-- Desktop Sidebar -->
  <div class="hidden lg:block flex-shrink-0">
    <Sidebar
      bind:collapsed={sidebarCollapsed}
      onToggle={() => (sidebarCollapsed = !sidebarCollapsed)}
    />
  </div>

  <!-- Mobile slide-over menu -->
  {#if mobileMenuOpen}
    <div class="fixed inset-0 z-50 lg:hidden">
      <button
        class="absolute inset-0 bg-black/40 backdrop-blur-sm animate-in fade-in duration-200"
        onclick={() => (mobileMenuOpen = false)}
        aria-label="Close menu"
      ></button>
      <div
        class="absolute left-0 top-0 h-full shadow-2xl animate-in slide-in-from-left duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Navigation menu"
      >
        <Sidebar collapsed={false} onToggle={() => (mobileMenuOpen = false)} />
      </div>
    </div>
  {/if}

  <!-- Main Content -->
  <div class="flex-1 flex flex-col overflow-hidden min-w-0">
    <Header
      onMenuClick={() => (mobileMenuOpen = true)}
      {showSearch}
      {headerContent}
    />

    <main class="flex-1 overflow-y-auto overflow-x-hidden has-mobile-nav">
      <div class="mx-auto w-full max-w-7xl px-4 py-5 sm:px-6 lg:px-8 lg:py-7">
        {@render children()}
      </div>
    </main>

    <!-- Mobile bottom tab bar -->
    <MobileNav />
  </div>
</div>
