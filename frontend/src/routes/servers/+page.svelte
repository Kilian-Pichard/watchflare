<script>
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { logout } from '$lib/api.js';
	import * as api from '$lib/api.js';
	import { toasts } from '$lib/stores/toasts';
	import { sidebarCollapsed } from '$lib/stores/sidebar';
	import DesktopSidebar from '$lib/components/DesktopSidebar.svelte';
	import MobileSidebar from '$lib/components/MobileSidebar.svelte';
	import Header from '$lib/components/Header.svelte';

	async function dismissReactivation(serverId) {
		try {
			await api.dismissReactivation(serverId);
			const response = await api.listServers();
			servers = response.servers || [];
		} catch (err) {
			console.error('Failed to dismiss reactivation:', err);
		}
	}

	let servers = [];
	let loading = true;
	let error = '';
	let showDeleteConfirm = false;
	let serverToDelete = null;
	let eventSource = null;

	async function handleLogout() {
		try {
			await logout();
			goto('/login');
		} catch (err) {
			console.error('Logout failed:', err);
			goto('/login');
		}
	}

	function connectSSE() {
		eventSource = new EventSource('http://localhost:8080/servers/events', {
			withCredentials: true
		});

		eventSource.addEventListener('connected', (e) => {
			const data = JSON.parse(e.data);
		});

		eventSource.addEventListener('server_update', (e) => {
			const update = JSON.parse(e.data);

			if (update.reactivated && update.hostname) {
				toasts.add(
					`Agent "${update.hostname}" was reactivated (same physical server detected via UUID)`,
					'info',
					8000
				);
			}

			const serverIndex = servers.findIndex((s) => s.id === update.id);
			if (serverIndex !== -1) {
				servers[serverIndex] = {
					...servers[serverIndex],
					status: update.status,
					ip_address_v4: update.ip_address_v4,
					ip_address_v6: update.ip_address_v6,
					configured_ip: update.configured_ip,
					ignore_ip_mismatch: update.ignore_ip_mismatch,
					last_seen: update.last_seen
				};
				servers = [...servers];
			}
		});

		eventSource.onerror = (err) => {
			console.error('SSE error:', err);
			setTimeout(() => {
				if (eventSource) {
					eventSource.close();
					connectSSE();
				}
			}, 5000);
		};
	}

	onMount(async () => {
		try {
			const response = await api.listServers();
			servers = response.servers || [];
			connectSSE();
		} catch (err) {
			error = err.message || 'Failed to load servers';
		} finally {
			loading = false;
		}
	});

	onDestroy(() => {
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
	});

	function getStatusClass(status) {
		switch (status) {
			case 'online':
				return 'bg-success/10 text-success border-success/20';
			case 'offline':
				return 'bg-muted text-muted-foreground border-border';
			case 'pending':
				return 'bg-warning/10 text-warning border-warning/20';
			case 'expired':
				return 'bg-muted text-muted-foreground border-border';
			default:
				return 'bg-muted text-muted-foreground border-border';
		}
	}

	function formatDate(dateString) {
		if (!dateString) return '-';
		const date = new Date(dateString);
		const now = new Date();
		const diff = Math.floor((now - date) / 1000);

		if (diff < 60) return `${diff}s ago`;
		if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
		if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
		return date.toLocaleDateString('fr-FR');
	}

	function hasIPMismatch(server) {
		return (
			server.configured_ip &&
			server.ip_address_v4 &&
			server.configured_ip !== server.ip_address_v4 &&
			!server.ignore_ip_mismatch
		);
	}

	function openDeleteModal(server, e) {
		e.stopPropagation();
		serverToDelete = server;
		showDeleteConfirm = true;
	}

	function cancelDelete() {
		showDeleteConfirm = false;
		serverToDelete = null;
	}

	async function handleDelete() {
		if (!serverToDelete) return;

		try {
			await api.deleteServer(serverToDelete.id);
			const response = await api.listServers();
			servers = response.servers || [];
			showDeleteConfirm = false;
			serverToDelete = null;
		} catch (err) {
			error = err.message || 'Failed to delete server';
			showDeleteConfirm = false;
			serverToDelete = null;
		}
	}
</script>

<svelte:head>
	<title>Servers - Watchflare</title>
</svelte:head>

<div class="min-h-screen bg-background">
	<!-- Header -->
	<Header title="Servers" />

	<!-- Desktop Sidebar -->
	<DesktopSidebar onLogout={handleLogout} />

	<!-- Mobile Sidebar -->
	<MobileSidebar onLogout={handleLogout} />

	<main class="min-h-screen pt-16 p-4 md:p-8 md:pt-20 {$sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-64'}">
		<!-- Header -->
		<div class="mb-6 flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-semibold text-foreground">Servers</h1>
				<p class="text-sm text-muted-foreground mt-1">Manage your monitored servers</p>
			</div>
			<button
				onclick={() => goto('/servers/new')}
				class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
			>
				Add Server
			</button>
		</div>

		{#if loading}
			<div class="flex items-center justify-center py-20">
				<p class="text-muted-foreground">Loading servers...</p>
			</div>
		{:else if error}
			<div class="rounded-lg border border-destructive bg-destructive/10 p-4">
				<p class="text-sm text-destructive">{error}</p>
			</div>
		{:else if servers.length === 0}
			<div class="flex flex-col items-center justify-center rounded-lg border bg-card py-20 text-center">
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
						d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
					/>
				</svg>
				<h3 class="text-lg font-medium text-foreground mb-2">No servers configured yet</h3>
				<p class="text-sm text-muted-foreground mb-6">Add your first server to start monitoring</p>
				<button
					onclick={() => goto('/servers/new')}
					class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
				>
					Add Your First Server
				</button>
			</div>
		{:else}
			<div class="rounded-lg border bg-card">
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="border-b bg-muted/30">
								<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
									Name
								</th>
								<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
									Status
								</th>
								<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
									IP Address
								</th>
								<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
									Last Seen
								</th>
								<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
									Actions
								</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border">
							{#each servers as server}
								<tr
									onclick={() => goto(`/servers/${server.id}`)}
									class="hover:bg-muted/20 transition-colors cursor-pointer"
								>
									<td class="px-4 py-3.5">
										<div class="flex flex-col">
											<span class="font-medium text-foreground">{server.name}</span>
											{#if server.hostname}
												<span class="text-xs text-muted-foreground">{server.hostname}</span>
											{/if}
										</div>
									</td>
									<td class="px-4 py-3.5">
										<div class="flex items-center gap-2">
											<span
												class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(server.status)}"
											>
												<span class="h-1.5 w-1.5 rounded-full {server.status === 'online' ? 'bg-success' : 'bg-muted-foreground'}"></span>
												{server.status}
											</span>
											{#if hasIPMismatch(server)}
												<span
													class="inline-flex items-center text-warning"
													title="IP mismatch detected"
												>
													<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
														<path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
													</svg>
												</span>
											{/if}
											{#if server.reactivated_at}
												<span
													class="inline-flex items-center gap-1 rounded-full border border-primary/20 bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
													title="Agent was reactivated (same physical server via UUID)"
												>
													<svg class="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
														<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd"/>
													</svg>
													Reactivated
													<button
														onclick={(e) => {
															e.stopPropagation();
															dismissReactivation(server.id);
														}}
														class="ml-0.5 text-primary hover:text-primary/80"
														title="Dismiss"
													>
														<svg class="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
															<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"/>
														</svg>
													</button>
												</span>
											{/if}
										</div>
									</td>
									<td class="px-4 py-3.5 text-sm text-foreground">
										{server.ip_address_v4 || server.configured_ip || '-'}
									</td>
									<td class="px-4 py-3.5 text-right text-sm text-muted-foreground">
										{formatDate(server.last_seen)}
									</td>
									<td class="px-4 py-3.5 text-right">
										<div class="flex items-center justify-end gap-3">
											<button
												onclick={(e) => {
													e.stopPropagation();
													goto(`/servers/${server.id}`);
												}}
												class="text-sm font-medium text-primary hover:text-primary/80 transition-colors"
											>
												View
											</button>
											<button
												onclick={(e) => openDeleteModal(server, e)}
												class="text-sm font-medium text-destructive hover:text-destructive/80 transition-colors"
											>
												Delete
											</button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	</main>
</div>

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={cancelDelete}
	>
		<div
			class="w-full max-w-md rounded-lg border bg-card p-6 shadow-lg"
			onclick={(e) => e.stopPropagation()}
		>
			<h3 class="text-lg font-semibold text-foreground mb-3">Confirm Delete</h3>
			<p class="text-sm text-muted-foreground mb-4">
				Are you sure you want to delete the server "{serverToDelete?.name}"?
			</p>
			{#if serverToDelete?.status === 'online'}
				<div class="mb-4 rounded-md border border-primary/20 bg-primary/5 p-3">
					<p class="text-sm text-foreground">
						Note: This will remove the server from the database, but the agent will remain
						installed on the server. You will need to uninstall it manually.
					</p>
				</div>
			{/if}
			<p class="text-sm font-medium text-destructive mb-6">This action cannot be undone.</p>
			<div class="flex gap-3 justify-end">
				<button
					onclick={cancelDelete}
					class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
				>
					Cancel
				</button>
				<button
					onclick={handleDelete}
					class="rounded-lg bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90"
				>
					Delete Server
				</button>
			</div>
		</div>
	</div>
{/if}
