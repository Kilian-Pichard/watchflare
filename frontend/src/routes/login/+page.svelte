<script>
	import { onMount } from 'svelte';
	import { login } from '$lib/api';
	import { goto } from '$app/navigation';

	let email = '';
	let password = '';
	let error = '';
	let loading = false;

	onMount(async () => {
		// Check if initial setup is required
		try {
			const response = await fetch('http://localhost:8080/auth/setup-required', {
				credentials: 'include'
			});
			const data = await response.json();
			if (data.setup_required) {
				// No users exist, redirect to registration
				goto('/register');
			}
		} catch (err) {
			console.error('Failed to check setup status:', err);
		}
	});

	async function handleLogin() {
		error = '';
		loading = true;

		try {
			await login(email, password);
			goto('/');
		} catch (err) {
			error = err.message;
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Login - Watchflare</title>
</svelte:head>

<div class="container">
	<div class="login-card">
		<h1>Watchflare</h1>
		<p class="subtitle">Server Monitoring Dashboard</p>

		<form on:submit|preventDefault={handleLogin}>
			<div class="form-group">
				<label for="email">Email</label>
				<input
					id="email"
					type="email"
					bind:value={email}
					required
					placeholder="admin@watchflare.io"
					disabled={loading}
				/>
			</div>

			<div class="form-group">
				<label for="password">Password</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					required
					placeholder="••••••••"
					disabled={loading}
				/>
			</div>

			{#if error}
				<div class="error">{error}</div>
			{/if}

			<button type="submit" disabled={loading}>
				{loading ? 'Logging in...' : 'Login'}
			</button>
		</form>
	</div>
</div>

<style>
	.container {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial,
			sans-serif;
	}

	.login-card {
		background: white;
		padding: 2.5rem;
		border-radius: 12px;
		box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
		width: 100%;
		max-width: 400px;
	}

	h1 {
		margin: 0 0 0.5rem 0;
		font-size: 2rem;
		color: #1a202c;
		text-align: center;
	}

	.subtitle {
		margin: 0 0 2rem 0;
		color: #718096;
		text-align: center;
		font-size: 0.875rem;
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
		width: 100%;
		padding: 0.75rem;
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
</style>
