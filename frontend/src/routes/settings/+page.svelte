<script>
	import { logout, changePassword } from '$lib/api';
	import { goto } from '$app/navigation';
	import Sidebar from '$lib/components/Sidebar.svelte';

	let currentPassword = '';
	let newPassword = '';
	let confirmPassword = '';
	let error = '';
	let success = '';
	let loading = false;

	async function handleLogout() {
		try {
			await logout();
			goto('/login');
		} catch (err) {
			console.error('Logout failed:', err);
			goto('/login');
		}
	}

	async function handleChangePassword() {
		error = '';
		success = '';

		if (newPassword.length < 8) {
			error = 'New password must be at least 8 characters';
			return;
		}

		if (newPassword !== confirmPassword) {
			error = 'New passwords do not match';
			return;
		}

		loading = true;

		try {
			await changePassword(currentPassword, newPassword);
			success = 'Password changed successfully!';
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
		} catch (err) {
			error = err.message;
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Settings - Watchflare</title>
</svelte:head>

<div class="min-h-screen bg-background">
	<Sidebar onLogout={handleLogout} />

	<main class="ml-64 min-h-screen p-8">
		<!-- Header -->
		<div class="mb-6">
			<h1 class="text-2xl font-semibold text-foreground">Settings</h1>
			<p class="text-sm text-muted-foreground mt-1">Manage your account settings</p>
		</div>

		<!-- Change Password Card -->
		<div class="max-w-2xl rounded-lg border bg-card p-6">
			<h2 class="text-lg font-semibold text-foreground mb-6">Change Password</h2>

			<form onsubmit={(e) => { e.preventDefault(); handleChangePassword(); }}>
				<!-- Current Password -->
				<div class="mb-4">
					<label for="current-password" class="block text-sm font-medium text-foreground mb-2">
						Current Password
					</label>
					<input
						id="current-password"
						type="password"
						bind:value={currentPassword}
						required
						placeholder="Enter current password"
						disabled={loading}
						class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
					/>
				</div>

				<!-- New Password -->
				<div class="mb-4">
					<label for="new-password" class="block text-sm font-medium text-foreground mb-2">
						New Password
					</label>
					<input
						id="new-password"
						type="password"
						bind:value={newPassword}
						required
						placeholder="Enter new password (min 8 characters)"
						disabled={loading}
						class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
					/>
				</div>

				<!-- Confirm Password -->
				<div class="mb-4">
					<label for="confirm-password" class="block text-sm font-medium text-foreground mb-2">
						Confirm New Password
					</label>
					<input
						id="confirm-password"
						type="password"
						bind:value={confirmPassword}
						required
						placeholder="Confirm new password"
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

				<!-- Success Message -->
				{#if success}
					<div class="mb-4 rounded-lg border border-success bg-success/10 p-3">
						<p class="text-sm text-success">{success}</p>
					</div>
				{/if}

				<!-- Submit Button -->
				<button
					type="submit"
					disabled={loading}
					class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{loading ? 'Changing Password...' : 'Change Password'}
				</button>
			</form>
		</div>
	</main>
</div>
