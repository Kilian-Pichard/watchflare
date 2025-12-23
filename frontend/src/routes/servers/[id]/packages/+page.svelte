<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { logout } from '$lib/api.js';
	import * as api from '$lib/api.js';

	let packages = [];
	let stats = null;
	let collections = [];
	let loading = true;
	let error = '';
	let searchTerm = '';
	let selectedManager = '';
	let totalCount = 0;
	let limit = 50;
	let offset = 0;
	let showCollections = false;

	$: serverId = $page.params.id;
	$: currentPage = Math.floor(offset / limit) + 1;
	$: totalPages = Math.ceil(totalCount / limit);

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
		await loadData();
	});

	async function loadData() {
		loading = true;
		try {
			// Load packages and stats in parallel
			const [packagesData, statsData, collectionsData] = await Promise.all([
				api.getServerPackages(serverId, {
					limit,
					offset,
					package_manager: selectedManager || undefined,
					search: searchTerm || undefined
				}),
				api.getPackageStats(serverId),
				api.getPackageCollections(serverId, { limit: 10 })
			]);

			packages = packagesData.packages || [];
			totalCount = packagesData.total_count || 0;
			stats = statsData;
			collections = collectionsData.collections || [];
		} catch (err) {
			error = err.message || 'Failed to load packages';
		} finally {
			loading = false;
		}
	}

	async function handleSearch() {
		offset = 0; // Reset to first page
		await loadData();
	}

	async function handleFilterChange() {
		offset = 0; // Reset to first page
		await loadData();
	}

	async function nextPage() {
		if (offset + limit < totalCount) {
			offset += limit;
			await loadData();
		}
	}

	async function prevPage() {
		if (offset > 0) {
			offset -= limit;
			await loadData();
		}
	}

	function formatDate(dateString) {
		if (!dateString) return '-';
		return new Date(dateString).toLocaleString('fr-FR');
	}

	function getPackageManagerBadge(manager) {
		const badges = {
			brew: { class: 'badge-brew', label: 'Homebrew' },
			dpkg: { class: 'badge-dpkg', label: 'apt/dpkg' },
			rpm: { class: 'badge-rpm', label: 'yum/rpm' },
			pacman: { class: 'badge-pacman', label: 'pacman' }
		};
		return badges[manager] || { class: 'badge-default', label: manager };
	}
</script>

<svelte:head>
	<title>Packages - Watchflare</title>
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
			<a href="/servers/{serverId}">← Back to Server</a>
		</div>

		{#if error}
			<div class="error-box">
				<p>{error}</p>
			</div>
		{/if}

		<!-- Stats Card -->
		{#if stats}
			<div class="card stats-card">
				<h2>Package Statistics</h2>
				<div class="stats-grid">
					<div class="stat-item">
						<div class="stat-value">{stats.total_packages}</div>
						<div class="stat-label">Total Packages</div>
					</div>
					<div class="stat-item">
						<div class="stat-value">{stats.recent_changes}</div>
						<div class="stat-label">Recent Changes</div>
					</div>
					{#each stats.by_package_manager || [] as pm}
						<div class="stat-item">
							<div class="stat-value">{pm.count}</div>
							<div class="stat-label">{getPackageManagerBadge(pm.package_manager).label}</div>
						</div>
					{/each}
				</div>
				{#if stats.last_collection}
					<div class="last-collection">
						<p>
							Last collection: {formatDate(stats.last_collection.timestamp)} ({stats
								.last_collection.collection_type}, {stats.last_collection.duration_ms}ms)
						</p>
					</div>
				{/if}
			</div>
		{/if}

		<!-- Collections Toggle -->
		<div class="card">
			<button class="toggle-btn" on:click={() => (showCollections = !showCollections)}>
				{showCollections ? '▼' : '▶'} Collection History ({collections.length})
			</button>

			{#if showCollections}
				<div class="collections-list">
					{#each collections as collection}
						<div class="collection-item">
							<div class="collection-header">
								<span class="collection-time">{formatDate(collection.timestamp)}</span>
								<span class="collection-type {collection.collection_type}">
									{collection.collection_type}
								</span>
								<span
									class="collection-status {collection.status === 'success'
										? 'status-success'
										: 'status-error'}"
								>
									{collection.status}
								</span>
							</div>
							<div class="collection-details">
								<span>{collection.package_count} packages</span>
								<span>{collection.changes_count} changes</span>
								<span>{collection.duration_ms}ms</span>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Filters Card -->
		<div class="card filters-card">
			<div class="filters">
				<div class="search-box">
					<input
						type="text"
						placeholder="Search packages..."
						bind:value={searchTerm}
						on:keyup={(e) => e.key === 'Enter' && handleSearch()}
					/>
					<button on:click={handleSearch}>Search</button>
				</div>
				<div class="filter-select">
					<select bind:value={selectedManager} on:change={handleFilterChange}>
						<option value="">All Package Managers</option>
						<option value="brew">Homebrew</option>
						<option value="dpkg">apt/dpkg</option>
						<option value="rpm">yum/rpm</option>
						<option value="pacman">pacman</option>
					</select>
				</div>
			</div>
		</div>

		<!-- Packages List -->
		<div class="card">
			<div class="card-header">
				<h3>Installed Packages</h3>
				<div class="package-count">{totalCount} total</div>
			</div>

			{#if loading}
				<div class="loading">Loading packages...</div>
			{:else if packages.length === 0}
				<div class="empty-state">
					<p>No packages found</p>
				</div>
			{:else}
				<div class="packages-table">
					<table>
						<thead>
							<tr>
								<th>Name</th>
								<th>Version</th>
								<th>Manager</th>
								<th>Description</th>
								<th>Last Seen</th>
							</tr>
						</thead>
						<tbody>
							{#each packages as pkg}
								<tr>
									<td class="package-name">{pkg.name}</td>
									<td class="package-version">{pkg.version}</td>
									<td>
										<span class="badge {getPackageManagerBadge(pkg.package_manager).class}">
											{getPackageManagerBadge(pkg.package_manager).label}
										</span>
									</td>
									<td class="package-description">{pkg.description || '-'}</td>
									<td class="package-date">{formatDate(pkg.last_seen)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Pagination -->
				{#if totalPages > 1}
					<div class="pagination">
						<button on:click={prevPage} disabled={offset === 0} class="pagination-btn">
							Previous
						</button>
						<span class="pagination-info">
							Page {currentPage} of {totalPages}
						</span>
						<button
							on:click={nextPage}
							disabled={offset + limit >= totalCount}
							class="pagination-btn"
						>
							Next
						</button>
					</div>
				{/if}
			{/if}
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

	.error-box {
		background: #fed7d7;
		color: #c53030;
		padding: 1rem;
		border-radius: 6px;
		border: 1px solid #fc8181;
		margin-bottom: 1.5rem;
	}

	.card {
		background: white;
		padding: 2rem;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		margin-bottom: 1.5rem;
	}

	.stats-card h2 {
		margin: 0 0 1.5rem 0;
		font-size: 1.5rem;
		color: #1a202c;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
		gap: 1.5rem;
		margin-bottom: 1rem;
	}

	.stat-item {
		text-align: center;
		padding: 1rem;
		background: #f7fafc;
		border-radius: 8px;
	}

	.stat-value {
		font-size: 2rem;
		font-weight: 700;
		color: #667eea;
		margin-bottom: 0.5rem;
	}

	.stat-label {
		font-size: 0.875rem;
		color: #718096;
		font-weight: 500;
	}

	.last-collection {
		margin-top: 1rem;
		padding-top: 1rem;
		border-top: 1px solid #e2e8f0;
	}

	.last-collection p {
		margin: 0;
		color: #718096;
		font-size: 0.875rem;
	}

	.toggle-btn {
		background: none;
		border: none;
		color: #667eea;
		font-weight: 600;
		cursor: pointer;
		font-size: 1rem;
		padding: 0;
		margin-bottom: 1rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.toggle-btn:hover {
		color: #5a67d8;
	}

	.collections-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.collection-item {
		padding: 1rem;
		background: #f7fafc;
		border-radius: 8px;
		border-left: 4px solid #667eea;
	}

	.collection-header {
		display: flex;
		gap: 1rem;
		align-items: center;
		margin-bottom: 0.5rem;
		flex-wrap: wrap;
	}

	.collection-time {
		font-weight: 500;
		color: #1a202c;
		font-size: 0.875rem;
	}

	.collection-type {
		padding: 0.25rem 0.75rem;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
	}

	.collection-type.full {
		background: #bee3f8;
		color: #2c5282;
	}

	.collection-type.delta {
		background: #c6f6d5;
		color: #2f855a;
	}

	.collection-status {
		padding: 0.25rem 0.75rem;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
	}

	.status-success {
		background: #c6f6d5;
		color: #2f855a;
	}

	.status-error {
		background: #fed7d7;
		color: #c53030;
	}

	.collection-details {
		display: flex;
		gap: 1.5rem;
		font-size: 0.875rem;
		color: #718096;
	}

	.filters-card {
		padding: 1.5rem;
	}

	.filters {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.search-box {
		flex: 1;
		min-width: 300px;
		display: flex;
		gap: 0.5rem;
	}

	.search-box input {
		flex: 1;
		padding: 0.75rem;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	.search-box input:focus {
		outline: none;
		border-color: #667eea;
	}

	.search-box button {
		padding: 0.75rem 1.5rem;
		background: #667eea;
		color: white;
		border: none;
		border-radius: 6px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	.search-box button:hover {
		background: #5a67d8;
	}

	.filter-select select {
		padding: 0.75rem;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		background: white;
		font-size: 0.875rem;
		cursor: pointer;
	}

	.filter-select select:focus {
		outline: none;
		border-color: #667eea;
	}

	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.5rem;
	}

	h3 {
		margin: 0;
		font-size: 1.25rem;
		color: #1a202c;
	}

	.package-count {
		color: #718096;
		font-size: 0.875rem;
		font-weight: 500;
	}

	.loading {
		text-align: center;
		padding: 3rem;
		color: #718096;
	}

	.empty-state {
		text-align: center;
		padding: 3rem;
		color: #718096;
	}

	.packages-table {
		overflow-x: auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
	}

	thead {
		background: #f7fafc;
	}

	th {
		text-align: left;
		padding: 0.75rem 1rem;
		font-size: 0.75rem;
		font-weight: 600;
		color: #718096;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	tbody tr {
		border-bottom: 1px solid #e2e8f0;
	}

	tbody tr:hover {
		background: #f7fafc;
	}

	td {
		padding: 1rem;
		font-size: 0.875rem;
	}

	.package-name {
		font-weight: 600;
		color: #1a202c;
		font-family: 'Monaco', 'Courier New', monospace;
	}

	.package-version {
		color: #4a5568;
		font-family: 'Monaco', 'Courier New', monospace;
	}

	.package-description {
		color: #718096;
		max-width: 400px;
	}

	.package-date {
		color: #718096;
		font-size: 0.8125rem;
	}

	.badge {
		display: inline-block;
		padding: 0.25rem 0.75rem;
		border-radius: 12px;
		font-size: 0.75rem;
		font-weight: 600;
	}

	.badge-brew {
		background: #fef5e7;
		color: #d69e2e;
	}

	.badge-dpkg {
		background: #e0f2fe;
		color: #0284c7;
	}

	.badge-rpm {
		background: #fee2e2;
		color: #dc2626;
	}

	.badge-pacman {
		background: #e0e7ff;
		color: #4338ca;
	}

	.badge-default {
		background: #e2e8f0;
		color: #4a5568;
	}

	.pagination {
		display: flex;
		justify-content: center;
		align-items: center;
		gap: 1rem;
		margin-top: 2rem;
	}

	.pagination-btn {
		padding: 0.5rem 1rem;
		background: #667eea;
		color: white;
		border: none;
		border-radius: 6px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	.pagination-btn:hover:not(:disabled) {
		background: #5a67d8;
	}

	.pagination-btn:disabled {
		background: #cbd5e0;
		cursor: not-allowed;
	}

	.pagination-info {
		color: #718096;
		font-size: 0.875rem;
		font-weight: 500;
	}
</style>
