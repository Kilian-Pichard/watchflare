<script lang="ts">
    import { changePassword } from "$lib/api";
    import { changePasswordSchema, validateForm } from "$lib/validation";

    let currentPassword = "";
    let newPassword = "";
    let confirmPassword = "";
    let error = "";
    let fieldErrors: Record<string, string> = {};
    let success = "";
    let loading = false;

    async function handleChangePassword() {
        error = "";
        fieldErrors = {};
        success = "";

        const result = validateForm(changePasswordSchema, {
            currentPassword,
            newPassword,
            confirmPassword,
        });
        if (!result.success) {
            fieldErrors = result.errors;
            return;
        }

        loading = true;

        try {
            await changePassword(currentPassword, newPassword);
            success = "Password changed successfully!";
            currentPassword = "";
            newPassword = "";
            confirmPassword = "";
        } catch (err) {
            error =
                err.message === "current password is incorrect"
                    ? "Current password is incorrect."
                    : err.message;
        } finally {
            loading = false;
        }
    }
</script>

<svelte:head>
    <title>Settings - Watchflare</title>
</svelte:head>

<!-- Header -->
<div class="mb-6">
    <h1 class="text-2xl font-semibold text-foreground">Settings</h1>
    <p class="text-sm text-muted-foreground mt-1">
        Manage your account settings
    </p>
</div>

<!-- Change Password Card -->
<div class="max-w-2xl rounded-lg border bg-card p-4 sm:p-6">
    <h2 class="text-lg font-semibold text-foreground mb-6">
        Change Password
    </h2>

    <form
        onsubmit={(e) => {
            e.preventDefault();
            handleChangePassword();
        }}
    >
        <!-- Current Password -->
        <div class="mb-4">
            <label
                for="current-password"
                class="block text-sm font-medium text-foreground mb-2"
            >
                Current Password
            </label>
            <input
                id="current-password"
                type="password"
                bind:value={currentPassword}
                placeholder="Enter current password"
                disabled={loading}
                aria-invalid={!!fieldErrors.currentPassword}
                aria-describedby={fieldErrors.currentPassword ? 'currentPassword-error' : undefined}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50 {fieldErrors.currentPassword
                    ? 'border-destructive'
                    : ''}"
            />
            {#if fieldErrors.currentPassword}<p
                    id="currentPassword-error"
                    class="mt-1 text-xs text-destructive"
                >
                    {fieldErrors.currentPassword}
                </p>{/if}
        </div>

        <!-- New Password -->
        <div class="mb-4">
            <label
                for="new-password"
                class="block text-sm font-medium text-foreground mb-2"
            >
                New Password
            </label>
            <input
                id="new-password"
                type="password"
                bind:value={newPassword}
                placeholder="Enter new password (min 12 characters)"
                disabled={loading}
                aria-invalid={!!fieldErrors.newPassword}
                aria-describedby={fieldErrors.newPassword ? 'newPassword-error' : undefined}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50 {fieldErrors.newPassword
                    ? 'border-destructive'
                    : ''}"
            />
            {#if fieldErrors.newPassword}<p
                    id="newPassword-error"
                    class="mt-1 text-xs text-destructive"
                >
                    {fieldErrors.newPassword}
                </p>{/if}
        </div>

        <!-- Confirm Password -->
        <div class="mb-4">
            <label
                for="confirm-password"
                class="block text-sm font-medium text-foreground mb-2"
            >
                Confirm New Password
            </label>
            <input
                id="confirm-password"
                type="password"
                bind:value={confirmPassword}
                placeholder="Confirm new password"
                disabled={loading}
                aria-invalid={!!fieldErrors.confirmPassword}
                aria-describedby={fieldErrors.confirmPassword ? 'confirmPassword-error' : undefined}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50 {fieldErrors.confirmPassword
                    ? 'border-destructive'
                    : ''}"
            />
            {#if fieldErrors.confirmPassword}<p
                    id="confirmPassword-error"
                    class="mt-1 text-xs text-destructive"
                >
                    {fieldErrors.confirmPassword}
                </p>{/if}
        </div>

        <!-- Error Message -->
        {#if error}
            <div
                class="mb-4 rounded-lg border border-destructive bg-destructive/10 p-3"
            >
                <p class="text-sm text-destructive">{error}</p>
            </div>
        {/if}

        <!-- Success Message -->
        {#if success}
            <div
                class="mb-4 rounded-lg border border-success bg-success/10 p-3"
            >
                <p class="text-sm text-success">{success}</p>
            </div>
        {/if}

        <!-- Submit Button -->
        <button
            type="submit"
            disabled={loading}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
            {loading ? "Changing Password..." : "Change Password"}
        </button>
    </form>
</div>
