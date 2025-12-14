<script lang="ts">
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import { ModeWatcher } from 'mode-watcher';
	import '../app.css';

	let { children } = $props();

	// TanStack Query client with sensible defaults
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: 1000 * 60 * 5, // 5 minutes
				gcTime: 1000 * 60 * 30, // 30 minutes
				refetchOnWindowFocus: true,
				retry: 1
			}
		}
	});
</script>

<!-- Theme management -->
<ModeWatcher defaultMode="system" />

<!-- Query client provider for data fetching -->
<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>
