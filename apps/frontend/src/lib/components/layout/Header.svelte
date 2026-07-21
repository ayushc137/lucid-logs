<script lang="ts">
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { page } from '$app/stores';
import { authStore } from '$lib/stores/auth.svelte';
import {
	ArrowLeft,
	ChevronRight,
	ClipboardList,
	Edit3,
	Home,
	LogOut,
	Menu,
	Search,
	Settings,
} from 'lucide-svelte';
import type { Snippet } from 'svelte';

interface Props {
	onMenuClick?: () => void;
	showSearch?: boolean;
	headerContent?: Snippet;
}

let { onMenuClick, showSearch = true, headerContent }: Props = $props();

const avatarLetter = $derived(
	authStore.userEmail?.charAt(0).toUpperCase() || 'U',
);

const isTaskCreatePage = $derived($page.url.pathname === '/tasks/create');
const isTaskEditPage = $derived(
	$page.url.pathname.startsWith('/tasks/') &&
		$page.url.pathname !== '/tasks' &&
		$page.url.pathname !== '/tasks/create',
);
const isTaskPage = $derived(isTaskCreatePage || isTaskEditPage);

let referrer = $state<string>('/tasks');

$effect(() => {
	if (browser && isTaskPage) {
		const storedReferrer = sessionStorage.getItem('task-form-referrer');
		if (storedReferrer) {
			referrer = storedReferrer;
		}
	}
});

function handleBack() {
	goto(referrer);
}

function handleLogout() {
	authStore.logout();
	goto('/login');
}
</script>

<header
  class="h-14 sm:h-16 flex items-center justify-between gap-3 px-4 sm:px-6 bg-base-100/80 backdrop-blur-md border-b border-base-300/70 z-40 sticky top-0"
>
  <!-- Left -->
  <div class="flex items-center gap-2 flex-1 min-w-0">
    <button
      class="btn btn-ghost btn-square btn-sm lg:hidden shrink-0 -ml-1"
      onclick={onMenuClick}
      aria-label="Open menu"
    >
      <Menu class="w-5 h-5" />
    </button>

    {#if isTaskPage}
      <div class="flex items-center gap-2 flex-1 min-w-0">
        <button class="btn btn-ghost btn-sm gap-1.5 -ml-1" onclick={handleBack}>
          <ArrowLeft class="w-4 h-4" />
          <span class="hidden sm:inline">Back</span>
        </button>

        <div class="hidden sm:flex items-center gap-1.5 text-sm text-base-content/50">
          <a href="/" class="flex items-center gap-1 hover:text-base-content transition-colors">
            <Home class="w-3.5 h-3.5" />
          </a>
          <ChevronRight class="w-3 h-3 opacity-40" />
          <a href="/tasks" class="hover:text-base-content transition-colors">Tasks</a>
          <ChevronRight class="w-3 h-3 opacity-40" />
          {#if isTaskEditPage}
            <span class="font-medium flex items-center gap-1.5 text-base-content">
              <Edit3 class="w-3.5 h-3.5 text-primary" />
              Edit Task
            </span>
          {:else}
            <span class="font-medium flex items-center gap-1.5 text-base-content">
              <ClipboardList class="w-3.5 h-3.5 text-primary" />
              New Task
            </span>
          {/if}
        </div>
      </div>
    {:else if headerContent}
      {@render headerContent()}
    {:else if showSearch}
      <div class="hidden sm:flex items-center">
        <label
          class="input input-sm input-bordered flex items-center gap-2 w-64 lg:w-72 bg-base-200/60 border-transparent hover:border-base-300 focus-within:bg-base-100 focus-within:border-primary/40 transition-colors"
        >
          <Search class="w-4 h-4 opacity-40" />
          <input
            type="text"
            placeholder="Search…"
            class="grow bg-transparent placeholder:text-base-content/40 text-sm"
          />
          <kbd class="kbd kbd-xs bg-base-300/70 border-none text-[10px] opacity-60">⌘K</kbd>
        </label>
      </div>
    {/if}
  </div>

  <!-- Right -->
  <div class="flex items-center gap-1.5">
    <!-- User menu -->
    <div class="dropdown dropdown-end">
      <button
        tabindex="0"
        class="flex items-center justify-center w-9 h-9 rounded-full bg-primary/10 text-primary font-semibold text-sm ring-1 ring-primary/20 hover:ring-primary/40 transition-shadow"
        aria-label="Account"
      >
        {avatarLetter}
      </button>
      <div
        class="dropdown-content z-50 mt-2 w-56 rounded-xl bg-base-100 shadow-xl border border-base-300/70 overflow-hidden"
        role="menu"
      >
        <div class="px-4 py-3 border-b border-base-300/60">
          <p class="text-sm font-medium truncate">{authStore.userEmail || "User"}</p>
        </div>
        <div class="p-1.5">
          <a
            href="/settings"
            class="flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm hover:bg-base-200 transition-colors"
          >
            <Settings class="w-4 h-4 opacity-60" />
            Settings
          </a>
          <button
            onclick={handleLogout}
            class="flex w-full items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm text-error hover:bg-error/10 transition-colors"
          >
            <LogOut class="w-4 h-4" />
            Logout
          </button>
        </div>
      </div>
    </div>
  </div>
</header>
