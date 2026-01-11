<script lang="ts">
import { goto } from '$app/navigation';
import { enableAuthRedirect } from '$lib/api/client';
import { authStore, isDevAuthBypassEnabled } from '$lib/stores/auth.svelte';
import { emotionStore } from '$lib/stores/emotions.svelte';
import { onMount } from 'svelte';

let { children } = $props();

let ready = $state(false);

// Prevent concurrent login attempts
let isLoggingIn = false;

onMount(async () => {
	// Prevent concurrent login attempts
	if (isLoggingIn) return;
	isLoggingIn = true;

	try {
		// Check if dev auth bypass is enabled
		if (isDevAuthBypassEnabled()) {
			// Only auto-login if we don't already have a valid token
			if (!authStore.hasToken) {
				const success = await authStore.devAutoLogin();
				if (!success) {
					console.error(
						'[Dev Auth] Auto-login failed, redirecting to login page',
					);
					goto('/login');
					return;
				}
			}
			// Initialize emotion store after auth is confirmed
			await emotionStore.init();
			// Enable 401 redirects now that auth is confirmed
			enableAuthRedirect();
			ready = true;
			return;
		}

		// Normal auth flow
		if (!authStore.hasToken) {
			goto('/login');
			return;
		}

		// Validate token and load user
		const isValid = await authStore.initialize();
		if (!isValid) {
			goto('/login');
			return;
		}

		// Initialize emotion store after auth is confirmed
		await emotionStore.init();
		// Enable 401 redirects now that auth is confirmed
		enableAuthRedirect();
		ready = true;
	} finally {
		isLoggingIn = false;
	}
});
</script>

{#if ready}
  {@render children()}
{:else}
  <!-- Loading state while checking auth -->
  <div class="min-h-screen flex items-center justify-center bg-base-200">
    <div class="flex flex-col items-center gap-3">
      <span class="loading loading-spinner loading-lg text-primary"></span>
      <p class="text-base-content/60 text-sm">Loading...</p>
    </div>
  </div>
{/if}
