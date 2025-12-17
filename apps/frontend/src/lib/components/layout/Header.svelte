<script lang="ts">
  import { Bell, Search, Plus, Menu } from "lucide-svelte";

  interface Props {
    onMenuClick?: () => void;
  }

  let { onMenuClick }: Props = $props();
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
      <button tabindex="0" class="btn btn-ghost btn-circle avatar">
        <div
          class="w-8 rounded-full bg-primary text-primary-content flex items-center justify-center"
        >
          <span class="text-xs font-semibold">U</span>
        </div>
      </button>
      <ul
        tabindex="0"
        class="dropdown-content menu bg-base-100 rounded-box z-50 w-48 p-2 shadow-xl border border-base-300"
      >
        <li class="menu-title">My Account</li>
        <li><a href="/settings">Settings</a></li>
        <li><a href="/profile">Profile</a></li>
        <li class="divider"></li>
        <li><button class="text-error">Logout</button></li>
      </ul>
    </div>
  </div>
</header>
