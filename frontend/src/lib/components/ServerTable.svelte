<script lang="ts">
	import { formatBytes, formatPercent, getStatusClass, getMetricClass, formatRelativeTime } from '$lib/utils';
	import type { ServerWithMetrics, Metric } from '$lib/types';

	const { servers, metricsData }: {
		servers: ServerWithMetrics[];
		metricsData: Record<string, Metric[]>;
	} = $props();

	let sortColumn = $state('name');
	let sortOrder = $state<'asc' | 'desc'>('asc');

	function handleSort(column: string) {
		if (sortColumn === column) {
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortColumn = column;
			sortOrder = 'asc';
		}
	}

	function getLastMetrics(serverId: string) {
		const metrics = metricsData[serverId];
		if (!metrics || metrics.length === 0) {
			return { hasData: false, cpu: 0, memory: 0, disk: 0, memoryUsed: 0, memoryTotal: 0, diskUsed: 0, diskTotal: 0 };
		}

		// If only 1-2 points (real-time SSE), use the latest
		if (metrics.length <= 2) {
			const latest = metrics[metrics.length - 1];
			return {
				hasData: true,
				cpu: latest.cpu_usage_percent || 0,
				memory: latest.memory_total_bytes > 0 ? (latest.memory_used_bytes / latest.memory_total_bytes) * 100 : 0,
				disk: latest.disk_total_bytes > 0 ? (latest.disk_used_bytes / latest.disk_total_bytes) * 100 : 0,
				memoryUsed: latest.memory_used_bytes || 0,
				memoryTotal: latest.memory_total_bytes || 0,
				diskUsed: latest.disk_used_bytes || 0,
				diskTotal: latest.disk_total_bytes || 0
			};
		}

		// Multiple points (historical range): compute averages
		let cpuSum = 0, memUsedSum = 0, memTotalSum = 0, diskUsedSum = 0, diskTotalSum = 0;
		let count = 0;
		for (const m of metrics) {
			cpuSum += m.cpu_usage_percent || 0;
			memUsedSum += m.memory_used_bytes || 0;
			memTotalSum += m.memory_total_bytes || 0;
			diskUsedSum += m.disk_used_bytes || 0;
			diskTotalSum += m.disk_total_bytes || 0;
			count++;
		}
		const avgMemTotal = memTotalSum / count;
		const avgMemUsed = memUsedSum / count;
		const avgDiskTotal = diskTotalSum / count;
		const avgDiskUsed = diskUsedSum / count;
		return {
			hasData: true,
			cpu: cpuSum / count,
			memory: avgMemTotal > 0 ? (avgMemUsed / avgMemTotal) * 100 : 0,
			disk: avgDiskTotal > 0 ? (avgDiskUsed / avgDiskTotal) * 100 : 0,
			memoryUsed: avgMemUsed,
			memoryTotal: avgMemTotal,
			diskUsed: avgDiskUsed,
			diskTotal: avgDiskTotal
		};
	}

	const sortedServers = $derived(() => {
		const sorted = [...servers].sort((a, b) => {
			let valA, valB;
			switch (sortColumn) {
				case 'name':
					valA = (a.server.name || '').toLowerCase();
					valB = (b.server.name || '').toLowerCase();
					break;
				case 'status':
					valA = a.server.status || '';
					valB = b.server.status || '';
					break;
				case 'cpu': {
					const mA = getLastMetrics(a.server.id);
					const mB = getLastMetrics(b.server.id);
					valA = mA.cpu;
					valB = mB.cpu;
					break;
				}
				case 'memory': {
					const mA = getLastMetrics(a.server.id);
					const mB = getLastMetrics(b.server.id);
					valA = mA.memory;
					valB = mB.memory;
					break;
				}
				case 'disk': {
					const mA = getLastMetrics(a.server.id);
					const mB = getLastMetrics(b.server.id);
					valA = mA.disk;
					valB = mB.disk;
					break;
				}
				case 'last_seen': {
					valA = a.server.last_seen ? new Date(a.server.last_seen).getTime() : 0;
					valB = b.server.last_seen ? new Date(b.server.last_seen).getTime() : 0;
					break;
				}
				default:
					return 0;
			}
			if (valA < valB) return sortOrder === 'asc' ? -1 : 1;
			if (valA > valB) return sortOrder === 'asc' ? 1 : -1;
			return 0;
		});
		return sorted;
	});

</script>

{#snippet sortIcon(column)}
	{#if sortColumn === column}
		<svg class="h-3 w-3" viewBox="0 0 12 12" fill="currentColor">
			{#if sortOrder === 'asc'}
				<path d="M6 2l4 5H2z" />
			{:else}
				<path d="M6 10l4-5H2z" />
			{/if}
		</svg>
	{/if}
{/snippet}

<div class="rounded-lg border bg-card">
	<!-- Mobile: Cards layout -->
	<div class="md:hidden divide-y divide-border">
		{#each sortedServers() as { server }}
			{@const metrics = getLastMetrics(server.id)}
			<a href="/servers/{server.id}" class="block p-4 hover:bg-muted/20 transition-colors">
				<div class="flex items-center justify-between mb-2">
					<div>
						<span class="font-medium text-foreground">{server.name}</span>
						{#if server.hostname}
							<span class="text-xs text-muted-foreground ml-2">{server.hostname}</span>
						{/if}
					</div>
					<span
						class="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium {getStatusClass(server.status)}"
					>
						<span class="h-1.5 w-1.5 rounded-full {server.status === 'online' ? 'bg-success' : 'bg-muted-foreground'}"></span>
						{server.status}
					</span>
				</div>
				{#if metrics.hasData}
					<div class="grid grid-cols-3 gap-3 text-sm">
						<div>
							<span class="text-xs text-muted-foreground">CPU</span>
							<p class={getMetricClass(metrics.cpu)}>{formatPercent(metrics.cpu)}</p>
						</div>
						<div>
							<span class="text-xs text-muted-foreground">Memory</span>
							<p class={getMetricClass(metrics.memory)}>{formatPercent(metrics.memory)}</p>
							<p class="text-xs text-muted-foreground">{formatBytes(metrics.memoryUsed)}</p>
						</div>
						<div>
							<span class="text-xs text-muted-foreground">Disk</span>
							<p class={getMetricClass(metrics.disk)}>{formatPercent(metrics.disk)}</p>
							<p class="text-xs text-muted-foreground">{formatBytes(metrics.diskUsed)}</p>
						</div>
					</div>
				{:else}
					<p class="text-xs text-muted-foreground">No metrics available</p>
				{/if}
			</a>
		{/each}
	</div>

	<!-- Desktop: Table layout -->
	<div class="hidden md:block overflow-x-auto">
		<table class="w-full min-w-[1000px]">
			<thead>
				<tr class="border-b bg-muted/30">
					<th scope="col"
						class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
						onclick={() => handleSort('name')}
					>
						<span class="inline-flex items-center gap-1">
							Server
							{@render sortIcon('name')}
						</span>
					</th>
					<th scope="col"
						class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
						onclick={() => handleSort('status')}
					>
						<span class="inline-flex items-center gap-1">
							Status
							{@render sortIcon('status')}
						</span>
					</th>
					<th scope="col"
						class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
						onclick={() => handleSort('cpu')}
					>
						<span class="inline-flex items-center gap-1 justify-end w-full">
							CPU
							{@render sortIcon('cpu')}
						</span>
					</th>
					<th scope="col"
						class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
						onclick={() => handleSort('memory')}
					>
						<span class="inline-flex items-center gap-1 justify-end w-full">
							Memory
							{@render sortIcon('memory')}
						</span>
					</th>
					<th scope="col"
						class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
						onclick={() => handleSort('disk')}
					>
						<span class="inline-flex items-center gap-1 justify-end w-full">
							Disk
							{@render sortIcon('disk')}
						</span>
					</th>
					<th scope="col" class="px-4 py-3 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider">
						Updates
					</th>
					<th scope="col"
						class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
						onclick={() => handleSort('last_seen')}
					>
						<span class="inline-flex items-center gap-1 justify-end w-full">
							Last Seen
							{@render sortIcon('last_seen')}
						</span>
					</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-border">
				{#each sortedServers() as { server }}
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
							{#if metrics.hasData}
								<span class={getMetricClass(metrics.cpu)}>
									{formatPercent(metrics.cpu)}
								</span>
							{:else}
								<span class="text-muted-foreground">-</span>
							{/if}
						</td>

						<!-- Memory -->
						<td class="px-4 py-3.5 text-right">
							{#if metrics.hasData}
								<div class="flex flex-col items-end">
									<span class={getMetricClass(metrics.memory)}>
										{formatPercent(metrics.memory)}
									</span>
									<span class="text-xs text-muted-foreground">
										{formatBytes(metrics.memoryUsed)} / {formatBytes(metrics.memoryTotal)}
									</span>
								</div>
							{:else}
								<span class="text-muted-foreground">-</span>
							{/if}
						</td>

						<!-- Disk -->
						<td class="px-4 py-3.5 text-right">
							{#if metrics.hasData}
								<div class="flex flex-col items-end">
									<span class={getMetricClass(metrics.disk)}>
										{formatPercent(metrics.disk)}
									</span>
									<span class="text-xs text-muted-foreground">
										{formatBytes(metrics.diskUsed)} / {formatBytes(metrics.diskTotal)}
									</span>
								</div>
							{:else}
								<span class="text-muted-foreground">-</span>
							{/if}
						</td>

						<!-- Updates (MCO/MCS) - Placeholder -->
						<td class="px-4 py-3.5 text-center">
							<span class="text-xs text-muted-foreground">
								-
							</span>
						</td>

						<!-- Last Seen -->
						<td class="px-4 py-3.5 text-right text-sm text-muted-foreground">
							{formatRelativeTime(server.last_seen)}
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
