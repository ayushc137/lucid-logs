<script lang="ts">
  import {
    Bell,
    Search,
    Plus,
    Menu,
    Settings,
    LogOut,
    ArrowLeft,
    Home,
    ChevronRight,
    ClipboardList,
    Edit3,
  } from "lucide-svelte";
  import { authStore } from "$lib/stores/auth.svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { browser } from "$app/environment";
  import type { Snippet } from "svelte";

  interface Props {
    onMenuClick?: () => void;
    showSearch?: boolean;
    headerContent?: Snippet;
  }

  let { onMenuClick, showSearch = true, headerContent }: Props = $props();

  // Get first letter of email for avatar
  const avatarLetter = $derived(
    authStore.userEmail?.charAt(0).toUpperCase() || "U",
  );

  // Task Page Detection
  const isTaskCreatePage = $derived($page.url.pathname === "/tasks/create");
  const isTaskEditPage = $derived(
    $page.url.pathname.startsWith("/tasks/") &&
      $page.url.pathname !== "/tasks" &&
      $page.url.pathname !== "/tasks/create",
  );
  const isTaskPage = $derived(isTaskCreatePage || isTaskEditPage);

  // Back Navigation Logic
  let referrer = $state<string>("/tasks");

  $effect(() => {
    if (browser && isTaskPage) {
      const storedReferrer = sessionStorage.getItem("task-form-referrer");
      if (storedReferrer) {
        referrer = storedReferrer;
      }
    }
  });

  function handleBack() {
    goto(referrer);
  }
</script>

<header
  class="h-16 flex items-center justify-between px-6 bg-base-200/95 backdrop-blur-sm border-b border-base-300 z-40 sticky top-0"
>
  <!-- Left -->
  <div class="flex items-center gap-4 flex-1">
    <button
      class="btn btn-ghost btn-square lg:hidden shrink-0"
      onclick={onMenuClick}
      aria-label="Menu"
    >
      <Menu class="w-6 h-6" />
    </button>

    {#if isTaskPage}
      <!-- Task Page Breadcrumbs (Built-in) -->
      <div class="flex items-center gap-3 flex-1">
        <button class="btn btn-ghost btn-sm gap-2" onclick={handleBack}>
          <ArrowLeft class="w-4 h-4" />
          <span class="hidden sm:inline">Back</span>
        </button>

        <div class="hidden sm:flex items-center gap-1.5 text-sm">
          <a
            href="/"
            class="flex items-center gap-1 opacity-50 hover:opacity-100 transition-opacity"
          >
            <Home class="w-3.5 h-3.5" />
          </a>
          <ChevronRight class="w-3 h-3 opacity-30" />
          <a
            href="/tasks"
            class="opacity-50 hover:opacity-100 transition-opacity">Tasks</a
          >
          <ChevronRight class="w-3 h-3 opacity-30" />
          {#if isTaskEditPage}
            <span class="font-medium flex items-center gap-1.5 text-primary">
              <Edit3 class="w-3.5 h-3.5" />
              Edit Task
            </span>
          {:else}
            <span class="font-medium flex items-center gap-1.5 text-primary">
              <ClipboardList class="w-3.5 h-3.5" />
              New Task
            </span>
          {/if}
        </div>
      </div>
    {:else if headerContent}
      <!-- Custom header content (breadcrumbs, back button, etc) -->
      {@render headerContent()}
    {:else if showSearch}
      <!-- Search Bar - only shown when showSearch is true -->
      <div class="hidden sm:flex items-center gap-2">
        <label
          class="input input-bordered flex items-center gap-3 w-72 h-10 bg-base-200/50 hover:bg-base-200 transition-colors focus-within:bg-base-100 focus-within:border-primary/50"
        >
          <Search class="w-4 h-4 opacity-50" />
          <input
            type="text"
            placeholder="Search logs..."
            class="grow bg-transparent placeholder:text-base-content/40"
          />
          <kbd
            class="kbd kbd-sm h-6 min-h-0 bg-base-300 border-none text-[10px] font-bold opacity-60"
            >⌘K</kbd
          >
        </label>
      </div>
    {/if}
  </div>

  <!-- Right -->
  <div class="flex items-center gap-3">
    <!-- Notifications -->
    <button
      class="btn btn-ghost btn-circle h-10 w-10 min-h-0 hover:bg-base-200"
      aria-label="Notifications"
    >
      <div class="indicator">
        <Bell class="w-5 h-5" />
        <span
          class="badge badge-error badge-xs indicator-item w-2 h-2 p-0 border-2 border-base-100"
        ></span>
      </div>
    </button>

    <!-- User Avatar with Dropdown -->
    <div class="dropdown dropdown-end">
      <button
        tabindex="0"
        class="btn btn-ghost btn-circle avatar h-10 w-10 min-h-0 hover:ring-2 hover:ring-primary/20 transition-all p-0"
      >
        <div
          class="w-10 rounded-full bg-gradient-to-br from-primary to-secondary text-primary-content flex items-center justify-center shadow-lg shadow-primary/20"
        >
          <span class="text-sm font-bold">{avatarLetter}</span>
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
