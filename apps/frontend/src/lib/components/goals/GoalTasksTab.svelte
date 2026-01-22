<script lang="ts">
import type { Goal, GoalTaskLink } from '$lib/api';
import { cn } from '$lib/utils';
import {
	ArrowUp,
	ArrowUpDown,
	Calendar,
	Check,
	ListTodo,
	Minus,
	Search,
	Tag,
	TrendingUp,
	X as XIcon,
} from 'lucide-svelte';

interface Props {
	linkedTasks: GoalTaskLink[];
	totalContributions: number;
}

let { linkedTasks = [], totalContributions = 0 }: Props = $props();

function formatDate(dateStr?: string): string {
	if (!dateStr) return '';
	const date = new Date(dateStr);
	return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function formatTime(dateStr?: string): string {
	if (!dateStr) return '';
	const date = new Date(dateStr);
	return date.toLocaleTimeString('en-US', {
		hour: 'numeric',
		minute: '2-digit',
	});
}

let taskFilter = $state<'all' | 'positive' | 'negative' | 'neutral'>('all');
let taskSearch = $state('');
let taskSortBy = $state<'date' | 'impact'>('date');

const filteredTasks = $derived.by(() => {
	let tasks = linkedTasks;

	if (taskFilter !== 'all') {
		tasks = tasks.filter((t) => t.impact_type === taskFilter);
	}

	if (taskSearch.trim()) {
		const search = taskSearch.toLowerCase();
		tasks = tasks.filter((t) => t.task_title.toLowerCase().includes(search));
	}

	// Sort by selected criteria
	if (taskSortBy === 'impact') {
		tasks = [...tasks].sort(
			(a, b) => (b.impact_magnitude || 0) - (a.impact_magnitude || 0),
		);
	} else if (taskSortBy === 'date') {
		// Sort by task start date (most recent first), then by linked_at
		tasks = [...tasks].sort((a, b) => {
			const dateA = a.task_start_date || a.linked_at || '';
			const dateB = b.task_start_date || b.linked_at || '';
			return new Date(dateB).getTime() - new Date(dateA).getTime();
		});
	}

	return tasks;
});

const taskCounts = $derived({
	all: linkedTasks.length,
	positive: linkedTasks.filter((t) => t.impact_type === 'positive').length,
	negative: linkedTasks.filter((t) => t.impact_type === 'negative').length,
	neutral: linkedTasks.filter((t) => t.impact_type === 'neutral').length,
});
</script>

<div class="p-6">
    <!-- Stats Bar -->
    <div class="flex items-center gap-4 mb-4 p-3 bg-base-200/50 rounded-xl">
        <div class="flex items-center gap-2 text-sm">
            <ListTodo class="w-4 h-4 text-primary" />
            <span class="font-medium">{linkedTasks.length}</span>
            <span class="text-base-content/60">linked tasks</span>
        </div>
        {#if totalContributions > 0}
            <div class="flex items-center gap-2 text-sm">
                <TrendingUp class="w-4 h-4 text-success" />
                <span class="font-medium">{totalContributions}</span>
                <span class="text-base-content/60">contributions</span>
            </div>
        {/if}
    </div>

    {#if linkedTasks.length > 0}
        <!-- Filter Bar -->
        <div class="flex items-center gap-3 mb-4">
            <!-- Search -->
            <div class="relative flex-1">
                <Search
                    class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-base-content/40"
                />
                <input
                    type="text"
                    placeholder="Search tasks..."
                    class="input input-sm input-bordered w-full pl-9 bg-base-200/50"
                    bind:value={taskSearch}
                />
            </div>

            <!-- Quick Filters -->
            <div class="flex gap-1">
                <button
                    class={cn(
                        "btn btn-sm gap-1",
                        taskFilter === "all" ? "btn-primary" : "btn-ghost",
                    )}
                    onclick={() => (taskFilter = "all")}
                >
                    All
                    <span class="badge badge-xs">{taskCounts.all}</span>
                </button>
                {#if taskCounts.positive > 0}
                    <button
                        class={cn(
                            "btn btn-sm gap-1",
                            taskFilter === "positive"
                                ? "btn-success"
                                : "btn-ghost text-success",
                        )}
                        onclick={() => (taskFilter = "positive")}
                    >
                        <Check class="w-3 h-3" />
                        <span class="badge badge-xs badge-success"
                            >{taskCounts.positive}</span
                        >
                    </button>
                {/if}
                {#if taskCounts.negative > 0}
                    <button
                        class={cn(
                            "btn btn-sm gap-1",
                            taskFilter === "negative"
                                ? "btn-error"
                                : "btn-ghost text-error",
                        )}
                        onclick={() => (taskFilter = "negative")}
                    >
                        <XIcon class="w-3 h-3" />
                        <span class="badge badge-xs badge-error"
                            >{taskCounts.negative}</span
                        >
                    </button>
                {/if}
            </div>

            <!-- Sort dropdown -->
            <div class="dropdown dropdown-end">
                <button class="btn btn-sm btn-ghost gap-1">
                    <ArrowUpDown class="w-3 h-3" />
                    {taskSortBy === 'date' ? 'Date' : 'Impact'}
                </button>
                <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
                <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow-lg bg-base-100 rounded-box w-40">
                    <li>
                        <button
                            class={cn(taskSortBy === 'date' && 'active')}
                            onclick={() => (taskSortBy = 'date')}
                        >
                            <Calendar class="w-4 h-4" />
                            By Date
                        </button>
                    </li>
                    <li>
                        <button
                            class={cn(taskSortBy === 'impact' && 'active')}
                            onclick={() => (taskSortBy = 'impact')}
                        >
                            <TrendingUp class="w-4 h-4" />
                            By Impact
                        </button>
                    </li>
                </ul>
            </div>
        </div>

        <!-- Task List -->
        <div class="space-y-2 max-h-[400px] overflow-y-auto pr-2">
            {#each filteredTasks as task (task.task_id)}
                <div
                    class="flex items-start gap-3 p-4 rounded-xl bg-base-200/30 hover:bg-base-200/60 transition-all border border-transparent hover:border-base-300 group"
                >
                    <!-- Impact indicator -->
                    <div
                        class={cn(
                            "w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 transition-transform group-hover:scale-110",
                            task.impact_type === "positive"
                                ? "bg-success/20 text-success"
                                : task.impact_type === "negative"
                                  ? "bg-error/20 text-error"
                                  : "bg-base-300 text-base-content/50",
                        )}
                    >
                        {#if task.impact_type === "positive"}
                            <Check class="w-5 h-5" />
                        {:else if task.impact_type === "negative"}
                            <XIcon class="w-5 h-5" />
                        {:else}
                            <Minus class="w-5 h-5" />
                        {/if}
                    </div>

                    <!-- Task info -->
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2">
                            <p class="font-medium truncate">{task.task_title}</p>
                            {#if task.task_completed}
                                <span class="badge badge-xs badge-success">Done</span>
                            {/if}
                        </div>

                        <!-- Date and time info -->
                        {#if task.task_start_date}
                            <div class="flex items-center gap-1 mt-1 text-xs text-base-content/50">
                                <Calendar class="w-3 h-3" />
                                <span>{formatDate(task.task_start_date)}</span>
                                {#if task.task_start_date && task.task_end_date}
                                    <span>{formatTime(task.task_start_date)} - {formatTime(task.task_end_date)}</span>
                                {/if}
                            </div>
                        {/if}

                        <div class="flex items-center gap-2 mt-2 flex-wrap">
                            {#if task.quantity_value}
                                <span class="badge badge-sm bg-base-300 gap-1">
                                    <ArrowUp class="w-3 h-3 text-success" />
                                    +{task.quantity_value}
                                    {task.unit_id?.replace("units:", "") || ""}
                                </span>
                            {/if}
                            {#if task.task_category}
                                <span
                                    class="badge badge-sm gap-1"
                                    style="background-color: {task.task_category.color}20; color: {task.task_category.color}; border-color: {task.task_category.color}40"
                                >
                                    <Tag class="w-3 h-3" />
                                    {task.task_category.name}
                                </span>
                            {/if}
                            {#if task.impact_magnitude && task.impact_magnitude > 0}
                                <span class="text-xs text-base-content/50">
                                    Impact: {task.impact_magnitude}/5
                                </span>
                            {/if}
                        </div>

                        {#if task.notes}
                            <p class="text-xs text-base-content/60 mt-2 line-clamp-2">{task.notes}</p>
                        {/if}
                    </div>

                    <!-- Linked time -->
                    {#if task.linked_at}
                        <div class="text-xs text-base-content/40 text-right">
                            {formatDate(task.linked_at)}
                        </div>
                    {/if}
                </div>
            {:else}
                <div class="text-center py-8 text-base-content/50">
                    <Search class="w-8 h-8 mx-auto mb-2 opacity-30" />
                    <p class="text-sm">No tasks match your filters</p>
                </div>
            {/each}
        </div>
    {:else}
        <div class="text-center py-16 text-base-content/50">
            <div
                class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-base-200 flex items-center justify-center"
            >
                <ListTodo class="w-8 h-8 opacity-30" />
            </div>
            <p class="font-medium text-lg">No linked tasks yet</p>
            <p class="text-sm mt-1 max-w-xs mx-auto">
                Tasks that contribute to this goal will appear here. Link tasks
                from the task details page.
            </p>
        </div>
    {/if}
</div>
