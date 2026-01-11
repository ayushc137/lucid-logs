<script lang="ts">
import { authStore } from '$lib/stores/auth.svelte';
import { Shield, User } from 'lucide-svelte';
</script>

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
                        <p class="text-xs opacity-50">
                            {authStore.userEmail || "Not set"}
                        </p>
                    </div>
                    <button class="btn btn-ghost btn-sm">Change</button>
                </div>
                <div class="flex items-center justify-between py-3">
                    <div>
                        <p class="font-medium text-sm">Password</p>
                        <p class="text-xs opacity-50">
                            Last changed 30 days ago
                        </p>
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
                <button class="btn btn-sm btn-error">Delete</button>
            </div>
        </div>
    </div>
</div>
