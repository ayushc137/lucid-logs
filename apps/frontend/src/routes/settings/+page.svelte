<script lang="ts">
  import { themeStore, THEMES } from '$lib/stores';
  import { Palette, User, Shield, Check, Moon, Sun } from 'lucide-svelte';
  import { cn } from '$lib/utils';
</script>

<svelte:head>
  <title>Settings - Lucid Logs</title>
</svelte:head>

<div class="max-w-2xl mx-auto space-y-6">
  <div>
    <h1 class="text-2xl font-extrabold">Settings</h1>
    <p class="text-sm opacity-60">Personalize your experience</p>
  </div>

  <!-- Theme Selection -->
  <div class="glass-card p-5">
    <div class="flex items-center gap-3 mb-4">
      <div class="icon-box icon-box-primary">
        <Palette class="w-5 h-5" />
      </div>
      <div>
        <h2 class="font-bold">Theme</h2>
        <p class="text-sm opacity-50">Choose your favorite look</p>
      </div>
    </div>

    <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
      {#each THEMES as theme}
        <button
          onclick={() => themeStore.set(theme.id)}
          class={cn(
            'relative p-4 rounded-xl border-2 transition-all text-left',
            themeStore.current === theme.id 
              ? 'border-primary bg-primary/10' 
              : 'border-base-content/10 hover:border-base-content/20'
          )}
          data-theme={theme.id}
        >
          <div class="flex items-center gap-2 mb-2">
            <span class="text-2xl">{theme.emoji}</span>
            {#if theme.isDark}
              <Moon class="w-3 h-3 opacity-50" />
            {:else}
              <Sun class="w-3 h-3 opacity-50" />
            {/if}
          </div>
          <p class="font-semibold text-sm">{theme.label}</p>
          <p class="text-xs opacity-50">{theme.description}</p>
          
          {#if themeStore.current === theme.id}
            <div class="absolute top-2 right-2 w-5 h-5 rounded-full bg-primary flex items-center justify-center">
              <Check class="w-3 h-3 text-primary-content" />
            </div>
          {/if}
        </button>
      {/each}
    </div>
  </div>

  <!-- Account Settings -->
  <div class="glass-card p-5">
    <div class="flex items-center gap-3 mb-4">
      <div class="icon-box icon-box-secondary">
        <User class="w-5 h-5" />
      </div>
      <div>
        <h2 class="font-bold">Account</h2>
        <p class="text-sm opacity-50">Manage your account</p>
      </div>
    </div>

    <div class="space-y-3">
      <div class="flex items-center justify-between py-3 border-b border-base-content/10">
        <div>
          <p class="font-medium text-sm">Email</p>
          <p class="text-xs opacity-50">user@example.com</p>
        </div>
        <button class="btn btn-ghost btn-sm">Change</button>
      </div>
      <div class="flex items-center justify-between py-3">
        <div>
          <p class="font-medium text-sm">Password</p>
          <p class="text-xs opacity-50">Last changed 30 days ago</p>
        </div>
        <button class="btn btn-ghost btn-sm">Update</button>
      </div>
    </div>
  </div>

  <!-- Danger Zone -->
  <div class="glass-card p-5 border-error/30">
    <div class="flex items-center gap-3 mb-4">
      <div class="icon-box icon-box-error">
        <Shield class="w-5 h-5" />
      </div>
      <div>
        <h2 class="font-bold text-error">Danger Zone</h2>
        <p class="text-sm opacity-50">Irreversible actions</p>
      </div>
    </div>

    <div class="flex items-center justify-between">
      <div>
        <p class="font-medium text-sm">Delete Account</p>
        <p class="text-xs opacity-50">Permanently delete your account and data</p>
      </div>
      <button class="btn btn-sm btn-error">
        Delete
      </button>
    </div>
  </div>
</div>
