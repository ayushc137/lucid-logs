<script lang="ts">
  import { Bell, Search, Plus, Menu, Settings, LogOut } from "lucide-svelte";
  import { authStore } from "$lib/stores/auth.svelte";

  interface Props {
    onMenuClick?: () => void;
  }

  let { onMenuClick }: Props = $props();

  // Get first letter of email for avatar
  const avatarLetter = $derived(
    authStore.userEmail?.charAt(0).toUpperCase() || "U",
  );
</script>

<header
  class="h-14 flex items-center justify-between px-4 bg-base-100 border-b border-base-300"
>
  <!-- Left -->
  <div class="flex items-center gap-3">
    <button
      class="btn btn-ghost btn-sm btn-square lg:hidden"
      onclick={onMenuClick}
      aria-label="Menu"
    >
      <Menu class="w-5 h-5" />
    </button>

    <div class="hidden sm:flex items-center gap-2">
      <label class="input input-sm input-bordered flex items-center gap-2 w-64">
        <Search class="w-4 h-4 opacity-50" />
        <input
          type="text"
          placeholder="Search logs..."
          class="grow bg-transparent"
        />
        <kbd class="kbd kbd-xs">⌘K</kbd>
      </label>
    </div>
  </div>

  <!-- Right -->
  <div class="flex items-center gap-2">
    <!-- Quick Log Button -->
    <button class="btn btn-primary btn-sm gap-2">
      <Plus class="w-4 h-4" />
      <span class="hidden sm:inline">Quick Log</span>
    </button>

    <!-- Notifications -->
    <button class="btn btn-ghost btn-sm btn-square" aria-label="Notifications">
      <div class="indicator">
        <Bell class="w-5 h-5" />
        <span class="badge badge-error badge-xs indicator-item"></span>
      </div>
    </button>

    <!-- User Avatar with Dropdown -->
    <div class="dropdown dropdown-end">
      <button
        tabindex="0"
        class="btn btn-ghost btn-circle avatar hover:ring-2 hover:ring-primary/20 transition-all"
      >
        <div
          class="w-8 rounded-full bg-gradient-to-br from-primary to-secondary text-primary-content flex items-center justify-center shadow-sm"
        >
          <span class="text-xs font-bold">{avatarLetter}</span>
        </div>
      </button>
      <div
        class="dropdown-content z-50 mt-2 w-52 rounded-xl bg-base-100 shadow-2xl border border-base-200 overflow-hidden"
        role="menu"
      >
        <!-- User Info Header -->
        <div class="px-4 py-3 bg-base-200/50 border-b border-base-200">
          <p class="text-sm font-medium text-base-content truncate">
            {authStore.userEmail || "User"}
          </p>
        </div>

        <!-- Menu Items -->
        <div class="p-2 flex flex-col gap-1">
          <a
            href="/settings"
            class="btn btn-ghost btn-sm w-full justify-start gap-3"
          >
            <Settings class="w-4 h-4" />
            <span>Settings</span>
          </a>

          <div class="divider my-1"></div>

          <button
            class="btn btn-ghost btn-sm w-full justify-start gap-3 text-error hover:bg-error/10"
          >
            <LogOut class="w-4 h-4" />
            <span>Logout</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</header>
