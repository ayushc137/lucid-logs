<script lang="ts">
    import { TriangleAlert } from "lucide-svelte";

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
        /** Confirm button is destructive */
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
</script>

<dialog class="modal" class:modal-open={open}>
    <div class="modal-box max-w-sm">
        <div class="flex items-start gap-4">
            {#if destructive}
                <div
                    class="w-12 h-12 rounded-full bg-error/10 flex items-center justify-center shrink-0"
                >
                    <TriangleAlert class="w-6 h-6 text-error" />
                </div>
            {/if}
            <div>
                <h3 class="font-bold text-lg">{title}</h3>
                <p class="py-2 opacity-70">{message}</p>
            </div>
        </div>
        <div class="modal-action">
            <button
                class="btn btn-ghost"
                onclick={handleCancel}
                disabled={loading}
            >
                {cancelText}
            </button>
            <button
                class="btn {destructive ? 'btn-error' : 'btn-primary'}"
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
    <form method="dialog" class="modal-backdrop">
        <button onclick={handleCancel}>close</button>
    </form>
</dialog>
