<script lang="ts">
  import { goto } from '$app/navigation';
  import { X } from 'lucide-svelte';

  let email = $state('');
  let password = $state('');
  let loading = $state(false);
  let error = $state('');

  async function handleLogin() {
    loading = true;
    error = '';
    try {
      await new Promise((r) => setTimeout(r, 1000));
      goto('/');
    } catch {
      error = 'Invalid email or password';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>Login - Lucid Logs</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center p-4 bg-base-200">
  <div class="card bg-base-100 w-full max-w-sm border border-base-300">
    <div class="card-body">
      <div class="text-center mb-4">
        <div class="w-14 h-14 mx-auto rounded-xl bg-primary flex items-center justify-center">
          <span class="text-primary-content font-bold text-xl">LL</span>
        </div>
        <h1 class="text-xl font-bold mt-3">Welcome back</h1>
        <p class="text-base-content/60 text-sm">Sign in to continue</p>
      </div>

      {#if error}
        <div class="alert alert-error text-sm py-2">
          <X class="w-4 h-4" />
          <span>{error}</span>
        </div>
      {/if}

      <form class="space-y-3" onsubmit={(e) => { e.preventDefault(); handleLogin(); }}>
        <div class="form-control">
          <label class="label py-1" for="email"><span class="label-text">Email</span></label>
          <input id="email" type="email" bind:value={email} placeholder="you@example.com" class="input input-bordered input-sm" required />
        </div>

        <div class="form-control">
          <label class="label py-1" for="password"><span class="label-text">Password</span></label>
          <input id="password" type="password" bind:value={password} placeholder="••••••••" class="input input-bordered input-sm" required />
        </div>

        <button type="submit" class="btn btn-primary btn-sm w-full" disabled={loading}>
          {#if loading}<span class="loading loading-spinner loading-xs"></span>{:else}Sign In{/if}
        </button>
      </form>

      <div class="divider text-xs">or</div>
      <p class="text-center text-sm">Don't have an account? <a href="/register" class="link link-primary">Sign up</a></p>
    </div>
  </div>
</div>
