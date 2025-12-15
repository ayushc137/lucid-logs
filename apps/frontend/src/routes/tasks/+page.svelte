<script lang="ts">
  import { ListTodo, Plus } from 'lucide-svelte';
</script>

<svelte:head>
  <title>Tasks - Lucid Logs</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold">Tasks</h1>
    <button class="btn btn-primary btn-sm gap-2">
      <Plus class="w-4 h-4" />
      New Task
    </button>
  </div>

  <div class="card bg-base-100 border border-base-300">
    <div class="card-body items-center text-center py-12">
      <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
        <ListTodo class="w-8 h-8 text-base-content/30" />
      </div>
      <h2 class="text-xl font-semibold text-base-content/70">No tasks yet</h2>
      <p class="text-base-content/50 mt-1 max-w-md">
        Create your first task to start tracking your activities
      </p>
      <button class="btn btn-primary btn-sm mt-4 gap-2">
        <Plus class="w-4 h-4" />
        Create Task
      </button>
    </div>
  </div>
</div>
