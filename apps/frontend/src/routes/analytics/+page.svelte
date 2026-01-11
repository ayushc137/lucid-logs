<script lang="ts">
import { EmptyState, PageHeader } from '$lib/components/ui';
import { BarChart3 } from 'lucide-svelte';
</script>

<svelte:head>
  <title>Analytics - Lucid Logs</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
  <PageHeader
    title="Analytics"
    subtitle="Insights and statistics about your productivity"
  />

  <EmptyState
    title="Coming Soon"
    description="Analytics will appear here once you have enough data to visualize your productivity trends"
    showButton={false}
  >
    {#snippet icon()}
      <BarChart3 class="w-10 h-10 text-primary" />
    {/snippet}
  </EmptyState>
</div>
