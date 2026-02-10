<script>
	import { register } from '$lib/api';
	import { goto } from '$app/navigation';

	let email = '';
	let password = '';
	let confirmPassword = '';
	let error = '';
	let loading = false;

	async function handleRegister() {
		error = '';

		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		loading = true;

		try {
			await register(email, password);
			goto('/');
		} catch (err) {
			error = err.message;
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Setup - Watchflare</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-background p-4">
	<div class="w-full max-w-md">
		<!-- Logo/Title -->
		<div class="mb-8 text-center">
			<h1 class="text-3xl font-semibold text-foreground mb-2">Watchflare</h1>
			<p class="text-sm text-muted-foreground">Initial Setup - Create Admin Account</p>
		</div>

		<!-- Register Card -->
		<div class="rounded-lg border bg-card p-8 shadow-sm">
			<h2 class="text-lg font-semibold text-foreground mb-6">Create your admin account</h2>

			<form onsubmit={(e) => { e.preventDefault(); handleRegister(); }}>
				<!-- Email -->
				<div class="mb-4">
					<label for="email" class="block text-sm font-medium text-foreground mb-2">
						Email
					</label>
					<input
						id="email"
						type="email"
						bind:value={email}
						required
						placeholder="admin@watchflare.io"
						disabled={loading}
						class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
					/>
				</div>

				<!-- Password -->
				<div class="mb-4">
					<label for="password" class="block text-sm font-medium text-foreground mb-2">
						Password
					</label>
					<input
						id="password"
						type="password"
						bind:value={password}
						required
						minlength="8"
						placeholder="••••••••"
						disabled={loading}
						class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
					/>
					<p class="mt-1 text-xs text-muted-foreground">Minimum 8 characters</p>
				</div>

				<!-- Confirm Password -->
				<div class="mb-6">
					<label for="confirmPassword" class="block text-sm font-medium text-foreground mb-2">
						Confirm Password
					</label>
					<input
						id="confirmPassword"
						type="password"
						bind:value={confirmPassword}
						required
						minlength="8"
						placeholder="••••••••"
						disabled={loading}
						class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
					/>
				</div>

				<!-- Error Message -->
				{#if error}
					<div class="mb-4 rounded-lg border border-destructive bg-destructive/10 p-3">
						<p class="text-sm text-destructive">{error}</p>
					</div>
				{/if}

				<!-- Submit Button -->
				<button
					type="submit"
					disabled={loading}
					class="w-full rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{loading ? 'Creating Account...' : 'Create Admin Account'}
				</button>
			</form>
		</div>

		<!-- Footer -->
		<p class="mt-6 text-center text-xs text-muted-foreground">
			Watchflare Server Monitoring
		</p>
	</div>
</div>
