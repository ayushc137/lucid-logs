<script lang="ts">
    import { TaskForm } from "$lib/components/tasks";
    import { LoadingCard, ErrorAlert } from "$lib/components/ui";
    import { getTask, type Task } from "$lib/api";
    import { createQuery } from "@tanstack/svelte-query";
    import { page } from "$app/stores";

    // Get task ID from URL
    const taskId = $derived($page.params.id);

    // Get referrer from sessionStorage
    let referrer = $state<string>("/tasks");

    $effect(() => {
        const storedReferrer = sessionStorage.getItem("task-form-referrer");
        if (storedReferrer) {
            referrer = storedReferrer;
        }
    });

    // Fetch task data
    const taskQuery = createQuery({
        queryKey: ["task", taskId],
        queryFn: () => getTask(taskId!),
        enabled: !!taskId,
    });

    const task = $derived($taskQuery.data as Task | undefined);
    const taskTitle = $derived(task?.title || "Edit Task");
</script>

<svelte:head>
    <title>{taskTitle} - Lucid Logs</title>
</svelte:head>

{#if $taskQuery.isLoading}
    <LoadingCard message="Loading task..." />
{:else if $taskQuery.isError}
    <ErrorAlert
        message="Failed to load task"
        onRetry={() => $taskQuery.refetch()}
    />
{:else if task}
    <TaskForm {task} {referrer} />
{/if}
