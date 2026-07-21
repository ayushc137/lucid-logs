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
      <div
        class="absolute inset-0 bg-black/40 backdrop-blur-sm"
        onclick={() => (mobileMenuOpen = false)}
        onkeydown={(e) => e.key === "Escape" && (mobileMenuOpen = false)}
        role="button"
        tabindex="0"
        aria-label="Close menu"
      ></div>
      <div class="absolute left-0 top-0 h-full shadow-2xl">
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
