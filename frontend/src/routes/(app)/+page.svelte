<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { slide } from 'svelte/transition';
	import { get } from 'svelte/store';
	import { formatPercent, handleSSEReactivation, logger } from '$lib/utils';
	import { DROPPED_METRICS_POLL_INTERVAL, TREND_24H_POLL_INTERVAL } from '$lib/constants';
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
		sseStore,
		uiStore
	} from '$lib/stores';
	import ServerTable from '$lib/components/ServerTable.svelte';
	import DashboardStats from '$lib/components/dashboard/DashboardStats.svelte';
	import DashboardCharts from '$lib/components/dashboard/DashboardCharts.svelte';
	import DroppedMetricsAlert from '$lib/components/dashboard/DroppedMetricsAlert.svelte';
	import TimeRangeSelector from '$lib/components/TimeRangeSelector.svelte';
	import { ChevronUp, ChevronDown } from 'lucide-svelte';
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
	let metricsCollapsed = $derived($uiStore.metricsCollapsed);

	// Time range from store
	let selectedTimeRange = $derived($currentTimeRange);

	async function loadServerMetrics(serverIds: string[], timeRangeValue: TimeRange) {
		if (serverIds.length > 0) {
			await metricsStore.loadForServers(serverIds, timeRangeValue);
		}
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

			// Load servers and get IDs directly from API response
			await serversStore.load();

			// Load dropped metrics alerts, aggregated metrics, and per-server metrics in parallel
			const serverIds = get(serversStore).servers.map(s => s.server.id);

			await Promise.all([
				loadServerMetrics(serverIds, migratedTimeRange as TimeRange),
				alertsStore.load(),
				aggregatedStore.load(migratedTimeRange),
				aggregatedStore.load24h()
			]);
		} catch (err) {
			logger.error('Failed to load data:', err);
			// Error will trigger redirect in apiRequest
		} finally {
			loading = false;
		}
	}

	async function handleTimeRangeChange(newTimeRange: TimeRange) {
		aggregatedStore.setTimeRange(newTimeRange);

		const serverIds = get(serversStore).servers.map(s => s.server.id);

		// Load aggregated data and per-server metrics in parallel
		await Promise.all([
			aggregatedStore.load(newTimeRange),
			loadServerMetrics(serverIds, newTimeRange)
		]);
	}

	function handleSSEMessage(event: SSEEvent) {
		handleSSEReactivation(event);

		if (event.type === 'server_update') {
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
		const droppedMetricsInterval = setInterval(() => alertsStore.load(), DROPPED_METRICS_POLL_INTERVAL);

		// Refresh 24h metrics for trend calculation every 5 minutes
		const trend24hInterval = setInterval(() => aggregatedStore.load24h(), TREND_24H_POLL_INTERVAL);

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

{#if loading}
			<div class="flex items-center justify-center py-20">
				<p class="text-muted-foreground">Loading dashboard...</p>
			</div>
		{:else}
			<!-- Header with Welcome + Time Range + Add Server -->
			<div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div>
					<h1 class="text-xl sm:text-2xl font-semibold text-foreground">
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
				<TimeRangeSelector
					bind:value={selectedTimeRange}
					onValueChange={handleTimeRangeChange}
				/>
			</div>

			<!-- Dropped Metrics Alerts -->
			<DroppedMetricsAlert alerts={droppedAlerts} />

			<!-- Global Metrics Section -->
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Global Metrics</h2>
				<button
					onclick={() => uiStore.toggleMetricsCollapsed()}
					class="p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
					aria-label={metricsCollapsed ? 'Expand metrics' : 'Collapse metrics'}
				>
					{#if metricsCollapsed}
						<ChevronDown class="h-5 w-5" />
					{:else}
						<ChevronUp class="h-5 w-5" />
					{/if}
				</button>
			</div>

			<!-- Dashboard Stats Cards (compact when collapsed) -->
			<DashboardStats {stats} compact={metricsCollapsed} />

			<!-- Dashboard Charts (hidden when collapsed) -->
			{#if !metricsCollapsed}
				<div transition:slide={{ duration: 250 }}>
					<DashboardCharts aggregatedMetrics={$aggregatedMetrics} {stats} />
				</div>
			{/if}

			<!-- Servers Table -->
			<div class="mb-6">
				<div class="mb-4 flex items-center justify-between">
					<h2 class="text-lg font-semibold">Server Summary</h2>
				</div>
				<ServerTable servers={serversList.filter(s => s.server.status !== 'pending')} metricsData={metrics} />
			</div>
	{/if}
