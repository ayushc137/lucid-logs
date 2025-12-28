<script lang="ts">
  import { Target } from "lucide-svelte";
  import { PageHeader, EmptyState } from "$lib/components/ui";

  function handleAdd() {
    // TODO: Open create goal modal
  }
</script>

<svelte:head>
  <title>Goals - Lucid Logs</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
  <PageHeader
    title="Goals"
    subtitle="Track your progress towards your objectives"
    showAddButton={true}
    addButtonLabel="New Goal"
    onAdd={handleAdd}
  />

  <EmptyState
    title="No goals yet"
    description="Set goals to track your progress and stay motivated on your journey"
    buttonLabel="Create Your First Goal"
    onButtonClick={handleAdd}
  >
    {#snippet icon()}
      <Target class="w-10 h-10 text-primary" />
    {/snippet}
  </EmptyState>
</div>
