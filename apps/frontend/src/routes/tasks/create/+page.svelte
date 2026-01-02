<script lang="ts">
    import { TaskForm } from "$lib/components/tasks";
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import { browser } from "$app/environment";
    import { onMount } from "svelte";
    import { ArrowLeft, Home, ClipboardList } from "lucide-svelte";

    // Get initial category from URL params
    const initialCategoryId = $derived(
        $page.url.searchParams.get("category") || undefined,
    );

    // Track where the user came from for proper "back" navigation
    let referrer = $state<string>("/tasks");

    onMount(() => {
        if (browser) {
            // Check sessionStorage for referrer
            const storedReferrer = sessionStorage.getItem("task-form-referrer");
            if (storedReferrer) {
                referrer = storedReferrer;
            }
        }
    });

    function handleBack() {
        goto(referrer);
    }
</script>

<svelte:head>
    <title>Create Task - Lucid Logs</title>
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
                <li class="font-medium flex items-center gap-1">
                    <ClipboardList class="w-3 h-3" />
                    New Task
                </li>
            </ul>
        </div>
    </div>

    <!-- Task Form -->
    <TaskForm task={null} {initialCategoryId} {referrer} />
</div>
