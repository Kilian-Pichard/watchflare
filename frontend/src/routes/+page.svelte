<script>
	import { logout } from '$lib/api';
	import { goto } from '$app/navigation';

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
</script>

<svelte:head>
	<title>Dashboard - Watchflare</title>
</svelte:head>

<div class="container">
	<nav class="navbar">
		<div class="nav-content">
			<h1>Watchflare</h1>
			<div class="nav-actions">
				<a href="/settings" class="nav-link">Settings</a>
				<button on:click={handleLogout} class="logout-btn">Logout</button>
			</div>
		</div>
	</nav>

	<main class="main">
		<div class="welcome-card">
			<h2>Welcome to Watchflare</h2>
			<p>Your server monitoring dashboard</p>

			<div class="info-box">
				<h3>Authentication successful!</h3>
				<p>You are now logged in to the Watchflare dashboard.</p>
				<p class="muted">Server monitoring features will be available soon.</p>
			</div>

			<div class="actions">
				<a href="/settings" class="btn btn-primary">Change Password</a>
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
		max-width: 1200px;
		margin: 0 auto;
		padding: 3rem 2rem;
	}

	.welcome-card {
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

	.welcome-card > p {
		margin: 0 0 2rem 0;
		color: #718096;
		font-size: 1.125rem;
	}

	.info-box {
		background: linear-gradient(135deg, #667eea15 0%, #764ba215 100%);
		border-left: 4px solid #667eea;
		padding: 1.5rem;
		border-radius: 6px;
		margin-bottom: 2rem;
	}

	.info-box h3 {
		margin: 0 0 0.5rem 0;
		color: #1a202c;
		font-size: 1.25rem;
	}

	.info-box p {
		margin: 0.5rem 0;
		color: #4a5568;
	}

	.muted {
		color: #a0aec0 !important;
		font-size: 0.875rem;
	}

	.actions {
		display: flex;
		gap: 1rem;
	}

	.btn {
		padding: 0.75rem 1.5rem;
		border-radius: 6px;
		font-weight: 500;
		text-decoration: none;
		cursor: pointer;
		transition: transform 0.2s;
		display: inline-block;
	}

	.btn:hover {
		transform: translateY(-1px);
	}

	.btn-primary {
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
	}
</style>
