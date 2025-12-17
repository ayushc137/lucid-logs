<script lang="ts">
  import { Clock } from "lucide-svelte";

  interface Task {
    id: string;
    title: string;
    startHour: number;
    endHour: number;
    color: "primary" | "secondary" | "accent" | "info" | "success" | "warning";
    emoji?: string;
  }

  interface Props {
    tasks?: Task[];
    startHour?: number;
    endHour?: number;
  }

  let { tasks = [], startHour = 6, endHour = 22 }: Props = $props();

  // Generate hour slots
  const hours = Array.from(
    { length: endHour - startHour + 1 },
    (_, i) => startHour + i,
  );

  // Format hour display
  function formatHour(hour: number): string {
    if (hour === 0) return "12 AM";
    if (hour === 12) return "12 PM";
    if (hour < 12) return `${hour} AM`;
    return `${hour - 12} PM`;
  }

  // Get tasks for a specific hour
  function getTasksForHour(hour: number): Task[] {
    return tasks.filter((t) => t.startHour <= hour && t.endHour > hour);
  }

  // Calculate current time position
  const now = new Date();
  const currentHour = now.getHours() + now.getMinutes() / 60;
  const nowPosition =
    ((currentHour - startHour) / (endHour - startHour + 1)) * 100;
  const showNowLine = currentHour >= startHour && currentHour <= endHour + 1;
</script>

<div class="timeline-container relative">
  <!-- Current time indicator -->
  {#if showNowLine}
    <div
      class="absolute left-0 right-0 h-0.5 bg-error z-10 pointer-events-none"
      style="top: {nowPosition}%"
    >
      <span
        class="absolute left-0 -translate-x-full pr-2 text-[10px] font-semibold text-error whitespace-nowrap -translate-y-1/2"
      >
        Now
      </span>
      <div
        class="absolute left-0 w-2 h-2 rounded-full bg-error -translate-y-1/2"
      ></div>
    </div>
  {/if}

  <!-- Timeline grid -->
  <div class="timeline-grid">
    {#each hours as hour}
      {@const hourTasks = getTasksForHour(hour)}

      <!-- Hour label -->
      <div class="timeline-hour">
        {formatHour(hour)}
      </div>

      <!-- Task row -->
      <div class="timeline-row">
        {#if hourTasks.length > 0}
          {#each hourTasks as task}
            <div
              class="timeline-task timeline-task-{task.color}"
              role="button"
              tabindex="0"
            >
              {#if task.emoji}
                <span>{task.emoji}</span>
              {/if}
              <span class="truncate">{task.title}</span>
            </div>
          {/each}
        {:else}
          <div class="flex-1 opacity-0">-</div>
        {/if}
      </div>
    {/each}
  </div>

  <!-- Empty state -->
  {#if tasks.length === 0}
    <div
      class="absolute inset-0 flex flex-col items-center justify-center text-center pointer-events-none"
    >
      <div
        class="w-12 h-12 rounded-xl bg-info/10 flex items-center justify-center mb-4"
      >
        <Clock class="w-6 h-6 text-info" />
      </div>
      <h4 class="text-lg font-semibold opacity-50">No logs yet today</h4>
      <p class="text-sm opacity-30 mt-1">Add your first log to see it here</p>
    </div>
  {/if}
</div>
