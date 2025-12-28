<script lang="ts">
  import { goto } from "$app/navigation";
  import { Mail, Lock, CircleAlert, UserPlus, ArrowRight } from "lucide-svelte";
  import { register } from "$lib/api";
  import { authStore } from "$lib/stores/auth.svelte";
  import { cn } from "$lib/utils";

  let email = $state("");
  let password = $state("");
  let confirmPassword = $state("");
  let loading = $state(false);
  let error = $state("");

  const passwordsMatch = $derived(
    confirmPassword === "" || password === confirmPassword,
  );

  async function handleRegister() {
    loading = true;
    error = "";
    if (password !== confirmPassword) {
      error = "Passwords do not match";
      loading = false;
      return;
    }
    if (password.length < 6) {
      error = "Password must be at least 6 characters";
      loading = false;
      return;
    }
    try {
      const response = await register({ username: email, password });
      authStore.loginWithResponse(response);
      await goto("/", { replaceState: true });
    } catch (err: unknown) {
      if (err && typeof err === "object" && "response" in err) {
        const httpErr = err as { response?: { status?: number } };
        if (httpErr.response?.status === 409) {
          error = "An account with this email already exists";
        } else {
          error = "Registration failed. Please try again.";
        }
      } else {
        error = "Network error. Please check your connection.";
      }
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>Register - Lucid Logs</title>
</svelte:head>

<div
  class="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-base-100 via-base-100 to-base-200"
>
  <!-- Decorative Elements -->
  <div class="absolute inset-0 overflow-hidden pointer-events-none">
    <div
      class="absolute -top-40 -left-40 w-80 h-80 bg-primary/10 rounded-full blur-3xl"
    ></div>
    <div
      class="absolute -bottom-40 -right-40 w-80 h-80 bg-primary/5 rounded-full blur-3xl"
    ></div>
  </div>

  <div class="card w-full max-w-md bg-base-100 shadow-2xl relative z-10">
    <div class="card-body">
      <!-- Header -->
      <div class="text-center mb-4">
        <div
          class="w-14 h-14 mx-auto rounded-2xl bg-gradient-to-br from-primary/80 to-primary flex items-center justify-center shadow-lg shadow-primary/20 mb-4"
        >
          <UserPlus class="w-7 h-7 text-primary-content" />
        </div>
        <h2 class="card-title text-2xl font-bold justify-center">
          Create Account
        </h2>
        <p class="text-base-content/60">Start your personal journey today</p>
      </div>

      <!-- Error Alert -->
      {#if error}
        <div class="alert alert-error">
          <CircleAlert class="h-4 w-4" />
          <span>{error}</span>
        </div>
      {/if}

      <!-- Form -->
      <form
        class="space-y-4"
        onsubmit={(e) => {
          e.preventDefault();
          handleRegister();
        }}
      >
        <div class="form-control">
          <label class="label" for="email">
            <span class="label-text">Email</span>
          </label>
          <label class="input input-bordered flex items-center gap-2 w-full">
            <Mail class="h-4 w-4 opacity-50" />
            <input
              id="email"
              type="email"
              placeholder="you@example.com"
              class="grow"
              bind:value={email}
              required
            />
          </label>
        </div>

        <div class="form-control">
          <label class="label" for="password">
            <span class="label-text">Password</span>
          </label>
          <label class="input input-bordered flex items-center gap-2 w-full">
            <Lock class="h-4 w-4 opacity-50" />
            <input
              id="password"
              type="password"
              placeholder="At least 6 characters"
              class="grow"
              bind:value={password}
              required
              minlength={6}
            />
          </label>
        </div>

        <div class="form-control">
          <label class="label" for="confirmPassword">
            <span class="label-text">Confirm Password</span>
          </label>
          <label
            class={cn(
              "input input-bordered flex items-center gap-2 w-full",
              !passwordsMatch && "input-error",
            )}
          >
            <Lock class="h-4 w-4 opacity-50" />
            <input
              id="confirmPassword"
              type="password"
              placeholder="Re-enter your password"
              class="grow"
              bind:value={confirmPassword}
              required
            />
          </label>
          {#if !passwordsMatch}
            <p class="label">
              <span class="label-text-alt text-error"
                >Passwords don't match</span
              >
            </p>
          {/if}
        </div>

        <button
          type="submit"
          class="btn btn-primary w-full"
          disabled={loading || !passwordsMatch}
        >
          {#if loading}
            <span class="loading loading-spinner loading-sm"></span>
            Creating account...
          {:else}
            Create Account
            <ArrowRight class="h-4 w-4" />
          {/if}
        </button>
      </form>

      <p class="text-center text-xs opacity-60 mt-2">
        By creating an account, you agree to our
        <a href="/terms" class="link link-primary">Terms of Service</a>
      </p>

      <!-- Divider -->
      <div class="divider"></div>

      <!-- Footer -->
      <p class="text-center text-sm opacity-60">
        Already have an account?
        <a href="/login" class="link link-primary font-medium">Sign in</a>
      </p>
    </div>
  </div>
</div>
