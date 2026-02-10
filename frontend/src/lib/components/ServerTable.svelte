<script>
	import { formatBytes, formatPercent } from '$lib/utils';

	const { servers, metricsData } = $props();

	function getStatusClass(status) {
		return status === 'online'
			? 'bg-success/10 text-success border-success/20'
			: 'bg-muted text-muted-foreground border-border';
	}

	function getMetricClass(percent) {
		if (percent >= 90) return 'text-danger font-semibold';
		if (percent >= 70) return 'text-warning font-medium';
		return 'text-foreground';
	}

	function getLastMetrics(serverId) {
		const metrics = metricsData[serverId];
		if (!metrics || metrics.length === 0) {
			return { cpu: 0, memory: 0, disk: 0, memoryTotal: 0, diskTotal: 0 };
		}
		const latest = metrics[metrics.length - 1];
		return {
			cpu: latest.cpu_usage_percent || 0,
			memory: (latest.memory_used_bytes / latest.memory_total_bytes) * 100 || 0,
			disk: (latest.disk_used_bytes / latest.disk_total_bytes) * 100 || 0,
			memoryUsed: latest.memory_used_bytes || 0,
			memoryTotal: latest.memory_total_bytes || 0,
			diskUsed: latest.disk_used_bytes || 0,
			diskTotal: latest.disk_total_bytes || 0
		};
	}

	function formatTimestamp(timestamp) {
		if (!timestamp) return 'Never';
		const date = new Date(timestamp);
		const now = new Date();
		const diff = Math.floor((now - date) / 1000); // seconds

		if (diff < 60) return `${diff}s ago`;
		if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
		if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
		return `${Math.floor(diff / 86400)}d ago`;
	}
</script>

<div class="rounded-lg border bg-card">
	<div class="overflow-x-auto">
		<table class="w-full">
			<thead>
				<tr class="border-b bg-muted/30">
					<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
						Server
					</th>
					<th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
						Status
					</th>
					<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
						CPU
					</th>
					<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
						Memory
					</th>
					<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
						Disk
					</th>
					<th class="px-4 py-3 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider">
						Updates
					</th>
					<th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
						Last Seen
					</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-border">
				{#each servers as { server }}
					{@const metrics = getLastMetrics(server.id)}
					<tr class="hover:bg-muted/20 transition-colors">
						<!-- Server Name -->
						<td class="px-4 py-3.5">
							<a
								href="/servers/{server.id}"
								class="group flex flex-col"
							>
								<span class="font-medium text-foreground group-hover:text-primary transition-colors">
									{server.name}
								</span>
								{#if server.hostname}
									<span class="text-xs text-muted-foreground">
										{server.hostname}
									</span>
								{/if}
							</a>
						</td>

						<!-- Status -->
						<td class="px-4 py-3.5">
							<span
								class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(server.status)}"
							>
								<span class="h-1.5 w-1.5 rounded-full {server.status === 'online' ? 'bg-success' : 'bg-muted-foreground'}"></span>
								{server.status}
							</span>
						</td>

						<!-- CPU -->
						<td class="px-4 py-3.5 text-right">
							<span class={getMetricClass(metrics.cpu)}>
								{formatPercent(metrics.cpu)}
							</span>
						</td>

						<!-- Memory -->
						<td class="px-4 py-3.5 text-right">
							<div class="flex flex-col items-end">
								<span class={getMetricClass(metrics.memory)}>
									{formatPercent(metrics.memory)}
								</span>
								<span class="text-xs text-muted-foreground">
									{formatBytes(metrics.memoryUsed)} / {formatBytes(metrics.memoryTotal)}
								</span>
							</div>
						</td>

						<!-- Disk -->
						<td class="px-4 py-3.5 text-right">
							<div class="flex flex-col items-end">
								<span class={getMetricClass(metrics.disk)}>
									{formatPercent(metrics.disk)}
								</span>
								<span class="text-xs text-muted-foreground">
									{formatBytes(metrics.diskUsed)} / {formatBytes(metrics.diskTotal)}
								</span>
							</div>
						</td>

						<!-- Updates (MCO/MCS) - Placeholder -->
						<td class="px-4 py-3.5 text-center">
							<span class="text-xs text-muted-foreground">
								-
							</span>
							<!-- Future: Update badge when backend ready -->
							<!-- <span class="inline-flex items-center gap-1 rounded-full bg-warning/10 px-2 py-0.5 text-xs font-medium text-warning">
								<svg class="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
									<path d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-11a1 1 0 10-2 0v2H7a1 1 0 100 2h2v2a1 1 0 102 0v-2h2a1 1 0 100-2h-2V7z"/>
								</svg>
								5
							</span> -->
						</td>

						<!-- Last Seen -->
						<td class="px-4 py-3.5 text-right text-sm text-muted-foreground">
							{formatTimestamp(server.last_seen)}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	{#if servers.length === 0}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<svg
				class="h-12 w-12 text-muted-foreground/50 mb-3"
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
			<p class="text-sm text-muted-foreground">No servers found</p>
			<p class="text-xs text-muted-foreground mt-1">
				Add your first server to start monitoring
			</p>
		</div>
	{/if}
</div>
