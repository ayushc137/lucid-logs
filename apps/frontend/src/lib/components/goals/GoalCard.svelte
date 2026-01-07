<script lang="ts">
    import type { Goal } from "$lib/api";
    import {
        Target,
        Repeat,
        Flame,
        Check,
        Pause,
        X,
        ChevronRight,
        Calendar,
        MoreVertical,
        Pencil,
        Trash2,
        Play,
    } from "lucide-svelte";
    import { cn } from "$lib/utils";
    import { fly } from "svelte/transition";
    import { cubicOut } from "svelte/easing";

    interface Props {
        goal: Goal;
        variant?: "card" | "compact";
        transitionDelay?: number;
        onEdit?: () => void;
        onDelete?: () => void;
        onStatusChange?: (
            status: "active" | "completed" | "paused" | "abandoned",
        ) => void;
        onClick?: () => void;
    }

    let {
        goal,
        variant = "card",
        transitionDelay = 0,
        onEdit,
        onDelete,
        onStatusChange,
        onClick,
    }: Props = $props();

    const isRecurring = $derived(!!goal.recurrence);
    const progress = $derived(
        goal.target
            ? Math.min(
                  100,
                  Math.round(
                      ((goal.stats?.current_value || 0) / goal.target.value) *
                          100,
                  ),
              )
            : 0,
    );

    const statusColors = {
        active: "badge-success",
        completed: "badge-info",
        paused: "badge-warning",
        abandoned: "badge-error",
    };

    const statusIcons = {
        active: Play,
        completed: Check,
        paused: Pause,
        abandoned: X,
    };

    // Derived status icon for the current goal
    const StatusIcon = $derived(
        statusIcons[goal.status as keyof typeof statusIcons] || Play,
    );

    const goalTypeLabels = {
        discrete: "One-time",
        measurable: "Measurable",
        epic: "Epic",
        avoidance: "Avoidance",
    };

    function formatRecurrence(rec: typeof goal.recurrence): string {
        if (!rec) return "";
        const times = rec.frequency === 1 ? "" : `${rec.frequency}x `;
        return `${times}${rec.period}ly`;
    }

    function formatDeadline(date: string | undefined): string {
        if (!date) return "";
        const d = new Date(date);
        const now = new Date();
        const diff = Math.ceil(
            (d.getTime() - now.getTime()) / (1000 * 60 * 60 * 24),
        );
        if (diff < 0) return "Overdue";
        if (diff === 0) return "Today";
        if (diff === 1) return "Tomorrow";
        if (diff <= 7) return `${diff}d left`;
        return d.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
        });
    }

    let menuOpen = $state(false);
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_interactive_supports_focus -->
{#if variant === "card"}
    <div
        class="card bg-base-100 border-2 border-base-200/80 hover:border-base-300 shadow-sm hover:shadow-md transition-all duration-200 cursor-pointer group"
        role="button"
        onclick={() => onClick?.()}
        in:fly={{
            y: 10,
            duration: 250,
            delay: transitionDelay,
            easing: cubicOut,
        }}
    >
        <!-- Color Bar -->
        <div
            class="h-1.5 rounded-t-xl"
            style="background-color: {goal.category?.color || '#6b7280'};"
        ></div>

        <div class="card-body p-4 gap-3">
            <!-- Header: Icon, Title, Menu -->
            <div class="flex items-start gap-3">
                <!-- Icon -->
                <div
                    class="w-10 h-10 rounded-xl flex items-center justify-center text-xl shrink-0"
                    style="background-color: {goal.category?.color ||
                        '#6b7280'}20;"
                >
                    {goal.icon || "🎯"}
                </div>

                <!-- Title & Meta -->
                <div class="flex-1 min-w-0">
                    <h3 class="font-semibold text-sm truncate">{goal.title}</h3>
                    <div class="flex items-center gap-2 mt-0.5 flex-wrap">
                        <span
                            class="badge badge-sm {statusColors[
                                goal.status
                            ]} gap-1"
                        >
                            <StatusIcon class="w-3 h-3" />
                            {goal.status}
                        </span>
                        <span
                            class="text-[10px] uppercase font-medium opacity-50"
                        >
                            {goal.recurrence
                                ? "Recurrence"
                                : goal.target
                                  ? "Measurable"
                                  : "Goal"}
                        </span>
                    </div>
                </div>

                <!-- Menu -->
                <div class="dropdown dropdown-end">
                    <button
                        type="button"
                        tabindex="0"
                        class="btn btn-ghost btn-xs btn-square opacity-0 group-hover:opacity-100 transition-opacity"
                        onclick={(e) => {
                            e.stopPropagation();
                            menuOpen = !menuOpen;
                        }}
                    >
                        <MoreVertical class="w-4 h-4" />
                    </button>
                    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
                    <ul
                        tabindex="0"
                        class="dropdown-content z-50 menu p-2 shadow-lg bg-base-100 rounded-xl border border-base-200 w-40"
                    >
                        <li>
                            <button
                                type="button"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    onEdit?.();
                                }}
                            >
                                <Pencil class="w-4 h-4" /> Edit
                            </button>
                        </li>
                        {#if goal.status === "active"}
                            <li>
                                <button
                                    type="button"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        onStatusChange?.("paused");
                                    }}
                                >
                                    <Pause class="w-4 h-4" /> Pause
                                </button>
                            </li>
                            <li>
                                <button
                                    type="button"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        onStatusChange?.("completed");
                                    }}
                                >
                                    <Check class="w-4 h-4" /> Complete
                                </button>
                            </li>
                        {:else if goal.status === "paused"}
                            <li>
                                <button
                                    type="button"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        onStatusChange?.("active");
                                    }}
                                >
                                    <Play class="w-4 h-4" /> Resume
                                </button>
                            </li>
                        {/if}
                        <li>
                            <button
                                type="button"
                                class="text-error"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    onDelete?.();
                                }}
                            >
                                <Trash2 class="w-4 h-4" /> Delete
                            </button>
                        </li>
                    </ul>
                </div>
            </div>

            <!-- Description -->
            {#if goal.description}
                <p class="text-xs text-base-content/60 line-clamp-2">
                    {goal.description}
                </p>
            {/if}

            <!-- Progress Bar (for measurable) -->
            {#if goal.target}
                <div class="space-y-1">
                    <div class="flex items-center justify-between text-xs">
                        <span class="font-medium">
                            {(goal.stats?.current_value || 0).toLocaleString()} /
                            {goal.target.value.toLocaleString()}
                            {goal.target.unit_id}
                        </span>
                        <span
                            class="font-bold"
                            style="color: {goal.category?.color || '#6b7280'};"
                            >{progress}%</span
                        >
                    </div>
                    <div
                        class="w-full h-2 bg-base-200 rounded-full overflow-hidden"
                    >
                        <div
                            class="h-full rounded-full transition-all duration-500"
                            style="width: {progress}%; background-color: {goal
                                .category?.color || '#6b7280'};"
                        ></div>
                    </div>
                </div>
            {/if}

            <!-- Footer: Recurrence, Streak, Deadline -->
            <div
                class="flex items-center justify-between pt-1 border-t border-base-200/60"
            >
                <div class="flex items-center gap-3">
                    {#if isRecurring}
                        <div class="flex items-center gap-1 text-xs opacity-60">
                            <Repeat class="w-3 h-3" />
                            {formatRecurrence(goal.recurrence)}
                        </div>
                    {/if}
                    {#if goal.current_streak > 0}
                        <div
                            class="flex items-center gap-1 text-xs text-warning font-medium"
                        >
                            <Flame class="w-3 h-3" />
                            {goal.current_streak}d streak
                        </div>
                    {/if}
                </div>

                {#if goal.deadline}
                    <div class="flex items-center gap-1 text-xs opacity-60">
                        <Calendar class="w-3 h-3" />
                        {formatDeadline(goal.deadline)}
                    </div>
                {/if}
            </div>
        </div>
    </div>
{:else}
    <!-- Compact variant for lists -->
    <div
        class="flex items-center gap-3 p-3 rounded-xl border border-base-200/60 hover:border-base-300 hover:bg-base-100 transition-all cursor-pointer group"
        role="button"
        onclick={() => onClick?.()}
        in:fly={{
            x: -8,
            duration: 200,
            delay: transitionDelay,
            easing: cubicOut,
        }}
    >
        <!-- Icon -->
        <div
            class="w-9 h-9 rounded-lg flex items-center justify-center text-lg shrink-0"
            style="background-color: {goal.category?.color || '#6b7280'}20;"
        >
            {goal.icon || "🎯"}
        </div>

        <!-- Content -->
        <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
                <h4 class="font-semibold text-sm truncate">{goal.title}</h4>
                {#if goal.current_streak > 0}
                    <div
                        class="flex items-center gap-0.5 text-[10px] text-warning font-bold"
                    >
                        <Flame class="w-3 h-3" />
                        {goal.current_streak}
                    </div>
                {/if}
            </div>
            <div class="flex items-center gap-2 mt-0.5">
                {#if isRecurring}
                    <span class="text-[10px] opacity-50"
                        >{formatRecurrence(goal.recurrence)}</span
                    >
                {/if}
                {#if goal.target}
                    <span class="text-[10px] opacity-50">
                        {goal.stats?.current_value || 0}/{goal.target.value}
                        {goal.target.unit_id}
                    </span>
                {/if}
            </div>
        </div>

        <!-- Progress or Status -->
        {#if goal.target}
            <div class="w-12">
                <div
                    class="radial-progress text-xs"
                    style="--value:{progress}; --size:2rem; --thickness:3px; color: {goal
                        .category?.color || '#6b7280'};"
                    role="progressbar"
                >
                    {progress}%
                </div>
            </div>
        {:else}
            <span class="badge badge-sm {statusColors[goal.status]}"
                >{goal.status}</span
            >
        {/if}

        <!-- Arrow -->
        <ChevronRight
            class="w-4 h-4 opacity-0 group-hover:opacity-50 transition-opacity"
        />
    </div>
{/if}
