<script>
	import { logout, changePassword } from '$lib/api';
	import { goto } from '$app/navigation';

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
			// Force redirect to login anyway
			goto('/login');
		}
	}

	async function handleChangePassword() {
		error = '';
		success = '';

		// Validation
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

<div class="container">
	<nav class="navbar">
		<div class="nav-content">
			<h1>Watchflare</h1>
			<div class="nav-actions">
				<a href="/" class="nav-link">Dashboard</a>
				<a href="/servers" class="nav-link">Servers</a>
				<button on:click={handleLogout} class="logout-btn">Logout</button>
			</div>
		</div>
	</nav>

	<main class="main">
		<div class="settings-card">
			<h2>Settings</h2>
			<p class="subtitle">Manage your account settings</p>

			<div class="section">
				<h3>Change Password</h3>

				<form on:submit|preventDefault={handleChangePassword}>
					<div class="form-group">
						<label for="current-password">Current Password</label>
						<input
							id="current-password"
							type="password"
							bind:value={currentPassword}
							required
							placeholder="Enter current password"
							disabled={loading}
						/>
					</div>

					<div class="form-group">
						<label for="new-password">New Password</label>
						<input
							id="new-password"
							type="password"
							bind:value={newPassword}
							required
							placeholder="Enter new password (min 8 characters)"
							disabled={loading}
						/>
					</div>

					<div class="form-group">
						<label for="confirm-password">Confirm New Password</label>
						<input
							id="confirm-password"
							type="password"
							bind:value={confirmPassword}
							required
							placeholder="Confirm new password"
							disabled={loading}
						/>
					</div>

					{#if error}
						<div class="error">{error}</div>
					{/if}

					{#if success}
						<div class="success">{success}</div>
					{/if}

					<button type="submit" disabled={loading}>
						{loading ? 'Changing Password...' : 'Change Password'}
					</button>
				</form>
			</div>
		</div>
	</main>
</div>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial,
			sans-serif;
		background: #f7fafc;
	}

	.container {
		min-height: 100vh;
	}

	.navbar {
		background: white;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		padding: 1rem 0;
	}

	.nav-content {
		max-width: 1200px;
		margin: 0 auto;
		padding: 0 2rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.navbar h1 {
		margin: 0;
		font-size: 1.5rem;
		color: #667eea;
	}

	.nav-actions {
		display: flex;
		gap: 1rem;
		align-items: center;
	}

	.nav-link {
		color: #4a5568;
		text-decoration: none;
		font-weight: 500;
		padding: 0.5rem 1rem;
		border-radius: 6px;
		transition: background-color 0.2s;
	}

	.nav-link:hover {
		background-color: #edf2f7;
	}

	.logout-btn {
		padding: 0.5rem 1rem;
		background: #e53e3e;
		color: white;
		border: none;
		border-radius: 6px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	.logout-btn:hover {
		background: #c53030;
	}

	.main {
		max-width: 800px;
		margin: 0 auto;
		padding: 3rem 2rem;
	}

	.settings-card {
		background: white;
		padding: 2.5rem;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	h2 {
		margin: 0 0 0.5rem 0;
		font-size: 2rem;
		color: #1a202c;
	}

	.subtitle {
		margin: 0 0 2rem 0;
		color: #718096;
	}

	.section {
		border-top: 1px solid #e2e8f0;
		padding-top: 2rem;
	}

	h3 {
		margin: 0 0 1.5rem 0;
		font-size: 1.25rem;
		color: #2d3748;
	}

	.form-group {
		margin-bottom: 1.5rem;
	}

	label {
		display: block;
		margin-bottom: 0.5rem;
		color: #4a5568;
		font-weight: 500;
		font-size: 0.875rem;
	}

	input {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		font-size: 1rem;
		transition: border-color 0.2s;
		box-sizing: border-box;
	}

	input:focus {
		outline: none;
		border-color: #667eea;
	}

	input:disabled {
		background-color: #f7fafc;
		cursor: not-allowed;
	}

	button {
		padding: 0.75rem 1.5rem;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		border: none;
		border-radius: 6px;
		font-size: 1rem;
		font-weight: 600;
		cursor: pointer;
		transition: transform 0.2s;
	}

	button:hover:not(:disabled) {
		transform: translateY(-1px);
	}

	button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.error {
		background: #fed7d7;
		color: #c53030;
		padding: 0.75rem;
		border-radius: 6px;
		margin-bottom: 1rem;
		font-size: 0.875rem;
	}

	.success {
		background: #c6f6d5;
		color: #2f855a;
		padding: 0.75rem;
		border-radius: 6px;
		margin-bottom: 1rem;
		font-size: 0.875rem;
	}
</style>
