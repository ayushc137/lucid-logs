<script lang="ts">
    import { TaskForm } from "$lib/components/tasks";
    import { page } from "$app/stores";

    // Get initial category from URL params
    const initialCategoryId = $derived(
        $page.url.searchParams.get("category") || undefined,
    );

    // Get referrer from sessionStorage
    let referrer = $state<string>("/tasks");

    $effect(() => {
        const storedReferrer = sessionStorage.getItem("task-form-referrer");
        if (storedReferrer) {
            referrer = storedReferrer;
        }
    });
</script>

<svelte:head>
    <title>Create Task - Lucid Logs</title>
</svelte:head>

<TaskForm task={null} {initialCategoryId} {referrer} />
