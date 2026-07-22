<script lang="ts">
import {
	getUserPreferences,
	updateUserPreferences,
	getAIModels,
	getAIDefaults,
	type AISettings,
} from '$lib/api';
import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
import { Bot, Check, Save } from 'lucide-svelte';

const queryClient = useQueryClient();

const prefsQuery = createQuery({
	queryKey: ['user-preferences'],
	queryFn: () => getUserPreferences(),
});

// Form state
let provider = $state('openrouter');
let baseUrl = $state('');
let model = $state('');
let apiKey = $state('');
let enabled = $state(false);
let loaded = $state(false);

// Model auto-fetch state
let availableModels = $state<string[]>([]);
let loadingModels = $state(false);
let modelsError = $state('');
let lastFetchedProvider = $state('');
let lastFetchedBaseUrl = $state<string | undefined>(undefined);

// Populate from fetched data
$effect(() => {
	const data = $prefsQuery.data;
	if (data && !loaded) {
		const ai = data.preferences?.ai;
		if (ai) {
			provider = ai.provider || 'openrouter';
			baseUrl = ai.base_url || '';
			model = ai.model || '';
			enabled = ai.enabled ?? false;
		} else {
			// No AI config — try env defaults
			fetchEnvDefaults();
		}
		loaded = true;
	}
});

// Fetch models when provider or baseUrl changes
$effect(() => {
	// Read reactive deps so the effect re-tracks them
	const p = provider;
	const b = provider === 'custom' ? baseUrl : undefined;
	if (!loaded) return;
	// Only refetch if something actually changed
	if (p === lastFetchedProvider && b === lastFetchedBaseUrl) return;
	fetchModels();
});

async function fetchModels() {
	loadingModels = true;
	modelsError = '';
	try {
		const res = await getAIModels();
		availableModels = res.models;
		lastFetchedProvider = provider;
		lastFetchedBaseUrl = provider === 'custom' ? baseUrl : undefined;
	} catch (e) {
		modelsError = 'Could not fetch models';
		availableModels = [];
	} finally {
		loadingModels = false;
	}
}

async function fetchEnvDefaults() {
	try {
		const res = await getAIDefaults();
		if (res.provider) {
			provider = res.provider;
			if (res.base_url) baseUrl = res.base_url;
			if (res.model) model = res.model;
			enabled = true;
		}
	} catch {
		// Env defaults not configured — silently ignore
	}
}

const saveMutation = createMutation({
	mutationFn: (ai: AISettings) => updateUserPreferences({ ai }),
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ['user-preferences'] });
	},
});

let saved = $state(false);

function handleSave() {
	const ai: AISettings = {
		enabled,
		provider,
		model,
		...(provider === 'custom' ? { base_url: baseUrl } : {}),
		...(apiKey ? { api_key: apiKey } : {}), // don't send empty key — preserves existing
	};
	$saveMutation.mutate(ai, {
		onSuccess: () => {
			saved = true;
			apiKey = ''; // clear after save
			setTimeout(() => (saved = false), 2000);
		},
	});
}

const providerPresets = [
	{ value: 'openrouter', label: 'OpenRouter', modelPlaceholder: 'deepseek/deepseek-chat' },
	{ value: 'openai', label: 'OpenAI', modelPlaceholder: 'gpt-4o-mini' },
	{ value: 'gemini', label: 'Google Gemini', modelPlaceholder: 'gemini-2.0-flash' },
	{ value: 'ollama', label: 'Ollama (Local)', modelPlaceholder: 'llama3.1' },
	{ value: 'custom', label: 'Custom', modelPlaceholder: 'model-name' },
];

const currentPreset = $derived(providerPresets.find((p) => p.value === provider) ?? providerPresets[0]);
const hasKey = $derived($prefsQuery.data?.preferences?.ai?.has_key ?? false);
</script>

<div class="bg-base-100 rounded-box border border-base-content/5 p-4 sm:p-5">
	<div class="flex items-center gap-3 mb-4">
		<div class="w-10 h-10 rounded-box bg-primary/10 flex items-center justify-center">
			<Bot class="w-5 h-5 text-primary" />
		</div>
		<div>
			<h2 class="text-sm font-semibold">AI Insights</h2>
			<p class="text-xs text-base-content/60">Configure AI-powered retrospective insights</p>
		</div>
	</div>

	{#if $prefsQuery.isPending}
		<div class="flex justify-center py-6">
			<span class="loading loading-spinner loading-sm text-primary"></span>
		</div>
	{:else}
		<div class="flex flex-col gap-4">
			<!-- Enable toggle -->
			<label class="flex items-center justify-between cursor-pointer">
				<div>
					<span class="text-sm font-medium">Enable AI Insights</span>
					<p class="text-xs text-base-content/50">Generate AI-powered analysis in retrospectives</p>
				</div>
				<input type="checkbox" class="toggle toggle-primary" bind:checked={enabled} />
			</label>

			<div class="h-px bg-base-content/5"></div>

			<!-- Provider -->
			<label class="form-control">
				<span class="label-text font-medium text-xs text-base-content/60 mb-1">Provider</span>
				<select class="select select-bordered select-sm" bind:value={provider}>
					{#each providerPresets as preset}
						<option value={preset.value}>{preset.label}</option>
					{/each}
				</select>
			</label>

			<!-- Base URL (custom only) -->
			{#if provider === 'custom'}
				<label class="form-control">
					<span class="label-text font-medium text-xs text-base-content/60 mb-1">Base URL</span>
					<input
						type="url"
						class="input input-bordered input-sm"
						placeholder="https://api.example.com/v1"
						bind:value={baseUrl}
					/>
				</label>
			{/if}

			<!-- Model -->
			<label class="form-control">
				<span class="label-text font-medium text-xs text-base-content/60 mb-1">Model</span>
				{#if availableModels.length > 0}
					<select class="select select-bordered select-sm" bind:value={model}>
						<option value="" disabled>Select a model</option>
						{#each availableModels as m}
							<option value={m}>{m}</option>
						{/each}
					</select>
				{:else}
					<input
						type="text"
						class="input input-bordered input-sm"
						placeholder={currentPreset.modelPlaceholder}
						bind:value={model}
					/>
				{/if}
				{#if loadingModels}
					<span class="text-xs text-base-content/40 mt-1 flex items-center gap-1">
						<span class="loading loading-spinner loading-xs"></span>
						Fetching available models…
					</span>
				{:else if modelsError}
					<span class="text-xs text-base-content/40 mt-1">{modelsError}</span>
				{/if}
			</label>

			<!-- API Key -->
			<label class="form-control">
				<span class="label-text font-medium text-xs text-base-content/60 mb-1">API Key</span>
				<input
					type="password"
					class="input input-bordered input-sm"
					placeholder={hasKey ? '•••••••• configured' : 'Enter your API key'}
					bind:value={apiKey}
				/>
				{#if hasKey}
					<span class="text-xs text-success mt-1 flex items-center gap-1">
						<Check class="w-3 h-3" /> Key configured — leave blank to keep existing
					</span>
				{:else}
					<span class="text-xs text-base-content/40 mt-1">Your key is stored securely and never shown again</span>
				{/if}
			</label>

			<!-- Save -->
			<div class="flex items-center gap-2">
				<button
					class="btn btn-primary btn-sm gap-2"
					disabled={$saveMutation.isPending}
					onclick={handleSave}
				>
					{#if $saveMutation.isPending}
						<span class="loading loading-spinner loading-xs"></span>
					{:else if saved}
						<Check class="w-4 h-4" />
					{:else}
						<Save class="w-4 h-4" />
					{/if}
					{saved ? 'Saved' : 'Save'}
				</button>
				{#if $saveMutation.isError}
					<span class="text-xs text-error">Failed to save</span>
				{/if}
			</div>
		</div>
	{/if}
</div>
