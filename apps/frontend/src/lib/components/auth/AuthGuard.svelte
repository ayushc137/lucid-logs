<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { authStore } from '$lib/stores/auth.svelte';

  let { children } = $props();

  let ready = $state(false);

  onMount(async () => {
    // Check if we have a token
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

    ready = true;
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
