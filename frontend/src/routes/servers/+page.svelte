<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { logout } from '$lib/api.js';
	import * as api from '$lib/api.js';

	let servers = [];
	let loading = true;
	let error = '';

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
		try {
			const response = await api.listServers();
			servers = response.servers || [];
		} catch (err) {
			error = err.message || 'Failed to load servers';
		} finally {
			loading = false;
		}
	});

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

	function formatDate(dateString) {
		if (!dateString) return '-';
		return new Date(dateString).toLocaleString('fr-FR');
	}
</script>

<svelte:head>
	<title>Servers - Watchflare</title>
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
		<div class="header">
			<h2>Servers</h2>
			<button class="btn-primary" on:click={() => goto('/servers/new')}>Add Server</button>
		</div>

		{#if loading}
			<div class="loading">
				<p>Loading servers...</p>
			</div>
		{:else if error}
			<div class="error-box">
				<p>{error}</p>
			</div>
		{:else if servers.length === 0}
			<div class="empty-state">
				<h3>No servers configured yet</h3>
				<p>Add your first server to start monitoring</p>
				<button class="btn-primary" on:click={() => goto('/servers/new')}>
					Add Your First Server
				</button>
			</div>
		{:else}
			<div class="servers-table">
				<table>
					<thead>
						<tr>
							<th>Name</th>
							<th>Type</th>
							<th>Status</th>
							<th>IP Address</th>
							<th>Last Seen</th>
							<th>Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each servers as server}
							<tr on:click={() => goto(`/servers/${server.id}`)}>
								<td>
									<div class="server-name">{server.name}</div>
									{#if server.hostname}
										<div class="server-hostname">{server.hostname}</div>
									{/if}
								</td>
								<td class="capitalize">{server.type}</td>
								<td>
									<span class="status-badge {getStatusClass(server.status)}">
										{server.status}
									</span>
								</td>
								<td>{server.ip_address_v4 || server.configured_ip || '-'}</td>
								<td>{formatDate(server.last_seen)}</td>
								<td>
									<button
										class="link-btn"
										on:click={(e) => {
											e.stopPropagation();
											goto(`/servers/${server.id}`);
										}}
									>
										View Details
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
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
		max-width: 1400px;
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
		max-width: 1400px;
		margin: 0 auto;
		padding: 2rem;
	}

	.header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 2rem;
	}

	h2 {
		margin: 0;
		font-size: 2rem;
		color: #1a202c;
	}

	.btn-primary {
		padding: 0.75rem 1.5rem;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		border: none;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		transition: transform 0.2s;
	}

	.btn-primary:hover {
		transform: translateY(-1px);
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
	}

	.empty-state {
		background: white;
		padding: 4rem 2rem;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		text-align: center;
	}

	.empty-state h3 {
		margin: 0 0 0.5rem 0;
		color: #1a202c;
		font-size: 1.5rem;
	}

	.empty-state p {
		margin: 0 0 2rem 0;
		color: #718096;
	}

	.servers-table {
		background: white;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		overflow: hidden;
	}

	table {
		width: 100%;
		border-collapse: collapse;
	}

	thead {
		background: #f7fafc;
	}

	th {
		padding: 1rem;
		text-align: left;
		font-size: 0.75rem;
		font-weight: 600;
		color: #718096;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-bottom: 1px solid #e2e8f0;
	}

	tbody tr {
		border-bottom: 1px solid #e2e8f0;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	tbody tr:hover {
		background-color: #f7fafc;
	}

	tbody tr:last-child {
		border-bottom: none;
	}

	td {
		padding: 1rem;
		color: #1a202c;
	}

	.server-name {
		font-weight: 600;
		color: #1a202c;
	}

	.server-hostname {
		font-size: 0.75rem;
		color: #718096;
		margin-top: 0.25rem;
	}

	.capitalize {
		text-transform: capitalize;
	}

	.status-badge {
		display: inline-block;
		padding: 0.25rem 0.75rem;
		border-radius: 12px;
		font-size: 0.75rem;
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

	.link-btn {
		background: none;
		border: none;
		color: #667eea;
		font-weight: 500;
		cursor: pointer;
		padding: 0;
	}

	.link-btn:hover {
		color: #5a67d8;
		text-decoration: underline;
	}
</style>
