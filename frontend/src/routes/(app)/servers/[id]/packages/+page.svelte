<script>
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
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

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		try {
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
		offset = 0;
		await loadData();
	}

	async function handleFilterChange() {
		offset = 0;
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

	function getManagerColor(manager) {
		const colors = {
			brew: 'bg-[var(--chart-4)]/10 text-[var(--chart-4)] border-[var(--chart-4)]/20',
			dpkg: 'bg-[var(--chart-2)]/10 text-[var(--chart-2)] border-[var(--chart-2)]/20',
			rpm: 'bg-[var(--chart-1)]/10 text-[var(--chart-1)] border-[var(--chart-1)]/20',
			pacman: 'bg-[var(--chart-5)]/10 text-[var(--chart-5)] border-[var(--chart-5)]/20'
		};
		return colors[manager] || 'bg-muted text-muted-foreground border-border';
	}

	function getManagerLabel(manager) {
		const labels = {
			brew: 'Homebrew',
			dpkg: 'apt/dpkg',
			rpm: 'yum/rpm',
			pacman: 'Pacman'
		};
		return labels[manager] || manager;
	}
</script>

<svelte:head>
	<title>Packages - Watchflare</title>
</svelte:head>

<!-- Back Link -->
<div class="mb-6">
	<a
		href="/servers/{serverId}"
		class="inline-flex items-center gap-2 text-sm font-medium text-primary hover:text-primary/80 transition-colors"
	>
		<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
		</svg>
		Back to Server
	</a>
</div>

<!-- Header -->
<div class="mb-6">
	<h1 class="text-2xl font-semibold text-foreground">Package Inventory</h1>
	<p class="text-sm text-muted-foreground mt-1">
		Installed packages and software on this server
	</p>
</div>

<!-- Error -->
{#if error}
	<div class="mb-6 rounded-lg border border-destructive bg-destructive/10 p-4">
		<p class="text-sm text-destructive">{error}</p>
	</div>
{/if}

<!-- Stats -->
{#if stats}
	<div class="grid gap-4 md:grid-cols-4 mb-6">
		<div class="rounded-lg border bg-card p-4">
			<p class="text-xs text-muted-foreground mb-1">Total Packages</p>
			<p class="text-2xl font-semibold text-foreground">{stats.total_packages || 0}</p>
		</div>
		<div class="rounded-lg border bg-card p-4">
			<p class="text-xs text-muted-foreground mb-1">Recent Changes (30d)</p>
			<p class="text-2xl font-semibold text-foreground">{stats.recent_changes || 0}</p>
		</div>
		{#each (stats.by_package_manager || []).slice(0, 2) as pm}
			<div class="rounded-lg border bg-card p-4">
				<p class="text-xs text-muted-foreground mb-1">{getManagerLabel(pm.package_manager)}</p>
				<p class="text-2xl font-semibold text-foreground">{pm.count}</p>
			</div>
		{/each}
	</div>
{/if}

<!-- Search & Filters -->
<div class="mb-6 rounded-lg border bg-card p-4">
	<div class="flex flex-col sm:flex-row gap-4">
		<div class="flex-1">
			<input
				type="text"
				bind:value={searchTerm}
				onkeydown={(e) => e.key === 'Enter' && handleSearch()}
				placeholder="Search packages..."
				class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
			/>
		</div>
		<select
			bind:value={selectedManager}
			onchange={handleFilterChange}
			class="rounded-lg border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
		>
			<option value="">All Package Managers</option>
			{#each stats?.by_package_manager || [] as pm}
				<option value={pm.package_manager}>{getManagerLabel(pm.package_manager)}</option>
			{/each}
		</select>
		<button
			onclick={handleSearch}
			class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
		>
			Search
		</button>
	</div>
</div>

<!-- Collections Toggle -->
<div class="mb-6">
	<button
		onclick={() => showCollections = !showCollections}
		class="inline-flex items-center gap-2 text-sm font-medium text-primary hover:text-primary/80 transition-colors"
	>
		<svg
			class="h-4 w-4 transition-transform {showCollections ? 'rotate-90' : ''}"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
		</svg>
		{showCollections ? 'Hide' : 'Show'} Collection History
	</button>

	{#if showCollections && collections.length > 0}
		<div class="mt-4 rounded-lg border bg-card overflow-hidden">
			<table class="w-full">
				<thead>
					<tr class="border-b bg-muted/30">
						<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Date</th>
						<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Type</th>
						<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">Packages</th>
						<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">Changes</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each collections as collection}
						<tr class="hover:bg-muted/20">
							<td class="px-4 py-3 text-sm text-foreground">
								{formatDate(collection.collected_at)}
							</td>
							<td class="px-4 py-3">
								<span class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {collection.collection_type === 'full' ? 'bg-primary/10 text-primary border-primary/20' : 'bg-muted text-muted-foreground border-border'}">
									{collection.collection_type}
								</span>
							</td>
							<td class="px-4 py-3 text-right text-sm text-foreground">
								{collection.total_packages}
							</td>
							<td class="px-4 py-3 text-right text-sm text-foreground">
								{collection.changes_detected || 0}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Packages Table -->
<div class="rounded-lg border bg-card overflow-hidden">
	{#if loading}
		<div class="flex items-center justify-center py-20">
			<p class="text-muted-foreground">Loading packages...</p>
		</div>
	{:else if packages.length === 0}
		<div class="flex flex-col items-center justify-center py-20 text-center">
			<svg
				class="h-12 w-12 text-muted-foreground/50 mb-4"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.5"
					d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
				/>
			</svg>
			<p class="text-sm text-muted-foreground">No packages found</p>
		</div>
	{:else}
		<div class="overflow-x-auto">
			<table class="w-full">
				<thead>
					<tr class="border-b bg-muted/30">
						<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Name</th>
						<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Version</th>
						<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Manager</th>
						<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Description</th>
						<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">Last Seen</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each packages as pkg}
						<tr class="hover:bg-muted/20 transition-colors">
							<td class="px-4 py-3 text-sm font-medium text-foreground">
								{pkg.name}
							</td>
							<td class="px-4 py-3 text-sm font-mono text-muted-foreground">
								{pkg.version}
							</td>
							<td class="px-4 py-3">
								<span class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {getManagerColor(pkg.package_manager)}">
									{getManagerLabel(pkg.package_manager)}
								</span>
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground max-w-md truncate">
								{pkg.description || '-'}
							</td>
							<td class="px-4 py-3 text-right text-sm text-muted-foreground">
								{formatDate(pkg.last_seen)}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<!-- Pagination -->
		<div class="border-t bg-muted/10 px-4 py-3 flex items-center justify-between">
			<div class="text-sm text-muted-foreground">
				Showing {offset + 1} to {Math.min(offset + limit, totalCount)} of {totalCount} packages
			</div>
			<div class="flex gap-2">
				<button
					onclick={prevPage}
					disabled={offset === 0}
					class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Previous
				</button>
				<div class="flex items-center px-3 py-1.5 text-sm text-muted-foreground">
					Page {currentPage} of {totalPages}
				</div>
				<button
					onclick={nextPage}
					disabled={offset + limit >= totalCount}
					class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Next
				</button>
			</div>
		</div>
	{/if}
</div>
