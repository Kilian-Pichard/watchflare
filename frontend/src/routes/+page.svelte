<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { logout } from '$lib/api';
	import { formatPercent } from '$lib/utils';
	import {
		userStore,
		currentUser,
		serversStore,
		servers,
		metricsStore,
		metricsData,
		aggregatedStore,
		aggregatedMetrics,
		currentTimeRange,
		dashboardStats,
		alertsStore,
		uiStore,
		sseStore,
		toasts,
		sidebarCollapsed
	} from '$lib/stores';
	import DesktopSidebar from '$lib/components/DesktopSidebar.svelte';
	import MobileSidebar from '$lib/components/MobileSidebar.svelte';
	import Header from '$lib/components/Header.svelte';
	import ServerTable from '$lib/components/ServerTable.svelte';
	import DashboardStats from '$lib/components/dashboard/DashboardStats.svelte';
	import DashboardCharts from '$lib/components/dashboard/DashboardCharts.svelte';
	import DroppedMetricsAlert from '$lib/components/dashboard/DroppedMetricsAlert.svelte';
	import RightSidebar from '$lib/components/RightSidebar.svelte';
	import type { SSEEvent, TimeRange } from '$lib/types';

	// SSE unsubscribe function
	let sseUnsubscribe: (() => void) | null = null;

	// Derived states from stores
	let loading = $state(true);
	let timeRange = $derived($currentTimeRange);
	let user = $derived($currentUser);
	let serversList = $derived($servers);
	let metrics = $derived($metricsData);
	let stats = $derived($dashboardStats);
	let droppedAlerts = $derived($alertsStore.droppedMetrics);
	let rightSidebarOpen = $derived($uiStore.rightSidebarOpen);

	async function handleLogout() {
		try {
			await logout();
			// Clear all stores
			userStore.clear();
			serversStore.clear();
			metricsStore.clear();
			aggregatedStore.clear();
			alertsStore.clear();
			goto('/login');
		} catch (err) {
			console.error('Logout failed:', err);
			goto('/login');
		}
	}

	function toggleRightSidebar() {
		uiStore.toggleRightSidebar();
	}

	async function loadData() {
		try {
			loading = true;

			// Load user preferences
			await userStore.load();

			// Get time range from user preferences (migrate old 6h to 12h)
			const userTimeRange = $currentUser?.default_time_range || '24h';
			const migratedTimeRange = userTimeRange === '6h' ? '12h' : userTimeRange;
			aggregatedStore.setTimeRange(migratedTimeRange);

			// Load servers
			await serversStore.load();

			// Load metrics for each server (for table display)
			const serverIds = $servers.map(item => item.server.id);
			await metricsStore.loadForServers(serverIds, '1h');

			// Load dropped metrics alerts
			await alertsStore.load();

			// Load aggregated metrics (for stats cards and charts)
			await aggregatedStore.load(migratedTimeRange);

			// Load 24h metrics for trend calculation
			await aggregatedStore.load24h();
		} catch (err) {
			console.error('Failed to load data:', err);
			// Error will trigger redirect in apiRequest
		} finally {
			loading = false;
		}
	}

	async function handleTimeRangeChange(newTimeRange: TimeRange) {
		// Update the store with new time range
		aggregatedStore.setTimeRange(newTimeRange);
		// Load new data
		await aggregatedStore.load(newTimeRange);
	}

	function handleSSEMessage(event: SSEEvent) {
		if (event.type === 'server_update') {
			// Show toast notification if agent was reactivated
			if (event.data.reactivated && event.data.hostname) {
				toasts.add(
					`Agent "${event.data.hostname}" was reactivated (same physical server detected via UUID)`,
					'info',
					8000 // 8 seconds
				);
			}

			// Update server status via store
			serversStore.updateStatus(event.data.id, event.data.status, event.data.last_seen);
		} else if (event.type === 'metrics_update') {
			// Update individual server metrics via store
			metricsStore.updateServerMetrics(event.data.server_id, event.data);
		} else if (event.type === 'aggregated_metrics_update') {
			// Add new aggregated metrics point via store
			aggregatedStore.addMetricPoint(event.data);
		}
	}

	onMount(() => {
		loadData();

		// Connect to SSE for server status updates and real-time aggregated metrics
		sseUnsubscribe = sseStore.connect(handleSSEMessage);

		// Refresh dropped metrics every 1 hour
		const droppedMetricsInterval = setInterval(() => alertsStore.load(), 3600000);

		// Refresh 24h metrics for trend calculation every 5 minutes
		const trend24hInterval = setInterval(() => aggregatedStore.load24h(), 300000);

		// Cleanup on unmount
		return () => {
			clearInterval(droppedMetricsInterval);
			clearInterval(trend24hInterval);
		};
	});

	onDestroy(() => {
		// Unsubscribe from SSE
		if (sseUnsubscribe) {
			sseUnsubscribe();
		}
	});
</script>

<svelte:head>
	<title>Dashboard - Watchflare</title>
</svelte:head>

<div class="min-h-screen bg-background">
	<!-- Header -->
	<Header />

	<!-- Desktop Sidebar -->
	<DesktopSidebar onLogout={handleLogout} />

	<!-- Mobile Sidebar -->
	<MobileSidebar onLogout={handleLogout} />

	<!-- Right Sidebar -->
	<RightSidebar {stats} servers={serversList} isOpen={rightSidebarOpen} onToggle={toggleRightSidebar} />

	<!-- Main Content -->
	<main
		class="min-h-screen pt-16 p-4 md:p-8 md:pt-20 {$sidebarCollapsed
			? 'lg:ml-16'
			: 'lg:ml-64'} {rightSidebarOpen ? 'xl:mr-80' : 'mr-0'}"
	>
		{#if loading}
			<div class="flex items-center justify-center py-20">
				<p class="text-muted-foreground">Loading dashboard...</p>
			</div>
		{:else}
			<!-- Header with Welcome + Add Server -->
			<div class="mb-6 flex items-center justify-between">
				<div>
					<h1 class="text-2xl font-semibold text-foreground">
						Welcome back, <span class="text-primary"
							>{user?.email?.split('@')[0] || 'User'}</span
						>
					</h1>
					<p class="text-sm text-muted-foreground mt-1">
						Global uptime at <span class="font-medium text-foreground"
							>{formatPercent((stats.onlineServers / stats.totalServers) * 100)}</span
						> in the last 24h
					</p>
				</div>
				<button
					onclick={() => goto('/servers/new')}
					class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
				>
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Add Server
				</button>
			</div>

			<!-- Dropped Metrics Alerts -->
			<DroppedMetricsAlert alerts={droppedAlerts} />

			<!-- Dashboard Stats Cards -->
			<DashboardStats {stats} />

			<!-- Dashboard Charts -->
			<DashboardCharts aggregatedMetrics={$aggregatedMetrics} {stats} {timeRange} onTimeRangeChange={handleTimeRangeChange} />

			<!-- Servers Table -->
			<div class="mb-6">
				<div class="mb-4 flex items-center justify-between">
					<h2 class="text-lg font-semibold">Server Summary</h2>
				</div>
				<ServerTable servers={serversList} metricsData={metrics} />
			</div>
		{/if}
	</main>
</div>
