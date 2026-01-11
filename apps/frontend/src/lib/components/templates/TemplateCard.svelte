<script lang="ts">
    import type { TaskTemplate } from "$lib/api";
    import {
        Zap,
        Clock,
        Hash,
        ChevronRight,
        MoreVertical,
        Pencil,
        Trash2,
        Play,
        Target,
    } from "lucide-svelte";
    import { cn } from "$lib/utils";
    import { fly } from "svelte/transition";
    import { cubicOut } from "svelte/easing";

    interface Props {
        template: TaskTemplate;
        variant?: "card" | "compact";
        transitionDelay?: number;
        onEdit?: () => void;
        onDelete?: () => void;
        onUse?: () => void;
        onClick?: () => void;
    }

    let {
        template,
        variant = "card",
        transitionDelay = 0,
        onEdit,
        onDelete,
        onUse,
        onClick,
    }: Props = $props();

    function formatDuration(seconds: number | undefined): string {
        if (!seconds) return "";
        const mins = Math.floor(seconds / 60);
        if (mins < 60) return `${mins}m`;
        const hrs = Math.floor(mins / 60);
        const remMins = mins % 60;
        return remMins > 0 ? `${hrs}h ${remMins}m` : `${hrs}h`;
    }

    function formatLastUsed(date: string | undefined): string {
        if (!date) return "Never used";
        const d = new Date(date);
        const now = new Date();
        const diff = Math.floor(
            (now.getTime() - d.getTime()) / (1000 * 60 * 60 * 24),
        );
        if (diff === 0) return "Used today";
        if (diff === 1) return "Used yesterday";
        if (diff < 7) return `Used ${diff}d ago`;
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
        class="card bg-base-100 border-2 border-base-200/80 hover:border-secondary/40 shadow-sm hover:shadow-md transition-all duration-200 cursor-pointer group"
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
            style="background-color: {template.category?.color || '#10b981'};"
        ></div>

        <div class="card-body p-4 gap-3">
            <!-- Header: Icon, Title, Menu -->
            <div class="flex items-start gap-3">
                <!-- Icon -->
                <div
                    class="w-10 h-10 rounded-xl flex items-center justify-center text-xl shrink-0"
                    style="background-color: {template.category?.color ||
                        '#10b981'}20;"
                >
                    {template.icon || "⚡"}
                </div>

                <!-- Title & Meta -->
                <div class="flex-1 min-w-0">
                    <h3 class="font-semibold text-sm truncate">
                        {template.title}
                    </h3>
                    <div class="flex items-center gap-2 mt-0.5 flex-wrap">
                        {#if template.is_quick_log}
                            <span class="badge badge-sm badge-secondary gap-1">
                                <Zap class="w-3 h-3" />
                                Quick Log
                            </span>
                        {/if}
                        {#if template.goals && template.goals.length > 0}
                            <span class="badge badge-sm badge-accent gap-1">
                                <Target class="w-3 h-3" />
                                Goal Linked
                            </span>
                        {/if}
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
                                    onUse?.();
                                }}
                            >
                                <Play class="w-4 h-4" /> Use Template
                            </button>
                        </li>
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
            {#if template.description}
                <p class="text-xs text-base-content/60 line-clamp-2">
                    {template.description}
                </p>
            {/if}

            <!-- Stats Row -->
            <div class="flex items-center gap-4 text-xs">
                {#if template.default_duration}
                    <div class="flex items-center gap-1 opacity-60">
                        <Clock class="w-3 h-3" />
                        {formatDuration(template.default_duration)}
                    </div>
                {/if}
                {#if template.quantity_enabled}
                    <div class="flex items-center gap-1 opacity-60">
                        <Hash class="w-3 h-3" />
                        {template.quantity_default || 0}
                        {template.goals?.[0]?.target?.unit_id || ""}
                    </div>
                {/if}
            </div>

            <!-- Footer: Usage & Actions -->
            <div
                class="flex items-center justify-between pt-1 border-t border-base-200/60"
            >
                <div class="flex items-center gap-2">
                    <span class="text-xs opacity-50">
                        {template.use_count} uses · {formatLastUsed(
                            template.last_used_at,
                        )}
                    </span>
                </div>

                <!-- Quick Use Button -->
                <button
                    type="button"
                    class="btn btn-secondary btn-xs gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                    onclick={(e) => {
                        e.stopPropagation();
                        onUse?.();
                    }}
                >
                    <Play class="w-3 h-3" />
                    Use
                </button>
            </div>
        </div>
    </div>
{:else}
    <!-- Compact variant for lists -->
    <div
        class="flex items-center gap-3 p-3 rounded-xl border border-base-200/60 hover:border-secondary/40 hover:bg-base-100 transition-all cursor-pointer group"
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
            style="background-color: {template.category?.color || '#10b981'}20;"
        >
            {template.icon || "⚡"}
        </div>

        <!-- Content -->
        <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
                <h4 class="font-semibold text-sm truncate">{template.title}</h4>
                {#if template.is_quick_log}
                    <Zap class="w-3 h-3 text-secondary" />
                {/if}
            </div>
            <div class="flex items-center gap-2 mt-0.5">
                {#if template.default_duration}
                    <span
                        class="text-[10px] opacity-50 flex items-center gap-0.5"
                    >
                        <Clock class="w-2.5 h-2.5" />
                        {formatDuration(template.default_duration)}
                    </span>
                {/if}
                {#if template.quantity_enabled}
                    <span class="text-[10px] opacity-50">
                        {template.quantity_default || ""}
                        {template.goals?.[0]?.target?.unit_id || ""}
                    </span>
                {/if}
                <span class="text-[10px] opacity-40"
                    >{template.use_count} uses</span
                >
            </div>
        </div>

        <!-- Use Button -->
        <button
            type="button"
            class="btn btn-secondary btn-xs btn-circle opacity-0 group-hover:opacity-100 transition-opacity"
            onclick={(e) => {
                e.stopPropagation();
                onUse?.();
            }}
            title="Use template"
        >
            <Play class="w-3.5 h-3.5" />
        </button>

        <!-- Arrow -->
        <ChevronRight
            class="w-4 h-4 opacity-0 group-hover:opacity-50 transition-opacity"
        />
    </div>
{/if}
