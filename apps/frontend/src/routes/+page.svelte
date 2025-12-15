<script lang="ts">
  import { Calendar, Target, ListTodo, Plus, Zap, Smile, Flame } from 'lucide-svelte';
  import { Timeline } from '$lib/components/timeline';

  // Demo tasks with overlapping times
  const demoTasks = [
    { id: '1', title: 'Morning Workout', startHour: 7, endHour: 8, color: 'primary' as const, emoji: '💪' },
    { id: '2', title: 'Breakfast', startHour: 8, endHour: 9, color: 'accent' as const, emoji: '🍳' },
    { id: '3', title: 'Work Session', startHour: 9, endHour: 12, color: 'secondary' as const, emoji: '💻' },
    { id: '4', title: 'Podcast', startHour: 9, endHour: 10, color: 'info' as const, emoji: '🎧' },
    { id: '5', title: 'Lunch Break', startHour: 12, endHour: 13, color: 'success' as const, emoji: '🍽️' },
    { id: '6', title: 'Reading', startHour: 14, endHour: 15, color: 'warning' as const, emoji: '📚' },
  ];

  const quickLogs = [
    { emoji: '☕', label: 'Morning' },
    { emoji: '💪', label: 'Workout' },
    { emoji: '📚', label: 'Reading' },
    { emoji: '🧘', label: 'Meditation' },
    { emoji: '🍽️', label: 'Meal' },
    { emoji: '💤', label: 'Sleep' },
  ];
</script>

<svelte:head>
  <title>Dashboard - Lucid Logs</title>
</svelte:head>

<div class="h-full flex flex-col lg:flex-row gap-4 lg:gap-6">
  <!-- Left Column: Stats + Quick Log -->
  <div class="lg:w-80 xl:w-96 flex-shrink-0 space-y-4">
    <!-- Welcome Card -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body p-4">
        <h1 class="text-xl sm:text-2xl font-extrabold text-gradient">Good Evening! ✨</h1>
        <p class="text-sm opacity-60">Ready to capture today's moments?</p>
        <button class="btn btn-primary btn-sm mt-3 w-full gap-2">
          <Plus class="w-4 h-4" />
          Start Logging
        </button>
      </div>
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-2 gap-3">
      <div class="card bg-base-100 shadow-sm p-3 sm:p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-[10px] sm:text-xs font-medium uppercase opacity-50">Streak</p>
            <p class="text-xl sm:text-2xl font-extrabold text-primary">7</p>
          </div>
          <div class="icon-box icon-box-primary">
            <Flame class="w-4 h-4 sm:w-5 sm:h-5" />
          </div>
        </div>
        <p class="text-[10px] sm:text-xs opacity-40 mt-1">days in a row</p>
      </div>

      <div class="card bg-base-100 shadow-sm p-3 sm:p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-[10px] sm:text-xs font-medium uppercase opacity-50">Tasks</p>
            <p class="text-xl sm:text-2xl font-extrabold text-secondary">5</p>
          </div>
          <div class="icon-box icon-box-secondary">
            <ListTodo class="w-4 h-4 sm:w-5 sm:h-5" />
          </div>
        </div>
        <p class="text-[10px] sm:text-xs opacity-40 mt-1">3 completed</p>
      </div>

      <div class="card bg-base-100 shadow-sm p-3 sm:p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-[10px] sm:text-xs font-medium uppercase opacity-50">Goals</p>
            <p class="text-xl sm:text-2xl font-extrabold text-accent">68%</p>
          </div>
          <div class="icon-box icon-box-accent">
            <Target class="w-4 h-4 sm:w-5 sm:h-5" />
          </div>
        </div>
        <p class="text-[10px] sm:text-xs opacity-40 mt-1">this week</p>
      </div>

      <div class="card bg-base-100 shadow-sm p-3 sm:p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-[10px] sm:text-xs font-medium uppercase opacity-50">Mood</p>
            <p class="text-xl sm:text-2xl font-extrabold text-success">😊</p>
          </div>
          <div class="icon-box icon-box-success">
            <Smile class="w-4 h-4 sm:w-5 sm:h-5" />
          </div>
        </div>
        <p class="text-[10px] sm:text-xs opacity-40 mt-1">avg today</p>
      </div>
    </div>

    <!-- Quick Log -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body p-4">
        <div class="flex items-center gap-2 mb-3">
          <div class="icon-box icon-box-warning">
            <Zap class="w-4 h-4" />
          </div>
          <h3 class="font-bold text-sm">Quick Log</h3>
        </div>
        <div class="grid grid-cols-3 gap-2">
          {#each quickLogs as log}
            <button class="btn btn-outline btn-sm flex-col h-auto py-2 sm:py-3 gap-0.5 sm:gap-1">
              <span class="text-lg sm:text-xl">{log.emoji}</span>
              <span class="text-[9px] sm:text-[10px] opacity-70">{log.label}</span>
            </button>
          {/each}
        </div>
      </div>
    </div>
  </div>

  <!-- Right Column: Timeline -->
  <div class="flex-1 min-w-0">
    <div class="card bg-base-100 shadow-sm h-full">
      <div class="card-body p-4 lg:p-6 flex flex-col">
        <div class="flex items-center justify-between mb-4 flex-shrink-0">
          <div class="flex items-center gap-2">
            <div class="icon-box icon-box-info">
              <Calendar class="w-4 h-4" />
            </div>
            <h3 class="font-bold text-sm">Today's Timeline</h3>
          </div>
          <button class="btn btn-ghost btn-sm gap-1">
            <Plus class="w-4 h-4" />
            <span class="hidden sm:inline">Add</span>
          </button>
        </div>
        
        <div class="flex-1 overflow-y-auto min-h-0">
          <Timeline tasks={demoTasks} startHour={6} endHour={18} />
        </div>
      </div>
    </div>
  </div>
</div>
