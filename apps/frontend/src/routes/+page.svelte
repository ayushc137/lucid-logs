<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { Calendar, Target, ListTodo, TrendingUp, Plus, Clock } from 'lucide-svelte';
	import { cn } from '$lib/utils';

	// Query hooks for dashboard data
	const todayDate = new Date().toISOString().split('T')[0];
</script>

<svelte:head>
	<title>Lucid Logs - Dashboard</title>
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

<div class="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-950 dark:to-slate-900">
	<!-- Header -->
	<header class="border-b border-slate-200 dark:border-slate-800 bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm sticky top-0 z-50">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex items-center justify-between h-16">
				<div class="flex items-center gap-3">
					<div class="w-8 h-8 rounded-lg bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center">
						<span class="text-white font-bold text-sm">LL</span>
					</div>
					<h1 class="text-xl font-semibold text-slate-900 dark:text-white">Lucid Logs</h1>
				</div>
				<nav class="hidden md:flex items-center gap-1">
					<a href="/" class="px-4 py-2 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-white font-medium text-sm">
						Dashboard
					</a>
					<a href="/tasks" class="px-4 py-2 rounded-lg text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 font-medium text-sm transition-colors">
						Tasks
					</a>
					<a href="/goals" class="px-4 py-2 rounded-lg text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 font-medium text-sm transition-colors">
						Goals
					</a>
					<a href="/analytics" class="px-4 py-2 rounded-lg text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 font-medium text-sm transition-colors">
						Analytics
					</a>
				</nav>
				<div class="flex items-center gap-2">
					<button class="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors">
						<Clock class="w-5 h-5 text-slate-600 dark:text-slate-400" />
					</button>
				</div>
			</div>
		</div>
	</header>

	<!-- Main Content -->
	<main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
		<!-- Welcome Section -->
		<div class="mb-8">
			<h2 class="text-3xl font-bold text-slate-900 dark:text-white mb-2">
				Good {new Date().getHours() < 12 ? 'morning' : new Date().getHours() < 17 ? 'afternoon' : 'evening'} 👋
			</h2>
			<p class="text-slate-600 dark:text-slate-400">
				Today is {new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })}
			</p>
		</div>

		<!-- Quick Stats Grid -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
			<!-- Tasks Today -->
			<div class="bg-white dark:bg-slate-900 rounded-2xl p-6 border border-slate-200 dark:border-slate-800 shadow-sm hover:shadow-md transition-shadow">
				<div class="flex items-center justify-between mb-4">
					<div class="p-2 rounded-lg bg-blue-100 dark:bg-blue-900/30">
						<ListTodo class="w-5 h-5 text-blue-600 dark:text-blue-400" />
					</div>
					<span class="text-sm text-slate-500 dark:text-slate-400">Today</span>
				</div>
				<p class="text-3xl font-bold text-slate-900 dark:text-white mb-1">0</p>
				<p class="text-sm text-slate-600 dark:text-slate-400">Tasks completed</p>
			</div>

			<!-- Goals Progress -->
			<div class="bg-white dark:bg-slate-900 rounded-2xl p-6 border border-slate-200 dark:border-slate-800 shadow-sm hover:shadow-md transition-shadow">
				<div class="flex items-center justify-between mb-4">
					<div class="p-2 rounded-lg bg-green-100 dark:bg-green-900/30">
						<Target class="w-5 h-5 text-green-600 dark:text-green-400" />
					</div>
					<span class="text-sm text-slate-500 dark:text-slate-400">Active</span>
				</div>
				<p class="text-3xl font-bold text-slate-900 dark:text-white mb-1">0</p>
				<p class="text-sm text-slate-600 dark:text-slate-400">Goals in progress</p>
			</div>

			<!-- Current Streak -->
			<div class="bg-white dark:bg-slate-900 rounded-2xl p-6 border border-slate-200 dark:border-slate-800 shadow-sm hover:shadow-md transition-shadow">
				<div class="flex items-center justify-between mb-4">
					<div class="p-2 rounded-lg bg-orange-100 dark:bg-orange-900/30">
						<TrendingUp class="w-5 h-5 text-orange-600 dark:text-orange-400" />
					</div>
					<span class="text-sm text-slate-500 dark:text-slate-400">Streak</span>
				</div>
				<p class="text-3xl font-bold text-slate-900 dark:text-white mb-1">0</p>
				<p class="text-sm text-slate-600 dark:text-slate-400">Days in a row</p>
			</div>

			<!-- Mood Today -->
			<div class="bg-white dark:bg-slate-900 rounded-2xl p-6 border border-slate-200 dark:border-slate-800 shadow-sm hover:shadow-md transition-shadow">
				<div class="flex items-center justify-between mb-4">
					<div class="p-2 rounded-lg bg-violet-100 dark:bg-violet-900/30">
						<Calendar class="w-5 h-5 text-violet-600 dark:text-violet-400" />
					</div>
					<span class="text-sm text-slate-500 dark:text-slate-400">Mood</span>
				</div>
				<p class="text-3xl font-bold text-slate-900 dark:text-white mb-1">😊</p>
				<p class="text-sm text-slate-600 dark:text-slate-400">How are you feeling?</p>
			</div>
		</div>

		<!-- Main Grid -->
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
			<!-- Today's Timeline -->
			<div class="lg:col-span-2 bg-white dark:bg-slate-900 rounded-2xl p-6 border border-slate-200 dark:border-slate-800 shadow-sm">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-lg font-semibold text-slate-900 dark:text-white">Today's Timeline</h3>
					<button class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-violet-100 dark:bg-violet-900/30 text-violet-600 dark:text-violet-400 font-medium text-sm hover:bg-violet-200 dark:hover:bg-violet-900/50 transition-colors">
						<Plus class="w-4 h-4" />
						Add Task
					</button>
				</div>

				<!-- Empty State -->
				<div class="flex flex-col items-center justify-center py-12 text-center">
					<div class="w-16 h-16 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center mb-4">
						<Calendar class="w-8 h-8 text-slate-400" />
					</div>
					<h4 class="text-lg font-medium text-slate-900 dark:text-white mb-2">No tasks yet</h4>
					<p class="text-sm text-slate-600 dark:text-slate-400 mb-4">Start logging your day by adding a task</p>
					<button class="flex items-center gap-2 px-4 py-2 rounded-lg bg-gradient-to-r from-violet-500 to-purple-600 text-white font-medium text-sm hover:from-violet-600 hover:to-purple-700 transition-all shadow-lg shadow-violet-500/25">
						<Plus class="w-4 h-4" />
						Create your first task
					</button>
				</div>
			</div>

			<!-- Quick Log Panel -->
			<div class="bg-white dark:bg-slate-900 rounded-2xl p-6 border border-slate-200 dark:border-slate-800 shadow-sm">
				<h3 class="text-lg font-semibold text-slate-900 dark:text-white mb-4">Quick Log</h3>
				<p class="text-sm text-slate-600 dark:text-slate-400 mb-4">Tap to log common activities</p>

				<!-- Quick Log Templates -->
				<div class="grid grid-cols-2 gap-2">
					{#each [
						{ icon: '💧', label: 'Water', color: 'bg-blue-50 dark:bg-blue-900/20' },
						{ icon: '🏃', label: 'Exercise', color: 'bg-green-50 dark:bg-green-900/20' },
						{ icon: '🧘', label: 'Meditate', color: 'bg-purple-50 dark:bg-purple-900/20' },
						{ icon: '📚', label: 'Reading', color: 'bg-amber-50 dark:bg-amber-900/20' },
						{ icon: '☕', label: 'Coffee', color: 'bg-orange-50 dark:bg-orange-900/20' },
						{ icon: '😴', label: 'Sleep', color: 'bg-indigo-50 dark:bg-indigo-900/20' }
					] as template}
						<button
							class={cn(
								'flex flex-col items-center gap-2 p-4 rounded-xl border border-slate-200 dark:border-slate-700 hover:border-violet-300 dark:hover:border-violet-700 transition-all hover:scale-105',
								template.color
							)}
						>
							<span class="text-2xl">{template.icon}</span>
							<span class="text-xs font-medium text-slate-700 dark:text-slate-300">{template.label}</span>
						</button>
					{/each}
				</div>
			</div>
		</div>

		<!-- Habits Section -->
		<div class="mt-6 bg-white dark:bg-slate-900 rounded-2xl p-6 border border-slate-200 dark:border-slate-800 shadow-sm">
			<div class="flex items-center justify-between mb-6">
				<h3 class="text-lg font-semibold text-slate-900 dark:text-white">Today's Habits</h3>
				<a href="/goals" class="text-sm text-violet-600 dark:text-violet-400 hover:text-violet-700 dark:hover:text-violet-300 font-medium">
					View all →
				</a>
			</div>

			<!-- Empty State -->
			<div class="flex flex-col items-center justify-center py-8 text-center">
				<div class="w-12 h-12 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center mb-3">
					<Target class="w-6 h-6 text-slate-400" />
				</div>
				<p class="text-sm text-slate-600 dark:text-slate-400">No habits configured yet</p>
				<a href="/goals" class="mt-2 text-sm text-violet-600 dark:text-violet-400 hover:underline">
					Set up your first habit
				</a>
			</div>
		</div>
	</main>
</div>
