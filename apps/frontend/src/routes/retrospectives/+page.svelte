<script lang="ts">
  import { Calendar } from "lucide-svelte";
  import { PageHeader, EmptyState } from "$lib/components/ui";

  function handleAdd() {
    // TODO: Open create retrospective modal
  }
</script>

<svelte:head>
  <title>Retrospectives - Lucid Logs</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
  <PageHeader
    title="Retrospectives"
    subtitle="Review your weekly and monthly progress"
    showAddButton={true}
    addButtonLabel="New Retrospective"
    onAdd={handleAdd}
  />

  <EmptyState
    title="No retrospectives yet"
    description="Weekly and monthly reviews will help you reflect on your progress and plan ahead"
    buttonLabel="Create Your First Retrospective"
    onButtonClick={handleAdd}
  >
    {#snippet icon()}
      <Calendar class="w-10 h-10 text-primary" />
    {/snippet}
  </EmptyState>
</div>
