<script lang="ts">
	import CPUChart from '$lib/components/CPUChart.svelte';
	import MemoryChart from '$lib/components/MemoryChart.svelte';
	import { formatBytes, formatPercent } from '$lib/utils';
	import type { AggregatedMetric, TimeRange } from '$lib/types';

	interface Stats {
		avgCPU: number;
		usedMemory: number;
		totalMemory: number;
	}

	const {
		aggregatedMetrics,
		stats,
		timeRange
	}: {
		aggregatedMetrics: AggregatedMetric[];
		stats: Stats;
		timeRange: TimeRange;
	} = $props();

	// Create a unique key based on the last metric's timestamp to force chart re-render
	let chartKey = $derived(
		aggregatedMetrics.length > 0
			? aggregatedMetrics[aggregatedMetrics.length - 1].timestamp
			: 'empty'
	);
</script>

<!-- Central Charts (CPU + Memory only) -->
<div class="mb-6">
	<div class="grid gap-4 xl:grid-cols-2">
		<!-- CPU Chart -->
		<div class="rounded-lg border bg-card p-4">
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-medium">CPU Usage</h3>
				<span class="text-xs text-muted-foreground">
					{formatPercent(stats.avgCPU)}
				</span>
			</div>
			{#key chartKey}
				<CPUChart data={aggregatedMetrics} {timeRange} />
			{/key}
		</div>

		<!-- Memory Chart -->
		<div class="rounded-lg border bg-card p-4">
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-medium">Memory Usage</h3>
				<span class="text-xs text-muted-foreground">
					<span class="sm:hidden">{formatPercent(stats.totalMemory > 0 ? (stats.usedMemory / stats.totalMemory) * 100 : 0)}</span>
					<span class="hidden sm:inline">{formatBytes(stats.usedMemory)} / {formatBytes(stats.totalMemory)} ({formatPercent(stats.totalMemory > 0 ? (stats.usedMemory / stats.totalMemory) * 100 : 0)})</span>
				</span>
			</div>
			{#key chartKey}
				<MemoryChart data={aggregatedMetrics} {timeRange} />
			{/key}
		</div>
	</div>
</div>
