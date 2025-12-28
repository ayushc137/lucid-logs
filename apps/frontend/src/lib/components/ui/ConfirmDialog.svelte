<script lang="ts">
    import { TriangleAlert, AlertCircle, X } from "lucide-svelte";

    interface Props {
        /** Control dialog visibility */
        open?: boolean;
        /** Dialog title */
        title?: string;
        /** Confirmation message */
        message?: string;
        /** Confirm button text */
        confirmText?: string;
        /** Cancel button text */
        cancelText?: string;
        /** Confirm button is destructive (red) vs warning (amber) */
        destructive?: boolean;
        /** Loading state */
        loading?: boolean;
        /** Confirm handler */
        onConfirm?: () => void;
        /** Cancel/close handler */
        onCancel?: () => void;
    }

    let {
        open = $bindable(false),
        title = "Are you sure?",
        message = "This action cannot be undone.",
        confirmText = "Confirm",
        cancelText = "Cancel",
        destructive = true,
        loading = false,
        onConfirm,
        onCancel,
    }: Props = $props();

    function handleCancel() {
        open = false;
        onCancel?.();
    }

    function handleConfirm() {
        onConfirm?.();
    }

    // Color schemes based on destructive prop
    const iconBg = $derived(destructive ? "bg-error/10" : "bg-warning/10");
    const iconColor = $derived(destructive ? "text-error" : "text-warning");
    const titleColor = $derived(destructive ? "text-error" : "text-warning");
    const buttonClass = $derived(destructive ? "btn-error" : "btn-warning");
</script>

<dialog class="modal modal-bottom sm:modal-middle" class:modal-open={open}>
    <div class="modal-box max-w-md p-0 overflow-hidden">
        <!-- Header -->
        <div
            class="flex items-center justify-between px-6 py-4 border-b border-base-300 bg-base-200/50"
        >
            <div class="flex items-center gap-3">
                <!-- Icon Box - matching design language -->
                <div
                    class="w-10 h-10 rounded-xl flex items-center justify-center {iconBg}"
                >
                    {#if destructive}
                        <TriangleAlert class="w-5 h-5 {iconColor}" />
                    {:else}
                        <AlertCircle class="w-5 h-5 {iconColor}" />
                    {/if}
                </div>
                <div>
                    <h3 class="font-bold text-lg {titleColor}">{title}</h3>
                    <p class="text-xs opacity-50">
                        {destructive
                            ? "This action is permanent"
                            : "Please confirm your action"}
                    </p>
                </div>
            </div>
            <button
                class="btn btn-ghost btn-sm btn-circle"
                onclick={handleCancel}
                disabled={loading}
            >
                <X class="w-5 h-5" />
            </button>
        </div>

        <!-- Content -->
        <div class="px-6 py-5">
            <p class="text-base-content/80 leading-relaxed">{message}</p>
        </div>

        <!-- Footer -->
        <div
            class="flex items-center justify-end gap-2 px-6 py-4 border-t border-base-300 bg-base-200/50"
        >
            <button
                class="btn btn-ghost"
                onclick={handleCancel}
                disabled={loading}
            >
                {cancelText}
            </button>
            <button
                class="btn {buttonClass} min-w-[120px]"
                onclick={handleConfirm}
                disabled={loading}
            >
                {#if loading}
                    <span class="loading loading-spinner loading-sm"></span>
                {/if}
                {confirmText}
            </button>
        </div>
    </div>
    <form
        method="dialog"
        class="modal-backdrop bg-base-content/20 backdrop-blur-sm"
    >
        <button onclick={handleCancel}>close</button>
    </form>
</dialog>
