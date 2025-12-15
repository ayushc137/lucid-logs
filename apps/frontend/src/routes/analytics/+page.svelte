<script lang="ts">
  import { BarChart3 } from 'lucide-svelte';
</script>

<svelte:head>
  <title>Analytics - Lucid Logs</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-6">
  <h1 class="text-2xl font-bold">Analytics</h1>

  <div class="card bg-base-100 border border-base-300">
    <div class="card-body items-center text-center py-12">
      <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
        <BarChart3 class="w-8 h-8 text-base-content/30" />
      </div>
      <h2 class="text-xl font-semibold text-base-content/70">Coming Soon</h2>
      <p class="text-base-content/50 mt-1 max-w-md">
        Analytics will appear here once you have data
      </p>
    </div>
  </div>
</div>
