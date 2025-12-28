<script lang="ts">
  import { Sidebar, Header } from "$lib/components/layout";

  interface Props {
    children: import("svelte").Snippet;
  }

  let { children }: Props = $props();
  let sidebarCollapsed = $state(false);
  let mobileMenuOpen = $state(false);

  // Apply saved theme on mount
  $effect(() => {
    const savedTheme = localStorage.getItem("theme");
    if (savedTheme === "dark") {
      document.documentElement.classList.add("dark");
    } else if (savedTheme === "light") {
      document.documentElement.classList.remove("dark");
    } else if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
      document.documentElement.classList.add("dark");
    }
  });
</script>

<div class="flex h-screen overflow-hidden bg-background">
  <!-- Desktop Sidebar -->
  <div class="hidden lg:block flex-shrink-0">
    <Sidebar
      bind:collapsed={sidebarCollapsed}
      onToggle={() => (sidebarCollapsed = !sidebarCollapsed)}
    />
  </div>

  <!-- Mobile Sidebar Overlay -->
  {#if mobileMenuOpen}
    <div class="fixed inset-0 z-50 lg:hidden">
      <div
        class="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onclick={() => (mobileMenuOpen = false)}
        onkeydown={(e) => e.key === "Escape" && (mobileMenuOpen = false)}
        role="button"
        tabindex="0"
        aria-label="Close menu"
      ></div>
      <div class="absolute left-0 top-0 h-full">
        <Sidebar collapsed={false} onToggle={() => (mobileMenuOpen = false)} />
      </div>
    </div>
  {/if}

  <!-- Main Content -->
  <div class="flex-1 flex flex-col overflow-hidden min-w-0">
    <Header onMenuClick={() => (mobileMenuOpen = true)} />

    <main class="flex-1 overflow-y-auto bg-muted/30">
      <div class="p-4 md:p-6 lg:p-8 h-full">
        <!-- Relaxed width constraint for wider timeline view -->
        <div class="w-full h-full">
          {@render children()}
        </div>
      </div>
    </main>
  </div>
</div>
