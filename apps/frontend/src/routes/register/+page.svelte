<script lang="ts">
  import { goto } from '$app/navigation';
  import { X } from 'lucide-svelte';

  let email = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let loading = $state(false);
  let error = $state('');

  async function handleRegister() {
    loading = true;
    error = '';
    if (password !== confirmPassword) {
      error = 'Passwords do not match';
      loading = false;
      return;
    }
    try {
      await new Promise((r) => setTimeout(r, 1000));
      goto('/login');
    } catch {
      error = 'Registration failed';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>Register - Lucid Logs</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center p-4 bg-base-200">
  <div class="card bg-base-100 w-full max-w-sm border border-base-300">
    <div class="card-body">
      <div class="text-center mb-4">
        <div class="w-14 h-14 mx-auto rounded-xl bg-primary flex items-center justify-center">
          <span class="text-primary-content font-bold text-xl">LL</span>
        </div>
        <h1 class="text-xl font-bold mt-3">Create Account</h1>
        <p class="text-base-content/60 text-sm">Start your journey</p>
      </div>

      {#if error}
        <div class="alert alert-error text-sm py-2">
          <X class="w-4 h-4" />
          <span>{error}</span>
        </div>
      {/if}

      <form class="space-y-3" onsubmit={(e) => { e.preventDefault(); handleRegister(); }}>
        <div class="form-control">
          <label class="label py-1" for="email"><span class="label-text">Email</span></label>
          <input id="email" type="email" bind:value={email} placeholder="you@example.com" class="input input-bordered input-sm" required />
        </div>

        <div class="form-control">
          <label class="label py-1" for="password"><span class="label-text">Password</span></label>
          <input id="password" type="password" bind:value={password} placeholder="••••••••" class="input input-bordered input-sm" required />
        </div>

        <div class="form-control">
          <label class="label py-1" for="confirmPassword"><span class="label-text">Confirm Password</span></label>
          <input id="confirmPassword" type="password" bind:value={confirmPassword} placeholder="••••••••" class="input input-bordered input-sm" required />
        </div>

        <button type="submit" class="btn btn-primary btn-sm w-full" disabled={loading}>
          {#if loading}<span class="loading loading-spinner loading-xs"></span>{:else}Create Account{/if}
        </button>
      </form>

      <div class="divider text-xs">or</div>
      <p class="text-center text-sm">Already have an account? <a href="/login" class="link link-primary">Sign in</a></p>
    </div>
  </div>
</div>
