<script lang="ts">
    import TimelineGantt from "./TimelineGantt.svelte";
    import TimelineAgenda from "./TimelineAgenda.svelte";
    import type { TimelineTask } from "./types";

    export type TimelineView = "timeline" | "agenda";

    interface Props {
        tasks?: TimelineTask[];
        selectedDate?: Date;
        onTaskClick?: (taskId: string) => void;
        onToggleComplete?: (taskId: string, completed: boolean) => void;
        onCategoryClick?: (categoryId: string) => void;
        onDateChange?: (date: Date) => void;
        onTaskTimeUpdate?: (
            taskId: string,
            startTime: Date,
            endTime: Date,
        ) => void;
        editMode?: boolean;
        onEditModeChange?: (enabled: boolean) => void;
        currentView?: TimelineView;
    }

    let {
        tasks = [],
        selectedDate = new Date(),
        onTaskClick,
        onToggleComplete,
        onCategoryClick,
        onDateChange,
        onTaskTimeUpdate,
        editMode = false,
        onEditModeChange,
        currentView = "timeline",
    }: Props = $props();

    const timelineProps = $derived({
        tasks,
        selectedDate,
        onTaskClick,
        onToggleComplete,
        onCategoryClick,
        onDateChange,
        onTaskTimeUpdate,
        editMode,
        onEditModeChange,
    });
</script>

<div class="h-full">
    {#if currentView === "timeline"}
        <TimelineGantt {...timelineProps} />
    {:else if currentView === "agenda"}
        <TimelineAgenda {...timelineProps} />
    {/if}
</div>
