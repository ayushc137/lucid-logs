<script lang="ts">
    import { TaskForm } from "$lib/components/tasks";
    import { PageHeader, LoadingCard, ErrorAlert } from "$lib/components/ui";
    import { getTask, type Task } from "$lib/api";
    import { createQuery } from "@tanstack/svelte-query";
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import { browser } from "$app/environment";
    import { onMount } from "svelte";
    import { ArrowLeft, Home, ClipboardList, Edit3 } from "lucide-svelte";

    // Get task ID from URL (if editing)
    const taskId = $derived($page.params.id);
    const isEditing = $derived(!!taskId);

    // Get initial category from URL params (for create mode)
    const initialCategoryId = $derived(
        $page.url.searchParams.get("category") || undefined,
    );

    // Track where the user came from for proper "back" navigation
    let referrer = $state<string>("/tasks");

    onMount(() => {
        if (browser) {
            // Check sessionStorage for referrer (set when navigating to this page)
            const storedReferrer = sessionStorage.getItem("task-form-referrer");
            if (storedReferrer) {
                referrer = storedReferrer;
            }
        }
    });

    // Fetch task data if editing
    const taskQuery = createQuery({
        queryKey: ["task", taskId],
        queryFn: () => getTask(taskId!),
        enabled: isEditing,
    });

    const task = $derived($taskQuery.data as Task | undefined);
    const pageTitle = $derived(
        isEditing ? task?.title || "Edit Task" : "Create Task",
    );

    function handleBack() {
        goto(referrer);
    }
</script>

<svelte:head>
    <title>{pageTitle} - Lucid Logs</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
    <!-- Navigation Header -->
    <div class="flex items-center gap-4 mb-6">
        <button class="btn btn-ghost btn-sm gap-2" onclick={handleBack}>
            <ArrowLeft class="w-4 h-4" />
            <span class="hidden sm:inline">Back</span>
        </button>

        <!-- Breadcrumbs -->
        <div class="breadcrumbs text-sm flex-1">
            <ul class="flex-wrap">
                <li>
                    <a
                        href="/"
                        class="flex items-center gap-1 opacity-60 hover:opacity-100"
                    >
                        <Home class="w-4 h-4" />
                    </a>
                </li>
                <li>
                    <a href="/tasks" class="opacity-60 hover:opacity-100"
                        >Tasks</a
                    >
                </li>
                <li class="font-medium">
                    {#if isEditing}
                        <span
                            class="max-w-[200px] truncate flex items-center gap-1"
                        >
                            <Edit3 class="w-3 h-3 shrink-0" />
                            {task?.title || "Edit"}
                        </span>
                    {:else}
                        <span class="flex items-center gap-1">
                            <ClipboardList class="w-3 h-3" />
                            New Task
                        </span>
                    {/if}
                </li>
            </ul>
        </div>
    </div>

    <!-- Content -->
    {#if isEditing && $taskQuery.isLoading}
        <LoadingCard message="Loading task..." />
    {:else if isEditing && $taskQuery.isError}
        <ErrorAlert
            message="Failed to load task"
            onRetry={() => $taskQuery.refetch()}
        />
    {:else}
        <TaskForm
            task={isEditing ? task : null}
            {initialCategoryId}
            {referrer}
        />
    {/if}
</div>
