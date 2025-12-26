<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import { authStore, isDevAuthBypassEnabled } from "$lib/stores/auth.svelte";

  let { children } = $props();

  let ready = $state(false);

  onMount(async () => {
    // Check if dev auth bypass is enabled
    if (isDevAuthBypassEnabled()) {
      // Always perform fresh login in dev mode to handle database resets
      // This ensures we always have a valid token for the current database state
      const success = await authStore.devAutoLogin();
      if (!success) {
        console.error(
          "[Dev Auth] Auto-login failed, redirecting to login page",
        );
        goto("/login");
        return;
      }
      ready = true;
      return;
    }

    // Normal auth flow
    if (!authStore.hasToken) {
      goto("/login");
      return;
    }

    // Validate token and load user
    const isValid = await authStore.initialize();
    if (!isValid) {
      goto("/login");
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
