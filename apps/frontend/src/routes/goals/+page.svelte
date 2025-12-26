<script lang="ts">
  import { Target, Plus } from "lucide-svelte";
</script>

<svelte:head>
  <title>Goals - Lucid Logs</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
  <!-- Header -->
  <div
    class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
  >
    <div>
      <h1
        class="text-3xl font-bold bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent"
      >
        Goals
      </h1>
      <p class="text-sm opacity-60 mt-1">
        Track your progress towards your objectives
      </p>
    </div>
    <button
      class="btn btn-primary gap-2 shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-all"
    >
      <Plus class="w-4 h-4" />
      New Goal
    </button>
  </div>

  <!-- Empty State -->
  <div class="card bg-base-100 shadow-lg border border-base-200">
    <div class="card-body py-12 text-center">
      <div
        class="w-20 h-20 rounded-full bg-gradient-to-br from-primary/20 to-secondary/20 flex items-center justify-center mx-auto mb-4"
      >
        <Target class="w-10 h-10 text-primary" />
      </div>
      <h2 class="text-2xl font-bold">No goals yet</h2>
      <p class="opacity-60 mt-2 max-w-md mx-auto">
        Set goals to track your progress and stay motivated on your journey
      </p>
      <button class="btn btn-primary mt-6 gap-2 shadow-lg shadow-primary/20">
        <Plus class="w-4 h-4" />
        Create Your First Goal
      </button>
    </div>
  </div>
</div>
