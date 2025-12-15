<script lang="ts">
  import { Calendar } from 'lucide-svelte';
</script>

<svelte:head>
  <title>Retrospectives - Lucid Logs</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-6">
  <h1 class="text-2xl font-bold">Retrospectives</h1>

  <div class="card bg-base-100 border border-base-300">
    <div class="card-body items-center text-center py-12">
      <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
        <Calendar class="w-8 h-8 text-base-content/30" />
      </div>
      <h2 class="text-xl font-semibold text-base-content/70">No retrospectives yet</h2>
      <p class="text-base-content/50 mt-1 max-w-md">
        Weekly and monthly reviews will appear here
      </p>
    </div>
  </div>
</div>
