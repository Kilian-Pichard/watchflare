<script lang="ts">
	import { formatBytes } from '$lib/utils';
	import CPUChart from '$lib/components/CPUChart.svelte';
	import MemoryChart from '$lib/components/MemoryChart.svelte';
	import DiskChart from '$lib/components/DiskChart.svelte';
	import TimeRangeSelector from '$lib/components/TimeRangeSelector.svelte';
	import type { Metric, TimeRange } from '$lib/types';

	let {
		metrics,
		timeRange = $bindable(),
		onTimeRangeChange,
	}: {
		metrics: Metric[];
		timeRange: TimeRange;
		onTimeRangeChange: (range: TimeRange) => void;
	} = $props();

	const latestMetric = $derived(metrics.length > 0 ? metrics[metrics.length - 1] : null);
</script>

<div class="mb-6">
	<div class="mb-4 flex items-center justify-between">
		<h2 class="text-lg font-semibold text-foreground">Metrics</h2>
		<TimeRangeSelector bind:value={timeRange} onValueChange={onTimeRangeChange} />
	</div>

	<div class="grid gap-4 xl:grid-cols-2">
		<div class="rounded-lg border bg-card p-4">
			<div class="flex items-center justify-between mb-3">
				<h3 class="text-sm font-medium text-foreground">CPU Usage</h3>
				{#if latestMetric}
					<span class="text-sm font-semibold text-foreground"
						>{latestMetric.cpu_usage_percent.toFixed(1)}%</span
					>
				{/if}
			</div>
			<CPUChart data={metrics} />
		</div>
		<div class="rounded-lg border bg-card p-4">
			<div class="flex items-center justify-between mb-3">
				<h3 class="text-sm font-medium text-foreground">Memory Usage</h3>
				{#if latestMetric}
					<span class="text-sm font-semibold text-foreground"
						>{formatBytes(latestMetric.memory_used_bytes)} / {formatBytes(latestMetric.memory_total_bytes)}</span
					>
				{/if}
			</div>
			<MemoryChart data={metrics} />
		</div>
		<div class="rounded-lg border bg-card p-4">
			<div class="flex items-center justify-between mb-3">
				<h3 class="text-sm font-medium text-foreground">Disk Usage</h3>
				{#if latestMetric}
					<span class="text-sm font-semibold text-foreground"
						>{formatBytes(latestMetric.disk_used_bytes)} / {formatBytes(latestMetric.disk_total_bytes)}</span
					>
				{/if}
			</div>
			<DiskChart data={metrics} />
		</div>
	</div>
</div>
