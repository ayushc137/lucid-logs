<script lang="ts">
import { AccountSettings, ThemeSettings } from '$lib/components/settings';
import { cn } from '$lib/utils';
import { Settings } from 'lucide-svelte';

let activeTab = $state<'appearance' | 'account'>('appearance');
</script>

<svelte:head>
  <title>Settings - Lucid Logs</title>
</svelte:head>

<div class="max-w-2xl mx-auto space-y-6 px-4 py-4 sm:px-6 sm:py-6">
  <div class="flex items-center gap-3.5">
    <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
      <Settings class="w-5.5 h-5.5" />
    </div>
    <div>
      <h1 class="text-2xl sm:text-[1.7rem] font-bold tracking-tight leading-tight">Settings</h1>
      <p class="text-sm text-base-content/55">Make it yours</p>
    </div>
  </div>

  <!-- Tabs -->
  <div role="tablist" class="tabs tabs-boxed">
    <button
      role="tab"
      class={cn("tab", activeTab === "appearance" && "tab-active")}
      onclick={() => (activeTab = "appearance")}
    >
      Appearance
    </button>
    <button
      role="tab"
      class={cn("tab", activeTab === "account" && "tab-active")}
      onclick={() => (activeTab = "account")}
    >
      Account
    </button>
  </div>

  <!-- Tab Content -->
  {#if activeTab === "appearance"}
    <ThemeSettings />
  {:else if activeTab === "account"}
    <AccountSettings />
  {/if}
</div>
