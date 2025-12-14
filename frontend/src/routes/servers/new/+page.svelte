<script>
	import { goto } from '$app/navigation';
	import { logout } from '$lib/api.js';
	import * as api from '$lib/api.js';

	let name = '';
	let configuredIP = '';
	let allowAnyIP = false;
	let error = '';
	let loading = false;
	let success = false;
	let createdServer = null;
	let token = '';
	let agentKey = '';
	let installCommand = '';

	async function handleLogout() {
		try {
			await logout();
			goto('/login');
		} catch (err) {
			console.error('Logout failed:', err);
			goto('/login');
		}
	}

	async function handleSubmit(e) {
		e.preventDefault();
		error = '';
		loading = true;

		try {
			const response = await api.createServer(name, configuredIP, allowAnyIP);
			success = true;
			createdServer = response.server;
			token = response.token;
			agentKey = response.agent_key;
			// Generate install command on frontend
			installCommand = `curl -sSL https://get.watchflare.io/ | bash -s -- --token ${token}`;
		} catch (err) {
			error = err.message || 'Failed to create server';
		} finally {
			loading = false;
		}
	}

	function copyToClipboard(text) {
		navigator.clipboard.writeText(text);
	}
</script>

<svelte:head>
	<title>Add Server - Watchflare</title>
</svelte:head>

<div class="container">
	<nav class="navbar">
		<div class="nav-content">
			<h1>Watchflare</h1>
			<div class="nav-actions">
				<a href="/" class="nav-link">Dashboard</a>
				<a href="/servers" class="nav-link active">Servers</a>
				<a href="/settings" class="nav-link">Settings</a>
				<button on:click={handleLogout} class="logout-btn">Logout</button>
			</div>
		</div>
	</nav>

	<main class="main">
		<div class="back-link">
			<a href="/servers">← Back to Servers</a>
		</div>

		{#if !success}
			<div class="card">
				<h2>Add New Server</h2>

				<form on:submit={handleSubmit}>
					<div class="form-group">
						<label for="name">
							Server Name <span class="required">*</span>
						</label>
						<input
							id="name"
							type="text"
							bind:value={name}
							required
							placeholder="e.g., web-server-01"
						/>
					</div>

					<div class="form-group">
						<label for="ip">
							Configured IP Address <span class="required">*</span>
						</label>
						<input
							id="ip"
							type="text"
							bind:value={configuredIP}
							required
							placeholder="e.g., 192.168.1.100"
						/>
						<p class="help-text">The IP address you expect this server to connect from</p>
					</div>

					<div class="form-group">
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={allowAnyIP} />
							<span>Allow registration from any IP address</span>
						</label>
						<p class="help-text checkbox-help">
							If enabled, the server can register from any IP address
						</p>
					</div>

					{#if error}
						<div class="error-box">
							<p>{error}</p>
						</div>
					{/if}

					<div class="form-actions">
						<button type="submit" class="btn-primary" disabled={loading}>
							{loading ? 'Creating...' : 'Create Server'}
						</button>
						<button type="button" class="btn-secondary" on:click={() => goto('/servers')}>
							Cancel
						</button>
					</div>
				</form>
			</div>
		{:else}
			<div class="card">
				<div class="success-header">
					<div class="success-icon">✓</div>
					<div>
						<h2>Server Created Successfully!</h2>
						<p>Server "{createdServer.name}" has been created with status: pending</p>
					</div>
				</div>

				<div class="token-section">
					<div class="token-item">
						<label>Registration Token</label>
						<div class="input-with-button">
							<input type="text" readonly value={token} class="code-input" />
							<button class="copy-btn" on:click={() => copyToClipboard(token)}>Copy</button>
						</div>
						<p class="warning-text">⚠️ Save this token securely. It won't be shown again!</p>
					</div>

					<div class="token-item">
						<label>Agent Key</label>
						<div class="input-with-button">
							<input type="text" readonly value={agentKey} class="code-input" />
							<button class="copy-btn" on:click={() => copyToClipboard(agentKey)}>Copy</button>
						</div>
					</div>

					<div class="token-item">
						<label>Installation Command</label>
						<div class="code-block-wrapper">
							<pre class="code-block">{installCommand}</pre>
							<button class="copy-btn-absolute" on:click={() => copyToClipboard(installCommand)}>
								Copy
							</button>
						</div>
					</div>
				</div>

				<div class="info-box">
					<h3>Next Steps:</h3>
					<ol>
						<li>Copy the registration token and keep it secure</li>
						<li>Run the installation command on your server</li>
						<li>The agent will register automatically using the token</li>
						<li>Once registered, the server status will change to "online"</li>
					</ol>
				</div>

				<div class="form-actions">
					<button class="btn-primary" on:click={() => goto(`/servers/${createdServer.id}`)}>
						View Server Details
					</button>
					<button class="btn-secondary" on:click={() => goto('/servers')}>Back to List</button>
				</div>
			</div>
		{/if}
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

	.nav-link.active {
		background-color: #edf2f7;
		color: #667eea;
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
		padding: 2rem;
	}

	.back-link {
		margin-bottom: 1.5rem;
	}

	.back-link a {
		color: #667eea;
		text-decoration: none;
		font-weight: 500;
	}

	.back-link a:hover {
		color: #5a67d8;
	}

	.card {
		background: white;
		padding: 2.5rem;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	h2 {
		margin: 0 0 1.5rem 0;
		font-size: 1.75rem;
		color: #1a202c;
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

	.required {
		color: #e53e3e;
	}

	input[type='text'],
	select {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		font-size: 1rem;
		transition: border-color 0.2s;
		box-sizing: border-box;
	}

	input[type='text']:focus,
	select:focus {
		outline: none;
		border-color: #667eea;
	}

	.help-text {
		margin: 0.5rem 0 0 0;
		font-size: 0.875rem;
		color: #718096;
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		cursor: pointer;
		font-weight: normal;
	}

	.checkbox-label input[type='checkbox'] {
		width: auto;
		margin-right: 0.5rem;
	}

	.checkbox-help {
		margin-left: 1.5rem;
	}

	.error-box {
		background: #fed7d7;
		color: #c53030;
		padding: 1rem;
		border-radius: 6px;
		border: 1px solid #fc8181;
		margin-bottom: 1.5rem;
	}

	.error-box p {
		margin: 0;
	}

	.form-actions {
		display: flex;
		gap: 1rem;
		margin-top: 2rem;
	}

	.btn-primary {
		flex: 1;
		padding: 0.75rem 1.5rem;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		border: none;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		transition: transform 0.2s;
	}

	.btn-primary:hover:not(:disabled) {
		transform: translateY(-1px);
	}

	.btn-primary:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.btn-secondary {
		padding: 0.75rem 1.5rem;
		background: white;
		color: #4a5568;
		border: 2px solid #e2e8f0;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	.btn-secondary:hover {
		background: #f7fafc;
	}

	.success-header {
		display: flex;
		gap: 1rem;
		margin-bottom: 2rem;
		padding-bottom: 1.5rem;
		border-bottom: 1px solid #e2e8f0;
	}

	.success-icon {
		width: 3rem;
		height: 3rem;
		background: #c6f6d5;
		color: #2f855a;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.5rem;
		font-weight: bold;
		flex-shrink: 0;
	}

	.success-header h2 {
		margin: 0 0 0.25rem 0;
		color: #2f855a;
	}

	.success-header p {
		margin: 0;
		color: #718096;
		font-size: 0.875rem;
	}

	.token-section {
		margin-bottom: 2rem;
	}

	.token-item {
		margin-bottom: 1.5rem;
	}

	.token-item label {
		display: block;
		margin-bottom: 0.5rem;
		color: #4a5568;
		font-weight: 600;
		font-size: 0.875rem;
	}

	.input-with-button {
		display: flex;
		gap: 0.5rem;
	}

	.code-input {
		flex: 1;
		padding: 0.75rem;
		background: #f7fafc;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		font-family: 'Monaco', 'Courier New', monospace;
		font-size: 0.875rem;
	}

	.copy-btn {
		padding: 0.75rem 1rem;
		background: #edf2f7;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	.copy-btn:hover {
		background: #e2e8f0;
	}

	.warning-text {
		margin: 0.5rem 0 0 0;
		color: #e53e3e;
		font-size: 0.875rem;
	}

	.code-block-wrapper {
		position: relative;
	}

	.code-block {
		background: #1a202c;
		color: #e2e8f0;
		padding: 1rem;
		border-radius: 6px;
		overflow-x: auto;
		font-family: 'Monaco', 'Courier New', monospace;
		font-size: 0.875rem;
		margin: 0;
	}

	.copy-btn-absolute {
		position: absolute;
		top: 0.5rem;
		right: 0.5rem;
		padding: 0.5rem 0.75rem;
		background: #2d3748;
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 0.75rem;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	.copy-btn-absolute:hover {
		background: #4a5568;
	}

	.info-box {
		background: linear-gradient(135deg, #667eea15 0%, #764ba215 100%);
		border-left: 4px solid #667eea;
		padding: 1.5rem;
		border-radius: 6px;
		margin-bottom: 2rem;
	}

	.info-box h3 {
		margin: 0 0 1rem 0;
		color: #1a202c;
		font-size: 1rem;
		font-weight: 600;
	}

	.info-box ol {
		margin: 0;
		padding-left: 1.5rem;
		color: #4a5568;
	}

	.info-box li {
		margin-bottom: 0.5rem;
		font-size: 0.875rem;
	}

	.info-box li:last-child {
		margin-bottom: 0;
	}
</style>
