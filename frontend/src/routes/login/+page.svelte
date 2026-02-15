<script lang="ts">
    import { onMount } from "svelte";
    import { login } from "$lib/api";
    import { goto } from "$app/navigation";
    import { loginSchema, validateForm } from "$lib/validation";

    let email = "";
    let password = "";
    let error = "";
    let fieldErrors: Record<string, string> = {};
    let loading = false;

    onMount(async () => {
        try {
            const response = await fetch(
                "http://localhost:8080/auth/setup-required",
                {
                    credentials: "include",
                },
            );
            const data = await response.json();
            if (data.setup_required) {
                goto("/register");
            }
        } catch (err) {
            console.error("Failed to check setup status:", err);
        }
    });

    async function handleLogin() {
        error = "";
        fieldErrors = {};

        const result = validateForm(loginSchema, { email, password });
        if (!result.success) {
            fieldErrors = result.errors;
            return;
        }

        loading = true;

        try {
            await login(email, password);
            goto("/");
        } catch (err) {
            error =
                err.message === "invalid credentials"
                    ? "Invalid credentials."
                    : err.message;
        } finally {
            loading = false;
        }
    }
</script>

<svelte:head>
    <title>Login - Watchflare</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-background p-4">
    <div class="w-full max-w-md">
        <!-- Logo/Title -->
        <div class="mb-8 text-center">
            <h1 class="text-3xl font-semibold text-foreground mb-2">
                Watchflare
            </h1>
            <p class="text-sm text-muted-foreground">
                Server Monitoring Dashboard
            </p>
        </div>

        <!-- Login Card -->
        <div class="rounded-lg border bg-card p-8 shadow-sm">
            <h2 class="text-lg font-semibold text-foreground mb-6">
                Login to your account
            </h2>

            <form
                onsubmit={(e) => {
                    e.preventDefault();
                    handleLogin();
                }}
            >
                <!-- Email -->
                <div class="mb-4">
                    <label
                        for="email"
                        class="block text-sm font-medium text-foreground mb-2"
                    >
                        Email
                    </label>
                    <input
                        id="email"
                        type="email"
                        bind:value={email}
                        placeholder="admin@watchflare.io"
                        disabled={loading}
                        class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50 {fieldErrors.email
                            ? 'border-destructive'
                            : ''}"
                    />
                    {#if fieldErrors.email}<p
                            class="mt-1 text-xs text-destructive"
                        >
                            {fieldErrors.email}
                        </p>{/if}
                </div>

                <!-- Password -->
                <div class="mb-6">
                    <label
                        for="password"
                        class="block text-sm font-medium text-foreground mb-2"
                    >
                        Password
                    </label>
                    <input
                        id="password"
                        type="password"
                        bind:value={password}
                        placeholder="••••••••"
                        disabled={loading}
                        class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50 {fieldErrors.password
                            ? 'border-destructive'
                            : ''}"
                    />
                    {#if fieldErrors.password}<p
                            class="mt-1 text-xs text-destructive"
                        >
                            {fieldErrors.password}
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

                <!-- Submit Button -->
                <button
                    type="submit"
                    disabled={loading}
                    class="w-full rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    {loading ? "Logging in..." : "Login"}
                </button>
            </form>
        </div>

        <!-- Footer -->
        <p class="mt-6 text-center text-xs text-muted-foreground">
            Watchflare Server Monitoring
        </p>
    </div>
</div>
