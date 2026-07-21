<script lang="ts">
import { X } from 'lucide-svelte';
import type { Snippet } from 'svelte';

type Size = 'sm' | 'md' | 'lg' | 'xl' | 'full';

interface Props {
	/** Control modal visibility */
	open?: boolean;
	/** Modal size */
	size?: Size;
	/** Modal title */
	title?: string;
	/** Subtitle/description */
	subtitle?: string;
	/** Show close button in header */
	showClose?: boolean;
	/** Close handler */
	onClose?: () => void;
	/** Header icon slot */
	icon?: Snippet;
	/** Main content slot */
	children: Snippet;
	/** Footer actions slot */
	actions?: Snippet;
}

let {
	open = $bindable(false),
	size = 'md',
	title = '',
	subtitle = '',
	showClose = true,
	onClose,
	icon,
	children,
	actions,
}: Props = $props();

const sizeClasses: Record<Size, string> = {
	sm: 'max-w-sm',
	md: 'max-w-lg',
	lg: 'max-w-3xl',
	xl: 'max-w-5xl',
	full: 'max-w-5xl h-[85vh] max-h-[800px]',
};

function handleClose() {
	open = false;
	onClose?.();
}
</script>

<dialog class="modal modal-bottom sm:modal-middle" class:modal-open={open}>
    <div
        class="modal-box {sizeClasses[size]} p-0 flex flex-col overflow-hidden"
    >
        <!-- Mobile drag handle -->
        <div class="sm:hidden flex justify-center pt-2.5 pb-1 shrink-0">
            <div class="w-10 h-1 rounded-full bg-base-content/15"></div>
        </div>
        <!-- Header -->
        {#if title || icon}
            <div
                class="flex items-center justify-between px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-content/5 shrink-0"
            >
                <div class="flex items-center gap-3 min-w-0">
                    {#if icon}
                        <div
                            class="w-10 h-10 rounded-box bg-primary/10 flex items-center justify-center shrink-0"
                        >
                            {@render icon()}
                        </div>
                    {/if}
                    <div class="min-w-0">
                        {#if title}
                            <h3 class="text-lg font-semibold truncate">{title}</h3>
                        {/if}
                        {#if subtitle}
                            <p class="text-xs text-base-content/60 truncate">{subtitle}</p>
                        {/if}
                    </div>
                </div>
                {#if showClose}
                    <button
                        class="btn btn-ghost btn-sm btn-square shrink-0"
                        onclick={handleClose}
                        aria-label="Close"
                    >
                        <X class="w-5 h-5" />
                    </button>
                {/if}
            </div>
        {/if}

        <!-- Content -->
        <div class="flex-1 overflow-y-auto px-4 sm:px-6 py-5">
            {@render children()}
        </div>

        <!-- Footer -->
        {#if actions}
            <div
                class="flex items-center justify-end gap-2 px-4 sm:px-6 py-4 border-t border-base-content/5 shrink-0"
            >
                {@render actions()}
            </div>
        {/if}
    </div>
    <form method="dialog" class="modal-backdrop bg-base-content/20 backdrop-blur-sm">
        <button onclick={handleClose} aria-label="Close">close</button>
    </form>
</dialog>
