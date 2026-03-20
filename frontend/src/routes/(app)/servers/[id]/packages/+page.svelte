<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import * as api from '$lib/api.js';
	import * as Select from '$lib/components/ui/select';
	import { PACKAGES_PER_PAGE, COLLECTIONS_PER_PAGE } from '$lib/constants';
	import type { Server, Package, PackageStats, PackageCollection } from '$lib/types';
	import Pagination from '$lib/components/Pagination.svelte';

	let server: Server | null = $state(null);
	let packages: Package[] = $state([]);
	let stats: PackageStats | null = $state(null);
	let collections: PackageCollection[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let searchTerm = $state('');
	let selectedManager = $state('');
	let totalCount = $state(0);
	let limit = PACKAGES_PER_PAGE;
	let offset = $state(0);
	let showCollections = $state(false);

	let serverId = $derived($page.params.id);
	let managerLabel = $derived(selectedManager ? getManagerLabel(selectedManager) : 'All Package Managers');
	let managerItems = $derived(['All Package Managers', ...(stats?.by_package_manager || []).map(pm => getManagerLabel(pm.package_manager))]);
	let currentPage = $derived(Math.floor(offset / limit) + 1);
	let totalPages = $derived(Math.ceil(totalCount / limit));

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		try {
			const [serverData, packagesData, statsData, collectionsData] = await Promise.all([
				api.getServer(serverId),
				api.getServerPackages(serverId, {
					limit,
					offset,
					package_manager: selectedManager || undefined,
					search: searchTerm || undefined
				}),
				api.getPackageStats(serverId),
				api.getPackageCollections(serverId, { limit: COLLECTIONS_PER_PAGE })
			]);

			server = serverData.server;
			packages = packagesData.packages || [];
			totalCount = packagesData.total_count || 0;
			stats = statsData;
			collections = collectionsData.collections || [];
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load packages';
		} finally {
			loading = false;
		}
	}

	async function handleSearch() {
		offset = 0;
		await loadData();
	}

	async function handleFilterChange(value: string) {
		selectedManager = value;
		offset = 0;
		await loadData();
	}

	function handlePageChange(page: number) {
		offset = (page - 1) * limit;
		loadData();
	}

	function formatDate(dateString: string) {
		if (!dateString) return '-';
		return new Date(dateString).toLocaleString('fr-FR');
	}

	function getManagerColor(manager: string): string {
		const colors: Record<string, string> = {
			brew: 'bg-(--chart-4)/10 text-(--chart-4) border-(--chart-4)/20',
			dpkg: 'bg-(--chart-2)/10 text-(--chart-2) border-(--chart-2)/20',
			rpm: 'bg-(--chart-1)/10 text-(--chart-1) border-(--chart-1)/20',
			pacman: 'bg-(--chart-5)/10 text-(--chart-5) border-(--chart-5)/20'
		};
		return colors[manager] || 'bg-muted text-muted-foreground border-border';
	}

	function getManagerLabel(manager: string): string {
		const labels: Record<string, string> = {
			brew: 'Homebrew',
			dpkg: 'apt/dpkg',
			rpm: 'yum/rpm',
			pacman: 'Pacman'
		};
		return labels[manager] || manager;
	}
</script>

<svelte:head>
	<title>Packages{server ? ` - ${server.name}` : ''} - Watchflare</title>
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
	{#if server}
		<p class="text-sm text-muted-foreground mt-1 flex items-center flex-wrap gap-x-3">
			<span>{server.name}</span>
			{#if server.hostname}
				<span class="text-border">|</span>
				<span>{server.hostname}</span>
			{/if}
			{#if server.ip_address_v4 || server.configured_ip}
				<span class="text-border">|</span>
				<span>{server.ip_address_v4 || server.configured_ip}</span>
			{/if}
		</p>
	{/if}
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
		<Select.Root type="single" value={selectedManager} onValueChange={handleFilterChange}>
			<Select.Trigger items={managerItems}>
				<span>{managerLabel}</span>
			</Select.Trigger>
			<Select.Content>
				<Select.Item value="" label="All Package Managers">All Package Managers</Select.Item>
				{#each stats?.by_package_manager || [] as pm}
					<Select.Item value={pm.package_manager} label={getManagerLabel(pm.package_manager)}>
						{getManagerLabel(pm.package_manager)}
					</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
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
						<th scope="col" class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Date</th>
						<th scope="col" class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Type</th>
						<th scope="col" class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">Packages</th>
						<th scope="col" class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">Changes</th>
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
						<th scope="col" class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Name</th>
						<th scope="col" class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Version</th>
						<th scope="col" class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Manager</th>
						<th scope="col" class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">Description</th>
						<th scope="col" class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">Last Seen</th>
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
		<Pagination {currentPage} {totalPages} totalItems={totalCount} pageSize={limit} itemLabel="packages" onPageChange={handlePageChange} />
	{/if}
</div>
