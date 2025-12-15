<script>
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { logout, getCurrentUser, listServers, getServerMetrics } from '$lib/api';
	import { connectSSE } from '$lib/sse';
	import { getTimeRangeTimestamps, getIntervalForTimeRange, formatBytes, formatPercent } from '$lib/utils';
	import TimeRangeSelector from '$lib/components/TimeRangeSelector.svelte';
	import StatCard from '$lib/components/StatCard.svelte';
	import CPUChart from '$lib/components/CPUChart.svelte';
	import MemoryChart from '$lib/components/MemoryChart.svelte';
	import DiskChart from '$lib/components/DiskChart.svelte';

	// State
	let user = $state(null);
	let servers = $state([]);
	let timeRange = $state('24h');
	let loading = $state(true);
	let metricsData = $state({});
	let sseDisconnect = null;

	// Computed aggregates
	let stats = $derived({
		totalServers: servers.length,
		onlineServers: servers.filter((s) => s.server.status === 'online').length,
		offlineServers: servers.filter((s) => s.server.status === 'offline').length,

		// Aggregate CPU (average across all servers)
		avgCPU: servers.length > 0
			? servers.reduce((sum, s) => sum + (s.lastMetrics?.cpu_usage_percent || 0), 0) / servers.length
			: 0,

		// Aggregate Memory (sum of all servers)
		totalMemory: servers.reduce((sum, s) => sum + (s.lastMetrics?.memory_total_bytes || 0), 0),
		usedMemory: servers.reduce((sum, s) => sum + (s.lastMetrics?.memory_used_bytes || 0), 0),

		// Aggregate Disk (sum of all servers)
		totalDisk: servers.reduce((sum, s) => sum + (s.lastMetrics?.disk_total_bytes || 0), 0),
		usedDisk: servers.reduce((sum, s) => sum + (s.lastMetrics?.disk_used_bytes || 0), 0)
	});

	// Aggregate chart data from all servers
	let aggregatedChartData = $derived.by(() => {
		// Get all unique timestamps
		const timestampMap = new Map();

		servers.forEach((server) => {
			server.metrics.forEach((metric) => {
				const timestamp = metric.timestamp;
				if (!timestampMap.has(timestamp)) {
					timestampMap.set(timestamp, {
						timestamp,
						cpuSum: 0,
						cpuCount: 0,
						memoryTotal: 0,
						memoryUsed: 0,
						diskTotal: 0,
						diskUsed: 0
					});
				}

				const entry = timestampMap.get(timestamp);
				entry.cpuSum += metric.cpu_usage_percent || 0;
				entry.cpuCount += 1;
				entry.memoryTotal += metric.memory_total_bytes || 0;
				entry.memoryUsed += metric.memory_used_bytes || 0;
				entry.diskTotal += metric.disk_total_bytes || 0;
				entry.diskUsed += metric.disk_used_bytes || 0;
			});
		});

		// Convert to array and calculate averages
		return Array.from(timestampMap.values())
			.map((entry) => ({
				timestamp: entry.timestamp,
				cpu_usage_percent: entry.cpuCount > 0 ? entry.cpuSum / entry.cpuCount : 0,
				memory_total_bytes: entry.memoryTotal,
				memory_used_bytes: entry.memoryUsed,
				memory_available_bytes: entry.memoryTotal - entry.memoryUsed,
				disk_total_bytes: entry.diskTotal,
				disk_used_bytes: entry.diskUsed
			}))
			.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
	});

	async function handleLogout() {
		try {
			await logout();
			goto('/login');
		} catch (err) {
			console.error('Logout failed:', err);
			goto('/login');
		}
	}

	async function loadData() {
		try {
			loading = true;

			// Load user preferences
			const userData = await getCurrentUser();
			user = userData.user;
			timeRange = user.default_time_range || '24h';

			// Load servers
			const serversData = await listServers();
			servers = serversData.servers.map((server) => ({
				server,
				lastMetrics: null,
				metrics: []
			}));

			// Load metrics for each server
			await loadMetrics();
		} catch (err) {
			console.error('Failed to load data:', err);
		} finally {
			loading = false;
		}
	}

	async function loadMetrics() {
		const { start, end } = getTimeRangeTimestamps(timeRange);
		const interval = getIntervalForTimeRange(timeRange);

		for (let i = 0; i < servers.length; i++) {
			const server = servers[i];
			if (server.server.status !== 'online') continue;

			try {
				const data = await getServerMetrics(server.server.id, { start, end, interval });
				servers[i].metrics = data.metrics || [];

				// Set last metrics (most recent point)
				if (servers[i].metrics.length > 0) {
					servers[i].lastMetrics = servers[i].metrics[servers[i].metrics.length - 1];
				}
			} catch (err) {
				console.error(`Failed to load metrics for server ${server.server.id}:`, err);
			}
		}
	}

	function handleTimeRangeChange() {
		loadMetrics();
	}

	function handleSSEMessage(event) {
		if (event.type === 'metrics_update') {
			const { server_id, ...metrics } = event.data;

			// Find server and update last metrics
			const serverIndex = servers.findIndex((s) => s.server.id === server_id);
			if (serverIndex !== -1) {
				servers[serverIndex].lastMetrics = metrics;

				// Add to metrics array for real-time chart update
				servers[serverIndex].metrics = [...servers[serverIndex].metrics, metrics];

				// Keep only last 100 points to avoid memory issues
				if (servers[serverIndex].metrics.length > 100) {
					servers[serverIndex].metrics = servers[serverIndex].metrics.slice(-100);
				}
			}
		} else if (event.type === 'server_update') {
			// Update server status
			const serverIndex = servers.findIndex((s) => s.server.id === event.data.id);
			if (serverIndex !== -1) {
				servers[serverIndex].server.status = event.data.status;
				servers[serverIndex].server.last_seen = event.data.last_seen;
			}
		}
	}

	onMount(() => {
		loadData();

		// Connect to SSE for real-time updates
		sseDisconnect = connectSSE(handleSSEMessage, (error) => {
			console.error('SSE error:', error);
		});
	});

	onDestroy(() => {
		if (sseDisconnect) {
			sseDisconnect();
		}
	});
</script>

<svelte:head>
	<title>Dashboard - Watchflare</title>
</svelte:head>

<div class="min-h-screen bg-gradient-to-br from-background via-background to-muted/20">
	<!-- Navbar -->
	<nav class="sticky top-0 z-50 border-b bg-card/80 backdrop-blur-lg">
		<div class="mx-auto max-w-7xl px-6 py-4">
			<div class="flex items-center justify-between">
				<h1 class="text-2xl font-bold tracking-tight bg-gradient-to-r from-primary to-primary/70 bg-clip-text text-transparent">
					Watchflare
				</h1>
				<div class="flex items-center gap-6">
					<a
						href="/servers"
						class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
					>
						Servers
					</a>
					<a
						href="/settings"
						class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
					>
						Settings
					</a>
					<button
						onclick={handleLogout}
						class="text-sm font-medium text-destructive hover:text-destructive/90 transition-colors"
					>
						Logout
					</button>
				</div>
			</div>
		</div>
	</nav>

	<!-- Main Content -->
	<main class="mx-auto max-w-7xl px-6 py-10">
		{#if loading}
			<div class="flex items-center justify-center py-20">
				<p class="text-muted-foreground">Loading dashboard...</p>
			</div>
		{:else}
			<!-- Header with Time Range Selector -->
			<div class="mb-10 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
				<div>
					<h2 class="text-4xl font-bold tracking-tight">Dashboard</h2>
					<p class="text-muted-foreground mt-2">
						Overview of {stats.totalServers} {stats.totalServers === 1 ? 'server' : 'servers'}
					</p>
				</div>
				<TimeRangeSelector bind:value={timeRange} onValueChange={handleTimeRangeChange} />
			</div>

			<!-- Bento Grid -->
			<div class="grid gap-5 md:grid-cols-2 lg:grid-cols-4 mb-8">
				<!-- Servers Status -->
				<StatCard
					title="Servers"
					value={stats.totalServers}
					subtitle="{stats.onlineServers} online, {stats.offlineServers} offline"
					icon="🖥️"
				/>

				<!-- Average CPU -->
				<StatCard
					title="Average CPU"
					value={formatPercent(stats.avgCPU)}
					subtitle="Across all servers"
					icon="💻"
				/>

				<!-- Total Memory -->
				<StatCard
					title="Total RAM"
					value={formatBytes(stats.usedMemory)}
					subtitle="of {formatBytes(stats.totalMemory)}"
					icon="💾"
				/>

				<!-- Total Disk -->
				<StatCard
					title="Total Disk"
					value={formatBytes(stats.usedDisk)}
					subtitle="of {formatBytes(stats.totalDisk)}"
					icon="💿"
				/>
			</div>

			<!-- Charts Section -->
			<div class="grid gap-6 lg:grid-cols-2">
				<!-- CPU Chart -->
				<div class="rounded-xl border bg-card p-6 shadow-sm hover:shadow-md transition-shadow">
					<h3 class="text-lg font-semibold mb-4">CPU Usage (Average)</h3>
					<CPUChart data={aggregatedChartData} />
				</div>

				<!-- Memory Chart -->
				<div class="rounded-xl border bg-card p-6 shadow-sm hover:shadow-md transition-shadow">
					<h3 class="text-lg font-semibold mb-4">Memory Usage (Total)</h3>
					<MemoryChart data={aggregatedChartData} />
				</div>

				<!-- Disk Chart -->
				<div class="rounded-xl border bg-card p-6 shadow-sm hover:shadow-md transition-shadow lg:col-span-2">
					<h3 class="text-lg font-semibold mb-4">Disk Usage (Total)</h3>
					<DiskChart data={aggregatedChartData} />
				</div>
			</div>
		{/if}
	</main>
</div>
