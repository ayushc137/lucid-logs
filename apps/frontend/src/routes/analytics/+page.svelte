<script lang="ts">
  import { BarChart3 } from "lucide-svelte";
</script>

<svelte:head>
  <title>Analytics - Lucid Logs</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
  <!-- Header -->
  <div
    class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
  >
    <div>
      <h1 class="text-3xl font-bold">Analytics</h1>
      <p class="text-sm opacity-60 mt-1">
        Insights and statistics about your productivity
      </p>
    </div>
  </div>

  <!-- Empty State -->
  <div class="card bg-base-100 shadow-lg border border-base-200">
    <div class="card-body py-12 text-center">
      <div
        class="w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4"
      >
        <BarChart3 class="w-10 h-10 text-primary" />
      </div>
      <h2 class="text-2xl font-bold">Coming Soon</h2>
      <p class="opacity-60 mt-2 max-w-md mx-auto">
        Analytics will appear here once you have enough data to visualize your
        productivity trends
      </p>
    </div>
  </div>
</div>
