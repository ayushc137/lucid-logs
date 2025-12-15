<script lang="ts">
  import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
  import { page } from '$app/stores';
  import { AppShell } from '$lib/components/layout';
  
  // UnoCSS (must be in JS, not CSS)
  import '@unocss/reset/tailwind.css';
  import 'virtual:uno.css';
  
  // App styles
  import '../app.css';

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
    $page.url.pathname.startsWith('/login') || $page.url.pathname.startsWith('/register')
  );
</script>

<QueryClientProvider client={queryClient}>
  {#if isAuthPage}
    <div class="min-h-screen bg-base-200">
      {@render children()}
    </div>
  {:else}
    <AppShell>
      {@render children()}
    </AppShell>
  {/if}
</QueryClientProvider>
