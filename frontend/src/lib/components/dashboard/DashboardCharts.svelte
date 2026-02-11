<script lang="ts">
	import CPUChart from '$lib/components/CPUChart.svelte';
	import MemoryChart from '$lib/components/MemoryChart.svelte';
	import TimeRangeSelector from '$lib/components/TimeRangeSelector.svelte';
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
		timeRange,
		onTimeRangeChange
	}: {
		aggregatedMetrics: AggregatedMetric[];
		stats: Stats;
		timeRange: TimeRange;
		onTimeRangeChange: (newTimeRange: TimeRange) => void;
	} = $props();

	// Bind timeRange for two-way binding
	let currentTimeRange = $state(timeRange);

	// Watch for changes and sync with parent
	$effect(() => {
		// Sync from parent to local
		currentTimeRange = timeRange;
	});

	// Create a unique key based on the last metric's timestamp to force chart re-render
	let chartKey = $derived(
		aggregatedMetrics.length > 0
			? aggregatedMetrics[aggregatedMetrics.length - 1].timestamp
			: 'empty'
	);

	// Handle time range change from selector
	function handleTimeRangeChange(newTimeRange: TimeRange) {
		currentTimeRange = newTimeRange;
		onTimeRangeChange(newTimeRange);
	}
</script>

<!-- Central Charts (CPU + Memory only) -->
<div class="mb-6">
	<div class="mb-4 flex items-center justify-between">
		<h2 class="text-lg font-semibold">Global Metrics</h2>
		<TimeRangeSelector
			bind:value={currentTimeRange}
			onValueChange={handleTimeRangeChange}
		/>
	</div>
	<div class="grid gap-4 lg:grid-cols-2">
		<!-- CPU Chart -->
		<div class="rounded-lg border bg-card p-4">
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-medium">CPU Usage</h3>
				<span class="text-xs text-muted-foreground">
					{formatPercent(stats.avgCPU)}
				</span>
			</div>
			{#key chartKey}
				<CPUChart data={aggregatedMetrics} />
			{/key}
		</div>

		<!-- Memory Chart -->
		<div class="rounded-lg border bg-card p-4">
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-medium">Memory Usage</h3>
				<span class="text-xs text-muted-foreground">
					{formatBytes(stats.usedMemory)} / {formatBytes(stats.totalMemory)}
				</span>
			</div>
			{#key chartKey}
				<MemoryChart data={aggregatedMetrics} />
			{/key}
		</div>
	</div>
</div>
