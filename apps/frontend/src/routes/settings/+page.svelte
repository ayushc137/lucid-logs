<script lang="ts">
  import { themeStore, THEMES } from "$lib/stores";
  import { Palette, User, Shield, Check, Moon, Sun } from "lucide-svelte";
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

  <!-- Appearance Content -->
  {#if activeTab === "appearance"}
    <div class="card bg-base-100 shadow">
      <div class="card-body">
        <div class="flex items-center gap-3 mb-4">
          <div
            class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center"
          >
            <Palette class="w-5 h-5 text-primary" />
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
                "relative p-4 rounded-xl border-2 transition-all text-left",
                themeStore.current === theme.id
                  ? "border-primary bg-primary/10"
                  : "border-base-300 hover:border-base-content/30",
              )}
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
                <div
                  class="absolute top-2 right-2 w-5 h-5 rounded-full bg-primary flex items-center justify-center"
                >
                  <Check class="w-3 h-3 text-primary-content" />
                </div>
              {/if}
            </button>
          {/each}
        </div>
      </div>
    </div>
  {/if}

  <!-- Account Content -->
  {#if activeTab === "account"}
    <div class="space-y-6">
      <div class="card bg-base-100 shadow">
        <div class="card-body">
          <div class="flex items-center gap-3 mb-4">
            <div
              class="w-10 h-10 rounded-lg bg-secondary/10 flex items-center justify-center"
            >
              <User class="w-5 h-5 text-secondary" />
            </div>
            <div>
              <h2 class="font-bold">Account</h2>
              <p class="text-sm opacity-50">Manage your account</p>
            </div>
          </div>

          <div class="space-y-3">
            <div
              class="flex items-center justify-between py-3 border-b border-base-300"
            >
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
      </div>

      <div class="card bg-base-100 shadow border border-error/30">
        <div class="card-body">
          <div class="flex items-center gap-3 mb-4">
            <div
              class="w-10 h-10 rounded-lg bg-error/10 flex items-center justify-center"
            >
              <Shield class="w-5 h-5 text-error" />
            </div>
            <div>
              <h2 class="font-bold text-error">Danger Zone</h2>
              <p class="text-sm opacity-50">Irreversible actions</p>
            </div>
          </div>

          <div class="flex items-center justify-between">
            <div>
              <p class="font-medium text-sm">Delete Account</p>
              <p class="text-xs opacity-50">
                Permanently delete your account and data
              </p>
            </div>
            <button class="btn btn-sm btn-error"> Delete </button>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>
