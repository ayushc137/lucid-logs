<script lang="ts">
    interface Props {
        /** Loading message */
        message?: string;
        /** Custom class for container */
        class?: string;
    }

    let { message = "Loading...", class: className = "" }: Props = $props();
</script>

<div class="card bg-base-100 shadow-lg border border-base-200 {className}">
    <div class="card-body py-16 text-center">
        <span class="loading loading-spinner loading-lg text-primary mx-auto"
        ></span>
        <p class="opacity-60 mt-4">{message}</p>
    </div>
</div>
