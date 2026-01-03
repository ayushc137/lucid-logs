<script lang="ts">
  import { QueryClient, QueryClientProvider } from "@tanstack/svelte-query";
  import { page } from "$app/stores";
  import { AppShell } from "$lib/components/layout";
  import { AuthGuard } from "$lib/components/auth";
  import { emotionStore } from "$lib/stores/emotions.svelte";
  import { onMount } from "svelte";

  // App styles (Tailwind via PostCSS)
  import "../app.css";

  let { children } = $props();

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 1000 * 60 * 5,
        gcTime: 1000 * 60 * 30,
        refetchOnWindowFocus: true,
        retry: 1,
      },
    },
  });

  const isAuthPage = $derived(
    $page.url.pathname.startsWith("/login") ||
      $page.url.pathname.startsWith("/register"),
  );

  // Show search ONLY on dashboard and tasks list
  const showSearch = $derived(
    $page.url.pathname === "/" || $page.url.pathname === "/tasks",
  );

  onMount(() => {
    emotionStore.init();
  });
</script>

<QueryClientProvider client={queryClient}>
  {#if isAuthPage}
    <div class="min-h-screen bg-background">
      {@render children()}
    </div>
  {:else}
    <AuthGuard>
      <AppShell {showSearch}>
        {@render children()}
      </AppShell>
    </AuthGuard>
  {/if}
</QueryClientProvider>
