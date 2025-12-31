<script lang="ts">
  import { ThemeSettings, AccountSettings } from "$lib/components/settings";
  import { cn } from "$lib/utils";

  let activeTab = $state<"appearance" | "account">("appearance");
</script>

<svelte:head>
  <title>Settings - Lucid Logs</title>
</svelte:head>

<div class="max-w-2xl mx-auto space-y-6">
  <div>
    <h1 class="text-2xl font-extrabold">Settings</h1>
    <p class="text-sm opacity-60">Personalize your experience</p>
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
