<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { logout } from '$lib/api.js';
	import * as api from '$lib/api.js';

	let server = null;
	let loading = true;
	let error = '';
	let showDeleteConfirm = false;
	let showRegenerateConfirm = false;
	let showChangeIP = false;
	let newIP = '';
	let regeneratedToken = '';
	let regeneratedCommand = '';

	$: serverId = $page.params.id;

	async function handleLogout() {
		try {
			await logout();
			goto('/login');
		} catch (err) {
			console.error('Logout failed:', err);
			goto('/login');
		}
	}

	onMount(async () => {
		await loadServer();
	});

	async function loadServer() {
		try {
			const response = await api.getServer(serverId);
			server = response.server;
		} catch (err) {
			error = err.message || 'Failed to load server';
		} finally {
			loading = false;
		}
	}

	async function handleDelete() {
		try {
			await api.deleteServer(serverId);
			goto('/servers');
		} catch (err) {
			error = err.message || 'Failed to delete server';
			showDeleteConfirm = false;
		}
	}

	async function handleRegenerateToken() {
		try {
			const response = await api.regenerateToken(serverId);
			regeneratedToken = response.token;
			// Generate install command on frontend
			regeneratedCommand = `curl -sSL https://get.watchflare.io/ | bash -s -- --token ${response.token}`;
			showRegenerateConfirm = false;
			await loadServer();
		} catch (err) {
			error = err.message || 'Failed to regenerate token';
			showRegenerateConfirm = false;
		}
	}

	async function handleChangeIP() {
		try {
			await api.updateConfiguredIP(serverId, newIP);
			showChangeIP = false;
			newIP = '';
			await loadServer();
		} catch (err) {
			error = err.message || 'Failed to update IP';
		}
	}

	function formatDate(dateString) {
		if (!dateString) return '-';
		return new Date(dateString).toLocaleString('fr-FR');
	}

	function getStatusClass(status) {
		switch (status) {
			case 'online':
				return 'status-online';
			case 'offline':
				return 'status-offline';
			case 'pending':
				return 'status-pending';
			case 'expired':
				return 'status-expired';
			default:
				return 'status-unknown';
		}
	}

	function copyToClipboard(text) {
		navigator.clipboard.writeText(text);
	}
</script>

<svelte:head>
	<title>Server Details - Watchflare</title>
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

		{#if loading}
			<div class="loading">
				<p>Loading server details...</p>
			</div>
		{:else if error && !server}
			<div class="error-box">
				<p>{error}</p>
			</div>
		{:else if server}
			{#if error}
				<div class="error-box">
					<p>{error}</p>
				</div>
			{/if}

			<!-- Header Card -->
			<div class="card header-card">
				<div class="header-content">
					<div>
						<h2>{server.name}</h2>
						<p class="agent-id">Agent ID: {server.agent_id}</p>
					</div>
					<span class="status-badge {getStatusClass(server.status)}">
						{server.status}
					</span>
				</div>
			</div>

			<!-- Server Information Card -->
			<div class="card">
				<h3>Server Information</h3>
				<div class="info-grid">
					<div class="info-item">
						<span class="info-label">Type</span>
						<span class="info-value capitalize">{server.type}</span>
					</div>
					<div class="info-item">
						<span class="info-label">Configured IP</span>
						<span class="info-value">{server.configured_ip || 'Not set'}</span>
					</div>
					<div class="info-item">
						<span class="info-label">Allow Any IP</span>
						<span class="info-value">{server.allow_any_ip ? 'Yes' : 'No'}</span>
					</div>
					<div class="info-item">
						<span class="info-label">Created At</span>
						<span class="info-value">{formatDate(server.created_at)}</span>
					</div>
				</div>
			</div>

			<!-- Agent Information Card (only if registered) -->
			{#if server.hostname || server.ip_address_v4}
				<div class="card">
					<h3>Agent Information</h3>
					<div class="info-grid">
						{#if server.hostname}
							<div class="info-item">
								<span class="info-label">Hostname</span>
								<span class="info-value">{server.hostname}</span>
							</div>
						{/if}
						{#if server.ip_address_v4}
							<div class="info-item">
								<span class="info-label">IPv4 Address</span>
								<span class="info-value">{server.ip_address_v4}</span>
							</div>
						{/if}
						{#if server.ip_address_v6}
							<div class="info-item">
								<span class="info-label">IPv6 Address</span>
								<span class="info-value">{server.ip_address_v6}</span>
							</div>
						{/if}
						{#if server.os}
							<div class="info-item">
								<span class="info-label">Operating System</span>
								<span class="info-value">{server.os} {server.os_version || ''}</span>
							</div>
						{/if}
						{#if server.last_seen}
							<div class="info-item">
								<span class="info-label">Last Seen</span>
								<span class="info-value">{formatDate(server.last_seen)}</span>
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- Regenerated Token Card -->
			{#if regeneratedToken}
				<div class="card success-card">
					<h3>New Registration Token</h3>
					<div class="token-section">
						<div class="token-item">
							<label>Token</label>
							<div class="input-with-button">
								<input type="text" readonly value={regeneratedToken} class="code-input" />
								<button class="copy-btn" on:click={() => copyToClipboard(regeneratedToken)}>
									Copy
								</button>
							</div>
						</div>
						<div class="token-item">
							<label>Installation Command</label>
							<div class="code-block-wrapper">
								<pre class="code-block">{regeneratedCommand}</pre>
								<button
									class="copy-btn-absolute"
									on:click={() => copyToClipboard(regeneratedCommand)}
								>
									Copy
								</button>
							</div>
						</div>
						<p class="warning-text">⚠️ Save this token securely. It won't be shown again!</p>
					</div>
				</div>
			{/if}

			<!-- Actions Card -->
			<div class="card">
				<h3>Actions</h3>
				<div class="actions-grid">
					{#if server.status === 'pending' || server.status === 'expired'}
						<button class="btn-action btn-primary" on:click={() => (showRegenerateConfirm = true)}>
							Regenerate Token
						</button>
					{/if}
					<button class="btn-action btn-secondary" on:click={() => (showChangeIP = true)}>
						Change Configured IP
					</button>
					{#if server.status === 'pending' || server.status === 'expired'}
						<button class="btn-action btn-danger" on:click={() => (showDeleteConfirm = true)}>
							Delete Server
						</button>
					{/if}
				</div>
			</div>

			<!-- Expiration Info -->
			{#if server.status === 'pending' && server.expires_at}
				<div class="warning-card">
					<p>
						<strong>Note:</strong> This registration token will expire on {formatDate(
							server.expires_at
						)}
					</p>
				</div>
			{/if}
		{/if}
	</main>
</div>

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm}
	<div class="modal-overlay" on:click={() => (showDeleteConfirm = false)}>
		<div class="modal" on:click={(e) => e.stopPropagation()}>
			<h3>Delete Server</h3>
			<p>Are you sure you want to delete "{server.name}"? This action cannot be undone.</p>
			<div class="modal-actions">
				<button class="btn-danger" on:click={handleDelete}>Delete</button>
				<button class="btn-secondary" on:click={() => (showDeleteConfirm = false)}>
					Cancel
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Regenerate Token Confirmation Modal -->
{#if showRegenerateConfirm}
	<div class="modal-overlay" on:click={() => (showRegenerateConfirm = false)}>
		<div class="modal" on:click={(e) => e.stopPropagation()}>
			<h3>Regenerate Token</h3>
			<p>
				This will invalidate the current registration token and generate a new one. Continue?
			</p>
			<div class="modal-actions">
				<button class="btn-primary" on:click={handleRegenerateToken}>Regenerate</button>
				<button class="btn-secondary" on:click={() => (showRegenerateConfirm = false)}>
					Cancel
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Change IP Modal -->
{#if showChangeIP}
	<div class="modal-overlay" on:click={() => (showChangeIP = false)}>
		<div class="modal" on:click={(e) => e.stopPropagation()}>
			<h3>Change Configured IP</h3>
			<form
				on:submit={(e) => {
					e.preventDefault();
					handleChangeIP();
				}}
			>
				<div class="form-group">
					<label for="new-ip">New IP Address</label>
					<input
						id="new-ip"
						type="text"
						bind:value={newIP}
						required
						placeholder="e.g., 192.168.1.200"
					/>
				</div>
				<div class="modal-actions">
					<button type="submit" class="btn-primary">Update</button>
					<button
						type="button"
						class="btn-secondary"
						on:click={() => {
							showChangeIP = false;
							newIP = '';
						}}
					>
						Cancel
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

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
		max-width: 1200px;
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

	.loading {
		background: white;
		padding: 3rem;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		text-align: center;
		color: #718096;
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

	.card {
		background: white;
		padding: 2rem;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		margin-bottom: 1.5rem;
	}

	.header-card {
		padding: 1.5rem 2rem;
	}

	.header-content {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	h2 {
		margin: 0 0 0.5rem 0;
		font-size: 2rem;
		color: #1a202c;
	}

	.agent-id {
		margin: 0;
		color: #718096;
		font-size: 0.875rem;
	}

	h3 {
		margin: 0 0 1.5rem 0;
		font-size: 1.25rem;
		color: #1a202c;
		font-weight: 600;
	}

	.status-badge {
		display: inline-block;
		padding: 0.5rem 1rem;
		border-radius: 12px;
		font-size: 0.875rem;
		font-weight: 600;
		text-transform: capitalize;
	}

	.status-online {
		background: #c6f6d5;
		color: #2f855a;
	}

	.status-offline {
		background: #fed7d7;
		color: #c53030;
	}

	.status-pending {
		background: #fef5e7;
		color: #d69e2e;
	}

	.status-expired {
		background: #e2e8f0;
		color: #4a5568;
	}

	.status-unknown {
		background: #e2e8f0;
		color: #718096;
	}

	.info-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: 1.5rem;
	}

	.info-item {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.info-label {
		font-size: 0.875rem;
		font-weight: 500;
		color: #718096;
	}

	.info-value {
		font-size: 0.875rem;
		color: #1a202c;
	}

	.capitalize {
		text-transform: capitalize;
	}

	.success-card {
		background: linear-gradient(135deg, #c6f6d515 0%, #9ae6b415 100%);
		border: 1px solid #c6f6d5;
	}

	.token-section {
		margin-top: 1rem;
	}

	.token-item {
		margin-bottom: 1.5rem;
	}

	.token-item:last-child {
		margin-bottom: 0;
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
		background: white;
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

	.warning-text {
		margin: 0.5rem 0 0 0;
		color: #e53e3e;
		font-size: 0.875rem;
	}

	.actions-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.btn-action {
		padding: 0.75rem 1.5rem;
		border: none;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		transition: transform 0.2s;
	}

	.btn-action:hover {
		transform: translateY(-1px);
	}

	.btn-primary {
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
	}

	.btn-secondary {
		background: #4a5568;
		color: white;
	}

	.btn-danger {
		background: #e53e3e;
		color: white;
	}

	.warning-card {
		background: #fef5e7;
		border: 1px solid #f6e05e;
		padding: 1rem;
		border-radius: 6px;
	}

	.warning-card p {
		margin: 0;
		color: #744210;
		font-size: 0.875rem;
	}

	/* Modal Styles */
	.modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal {
		background: white;
		border-radius: 12px;
		padding: 2rem;
		max-width: 500px;
		width: 90%;
		margin: 1rem;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
	}

	.modal h3 {
		margin: 0 0 1rem 0;
		font-size: 1.25rem;
		color: #1a202c;
	}

	.modal p {
		margin: 0 0 1.5rem 0;
		color: #718096;
	}

	.form-group {
		margin-bottom: 1.5rem;
	}

	.form-group label {
		display: block;
		margin-bottom: 0.5rem;
		color: #4a5568;
		font-weight: 500;
		font-size: 0.875rem;
	}

	.form-group input {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		font-size: 1rem;
		transition: border-color 0.2s;
		box-sizing: border-box;
	}

	.form-group input:focus {
		outline: none;
		border-color: #667eea;
	}

	.modal-actions {
		display: flex;
		gap: 0.75rem;
	}

	.modal-actions button {
		flex: 1;
		padding: 0.75rem 1rem;
		border: none;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		transition: transform 0.2s;
	}

	.modal-actions button:hover {
		transform: translateY(-1px);
	}
</style>
