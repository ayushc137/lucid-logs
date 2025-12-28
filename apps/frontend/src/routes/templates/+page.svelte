<script lang="ts">
  import { Zap } from "lucide-svelte";
  import { PageHeader, EmptyState } from "$lib/components/ui";

  function handleAdd() {
    // TODO: Open create template modal
  }
</script>

<svelte:head>
  <title>Templates - Lucid Logs</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
  <PageHeader
    title="Templates"
    subtitle="Quick-start your tasks with reusable templates"
    showAddButton={true}
    addButtonLabel="New Template"
    onAdd={handleAdd}
  />

  <EmptyState
    title="No templates yet"
    description="Create templates to speed up your task creation with predefined settings"
    buttonLabel="Create Your First Template"
    onButtonClick={handleAdd}
  >
    {#snippet icon()}
      <Zap class="w-10 h-10 text-primary" />
    {/snippet}
  </EmptyState>
</div>
