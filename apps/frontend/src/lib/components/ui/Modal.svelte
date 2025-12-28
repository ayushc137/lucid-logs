<script lang="ts">
    import type { Snippet } from "svelte";
    import { X } from "lucide-svelte";

    type Size = "sm" | "md" | "lg" | "xl" | "full";

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
        size = "md",
        title = "",
        subtitle = "",
        showClose = true,
        onClose,
        icon,
        children,
        actions,
    }: Props = $props();

    const sizeClasses: Record<Size, string> = {
        sm: "max-w-sm",
        md: "max-w-lg",
        lg: "max-w-3xl",
        xl: "max-w-5xl",
        full: "max-w-5xl h-[85vh] max-h-[800px]",
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
        <!-- Header -->
        {#if title || icon}
            <div
                class="flex items-center justify-between px-6 py-4 border-b border-base-300 bg-base-200/50"
            >
                <div class="flex items-center gap-3">
                    {#if icon}
                        <div
                            class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center"
                        >
                            {@render icon()}
                        </div>
                    {/if}
                    <div>
                        {#if title}
                            <h3 class="font-bold text-lg">{title}</h3>
                        {/if}
                        {#if subtitle}
                            <p class="text-xs opacity-50">{subtitle}</p>
                        {/if}
                    </div>
                </div>
                {#if showClose}
                    <button
                        class="btn btn-ghost btn-sm btn-circle"
                        onclick={handleClose}
                    >
                        <X class="w-5 h-5" />
                    </button>
                {/if}
            </div>
        {/if}

        <!-- Content -->
        <div class="flex-1 overflow-y-auto p-6">
            {@render children()}
        </div>

        <!-- Footer -->
        {#if actions}
            <div
                class="flex items-center justify-end gap-2 px-6 py-4 border-t border-base-300 bg-base-200/50"
            >
                {@render actions()}
            </div>
        {/if}
    </div>
    <form method="dialog" class="modal-backdrop bg-base-content/20">
        <button onclick={handleClose}>close</button>
    </form>
</dialog>
