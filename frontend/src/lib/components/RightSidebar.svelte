<script>
	const { stats, servers, isOpen, onToggle } = $props();

	// Calculate resource usage percentages
	const avgCpuPercent = $derived(stats.avgCPU || 0);
	const avgMemoryPercent = $derived(stats.avgMemory || 0);
	const avgDiskPercent = $derived(stats.avgDisk || 0);

	// Generate alerts based on server status and metrics
	const alerts = $derived(generateAlerts(servers));

	function generateAlerts(servers) {
		const alerts = [];

		servers.forEach(({ server, latestMetric }) => {
			// Offline servers
			if (server.status === 'offline') {
				alerts.push({
					type: 'critical',
					server: server.name,
					message: 'Server is offline',
					time: 'Just now'
				});
			}

			// IP mismatch
			if (server.status === 'ip_mismatch') {
				alerts.push({
					type: 'warning',
					server: server.name,
					message: 'IP address mismatch detected',
					time: 'Just now'
				});
			}

			// High CPU
			if (latestMetric && latestMetric.cpu_usage_percent > 90) {
				alerts.push({
					type: 'warning',
					server: server.name,
					message: `High CPU usage: ${latestMetric.cpu_usage_percent.toFixed(1)}%`,
					time: 'Just now'
				});
			}

			// High Memory
			if (latestMetric && latestMetric.memory_total_bytes > 0) {
				const memPercent = (latestMetric.memory_used_bytes / latestMetric.memory_total_bytes) * 100;
				if (memPercent > 90) {
					alerts.push({
						type: 'warning',
						server: server.name,
						message: `High memory usage: ${memPercent.toFixed(1)}%`,
						time: 'Just now'
					});
				}
			}
		});

		return alerts.slice(0, 5); // Limit to 5 most recent alerts
	}

	function getAlertIcon(type) {
		if (type === 'critical') {
			return '<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/></svg>';
		}
		return '<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/></svg>';
	}

	function getAlertColor(type) {
		if (type === 'critical') {
			return 'bg-destructive/10 text-destructive border-destructive/20';
		}
		return 'bg-warning/10 text-warning border-warning/20';
	}
</script>

<!-- Toggle Button -->
<button
	onclick={onToggle}
	class="fixed right-0 top-20 z-40 flex h-10 w-10 items-center justify-center rounded-l-lg border border-r-0 bg-card text-muted-foreground transition-all hover:bg-muted hover:text-foreground {isOpen ? 'translate-x-[-320px]' : ''}"
	aria-label={isOpen ? 'Close sidebar' : 'Open sidebar'}
>
	{#if isOpen}
		<!-- Chevron Right (close) -->
		<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
		</svg>
	{:else}
		<!-- Chevron Left (open) -->
		<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
		</svg>
	{/if}
</button>

<!-- Sidebar -->
<aside class="fixed right-0 top-0 z-30 h-screen w-80 border-l bg-sidebar p-6 overflow-y-auto transition-transform duration-300 {isOpen ? 'translate-x-0' : 'translate-x-full'}">
	<!-- Resource Usage -->
	<div class="mb-8">
		<h3 class="text-sm font-semibold text-foreground mb-4">Global Resource Usage</h3>

		<!-- CPU -->
		<div class="mb-4">
			<div class="flex items-center justify-between mb-2">
				<span class="text-xs text-muted-foreground">CPU Load</span>
				<span class="text-xs font-medium text-foreground">{avgCpuPercent.toFixed(1)}%</span>
			</div>
			<div class="h-2 rounded-full bg-muted overflow-hidden">
				<div
					class="h-full bg-primary transition-all duration-300"
					style="width: {avgCpuPercent}%"
				></div>
			</div>
		</div>

		<!-- Memory -->
		<div class="mb-4">
			<div class="flex items-center justify-between mb-2">
				<span class="text-xs text-muted-foreground">Memory</span>
				<span class="text-xs font-medium text-foreground">{avgMemoryPercent.toFixed(1)}%</span>
			</div>
			<div class="h-2 rounded-full bg-muted overflow-hidden">
				<div
					class="h-full bg-[var(--chart-2)] transition-all duration-300"
					style="width: {avgMemoryPercent}%"
				></div>
			</div>
		</div>

		<!-- Disk -->
		<div>
			<div class="flex items-center justify-between mb-2">
				<span class="text-xs text-muted-foreground">Disk I/O</span>
				<span class="text-xs font-medium text-foreground">{avgDiskPercent.toFixed(1)}%</span>
			</div>
			<div class="h-2 rounded-full bg-muted overflow-hidden">
				<div
					class="h-full bg-[var(--chart-3)] transition-all duration-300"
					style="width: {avgDiskPercent}%"
				></div>
			</div>
		</div>
	</div>

	<!-- Alerts -->
	<div>
		<div class="flex items-center justify-between mb-4">
			<h3 class="text-sm font-semibold text-foreground">Recent Alerts</h3>
			{#if alerts.length > 0}
				<span class="flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-xs font-medium text-primary-foreground">
					{alerts.length}
				</span>
			{/if}
		</div>

		{#if alerts.length === 0}
			<div class="rounded-lg border border-dashed bg-muted/20 p-6 text-center">
				<svg class="mx-auto h-8 w-8 text-muted-foreground/50 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
				</svg>
				<p class="text-xs text-muted-foreground">No alerts</p>
			</div>
		{:else}
			<div class="space-y-2">
				{#each alerts as alert}
					<div class="rounded-lg border p-3 {getAlertColor(alert.type)}">
						<div class="flex items-start gap-2">
							<div class="mt-0.5">
								{@html getAlertIcon(alert.type)}
							</div>
							<div class="flex-1 min-w-0">
								<p class="text-xs font-medium mb-0.5">{alert.server}</p>
								<p class="text-xs opacity-90">{alert.message}</p>
								<p class="text-xs opacity-60 mt-1">{alert.time}</p>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</aside>
