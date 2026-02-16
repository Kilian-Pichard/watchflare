<script>
	import { fade } from 'svelte/transition';

	const { servers, isOpen, onClose } = $props();

	// Generate alerts based on server status and metrics
	const alerts = $derived(generateAlerts(servers));

	function generateAlerts(servers) {
		const alerts = [];

		servers.forEach(({ server, latestMetric }) => {
			if (server.status === 'offline') {
				alerts.push({
					type: 'critical',
					server: server.name,
					message: 'Server is offline',
					time: 'Just now'
				});
			}

			if (server.status === 'ip_mismatch') {
				alerts.push({
					type: 'warning',
					server: server.name,
					message: 'IP address mismatch detected',
					time: 'Just now'
				});
			}

			if (latestMetric && latestMetric.cpu_usage_percent > 90) {
				alerts.push({
					type: 'warning',
					server: server.name,
					message: `High CPU usage: ${latestMetric.cpu_usage_percent.toFixed(1)}%`,
					time: 'Just now'
				});
			}

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

		return alerts.slice(0, 10);
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

{#if isOpen}
	<!-- Backdrop -->
	<div
		transition:fade={{ duration: 200 }}
		class="fixed inset-0 z-40 bg-black/50"
		onclick={onClose}
	></div>

	<!-- Panel -->
	<aside class="fixed right-0 top-0 z-50 h-screen w-80 max-w-[85vw] bg-sidebar border-l shadow-lg overflow-y-auto">
		<!-- Header -->
		<div class="flex items-center justify-between border-b px-6 py-4">
			<h2 class="text-sm font-semibold text-foreground">Active Alerts</h2>
			<div class="flex items-center gap-2">
				{#if alerts.length > 0}
					<span class="flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-xs font-medium text-primary-foreground">
						{alerts.length}
					</span>
				{/if}
				<button
					onclick={onClose}
					class="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
					aria-label="Close alerts"
				>
					<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
					</svg>
				</button>
			</div>
		</div>

		<!-- Alerts list -->
		<div class="p-6">
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
{/if}
