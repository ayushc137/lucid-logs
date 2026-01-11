<script lang="ts">
import { page } from '$app/stores';
import { type Task, getTask } from '$lib/api';
import { TaskForm } from '$lib/components/tasks';
import { ErrorAlert, LoadingCard } from '$lib/components/ui';
import { createQuery } from '@tanstack/svelte-query';
import { writable } from 'svelte/store';

// Get task ID from URL
const taskId = $derived($page.params.id);

// Get referrer from sessionStorage
let referrer = $state<string>('/tasks');

$effect(() => {
	const storedReferrer = sessionStorage.getItem('task-form-referrer');
	if (storedReferrer) {
		referrer = storedReferrer;
	}
});

// Fetch task data
const taskOptions = writable({
	queryKey: ['task', null as string | null | undefined],
	queryFn: () => Promise.reject(new Error('No task ID')) as Promise<Task>,
	enabled: false,
});

$effect(() => {
	taskOptions.set({
		queryKey: ['task', taskId],
		queryFn: () => getTask(taskId!),
		enabled: !!taskId,
	});
});

const taskQuery = createQuery<Task>(taskOptions);

const task = $derived($taskQuery.data as Task | undefined);
const taskTitle = $derived(task?.title || 'Edit Task');
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
